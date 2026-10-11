package main

import (
 "context"
 "errors"
 "fmt"
 "os/exec"
 "strings"
 "time"
)

type VideoDiagnostic struct {
 File string `json:"file"`
 AnalyzedSeconds int `json:"analyzed_seconds"`
 BlackEvents []string `json:"black_events"`
 FreezeEvents []string `json:"freeze_events"`
 Notes []string `json:"notes"`
}

// Bounded read-only decoding; detects candidate black/frozen intervals.
// These heuristics do not assert whether a creative choice is a defect.
func (a *App) videoDiagnose(rootID,rel string)(any,error){
 root,err:=a.getRoot(rootID);if err!=nil{return nil,err}
 path,err:=allowedVideoPath(root,rel);if err!=nil{return nil,err}
 bin,err:=exec.LookPath("ffmpeg");if err!=nil{return nil,errors.New("FFmpeg nao disponivel")}
 ctx,cancel:=context.WithTimeout(context.Background(),35*time.Second);defer cancel()
 cmd:=exec.CommandContext(ctx,bin,
  "-nostdin","-hide_banner","-loglevel","info",
  "-protocol_whitelist","file,pipe","-i",path,
  "-map","0:v:0","-an","-t","20","-threads","2",
  "-vf","blackdetect=d=0.35:pix_th=0.10,freezedetect=n=0.003:d=2",
  "-f","null","-",
 )
 output,err:=cmd.CombinedOutput()
 if ctx.Err()!=nil{return nil,errors.New("analise excedeu 35 segundos")}
 if err!=nil{return nil,fmt.Errorf("falha na analise de video: %s (%v)",clip(string(output),400),err)}
 result:=VideoDiagnostic{File:rel,AnalyzedSeconds:20,
  BlackEvents:[]string{},FreezeEvents:[]string{},
  Notes:[]string{"Apenas primeiros 20 segundos analisados","Quadros escuros e congelados podem ser intencionais; revise visualmente"}}
 // Parse evidence only from the two filters, limiting returned events.
 for _,line:=range strings.Split(string(output),"\n"){
  line=strings.TrimSpace(line)
  if strings.Contains(line,"black_start:")&&len(result.BlackEvents)<20{result.BlackEvents=append(result.BlackEvents,clip(line,260))}
  if (strings.Contains(line,"freeze_start:")||strings.Contains(line,"freeze_end:"))&&len(result.FreezeEvents)<20{result.FreezeEvents=append(result.FreezeEvents,clip(line,260))}
 }
 return result,nil
}
