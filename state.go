package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Root struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Writable bool   `json:"writable"`
}
type Config struct {
	Token         string            `json:"token"`
	Roots         []Root            `json:"roots"`
	DevBindings   []DevServiceBinding `json:"dev_bindings,omitempty"`
	ContextVault string `json:"context_vault,omitempty"`
	ContextSources []ContextSource `json:"context_sources,omitempty"`
	ContextConnections map[string]bool `json:"context_connections,omitempty"`
	ContextNotes []ContextNote `json:"context_notes,omitempty"`
	PreviousPower string            `json:"previous_power,omitempty"`
	ChangedPower  string            `json:"changed_power,omitempty"`
	ApprovalMode  string            `json:"approval_mode,omitempty"`
	Browser       BrowserPreference `json:"browser,omitempty"`
	Updates       UpdateSettings    `json:"updates,omitempty"`
	CloudInference CloudInferenceSettings `json:"cloud_inference,omitempty"`
}
type App struct {
	mu             sync.Mutex
	cfg            Config
	path           string
	actions        map[string]*Pending
	csrf           string
	lastMCP        time.Time
	startedAt      time.Time
	agents         map[string]*Agent
	browserControl BrowserControl
	agentBrowser AgentBrowserSession
	devJobs DevJobManager
	devPreviews DevPreviewManager
	workflows WorkflowManager
	renders RenderManager
	live *LiveLab
}
type ProcessTarget struct {
	PID  int    `json:"pid"`
	Name string `json:"name"`
}
type Pending struct {
	ID          string          `json:"id"`
	Kind        string          `json:"kind"`
	Description string          `json:"description"`
	CreatedAt   time.Time       `json:"created_at"`
	Path        string          `json:"path,omitempty"`
	Content     string          `json:"-"`
	Action      string          `json:"action,omitempty"`
	Processes   []ProcessTarget `json:"processes,omitempty"`
}

