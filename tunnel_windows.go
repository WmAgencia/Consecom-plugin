//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type tunnelSettings struct {
	ID  string `json:"tunnel_id"`
	Key string `json:"key"`
}
type tunnelRuntime struct {
	sync.Mutex
	cmd      *exec.Cmd
	done     chan struct{}
	started  time.Time
	issue    string
	tunnelID string
}

var localTunnel tunnelRuntime
var tunnelRunLock sync.Mutex
var tunnelIDPattern = regexp.MustCompile(`^tunnel_[A-Za-z0-9_-]{12,128}$`)

// Windows DPAPI encrypts the credential for the current Windows user;
// neither the bearer key nor its ciphertext is exposed over /api/status.
type dataBlob struct {
	size uint32
	data *byte
}

var crypt32 = syscall.NewLazyDLL("crypt32.dll")
var kernel32Tunnel = syscall.NewLazyDLL("kernel32.dll")
var protect = crypt32.NewProc("CryptProtectData")
var unprotect = crypt32.NewProc("CryptUnprotectData")
var localFree = kernel32Tunnel.NewProc("LocalFree")

func blobOf(b []byte) dataBlob {
	if len(b) == 0 {
		return dataBlob{}
	}
	return dataBlob{size: uint32(len(b)), data: &b[0]}
}
func dpapi(b []byte, encode bool) ([]byte, error) {
	if len(b) == 0 {
		return nil, errors.New("credencial vazia")
	}
	input := blobOf(b)
	var output dataBlob
	fn := unprotect
	if encode {
		fn = protect
	}
	var ok uintptr
	if encode {
		ok, _, _ = fn.Call(uintptr(unsafe.Pointer(&input)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&output)))
	} else {
		ok, _, _ = fn.Call(uintptr(unsafe.Pointer(&input)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&output)))
	}
	if ok == 0 {
		return nil, errors.New("nao foi possivel proteger ou ler a credencial do Windows")
	}
	defer localFree.Call(uintptr(unsafe.Pointer(output.data)))
	if output.size > 1024*1024 {
		return nil, errors.New("credencial protegida muito grande")
	}
	raw := unsafe.Slice(output.data, int(output.size))
	return append([]byte(nil), raw...), nil
}
func tunnelConfigPath(a *App) string {
	return filepath.Join(filepath.Dir(a.path), "tunnel-credentials.dpapi")
}
func readTunnelConfig(a *App) (tunnelSettings, error) {
	var c tunnelSettings
	b, e := os.ReadFile(tunnelConfigPath(a))
	if e != nil {
		return c, e
	}
	plaintext, e := dpapi(b, false)
	if e != nil {
		return c, e
	}
	e = json.Unmarshal(plaintext, &c)
	for i := range plaintext {
		plaintext[i] = 0
	}
	return c, e
}
func writeTunnelConfig(a *App, c tunnelSettings) error {
	b, e := json.Marshal(c)
	if e != nil {
		return e
	}
	cipher, e := dpapi(b, true)
	for i := range b {
		b[i] = 0
	}
	if e != nil {
		return e
	}
	path := tunnelConfigPath(a)
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	temp := path + ".new"
	if e = os.WriteFile(temp, cipher, 0600); e != nil {
		return e
	}
	if e = os.Rename(temp, path); e != nil {
		os.Remove(temp)
		return e
	}
	return nil
}
func tunnelBinPath(a *App) string {
	return filepath.Join(filepath.Dir(a.path), "bin", "tunnel-client.exe")
}
func startTunnel(a *App, c tunnelSettings) error {
	if !tunnelIDPattern.MatchString(c.ID) {
		return errors.New("ID invalido: copie o tunnel_... completo da OpenAI")
	}
	if len(c.Key) < 20 || strings.ContainsAny(c.Key, "\r\n\x00 ") {
		return errors.New("chave de execucao invalida; crie uma API key com Tunnels Read + Use")
	}
	exe := tunnelBinPath(a)
	if info, e := os.Stat(exe); e != nil || info.IsDir() {
		return errors.New("cliente de tunel nao instalado; execute novamente o instalador atualizado com internet")
	}
	tunnelRunLock.Lock()
	defer tunnelRunLock.Unlock()
	localTunnel.Lock()
	old := localTunnel.cmd
	done := localTunnel.done
	localTunnel.Unlock()
	if old != nil && old.Process != nil {
		_ = old.Process.Kill()
		if done != nil {
			select {
			case <-done:
			case <-time.After(6 * time.Second):
				return errors.New("cliente anterior nao encerrou")
			}
		}
	}
	cmd := exec.Command(exe, "run", "--mcp.command", buildTunnelCommand(filepath.Join(filepath.Dir(exe), "MCP-PC.exe")), "--health.listen-addr", "127.0.0.1:8766")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	env := []string{}
	for _, item := range os.Environ() {
		upper := strings.ToUpper(strings.SplitN(item, "=", 2)[0])
		if upper != "CONTROL_PLANE_API_KEY" && upper != "CONTROL_PLANE_TUNNEL_ID" && upper != "MCP_COMMAND" {
			env = append(env, item)
		}
	}
	cmd.Env = append(env, "CONTROL_PLANE_API_KEY="+c.Key, "CONTROL_PLANE_TUNNEL_ID="+c.ID)
	logPath := filepath.Join(filepath.Dir(a.path), "tunnel.log")
	f, e := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	cmd.Stdout = f
	cmd.Stderr = f
	if e = cmd.Start(); e != nil {
		f.Close()
		return fmt.Errorf("nao foi possivel iniciar cliente: %w", e)
	}
	localTunnel.Lock()
	localTunnel.cmd = cmd
	finish := make(chan struct{})
	localTunnel.done = finish
	localTunnel.tunnelID = c.ID
	localTunnel.started = time.Now()
	localTunnel.issue = ""
	localTunnel.Unlock()
	go func() {
		err := cmd.Wait()
		f.Close()
		localTunnel.Lock()
		if localTunnel.cmd == cmd {
			localTunnel.cmd = nil
			if err != nil {
				localTunnel.issue = "cliente de tunel encerrou; verifique tunnel.log e a chave de execucao"
			} else {
				localTunnel.issue = "cliente de tunel desconectado"
			}
		}
		localTunnel.Unlock()
		close(finish)
	}()
	return nil
}
func tunnelConnect(a *App, id, key string) error {
	c := tunnelSettings{ID: strings.TrimSpace(id), Key: strings.TrimSpace(key)}
	if !tunnelIDPattern.MatchString(c.ID) {
		return errors.New("ID do tunel invalido")
	}
	if len(c.Key) < 20 || strings.ContainsAny(c.Key, "\r\n\x00 ") {
		return errors.New("chave de execucao invalida")
	}
	if e := startTunnel(a, c); e != nil {
		return e
	}
	return writeTunnelConfig(a, c)
}
func tunnelDisconnect(a *App) error {
	tunnelRunLock.Lock()
	defer tunnelRunLock.Unlock()
	localTunnel.Lock()
	old := localTunnel.cmd
	done := localTunnel.done
	localTunnel.Unlock()
	if old != nil && old.Process != nil {
		_ = old.Process.Kill()
		if done != nil {
			select {
			case <-done:
			case <-time.After(6 * time.Second):
				return errors.New("cliente de tunel nao encerrou")
			}
		}
	}
	err := os.Remove(tunnelConfigPath(a))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func tunnelAutoStart(a *App) {
	c, e := readTunnelConfig(a)
	if e == nil {
		if e = startTunnel(a, c); e != nil {
			localTunnel.Lock()
			localTunnel.issue = e.Error()
			localTunnel.Unlock()
		}
	}
}
func tunnelStatus(a *App) map[string]any {
	localTunnel.Lock()
	c := localTunnel.cmd
	issue := localTunnel.issue
	id := localTunnel.tunnelID
	localTunnel.Unlock()
	saved := false
	if _, e := os.Stat(tunnelConfigPath(a)); e == nil {
		saved = true
	}
	installed := false
	if s, e := os.Stat(tunnelBinPath(a)); e == nil && s.Size() > 500000 {
		installed = true
	}
	status := "desconectado"
	if c != nil {
		status = "iniciando"
		client := http.Client{Timeout: 650 * time.Millisecond}
		r, e := client.Get("http://127.0.0.1:8766/readyz")
		if e == nil {
			r.Body.Close()
			if r.StatusCode >= 200 && r.StatusCode < 300 {
				status = "pronto"
			} else {
				status = "conectando"
			}
		}
	}
	return map[string]any{"installed": installed, "configured": saved, "status": status, "tunnel_id": id, "message": issue}
}
