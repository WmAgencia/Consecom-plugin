package main

import (
 "context"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "strings"
 "testing"
 "time"
)

func TestTenAutomaticLocalBridgeCyclesWithoutOpenAI(t *testing.T){
 // This verifies queue/atomic claim/timestamps/persistence only.
 // It MUST NOT be presented as ten end-to-end ChatGPT turns.
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"ten-cycles.json"))
 if err!=nil{t.Fatal(err)}
 var cursor uint64
 var maxDispatchMs int64
 for i:=1;i<=10;i++{
  b.mu.Lock()
  b.lastPanelSent=time.Now().Add(-panelMessageCooldown-time.Second)
  b.mu.Unlock()
  type outcome struct{response map[string]any;err error}
  waiting:=make(chan outcome,1)
  go func(c uint64){
   response,e:=b.waitAndClaim(context.Background(),c,2000)
   waiting<-outcome{response,e}
  }(cursor)
  time.Sleep(15*time.Millisecond)
  ev,e:=b.enqueuePanelMessage("Teste de integracao local")
  if e!=nil{t.Fatalf("cycle %d enqueue: %v",i,e)}
  var result outcome
  select{
  case result=<-waiting:
  case <-time.After(2500*time.Millisecond):t.Fatalf("cycle %d local claim timeout",i)
  }
  if result.err!=nil{t.Fatalf("cycle %d claim failed: %v",i,result.err)}
  ids,ok:=result.response["claimed_ids"].([]string)
  if !ok||len(ids)!=1||ids[0]!=ev.ID{t.Fatalf("cycle %d incorrect claimed IDs: %#v",i,result.response)}
  cursor=ev.Sequence
  // Fake host and fake model are explicit test doubles; no API calls.
  if e=b.report(ev.ID,"auto","attempt");e!=nil{t.Fatal(e)}
  if e=b.report(ev.ID,"auto","host_resolved");e!=nil{t.Fatal(e)}
  reply,e:=b.publishChatReply(ev.ID,"OK simulado","final","")
  if e!=nil{t.Fatalf("cycle %d final: %v",i,e)}
  if reply.EventID!=ev.ID{t.Fatalf("cycle %d mismatched reply",i)}
  if _,e=b.markPanelDisplayed(ev.ID,reply.ID,"");e!=nil{t.Fatal(e)}
  current,_:=b.get(ev.ID)
  duration,valid:=liveStageMS(current.CreatedAt,current.FollowUpAt)
  if !valid{t.Fatalf("cycle %d incomplete dispatch metrics: %+v",i,current)}
  if duration>maxDispatchMs{maxDispatchMs=duration}
  if duration>15000{t.Fatalf("cycle %d local claim unexpectedly slower than 15s: %dms",i,duration)}
  t.Logf("SIMULACAO local %02d/10: fila+claim+despacho=%dms (host/modelo simulados)",i,duration)
 }
 metrics:=b.latencyReport()
 if metrics["summary"].(map[string]any)["completed"]!=10{t.Fatalf("10 final replies expected: %+v",metrics["summary"])}
 if len(metrics["messages"].([]map[string]any))!=10{t.Fatal("expected exactly 10 unique events")}
 restarted,e:=newLiveLab(b.journal)
 if e!=nil{t.Fatal(e)}
 if len(restarted.chatHistory()["messages"].([]LiveChatEntry))!=20{t.Fatal("10 requests and 10 responses must persist")}
 t.Logf("10/10 CICLOS SIMULADOS: maior atraso local=%dms; nao mede ChatGPT real",maxDispatchMs)
}

func TestChatTypingPlaceholderTruthfulnessAndBenchmarkUI(t *testing.T){
 for _,token:=range []string{
  "ChatGPT está digitando",
  "dispatched_at",
  "m.status==='host_resolved'||Boolean(m.dispatched_at)",
  "typingFor",
  "live-chat-typing-dots",
  "prefers-reduced-motion",
  "runConsecomTenTests",
  "▶ Testar 10x",
  "for(let i=1;i<=10;i++)",
  "kind==='final'",
  "const p90=values[Math.ceil(values.length*.9)-1]",
  "liveBenchRunning=true",
 }{
  if !strings.Contains(uiHTML,token){t.Fatalf("missing UI safeguard %q",token)}
 }
 b,e:=newLiveLab(filepath.Join(t.TempDir(),"chat.json"));if e!=nil{t.Fatal(e)}
 ev,e:=b.enqueuePanelMessage("mensagem inicial");if e!=nil{t.Fatal(e)}
 before:=b.chatHistory()["messages"].([]LiveChatEntry)
 if before[0].DispatchedAt!=""{t.Fatal("queued message must not show typing")}
 if ok,e:=b.claimPanelMessage(ev.ID);e!=nil||!ok{t.Fatal(e)}
 claimed:=b.chatHistory()["messages"].([]LiveChatEntry)
 if claimed[0].DispatchedAt!=""{t.Fatal("claimed alone must not impersonate generated text")}
 if e:=b.report(ev.ID,"auto","attempt");e!=nil{t.Fatal(e)}
 sent:=b.chatHistory()["messages"].([]LiveChatEntry)
 if sent[0].DispatchedAt==""{t.Fatal("verified dispatch milestone not exposed")}
}

func TestConsecomPanelEmbeddedJavaScriptSyntaxWithNode(t *testing.T){
 node,err:=exec.LookPath("node")
 if err!=nil{t.Skip("node not available")}
 parts:=regexp.MustCompile("(?s)<script>(.*?)</script>").FindStringSubmatch(uiHTML)
 if len(parts)!=2{t.Fatal("embedded dashboard script missing")}
 path:=filepath.Join(t.TempDir(),"panel-embed.js")
 if err=os.WriteFile(path,[]byte(parts[1]),0600);err!=nil{t.Fatal(err)}
 command:=exec.Command(node,"--check",path)
 if out,err:=command.CombinedOutput();err!=nil{t.Fatalf("embedded dashboard JS invalid: %v\n%s",err,out)}
}
