package main

import (
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "strings"
 "testing"
)

func TestChatTypingIndicatorOnlyAfterDispatch(t *testing.T) {
 for _,token:=range []string{
  "ChatGPT está digitando",
  "live-chat-typing-dots",
  "prefers-reduced-motion",
  "Boolean(m.dispatched_at)",
  "typingFor",
  "r.kind==='final'",
 } {
  if !strings.Contains(uiHTML,token) {t.Fatalf("missing typing indicator: %q",token)}
 }
 for _,obsolete:=range []string{"Testar 10x","runConsecomTenTests","liveBenchButton","liveBenchState","liveBenchRunning","live-chat-bench"} {
  if strings.Contains(uiHTML,obsolete) {t.Fatalf("obsolete benchmark code: %q",obsolete)}
 }
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"chat.json"))
 if err!=nil {t.Fatal(err)}
 ev,err:=b.enqueuePanelMessage("Mensagem de teste manual")
 if err!=nil {t.Fatal(err)}
 entries:=b.chatHistory()["messages"].([]LiveChatEntry)
 if entries[0].DispatchedAt!="" {t.Fatal("queued message must not show typing")}
 claimed,err:=b.claimPanelMessage(ev.ID)
 if err!=nil||!claimed {t.Fatalf("claim: %v",err)}
 entries=b.chatHistory()["messages"].([]LiveChatEntry)
 if entries[0].DispatchedAt!="" {t.Fatal("claim alone does not prove dispatch")}
 if err=b.report(ev.ID,"auto","attempt");err!=nil {t.Fatal(err)}
 entries=b.chatHistory()["messages"].([]LiveChatEntry)
 if entries[0].DispatchedAt=="" {t.Fatal("dispatched message timestamp missing")}
 if _,err=b.publishChatReply(ev.ID,"Resposta real registrada","final","final");err!=nil {t.Fatal(err)}
 entries=b.chatHistory()["messages"].([]LiveChatEntry)
 if len(entries)!=2||entries[0].Status!="answered"||entries[1].Kind!="final" {
  t.Fatalf("typing must stop after final response: %+v",entries)
 }
}

func TestDashboardJavaScriptSyntaxWithNode(t *testing.T) {
 node,err:=exec.LookPath("node")
 if err!=nil {t.Skip("Node.js not installed on runner")}
 scripts:=regexp.MustCompile("(?s)<script>(.*?)</script>").FindStringSubmatch(uiHTML)
 if len(scripts)!=2 {t.Fatal("dashboard script not found")}
 path:=filepath.Join(t.TempDir(),"dashboard.js")
 if err=os.WriteFile(path,[]byte(scripts[1]),0600);err!=nil {t.Fatal(err)}
 if output,err:=exec.Command(node,"--check",path).CombinedOutput();err!=nil {
  t.Fatalf("invalid JavaScript: %v\n%s",err,output)
 }
}
