package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTestApp(t *testing.T) (*App, string) {
	t.Helper()
	dir := t.TempDir()
	a, e := NewApp(filepath.Join(dir, "settings", "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	r := filepath.Join(dir, "Projeto")
	if e = os.MkdirAll(r, 0700); e != nil {
		t.Fatal(e)
	}
	return a, r
}
func TestBroadRootRejected(t *testing.T) {
	a, _ := makeTestApp(t)
	home, _ := os.UserHomeDir()
	if home != "" {
		if _, e := a.addRoot("Minha casa", home); e == nil {
			t.Fatal("home should be rejected")
		}
	}
}
func TestAllowedPathsAndBlockedSecrets(t *testing.T) {
	a, root := makeTestApp(t)
	r, e := a.addRoot("Projeto teste", root)
	if e != nil {
		t.Fatal(e)
	}
	os.Mkdir(filepath.Join(root, "src"), 0700)
	os.WriteFile(filepath.Join(root, "src", "x.ts"), []byte("OK"), 0600)
	os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET"), 0600)
	if _, e := safePath(r, "../outside.txt", false); e == nil {
		t.Fatal("traversal allowed")
	}
	if _, e := safePath(r, ".env", false); e == nil {
		t.Fatal("secret path allowed")
	}
	if _, e := safePath(r, "src/x.ts", false); e != nil {
		t.Fatal(e)
	}
	if _, e := safePath(r, "src/x.ts", true); e == nil {
		t.Fatal("write enabled by default")
	}
	os.Symlink(t.TempDir(), filepath.Join(root, "link"))
	if _, e := safePath(r, "link/test.md", false); e == nil {
		t.Fatal("symlink escape allowed")
	}
	if _, e := a.dispatch("ler_arquivo", json.RawMessage(`{"root_id":"`+r.ID+`","caminho":"src/x.ts"}`)); e != nil {
		t.Fatal(e)
	}
}
func TestApprovalRequiredForWrites(t *testing.T) {
	a, root := makeTestApp(t)
	r, e := a.addRoot("Projeto Teste", root)
	if e != nil {
		t.Fatal(e)
	}
	a.setWritable(r.ID, true)
	req, _ := json.Marshal(map[string]string{"root_id": r.ID, "caminho": "notas.md", "conteudo": "novo texto"})
	res, e := a.dispatch("solicitar_gravacao_arquivo", req)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(root, "notas.md")); !os.IsNotExist(e) {
		t.Fatal("write happened without approval")
	}
	id := res.(map[string]any)["solicitacao"].(string)
	if _, e := a.approve(id); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(root, "notas.md"))
	if e != nil || string(b) != "novo texto" {
		t.Fatalf("bad file: %s, %v", b, e)
	}
	if _, e := a.approve(id); e == nil {
		t.Fatal("replayed approval")
	}
	os.WriteFile(filepath.Join(root, "notas.md"), []byte("texto velho"), 0600)
	res, e = a.dispatch("solicitar_gravacao_arquivo", req)
	if e != nil {
		t.Fatal(e)
	}
	_, e = a.approve(res.(map[string]any)["solicitacao"].(string))
	if e != nil {
		t.Fatal(e)
	}
	matches, _ := filepath.Glob(filepath.Join(root, "notas.md.bak-*"))
	if len(matches) != 1 {
		t.Fatalf("expected backup, got %v", matches)
	}
}
func TestAuthAndMCPTools(t *testing.T) {
	a, _ := makeTestApp(t)
	srv := httptest.NewServer(a.handler())
	defer srv.Close()
	payload := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req0, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(payload))
	req0.Host = listenAddr
	req0.Header.Set("Content-Type", "application/json")
	r, e := http.DefaultClient.Do(req0)
	if e != nil {
		t.Fatal(e)
	}
	if r.StatusCode != 401 {
		t.Fatalf("expected 401 got %d", r.StatusCode)
	}
	r.Body.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(payload))
	req.Host = listenAddr
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.Token)
	r, e = http.DefaultClient.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	if r.StatusCode != 200 || !strings.Contains(string(b), "listar_arquivos") {
		t.Fatalf("bad MCP response %d %s", r.StatusCode, b)
	}
	req2, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(payload))
	req2.Host = listenAddr
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+a.cfg.Token)
	req2.Header.Set("Origin", "https://evil.example")
	r2, e := http.DefaultClient.Do(req2)
	if e != nil {
		t.Fatal(e)
	}
	defer r2.Body.Close()
	if r2.StatusCode != 403 {
		t.Fatal("Origin not rejected")
	}
}
func TestCommandsAreAllowlisted(t *testing.T) {
	a, _ := makeTestApp(t)
	req := json.RawMessage(`{"acao":"powershell -Command rm -r C:\\"}`)
	_, e := a.dispatch("terminal_restrito", req)
	if e == nil {
		t.Fatal("arbitrary command accepted")
	}
}
