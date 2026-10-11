package main

import (
 "context"
 "errors"
 "os/exec"
 "strings"
 "time"
)

// Probe actual executable behavior, not merely a PATH or WindowsApps alias.
func verifyCreativeProgram(name,executable string)(string,error){
 var args []string
 switch name{
 case "FFmpeg","FFprobe","ImageMagick","Blender":args=[]string{"-version"}
 case "Python":args=[]string{"--version"}
 case "Node":args=[]string{"--version"}
 default:return "",errors.New("programa desconhecido")
 }
 ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel()
 command:=exec.CommandContext(ctx,executable,args...)
 out,err:=command.CombinedOutput()
 if ctx.Err()!=nil{return "",errors.New("verificacao de programa excedeu limite de tempo")}
 if err!=nil{return "",err}
 first:=strings.TrimSpace(strings.SplitN(string(out),"\n",2)[0])
 if len(first)>180{first=first[:180]}
 if first==""{return "",errors.New("versao nao informada")}
 return first,nil
}
