package main

import(
 "strings"
 "testing"
)

func TestAgentBrowserURLSafety(t *testing.T){
 allowed:=[]struct{input,host string}{
  {"https://example.org/docs","example.org"},
  {"https://example.org:443","example.org"},
  {"https://github.com/topics/agentic-browser","github.com"},
  {"http://127.0.0.1:5173","127.0.0.1"},
  {"http://localhost:3000","localhost"},
 }
 for _,tc:=range allowed{
  got,host,err:=agentBrowserURL(tc.input)
  if err!=nil||got==""||host!=tc.host{t.Fatalf("expected safe URL %s, got %s, %s, %v",tc.input,got,host,err)}
 }
 rejected:=[]string{
  "file:///C:/Windows/System32","javascript:alert(1)",
  "http://example.com","https://user:password@example.com",
  "https://example.com/?api_key=secret",
  "https://example.com/#password=secret",
  "http://192.168.1.2:3000","http://10.0.0.1",
  "http://127.0.0.2:3000","http://localhost",
  "https://sub..domain.com","https://-bad.example","data:text/html,abc",
  "https://example.com\\r\\nevil",
 }
 for _,input:=range rejected{
  if _,_,err:=agentBrowserURL(input);err==nil{t.Fatalf("accepted forbidden URL: %q",input)}
 }
}

func TestAgentBrowserSnapshotRefAndSensitiveActions(t *testing.T){
 snap:=`- link "Pagina Inicial" [ref=e1]
- button "Comprar agora" [ref=e2]
- textbox "Busca" [ref=e3]
- textbox "Senha" [ref=e4]
- link "Delete Account" [ref=e5]
- link "Artigo" [ref=e20]`
 cases:=[]struct{ref,contains string}{
  {"@e1","Pagina Inicial"},{"@e2","Comprar"},{"@e3","Busca"},{"@e4","Senha"},{"@e5","Delete"},{"@e20","Artigo"},
 }
 for _,tc:=range cases{
  line,err:=agentBrowserRefLine(snap,tc.ref)
  if err!=nil||!strings.Contains(line,tc.contains){t.Fatalf("wrong ref match for %s: %q, %v",tc.ref,line,err)}
 }
 for _,ref:=range []string{"@e0","@e","@e-1","@e21","@e1;rm -rf /","@e000000000000000000"}{
  if _,err:=agentBrowserRefLine(snap,ref);err==nil{t.Fatalf("forbidden/stale ref accepted: %q",ref)}
 }
 for _,text:=range []string{"Comprar agora","Delete Account","Password","Senha","Api Key","checkout","Publicar","Cartão"}{
  if !agentBrowserUnsafeRE.MatchString(text){t.Fatalf("sensitive text unrecognized: %q",text)}
 }
}

func TestAgentBrowserToolDeclarationsAndNoShell(t *testing.T){
 expect:=map[string]bool{
  "navegador_agente_status":true,"navegador_agente_abrir":false,
  "navegador_agente_ler":true,"navegador_agente_clicar_link":false,
  "navegador_agente_preencher":false,"navegador_agente_capturar":true,
  "navegador_agente_fechar":false,
 }
 for _,tool:=range tools(){
  readOnly,exists:=expect[tool.Name]
  if !exists{continue}
  if tool.Annotations["readOnlyHint"]!=readOnly{t.Fatalf("incorrect annotation: %s",tool.Name)}
  delete(expect,tool.Name)
 }
 if len(expect)>0{t.Fatalf("missing agent-browser tools: %v",expect)}
 var app App
 if _,err:=app.agentBrowserSnapshot();err==nil{t.Fatal("read before open accepted")}
 if _,err:=app.agentBrowserClick("@e1");err==nil{t.Fatal("click before snapshot accepted")}
 if _,err:=app.agentBrowserFill("@e1","text");err==nil{t.Fatal("fill before snapshot accepted")}
 if _,err:=app.agentBrowserOpen("https://example.org?secret=1");err==nil{t.Fatal("secret URL accepted")}
 if _,err:=app.agentBrowserClose();err!=nil{t.Fatal(err)}
}

func TestAgentBrowserDeniedActionsNoExternalRun(t *testing.T){
 a:=&App{}
 a.agentBrowser.Name="consecom-test"
 a.agentBrowser.Host="example.org"
 a.agentBrowser.Allowed="example.org,*.example.org"
 a.agentBrowser.LastSnapshot=`- link "Home" [ref=e1]
- button "Pay Now" [ref=e2]
- textbox "Password" [ref=e3]
- link "Delete Account" [ref=e4]`
 if _,err:=a.agentBrowserClick("@e2");err==nil{t.Fatal("buttons cannot be clicked")}
 if _,err:=a.agentBrowserClick("@e4");err==nil{t.Fatal("delete links cannot be clicked")}
 if _,err:=a.agentBrowserFill("@e3","topsecret");err==nil{t.Fatal("password field cannot be filled")}
 if _,err:=a.agentBrowserClick("@e9");err==nil{t.Fatal("unknown element must fail")}
 if _,err:=a.agentBrowserFill("@e1","not an input");err==nil{t.Fatal("link cannot be filled")}
}
