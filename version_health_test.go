package main

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "testing"
)

// Regressao: respostas /health nao podem ter versao fixa de uma release antiga.
func TestHealthReportsCurrentVersion(t *testing.T) {
    app, err := NewApp(filepath.Join(t.TempDir(), "config.json"))
    if err != nil { t.Fatal(err) }
    req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/health", nil)
    recorder := httptest.NewRecorder()
    app.handler().ServeHTTP(recorder, req)
    if recorder.Code != http.StatusOK { t.Fatalf("status inesperado: %d", recorder.Code) }
    var payload struct {
        Service string `json:"service"`
        Version string `json:"version"`
    }
    if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil { t.Fatal(err) }
    if payload.Service != "consecom-mcp" || payload.Version != consecomVersion {
        t.Fatalf("/health declarou versao %q, esperado %q", payload.Version, consecomVersion)
    }
}
