package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskInboxPersistenceAndBuiltin(t *testing.T) {
	dir := t.TempDir()
	q := newTaskInbox(dir)
	diag, e := q.add("versao", "Diga a versao do plugin")
	if e != nil {
		t.Fatal(e)
	}
	free, e := q.add("livre", "Analise um projeto antes de executar")
	if e != nil {
		t.Fatal(e)
	}
	a, e := NewApp(filepath.Join(dir, "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	q.runBuiltin(a)
	items := newTaskInbox(dir).list()
	if len(items) != 2 {
		t.Fatalf("esperado 2, got %d", len(items))
	}
	if items[0].ID != diag.ID || items[0].Status != "concluida" || !strings.Contains(items[0].Result, "2.6.1") {
		t.Fatalf("versao nao resolvida: %#v", items[0])
	}
	if items[1].ID != free.ID || items[1].Status != "pendente" {
		t.Fatal("pedido livre nao deve ser executado automaticamente")
	}
	if _, e = q.finish(free.ID, "analise finalizada", "concluida"); e != nil {
		t.Fatal(e)
	}
	if _, e = q.finish(free.ID, "duplicado", "concluida"); e == nil {
		t.Fatal("tarefa finalizada deve ser imutavel")
	}
	if _, e = q.add("livre", "x"); e == nil {
		t.Fatal("aceitou prompt insuficiente")
	}
}
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
	for _, name := range []string{"dev_execucao", "tarefas_listar", "tarefas_responder", "browser_abas", "browser_nova_aba", "browser_ativar_aba"} {
		if !present[name] {
			t.Fatal("missing " + name)
		}
	}
	if len(tools()) != 42 {
		t.Fatalf("expected 42 tools got %d", len(tools()))
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
