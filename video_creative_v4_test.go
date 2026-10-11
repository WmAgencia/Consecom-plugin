package main

import (
 "bytes"
 "encoding/base64"
 "math"
 "os"
 "os/exec"
 "path/filepath"
 "testing"
)

func TestV4VideoFrameTimestampAndRoots(t *testing.T){
 for _,invalid:=range []float64{-1,36001,math.NaN(),math.Inf(1),math.Inf(-1)}{
  if frameSecondsOK(invalid){t.Fatalf("timestamp invalido aceito: %v",invalid)}
 }
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 if _,err=a.videoFrame("not-allowed","example.mp4",0);err==nil{t.Fatal("project authorization bypass")}
 project:=t.TempDir()
 a.cfg.Roots=[]Root{{ID:"project",Path:project,Name:"Video Test",Writable:false}}
 if _,err=a.videoFrame("project","../outside.mp4",0);err==nil{t.Fatal("path traversal bypass")}
 if _,err=a.videoDiagnose("project","../outside.mp4");err==nil{t.Fatal("diagnostic path traversal bypass")}
 if err=os.WriteFile(filepath.Join(project,"bad.txt"),[]byte("not a video"),0600);err!=nil{t.Fatal(err)}
 if _,err=a.videoFrame("project","bad.txt",0);err==nil{t.Fatal("disallowed extension accepted")}
}

func TestV4FFmpegVideoFrameAndDiagnostics(t *testing.T){
 ffmpeg,err:=exec.LookPath("ffmpeg")
 if err!=nil{t.Skip("FFmpeg not installed on CI runner")}
 dir:=t.TempDir()
 file:=filepath.Join(dir,"test.mp4")
 generate:=exec.Command(ffmpeg,"-nostdin","-hide_banner","-loglevel","error","-f","lavfi",
  "-i","color=c=black:s=64x64:r=10:d=2","-t","2","-c:v","mpeg4","-y",file)
 if output,err:=generate.CombinedOutput();err!=nil{t.Skipf("FFmpeg build cannot produce lavfi test clip: %v: %s",err,output)}
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 a.cfg.Roots=[]Root{{ID:"film",Name:"Film",Path:dir}}
 preview,err:=a.videoFrame("film","test.mp4",0.1);if err!=nil{t.Fatal(err)}
 frame:=preview.(VideoFrame)
 raw,err:=base64.StdEncoding.DecodeString(frame.JPEGBase64);if err!=nil{t.Fatal(err)}
 if frame.MIME!="image/jpeg"||frame.Bytes!=len(raw)||!bytes.HasPrefix(raw,[]byte{0xff,0xd8}){t.Fatal("invalid JPEG preview")}
 report,err:=a.videoDiagnose("film","test.mp4");if err!=nil{t.Fatal(err)}
 scan:=report.(VideoDiagnostic)
 if scan.AnalyzedSeconds!=20||scan.BlackEvents==nil||scan.FreezeEvents==nil{t.Fatal("invalid report schema")}
 if len(scan.BlackEvents)==0{t.Log("FFmpeg did not identify any sustained dark segment in the generated clip")}
}

func TestV4NewVideoToolsAreRegistered(t *testing.T){
 expected:=map[string]bool{"video_extrair_frame":false,"video_diagnosticar":false}
 for _,v:=range tools(){if _,ok:=expected[v.Name];ok {expected[v.Name]=true}}
 for n,exists:=range expected{if !exists{t.Errorf("MCP tool not registered: %s",n)}}
 if len(tools())!=77{t.Fatalf("expected 77 tools; got %d",len(tools()))}
}
