package main

import (
 "context"
 "crypto/sha256"
 "encoding/base64"
 "encoding/hex"
 "errors"
 "fmt"
 "math"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "strings"
 "time"
)

// VideoFrame is a read-only, bounded JPEG capture for creative review.
type VideoFrame struct {
 File string `json:"file"`
 AtSeconds float64 `json:"at_seconds"`
 JPEGBase64 string `json:"jpeg_base64"`
 SHA256 string `json:"sha256"`
 Bytes int `json:"bytes"`
 MIME string `json:"mime"`
}

func allowedVideoPath(root Root,relative string)(string,error){
 path,err:=safePath(root,relative,false);if err!=nil{return "",err}
 switch strings.ToLower(filepath.Ext(path)){
 case ".mp4",".mov",".mkv",".webm":
 default:return "",errors.New("extensao de video nao autorizada")
 }
 stat,err:=os.Lstat(path);if err!=nil{return "",err}
 if !stat.Mode().IsRegular()||stat.Size()>5*1024*1024*1024{return "",errors.New("video irregular ou acima de 5 GiB")}
 return path,nil
}

func frameSecondsOK(seconds float64)bool{
 return seconds>=0&&seconds<=36000&&!math.IsInf(seconds,0)&&!math.IsNaN(seconds)
}

func (a *App) videoFrame(rootID,relative string,seconds float64)(any,error){
 if !frameSecondsOK(seconds){return nil,errors.New("timestamp deve ser de 0 a 36000 segundos")}
 root,err:=a.getRoot(rootID);if err!=nil{return nil,err}
 path,err:=allowedVideoPath(root,relative);if err!=nil{return nil,err}
 bin,err:=exec.LookPath("ffmpeg");if err!=nil{return nil,errors.New("FFmpeg ausente; verifique programas_criativos")}
 ctx,cancel:=context.WithTimeout(context.Background(),25*time.Second);defer cancel()
 cmd:=exec.CommandContext(ctx,bin,
  "-nostdin","-hide_banner","-loglevel","error",
  "-ss",strconv.FormatFloat(seconds,'f',3,64),
  "-protocol_whitelist","file,pipe",
  "-i",path,
  "-map","0:v:0","-an","-frames:v","1",
  "-vf","scale=640:360:force_original_aspect_ratio=decrease",
  "-q:v","5","-f","image2pipe","-vcodec","mjpeg","pipe:1",
 )
 out,err:=cmd.Output()
 if ctx.Err()!=nil{return nil,errors.New("extracao de frame excedeu 25 segundos")}
 if err!=nil{return nil,fmt.Errorf("FFmpeg nao conseguiu extrair frame: %w",err)}
 if len(out)<4||out[0]!=0xff||out[1]!=0xd8||out[len(out)-2]!=0xff||out[len(out)-1]!=0xd9{return nil,errors.New("FFmpeg retornou JPEG invalido")}
 if len(out)>512*1024{return nil,errors.New("frame excede limite de 512 KiB")}
 sum:=sha256.Sum256(out)
 return VideoFrame{File:relative,AtSeconds:seconds,JPEGBase64:base64.StdEncoding.EncodeToString(out),SHA256:hex.EncodeToString(sum[:]),Bytes:len(out),MIME:"image/jpeg"},nil
}
