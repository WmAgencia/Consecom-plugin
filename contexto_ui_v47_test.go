package main

import(
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
)

func TestContextUIHasNavigationGraphAndOptInVault(t *testing.T){
 required:=[]string{
  "id=\"navContext\"","id=\"contextPage\"","id=\"contextSVG\"",
  "contextVaultSelect","selectContextVault()","openContextSource()","openContextNote()",
  "contextZoom(","function drawContext()","function showContextNode(",
  "Contexto","Consecom/Contexto","api('contexto'",
 }
 for _,fragment:=range required{
  if !strings.Contains(uiHTML,fragment){t.Errorf("missing Contexto UI: %s",fragment)}
 }
 if strings.Contains(uiHTML,"location.hash==='#contexto')page('panel')"){
  t.Fatal("Contexto route should not show main panel")
 }
}

func TestContextUIJavaScriptSyntaxInActualNode(t *testing.T){
 node,err:=exec.LookPath("node")
 if err!=nil{t.Skip("Node.js not installed")}
 start:=strings.Index(uiHTML,"<script>")
 end:=strings.LastIndex(uiHTML,"</script>")
 if start<0||end<start{t.Fatal("panel HTML has no JS script")}
 js:=uiHTML[start+len("<script>"):end]
 if len(js)<1000{t.Fatal("empty UI JS")}
 file:=filepath.Join(t.TempDir(),"consecom-ui-check.js")
 if err:=os.WriteFile(file,[]byte(js),0600);err!=nil{t.Fatal(err)}
 cmd:=exec.Command(node,"--check",file)
 if out,err:=cmd.CombinedOutput();err!=nil{
  t.Fatalf("panel JavaScript failed node --check: %v %s",err,string(out))
 }
}
