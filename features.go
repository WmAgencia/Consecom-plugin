package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Approval modes are preferences of the LOCAL server only. They never override
// host-side consent or MCP client authorizations.
var modes = map[string]bool{"always": true, "category": true, "maximum": true}

func (a *App) mode() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if modes[a.cfg.ApprovalMode] {
		return a.cfg.ApprovalMode
	}
	return "always"
}
func (a *App) setMode(mode, acknowledgement string) error {
	if !modes[mode] {
		return errors.New("modo desconhecido")
	}
	if mode == "maximum" && acknowledgement != "LIBERAR" {
		return errors.New("para habilitar aprovacao maxima, confirme LIBERAR no painel local")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.ApprovalMode = mode
	return a.saveLocked()
}
func (a *App) canAuto(kind string) bool {
	mode := a.mode()
	if mode == "maximum" {
		return true
	}
	return mode == "category" && kind == "write"
}


// approvalPolicy reports the effective server-side setting. It is read-only:
// client/platform permissions, Windows UAC, protected paths and project roots
// are enforced independently of the locally selected approval mode.
func (a *App) approvalPolicy() map[string]any {
	mode := a.mode()
	roots := a.roots()
	return map[string]any{
		"mode": mode,
		"local_reprompt_disabled": mode == "maximum",
		"authorized_project_roots": len(roots),
		"project_roots_still_required": true,
		"remote_updates_separate_opt_in": true,
		"remote_updates_authorized": a.updateStatus().RemoteApproved,
		"platform_and_windows_permissions_still_apply": true,
		"note": "Aprovacao maxima elimina prompts adicionais do Consecom para acoes locais permitidas, mas nao concede acesso a pastas nao cadastradas, nao substitui permissao do cliente MCP/ChatGPT nem desativa UAC ou as regras de protecao.",
	}
}

func (a *App) markMCP() { a.mu.Lock(); a.lastMCP = time.Now(); a.mu.Unlock() }
func (a *App) statusExtra() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	mode := a.cfg.ApprovalMode
	if !modes[mode] {
		mode = "always"
	}
	connected := !a.lastMCP.IsZero() && time.Since(a.lastMCP) < 5*time.Minute
	return map[string]any{"mode": mode, "mcp_recent": connected, "mcp_last": a.lastMCP.Format(time.RFC3339), "version": consecomVersion, "started_at": a.startedAt.Format(time.RFC3339)}
}

// Even maximum approval does not expose credential stores, system files,
// secrets, symlinks/junctions, network shares or unrestricted shell execution.
func safeFullPath(abs string, forWrite bool) (string, error) {
	if !filepath.IsAbs(abs) || strings.HasPrefix(abs, `\\`) {
		return "", errors.New("caminho local absoluto necessario (sem pastas de rede)")
	}
	clean := filepath.Clean(abs)
	if isProtectedAbsolute(clean) {
		return "", errors.New("pasta de sistema ou area privada protegida")
	}
	for _, part := range strings.FieldsFunc(clean, func(r rune) bool { return r == '/' || r == '\\' }) {
		if blockedPart(part) {
			return "", fmt.Errorf("parte confidencial bloqueada: %s", part)
		}
	}
	if err := ensureNoLinks(clean); err != nil {
		return "", err
	}
	if forWrite && !permittedExt[strings.ToLower(filepath.Ext(clean))] {
		return "", errors.New("extensao nao permitida para escrita")
	}
	return clean, nil
}
func isProtectedAbsolute(path string) bool {
	clean := strings.ToLower(filepath.Clean(path))
	vol := filepath.VolumeName(clean)
	if vol != "" && (clean == vol+string(os.PathSeparator) || clean == vol) {
		return true
	}
	for _, candidate := range []string{os.Getenv("WINDIR"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA"), os.Getenv("ProgramData")} {
		if candidate != "" {
			v := strings.ToLower(filepath.Clean(candidate))
			if clean == v || strings.HasPrefix(clean, v+string(os.PathSeparator)) {
				return true
			}
		}
	}
	return false
}

// Agent records represent REAL activity reported by a tool client. No demo
// agents or fabricated task progress are inserted in production.
type Agent struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Source    string      `json:"source"`
	Status    string      `json:"status"`
	Task      string      `json:"task"`
	UpdatedAt time.Time   `json:"updated_at"`
	StartedAt time.Time   `json:"started_at"`
	Logs      []AgentLine `json:"logs"`
}
type AgentLine struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}

func validAgentStatus(s string) bool {
	return s == "running" || s == "waiting" || s == "done" || s == "error"
}
func (a *App) registerAgent(name, source, task string) (Agent, error) {
	name = strings.TrimSpace(name)
	task = strings.TrimSpace(task)
	source = strings.TrimSpace(source)
	if len(name) < 2 || len(name) > 64 || len(source) > 60 || len(task) > 240 {
		return Agent{}, errors.New("nome, origem ou tarefa fora do limite")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.agents) >= 36 {
		return Agent{}, errors.New("limite de 36 agentes simultaneos")
	}
	now := time.Now()
	p := &Agent{ID: newID(), Name: name, Source: source, Task: task, Status: "running", StartedAt: now, UpdatedAt: now, Logs: []AgentLine{}}
	a.agents[p.ID] = p
	return *p, nil
}
func (a *App) updateAgent(id, status, task, line string) (Agent, error) {
	if !validAgentStatus(status) || len(task) > 240 || len(line) > 2000 {
		return Agent{}, errors.New("status ou texto invalido")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	p, ok := a.agents[id]
	if !ok {
		return Agent{}, errors.New("agente nao encontrado")
	}
	p.Status = status
	p.Task = task
	p.UpdatedAt = time.Now()
	if strings.TrimSpace(line) != "" {
		p.Logs = append(p.Logs, AgentLine{At: p.UpdatedAt, Text: line})
		if len(p.Logs) > 80 {
			p.Logs = p.Logs[len(p.Logs)-80:]
		}
	}
	return *p, nil
}
func (a *App) getAgents() []Agent {
	a.mu.Lock()
	defer a.mu.Unlock()
	agents := make([]Agent, 0, len(a.agents))
	for _, p := range a.agents {
		agents = append(agents, *p)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].StartedAt.Before(agents[j].StartedAt) })
	return agents
}
