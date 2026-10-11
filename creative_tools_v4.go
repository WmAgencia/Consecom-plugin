package main

import(
 "context"
 "encoding/json"
 "errors"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "time"
)
type CreativeProgram struct{
 Name string `json:"name"`
 Available bool `json:"available"`
 Executable string `json:"executable,omitempty"`
 Purpose string `json:"purpose"`
}
func creativeTools()(any,error){
 targets:=[]struct{name string;bin string;purpose string}{
  {"FFmpeg","ffmpeg","Composicao, cortes e exportacao de video"},
  {"FFprobe","ffprobe","Analise de formatos e trilhas de video"},
  {"Blender","blender","Modelagem, animacao e renderizacao 3D"},
  {"ImageMagick","magick","Tratamento e conversao de imagens"},
  {"Python","python","Automacao de tarefas criativas"},
  {"Node","node","Execucao de projetos Remotion"},
 }
 list:=[]CreativeProgram{}
 for _,p:=range targets{
  item:=CreativeProgram{Name:p.name,Purpose:p.purpose}
  if found,e:=exec.LookPath(p.bin);e==nil {item.Available=true;item.Executable=found}
  if !item.Available&&runtime.GOOS=="windows"{
   // Read only expected install paths: no broad disk scanning or credentials.
   for _,base:=range []string{os.Getenv("PROGRAMFILES"),os.Getenv("PROGRAMFILES(X86)") ,os.Getenv("LOCALAPPDATA")}{
    if base==""{continue}
    candidates:=[]string{}
    if p.bin=="blender"{candidates=[]string{"Blender Foundation/Blender 5.2/blender.exe","Blender Foundation/Blender 4.5/blender.exe"}}
    if p.bin=="magick"{candidates=[]string{"ImageMagick-7.1.2-Q16-HDRI/magick.exe"}}
    for _,sub:=range candidates{
     f:=filepath.Join(base,filepath.FromSlash(sub))
     if stat,e:=os.Stat(f);e==nil&&stat.Mode().IsRegular(){item.Available=true;item.Executable=f;break}
    }
    if item.Available{break}
   }
  }
  list=append(list,item)
 }
 return map[string]any{"programs":list,"note":"Remotion e uma dependencia por projeto npm; Node instalado nao significa Remotion instalado"},nil
}
func (a *App) inspectVideo(rootID,relative string)(any,error){
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 path,e:=safePath(root,relative,false);if e!=nil{return nil,e}
 ext:=strings.ToLower(filepath.Ext(path))
 if ext!=".mp4"&&ext!=".mov"&&ext!=".mkv"&&ext!=".webm"{return nil,errors.New("formato de video nao permitido")}
 stat,e:=os.Lstat(path);if e!=nil{return nil,e}
 if !stat.Mode().IsRegular()||stat.Size()>5*1024*1024*1024{return nil,errors.New("arquivo de video irregular ou acima de 5 GiB")}
 bin,e:=exec.LookPath("ffprobe");if e!=nil{return nil,errors.New("FFprobe nao instalado; verifique programas_criativos")}
 ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel()
 cmd:=exec.CommandContext(ctx,bin,"-v","error","-show_entries","format=duration,size,format_name:stream=codec_name,codec_type,width,height,r_frame_rate","-of","json","-i",path)
 out,e:=cmd.Output();if e!=nil{return nil,e}
 if len(out)>16000{return nil,errors.New("metadados acima do limite")}
 var data any
 if e=json.Unmarshal(out,&data);e!=nil{return nil,e}
 return map[string]any{"file":relative,"metadata":data,"next_step":"Comparar resolucao/FPS/duracao e inspecionar frames para avaliar qualidade; ffprobe nao julga estetica"},nil
}
