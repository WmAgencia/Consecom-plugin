package main

import (
 "context"
 "errors"
 "fmt"
 "io"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "sync"
 "time"
)

type DevJob struct {
 ID string `json:"id"`
 RootID string `json:"root_id"`
 Action string `json:"action"`
 Status string `json:"status"`
 Output string `json:"output"`
 Error string `json:"error,omitempty"`
 Started time.Time `json:"started"`
 Finished time.Time `json:"finished,omitempty"`
 cancel context.CancelFunc
}
type DevJobManager struct {
 mu sync.Mutex
 jobs map[string]*DevJob
}
var longDevActions = map[string][]string{
 "git_status": {"git","status","--short","--branch"},
 "git_diff": {"git","diff","--no-ext-diff","--stat"},
 "git_log": {"git","log","-12","--oneline"},
 "go_test": {"go","test","./..."},
 "go_build": {"go","build","./..."},
 "npm_test": {"npm","run","test"},
 "npm_build": {"npm","run","build"},
 "npm_lint": {"npm","run","lint"},
}
func jobNeedsApproval(act string) bool {
 return strings.HasPrefix(act,"npm_") || act=="go_build" || act=="go_test"
}
func (a *App) devJobStart(rootID, action string, approved bool) (any,error) {
 argv,ok:=longDevActions[action]
 if !ok {return nil,errors.New("comando fora da lista permitida")}
 root,err:=a.getRoot(rootID)
 if err!=nil{return nil,err}
 if isBroadOrDangerousRoot(root.Path) {return nil,errors.New("projeto fora do escopo")}
 if err=ensureNoLinks(root.Path);err!=nil{return nil,err}
 if jobNeedsApproval(action) && !approved && a.mode()!="maximum" {
  id,e:=a.queue(&Pending{Kind:"devjob",Path:root.Path,Action:action,Description:"Terminal: executar "+action+" no projeto "+root.Name+". Pode executar scripts do projeto; verifique antes de aprovar."})
  if e!=nil{return nil,e}
  return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
 }
 a.devJobs.mu.Lock()
 if a.devJobs.jobs==nil {a.devJobs.jobs=map[string]*DevJob{}}
 running:=0
 for _,j:=range a.devJobs.jobs {if j.Status=="running" {running++}}
 if running>=2 {a.devJobs.mu.Unlock();return nil,errors.New("limite de 2 comandos simultaneos")}
 if len(a.devJobs.jobs)>=35 {
  var oldest string;var old time.Time
  for id,j:=range a.devJobs.jobs {if j.Status=="running" {continue};if oldest==""||j.Started.Before(old) {oldest=id;old=j.Started}}
  if oldest!="" {delete(a.devJobs.jobs,oldest)}
 }
 ctx,cancel:=context.WithTimeout(context.Background(),120*time.Second)
 j:=&DevJob{ID:newID(),RootID:rootID,Action:action,Status:"running",Started:time.Now(),cancel:cancel}
 a.devJobs.jobs[j.ID]=j
 a.devJobs.mu.Unlock()
 go func(){
  defer cancel()
  var cmd *exec.Cmd
  if runtime.GOOS=="windows"&&argv[0]=="npm" {
   cmd=exec.CommandContext(ctx,"cmd.exe","/D","/S","/C","npm run "+argv[2])
  }else{cmd=exec.CommandContext(ctx,argv[0],argv[1:]...)}
  cmd.Dir=root.Path
  cmd.Env=append(filteredDeveloperEnvironment(),"GIT_TERMINAL_PROMPT=0","GIT_PAGER=cat","NO_COLOR=1")
  sink:=&jobWriter{manager:&a.devJobs,id:j.ID}
  cmd.Stdout=sink;cmd.Stderr=sink
  e:=cmd.Run()
  a.devJobs.mu.Lock()
  j.Finished=time.Now()
  switch {case ctx.Err()==context.DeadlineExceeded:j.Status="timeout";j.Error="limite de 120 segundos"
   case ctx.Err()==context.Canceled:j.Status="cancelled"
   case e!=nil:j.Status="error";j.Error=e.Error()
   default:j.Status="done"}
  j.cancel=nil
  a.devJobs.mu.Unlock()
 }()
 return map[string]any{"id":j.ID,"status":"running","action":action,"root_id":rootID},nil
}
type jobWriter struct {manager *DevJobManager;id string}
func (w *jobWriter) Write(b []byte)(int,error) {
 w.manager.mu.Lock();defer w.manager.mu.Unlock()
 j:=w.manager.jobs[w.id]
 if j!=nil {
  // Keep only a bounded tail; output may include escape sequences.
  text:=strings.Map(func(r rune)rune{if r==0{return -1};return r},string(b))
  j.Output+=text
  if len(j.Output)>48000 {j.Output=j.Output[len(j.Output)-48000:]}
 }
 return len(b),nil
}
func (a *App) devJobStatus(id string)(any,error){
 a.devJobs.mu.Lock();defer a.devJobs.mu.Unlock()
 j:=a.devJobs.jobs[id]
 if j==nil{return nil,errors.New("execucao nao encontrada")}
 copy:=*j;copy.cancel=nil
 return copy,nil
}
func (a *App) devJobCancel(id string)(any,error){
 a.devJobs.mu.Lock();defer a.devJobs.mu.Unlock()
 j:=a.devJobs.jobs[id]
 if j==nil {return nil,errors.New("execucao nao encontrada")}
 if j.Status!="running"||j.cancel==nil {return nil,errors.New("execucao ja encerrada")}
 j.cancel()
 return map[string]any{"id":id,"status":"cancel_requested"},nil
}
func (a *App) approveDevJob(p *Pending)(string,error){
 for _,r:=range a.roots(){
  if filepath.Clean(r.Path)==filepath.Clean(p.Path){
   x,e:=a.devJobStart(r.ID,p.Action,true)
   if e!=nil{return "",e}
   return fmt.Sprintf("Execucao autorizada: %v",x),nil
  }
 }
 return "",errors.New("projeto nao esta mais autorizado")
}
var _ io.Writer = (*jobWriter)(nil)
var _ = os.ErrNotExist
