package main

import (
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestDevStackHubReadOnlyAndNoCredentials(t *testing.T) {
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 result,err:=a.dispatch("dev_integracoes_status",json.RawMessage(`{}`));if err!=nil{t.Fatal(err)}
 report:=result.(map[string]any)
 if report["authorized_project_count"].(int)!=0{t.Fatal("must not infer or add project roots")}
 cli,ok:=report["local_clis"].([]LocalDevCLI);if !ok||len(cli)<7{t.Fatalf("missing local stack probes: %#v",report)}
 for _,item:=range cli{
  if item.Provider==""||item.Executable==""{t.Fatalf("malformed CLI descriptor: %#v",item)}
 }
 data,_:=json.Marshal(report)
 if strings.Contains(strings.ToLower(string(data)),"access_token")||strings.Contains(strings.ToLower(string(data)),"private_key"){
  t.Fatal("integration diagnosis leaked credentials")
 }
 if len(a.pendings())!=0{t.Fatal("status should never queue approvals")}
}

func TestDevProjectInspectFrameworkAndSecretRedaction(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 dir:=filepath.Join(t.TempDir(),"web-project")
 if err=os.MkdirAll(filepath.Join(dir,"src"),0700);err!=nil{t.Fatal(err)}
 manifest:=`{"name":"public-site","scripts":{"build":"echo TOP_SECRET_SHOULD_NOT_APPEAR","dev":"vite"},"dependencies":{"react":"^18.0.0"},"devDependencies":{"vite":"^5.0.0"}}`
 if err=os.WriteFile(filepath.Join(dir,"package.json"),[]byte(manifest),0600);err!=nil{t.Fatal(err)}
 if err=os.WriteFile(filepath.Join(dir,".env.local"),[]byte("TOKEN=VERY_PRIVATE_VALUE"),0600);err!=nil{t.Fatal(err)}
 if err=os.WriteFile(filepath.Join(dir,"vercel.json"),[]byte("{}"),0600);err!=nil{t.Fatal(err)}
 root,err:=a.addRoot("web-project",dir);if err!=nil{t.Fatal(err)}
 args,_:=json.Marshal(map[string]string{"root_id":root.ID})
 res,err:=a.dispatch("dev_projeto_inspecionar",args);if err!=nil{t.Fatal(err)}
 r:=res.(map[string]any)
 if r["framework"]!="Vite"||r["vercel_configured"]!=true{t.Fatalf("did not discover Vite/Vercel: %#v",r)}
 if r["root_id"]!=root.ID{t.Fatalf("wrong project scope: %#v",r)}
 scripts:=r["scripts"].([]string)
 if len(scripts)!=2||scripts[0]!="build"||scripts[1]!="dev"{t.Fatalf("scripts must contain sorted keys only: %v",scripts)}
 payload,_:=json.Marshal(r)
 if strings.Contains(string(payload),"TOP_SECRET_SHOULD_NOT_APPEAR")||strings.Contains(string(payload),"VERY_PRIVATE_VALUE"){
  t.Fatal("project diagnosis leaked script or .env values")
 }
 if len(a.pendings())!=0{t.Fatal("read-only inspection queued approval")}
 if _,err=a.devProjectInspect("unknown-project");err==nil{t.Fatal("unauthorized project inspected")}
}

func TestDevProjectInspectsNoPackageJSON(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 dir:=filepath.Join(t.TempDir(),"static-site");if err=os.MkdirAll(dir,0700);err!=nil{t.Fatal(err)}
 if err=os.WriteFile(filepath.Join(dir,"index.html"),[]byte("<p>ok</p>"),0600);err!=nil{t.Fatal(err)}
 root,err:=a.addRoot("static-site",dir);if err!=nil{t.Fatal(err)}
 res,err:=a.devProjectInspect(root.ID);if err!=nil{t.Fatal(err)}
 if res.(map[string]any)["framework"]!="HTML estatico"{t.Fatalf("static site not discovered: %#v",res)}
}

func TestDevStackToolsDeclaredReadOnly(t *testing.T){
 wanted:=map[string]bool{"dev_integracoes_status":true,"dev_projeto_inspecionar":true}
 for _,tool:=range tools(){if !wanted[tool.Name]{continue};if tool.Annotations["readOnlyHint"]!=true{t.Fatalf("%s should be read only",tool.Name)};delete(wanted,tool.Name)}
 if len(wanted)!=0{t.Fatalf("missing tools: %#v",wanted)}
}
