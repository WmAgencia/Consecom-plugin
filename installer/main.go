//go:build windows

package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed payload/MCP-PC.exe payload/INSTALACAO_E_CONEXAO.md payload/BROWSER_POLICY.md payload/ConsecomMCP.ico
var assets embed.FS

const createNoWindow = 0x08000000

func msg(title, body string, style uintptr) int {
	dll := syscall.NewLazyDLL("user32.dll")
	proc := dll.NewProc("MessageBoxW")
	titleUTF, _ := syscall.UTF16PtrFromString(title)
	bodyUTF, _ := syscall.UTF16PtrFromString(body)
	r, _, _ := proc.Call(0, uintptr(unsafe.Pointer(bodyUTF)), uintptr(unsafe.Pointer(titleUTF)), style)
	return int(r)
}
func quotePS(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func hidden(c *exec.Cmd) *exec.Cmd {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	return c
}

// Create a normal desktop shortcut and a separate startup shortcut using
// the silent/headless mode. The startup shortcut is visible and removable in
// the user's Startup folder; no Windows service or admin task is installed.
func shortcuts(exe, icon string) error {
	script := fmt.Sprintf(`$ErrorActionPreference = 'Stop';
$w = New-Object -ComObject WScript.Shell;
$target = %s;
$icon = %s;
$desk = [Environment]::GetFolderPath('Desktop');
$programs = [Environment]::GetFolderPath('Programs');
$startup = [Environment]::GetFolderPath('Startup');
foreach($folder in @($desk,$programs)) {
  if([string]::IsNullOrWhiteSpace($folder)) { continue }
  $lnk = $w.CreateShortcut((Join-Path $folder 'Consecom Plugin.lnk'));
  $lnk.TargetPath = $target;
  $lnk.WorkingDirectory = Split-Path $target;
  $lnk.Description = 'Consecom Plugin - Abrir painel local';
  $lnk.IconLocation = $icon;
  $old = Join-Path $folder 'MCP Controle PC.lnk';
  if(Test-Path $old){Remove-Item -LiteralPath $old -Force -ErrorAction SilentlyContinue}
  $legacy = Join-Path $folder 'Consecom MCP.lnk';
  if(Test-Path $legacy){Remove-Item -LiteralPath $legacy -Force -ErrorAction SilentlyContinue}
  $lnk.Save();
}
if([string]::IsNullOrWhiteSpace($startup)){ throw 'Pasta Startup nao encontrada' }
$lnk = $w.CreateShortcut((Join-Path $startup 'Consecom Plugin (Segundo Plano).lnk'));
$lnk.TargetPath = $target;
$lnk.Arguments = '--headless';
$lnk.WorkingDirectory = Split-Path $target;
$lnk.Description = 'Consecom Plugin - servidor silencioso no Windows';
$lnk.IconLocation = $icon;
$old = Join-Path $startup 'MCP Controle PC (Segundo Plano).lnk';
if(Test-Path $old){Remove-Item -LiteralPath $old -Force -ErrorAction SilentlyContinue}
$legacy = Join-Path $startup 'Consecom MCP (Segundo Plano).lnk';
if(Test-Path $legacy){Remove-Item -LiteralPath $legacy -Force -ErrorAction SilentlyContinue}
$lnk.WindowStyle = 7;
$lnk.Save();`, quotePS(exe), quotePS(icon))
	c := hidden(exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script))
	o, e := c.CombinedOutput()
	if e != nil {
		return fmt.Errorf("nao foi possivel criar atalhos: %w (%s)", e, o)
	}
	return nil
}

var kernel = syscall.NewLazyDLL("kernel32.dll")
var queryFullProcessImageName = kernel.NewProc("QueryFullProcessImageNameW")
var terminateProcess = kernel.NewProc("TerminateProcess")

