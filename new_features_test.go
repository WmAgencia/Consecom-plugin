package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeveloperCommandRequiresTrustedRootAndApproval(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	project := filepath.Join(t.TempDir(), "project")
	if e = os.MkdirAll(project, 0700); e != nil {
		t.Fatal(e)
	}
	root, e := a.addRoot("teste_dev", project)
	if e != nil {
		t.Fatal(e)
	}
	v, e := a.developerCommand(root.ID, "npm_build", false)
	if e != nil {
		t.Fatal(e)
	}
	got := v.(map[string]any)
	if got["status"] != "aguardando_aprovacao_local" {
		t.Fatal(got)
	}
	if _, e = a.developerCommand(root.ID, "powershell", false); e == nil {
		t.Fatal("comando livre aceito")
	}
	if _, e = a.developerCommand("ID-INVALIDO", "git_status", false); e == nil {
		t.Fatal("projeto arbitrario aceito")
	}
}
func TestMCPNewToolsAreValid(t *testing.T) {
	present := map[string]bool{}
	for _, tool := range tools() {
		present[tool.Name] = true
		schema := tool.InputSchema
		if _, ok := schema["required"]; ok {
			b, _ := json.Marshal(schema["required"])
			if string(b) == "null" {
				t.Fatal("required null")
			}
		}
	}
	for _, name := range []string{"dev_execucao", "browser_abas", "browser_nova_aba", "browser_ativar_aba"} {
		if !present[name] {
			t.Fatal("missing " + name)
		}
	}
	if len(tools()) != 84 {
		t.Fatalf("expected 84 tools got %d", len(tools()))
	}
}
func TestBrowserOwnedTabs(t *testing.T) {
	session := &BrowserSession{}
	ownedTargetAdd(session, "task-a")
	ownedTargetAdd(session, "task-b")
	ownedTargetAdd(session, "task-a")
	if len(session.OwnedTargets) != 2 {
		t.Fatal(session.OwnedTargets)
	}
	session.Target = "task-a"
	ownedTargetRemove(session, "task-a")
	if session.Target != "task-b" {
		t.Fatalf("expected b got %s", session.Target)
	}
}
