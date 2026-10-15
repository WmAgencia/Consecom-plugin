package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "strings"
 "sync"
 "time"
)

const desktopWorkerName="owner-desktop"

// Remote exposure is intentionally read-only for the initial Railway migration.
// The standard local tool permissions (roots, secrets and shell policy) still apply.
var desktopReadOnlyTools=map[string]bool{
 "diagnostico_pc":true,"listar_processos":true,"pastas_autorizadas":true,
 "politica_aprovacao":true,"dev_integracoes_status":true,
 "dev_projeto_inspecionar":true,"dev_servicos_inspecionar":true,
 "dev_deploy_preflight":true,"listar_arquivos":true,"ler_arquivo":true,
 "terminal_restrito":true,"agentes_ativos":true,"game_booster_estado":true,
}

type desktopRelayRuntime struct {
 mu sync.Mutex
 connected bool
 lastOK time.Time
 lastError string
 lastJob string
 lastDelivered string
}
var desktopRelay desktopRelayRuntime

func (a *App) desktopRelayStatus()map[string]any{
 settings:=a.cloudInferenceSettings()
 desktopRelay.mu.Lock();defer desktopRelay.mu.Unlock()
 online:=settings.Enabled&&desktopRelay.connected&&time.Since(desktopRelay.lastOK)<60*time.Second
 return map[string]any{
  "enabled":settings.Enabled,
  "token_configured":validCloudWorkerToken(settings.WorkerToken),
  "connected":online,
  "transport":"railway-https-outbound",
  "tunnel_required":false,
  "gateway":cloudInferenceGateway,
  "last_ok":desktopRelay.lastOK.Format(time.RFC3339),
  "last_error":desktopRelay.lastError,
  "last_job":desktopRelay.lastJob,
  "last_message":desktopRelay.lastDelivered,
  "permissions":"local MCP approval policy and protected project roots",
  "local_tool_count":len(tools()),
 }
}
func desktopUpdate(success bool,err error,job,msg string){
 desktopRelay.mu.Lock();defer desktopRelay.mu.Unlock()
 if success {desktopRelay.connected=true;desktopRelay.lastOK=time.Now();desktopRelay.lastError=""}
 if err!=nil {desktopRelay.connected=false;desktopRelay.lastError=err.Error()}
 if job!="" {desktopRelay.lastJob=job}
 if msg!="" {desktopRelay.lastDelivered=msg}
}
func (a *App) desktopExternalID(eventID string)string{
 a.mu.Lock();token:=a.cfg.Token;a.mu.Unlock()
 digest:=sha256.Sum256([]byte(token))
 return "pc-"+hex.EncodeToString(digest[:6])+"-"+eventID
}
func (b *LiveLab) desktopPendingMessages()[]LiveEvent{
 b.mu.Lock();defer b.mu.Unlock()
 result:=make([]LiveEvent,0,10)
 for _,ev:=range b.events {
  if ev.Kind=="PANEL_MESSAGE"&&(ev.Delivery=="queued"||ev.Delivery=="claimed"||ev.Delivery=="host_resolved") {result=append(result,ev)}
 }
 return result
}
type desktopJob struct {
 ID string `json:"id"`
 Tool string `json:"tool"`
 Arguments json.RawMessage `json:"arguments"`
}
func desktopExecuteReadonly(a *App,job desktopJob)(any,error){
 if !desktopReadOnlyTools[job.Tool]{return nil,errors.New("ferramenta bloqueada: operacoes de escrita nao foram habilitadas no relay")}
 // Defense in depth: tool must still be advertised as readOnly by the Windows MCP.
 registered:=false
 for _,t:=range tools(){
  if t.Name!=job.Tool {continue}
  readonly,_:=t.Annotations["readOnlyHint"].(bool)
  registered=readonly
  break
 }
 if !registered {return nil,errors.New("ferramenta nao classificada como somente leitura")}
 if len(job.Arguments)>65000 {return nil,errors.New("argumentos excedem 65 KiB")}
 args:=job.Arguments
 if len(args)==0 {args=json.RawMessage("{}")}
 return a.dispatch(job.Tool,args)
}
// Authenticated relay calls retain local MCP approval, root and browser checks.
func desktopExecuteTool(a *App,job desktopJob)(any,error){
 if len(job.Arguments)>65000{return nil,errors.New("argumentos excedem 65 KiB")}
 found:=false
 for _,t:=range tools(){if t.Name==job.Tool{found=true;break}}
 if !found{return nil,errors.New("ferramenta nao registrada no MCP Windows")}
 args:=job.Arguments
 if len(args)==0||string(args)=="null"{args=json.RawMessage("{}")}
 return a.dispatch(job.Tool,args)
}
func (a *App) desktopSyncChat(client *http.Client,token string){
 if a.live==nil{return}
 for _,ev:=range a.live.desktopPendingMessages(){
  external:=a.desktopExternalID(ev.ID)
  if ev.Delivery=="queued"{
   var sent struct{ID string `json:"id"`}
   err:=cloudWorkerCall(client,"/desktop/worker/send",token,map[string]any{
    "worker_id":desktopWorkerName,"external_id":external,"message":ev.Message,
   },&sent)
   if err!=nil {desktopUpdate(false,err,"","");return}
   if sent.ID=="" {desktopUpdate(false,errors.New("Railway nao retornou ID da mensagem"),"","");return}
   if _,err=a.live.claimPanelMessage(ev.ID);err!=nil {desktopUpdate(false,err,"","");return}
   desktopUpdate(true,nil,"",ev.ID)
  }
  var reply struct {
   ID string `json:"id"`
   State string `json:"state"`
   Answer string `json:"answer"`
  }
  err:=cloudWorkerCall(client,"/desktop/worker/reply",token,map[string]any{"worker_id":desktopWorkerName,"external_id":external},&reply)
  if err!=nil{
   // Mensagem marcada como claimed pelo túnel antigo não deve ser reenviada automaticamente.
   // O Railway não conhece esse ID antigo; deixe para recuperação manual e siga a fila.
   if strings.Contains(err.Error(),"HTTP 404"){continue}
   desktopUpdate(false,err,"","");return
  }
  if reply.State=="answered"&&strings.TrimSpace(reply.Answer)!=""{
   _,err=a.live.publishChatReply(ev.ID,reply.Answer,"final","railway-"+ev.ID)
   if err!=nil && !strings.Contains(err.Error(),"resposta final ja publicada") {desktopUpdate(false,err,"","");return}
  }
 }
}
func (a *App) startDesktopRailwayRelay(){
 go func(){
  client:=&http.Client{Timeout:30*time.Second}
  var lastHeartbeat time.Time
  var lastCatalog time.Time
  var priorToken string
  for {
   settings:=a.cloudInferenceSettings()
   if !settings.Enabled||!validCloudWorkerToken(settings.WorkerToken){
    desktopRelay.mu.Lock();desktopRelay.connected=false;desktopRelay.mu.Unlock()
    time.Sleep(2500*time.Millisecond)
    continue
   }
   if settings.WorkerToken!=priorToken {priorToken=settings.WorkerToken;lastHeartbeat=time.Time{};lastCatalog=time.Time{}}
   if time.Since(lastHeartbeat)>15*time.Second{
    var ping struct{OK bool `json:"ok"`; CatalogRegistered bool `json:"catalog_registered"`}
    err:=cloudWorkerCall(client,"/desktop/worker/heartbeat",settings.WorkerToken,map[string]any{"worker_id":desktopWorkerName},&ping)
    if err!=nil||!ping.OK{
     if err==nil{err=errors.New("Railway rejeitou a conexao")}
     desktopUpdate(false,err,"","");time.Sleep(4*time.Second);continue
    }
    lastHeartbeat=time.Now()
    if !ping.CatalogRegistered{lastCatalog=time.Time{}}
    desktopUpdate(true,nil,"","")
   }
   // Register all MCP tool schemas after restarts and periodically.
   if time.Since(lastCatalog)>2*time.Minute {
    catalog:=tools()
    var registered struct{OK bool `json:"ok"`;ToolCount int `json:"tool_count"`}
    err:=cloudWorkerCall(client,"/desktop/worker/register",settings.WorkerToken,map[string]any{
     "worker_id":desktopWorkerName,"tools":catalog,
    },&registered)
    if err!=nil||!registered.OK||registered.ToolCount!=len(catalog){
     if err==nil{err=fmt.Errorf("Railway registrou %d de %d ferramentas",registered.ToolCount,len(catalog))}
     desktopUpdate(false,err,"","");time.Sleep(4*time.Second);continue
    }
    lastCatalog=time.Now()
   }
   a.desktopSyncChat(client,settings.WorkerToken)
   var polled struct{Job *desktopJob `json:"job"`}
   err:=cloudWorkerCall(client,"/desktop/worker/poll",settings.WorkerToken,map[string]any{"worker_id":desktopWorkerName},&polled)
   if err!=nil{desktopUpdate(false,err,"","");time.Sleep(4*time.Second);continue}
   if polled.Job!=nil{
    value,execErr:=desktopExecuteTool(a,*polled.Job)
    data:=map[string]any{"worker_id":desktopWorkerName,"id":polled.Job.ID}
    if execErr!=nil {data["error"]=clip(execErr.Error(),250)}else{data["result"]=value}
    var done struct{OK bool `json:"ok"`}
    err=cloudWorkerCall(client,"/desktop/worker/complete",settings.WorkerToken,data,&done)
    if err!=nil||!done.OK{
     if err==nil{err=errors.New("Railway nao confirmou resultado da ferramenta")}
     desktopUpdate(false,err,polled.Job.ID,"");time.Sleep(4*time.Second);continue
    }
    desktopUpdate(true,nil,polled.Job.ID,"")
   }
   time.Sleep(1350*time.Millisecond)
  }
 }()
}
