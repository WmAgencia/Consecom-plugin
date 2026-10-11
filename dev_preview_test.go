package main

import (
 "encoding/json"
 "net"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "testing"
 "time"
)

func previewFixture(t *testing.T)(*App,string,int){
 t.Helper()
 a,err:=NewApp(filepath.Join(t.TempDir(),"cfg","config.json"));if err!=nil{t.Fatal(err)}
 rootPath:=filepath.Join(t.TempDir(),"preview-project")
 if err=os.MkdirAll(filepath.Join(rootPath,"node_modules"),0700);err!=nil{t.Fatal(err)}
 pkg:=`{"name":"consecom-preview-fixture","version":"1.0.0","scripts":{"dev":"node server.js"},"devDependencies":{"vite":"^6.0.0"}}`
 if err=os.WriteFile(filepath.Join(rootPath,"package.json"),[]byte(pkg),0600);err!=nil{t.Fatal(err)}
 script:=`const http=require('http');const p=Number(process.env.PORT);const h=process.env.HOST;http.createServer((_,res)=>{res.writeHead(200,{'Content-Type':'text/html'});res.end('<html><head><title>Preview</title><meta name="viewport" content="width=device-width"><meta name="description" content="Teste"></head><body><h1>OK</h1></body></html>')}).listen(p,h);`
 if err=os.WriteFile(filepath.Join(rootPath,"server.js"),[]byte(script),0600);err!=nil{t.Fatal(err)}
 root,err:=a.addRoot("preview fixture",rootPath);if err!=nil{t.Fatal(err)}
 if err=a.setWritable(root.ID,true);err!=nil{t.Fatal(err)}
 l,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil{t.Fatal(err)}
 port:=l.Addr().(*net.TCPAddr).Port;l.Close()
 return a,root.ID,port
}

func TestPreviewSecurityAndApproval(t *testing.T){
 if runtime.GOOS=="windows"{if _,err:=exec.LookPath("npm");err!=nil{t.Fatalf("Windows CI requires npm: %v",err)}}
 a,rootID,port:=previewFixture(t)
 if _,err:=a.previewStart("unauthorized",port,false);err==nil{t.Fatal("unregistered project accepted")}
 if _,err:=a.previewStart(rootID,80,false);err==nil{t.Fatal("reserved port accepted")}
 if _,err:=a.previewCheck("does-not-exist");err==nil{t.Fatal("unknown id accepted")}
 v,err:=a.previewStart(rootID,port,false);if err!=nil{t.Fatal(err)}
 x:=v.(map[string]any)
 if x["status"]!="aguardando_aprovacao_local"{t.Fatalf("normal mode must queue: %v",x)}
 if len(a.pendings())!=1{t.Fatal("expected exactly one local approval")}
 for _,p:=range a.pendings(){_,_ = a.pop(p["id"].(string))}
 if err:=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 if len(a.pendings())!=0{t.Fatal("stale local prompts remain")}
 if _,err:=a.previewStart(rootID,port,false);err!=nil{t.Fatalf("maximum mode should start: %v",err)}
 active,err:=a.previewList();if err!=nil{t.Fatal(err)}
 list:=active.([]map[string]any)
 if len(list)!=1||list[0]["status"]!="running"{t.Fatalf("preview not running: %v",list)}
 id:=list[0]["id"].(string)
 t.Cleanup(func(){_,_=a.previewStop(id)})
 if _,err:=a.previewStart(rootID,port,false);err==nil{t.Fatal("duplicate preview accepted")}
 deadline:=time.Now().Add(15*time.Second)
 healthy:=false
 for time.Now().Before(deadline){
  checked,err:=a.previewCheck(id);if err!=nil{t.Fatal(err)}
  report:=checked.(map[string]any)
  if report["ready"]==true{
   if report["http_status"]!=200||report["title_present"]!=true||report["h1_present"]!=true||report["viewport_meta_present"]!=true{
    t.Fatalf("HTTP diagnostics did not find expected HTML markers: %v",report)
   }
   healthy=true;break
  }
  time.Sleep(250*time.Millisecond)
 }
 if !healthy{t.Fatal("local dev server never became healthy")}
 status,err:=a.previewStatus(id);if err!=nil{t.Fatal(err)}
 if !strings.HasPrefix(status.(map[string]any)["url"].(string),"http://127.0.0.1:"){t.Fatal("preview bound outside loopback")}
 stopped,err:=a.previewStop(id);if err!=nil{t.Fatal(err)}
 if stopped.(map[string]any)["status"]!="stopped"{t.Fatal("stop unsuccessful")}
 if _,err:=a.previewStop(id);err==nil{t.Fatal("second stop should be rejected")}
 if len(a.pendings())!=0{t.Fatal("maximum mode queued approval")}
}

func TestPreviewFrameworkRestrictions(t *testing.T){
 a,rootID,port:=previewFixture(t)
 root,err:=a.getRoot(rootID);if err!=nil{t.Fatal(err)}
 data,_:=json.Marshal(map[string]any{"name":"fake","scripts":map[string]any{"dev":"node server.js"},"dependencies":map[string]any{"express":"*"}})
 if err:=os.WriteFile(filepath.Join(root.Path,"package.json"),data,0600);err!=nil{t.Fatal(err)}
 if _,err:=a.previewStart(rootID,port,false);err==nil{t.Fatal("unsupported framework accepted")}
 if err:=os.Remove(filepath.Join(root.Path,"package.json"));err!=nil{t.Fatal(err)}
 if _,err:=a.previewStart(rootID,port,false);err==nil{t.Fatal("missing manifest accepted")}
 if err:=a.setWritable(rootID,false);err!=nil{t.Fatal(err)}
 if _,err:=a.previewStart(rootID,port,false);err==nil{t.Fatal("read-only project executed scripts")}
 if _,err:=a.previewStatus("missing");err==nil{t.Fatal("invalid id accepted")}
}

func TestPreviewToolSchema(t *testing.T){
 expected:=map[string]bool{
  "dev_preview_iniciar":false,"dev_preview_status":true,
  "dev_preview_listar":true,"dev_preview_parar":false,"dev_preview_verificar":true,
 }
 for _,tool:=range tools(){readonly,ok:=expected[tool.Name];if !ok{continue}
  if tool.Annotations["readOnlyHint"]!=readonly{t.Fatalf("read-only annotation wrong for %s",tool.Name)}
  delete(expected,tool.Name)
 }
 if len(expected)>0{t.Fatalf("missing tools %v",expected)}
}
