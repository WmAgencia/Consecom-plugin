package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "sync"
 "time"
)

// Consecom Live LAB is intentionally restricted to synthetic events.
// No actual PC error, credentials, files or conversation data enter the lab.
const liveResourceURI="ui://consecom/live/v2.html"
const liveMaxEvents=50
const liveMaxWait=12*time.Second
var liveEventPattern=regexp.MustCompile("^evt-[0-9]{6}$")

type LiveEvent struct {
 ID string `json:"id"`
 Sequence uint64 `json:"sequence"`
 Kind string `json:"kind"`
 Message string `json:"message"`
 CreatedAt string `json:"created_at"`
 Delivery string `json:"delivery,omitempty"`
 ClaimedAt string `json:"claimed_at,omitempty"`
}
type LiveHostReport struct {
 ID string `json:"id"`
 Mode string `json:"mode"`
 Status string `json:"status"`
 Detail string `json:"detail"`
 CreatedAt string `json:"created_at"`
}
type LiveJournal struct {
 Version int `json:"version"`
 Sequence uint64 `json:"sequence"`
 Events []LiveEvent `json:"events"`
}
type LiveLab struct {
 mu sync.Mutex
 journal string
 seq uint64
 events []LiveEvent
 reports []LiveHostReport
 notify chan struct{}
 running bool
 until time.Time
 lastPanelSent time.Time
}
func newLiveLab(path string)(*LiveLab,error){
 b:=&LiveLab{journal:path,notify:make(chan struct{})}
 raw,err:=os.ReadFile(path)
 if errors.Is(err,os.ErrNotExist){return b,nil}
 if err!=nil{return nil,err}
 if len(raw)>131072{return nil,errors.New("live lab journal exceeds limit")}
 var j LiveJournal
 if err=json.Unmarshal(raw,&j);err!=nil{return nil,err}
 if j.Version!=1||len(j.Events)>liveMaxEvents{return nil,errors.New("invalid live lab journal")}
 for i,ev:=range j.Events{
  if !liveEventPattern.MatchString(ev.ID)||ev.Sequence>j.Sequence||ev.Sequence==0{return nil,errors.New("invalid live lab event")}
  if i>0&&ev.Sequence<=j.Events[i-1].Sequence{return nil,errors.New("live lab sequence not monotonic")}
 }
 b.seq=j.Sequence;b.events=append([]LiveEvent(nil),j.Events...)
 return b,nil
}
func (b *LiveLab) saveLocked(seq uint64,events []LiveEvent)error{
 raw,err:=json.Marshal(LiveJournal{Version:1,Sequence:seq,Events:events});if err!=nil{return err}
 if len(raw)>131072{return errors.New("lab journal too large")}
 dir:=filepath.Dir(b.journal)
 if err:=os.MkdirAll(dir,0700);err!=nil{return err}
 file,err:=os.CreateTemp(dir,"live-*.tmp");if err!=nil{return err}
 defer os.Remove(file.Name())
 if err=file.Chmod(0600);err!=nil{file.Close();return err}
 if _,err=file.Write(raw);err!=nil{file.Close();return err}
 if err=file.Close();err!=nil{return err}
 return os.Rename(file.Name(),b.journal)
}
func (b *LiveLab) emitSynthetic()(LiveEvent,error){
 b.mu.Lock();defer b.mu.Unlock()
 return b.appendEventLocked(LiveEvent{
  Kind:"BUILD_FAILED",
  Message:"ERRO SINTETICO: modulo de demonstracao nao encontrado. Nenhum arquivo foi alterado.",
 })
}
func (b *LiveLab) launch(){
 b.mu.Lock()
 b.until=time.Now().Add(6*time.Minute)
 if b.running {b.mu.Unlock();return}
 b.running=true
 b.mu.Unlock()
 go func(){
  defer func(){b.mu.Lock();b.running=false;b.mu.Unlock()}()
  ticker:=time.NewTicker(25*time.Second)
  defer ticker.Stop()
  for i:=0;i<9;i++{
   <-ticker.C
   b.mu.Lock();valid:=time.Now().Before(b.until);b.mu.Unlock()
   if !valid{return}
   if _,err:=b.emitSynthetic();err!=nil{return}
  }
 }()
}
func (b *LiveLab) snapshot(after uint64)map[string]any{
 b.mu.Lock();defer b.mu.Unlock()
 latest:=make([]LiveEvent,0,8)
 for _,ev:=range b.events{
  if ev.Sequence>after{latest=append(latest,ev)}
 }
 if len(latest)>8{latest=latest[:8]}
 first:=uint64(0);if len(b.events)>0{first=b.events[0].Sequence}
 return map[string]any{
  "status":"synthetic_only","sequence":b.seq,"retained":len(b.events),
  "events":latest,"gap":first>0&&after+1<first,
  "running":b.running,"can_wake_new_chat":false,
  "note":"Polling in an active widget does not prove ChatGPT accepts an automatic new turn.",
 }
}
func (b *LiveLab) wait(ctx context.Context,after uint64,ms int)map[string]any{
 if ms<0{ms=0};if ms>12000{ms=12000}
 start:=time.Now()
 timeout:=time.NewTimer(time.Duration(ms)*time.Millisecond);defer timeout.Stop()
 for {
  snap:=b.snapshot(after)
  if len(snap["events"].([]LiveEvent))>0{
   snap["waited_ms"]=time.Since(start).Milliseconds();return snap
  }
  b.mu.Lock();ch:=b.notify;b.mu.Unlock()
  // Double-check after channel snapshot to avoid missed events.
  snap=b.snapshot(after)
  if len(snap["events"].([]LiveEvent))>0{
   snap["waited_ms"]=time.Since(start).Milliseconds();return snap
  }
  if ms==0 {snap["waited_ms"]=int64(0);return snap}
  select{
  case <-ctx.Done():
   return map[string]any{"status":"cancelled","sequence":snap["sequence"],"waited_ms":time.Since(start).Milliseconds()}
  case <-timeout.C:
   snap["waited_ms"]=time.Since(start).Milliseconds();return snap
  case <-ch:
  }
 }
}
func (b *LiveLab) get(id string)(LiveEvent,bool){
 b.mu.Lock();defer b.mu.Unlock()
 for _,ev:=range b.events{if ev.ID==id{return ev,true}}
 return LiveEvent{},false
}
func (b *LiveLab) report(id,mode,status string)error{
 if !liveEventPattern.MatchString(id){return errors.New("invalid synthetic event id")}
 if mode!="auto"&&mode!="manual"{return errors.New("invalid mode")}
 switch status {
 case "attempt","host_rejected","host_resolved","unavailable","pending":
 default:return errors.New("invalid host status")
 }
 b.mu.Lock();defer b.mu.Unlock()
 found:=false
 for i,ev:=range b.events {
  if ev.ID!=id {continue}
  found=true
  if ev.Kind=="PANEL_MESSAGE" {
   next:=ev
   switch status {
   case "host_resolved": next.Delivery="host_resolved"
   case "host_rejected","unavailable": next.Delivery="host_rejected"
   }
   if next.Delivery!=ev.Delivery {
    updated:=append([]LiveEvent(nil),b.events...)
    updated[i]=next
    if err:=b.saveLocked(b.seq,updated);err!=nil{return err}
    b.events=updated
   }
  }
  break
 }
 if !found{return errors.New("unknown event")}
 b.reports=append(b.reports,LiveHostReport{ID:id,Mode:mode,Status:status,
  Detail:"Host outcome only; no proof of new ChatGPT model turn.",
  CreatedAt:time.Now().UTC().Format(time.RFC3339Nano)})
 if len(b.reports)>70{b.reports=b.reports[len(b.reports)-70:]}
 return nil
}
func (b *LiveLab) reportsView()map[string]any{
 b.mu.Lock();defer b.mu.Unlock()
 vals:=append([]LiveHostReport(nil),b.reports...)
 return map[string]any{"observations":vals,"total":len(vals),
  "note":"host_resolved does not mean a new model turn started",
 }
}
type LiveTool struct{
 Name string `json:"name"`
 Description string `json:"description"`
 InputSchema map[string]any `json:"inputSchema"`
 Annotations map[string]any `json:"annotations"`
 Meta map[string]any `json:"_meta,omitempty"`
}
func liveToolSpecs()[]LiveTool{
 readonly:=map[string]any{"readOnlyHint":true,"openWorldHint":false}
 obs:=map[string]any{"readOnlyHint":false,"destructiveHint":false,"openWorldHint":false}
 base:=[]LiveTool{
  {Name:"consecom_live_abrir",Description:"Abrir ponte de mensagens do painel Consecom no MESMO MCP conectado. Mantenha o widget visivel no ChatGPT para receber mensagens. Opcao modo=teste reativa os erros sinteticos. Nao executa comandos.",InputSchema:schema(map[string]any{"modo":map[string]any{"type":"string","enum":[]string{"mensagens","teste"}}}),Annotations:readonly,
   Meta:map[string]any{"ui":map[string]any{"resourceUri":liveResourceURI},"openai/outputTemplate":liveResourceURI}},
  {Name:"consecom_live_status",Description:"Consultar eventos do Consecom Live; inclui mensagens escritas manualmente no painel local.",InputSchema:schema(map[string]any{
   "after_sequence":map[string]any{"type":"integer","minimum":0}},),Annotations:readonly,
   Meta:map[string]any{"ui":map[string]any{"visibility":[]string{"model","app"}},"openai/widgetAccessible":true}},
  {Name:"consecom_live_aguardar",Description:"Aguarda ate 12s por evento do Consecom, inclusive mensagens do painel, em uma chamada MCP ativa.",InputSchema:schema(map[string]any{
   "after_sequence":map[string]any{"type":"integer","minimum":0},
   "wait_ms":map[string]any{"type":"integer","minimum":0,"maximum":12000}},"after_sequence"),Annotations:readonly,
   Meta:map[string]any{"ui":map[string]any{"visibility":[]string{"model","app"}},"openai/widgetAccessible":true}},
  {Name:"consecom_live_ler_evento",Description:"Ler evento recebido, inclusive uma mensagem digitada no painel Consecom. Nao executa comandos.",InputSchema:schema(map[string]any{"id":str("evt-000001")},"id"),Annotations:readonly},
  {Name:"consecom_live_reportar_host",Description:"Registrar resultado da tentativa do widget no host. Nao confirma se o modelo acordou.",InputSchema:schema(map[string]any{
   "id":str("ID do evento sintetico"),
   "mode":map[string]any{"type":"string","enum":[]string{"auto","manual"}},
   "status":map[string]any{"type":"string","enum":[]string{"attempt","host_rejected","host_resolved","unavailable","pending"}}},"id","mode","status"),Annotations:obs,
   Meta:map[string]any{"ui":map[string]any{"visibility":[]string{"app"}},"openai/widgetAccessible":true}},
  {Name:"consecom_live_assumir_mensagem",Description:"Reivindicar atomicamente uma mensagem manual do painel para impedir envio duplicado por widgets simultaneos.",InputSchema:schema(map[string]any{"id":str("ID de mensagem evt-000001")},"id"),Annotations:obs,
   Meta:map[string]any{"ui":map[string]any{"visibility":[]string{"app"}},"openai/widgetAccessible":true}},
  {Name:"consecom_live_resultados",Description:"Consultar tentativas feitas pelo componente Live LAB e seu retorno sem dados privados.",InputSchema:schema(map[string]any{}),Annotations:readonly},
 }
 return base
}
func liveArgMap(raw json.RawMessage)(map[string]json.RawMessage,error){
 m:=map[string]json.RawMessage{}
 if len(raw)>0&&string(raw)!="null"{if err:=json.Unmarshal(raw,&m);err!=nil{return nil,err}}
 return m,nil
}
func liveArgSeq(m map[string]json.RawMessage, field string)(uint64,error){
 if len(m[field])==0{return 0,nil}
 var seq int64
 if err:=json.Unmarshal(m[field],&seq);err!=nil||seq<0{return 0,errors.New("invalid "+field)}
 return uint64(seq),nil
}
func liveArgString(m map[string]json.RawMessage,k string)(string,error){
 var s string
 if err:=json.Unmarshal(m[k],&s);err!=nil||s==""||len(s)>64{return "",errors.New("invalid "+k)}
 return s,nil
}
func (a *App) liveCall(ctx context.Context,name string,raw json.RawMessage)(map[string]any,error){
 if a.live==nil{return nil,errors.New("lab unavailable")}
 m,err:=liveArgMap(raw);if err!=nil{return nil,err}
 var v any
 switch name{
 case "consecom_live_abrir":
  modo:="mensagens"
  if len(m["modo"])>0 {if err=json.Unmarshal(m["modo"],&modo);err!=nil{return nil,errors.New("modo invalido")}}
  if modo!="mensagens"&&modo!="teste"{return nil,errors.New("modo invalido")}
  if modo=="teste"{a.live.launch()}
  v=a.live.snapshot(0)
 case "consecom_live_status":
  seq,err:=liveArgSeq(m,"after_sequence");if err!=nil{return nil,err}
  v=a.live.snapshot(seq)
 case "consecom_live_aguardar":
  seq,err:=liveArgSeq(m,"after_sequence");if err!=nil{return nil,err}
  var ms int
  if len(m["wait_ms"])>0{
   if err=json.Unmarshal(m["wait_ms"],&ms);err!=nil||ms<0||ms>12000{return nil,errors.New("invalid wait_ms")}
  }
  v=a.live.wait(ctx,seq,ms)
 case "consecom_live_assumir_mensagem":
  key,err:=liveArgString(m,"id");if err!=nil{return nil,err}
  claimed,err:=a.live.claimPanelMessage(key);if err!=nil{return nil,err}
  v=map[string]any{"claimed":claimed,"id":key}
 case "consecom_live_ler_evento":
  key,err:=liveArgString(m,"id");if err!=nil{return nil,err}
  ev,ok:=a.live.get(key);if !ok{return nil,errors.New("evento sintetico nao encontrado")}
  v=ev
 case "consecom_live_reportar_host":
  key,err:=liveArgString(m,"id");if err!=nil{return nil,err}
  mode,err:=liveArgString(m,"mode");if err!=nil{return nil,err}
  status,err:=liveArgString(m,"status");if err!=nil{return nil,err}
  if err:=a.live.report(key,mode,status);err!=nil{return nil,err}
  v=map[string]any{"ok":true}
 case "consecom_live_resultados":v=a.live.reportsView()
 default:return nil,errors.New("unknown Live LAB tool")
 }
 rawResult,_:=json.Marshal(v)
 return map[string]any{"structuredContent":v,"content":[]map[string]any{{"type":"text","text":string(rawResult)}}},nil
}
func isLiveTool(s string)bool {return strings.HasPrefix(s,"consecom_live_")}
