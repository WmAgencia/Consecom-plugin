package main

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "strings"
    "testing"
)

func TestConversationFeatureCompletelyRemoved(t *testing.T) {
    app, err := NewApp(filepath.Join(t.TempDir(), "config.json"))
    if err != nil { t.Fatal(err) }
    h := app.handler()
    req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/", nil)
    page := httptest.NewRecorder()
    h.ServeHTTP(page, req)
    if page.Code != 200 { t.Fatalf("dashboard HTTP %d", page.Code) }
    html := page.Body.String()
    for _, removed := range []string{"chat-ui.css", "chat-ui.js", "Conversar</button>", "Pedidos ao Consecom", "taskPrompt", "sendLocalTask", "taskInboxStatus"} {
        if strings.Contains(html, removed) { t.Errorf("interface ainda referencia %q", removed) }
    }
    for _, route := range []string{"/chat-ui.js", "/chat-ui.css", "/api/tasks", "/api/tasks/add"} {
        r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765"+route, nil)
        if strings.HasPrefix(route, "/api/") { r.Header.Set("X-Local-Key", app.csrf) }
        record := httptest.NewRecorder()
        h.ServeHTTP(record, r)
        if record.Code != http.StatusNotFound { t.Errorf("rota removida %s retornou HTTP %d", route, record.Code) }
    }
    tools := tools()
    if len(tools) != 72 { t.Errorf("esperadas 72 ferramentas MCP, foram %d", len(tools)) }
    for _, tool := range tools {
        if tool.Name == "tarefas_listar" || tool.Name == "tarefas_responder" {
            t.Errorf("ferramenta de conversa ainda disponivel: %s", tool.Name)
        }
    }
    status := httptest.NewRecorder()
    r2 := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8765/api/status", nil)
    r2.Header.Set("X-Local-Key", app.csrf)
    h.ServeHTTP(status, r2)
    if status.Code != 200 { t.Fatalf("status HTTP %d", status.Code) }
    var fields map[string]any
    if err := json.Unmarshal(status.Body.Bytes(), &fields); err != nil { t.Fatal(err) }
    if _, present := fields["inbox"]; present { t.Fatal("fila de conversa continua no status publico") }
}
