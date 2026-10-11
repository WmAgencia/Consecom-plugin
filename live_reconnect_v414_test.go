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

func TestReconnectClaimsOlderQueuedMessages(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"bridge.json"))
 if err!=nil{t.Fatal(err)}
 queued,err:=b.enqueuePanelMessage("Mensagem enviada enquanto auto estava desativado")
 if err!=nil{t.Fatal(err)}
 // Passive polling has already advanced the widget cursor past this message.
 earlier:=b.wait(context.Background(),0,0)
 if earlier["sequence"]!=queued.Sequence{t.Fatal(earlier)}
 claimed,err:=b.waitAndClaim(context.Background(),queued.Sequence,0)
 if err!=nil{t.Fatal(err)}
 ids:=claimed["claimed_ids"].([]string)
 if len(ids)!=1||ids[0]!=queued.ID{t.Fatalf("stale cursor stranded pending message: %+v",claimed)}
 stored,_:=b.get(queued.ID)
 if stored.Delivery!="claimed"||stored.ClaimedAt==""{t.Fatal(stored)}
 duplicate,err:=b.waitAndClaim(context.Background(),queued.Sequence,0)
 if err!=nil{t.Fatal(err)}
 if len(duplicate["claimed_ids"].([]string))!=0{t.Fatal("duplicate ownership")}
}

func TestReconnectDoesNotReplayUncertainDispatch(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"bridge.json"))
 if err!=nil{t.Fatal(err)}
 first,err:=b.enqueuePanelMessage("Sem confirmação da API")
 if err!=nil{t.Fatal(err)}
 if claimed,err:=b.claimPanelMessage(first.ID);err!=nil||!claimed{t.Fatal(err)}
 second:=time.Now()
 b.mu.Lock();b.lastPanelSent=second.Add(-panelMessageCooldown-time.Second);b.mu.Unlock()
 next,err:=b.enqueuePanelMessage("Fila nova")
 if err!=nil{t.Fatal(err)}
 result,err:=b.waitAndClaim(context.Background(),next.Sequence,0)
 if err!=nil{t.Fatal(err)}
 ids:=result["claimed_ids"].([]string)
 if len(ids)!=1||ids[0]!=next.ID{t.Fatalf("replayed uncertain first dispatch: %+v",result)}
}

func TestWidgetReconnectNoHardLimitAndSyntax(t *testing.T){
 for _,rule:=range []string{
  "if(polling||!active)return",
  "window.addEventListener(\"pageshow\"",
  "document.addEventListener(\"visibilitychange\"",
  "window.openai.widgetState",
  "host.setWidgetState",
  "await delay(1200)",
  "if(ownedClaim&&ev.kind===\"PANEL_MESSAGE\")ask(ev,false,true)",
 }{
  if !strings.Contains(liveWidgetHTML,rule){t.Errorf("missing recovery rule %q",rule)}
 }
 if strings.Contains(liveWidgetHTML,"MAX_ATTEMPTS")||strings.Contains(liveWidgetHTML,"Limite de envios nesta sessão"){
  t.Fatal("arbitrary 50-message session cutoff still exists")
 }
 node,err:=exec.LookPath("node")
 if err!=nil{t.Skip("node not available")}
 parts:=regexp.MustCompile("(?s)<script>(.*?)</script>").FindStringSubmatch(liveWidgetHTML)
 if len(parts)!=2{t.Fatal("Live widget script not found")}
 path:=filepath.Join(t.TempDir(),"live-widget.js")
 if err=os.WriteFile(path,[]byte(parts[1]),0600);err!=nil{t.Fatal(err)}
 if output,err:=exec.Command(node,"--check",path).CombinedOutput();err!=nil{
  t.Fatalf("Live widget JavaScript syntax: %v\\n%s",err,output)
 }
}

func TestClaimTimestampAvailableForManualRecovery(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"bridge.json"))
 if err!=nil{t.Fatal(err)}
 ev,err:=b.enqueuePanelMessage("Mensagem sem confirmação")
 if err!=nil{t.Fatal(err)}
 if ok,err:=b.claimPanelMessage(ev.ID);err!=nil||!ok{t.Fatal(err)}
 entries:=b.chatHistory()["messages"].([]LiveChatEntry)
 if entries[0].ClaimedAt==""||entries[0].DispatchedAt!=""{t.Fatal(entries)}
 if !strings.Contains(uiHTML,"Verificar e reenviar")||!strings.Contains(uiHTML,"claimWarning"){
  t.Fatal("safe retry UI missing")
 }
}
