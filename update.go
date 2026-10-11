package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const consecomVersion = "4.9.0"
// A única origem autorizada para instalar atualizações do Consecom Plugin.
const officialUpdateRepo = "WmAgencia/Consecom-plugin"
const updateMaxBytes int64 = 64 << 20

var validRepo = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,80}/[A-Za-z0-9_.-]{1,100}$`)
var validSha = regexp.MustCompile(`^[0-9a-f]{64}$`)
var validVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

type UpdateSettings struct {
	Repo           string `json:"repo"`
	RemoteApproved bool   `json:"remote_approved"`
}
type UpdateManifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}
type UpdateStatus struct {
	Version        string `json:"version"`
	Repo           string `json:"repo"`
	RemoteApproved bool   `json:"remote_approved"`
	Configured     bool   `json:"configured"`
	ManifestURL    string `json:"manifest_url,omitempty"`
}

func updateManifestURL(repo string) string {
	return "https://github.com/" + repo + "/releases/latest/download/consecom-update.json"
}
func validateUpdateRepo(repo string) error {
	if !validRepo.MatchString(repo) || strings.Contains(repo, "..") || strings.HasPrefix(repo, ".") || strings.Contains(repo, "/.") {
		return errors.New("repositório GitHub inválido: use proprietario/repositorio")
	}
	return nil
}
func compareSemver(a, b string) (int, error) {
	if !validVersion.MatchString(a) || !validVersion.MatchString(b) {
		return 0, errors.New("versão inválida: esperado X.Y.Z")
	}
	ap, bp := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		x, e := strconv.ParseUint(ap[i], 10, 32)
		if e != nil {
			return 0, e
		}
		y, e := strconv.ParseUint(bp[i], 10, 32)
		if e != nil {
			return 0, e
		}
		if x < y {
			return -1, nil
		}
		if x > y {
			return 1, nil
		}
	}
	return 0, nil
}
func validateUpdateManifest(repo string, m UpdateManifest) error {
	if err := validateUpdateRepo(repo); err != nil {
		return err
	}
	if _, err := compareSemver(m.Version, consecomVersion); err != nil {
		return err
	}
	if !validSha.MatchString(m.SHA256) {
		return errors.New("SHA-256 inválido")
	}
	u, err := url.Parse(m.URL)
	if err != nil {
		return err
	}
	if u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "github.com") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Port() != "" {
		return errors.New("instalador deve estar em GitHub Releases por HTTPS")
	}
	prefix := "/" + repo + "/releases/download/"
	if !strings.HasPrefix(u.EscapedPath(), prefix) {
		return errors.New("instalador não pertence ao repositório confiável")
	}
	versionPrefix := prefix + "v" + m.Version + "/"
	if !strings.HasPrefix(u.EscapedPath(), versionPrefix) {
		return errors.New("o instalador não pertence à versão declarada no manifesto")
	}
	name := filepath.Base(u.Path)
	if !strings.HasSuffix(strings.ToLower(name), ".exe") || len(name) > 160 || strings.Contains(name, "%") {
		return errors.New("a atualização deve ser um instalador .exe")
	}
	return nil
}
func (a *App) updateStatus() UpdateStatus {
 a.mu.Lock()
 s := a.cfg.Updates
 a.mu.Unlock()
 // Pré-autorizações antigas de outro repositório não valem para a origem oficial.
 approved := s.RemoteApproved && strings.EqualFold(s.Repo, officialUpdateRepo)
 return UpdateStatus{Version: consecomVersion, Repo: officialUpdateRepo, RemoteApproved: approved, Configured: true, ManifestURL: updateManifestURL(officialUpdateRepo)}
}
func (a *App) updateSettings(s UpdateSettings, confirm string) error {
 s.Repo = strings.TrimSpace(s.Repo)
 if s.Repo != "" && !strings.EqualFold(s.Repo, officialUpdateRepo) {
   return errors.New("somente o repositório oficial WmAgencia/Consecom-plugin é permitido")
 }
 s.Repo = officialUpdateRepo
 if s.RemoteApproved && confirm != "CONFIO NAS ATUALIZACOES" {
   return errors.New("para liberar atualizações remotas, confirme CONFIO NAS ATUALIZACOES no painel local")
 }
 a.mu.Lock()
 defer a.mu.Unlock()
 a.cfg.Updates = s
 return a.saveLocked()
}

var updateHTTPClient = &http.Client{Timeout: 90 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if len(via) > 7 {
		return errors.New("redirecionamentos em excesso")
	}
	if req.URL.Scheme != "https" || req.URL.User != nil {
		return errors.New("redirecionamento HTTP não permitido")
	}
	host := strings.ToLower(req.URL.Hostname())
	switch host {
	case "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return nil
	}
	return errors.New("redirecionamento para host não confiável: " + host)
}}

func fetchUpdateManifest(ctx context.Context, repo string) (UpdateManifest, error) {
	var m UpdateManifest
	if err := validateUpdateRepo(repo); err != nil {
		return m, err
	}
	req, e := http.NewRequestWithContext(ctx, "GET", updateManifestURL(repo), nil)
	if e != nil {
		return m, e
	}
	resp, e := updateHTTPClient.Do(req)
	if e != nil {
		return m, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return m, fmt.Errorf("manifesto de atualização indisponível (HTTP %d); publique consecom-update.json na release mais recente", resp.StatusCode)
	}
	body, e := io.ReadAll(io.LimitReader(resp.Body, 4097))
	if e != nil {
		return m, e
	}
	if len(body) > 4096 {
		return m, errors.New("manifesto muito grande")
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return m, err
	}
	if err := validateUpdateManifest(repo, m); err != nil {
		return m, err
	}
	return m, nil
}
func (a *App) updateCheck() (map[string]any, error) {
	s := a.updateStatus()
	if !s.Configured {
		return nil, errors.New("configure primeiro um repositório GitHub de releases no painel local do Consecom")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	m, e := fetchUpdateManifest(ctx, s.Repo)
	if e != nil {
		return nil, e
	}
	cmp, e := compareSemver(m.Version, consecomVersion)
	if e != nil {
		return nil, e
	}
	return map[string]any{"installed": consecomVersion, "available": m.Version, "update_available": cmp > 0, "repo": s.Repo, "remote_approved": s.RemoteApproved}, nil
}
func updatesDir(a *App) string { return filepath.Join(filepath.Dir(a.path), "updates") }
func stageUpdate(a *App, m UpdateManifest) (string, error) {
	dir := updatesDir(a)
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	// Do not overwrite the currently running installer or any arbitrary local path.
	target := filepath.Join(dir, "consecom-"+m.Version+"-"+m.SHA256[:12]+".exe")
	if err := verifyFileSHA(target, m.SHA256); err == nil {
		return target, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, "GET", m.URL, nil)
	if e != nil {
		return "", e
	}
	res, e := updateHTTPClient.Do(req)
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("download falhou: HTTP %d", res.StatusCode)
	}
	if res.ContentLength > updateMaxBytes {
		return "", errors.New("instalador acima de 64 MiB")
	}
	f, e := os.CreateTemp(dir, ".consecom-download-*.tmp")
	if e != nil {
		return "", e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	h := sha256.New()
	n, e := io.Copy(io.MultiWriter(f, h), io.LimitReader(res.Body, updateMaxBytes+1))
	closeErr := f.Close()
	if e != nil {
		return "", e
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n > updateMaxBytes {
		return "", errors.New("instalador excede limite de 64 MiB")
	}
	got := hex.EncodeToString(h.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(got), []byte(m.SHA256)) != 1 {
		return "", errors.New("SHA-256 divergente; atualização bloqueada")
	}
	if e := os.Rename(tmp, target); e != nil {
		return "", e
	}
	return target, nil
}
func verifyFileSHA(path, expected string) error {
	if !validSha.MatchString(expected) {
		return errors.New("SHA-256 inválido")
	}
	f, e := os.Open(path)
	if e != nil {
		return e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return e
	}
	if info.Size() > updateMaxBytes {
		return errors.New("arquivo grande demais")
	}
	h := sha256.New()
	if _, e = io.Copy(h, io.LimitReader(f, updateMaxBytes+1)); e != nil {
		return e
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return errors.New("checksum não confere")
	}
	return nil
}
func runInstaller(absoluteInstaller, expectedSHA string) error {
	if runtime.GOOS != "windows" {
		return errors.New("atualizações só são executadas no Windows")
	}
	if e := verifyFileSHA(absoluteInstaller, expectedSHA); e != nil {
		return e
	}
	// Never invoke a shell; the detached signed-off installer performs the swap
	// after the current MCP shuts down. Credentials/config.json are untouched.
	cmd := exec.Command(absoluteInstaller, "--update-approved")
	if err := platformHiddenInstaller(cmd); err != nil {
		return err
	}
	return cmd.Start()
}
func (a *App) updateRequest() (map[string]any, error) {
 return a.updateRequestWithMode(false)
}
// Somente o clique no painel local pode instalar diretamente; MCP remoto exige permissão.
func (a *App) updateRequestLocal() (map[string]any, error) {
 return a.updateRequestWithMode(true)
}
func (a *App) updateRequestWithMode(localClick bool) (map[string]any, error) {
	s := a.updateStatus()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	m, e := fetchUpdateManifest(ctx, s.Repo)
	if e != nil {
		return nil, e
	}
	cmp, e := compareSemver(m.Version, consecomVersion)
	if e != nil {
		return nil, e
	}
	if cmp <= 0 {
		return map[string]any{"status": "sem_atualizacao", "installed": consecomVersion, "latest": m.Version}, nil
	}
	path, e := stageUpdate(a, m)
	if e != nil {
		return nil, e
	}
	// A remote opt-in belongs to this locally configured trusted repository ONLY.
	if localClick || s.RemoteApproved {
		if err := runInstaller(path, m.SHA256); err != nil {
			return nil, err
		}
		audit("atualizacao iniciada (clique local=%t): %s para %s", localClick, s.Repo, m.Version)
		return map[string]any{"status": "iniciada", "version": m.Version, "message": "instalador validado e iniciado; tunel pode cair temporariamente"}, nil
	}
	id, e := a.queue(&Pending{Kind: "update", Description: "Instalar Consecom " + m.Version + " do repositório confiável " + s.Repo + " (SHA-256 " + m.SHA256[:16] + "…)", Path: path, Content: m.SHA256})
	if e != nil {
		return nil, e
	}
	return map[string]any{"status": "aguardando_aprovacao_local", "version": m.Version, "approval_id": id, "message": "Abra o painel Consecom no Windows e aprove esta atualização"}, nil
}
func (a *App) approveUpdate(p *Pending) (string, error) {
	// Reject stale jobs after a local repo permission change.
	s := a.updateStatus()
	if !strings.Contains(p.Description, " "+s.Repo+" (") {
		return "", errors.New("repositório confiável alterado; solicite novamente")
	}
	if err := runInstaller(p.Path, p.Content); err != nil {
		return "", err
	}
	return "Atualizador iniciado. O túnel pode desconectar durante a instalação e reconectar quando o servidor reiniciar.", nil
}
