package main

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net/http"
 "regexp"
 "strings"
 "sync"
 "time"
)

const cloudInferenceGateway = "https://consecom-mcp-relay-production.up.railway.app"
const cloudOllamaURL = "http://127.0.0.1:11434"
const cloudWorkerName = "owner-local"

type CloudInferenceSettings struct {
 Enabled bool   `json:"enabled"`
 WorkerToken string `json:"worker_token,omitempty"`
}
type CloudInferenceState struct {
 mu sync.Mutex
 lastSeen time.Time
 lastSuccess time.Time
 lastError string
 lastJob string
 availableModels []string
 connected bool
}
var cloudInferenceRuntime CloudInferenceState

var cloudModelSanitize=regexp.MustCompile(`[^a-z0-9._-]+`)
func cloudModelAlias(id string) string {
 normalized:=strings.Trim(cloudModelSanitize.ReplaceAllString(strings.ToLower(id),"-"),"-")
 if len(normalized)>105 {normalized=normalized[:105]}
 if len(normalized)<2{return ""}
 return "claude-local-"+normalized
}
func (a *App) cloudInferenceSettings() CloudInferenceSettings {
 a.mu.Lock(); defer a.mu.Unlock()
 return a.cfg.CloudInference
}
func (a *App) configureCloudInference(value CloudInferenceSettings) error {
 value.WorkerToken=strings.TrimSpace(value.WorkerToken)
 a.mu.Lock(); defer a.mu.Unlock()
 if value.WorkerToken=="" {value.WorkerToken=a.cfg.CloudInference.WorkerToken}
 if value.Enabled && !validCloudWorkerToken(value.WorkerToken) {
  return errors.New("Informe o token privado CONSECOM_LOCAL_WORKER_TOKEN do Railway")
 }
 if value.WorkerToken!="" && !validCloudWorkerToken(value.WorkerToken){
  return errors.New("Token Consecom Cloud invalido")
 }
 old:=a.cfg.CloudInference
 a.cfg.CloudInference=value
 if err:=a.saveLocked();err!=nil{a.cfg.CloudInference=old;return err}
 return nil
}
func validCloudWorkerToken(token string)bool{
 if len(token)<32||len(token)>200||!strings.HasPrefix(token,"cci_"){return false}
 for _,ch:=range token {if !(ch>='A'&&ch<='Z'||ch>='a'&&ch<='z'||ch>='0'&&ch<='9'||ch=='_'||ch=='-'){return false}}
 return true
}
func (a *App) cloudInferenceStatus()map[string]any{
 state:=a.cloudInferenceSettings()
 cloudInferenceRuntime.mu.Lock()
 defer cloudInferenceRuntime.mu.Unlock()
 return map[string]any{
  "enabled":state.Enabled,
  "token_configured":validCloudWorkerToken(state.WorkerToken),
  "worker_connected":cloudInferenceRuntime.connected&&time.Since(cloudInferenceRuntime.lastSeen)<90*time.Second,
  "last_seen":cloudInferenceRuntime.lastSeen.Format(time.RFC3339),
  "last_success":cloudInferenceRuntime.lastSuccess.Format(time.RFC3339),
  "last_error":cloudInferenceRuntime.lastError,
  "last_job":cloudInferenceRuntime.lastJob,
  "models":append([]string(nil),cloudInferenceRuntime.availableModels...),
  "gateway":cloudInferenceGateway,
  "ollama":cloudOllamaURL,
  "note":"Motor LLM local Ollama; nao utiliza assinatura ChatGPT Chat como API. Nao executa comandos no Windows.",
 }
}
func cloudStateUpdate(ok bool,err error, models []string, job string){
 cloudInferenceRuntime.mu.Lock();defer cloudInferenceRuntime.mu.Unlock()
 cloudInferenceRuntime.lastSeen=time.Now()
 cloudInferenceRuntime.connected=ok
 if ok {cloudInferenceRuntime.lastSuccess=time.Now();cloudInferenceRuntime.lastError=""}
 if err!=nil {cloudInferenceRuntime.lastError=err.Error()}
 if models!=nil {cloudInferenceRuntime.availableModels=append([]string(nil),models...)}
 if job!="" {cloudInferenceRuntime.lastJob=job}
}
type cloudModel struct {
 ID string `json:"id"`
 OllamaID string `json:"ollama_id"`
 DisplayName string `json:"display_name"`
}
func cloudFindModels(ctx context.Context,client *http.Client)([]cloudModel,error){
 req,err:=http.NewRequestWithContext(ctx,"GET",cloudOllamaURL+"/api/tags",nil);if err!=nil{return nil,err}
 resp,err:=client.Do(req);if err!=nil{return nil,err}
 defer resp.Body.Close()
 if resp.StatusCode!=200{return nil,fmt.Errorf("Ollama /api/tags HTTP %d",resp.StatusCode)}
 var payload struct {Models []struct{Name string `json:"name"`} `json:"models"`}
 if err=json.NewDecoder(io.LimitReader(resp.Body,1024*1024)).Decode(&payload);err!=nil{return nil,err}
 found:=[]cloudModel{}
 used:=map[string]bool{}
 for _,m:=range payload.Models{
  alias:=cloudModelAlias(m.Name)
  if alias==""||used[alias]||len(m.Name)>130 {continue}
  used[alias]=true
  found=append(found,cloudModel{ID:alias,OllamaID:m.Name,DisplayName:"Modelo local: "+m.Name})
  if len(found)==24 {break}
 }
 return found,nil
}
func cloudHTTPJSON(ctx context.Context,client *http.Client,endpoint,token string,in any,out any)error{
 body,err:=json.Marshal(in);if err!=nil{return err}
 req,err:=http.NewRequestWithContext(ctx,"POST",cloudInferenceGateway+endpoint,bytes.NewReader(body));if err!=nil{return err}
 req.Header.Set("Authorization","Bearer "+token)
 req.Header.Set("Content-Type","application/json")
 resp,err:=client.Do(req);if err!=nil{return err};defer resp.Body.Close()
 if resp.StatusCode<200||resp.StatusCode>=300{return fmt.Errorf("Consecom Cloud %s HTTP %d",endpoint,resp.StatusCode)}
 return json.NewDecoder(io.LimitReader(resp.Body,2*1024*1024)).Decode(out)
}
func cloudWorkerCall(client *http.Client,route,token string,reqData any,result any)error{
 ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel()
 return cloudHTTPJSON(ctx,client,route,token,reqData,result)
}
func cloudRunModel(client *http.Client,model string,request json.RawMessage)(json.RawMessage,error){
 var data map[string]json.RawMessage
 if err:=json.Unmarshal(request,&data);err!=nil{return nil,err}
 if len(request)>2*1024*1024{return nil,errors.New("Pedido de inferencia excede 2 MiB")}
 b,_:=json.Marshal(model)
 data["model"]=b
 data["stream"]=[]byte("false")
 payload,err:=json.Marshal(data);if err!=nil{return nil,err}
 ctx,cancel:=context.WithTimeout(context.Background(),110*time.Second);defer cancel()
 req,err:=http.NewRequestWithContext(ctx,"POST",cloudOllamaURL+"/v1/messages",bytes.NewReader(payload))
 if err!=nil{return nil,err}
 req.Header.Set("Content-Type","application/json")
 req.Header.Set("x-api-key","ollama")
 req.Header.Set("anthropic-version","2023-06-01")
 resp,err:=client.Do(req);if err!=nil{return nil,err};defer resp.Body.Close()
 if resp.StatusCode!=200{return nil,fmt.Errorf("Ollama Messages HTTP %d",resp.StatusCode)}
 result,err:=io.ReadAll(io.LimitReader(resp.Body,1600001));if err!=nil{return nil,err}
 if len(result)>1600000{return nil,errors.New("Resposta Ollama excede limite")}
 var envelope struct{Type string `json:"type"`;Role string `json:"role"`;Content []json.RawMessage `json:"content"`;StopReason string `json:"stop_reason"`}
 if err=json.Unmarshal(result,&envelope);err!=nil{return nil,err}
 if envelope.Type!="message"||envelope.Role!="assistant"||envelope.Content==nil||envelope.StopReason=="" {return nil,errors.New("Ollama nao retornou formato Anthropic Messages valido")}
 return result,nil
}
func (a *App) startCloudInferenceWorker(){
 go func(){
  client:=&http.Client{Timeout:115*time.Second}
  lastRegister:=time.Time{}
  var lastToken string
  for {
   settings:=a.cloudInferenceSettings()
   if !settings.Enabled||!validCloudWorkerToken(settings.WorkerToken) {
    cloudStateUpdate(false,nil,[]string{},"")
    time.Sleep(3500*time.Millisecond)
    continue
   }
   if settings.WorkerToken!=lastToken{lastToken=settings.WorkerToken;lastRegister=time.Time{}}
   if time.Since(lastRegister)>25*time.Second {
    ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second)
    models,err:=cloudFindModels(ctx,client);cancel()
    if err!=nil{cloudStateUpdate(false,err,[]string{},"");time.Sleep(5*time.Second);continue}
    var result struct{OK bool `json:"ok"`}
    err=cloudWorkerCall(client,"/worker/inference/register",settings.WorkerToken,map[string]any{"worker_id":cloudWorkerName,"models":models},&result)
    if err!=nil||!result.OK{cloudStateUpdate(false,err,[]string{},"");time.Sleep(5*time.Second);continue}
    aliases:=make([]string,0,len(models));for _,m:=range models {aliases=append(aliases,m.ID)}
    cloudStateUpdate(true,nil,aliases,"")
    lastRegister=time.Now()
   }
   var polled struct{Job *struct{ID string `json:"id"`;ModelID string `json:"model_id"`;Request json.RawMessage `json:"request"`} `json:"job"`}
   err:=cloudWorkerCall(client,"/worker/inference/poll",settings.WorkerToken,map[string]string{"worker_id":cloudWorkerName},&polled)
   if err!=nil {cloudStateUpdate(false,err,nil,"");time.Sleep(4*time.Second);continue}
   if polled.Job==nil {time.Sleep(1400*time.Millisecond);continue}
   job:=polled.Job
   cloudStateUpdate(true,nil,nil,job.ID)
   var models []cloudModel
   ctx,cancel:=context.WithTimeout(context.Background(),8*time.Second)
   models,err=cloudFindModels(ctx,client);cancel()
   var answer json.RawMessage
   if err==nil {
    chosen:=""
    for _,m:=range models {if m.ID==job.ModelID {chosen=m.OllamaID;break}}
    if chosen==""{err=errors.New("Modelo local nao instalado ou nao registrado")}else{answer,err=cloudRunModel(client,chosen,job.Request)}
   }
   payload:=map[string]any{"worker_id":cloudWorkerName,"id":job.ID}
   if err!=nil{payload["error"]=clip(err.Error(),220)}else{payload["response"]=json.RawMessage(answer)}
   var confirmation struct{OK bool `json:"ok"`}
   sendErr:=cloudWorkerCall(client,"/worker/inference/complete",settings.WorkerToken,payload,&confirmation)
   if sendErr!=nil||!confirmation.OK{cloudStateUpdate(false,sendErr,nil,job.ID);time.Sleep(4*time.Second);continue}
   cloudStateUpdate(true,nil,nil,job.ID)
  }
 }()
}

