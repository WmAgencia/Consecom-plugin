package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
 "strings"
 "sync"
 "time"
)

type RenderSpec struct{
 RootID string `json:"root_id"`
 Input string `json:"input"`
 Preset string `json:"preset"`
 Start int `json:"start"`
 Duration int `json:"duration"`
}
type RenderTask struct{
 ID string `json:"id"`
 Spec RenderSpec `json:"spec"`
 Status string `json:"status"`
 Output string `json:"output,omitempty"`
 OutputBytes int64 `json:"output_bytes,omitempty"`
 Error string `json:"error,omitempty"`
 Logs string `json:"logs,omitempty"`
 Started time.Time `json:"started"`
 Finished time.Time `json:"finished,omitempty"`
 cancel context.CancelFunc
}
type RenderManager struct{mu sync.Mutex;jobs map[string]*RenderTask}

func renderPreset(preset string)(string,error){
 switch preset{
 case "story":return "scale=720:1280:force_original_aspect_ratio=decrease,pad=720:1280:(ow-iw)/2:(oh-ih)/2",nil
 case "landscape":return "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2",nil
 case "square":return "scale=720:720:force_original_aspect_ratio=decrease,pad=720:720:(ow-iw)/2:(oh-ih)/2",nil
 default:return "",errors.New("preset invalido: story, landscape ou square")
 }
}
func (a *App) renderValidate(spec RenderSpec)(Root,string,error){
 if spec.Start<0||spec.Start>36000||spec.Duration<1||spec.Duration>180{return Root{},"",errors.New("intervalo de render invalido")}
 if _,err:=renderPreset(spec.Preset);err!=nil{return Root{},"",err}
 root,err:=a.getRoot(spec.RootID);if err!=nil{return Root{},"",err}
 if !root.Writable||isBroadOrDangerousRoot(root.Path){return Root{},"",errors.New("projeto sem escrita ou demasiado amplo")}
 path,err:=allowedVideoPath(root,spec.Input);if err!=nil{return Root{},"",err}
 if _,err=exec.LookPath("ffmpeg");err!=nil{return Root{},"",errors.New("FFmpeg nao disponivel")}
 return root,path,nil
}
func (a *App) videoRenderRequest(spec RenderSpec)(any,error){
 root,_,err:=a.renderValidate(spec);if err!=nil{return nil,err}
 if a.mode()=="maximum"{return a.renderStart(spec)}
 serialized,err:=json.Marshal(spec);if err!=nil{return nil,err}
 id,err:=a.queue(&Pending{Kind:"videorender",Path:root.Path,Content:string(serialized),
  Description:fmt.Sprintf("Exportar video %q preset %s, de %ds por %ds para arquivo novo",clip(spec.Input,80),spec.Preset,spec.Start,spec.Duration)})
 if err!=nil{return nil,err}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) videoRenderApproved(p *Pending)(string,error){
 var spec RenderSpec
 if err:=json.Unmarshal([]byte(p.Content),&spec);err!=nil{return "",err}
 root,_,err:=a.renderValidate(spec);if err!=nil{return "",err}
 if filepath.Clean(root.Path)!=filepath.Clean(p.Path){return "",errors.New("a autorizacao de projeto mudou")}
 result,err:=a.renderStart(spec);if err!=nil{return "",err}
 return fmt.Sprintf("Render iniciado: %v",result),nil
}
func (a *App) renderStart(spec RenderSpec)(any,error){
 _,_,err:=a.renderValidate(spec);if err!=nil{return nil,err}
 a.renders.mu.Lock()
 if a.renders.jobs==nil{a.renders.jobs=map[string]*RenderTask{}}
 for _,j:=range a.renders.jobs{if j.Status=="running"{a.renders.mu.Unlock();return nil,errors.New("ja existe um video sendo renderizado")}}
 if len(a.renders.jobs)>=20{
  var oldest string;var started time.Time
  for k,j:=range a.renders.jobs{if oldest==""||j.Started.Before(started){oldest=k;started=j.Started}}
  delete(a.renders.jobs,oldest)
 }
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Minute)
 j:=&RenderTask{ID:newID(),Spec:spec,Status:"running",Started:time.Now(),cancel:cancel}
 a.renders.jobs[j.ID]=j
 a.renders.mu.Unlock()
 go a.renderRun(ctx,j)
 return map[string]any{"id":j.ID,"status":"running","preset":spec.Preset,"note":"Exportacao MP4 assincrona; consulte video_render_status ou use video_render_cancelar"},nil
}

