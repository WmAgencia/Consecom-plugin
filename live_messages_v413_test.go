package main

import (
 "context"
 "encoding/json"
 "net/http/httptest"
 "path/filepath"
 "strings"
 "sync"
 "testing"
 "time"
)
func TestPanelMessageRoundTripAndPersistence(t *testing.T){
 dir:=t.TempDir();path:=filepath.Join(dir,"journal.json")
 b,err:=newLiveLab(path);if err!=nil{t.Fatal(err)}
 evt,err:=b.enqueuePanelMessage("Olá ChatGPT, esta mensagem veio do meu computador.")
 if err!=nil{t.Fatal(err)}
 if evt.Kind!="PANEL_MESSAGE"||evt.Delivery!="queued"||evt.ID!="evt-000001"{t.Fatal(evt)}
 snap:=b.snapshot(0)
 if len(snap["events"].([]LiveEvent))!=1{t.Fatal("snapshot missing panel message")}
 b2,err:=newLiveLab(path);if err!=nil{t.Fatal(err)}
 if after,ok:=b2.get(evt.ID);!ok||after.Message!=evt.Message||after.Delivery!="queued"{t.Fatal("durable pending event missing")}
 claimed,err:=b2.claimPanelMessage(evt.ID);if err!=nil||!claimed{t.Fatalf("claim failed %v %v",claimed,err)}
 claimed,err=b2.claimPanelMessage(evt.ID);if err!=nil||claimed{t.Fatalf("duplicate claim accepted %v %v",claimed,err)}
 b3,err:=newLiveLab(path);if err!=nil{t.Fatal(err)}
 if after,_:=b3.get(evt.ID);after.Delivery!="claimed"{t.Fatal("claim not durable")}
 if err:=b3.report(evt.ID,"auto","host_resolved");err!=nil{t.Fatal(err)}
 b4,err:=newLiveLab(path);if err!=nil{t.Fatal(err)}
 if after,_:=b4.get(evt.ID);after.Delivery!="host_resolved"{t.Fatal("host result not durable")}
 if _,err=b4.retryPanelMessage(evt.ID);err==nil{t.Fatal("delivered event should not be retried")}
}
func TestPanelMessageSizeAndControlLimits(t *testing.T){
 for _,bad:=range []string{"","  \n ","\x00x","x\x07",strings.Repeat("X",maxPanelMessageBytes+1),string([]byte{0xff})}{
  if _,err:=validatePanelMessage(bad);err==nil{t.Fatalf("accepted invalid message %q",bad[:min(len(bad),20)])}
 }
 good,err:=validatePanelMessage(" \tOlá\nChatGPT\t ")
 if err!=nil||good!="Olá\nChatGPT"{t.Fatalf("lost content %q %v",good,err)}
}
func TestPanelQueueNotDroppedAndManualRetry(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"journal.json"));if err!=nil{t.Fatal(err)}
 first,err:=b.enqueuePanelMessage("Primeira");if err!=nil{t.Fatal(err)}
 if _,err=b.enqueuePanelMessage("Outra imediatamente");err==nil{t.Fatal("should rate limit")}
 // Directly fill with synthetic events and preserve a queued panel message.
 for i:=0;i<liveMaxEvents-1;i++{if _,err=b.emitSynthetic();err!=nil{t.Fatal(err)}}
 if _,err=b.emitSynthetic();err==nil{t.Fatal("unsent panel message dropped from journal")}
 claimed,err:=b.claimPanelMessage(first.ID);if err!=nil||!claimed{t.Fatalf("claim %v %v",claimed,err)}
 if err=b.report(first.ID,"auto","host_rejected");err!=nil{t.Fatal(err)}
 b.mu.Lock();b.lastPanelSent=time.Time{};b.mu.Unlock()
 second,err:=b.retryPanelMessage(first.ID);if err!=nil{t.Fatal(err)}
 if second.ID==first.ID||second.Delivery!="queued"{t.Fatal("retry should use new ID")}
 if old,ok:=b.get(first.ID);ok&&old.Delivery!="superseded"{t.Fatalf("superseded not saved: %+v",old)}
 if ok,err:=b.claimPanelMessage(second.ID);err!=nil||!ok{t.Fatal("new retry cannot be claimed")}
}
func TestPanelConcurrentClaimOnlyOnce(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"journal.json"));if err!=nil{t.Fatal(err)}
 evt,err:=b.enqueuePanelMessage("Somente uma janela");if err!=nil{t.Fatal(err)}
 var wg sync.WaitGroup
 wins:=make(chan bool,20)
 for i:=0;i<20;i++{wg.Add(1);go func(){defer wg.Done();claimed,err:=b.claimPanelMessage(evt.ID);wins<-err==nil&&claimed}()}
 wg.Wait();close(wins)
 count:=0;for yes:=range wins{if yes{count++}}
 if count!=1{t.Fatalf("duplicate host sends risk: claims %d",count)}
}
func TestPanelAPILocalCSRFAndAuth(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 do:=func(path,method,token,origin,body string)*httptest.ResponseRecorder{
  req:=httptest.NewRequest(method,"http://127.0.0.1:8765"+path,strings.NewReader(body))
  req.Header.Set("Content-Type","application/json")
  req.Header.Set("X-Local-Key",token)
  if origin!=""{req.Header.Set("Origin",origin)}
  r:=httptest.NewRecorder();a.handler().ServeHTTP(r,req);return r
 }
 if r:=do("/api/live/send","POST","", "",`{"message":"nao autorizado"}`);r.Code!=403{t.Fatalf("CSRF ignored %d",r.Code)}
 if r:=do("/api/live/send","POST",a.csrf,"https://evil.example",`{"message":"origem"}`);r.Code!=403{t.Fatalf("Origin ignored %d",r.Code)}
 if r:=do("/api/live/send","GET",a.csrf,"","");r.Code!=405{t.Fatalf("GET write allowed %d",r.Code)}
 r:=do("/api/live/send","POST",a.csrf,"http://127.0.0.1:8765",`{"message":"Mensagem local aprovada"}`)
 if r.Code!=200{t.Fatalf("dashboard send failed: %d %s",r.Code,r.Body.String())}
 var payload map[string]any
 if err=json.Unmarshal(r.Body.Bytes(),&payload);err!=nil{t.Fatal(err)}
 if payload["ok"]!=true{t.Fatal(payload)}
 out:=do("/api/live/messages","GET",a.csrf,"","")
 if out.Code!=200||!strings.Contains(out.Body.String(),"Mensagem local aprovada"){t.Fatalf("message missing from outbox: %d %s",out.Code,out.Body.String())}
 rpc:=liveTestRPC(t,a,"tools/list",map[string]any{})
 if !strings.Contains(liveTestJSON(rpc),"consecom_live_assumir_mensagem"){t.Fatal("claim tool not exposed through same MCP")}
 // Existing live wait still detects manually authored messages, no synthetic launch.
 waited:=a.live.wait(context.Background(),0,0)
 if len(waited["events"].([]LiveEvent))!=1{t.Fatal("missing panel message via live wait")}
}
func TestPanelMainUIContainsMessageComposer(t *testing.T){
 for _,phrase:=range []string{"Mensagem para o ChatGPT","id=\"liveUserMessage\"","liveSendButton","api('live/send'","api('live/messages'","Versão 4.13.0"}{
  if !strings.Contains(uiHTML,phrase){t.Fatalf("missing composer element %q",phrase)}
 }
 if !strings.Contains(liveWidgetHTML,"consecom_live_assumir_mensagem")||!strings.Contains(liveWidgetHTML,"PANEL_MESSAGE"){t.Fatal("widget missing manual-message delivery")}
 if strings.Contains(liveWidgetHTML,"trycloudflare.com"){t.Fatal("widget may not use a second tunnel")}
}
