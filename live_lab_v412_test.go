package main

import (
 "context"
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func liveTestRPC(t *testing.T,a *App,method string,params any)map[string]any{
 t.Helper()
 encoded,_:=json.Marshal(map[string]any{"jsonrpc":"2.0","id":3,"method":method,"params":params})
 out,ok:=a.handleRPC(encoded)
 if !ok{t.Fatal("RPC did not respond")}
 return out.(map[string]any)
}
func liveTestJSON(v any)string{raw,_:=json.Marshal(v);return string(raw)}
func TestLiveSingleMCPToolDiscoveryAndWidget(t *testing.T){
 a,e:=NewApp(filepath.Join(t.TempDir(),"config.json"));if e!=nil{t.Fatal(e)}
 listed:=liveTestRPC(t,a,"tools/list",map[string]any{})
 payload:=listed["result"].(map[string]any)
 list:=payload["tools"].([]any)
 seen:=map[string]bool{}
 for _,item:=range list{
  switch tool:=item.(type){
  case Tool:seen[tool.Name]=true
  case LiveTool:seen[tool.Name]=true
  }
 }
 for _,name:=range []string{"diagnostico_pc","consecom_live_abrir","consecom_live_aguardar","consecom_live_status","consecom_live_ler_evento","consecom_live_reportar_host","consecom_live_resultados"}{
  if !seen[name]{t.Fatalf("missing existing or live tool %s",name)}
 }
 for _,item:=range list{
  tool,ok:=item.(LiveTool)
  if ok&&tool.Name=="consecom_live_abrir"{
   if tool.Meta["openai/outputTemplate"]!=liveResourceURI{t.Fatalf("missing template metadata: %+v",tool.Meta)}
  }
 }
 resources:=liveTestRPC(t,a,"resources/list",map[string]any{})
 if !strings.Contains(liveTestJSON(resources),liveResourceURI){t.Fatal("resource not available in current MCP")}
 widget:=liveTestRPC(t,a,"resources/read",map[string]any{"uri":liveResourceURI})
 if !strings.Contains(liveTestJSON(widget),"window.openai.callTool")||!strings.Contains(liveTestJSON(widget),"sendFollowUpMessage"){t.Fatal("widget must use bridge tools + follow up")}
 if strings.Contains(liveWidgetHTML,"trycloudflare.com")||strings.Contains(liveWidgetHTML,"chat.openai.com"){t.Fatal("widget must not rely on external tunnel or browser automation")}
}
func TestLiveQueueDurableAndSyntheticOnly(t *testing.T){
 root:=t.TempDir()
 b,e:=newLiveLab(filepath.Join(root,"live-lab.json"));if e!=nil{t.Fatal(e)}
 ev,e:=b.emitSynthetic();if e!=nil{t.Fatal(e)}
 if ev.ID!="evt-000001"||!strings.Contains(ev.Message,"SINTETICO"){t.Fatalf("bad synthetic event %+v",ev)}
 if strings.Contains(ev.Message,root){t.Fatal("leaked machine file path")}
 fresh,e:=newLiveLab(filepath.Join(root,"live-lab.json"));if e!=nil{t.Fatal(e)}
 next,e:=fresh.emitSynthetic();if e!=nil{t.Fatal(e)}
 if next.Sequence!=2{t.Fatalf("journal replay: %+v",next)}
 info,e:=os.Stat(filepath.Join(root,"live-lab.json"));if e!=nil{t.Fatal(e)}
 if info.Size()>131072{t.Fatal("unexpected large journal")}
}
func TestLiveWaitWakeWithinActiveTool(t *testing.T){
 b,e:=newLiveLab(filepath.Join(t.TempDir(),"journal.json"));if e!=nil{t.Fatal(e)}
 done:=make(chan map[string]any,1)
 go func(){done<-b.wait(context.Background(),0,2000)}()
 time.Sleep(50*time.Millisecond)
 ev,e:=b.emitSynthetic();if e!=nil{t.Fatal(e)}
 select{
 case result:=<-done:
  if result["status"]!="synthetic_only"{t.Fatal(result)}
  events:=result["events"].([]LiveEvent)
  if len(events)!=1||events[0].ID!=ev.ID{t.Fatal("missed reverse event")}
 case <-time.After(2500*time.Millisecond):t.Fatal("active MCP wait not awakened")
 }
 if got:=b.wait(context.Background(),1,40);got["events"]==nil{t.Fatal("missing valid response")}
}
func TestLiveHostTelemetryNotModelWakeProof(t *testing.T){
 b,e:=newLiveLab(filepath.Join(t.TempDir(),"journal.json"));if e!=nil{t.Fatal(e)}
 ev,e:=b.emitSynthetic();if e!=nil{t.Fatal(e)}
 if err:=b.report(ev.ID,"auto","host_rejected");err!=nil{t.Fatal(err)}
 report:=b.reportsView()
 entries:=report["observations"].([]LiveHostReport)
 if len(entries)!=1||entries[0].Status!="host_rejected"{t.Fatalf("not recorded: %+v",entries)}
 if err:=b.report("evt-999999","auto","host_resolved");err==nil{t.Fatal("accepted fabricated ID")}
 if !strings.Contains(report["note"].(string),"does not mean a new model turn"){t.Fatal("should not claim model woke up")}
}
