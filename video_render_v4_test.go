package main

import (
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func TestV4RenderPresetAndAuthorization(t *testing.T){
 for _,name:=range []string{"story","landscape","square"}{
  f,err:=renderPreset(name);if err!=nil||!strings.Contains(f,"scale="){t.Fatalf("preset %s: %v",name,err)}
 }
 if _,err:=renderPreset("shell");err==nil{t.Fatal("unknown preset accepted")}
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 if _,err=a.videoRenderRequest(RenderSpec{RootID:"unknown",Preset:"story",Input:"x.mp4",Duration:2});err==nil{t.Fatal("unauthorized root accepted")}
 dir:=t.TempDir()
 if err=os.WriteFile(filepath.Join(dir,"example.mp4"),[]byte("invalid video"),0600);err!=nil{t.Fatal(err)}
 a.cfg.Roots=[]Root{{ID:"root",Name:"Read Only",Path:dir,Writable:false}}
 for _,duration:=range []int{0,-1,181}{
  if _,err=a.videoRenderRequest(RenderSpec{RootID:"root",Preset:"story",Input:"example.mp4",Duration:duration});err==nil{t.Fatal("invalid duration accepted")}
 }
 if _,err=a.videoRenderRequest(RenderSpec{RootID:"root",Preset:"story",Input:"example.mp4",Duration:2});err==nil{t.Fatal("read only project accepted")}
 if _,err=a.videoRenderStatus("unknown");err==nil{t.Fatal("unknown render status accepted")}
 if _,err=a.videoRenderCancel("unknown");err==nil{t.Fatal("unknown render cancelled")}
}

func TestV4VideoRenderWithRealFFmpeg(t *testing.T){
 ffmpeg,err:=exec.LookPath("ffmpeg")
 if err!=nil{t.Skip("No FFmpeg present on Windows runner")}
 dir:=t.TempDir()
 input:=filepath.Join(dir,"test.mp4")
 generation:=exec.Command(ffmpeg,"-nostdin","-hide_banner","-loglevel","error","-f","lavfi",
  "-i","testsrc2=s=160x90:r=10:d=1","-t","1","-c:v","mpeg4","-y",input)
 if out,err:=generation.CombinedOutput();err!=nil{t.Skipf("This FFmpeg lacks lavfi test generator: %v %s",err,out)}
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 a.cfg.Roots=[]Root{{ID:"renders",Name:"Renders",Path:dir,Writable:true}}
 spec:=RenderSpec{RootID:"renders",Input:"test.mp4",Preset:"square",Start:0,Duration:1}
 preliminary,err:=a.videoRenderRequest(spec);if err!=nil{t.Fatal(err)}
 if preliminary.(map[string]any)["status"]!="aguardando_aprovacao_local"{t.Fatal("normal mode should queue render")}
 if _,err=os.Stat(filepath.Join(dir,"consecom-renders"));!os.IsNotExist(err){t.Fatal("render started before approval")}
 if err=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 resp,err:=a.videoRenderRequest(spec);if err!=nil{t.Fatal(err)}
 id:=resp.(map[string]any)["id"].(string)
 deadline:=time.After(60*time.Second)
 for{
  select{case <-deadline:t.Fatal("render timeout");default:}
  result,err:=a.videoRenderStatus(id);if err!=nil{t.Fatal(err)}
  j:=result.(RenderTask)
  if j.Status=="running"{time.Sleep(250*time.Millisecond);continue}
  if j.Status!="done"{t.Fatalf("render status=%s error=%s log=%s",j.Status,j.Error,j.Logs)}
  path:=filepath.Join(dir,j.Output)
  info,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
  if !info.Mode().IsRegular()||info.Size()!=j.OutputBytes||info.Size()<128{t.Fatal("rendered MP4 size mismatch")}
  if _,err=a.videoRenderCancel(id);err==nil{t.Fatal("finished render was cancellable")}
  break
 }
}
