// Consecom Live POC is an isolated MCP Apps + SSE experiment.
// It deliberately exposes *synthetic* events only. It does not access the PC,
// automate the ChatGPT UI, or reuse a production Consecom credential.
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const resourceURI = "ui://consecom/live-poc-v1.html"
const maxEvents = 50

type event struct {
	ID        string `json:"id"`
	Sequence  uint64 `json:"sequence"`
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}
type broker struct {
	mu        sync.Mutex
	seq       uint64
	events    []event
	listeners map[chan event]struct{}
}

func newBroker() *broker { return &broker{listeners: make(map[chan event]struct{})} }
func (b *broker) emit() event {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	ev := event{ID: fmt.Sprintf("evt-%06d", b.seq), Sequence: b.seq, Kind: "BUILD_FAILED", Message: "Erro de build de demonstracao: modulo ficticio ausente", CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	b.events = append(b.events, ev)
	if len(b.events) > maxEvents {
		b.events = b.events[len(b.events)-maxEvents:]
	}
	for ch := range b.listeners {
		select {
		case ch <- ev:
		default:
		}
	}
	return ev
}
func (b *broker) get(id string) (event, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ev := range b.events {
		if ev.ID == id {
			return ev, true
		}
	}
	return event{}, false
}
func (b *broker) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events)
}
func (b *broker) subscribe() (chan event, []event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan event, 16)
	b.listeners[ch] = struct{}{}
	snapshot := append([]event(nil), b.events...)
	return ch, snapshot, func() { b.mu.Lock(); delete(b.listeners, ch); b.mu.Unlock() }
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}
type app struct {
	token, channel, origin string
	broker                 *broker
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func newApp(token, origin string) (*app, error) {
	if len(token) < 24 {
		return nil, errors.New("CONSECOM_LIVE_TOKEN precisa de ao menos 24 caracteres")
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && parsed.Hostname() == "127.0.0.1")) {
		return nil, errors.New("CONSECOM_LIVE_PUBLIC_ORIGIN deve ser HTTPS (ou 127.0.0.1 em teste local)")
	}
	channel, err := randomHex(24)
	if err != nil {
		return nil, err
	}
	return &app{token: token, channel: channel, origin: strings.TrimSuffix(origin, "/"), broker: newBroker()}, nil
}
func (a *app) authorized(r *http.Request) bool {
	s := r.Header.Get("Authorization")
	if !strings.HasPrefix(s, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(s, "Bearer ")
	return subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) == 1
}
func jsonResponse(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func result(id json.RawMessage, v any) any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": v}
}
func rpcError(id json.RawMessage, code int, message string) any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}}
}
func toolSpec(name, desc string, props map[string]any, required []string, widget bool) map[string]any {
	s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	t := map[string]any{"name": name, "description": desc, "inputSchema": s, "annotations": map[string]any{"readOnlyHint": name != "consecom_live_emitir_teste"}}
	if widget {
		t["_meta"] = map[string]any{"ui": map[string]any{"resourceUri": resourceURI}, "openai/outputTemplate": resourceURI}
	}
	return t
}
func (a *app) tools() []map[string]any {
	return []map[string]any{
		toolSpec("consecom_live_abrir", "Exibe um painel de eventos exclusivamente simulados do Consecom Live. Nao executa comandos no PC.", map[string]any{}, nil, true),
		toolSpec("consecom_live_ler_evento", "Le dados de um evento sintético previamente emitido.", map[string]any{"id": map[string]any{"type": "string"}}, []string{"id"}, false),
		toolSpec("consecom_live_emitir_teste", "Cria evento sintético BUILD_FAILED para testar o painel. Nao altera arquivos.", map[string]any{}, nil, false),
	}
}
func (a *app) callTool(name string, args json.RawMessage) (any, error) {
	switch name {
	case "consecom_live_abrir":
		v := map[string]any{"status": "demo", "recent": a.broker.count(), "note": "Apenas eventos simulados"}
		return map[string]any{"structuredContent": v, "content": []any{map[string]any{"type": "text", "text": "Painel Consecom Live de demonstracao. Os eventos sao simulados."}}}, nil
	case "consecom_live_emitir_teste":
		ev := a.broker.emit()
		return map[string]any{"structuredContent": ev, "content": []any{map[string]any{"type": "text", "text": fmt.Sprintf("Evento de teste criado: %s", ev.ID)}}}, nil
	case "consecom_live_ler_evento":
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(args, &p); err != nil || p.ID == "" {
			return nil, errors.New("id obrigatorio")
		}
		ev, ok := a.broker.get(p.ID)
		if !ok {
			return nil, errors.New("evento nao encontrado")
		}
		return map[string]any{"structuredContent": ev, "content": []any{map[string]any{"type": "text", "text": fmt.Sprintf("Evento %s: %s (simulado)", ev.ID, ev.Message)}}}, nil
	default:
		return nil, errors.New("ferramenta desconhecida")
	}
}
func (a *app) routeMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	if r.Header.Get("Origin") != "" {
		http.Error(w, "origin not allowed", 403)
		return
	}
	if !a.authorized(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "unsupported content type", 415)
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024*1024))
	if err != nil {
		http.Error(w, "payload too large", 413)
		return
	}
	var q rpcRequest
	if err := json.Unmarshal(b, &q); err != nil {
		jsonResponse(w, 200, rpcError(nil, -32700, "invalid JSON"))
		return
	}
	if q.JSONRPC != "2.0" || q.Method == "" {
		jsonResponse(w, 200, rpcError(q.ID, -32600, "invalid request"))
		return
	}
	if len(q.ID) == 0 || string(q.ID) == "null" {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	switch q.Method {
	case "initialize":
		jsonResponse(w, 200, result(q.ID, map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}, "resources": map[string]any{}}, "serverInfo": map[string]any{"name": "consecom-live-poc", "version": "0.1.0"}}))
	case "ping":
		jsonResponse(w, 200, result(q.ID, map[string]any{}))
	case "tools/list":
		jsonResponse(w, 200, result(q.ID, map[string]any{"tools": a.tools()}))
	case "resources/list":
		jsonResponse(w, 200, result(q.ID, map[string]any{"resources": []any{map[string]any{"uri": resourceURI, "name": "Consecom Live POC", "mimeType": "text/html;profile=mcp-app"}}}))
	case "resources/read":
		var p struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(q.Params, &p)
		if p.URI != resourceURI {
			jsonResponse(w, 200, rpcError(q.ID, -32602, "resource not found"))
			return
		}
		html, err := a.renderWidget()
		if err != nil {
			jsonResponse(w, 200, rpcError(q.ID, -32603, "widget template error"))
			return
		}
		jsonResponse(w, 200, result(q.ID, map[string]any{"contents": []any{map[string]any{"uri": resourceURI, "mimeType": "text/html;profile=mcp-app", "text": html, "_meta": map[string]any{"ui": map[string]any{"prefersBorder": true, "csp": map[string]any{"connectDomains": []string{a.origin}}}}}}}))
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(q.Params, &p); err != nil {
			jsonResponse(w, 200, rpcError(q.ID, -32602, "invalid params"))
			return
		}
		out, err := a.callTool(p.Name, p.Arguments)
		if err != nil {
			jsonResponse(w, 200, result(q.ID, map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": err.Error()}}}))
			return
		}
		jsonResponse(w, 200, result(q.ID, out))
	default:
		jsonResponse(w, 200, rpcError(q.ID, -32601, "method not found"))
	}
}
func (a *app) routeEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get("channel")), []byte(a.channel)) != 1 {
		http.Error(w, "forbidden", 403)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*") // demo-only synthetic events
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unavailable", 500)
		return
	}
	ch, snapshot, unsubscribe := a.broker.subscribe()
	defer unsubscribe()
	send := func(ev event) error {
		raw, _ := json.Marshal(ev)
		_, err := fmt.Fprintf(w, "id: %d\nevent: consecom\ndata: %s\n\n", ev.Sequence, raw)
		if err == nil {
			f.Flush()
		}
		return err
	}
	_, _ = io.WriteString(w, ": connected\n\n")
	f.Flush()
	// Reconnects intentionally resend the current bounded snapshot; client deduplicates by ID.
	for _, ev := range snapshot {
		if err := send(ev); err != nil {
			return
		}
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			if err := send(ev); err != nil {
				return
			}
		case <-ticker.C:
			_, err := io.WriteString(w, ": ping\n\n")
			if err != nil {
				return
			}
			f.Flush()
		}
	}
}
func (a *app) routeEmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	if !a.authorized(r) {
		http.Error(w, "unauthorized", 401)
		return
	}
	jsonResponse(w, 201, a.broker.emit())
}
func (a *app) mux() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/mcp", a.routeMCP)
	m.HandleFunc("/events", a.routeEvents)
	m.HandleFunc("/demo/emit", a.routeEmit)
	m.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]string{"status": "ok", "mode": "synthetic-only"})
	})
	return m
}
func main() {
	token := os.Getenv("CONSECOM_LIVE_TOKEN")
	origin := os.Getenv("CONSECOM_LIVE_PUBLIC_ORIGIN")
	if origin == "" {
		origin = "http://127.0.0.1:8793"
	}
	a, err := newApp(token, origin)
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("CONSECOM_LIVE_LISTEN")
	if addr == "" {
		addr = "127.0.0.1:8793"
	}
	log.Printf("Consecom Live POC (demo) listening on %s, public origin %s", addr, origin)
	log.Fatal(http.ListenAndServe(addr, a.mux()))
}

// Escaping through html/template intentionally prevents unsafe configuration injection.
func (a *app) renderWidget() (string, error) {
	t, err := template.New("ui").Parse(widgetHTML)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	data := struct{ Origin, Channel string }{Origin: a.origin, Channel: a.channel}
	err = t.Execute(&buf, data)
	return buf.String(), err
}