package main

import (
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "strings"
    "testing"
)

func TestConversationAssetsAllowedByCSP(t *testing.T) {
    a, err := NewApp(filepath.Join(t.TempDir(), "config.json"))
    if err != nil { t.Fatal(err) }
    h := a.handler()
    get := func(path string) *httptest.ResponseRecorder {
        t.Helper()
        req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765"+path, nil)
        rec := httptest.NewRecorder()
        h.ServeHTTP(rec, req)
        if rec.Code != http.StatusOK { t.Fatalf("%s respondeu HTTP %d",path,rec.Code) }
        return rec
    }
    root := get("/")
    policy := root.Header().Get("Content-Security-Policy")
    if !strings.Contains(policy, "script-src 'self' 'unsafe-inline'") || !strings.Contains(policy, "style-src 'self' 'unsafe-inline'") {
        t.Fatalf("CSP bloqueia recursos locais do chat: %s",policy)
    }
    if !strings.Contains(policy, "default-src 'none'") || !strings.Contains(policy, "frame-ancestors 'none'") {
        t.Fatalf("protecoes CSP ausentes: %s",policy)
    }
    page := root.Body.String()
    if !strings.Contains(page, "/chat-ui.js") || !strings.Contains(page, "/chat-ui.css") || !strings.Contains(page, "id=\"navLab\"") {
        t.Fatal("painel nao inclui recursos da conversa")
    }
    js := get("/chat-ui.js")
    if !strings.Contains(js.Header().Get("Content-Type"), "javascript") || !strings.Contains(js.Body.String(),"navChat") || !strings.Contains(js.Body.String(),"chatDock") {
        t.Fatal("JavaScript da interface de conversa nao foi disponibilizado corretamente")
    }
    css := get("/chat-ui.css")
    if !strings.Contains(css.Header().Get("Content-Type"), "text/css") || !strings.Contains(css.Body.String(),".chat-dock") || !strings.Contains(css.Body.String(),".app.chat-open") {
        t.Fatal("CSS da interface de conversa nao foi disponibilizado corretamente")
    }
}
