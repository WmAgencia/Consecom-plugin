package main

import (
 "context"
 "path/filepath"
 "sync"
 "strings"
 "testing"
 "time"
)

func TestLiveFastClaimExactlyOnceWithCompetingWidgets(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"live-fast.json"));if err!=nil{t.Fatal(err)}
 message,err:=b.enqueuePanelMessage("Um unico despacho");if err!=nil{t.Fatal(err)}
 var wg sync.WaitGroup
 results:=make(chan map[string]any,6)
 for i:=0;i<6;i++{
  wg.Add(1)
  go func(){
   defer wg.Done()
   result,err:=b.waitAndClaim(context.Background(),0,0)
   if err!=nil{t.Errorf("waitAndClaim: %v",err);return}
   results<-result
  }()
 }
 wg.Wait();close(results)
 totalClaims:=0
 for r:=range results{
  ids,ok:=r["claimed_ids"].([]string)
  if !ok{t.Fatalf("missing claimed_ids: %#v",r)}
  for _,id:=range ids{if id!=message.ID{t.Fatalf("foreign event %s",id)}}
  totalClaims+=len(ids)
 }
 if totalClaims!=1{t.Fatalf("duplicate dispatch risk, %d widgets claimed",totalClaims)}
 stored,ok:=b.get(message.ID)
 if !ok||stored.Delivery!="claimed"||stored.ClaimedAt==""{t.Fatalf("not persisted: %+v",stored)}
 afterRestart,err:=newLiveLab(b.journal);if err!=nil{t.Fatal(err)}
 persistent,_:=afterRestart.get(message.ID)
 if persistent.Delivery!="claimed"{t.Fatal("claim lost on restart")}
}

func TestLiveFastClaimWakeWithMessageAndHeartbeat(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"bridge.json"));if err!=nil{t.Fatal(err)}
 type outcome struct{ data map[string]any; err error }
 done:=make(chan outcome,1)
 go func(){r,e:=b.waitAndClaim(context.Background(),0,3500);done<-outcome{r,e}}()
 time.Sleep(70*time.Millisecond)
 ev,err:=b.enqueuePanelMessage("Acordou");if err!=nil{t.Fatal(err)}
 select{
 case result:=<-done:
  if result.err!=nil{t.Fatal(result.err)}
  ids:=result.data["claimed_ids"].([]string)
  if len(ids)!=1||ids[0]!=ev.ID{t.Fatalf("event not owned by waiting widget: %+v",result)}
  events:=result.data["events"].([]LiveEvent)
  if len(events)!=1||events[0].Delivery!="claimed"{t.Fatalf("claim not in response: %+v",events)}
 case <-time.After(2500*time.Millisecond):t.Fatal("fast path did not wake on local message")
 }
 status:=b.connectionStatus()
 if status["status"]!="ready"||status["automatic_ready"]!=true{t.Fatalf("bridge was not marked ready: %+v",status)}
}

func TestLiveFastClaimRespectsLegacyClaimAndPassiveStatus(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"bridge.json"));if err!=nil{t.Fatal(err)}
 ev,err:=b.enqueuePanelMessage("Outra janela");if err!=nil{t.Fatal(err)}
 claimed,err:=b.claimPanelMessage(ev.ID);if err!=nil||!claimed{t.Fatal(err)}
 r,err:=b.waitAndClaim(context.Background(),0,0);if err!=nil{t.Fatal(err)}
 if len(r["claimed_ids"].([]string))!=0{t.Fatal("claimed already reserved event")}
 if _,err=b.publishChatReply(ev.ID,"Teste","final","");err!=nil{t.Fatal(err)}
 b.mu.Lock()
 b.lastLivePoll=time.Now().Add(-20*time.Second)
 b.lastLiveAutoPoll=time.Now().Add(-20*time.Second)
 b.mu.Unlock()
 if s:=b.connectionStatus();s["status"]!="disconnected"{t.Fatalf("stale bridge reported ready: %+v",s)}
 // Old resource protocol remains usable and must not assert automatic readiness.
 b.wait(context.Background(),ev.Sequence,0)
 if s:=b.connectionStatus();s["status"]!="passive"||s["automatic_ready"]!=false{t.Fatalf("legacy mode incorrectly marked ready: %+v",s)}
}

func TestLiveFastClaimWidgetUsesOwnedIDsAndNoOverlappingPoll(t *testing.T){
 for _,required:=range []string{
  "consecom_live_aguardar_e_assumir",
  "const owned=new Set(",
  "await show(ev,owned.has(ev.id))",
  "if(ownedClaim)ask(ev,false,true)",
  "else if(ev.delivery===\"queued\")await claimAndSend(ev,false)",
 }{
  if !containsLiveWidget(required){t.Fatalf("missing fast-path rule %q",required)}
 }
}

func containsLiveWidget(s string)bool{return strings.Contains(liveWidgetHTML,s)}