// The UI does not display or return the saved worker token, and each modifying
// endpoint uses the dashboard's existing X-Local-Key and same-origin restrictions.
const cloudInferenceHTML = `<!doctype html><html lang="pt-BR"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Consecom Cloud – Inferência</title><style>body{background:#081526;color:#eaf5fb;font:15px system-ui;padding:28px;max-width:760px;margin:auto}main{border:1px solid #284361;background:#11243b;border-radius:16px;padding:22px}input,button{padding:12px;border-radius:9px;font:inherit}input{width:100%;box-sizing:border-box;background:#081525;color:#fff;border:1px solid #45627c}button{background:#0b9275;color:white;border:0;cursor:pointer;margin-top:12px}p{line-height:1.5}.muted{color:#a9c0d3}#status{white-space:pre-wrap;overflow-wrap:anywhere;background:#071323;padding:14px;border-radius:9px}.flag{display:flex;gap:8px;align-items:center;margin:15px 0}.flag input{width:auto}</style><main><a href="/" style="color:#75d6fa">← Central Consecom</a><h1>Inferência Consecom Cloud</h1><p>Use seu próprio modelo Ollama no Windows com Claude Code via Railway. <strong>Isto não utiliza esta sessão ChatGPT como uma API de modelos.</strong></p><p class="muted">Instale Ollama e execute <code>ollama pull qwen3:4b</code>. Em Railway → consecom-mcp-relay → Variables, copie o valor privado CONSECOM_LOCAL_WORKER_TOKEN. Não compartilhe esse token com clientes.</p><label>Token privado do worker (deixe em branco para manter o anterior)<input type="password" id="token" placeholder="cci_..." autocomplete="off"></label><label class="flag"><input id="enabled" type="checkbox"> Ativar conexão automática ao Railway</label><button id="save">Salvar e conectar</button><p id="message"></p><h2>Estado da conexão</h2><pre id="status">Carregando...</pre><p class="muted">Servidor externo: https://consecom-mcp-relay-production.up.railway.app · modelo aparece no catálogo quando o Ollama estiver ativo.</p></main><script>const k="___TOKEN___";async function api(path,options={}){let r=await fetch(path,{...options,headers:{"X-Local-Key":k,"Content-Type":"application/json",...(options.headers||{})}});let v=await r.json();if(!r.ok||v.ok===false)throw Error(v.error||"HTTP "+r.status);return v}async function refresh(){try{const data=await api("/api/cloud-inference/status");document.querySelector("#enabled").checked=Boolean(data.enabled);document.querySelector("#status").textContent=JSON.stringify(data,null,2)}catch(e){document.querySelector("#status").textContent=e.message}}document.querySelector("#save").onclick=async()=>{const b=document.querySelector("#save");b.disabled=true;try{await api("/api/cloud-inference/settings",{method:"POST",body:JSON.stringify({enabled:document.querySelector("#enabled").checked,worker_token:document.querySelector("#token").value})});document.querySelector("#token").value="";document.querySelector("#message").textContent="Configuração salva; aguardando conexão.";await refresh()}catch(e){document.querySelector("#message").textContent=e.message}finally{b.disabled=false}};refresh();setInterval(refresh,8000);</script></html>`

func (a *App) cloudInferenceUI(w http.ResponseWriter,r *http.Request){
 if r.Method!="GET"{http.Error(w,"GET obrigatorio",405);return}
 w.Header().Set("Content-Type","text/html; charset=utf-8");w.Header().Set("Cache-Control","no-store")
 _,_=io.WriteString(w,strings.ReplaceAll(cloudInferenceHTML,"___TOKEN___",a.csrf))
}
func (a *App) cloudInferenceAPI(w http.ResponseWriter,r *http.Request) {
 switch r.URL.Path{
 case "/api/cloud-inference/status":
  if r.Method!="GET"{http.Error(w,"GET obrigatorio",405);return}
  content(w,a.cloudInferenceStatus())
 case "/api/cloud-inference/settings":
  if r.Method!="POST"{http.Error(w,"POST obrigatorio",405);return}
  var p CloudInferenceSettings
  if !decode(w,r,&p){return}
  respond(w,"Configuracao de inferencia salva",a.configureCloudInference(p))
 default: http.NotFound(w,r)
 }
}
