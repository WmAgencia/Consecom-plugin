package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// All development commands are fixed argv (no shell, redirections or arbitrary commands).
// npm scripts may execute project code. In normal/category mode they require
// local approval; in maximum mode the user's explicit local setting waives
// only this additional Consecom prompt, not project scope or client policy.
var developerActions = map[string][]string{
	"git_status":    {"git", "status", "--short", "--branch"},
	"git_branch":    {"git", "branch", "--show-current"},
	"git_diff_stat": {"git", "diff", "--stat", "--"},
	"git_log":       {"git", "log", "-5", "--oneline"},
	"node_version":  {"node", "--version"},
	"npm_version":   {"npm", "--version"},
	"npm_test":      {"npm", "run", "test"},
	"npm_build":     {"npm", "run", "build"},
	"npm_lint":      {"npm", "run", "lint"},
}

func npmScriptAction(action string) bool {
	return strings.HasPrefix(action, "npm_") && action != "npm_version"
}
func (a *App) developerCommand(rootID, action string, approved bool) (any, error) {
	args, found := developerActions[action]
	if !found {
		return nil, errors.New("acao de desenvolvimento nao permitida")
	}
	root, e := a.getRoot(rootID)
	if e != nil {
		return nil, e
	}
	if isBroadOrDangerousRoot(root.Path) {
		return nil, errors.New("pasta de projeto nao autorizada")
	}
	if e = ensureNoLinks(root.Path); e != nil {
		return nil, e
	}
	if npmScriptAction(action) && !approved && a.mode() != "maximum" {
		id, er := a.queue(&Pending{Kind: "dev", Description: "Executar script do projeto " + root.Name + " em " + root.Path + ": " + action + " (scripts npm podem executar codigo arbitrario)", Path: root.Path, Action: action})
		if er != nil {
			return nil, er
		}
		return map[string]any{"status": "aguardando_aprovacao_local", "approval_id": id}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	bin := args[0]
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" && bin == "npm" {
		// Windows npm is npm.cmd, and CreateProcess cannot execute batch files
		// without cmd.exe. The command here consists entirely of fixed action
		// strings from developerActions, never user-supplied shell input.
		windowsCmd := filepath.Join(os.Getenv("SYSTEMROOT"), "System32", "cmd.exe")
		if os.Getenv("SYSTEMROOT") == "" {
			windowsCmd = "cmd.exe"
		}
		cmd = exec.CommandContext(ctx, windowsCmd, "/D", "/S", "/C", strings.Join(args, " "))
	} else {
		cmd = exec.CommandContext(ctx, bin, args[1:]...)
	}
	cmd.Dir = root.Path
	cmd.Env = filteredDeveloperEnvironment()
	// No environment variables with secrets, tokens or credentials are forwarded.
	output, e := cmd.CombinedOutput()
	const maxOutput = 18 * 1024
	out := string(output)
	truncated := len(out) > maxOutput
	if truncated {
		out = out[:maxOutput]
	}
	result := map[string]any{"acao": action, "projeto": root.Name, "saida": out, "truncado": truncated, "aprovado_localmente": approved, "modo_aprovacao": a.mode(), "sem_prompt_local_modo_maximo": !approved && a.mode() == "maximum"}
	if ctx.Err() != nil {
		return result, fmt.Errorf("comando terminou por limite de 45 segundos: %w", ctx.Err())
	}
	if e != nil {
		return result, fmt.Errorf("comando falhou: %v; saida: %s", e, out)
	}
	return result, nil
}
func filteredDeveloperEnvironment() []string {
	keep := map[string]bool{"PATH": true, "SYSTEMROOT": true, "WINDIR": true, "COMSPEC": true, "PATHEXT": true, "TEMP": true, "TMP": true, "APPDATA": true, "LOCALAPPDATA": true, "USERPROFILE": true, "HOMEDRIVE": true, "HOMEPATH": true, "PROGRAMFILES": true, "PROGRAMFILES(X86)": true}
	result := []string{}
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if keep[strings.ToUpper(key)] {
			result = append(result, item)
		}
	}
	return result
}
func (a *App) approveDeveloper(p *Pending) (string, error) {
	if !npmScriptAction(p.Action) {
		return "", errors.New("acao de desenvolvimento inválida")
	}
	// Revalidate that the root remains authorized before executing.
	roots := a.roots()
	for _, root := range roots {
		if filepath.Clean(root.Path) == filepath.Clean(p.Path) {
			r, e := a.developerCommand(root.ID, p.Action, true)
			if e != nil {
				return "", e
			}
			return fmt.Sprintf("Script executado: %v", r), nil
		}
	}
	return "", errors.New("pasta nao esta mais autorizada")
}
