package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApprovalModesAndPersistence(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	if a.mode() != "always" {
		t.Fatal("default must be safe")
	}
	if e = a.setMode("maximum", ""); e == nil {
		t.Fatal("max needs local acknowledgement")
	}
	if e = a.setMode("maximum", "LIBERAR"); e != nil {
		t.Fatal(e)
	}
	if !a.canAuto("close") || !a.canAuto("write") || !a.canAuto("power") {
		t.Fatal("maximum not applying")
	}
	b, e := NewApp(a.path)
	if e != nil || b.mode() != "maximum" {
		t.Fatal("not persisted", e)
	}
	if e = a.setMode("category", ""); e != nil {
		t.Fatal(e)
	}
	if !a.canAuto("write") || a.canAuto("close") || a.canAuto("power") {
		t.Fatal("category invalid")
	}
	if e = a.setMode("always", ""); e != nil || a.canAuto("write") {
		t.Fatal("safe mode invalid", e)
	}
}
func TestAgentLogsRealOnly(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	if len(a.getAgents()) != 0 {
		t.Fatal("fabricated agents")
	}
	ag, e := a.registerAgent("Codex", "cli", "analisar projeto")
	if e != nil {
		t.Fatal(e)
	}
	ag, e = a.updateAgent(ag.ID, "running", "verificando testes", "tests passing")
	if e != nil || len(ag.Logs) != 1 {
		t.Fatal("logs invalid", e)
	}
	if _, e = a.updateAgent(ag.ID, "hacked", "", ""); e == nil {
		t.Fatal("unknown status allowed")
	}
}
func TestLocalApiPolicyAndAuth(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	server := a.handler()
	req := httptest.NewRequest("POST", "http://127.0.0.1:8765/api/policy", strings.NewReader(`{"mode":"maximum","confirm":"LIBERAR"}`))
	req.Header.Set("Content-Type", "application/json")
	bad := httptest.NewRecorder()
	server.ServeHTTP(bad, req)
	if bad.Code != 403 {
		t.Fatalf("expected 403 got %d", bad.Code)
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1:8765/api/policy", strings.NewReader(`{"mode":"maximum","confirm":"LIBERAR"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Local-Key", a.csrf)
	good := httptest.NewRecorder()
	server.ServeHTTP(good, req)
	if good.Code != 200 || a.mode() != "maximum" {
		t.Fatalf("policy failed %d %s", good.Code, good.Body.String())
	}
	req = httptest.NewRequest("GET", "http://malicious.example/health", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatal("DNS rebinding host check failed")
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1:8765/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatal("unauthenticated MCP permitted")
	}
	req = httptest.NewRequest("POST", "http://127.0.0.1:8765/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.cfg.Token)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	var parsed map[string]any
	if json.Unmarshal(rec.Body.Bytes(), &parsed) != nil || parsed["result"] == nil {
		t.Fatal("authorized MCP failed", rec.Code)
	}
}
func TestFullSafeWrites(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	_ = a.setMode("maximum", "LIBERAR")
	dir := t.TempDir()
	f := filepath.Join(dir, "hello.txt")
	if _, e = a.writeFullApproved(&Pending{Path: f, Content: "one"}); e != nil {
		t.Fatal(e)
	}
	if _, e = a.writeFullApproved(&Pending{Path: f, Content: "two"}); e != nil {
		t.Fatal(e)
	}
	data, e := os.ReadFile(f)
	if e != nil || string(data) != "two" {
		t.Fatal("write mismatch", e)
	}
	backups, _ := filepath.Glob(f + ".bak-*")
	if len(backups) != 1 {
		t.Fatalf("backups = %d", len(backups))
	}
	if _, e = safeFullPath(filepath.Join(dir, ".env"), false); e == nil {
		t.Fatal("secret allowed")
	}
	if _, e = safeFullPath("../something.txt", false); e == nil {
		t.Fatal("relative allowed")
	}
}
func TestNavigationOnlyTwo(t *testing.T) {
	if !strings.Contains(uiHTML, "Laboratório de Agentes") || !strings.Contains(uiHTML, "Visualizar") {
		t.Fatal("missing lab")
	}
	if strings.Contains(uiHTML, "navProjects") || strings.Contains(uiHTML, "Game Booster") || strings.Contains(uiHTML, "Terminal Restrito") {
		t.Fatal("unexpected extra UI")
	}
}

var _ = http.MethodGet
