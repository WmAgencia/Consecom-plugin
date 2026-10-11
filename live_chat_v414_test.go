package main

import (
 "encoding/json"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestChatBidirectionalRepliesAndPersistence(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"live-lab-v412.json")
 b,err:=newLiveLab(path);if err!=nil{t.Fatal(err)}
 ev,err:=b.enqueuePanelMessage("Como voce esta?");if err!=nil{t.Fatal(err)}
 if _,err=b.publishChatReply(ev.ID,"Nao posso responder antes de claim","final","");err==nil{t.Fatal("reply before delivery accepted")}
 ok,err:=b.claimPanelMessage(ev.ID);if err!=nil||!ok{t.Fatalf("claim %v %v",ok,err)}
 first,err:=b.publishChatReply(ev.ID,"Vou conferir o sistema.","andamento","first_step")
 if err!=nil{t.Fatal(err)}
 repeated,err:=b.publishChatReply(ev.ID,"Outro texto","andamento","first_step")
 if err!=nil||repeated.ID!=first.ID{t.Fatalf("idempotency failed: %+v %v",repeated,err)}
 done,err:=b.publishChatReply(ev.ID,"Tudo certo. Em que posso ajudar?","final","finish")
 if err!=nil{t.Fatal(err)}
 if done.Kind!="final"{t.Fatal(done)}
 if _,err=b.publishChatReply(ev.ID,"Resposta duplicada","final","second_finish");err==nil{t.Fatal("second final allowed")}
 snapshot:=b.chatHistory()
 entries:=snapshot["messages"].([]LiveChatEntry)
 if len(entries)!=3||entries[0].Role!="user"||entries[1].Kind!="andamento"||entries[2].Kind!="final"{t.Fatalf("not ordered like a chat %+v",entries)}
 if entries[0].Status!="answered"{t.Fatalf("not marked answered %+v",entries[0])}
 reopened,err:=newLiveLab(path);if err!=nil{t.Fatal(err)}
 stored:=reopened.chatHistory()["messages"].([]LiveChatEntry)
 if len(stored)!=3||stored[2].Text!="Tudo certo. Em que posso ajudar?"{t.Fatalf("reply was lost after restart: %+v",stored)}
 if info,err:=os.Stat(filepath.Join(filepath.Dir(path),"live-chat-replies.json"));err!=nil||info.Size()==0{t.Fatal("replies file missing",err)}
}
func TestChatReplyValidationAndAuthorization(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"lab.json"));if err!=nil{t.Fatal(err)}
 ev,err:=b.enqueuePanelMessage("Ola");if err!=nil{t.Fatal(err)}
 if _,err=b.publishChatReply("evt-999999","A","final","");err==nil{t.Fatal("accepted fabricated event")}
 claimed,err:=b.claimPanelMessage(ev.ID);if err!=nil||!claimed{t.Fatal(err)}
 for _,tc:=range []struct{text,kind,key string}{
  {"","final",""},{"normal","unknown",""},{"bad\x00text","final",""},{"ok","andamento","bad space"},
  {strings.Repeat("a",maxChatReplyBytes+1),"final",""},
 }{
  if _,err=b.publishChatReply(ev.ID,tc.text,tc.kind,tc.key);err==nil{t.Fatalf("accepted invalid reply %+v",tc)}
 }
 if _,err=b.publishChatReply(ev.ID,"Posso ajudar.","final","done");err!=nil{t.Fatal(err)}
 for i:=0;i<4;i++{if _,err=b.publishChatReply(ev.ID,"andamento posterior","andamento","");err!=nil{t.Fatal(err)}}
 // The read-only chat HTTP endpoint must reject unauthenticated requests.
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 request:=httptest.NewRequest("GET","http://127.0.0.1:8765/api/live/chat",nil)
 response:=httptest.NewRecorder()
 a.handler().ServeHTTP(response,request)
 if response.Code!=403{t.Fatalf("chat leaked without local key HTTP %d",response.Code)}
 request=httptest.NewRequest("GET","http://127.0.0.1:8765/api/live/chat",nil)
 request.Header.Set("X-Local-Key",a.csrf)
 response=httptest.NewRecorder()
 a.handler().ServeHTTP(response,request)
 if response.Code!=200{t.Fatalf("authorized chat unavailable HTTP %d",response.Code)}
}
func TestChatToolsExposedInSameMCP(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 toolsResponse:=liveTestRPC(t,a,"tools/list",map[string]any{})
 for _,tool:=range []string{"consecom_chat_responder","consecom_chat_historico","consecom_live_abrir","diagnostico_pc"}{
  if !strings.Contains(liveTestJSON(toolsResponse),tool){t.Fatalf("tool %s not exposed",tool)}
 }
 ev,err:=a.live.enqueuePanelMessage("Qual o status?");if err!=nil{t.Fatal(err)}
 if ok,err:=a.live.claimPanelMessage(ev.ID);err!=nil||!ok{t.Fatal(err)}
 args:=map[string]any{"id":ev.ID,"texto":"Tudo operando normalmente.","tipo":"final","chave":"test-1"}
 call:=liveTestRPC(t,a,"tools/call",map[string]any{"name":"consecom_chat_responder","arguments":args})
 if !strings.Contains(liveTestJSON(call),"Tudo operando normalmente."){t.Fatalf("reply not surfaced %+v",call)}
 hist:=liveTestRPC(t,a,"tools/call",map[string]any{"name":"consecom_chat_historico","arguments":map[string]any{}})
 if !strings.Contains(liveTestJSON(hist),"Qual o status?"){t.Fatal("chat transcript not accessible to ChatGPT")}
 if !strings.Contains(liveTestJSON(hist),"Tudo operando normalmente."){t.Fatal("model reply not persisted")}
}
func TestChatDashboardUIAndPrompt(t *testing.T){
 for _,part:=range []string{"Chat com ChatGPT","live-chat-stream","live-chat-bubble","api('live/chat'","api('live/send'","Versão 4.14.6"}{
  if !strings.Contains(uiHTML,part){t.Fatalf("chat UI missing %q",part)}
 }
 if !strings.Contains(liveWidgetHTML,"consecom_chat_responder"){t.Fatal("widget does not instruct model to publish reply")}
 if strings.Contains(liveWidgetHTML,"trycloudflare.com")||strings.Contains(liveWidgetHTML,"chat.openai.com"){t.Fatal("widget cannot require separate connection")}
}
func TestChatFastPathTelemetryAndMeasuredLatency(t *testing.T){
 followup:=strings.Index(liveWidgetHTML,"const followUp=fn({prompt:prompt,scrollToBottom:false})")
 telemetry:=strings.Index(liveWidgetHTML,"observe(ev,mode,\"attempt\",dispatchedAt)")
 if followup<0||telemetry<0||followup>=telemetry{t.Fatal("Host message must be dispatched before telemetry RPC")}
 if !strings.Contains(liveWidgetHTML,"retryMs=300*Math.pow(2,retry)"){t.Fatal("Missing fast transient claim retry")}
 if !strings.Contains(liveWidgetHTML,"failures=Math.min(failures+1,5)"){t.Fatal("Missing bounded MCP reconnect backoff")}
 if !strings.Contains(uiHTML,"Math.round((endTime-startTime)/1000)"){t.Fatal("Missing measured end-to-end response latency")}
 if !strings.Contains(uiHTML,"refreshConsecomChat()},900)"){t.Fatal("Expected sub-second local chat updates")}
}
func TestChatResponseCannotBeAuthoredByDashboardAPIPost(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 request:=httptest.NewRequest("POST","http://127.0.0.1:8765/api/live/chat",strings.NewReader("{}"))
 request.Header.Set("X-Local-Key",a.csrf)
 request.Header.Set("Content-Type","application/json")
 rr:=httptest.NewRecorder()
 a.handler().ServeHTTP(rr,request)
 if rr.Code!=405{t.Fatalf("dashboard can inject assistant reply: %d",rr.Code)}
 var p map[string]any
 if err=json.Unmarshal(rr.Body.Bytes(),&p);err==nil && p["ok"]==true{t.Fatal("response injection")}
}
