package main

import(
 "encoding/json"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
)

func testProjectV45(t *testing.T)(*App,Root){
 t.Helper()
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 dir:=filepath.Join(t.TempDir(),"project")
 if err:=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
 pkg:=`{"name":"v45-fixture","scripts":{"lint":"node -e \"console.log('lint ok')\"","test":"node -e \"console.log('test ok')\"","build":"node -e \"console.log('build ok')\""},"dependencies":{"vite":"^6"}}`
 if err:=os.WriteFile(filepath.Join(dir,"package.json"),[]byte(pkg),0600);err!=nil{t.Fatal(err)}
 root,err:=a.addRoot("test project",dir);if err!=nil{t.Fatal(err)}
 if err:=a.setWritable(root.ID,true);err!=nil{t.Fatal(err)}
 root,err=a.getRoot(root.ID);if err!=nil{t.Fatal(err)}
 return a,root
}

func TestPhase2BrowserQAToolSchemas(t *testing.T){
 names:=map[string]bool{"browser_qa_auditar":true,"browser_qa_responsivo":true,"browser_qa_erros":true}
 for _,item:=range tools(){
  if !names[item.Name]{continue}
  if item.Annotations["readOnlyHint"]!=true{t.Fatalf("QA tool should be read-only: %s",item.Name)}
  delete(names,item.Name)
 }
 if len(names)!=0{t.Fatalf("QA tools missing: %v",names)}
 for _,v:=range []string{"mobile","tablet","desktop"}{
  d,e:=qaSafeViewport(v);if e!=nil||d[0]<320||d[1]<600{t.Fatalf("viewport failure for %q: %v",v,e)}
 }
 if _,e:=qaSafeViewport("999999");e==nil{t.Fatal("arbitrary CDP viewport accepted")}
 if !strings.Contains(browserQAAuditScript,"horizontal_overflow")||strings.Contains(browserQAAuditScript,"document.cookie"){
  t.Fatal("unexpected browser QA script or cookie access")
 }
}

func TestPhase3PlanAndQuality(t *testing.T){
 a,root:=testProjectV45(t)
 steps,err:=repairStepsForRoot(root);if err!=nil{t.Fatal(err)}
 want:=[]string{"npm_lint","npm_test","npm_build"}
 if len(steps)!=len(want){t.Fatalf("unexpected steps: %v",steps)}
 for i:=range steps{if steps[i]!=want[i]{t.Fatalf("wrong order %v",steps)}}
 plan,err:=a.repairPlan(root.ID);if err!=nil{t.Fatal(err)}
 if plan.(map[string]any)["automatic_code_generation"]!=false{t.Fatal("must not promise hidden AI model")}
 if _,err:=a.repairPlan("unauthorized");err==nil{t.Fatal("unauthorized root accepted")}
 if _,err:=a.repairStart("unauthorized");err==nil{t.Fatal("unauthorized workflow accepted")}
}

func TestPhase4MetadataAndNoRemoteDeployment(t *testing.T){
 a,root:=testProjectV45(t)
 b:=DevServiceBinding{RootID:root.ID,GitHubRepository:"WmAgencia/demo-site",VercelProjectID:"prj_demo",
  VercelTeamID:"team_demo",SupabaseProjectRef:"sampleprojectref",RailwayProjectID:"demo-project",Environment:"preview"}
 if err:=validateServiceBinding(b);err!=nil{t.Fatal(err)}
 for _,bad:=range []DevServiceBinding{
  {RootID:root.ID,GitHubRepository:"../../secrets",Environment:"preview"},
  {RootID:root.ID,GitHubRepository:"a/b",VercelProjectID:"prj_/../../token",Environment:"preview"},
  {RootID:root.ID,GitHubRepository:"a/b",Environment:"develop"},
 }{if err:=validateServiceBinding(bad);err==nil{t.Fatalf("invalid binding accepted: %+v",bad)}}
 normal,err:=a.serviceBindingSave(b,false);if err!=nil{t.Fatal(err)}
 if normal.(map[string]any)["status"]!="aguardando_aprovacao_local"{t.Fatal("normal mode must request local approval")}
 if err:=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 saved,err:=a.serviceBindingSave(b,false);if err!=nil{t.Fatal(err)}
 if saved.(map[string]any)["remote_modified"]!=false{t.Fatal("mapping should never modify cloud")}
 found,err:=a.serviceBindingGet(root.ID);if err!=nil{t.Fatal(err)}
 actual:=found.(map[string]any)["binding"].(DevServiceBinding)
 if actual.GitHubRepository!=b.GitHubRepository{t.Fatal("binding not persisted")}
 raw,_:=json.Marshal(actual)
 if strings.Contains(string(raw),"api_key")||strings.Contains(string(raw),"password"){t.Fatal("binding leaked credentials")}
 result,err:=a.deployPreflight(root.ID);if err!=nil{t.Fatal(err)}
 report:=result.(map[string]any)
 if report["ready"]==true||report["remote_modified"]!=false{t.Fatalf("unsafe preflight outcome: %+v",report)}
 // Never invoke a live provider API from these tests.
 if _,err:=a.serviceBindingGet("unauthorized");err==nil{t.Fatal("unapproved root metadata read")}
}

func TestPhase4GitCleanPreflightIsReadOnly(t *testing.T){
 if _,err:=exec.LookPath("git");err!=nil{t.Skip("Git unavailable")}
 a,root:=testProjectV45(t)
 cmd:=exec.Command("git","init");cmd.Dir=root.Path
 if out,err:=cmd.CombinedOutput();err!=nil{t.Fatalf("git init: %s: %v",string(out),err)}
 b:=DevServiceBinding{RootID:root.ID,GitHubRepository:"someone/demo",VercelProjectID:"prj_demo",Environment:"preview"}
 if err:=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 if _,err:=a.serviceBindingSave(b,false);err!=nil{t.Fatal(err)}
 result,err:=a.deployPreflight(root.ID);if err!=nil{t.Fatal(err)}
 info:=result.(map[string]any)
 if info["git_clean"]!=false||info["ready_for_provider_verification"]!=false{
  t.Fatalf("untracked package file must block deployment: %+v",info)
 }
}

func TestPhase3And4ReadOnlyMetadata(t *testing.T){
 need:=map[string]bool{
  "dev_reparo_planejar":true,"dev_reparo_iniciar":false,
  "dev_servicos_associar":false,"dev_servicos_inspecionar":true,"dev_deploy_preflight":true,
 }
 for _,tool:=range tools(){ro,ok:=need[tool.Name];if !ok{continue}
  if tool.Annotations["readOnlyHint"]!=ro{t.Fatalf("wrong read-only flag for %s",tool.Name)}
  delete(need,tool.Name)
 }
 if len(need)>0{t.Fatalf("missing tools: %v",need)}
}


func TestBrowserQARecorderDoesNotExportConsolePayloads(t *testing.T){
 if !strings.Contains(browserQARecorderScript,"console.error")||!strings.Contains(browserQARecorderReadScript,"categories"){
  t.Fatal("console error recorder missing")
 }
 if strings.Contains(browserQARecorderReadScript,"JSON.stringify(args)")||strings.Contains(browserQARecorderReadScript,"document.cookie"){
  t.Fatal("QA recorder may expose sensitive content")
 }
 found:=false
 for _,tool:=range tools(){if tool.Name=="browser_qa_instrumentar"{
  found=true
  if tool.Annotations["readOnlyHint"]!=false{t.Fatal("runtime instrumentation is not truly read-only")}
 }}
 if !found{t.Fatal("QA instrumentation not found")}
}
