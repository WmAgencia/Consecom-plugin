package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const listenAddr = "127.0.0.1:8765"

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func rpcResult(id json.RawMessage, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}
func rpcError(id json.RawMessage, code int, msg string) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": msg}}
}
func (a *App) handleRPC(body []byte) (any,bool){return a.handleRPCContext(context.Background(),body)}
func (a *App) handleRPCContext(ctx context.Context,body []byte) (any, bool) {
	var q rpcReq
	if e := json.Unmarshal(body, &q); e != nil {
		return rpcError(nil, -32700, "JSON invalido"), true
	}
	if q.JSONRPC != "2.0" || q.Method == "" {
		return rpcError(q.ID, -32600, "Requisicao invalida"), true
	}
	if len(q.ID) == 0 || string(q.ID) == "null" {
		return nil, false
	}
	switch q.Method {
	case "initialize":
        var hello struct{ClientInfo struct{Name string `json:"name"`} `json:"clientInfo"`}
        if json.Unmarshal(q.Params,&hello)==nil && hello.ClientInfo.Name!=""{
            a.contextObserveMCPClient(hello.ClientInfo.Name)
        }
		return rpcResult(q.ID, map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{"tools": map[string]any{"listChanged": false},"resources":map[string]any{}}, "serverInfo": map[string]string{"name": "consecom-mcp", "version": consecomVersion}}), true
	case "ping":
		return rpcResult(q.ID, map[string]any{}), true
    case "resources/list":
        return rpcResult(q.ID,map[string]any{"resources":[]any{map[string]any{"uri":liveResourceURI,"name":"Consecom Live","mimeType":"text/html;profile=mcp-app"},map[string]any{"uri":liveLegacyResourceURI,"name":"Consecom Live (compatibilidade)","mimeType":"text/html;profile=mcp-app"}}}),true
    case "resources/read":
        var p struct{URI string `json:"uri"`}
        if err:=json.Unmarshal(q.Params,&p);err!=nil||(p.URI!=liveResourceURI&&p.URI!=liveLegacyResourceURI){return rpcError(q.ID,-32602,"recurso nao encontrado"),true}
        return rpcResult(q.ID,map[string]any{"contents":[]any{map[string]any{
            "uri":p.URI,"mimeType":"text/html;profile=mcp-app",
            "text":liveWidgetHTML,
            "_meta":map[string]any{"ui":map[string]any{"prefersBorder":true}},
        }}}),true
	case "tools/list":
        list:=make([]any,0,len(tools())+len(liveToolSpecs()))
        for _,tool:=range tools(){list=append(list,tool)}
        for _,tool:=range liveToolSpecs(){list=append(list,tool)}
		return rpcResult(q.ID, map[string]any{"tools":list}), true
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if e := json.Unmarshal(q.Params, &p); e != nil {
			return rpcError(q.ID, -32602, "Parametros invalidos"), true
		}
        if isLiveTool(p.Name){
            out,err:=a.liveCall(ctx,p.Name,p.Arguments)
            if err!=nil{return rpcResult(q.ID,map[string]any{
              "isError":true,"content":[]map[string]string{{"type":"text","text":err.Error()}},
            }),true}
            return rpcResult(q.ID,out),true
        }
		result, e := a.dispatch(p.Name, p.Arguments)
		if e != nil {
			return rpcResult(q.ID, map[string]any{"isError": true, "content": []map[string]string{{"type": "text", "text": e.Error()}}}), true
		}
		if shot, ok := result.(BrowserScreenshot); ok {
			return rpcResult(q.ID, map[string]any{"content": []any{map[string]any{"type": "image", "mimeType": "image/jpeg", "data": shot.Data}}}), true
		}
		s, e := json.Marshal(result)
		if e != nil {
			return rpcError(q.ID, -32603, "Erro interno"), true
		}
		return rpcResult(q.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": string(s)}}}), true
	default:
		return rpcError(q.ID, -32601, "Metodo desconhecido"), true
	}
}
func content(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}
func (a *App) mcpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(405)
		return
	}
	if r.Header.Get("Origin") != "" {
		http.Error(w, "origin nao permitido", 403)
		return
	}
	key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(key), []byte(a.cfg.Token)) != 1 {
		http.Error(w, "nao autorizado", 401)
		return
	}
	a.markMCP()
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content type incorreto", 415)
		return
	}
	b, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024*1024))
	if e != nil {
		http.Error(w, "pedido muito grande", 413)
		return
	}
	res, ok := a.handleRPCContext(r.Context(),b)
	if !ok {
		w.WriteHeader(202)
		return
	}
	content(w, res)
}
func hostOK(r *http.Request) bool {
	return strings.EqualFold(r.Host, listenAddr) || r.Host == "localhost:8765"
}
func validOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return origin == "" || origin == "http://127.0.0.1:8765" || origin == "http://localhost:8765"
}
func (a *App) apiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" && r.Method != "GET" {
		w.WriteHeader(405)
		return
	}
	if !validOrigin(r) {
		http.Error(w, "origem bloqueada", 403)
		return
	}
	if r.Header.Get("X-Local-Key") != a.csrf {
		http.Error(w, "painel local nao autorizado", 403)
		return
	}
	if strings.HasPrefix(r.URL.Path,"/api/cloud-inference/"){a.cloudInferenceAPI(w,r);return}
	switch r.URL.Path {
	case "/api/status":
		m, e := systemMetrics()
		if e != nil {
			m = map[string]any{"erro": e.Error()}
		}
		processes, pe := processList()
		pErr := ""
		if pe != nil {
			pErr = pe.Error()
		}
		power, _ := a.boosterStatus()
		content(w, map[string]any{"metrics": m, "processes": processes, "process_error": pErr, "roots": a.roots(), "power": power, "pending": a.pendings(), "system": a.statusExtra(), "disks": diskList(), "agents": a.getAgents(), "desktop": a.desktopRelayStatus(), "browser": a.browserStatus(), "updates": a.updateStatus()})
    case "/api/live/send":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{Message string `json:"message"`}
        if !decode(w,r,&p){return}
        if a.live==nil{respond(w,nil,errors.New("fila Consecom Live indisponivel"));return}
        ev,err:=a.live.enqueuePanelMessage(p.Message)
        if err!=nil{respond(w,nil,err);return}
        respond(w,map[string]any{"id":ev.ID,"delivery":ev.Delivery,"created_at":ev.CreatedAt},nil)
    case "/api/live/chat":
        if r.Method!="GET"{http.Error(w,"GET necessario",405);return}
        if a.live==nil{respond(w,nil,errors.New("chat Consecom indisponivel"));return}
        respond(w,a.live.chatHistory(),nil)
    case "/api/live/latency":
        if r.Method!="GET"{http.Error(w,"GET necessario",405);return}
        if a.live==nil{respond(w,nil,errors.New("diagnostico indisponivel"));return}
        respond(w,a.live.latencyReport(),nil)
    case "/api/live/displayed":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{
            EventID string `json:"event_id"`
            ReplyID string `json:"reply_id"`
            ObservedAt string `json:"observed_at"`
        }
        if !decode(w,r,&p){return}
        if a.live==nil{respond(w,nil,errors.New("chat indisponivel"));return}
        recorded,err:=a.live.markPanelDisplayed(p.EventID,p.ReplyID,p.ObservedAt)
        respond(w,map[string]any{"recorded":recorded},err)
    case "/api/live/messages":
        if r.Method!="GET"{http.Error(w,"GET necessario",405);return}
        if a.live==nil{respond(w,nil,errors.New("fila Consecom Live indisponivel"));return}
        respond(w,a.live.panelOutbox(),nil)
    case "/api/live/retry":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{ID string `json:"id"`}
        if !decode(w,r,&p){return}
        if a.live==nil{respond(w,nil,errors.New("fila Consecom Live indisponivel"));return}
        ev,err:=a.live.retryPanelMessage(p.ID)
        if err!=nil{respond(w,nil,err);return}
        respond(w,map[string]any{"id":ev.ID,"delivery":ev.Delivery,"created_at":ev.CreatedAt},nil)
	case "/api/browser/options":
		content(w, a.browserOptions())
	case "/api/browser/default":
		var p BrowserPreference
		if !decode(w, r, &p) {
			return
		}
		respond(w, "preferencia de navegador atualizada", a.setBrowserPreference(p))
	case "/api/updates/status":
		content(w, a.updateStatus())
	case "/api/updates/settings":
		var p struct {
			Repo           string `json:"repo"`
			RemoteApproved bool   `json:"remote_approved"`
			Confirm        string `json:"confirm"`
		}
		if !decode(w, r, &p) {
			return
		}
		respond(w, "configuracao de atualizacoes salva", a.updateSettings(UpdateSettings{Repo: p.Repo, RemoteApproved: p.RemoteApproved}, p.Confirm))
	case "/api/updates/check":
		if r.Method != "POST" {
			http.Error(w, "POST necessario", 405)
			return
		}
		v, e := a.updateCheck()
		respond(w, v, e)
	case "/api/updates/local":
        if r.Method != "POST" { http.Error(w,"POST necessario",405); return }
        v, e := a.updateRequestLocal()
        respond(w, v, e)
    case "/api/updates/request":
		if r.Method != "POST" {
			http.Error(w, "POST necessario", 405)
			return
		}
		v, e := a.updateRequest()
		respond(w, v, e)
	case "/api/tunnel/connect", "/api/tunnel/disconnect":
        http.Error(w,"O tunel foi desativado. Use a conexao direta Railway em Consecom Plugin → Conectar ao ChatGPT.",http.StatusGone)
        return
	case "/api/contexto":
        v,e:=a.contextStatus();respond(w,v,e)
    case "/api/contexto/indice":
        v,e:=a.contextIndex();respond(w,v,e)
    case "/api/contexto/integracoes":
        v,e:=a.contextIntegrations();respond(w,v,e)
    case "/api/contexto/integracoes/conectar":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{ID string `json:"id"`}
        if !decode(w,r,&p){return}
        v,e:=a.contextConnectIntegration(p.ID);respond(w,v,e)
    case "/api/contexto/integracoes/indexar":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{ID string `json:"id"`}
        if !decode(w,r,&p){return}
        v,e:=a.contextIndexIntegration(p.ID);respond(w,v,e)
    case "/api/contexto/vault/auto":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        v,e:=a.contextVaultAuto();respond(w,v,e)
    case "/api/contexto/vault/select":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{ID string `json:"id"`}
        if !decode(w,r,&p){return}
        v,e:=a.contextVaultSelect(p.ID);respond(w,v,e)
    case "/api/contexto/source/register":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{Name string `json:"name"`}
        if !decode(w,r,&p){return}
        v,e:=a.contextRegisterSource(p.Name,"panel-manual");respond(w,v,e)
    case "/api/contexto/note":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{SourceID string `json:"source_id"`;Title string `json:"title"`;Content string `json:"content"`;Links []string `json:"links"`}
        if !decode(w,r,&p){return}
        v,e:=a.contextSaveNote(p.SourceID,p.Title,p.Content,p.Links);respond(w,v,e)
    case "/api/contexto/note/read":
        if r.Method!="POST"{http.Error(w,"POST necessario",405);return}
        var p struct{ID string `json:"id"`}
        if !decode(w,r,&p){return}
        v,e:=a.contextReadNote(p.ID);respond(w,v,e)
	case "/api/agents":
		content(w, map[string]any{"agents": a.getAgents(), "desktop": a.desktopRelayStatus(), "browser": a.browserStatus(), "updates": a.updateStatus()})
	case "/api/policy":
		var p struct {
			Mode    string `json:"mode"`
			Confirm string `json:"confirm"`
		}
		if !decode(w, r, &p) {
			return
		}
		respond(w, "modo atualizado", a.setMode(p.Mode, p.Confirm))
	case "/api/roots/add":
		var p struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if !decode(w, r, &p) {
			return
		}
		v, e := a.addRoot(p.Name, p.Path)
		respond(w, v, e)
	case "/api/roots/delete":
		var p struct {
			ID string `json:"id"`
		}
		if !decode(w, r, &p) {
			return
		}
		respond(w, "excluida", a.deleteRoot(p.ID))
	case "/api/roots/writable":
		var p struct {
			ID       string `json:"id"`
			Writable bool   `json:"writable"`
		}
		if !decode(w, r, &p) {
			return
		}
		respond(w, "permissao atualizada", a.setWritable(p.ID, p.Writable))
	case "/api/actions/approve":
		var p struct {
			ID string `json:"id"`
		}
		if !decode(w, r, &p) {
			return
		}
		s, e := a.approve(p.ID)
		audit("aprovacao %s: %v", p.ID, e)
		respond(w, s, e)
	case "/api/actions/deny":
		var p struct {
			ID string `json:"id"`
		}
		if !decode(w, r, &p) {
			return
		}
		_, e := a.pop(p.ID)
		audit("negacao %s", p.ID)
		respond(w, "solicitacao recusada", e)
	default:
		http.NotFound(w, r)
	}
}
func decode(w http.ResponseWriter, r *http.Request, out any) bool {
	if r.Method != "POST" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "POST JSON necessario", 405)
		return false
	}
	e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 100*1024)).Decode(out)
	if e != nil {
		http.Error(w, e.Error(), 400)
		return false
	}
	return true
}
func respond(w http.ResponseWriter, result any, e error) {
	if e != nil {
		w.WriteHeader(400)
		content(w, map[string]any{"ok": false, "error": e.Error()})
		return
	}
	content(w, map[string]any{"ok": true, "result": result})
}
func (a *App) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", a.mcpHandler)
	mux.HandleFunc("/cloud-inference",a.cloudInferenceUI)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte(strings.ReplaceAll(uiHTML, "___TOKEN___", a.csrf)))
	})
	mux.HandleFunc("/logo.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(iconPNG)
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(iconICO)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		content(w, map[string]any{"status": "ok", "service": "consecom-mcp", "version": consecomVersion})
	})
	mux.HandleFunc("/api/", a.apiHandler)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostOK(r) {
			http.Error(w, "host bloqueado", 403)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		mux.ServeHTTP(w, r)
	})
}
func audit(format string, v ...any) { log.Printf(format, v...) }
func stdioProxy(a *App) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	client := &http.Client{Timeout: 25 * time.Second}
	for scanner.Scan() {
		line := scanner.Bytes()
		var q rpcReq
		if json.Unmarshal(line, &q) != nil {
			continue
		}
		req, e := http.NewRequest("POST", "http://"+listenAddr+"/mcp", bytes.NewReader(line))
		if e != nil {
			return e
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+a.cfg.Token)
		resp, e := client.Do(req)
		if e != nil {
			if len(q.ID) > 0 {
				b, _ := json.Marshal(rpcError(q.ID, -32000, "Servidor local indisponivel; abra MCP-PC.exe no Windows"))
				fmt.Fprintln(os.Stdout, string(b))
			}
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
		resp.Body.Close()
		if resp.StatusCode != 200 && resp.StatusCode != 202 {
			if len(q.ID) > 0 {
				b, _ := json.Marshal(rpcError(q.ID, -32000, fmt.Sprintf("Erro HTTP local %d", resp.StatusCode)))
				fmt.Fprintln(os.Stdout, string(b))
			}
			continue
		}
		if resp.StatusCode == 202 {
			continue
		}
		fmt.Fprintln(os.Stdout, string(bytes.TrimSpace(body)))
	}
	return scanner.Err()
}
func main() {
	stdio := flag.Bool("stdio", false, "Proxy MCP stdio para OpenAI Secure MCP Tunnel")
	headless := flag.Bool("headless", false, "Inicia sem abrir navegador")
	portTest := flag.Bool("check", false, "Verifica se o servidor local esta online")
	flag.Parse()
	a, e := NewApp(configLocation())
	if e != nil {
		fmt.Fprintln(os.Stderr, "Erro:", e)
		return
	}
	if *stdio {
		if e = stdioProxy(a); e != nil {
			fmt.Fprintln(os.Stderr, e)
		}
		return
	}
	if *portTest {
		c := http.Client{Timeout: 2 * time.Second}
		r, e := c.Get("http://" + listenAddr + "/health")
		if e != nil {
			fmt.Println("offline")
			return
		}
		r.Body.Close()
		fmt.Println(r.Status)
		return
	}
	f, e := os.OpenFile(strings.TrimSuffix(a.path, "config.json")+"service.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if e == nil {
		log.SetOutput(f)
		defer f.Close()
	}
	listener, e := net.Listen("tcp", listenAddr)
	if e != nil {
		c := http.Client{Timeout: 1 * time.Second}
		r, x := c.Get("http://" + listenAddr + "/health")
		if x == nil && r.StatusCode == 200 {
			var health struct {
				Service string `json:"service"`
			}
			_ = json.NewDecoder(r.Body).Decode(&health)
			r.Body.Close()
			if health.Service != "consecom-mcp" {
				audit("porta ocupada por outro servidor")
				return
			}
			if !*headless {
				openDashboard("http://" + listenAddr + "/")
			}
			return
		}
		audit("bind falhou: %v", e)
		return
	}
	// A conexão antiga por túnel foi substituída pelo relay HTTPS de saída no Railway.
	a.startDesktopRailwayRelay()
	a.startCloudInferenceWorker()
	server := &http.Server{Handler: a.handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 28 * time.Second, IdleTimeout: 30 * time.Second}
	if !*headless {
		go func() { time.Sleep(450 * time.Millisecond); openDashboard("http://" + listenAddr + "/") }()
	}
	audit("MCP local em %s; nao exponha portas; perfil read-only padrao", listenAddr)
	if e = server.Serve(listener); e != nil && !errors.Is(e, http.ErrServerClosed) {
		audit("erro do servidor: %v", e)
	}
}
