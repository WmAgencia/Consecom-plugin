package main

import (
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
)

func TestMaximumPolicyReportedViaMCP(t *testing.T) {
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 result,err:=a.dispatch("politica_aprovacao",nil);if err!=nil{t.Fatal(err)}
 p:=result.(map[string]any)
 if p["mode"]!="always"||p["local_reprompt_disabled"]!=false{t.Fatalf("safe mode incorrectly reported: %#v",p)}
 if p["remote_updates_authorized"]!=false||p["project_roots_still_required"]!=true{t.Fatalf("independent protections must remain: %#v",p)}
 if err:=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 result,err=a.dispatch("politica_aprovacao",nil);if err!=nil{t.Fatal(err)}
 p=result.(map[string]any)
 if p["mode"]!="maximum"||p["local_reprompt_disabled"]!=true{t.Fatalf("maximum mode not reported: %#v",p)}
 if p["remote_updates_authorized"]!=false{t.Fatal("maximum mode unexpectedly authorized remote self-update")}
 if len(a.pendings())!=0{t.Fatal("read-only policy inspection must never queue an action")}
 found:=false
 for _,tool:=range tools(){if tool.Name=="politica_aprovacao" {
  found=true
  if tool.Annotations["readOnlyHint"]!=true{t.Fatal("policy tool must be read only")}
 }}
 if !found{t.Fatal("approval policy tool missing from MCP catalog")}
}

func TestMaximumLegacyNPMExecutionDoesNotQueue(t *testing.T) {
 if _,err:=exec.LookPath("node");err!=nil{t.Skip("Node unavailable on this runner")}
 if _,err:=exec.LookPath("npm");err!=nil{t.Skip("npm unavailable on this runner")}
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 rootPath:=filepath.Join(t.TempDir(),"developer")
 if err=os.MkdirAll(rootPath,0700);err!=nil{t.Fatal(err)}
 fixture:=`{"name":"consecom-policy-test","version":"1.0.0","scripts":{"lint":"node -e \"console.log('CONSECOM_MAXIMUM_EXECUTED')\""}}`
 if err=os.WriteFile(filepath.Join(rootPath,"package.json"),[]byte(fixture),0600);err!=nil{t.Fatal(err)}
 root,err:=a.addRoot("npm-test",rootPath);if err!=nil{t.Fatal(err)}
 queued,err:=a.developerCommand(root.ID,"npm_lint",false);if err!=nil{t.Fatal(err)}
 if queued.(map[string]any)["status"]!="aguardando_aprovacao_local" {t.Fatalf("normal mode must queue: %#v",queued)}
 for _,item:=range a.pendings(){if _,err=a.pop(item["id"].(string));err!=nil{t.Fatal(err)}}
 if err=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 result,err:=a.developerCommand(root.ID,"npm_lint",false)
 if err!=nil{t.Fatalf("maximum mode should execute fixed npm action: %v",err)}
 report:=result.(map[string]any)
 if !strings.Contains(report["saida"].(string),"CONSECOM_MAXIMUM_EXECUTED"){t.Fatalf("npm command was not executed: %#v",report)}
 if report["sem_prompt_local_modo_maximo"]!=true{t.Fatalf("mode not reflected in result: %#v",report)}
 if len(a.pendings())!=0{t.Fatal("maximum mode created a local approval for npm script")}
 if _,err=a.developerCommand(root.ID,"powershell",false);err==nil{t.Fatal("maximum mode exposed an arbitrary shell")}
 if _,err=a.developerCommand("not-authorized","npm_lint",false);err==nil{t.Fatal("maximum mode bypassed project scope")}
}

func TestMaximumDoesNotOverrideBrowserChoiceOrUpdateOptIn(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 if err=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 options:=a.browserOptions()
 if options==nil{t.Fatal("browser options missing")}
 if a.updateStatus().RemoteApproved{t.Fatal("remote updates silently enabled by maximum")}
 if len(a.roots())!=0{t.Fatal("maximum automatically added an unapproved project root")}
}
