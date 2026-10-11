package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func opArgs(m map[string]string) json.RawMessage { b, _ := json.Marshal(m); return b }
func newTestApp(t *testing.T) *App {
	t.Helper()
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func TestFileOperationsRoundTripAndPersistence(t *testing.T) {
	a := newTestApp(t)
	base := t.TempDir()
	folder := filepath.Join(base, "mcp-qa")
	// Policy isn't bypassed by a remote tool call.
	if _, e := a.dispatch("criar_pasta_pc", opArgs(map[string]string{"caminho_absoluto": folder})); e == nil {
		t.Fatal("write worked without local approval")
	}
	if e := a.setMode("maximum", "LIBERAR"); e != nil {
		t.Fatal(e)
	}
	if _, e := a.dispatch("criar_pasta_pc", opArgs(map[string]string{"caminho_absoluto": folder})); e != nil {
		t.Fatal(e)
	}
	initial := filepath.Join(folder, "original.txt")
	if _, e := a.dispatch("gravar_arquivo_pc", opArgs(map[string]string{"caminho_absoluto": initial, "conteudo": "version 1"})); e != nil {
		t.Fatal(e)
	}
	if _, e := a.dispatch("gravar_arquivo_pc", opArgs(map[string]string{"caminho_absoluto": initial, "conteudo": "version 2"})); e != nil {
		t.Fatal(e)
	}
	before, e := a.recoveryList()
	if e != nil || len(before) != 1 || before[0].Type != "backup" {
		t.Fatalf("backup before overwrite missing: %#v %v", before, e)
	}
	moved := filepath.Join(folder, "renamed.txt")
	if _, e := a.dispatch("mover_arquivo_pc", opArgs(map[string]string{"origem": initial, "destino": moved})); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(initial); !os.IsNotExist(e) {
		t.Fatal("origin not removed")
	}
	if b, _ := os.ReadFile(moved); string(b) != "version 2" {
		t.Fatal("move corrupted content")
	}
	if _, e := a.dispatch("excluir_arquivo_pc", opArgs(map[string]string{"caminho_absoluto": moved})); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(moved); !os.IsNotExist(e) {
		t.Fatal("delete not effective")
	}
	entries, e := a.recoveryList()
	if e != nil || len(entries) != 3 {
		t.Fatalf("expected 3 entries got %v: %v", len(entries), e)
	}
	var trashed string
	for _, ent := range entries {
		if ent.Type == "trash" {
			trashed = ent.ID
		}
	}
	if trashed == "" {
		t.Fatal("trash version missing")
	}
	if _, e := a.dispatch("restaurar_recuperacao_pc", opArgs(map[string]string{"id": trashed})); e != nil {
		t.Fatal(e)
	}
	if b, _ := os.ReadFile(moved); string(b) != "version 2" {
		t.Fatal("restoration mismatch")
	}
	// Restart and ensure index survives for mobile/remote recovery.
	second, e := NewApp(a.path)
	if e != nil {
		t.Fatal(e)
	}
	records, e := second.recoveryList()
	if e != nil || len(records) != 3 {
		t.Fatalf("index lost: %d %v", len(records), e)
	}
	if _, e = second.restoreRecovery(trashed); e == nil {
		t.Fatal("double restoration allowed")
	}
	if _, e = a.dispatch("excluir_pasta_vazia_pc", opArgs(map[string]string{"caminho_absoluto": folder})); e == nil {
		t.Fatal("nonempty folder removed")
	}
	if e = os.Remove(moved); e != nil {
		t.Fatal(e)
	}
	// Original-version sidecar backups are intentionally kept alongside files.
	for _, sidecar := range []string{initial, moved} {
		matches, _ := filepath.Glob(sidecar + ".bak-*")
		for _, path := range matches {
			_ = os.Remove(path)
		}
	}
	if _, e = a.dispatch("excluir_pasta_vazia_pc", opArgs(map[string]string{"caminho_absoluto": folder})); e != nil {
		t.Fatal(e)
	}
}
func TestBackupsAndSafeRestore(t *testing.T) {
	a := newTestApp(t)
	_ = a.setMode("maximum", "LIBERAR")
	target := filepath.Join(t.TempDir(), "check.md")
	if e := os.WriteFile(target, []byte("before"), 0600); e != nil {
		t.Fatal(e)
	}
	r, e := a.dispatch("criar_backup_pc", opArgs(map[string]string{"caminho_absoluto": target}))
	if e != nil {
		t.Fatal(e)
	}
	id := r.(RecoveryResult).RecoveryID
	_ = os.WriteFile(target, []byte("after"), 0600)
	if _, e = a.dispatch("restaurar_recuperacao_pc", opArgs(map[string]string{"id": id})); e != nil {
		t.Fatal(e)
	}
	if b, _ := os.ReadFile(target); string(b) != "before" {
		t.Fatal("backup not restored")
	}
	entries, _ := a.recoveryList()
	if len(entries) != 2 {
		t.Fatalf("restore should back up current file; got %d", len(entries))
	}
}
func TestProtectedOperationsNoClobber(t *testing.T) {
	a := newTestApp(t)
	_ = a.setMode("maximum", "LIBERAR")
	folder := t.TempDir()
	source := filepath.Join(folder, "source.txt")
	destination := filepath.Join(folder, "destination.txt")
	_ = os.WriteFile(source, []byte("safe"), 0600)
	_ = os.WriteFile(destination, []byte("existing"), 0600)
	if _, e := a.dispatch("mover_arquivo_pc", opArgs(map[string]string{"origem": source, "destino": destination})); e == nil {
		t.Fatal("move overwrote destination")
	}
	if b, _ := os.ReadFile(destination); string(b) != "existing" {
		t.Fatal("destination modified")
	}
	if _, e := a.dispatch("excluir_arquivo_pc", opArgs(map[string]string{"caminho_absoluto": filepath.Join(folder, ".env")})); e == nil {
		t.Fatal("protected file allowed")
	}
	if _, e := a.dispatch("criar_pasta_pc", opArgs(map[string]string{"caminho_absoluto": filepath.Join(folder, ".git")})); e == nil {
		t.Fatal("protected directory allowed")
	}
	if _, e := a.dispatch("excluir_arquivo_pc", opArgs(map[string]string{"caminho_absoluto": filepath.Join(folder, "missing.txt")})); e == nil {
		t.Fatal("missing file allowed")
	}
	if _, e := a.dispatch("restaurar_recuperacao_pc", opArgs(map[string]string{"id": "invalid"})); e == nil {
		t.Fatal("bad id allowed")
	}
	big := filepath.Join(folder, "oversized.txt")
	if e := os.WriteFile(big, []byte(strings.Repeat("z", maxVaultFile+1)), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := a.dispatch("excluir_arquivo_pc", opArgs(map[string]string{"caminho_absoluto": big})); e == nil {
		t.Fatal("oversized deleted")
	}
	if _, e := os.Stat(big); e != nil {
		t.Fatal("oversized original lost")
	}
	symlink := filepath.Join(folder, "linked.txt")
	if e := os.Symlink(source, symlink); e == nil {
		if _, e = a.dispatch("excluir_arquivo_pc", opArgs(map[string]string{"caminho_absoluto": symlink})); e == nil {
			t.Fatal("symlink deletion allowed")
		}
	}
}

func TestFileOpsThroughJSONRPC(t *testing.T) {
	a := newTestApp(t)
	if err := a.setMode("maximum", "LIBERAR"); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(t.TempDir(), "rpc-test")
	params := map[string]any{"name": "criar_pasta_pc", "arguments": map[string]string{"caminho_absoluto": folder}}
	b, _ := json.Marshal(params)
	req, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 111, "method": "tools/call", "params": json.RawMessage(b)})
	res, ok := a.handleRPC(req)
	if !ok {
		t.Fatal("rpc response omitted")
	}
	by, _ := json.Marshal(res)
	if strings.Contains(string(by), `"isError":true`) {
		t.Fatalf("create folder rpc rejected: %s", by)
	}
	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		t.Fatal("rpc did not create folder", err)
	}
	if len(tools()) != 96 {
		t.Fatalf("unexpected tool count %d", len(tools()))
	}
}