// Stop ONLY an older MCP-PC.exe running from the exact previous install path.
// The user is asked first; no unrelated programs are terminated.
func oldPIDAtPath(exe string) ([]int, error) { return pidAtNamedPath(exe, "MCP-PC.exe") }
func pidAtNamedPath(exe string, nameWanted string) ([]int, error) {
	snap, e := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return nil, e
	}
	defer syscall.CloseHandle(snap)
	entry := syscall.ProcessEntry32{Size: uint32(unsafe.Sizeof(syscall.ProcessEntry32{}))}
	if e = syscall.Process32First(snap, &entry); e != nil {
		return nil, e
	}
	var found []int
	for {
		name := syscall.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, nameWanted) {
			const queryLimited = 0x1000
			h, openErr := syscall.OpenProcess(queryLimited, false, entry.ProcessID)
			if openErr == nil {
				var path [32768]uint16
				n := uint32(len(path))
				ok, _, _ := queryFullProcessImageName.Call(uintptr(h), 0, uintptr(unsafe.Pointer(&path[0])), uintptr(unsafe.Pointer(&n)))
				if ok != 0 && strings.EqualFold(filepath.Clean(syscall.UTF16ToString(path[:n])), filepath.Clean(exe)) {
					found = append(found, int(entry.ProcessID))
				}
				syscall.CloseHandle(h)
			}
		}
		entry.Size = uint32(unsafe.Sizeof(syscall.ProcessEntry32{}))
		if e = syscall.Process32Next(snap, &entry); e != nil {
			break
		}
	}
	return found, nil
}
func stopPrevious(exe string, remotelyApproved bool) error {
	pids, e := oldPIDAtPath(exe)
	if e != nil {
		return e
	}
	if len(pids) == 0 {
		return nil
	}
	const yesnoWarning = 0x34 // MB_YESNO | MB_ICONWARNING
	if !remotelyApproved && msg("Consecom Plugin - Atualização", "O servidor MCP anterior esta em execucao.\n\nPara atualizar, preciso encerrá-lo e reiniciar em segundo plano. Solicitações pendentes que ainda não foram aprovadas serão descartadas.\n\nDeseja continuar?", yesnoWarning) != 6 {
		return errors.New("atualizacao cancelada; feche o MCP anterior para atualizar depois")
	}
	for _, pid := range pids {
		const terminate = 0x0001
		h, err := syscall.OpenProcess(terminate, false, uint32(pid))
		if err != nil {
			return fmt.Errorf("sem permissao para finalizar a versao anterior: %w", err)
		}
		ok, _, callErr := terminateProcess.Call(uintptr(h), 0)
		syscall.CloseHandle(h)
		if ok == 0 {
			return fmt.Errorf("nao foi possivel parar a versao anterior: %v", callErr)
		}
	}
	// Nao prosseguir enquanto houver um executavel antigo ainda ativo.
	deadline := time.Now().Add(7 * time.Second)
	for time.Now().Before(deadline) {
		remaining, checkErr := oldPIDAtPath(exe)
		if checkErr != nil { return checkErr }
		if len(remaining) == 0 {
			time.Sleep(250 * time.Millisecond)
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return errors.New("o processo antigo do Consecom nao encerrou; nenhuma substituicao sera efetuada")
}

func stopOwnedTunnel(binDir string) error {
	// O tunnel-client pode religar um servidor MCP antigo. Primeiro pare o
	// supervisor, depois encerre todos os processos MCP-PC.exe.
	tunnelExe := filepath.Join(binDir, "tunnel-client.exe")
	pids, err := pidAtNamedPath(tunnelExe, "tunnel-client.exe")
	if err != nil { return err }
	for _, pid := range pids {
		h, openErr := syscall.OpenProcess(0x0001, false, uint32(pid))
		if openErr != nil { return fmt.Errorf("nao foi possivel parar o tunel do Consecom (PID %d): %w", pid, openErr) }
		ok, _, callErr := terminateProcess.Call(uintptr(h), 0)
		syscall.CloseHandle(h)
		if ok == 0 { return fmt.Errorf("erro ao encerrar tunel do Consecom (PID %d): %v", pid, callErr) }
	}
	deadline := time.Now().Add(7 * time.Second)
	for time.Now().Before(deadline) {
		remaining, err := pidAtNamedPath(tunnelExe, "tunnel-client.exe")
		if err != nil { return err }
		if len(remaining) == 0 { return nil }
		time.Sleep(150 * time.Millisecond)
	}
	return errors.New("cliente de tunel anterior nao encerrou; atualizacao interrompida")
}

// Exigir que a porta esteja realmente livre antes de copiar executaveis.
// Nunca encerra outros aplicativos que nao sejam os binarios do Consecom.
func waitForPortReleased(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
		if err != nil { return nil }
		conn.Close()
		if time.Now().After(deadline) {
			return fmt.Errorf("a porta %s continua ocupada; feche outras instancias MCP antes de atualizar", address)
		}
		time.Sleep(180 * time.Millisecond)
	}
}

// Confirms that the server actually serving the dashboard is THIS version,
// not another MCP instance still listening on the same port.
func verifyServer() error {
	client := http.Client{Timeout: 600 * time.Millisecond}
	var lastVersion string
	for i := 0; i < 60; i++ {
		res, err := client.Get("http://127.0.0.1:8765/health")
		if err == nil {
			var state struct {
				Version string `json:"version"`
				Service string `json:"service"`
			}
			decodeErr := json.NewDecoder(res.Body).Decode(&state)
			res.Body.Close()
			if decodeErr == nil && state.Version == "4.11.0" && state.Service == "consecom-mcp" {
				return nil
			}
			lastVersion = state.Version
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastVersion != "" {
		return fmt.Errorf("porta 8765 ocupada por outra instancia (versao %q); a nova versao nao assumiu o servidor", lastVersion)
	}
	return errors.New("o novo servidor MCP nao iniciou; confira o log em %LOCALAPPDATA%\\ConsecomMCP\\service.log")
}

func rollbackExecutable(exe string) error {
	backup := exe + ".previous"
	data, e := os.ReadFile(backup)
	if e != nil {
		return e
	}
	if e := stopPrevious(exe, true); e != nil {
		return e
	}
	tmp := exe + ".rollback"
	if e := os.WriteFile(tmp, data, 0700); e != nil {
		return e
	}
	if e := os.Rename(tmp, exe); e != nil {
		os.Remove(tmp)
		return e
	}
	if e := hidden(exec.Command(exe, "--headless")).Start(); e != nil {
		return e
	}
	return nil
}

func install(remotelyApproved bool) error {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return fmt.Errorf("LOCALAPPDATA nao encontrado")
	}
	dir := filepath.Join(base, "ConsecomMCP", "bin")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	exe := filepath.Join(dir, "MCP-PC.exe")
	// Ordem obrigatoria: parar supervisor que poderia recriar processos antigos,
	// parar processos do Consecom e confirmar ausencia de listener antes da troca.
	if e := stopOwnedTunnel(dir); e != nil { return e }
	if e := stopPrevious(exe, remotelyApproved); e != nil { return e }
	if e := waitForPortReleased("127.0.0.1:8765", 8*time.Second); e != nil {
		// Nenhum binario foi substituido ainda; nao apagar/alterar dados pessoais.
		return e
	}
	data, e := assets.ReadFile("payload/MCP-PC.exe")
	if e != nil {
		return e
	}
	// Preserve a recovery copy of the last executable for manual rollback.
	// Credentials and authorized roots live outside /bin and are not overwritten.
	if old, readErr := os.ReadFile(exe); readErr == nil {
		_ = os.WriteFile(exe+".previous", old, 0700)
	}
	// Write a temp EXE, then atomically move it into the installation path.
	// If an older instance is still running, we report a clear error.
	tmp := exe + ".new"
	if e = os.WriteFile(tmp, data, 0700); e != nil {
		return e
	}
	if e = os.Rename(tmp, exe); e != nil {
		os.Remove(tmp)
		// Go's os.Rename on Windows generally replaces existing files when unlocked.
		return fmt.Errorf("nao foi possivel atualizar o executavel; feche MCP-PC.exe e tente novamente: %w", e)
	}
	docs, e := assets.ReadFile("payload/INSTALACAO_E_CONEXAO.md")
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(dir, "INSTALACAO_E_CONEXAO.md"), docs, 0600); e != nil {
		return e
	}
	policy, e := assets.ReadFile("payload/BROWSER_POLICY.md")
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(dir, "BROWSER_POLICY.md"), policy, 0600); e != nil {
		return e
	}
	iconData, e := assets.ReadFile("payload/ConsecomMCP.ico")
	if e != nil {
		return e
	}
	iconPath := filepath.Join(dir, "ConsecomMCP.ico")
	if e = os.WriteFile(iconPath, iconData, 0600); e != nil {
		return e
	}
	// Download verified official client on first installation, not from an
	// arbitrary mirror. A blocked network does not damage the working MCP.
	if err := downloadOfficialTunnelClient(dir); err != nil {
		_ = os.WriteFile(filepath.Join(dir, "download-tunnel-error.txt"), []byte(err.Error()), 0600)
		if !remotelyApproved {
			msg("Consecom Plugin - Aviso", "O Consecom foi atualizado, mas o cliente oficial de tunel nao foi baixado. Verifique sua internet e execute o instalador novamente.\n\n"+err.Error(), 0x30)
		}
	} else {
		_ = os.Remove(filepath.Join(dir, "download-tunnel-error.txt"))
	}

	if e = shortcuts(exe, iconPath); e != nil {
		return e
	}
	if e = hidden(exec.Command(exe, "--headless")).Start(); e != nil {
		return fmt.Errorf("instalado, mas nao conseguiu iniciar: %w", e)
	}
	if e = verifyServer(); e != nil {
		// Restore the previously installed version if the newly installed binary
		// fails to start. This does not touch tunnel credentials or config.json.
		rollbackErr := rollbackExecutable(exe)
		if rollbackErr == nil {
			return fmt.Errorf("nova versão falhou (%w); versão anterior restaurada e reiniciada", e)
		}
		return fmt.Errorf("nova versão falhou (%w); recuperação automática falhou: %v. Veja MCP-PC.exe.previous", e, rollbackErr)
	}

	// Unattended updates should not open a browser on a remote Windows session.
	if !remotelyApproved {
		_ = hidden(exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", "http://127.0.0.1:8765/")).Start()
	}
	return nil
}
func main() {
	remoteApproved := flag.Bool("update-approved", false, "Executar atualização solicitada e validada previamente pelo Consecom")
	flag.Parse()
	if e := install(*remoteApproved); e != nil {
		msg("Consecom Plugin - Erro", "A atualizacao nao foi concluida.\n\n"+e.Error(), 0x10)
		os.Exit(1)
	}
	if *remoteApproved {
		return
	}
	msg("Consecom Plugin v4.11.0", "Atualizacao concluida!\n\nO servidor roda em SEGUNDO PLANO, sem abrir CMD ou PowerShell.\nO painel foi aberto no navegador.\nO MCP vai iniciar automaticamente quando voce entrar no Windows.\n\nPara desativar a inicializacao automatica, pressione Win+R e abra shell:startup; apague o atalho 'Consecom Plugin (Segundo Plano)'.\n\nO cliente oficial do tunel sera instalado automaticamente com internet.\nAbra Conectar ao ChatGPT no painel e informe o ID do tunel e a chave de execucao.\n\nSuas pastas e permissoes existentes foram preservadas.\n\nNovo: navegador exclusivo Consecom (Chrome ou Edge), controle de permissoes compacto e indexacao de metadados Markdown do Obsidian, sem abrir o Chrome pessoal.", 0x40)
}