func newID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func configLocation() string {
	if v := os.Getenv("MCP_PC_DATA_DIR"); v != "" {
		return filepath.Join(v, "config.json")
	}
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "ConsecomMCP", "config.json")
}
func NewApp(path string) (*App, error) {
	a := &App{path: path, actions: map[string]*Pending{}, csrf: newID(), agents: map[string]*Agent{}, startedAt: time.Now()}
	if b, e := os.ReadFile(path); e == nil {
		if e = json.Unmarshal(b, &a.cfg); e != nil {
			return nil, e
		}
		if len(a.cfg.Token) < 32 {
			return nil, errors.New("token de instalacao invalido")
		}
	} else if errors.Is(e, os.ErrNotExist) {
		a.cfg = Config{Token: newID(), Roots: []Root{}}
		if e := a.saveLocked(); e != nil {
			return nil, e
		}
	} else {
		return nil, e
	}
 // Migration: old personal Chrome defaults must not trigger remote-debugging prompts.
 if p:=exclusiveBrowserPreference(a.cfg.Browser);p.Browser!=""&&p!=a.cfg.Browser{
  a.cfg.Browser=p
  if e:=a.saveLocked();e!=nil{return nil,e}
 }
    // The LAB shares the existing MCP and authentication. Corrupt LAB state
    // must never prevent the normal Consecom server from starting.
    a.live,_=newLiveLab(filepath.Join(filepath.Dir(path),"live-lab-v412.json"))
	return a, nil
}
func (a *App) saveLocked() error {
	if e := os.MkdirAll(filepath.Dir(a.path), 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(a.cfg, "", "  ")
	if e != nil {
		return e
	}
	tmp := a.path + ".tmp"
	if e = os.WriteFile(tmp, b, 0600); e != nil {
		return e
	}
	if e = os.Rename(tmp, a.path); e != nil {
		os.Remove(tmp)
		return e
	}
	return nil
}
func (a *App) roots() []Root {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Root{}, a.cfg.Roots...)
}
func isBroadOrDangerousRoot(path string) bool {
	p := filepath.Clean(path)
	vol := filepath.VolumeName(p)
	if strings.EqualFold(p, vol+string(os.PathSeparator)) {
		return true
	}
	home, _ := os.UserHomeDir()
	if home != "" && strings.EqualFold(p, home) {
		return true
	}
	if len(strings.Trim(p, `\ /`)) < 4 {
		return true
	}
	for _, v := range []string{os.Getenv("WINDIR"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA")} {
		if v != "" {
			protected := filepath.Clean(v)
			if strings.EqualFold(p, protected) || strings.HasPrefix(strings.ToLower(p), strings.ToLower(protected+string(os.PathSeparator))) {
				return true
			}
		}
	}
	return false
}
func (a *App) addRoot(name, path string) (Root, error) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 48 {
		return Root{}, errors.New("nome deve ter entre 2 e 48 caracteres")
	}
	if !filepath.IsAbs(path) {
		return Root{}, errors.New("use um caminho absoluto")
	}
	if strings.HasPrefix(path, `\\`) {
		return Root{}, errors.New("pastas de rede nao sao permitidas")
	}
	full, e := filepath.Abs(filepath.Clean(path))
	if e != nil {
		return Root{}, e
	}
	if isBroadOrDangerousRoot(full) {
		return Root{}, errors.New("escolha uma pasta de projeto especifica; raiz, pasta do usuario e pastas de sistema sao proibidas")
	}
	fi, e := os.Stat(full)
	if e != nil || !fi.IsDir() {
		return Root{}, errors.New("pasta inexistente ou inacessivel")
	}
	if e = ensureNoLinks(full); e != nil {
		return Root{}, e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, r := range a.cfg.Roots {
		if strings.EqualFold(r.Path, full) {
			return Root{}, errors.New("pasta ja autorizada")
		}
	}
	if len(a.cfg.Roots) >= 20 {
		return Root{}, errors.New("limite de 20 pastas")
	}
	r := Root{ID: newID(), Name: name, Path: full}
	a.cfg.Roots = append(a.cfg.Roots, r)
	return r, a.saveLocked()
}
func (a *App) deleteRoot(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, r := range a.cfg.Roots {
		if r.ID == id {
			a.cfg.Roots = append(a.cfg.Roots[:i], a.cfg.Roots[i+1:]...)
			return a.saveLocked()
		}
	}
	return errors.New("pasta nao encontrada")
}
func (a *App) setWritable(id string, flag bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.cfg.Roots {
		if a.cfg.Roots[i].ID == id {
			a.cfg.Roots[i].Writable = flag
			return a.saveLocked()
		}
	}
	return errors.New("pasta nao encontrada")
}
func (a *App) getRoot(id string) (Root, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, r := range a.cfg.Roots {
		if r.ID == id {
			return r, nil
		}
	}
	return Root{}, errors.New("pasta nao autorizada")
}
func (a *App) queue(p *Pending) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, q := range a.actions {
		if time.Since(q.CreatedAt) > 10*time.Minute {
			delete(a.actions, id)
		}
	}
	if len(a.actions) >= 20 {
		return "", errors.New("fila de aprovacoes cheia")
	}
	p.ID = newID()
	p.CreatedAt = time.Now()
	a.actions[p.ID] = p
	return p.ID, nil
}
func (a *App) pendings() []map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	list := []map[string]any{}
	for id, p := range a.actions {
		if time.Since(p.CreatedAt) > 10*time.Minute {
			delete(a.actions, id)
			continue
		}
		m := map[string]any{"id": id, "kind": p.Kind, "description": p.Description, "created_at": p.CreatedAt}
		if p.Kind == "write" {
			m["preview"] = clip(p.Content, 380)
		}
		list = append(list, m)
	}
	return list
}
func (a *App) pop(id string) (*Pending, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	p, ok := a.actions[id]
	if !ok {
		return nil, errors.New("aprovacao inexistente ou ja utilizada")
	}
	delete(a.actions, id)
	if time.Since(p.CreatedAt) > 10*time.Minute {
		return nil, errors.New("aprovacao expirou")
	}
	return p, nil
}
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
func printErr(e error) string {
	if e == nil {
		return ""
	}
	return fmt.Sprint(e)
}