func (a *App) renderRun(ctx context.Context,task *RenderTask){
 defer func(){
  a.renders.mu.Lock()
  task.Finished=time.Now()
  task.cancel=nil
  a.renders.mu.Unlock()
 }()
 root,path,err:=a.renderValidate(task.Spec);if err!=nil{a.renderFail(task,err);return}
 dir:=filepath.Join(root.Path,"consecom-renders")
 if err=ensureNoLinks(dir);err!=nil{a.renderFail(task,err);return}
 if err=os.MkdirAll(dir,0700);err!=nil{a.renderFail(task,err);return}
 if err=ensureNoLinks(dir);err!=nil{a.renderFail(task,err);return}
 outputName:="consecom-"+task.ID+".mp4"
 output:=filepath.Join(dir,outputName)
 temp:=filepath.Join(dir,".consecom-"+task.ID+".temp.mp4")
 defer os.Remove(temp)
 if _,err=os.Lstat(output);err==nil{a.renderFail(task,errors.New("arquivo ja existe"));return}else if !os.IsNotExist(err){a.renderFail(task,err);return}
 vf,_:=renderPreset(task.Spec.Preset)
 bin,err:=exec.LookPath("ffmpeg");if err!=nil{a.renderFail(task,err);return}
 cmd:=exec.CommandContext(ctx,bin,
  "-nostdin","-hide_banner","-loglevel","error","-y",
  "-ss",strconv.Itoa(task.Spec.Start),
  "-protocol_whitelist","file,pipe","-i",path,
  "-t",strconv.Itoa(task.Spec.Duration),"-map","0:v:0","-map","0:a?",
  "-vf",vf,"-r","30",
  "-c:v","libx264","-preset","veryfast","-crf","20","-pix_fmt","yuv420p",
  "-c:a","aac","-b:a","160k","-threads","2","-movflags","+faststart",
  "-progress","pipe:1","-nostats",temp,
 )
 writer:=&renderLogWriter{app:a,id:task.ID}
 cmd.Stdout=writer;cmd.Stderr=writer
 err=cmd.Run()
 if ctx.Err()!=nil{a.renderFail(task,fmt.Errorf("render interrompido: %w",ctx.Err()));return}
 if err!=nil{a.renderFail(task,fmt.Errorf("FFmpeg: %w",err));return}
 stat,err:=os.Lstat(temp)
 if err!=nil||!stat.Mode().IsRegular()||stat.Size()<128||stat.Size()>1024*1024*1024{
  a.renderFail(task,errors.New("arquivo de saida ausente, irregular ou acima de 1 GiB"));return
 }
 if err=os.Rename(temp,output);err!=nil{a.renderFail(task,err);return}
 a.renders.mu.Lock()
 task.Output=filepath.Join("consecom-renders",outputName)
 task.OutputBytes=stat.Size()
 task.Status="done"
 a.renders.mu.Unlock()
 audit("video render concluido: %s (root=%s bytes=%d)",task.ID,task.Spec.RootID,stat.Size())
}
type renderLogWriter struct{app *App;id string}
func(w *renderLogWriter)Write(p []byte)(int,error){
 w.app.renders.mu.Lock()
 if job:=w.app.renders.jobs[w.id];job!=nil{
  clean:=strings.Map(func(r rune)rune{if r==0||r==27{return -1};return r},string(p))
  job.Logs+=clean
  if len(job.Logs)>6000{job.Logs=job.Logs[len(job.Logs)-6000:]}
 }
 w.app.renders.mu.Unlock()
 return len(p),nil
}
func (a *App) renderFail(task *RenderTask,err error){
 a.renders.mu.Lock()
 if errors.Is(err,context.Canceled){task.Status="cancelled"
 }else if errors.Is(err,context.DeadlineExceeded){task.Status="timeout"
 }else{task.Status="error"}
 task.Error=clip(err.Error(),500)
 a.renders.mu.Unlock()
}
func (a *App) videoRenderStatus(id string)(any,error){
 a.renders.mu.Lock();defer a.renders.mu.Unlock()
 j:=a.renders.jobs[id];if j==nil{return nil,errors.New("render nao encontrado")}
 result:=*j;result.cancel=nil
 return result,nil
}
func (a *App) videoRenderCancel(id string)(any,error){
 a.renders.mu.Lock();defer a.renders.mu.Unlock()
 j:=a.renders.jobs[id];if j==nil||j.Status!="running"||j.cancel==nil{return nil,errors.New("render nao esta em andamento")}
 j.cancel()
 return map[string]any{"id":id,"status":"cancel_requested"},nil
}
