package main

import(
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "path/filepath"
 "strings"
 "sync"
 "time"
)
type WorkflowTask struct{
 ID string `json:"id"`
 RootID string `json:"root_id"`
 Steps []string `json:"steps"`
 Current int `json:"current"`
 Status string `json:"status"`
 Logs []string `json:"logs"`
 Started time.Time `json:"started"`
 Finished time.Time `json:"finished,omitempty"`
 cancel context.CancelFunc
}
type WorkflowManager struct{
 mu sync.Mutex
 jobs map[string]*WorkflowTask
}
type workflowSpec struct{
 RootID string `json:"root_id"`
 Steps []string `json:"steps"`
}
func validateWorkflow(steps []string)error {
 if len(steps)<1||len(steps)>6{return errors.New("fluxo deve conter de 1 a 6 etapas")}
 for _,step:=range steps{if _,ok:=longDevActions[step];!ok{return fmt.Errorf("etapa %q nao permitida",step)}}
 return nil
}
func (a *App) workflowRequest(rootID string,steps []string)(any,error){
 if e:=validateWorkflow(steps);e!=nil{return nil,e}
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 if isBroadOrDangerousRoot(root.Path){return nil,errors.New("pasta de projeto excessivamente ampla")}
 if e=ensureNoLinks(root.Path);e!=nil{return nil,e}
 copySteps:=append([]string{},steps...)
 if a.mode()=="maximum"{return a.workflowStart(rootID,copySteps)}
 payload,_:=json.Marshal(workflowSpec{RootID:rootID,Steps:copySteps})
 id,e:=a.queue(&Pending{Kind:"workflow",Path:root.Path,Content:string(payload),Description:"Executar fluxo de "+fmt.Sprint(len(steps))+" etapas em "+root.Name+": "+strings.Join(steps,", ")})
 if e!=nil{return nil,e}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) workflowApproved(p *Pending)(string,error){
 var spec workflowSpec
 if e:=json.Unmarshal([]byte(p.Content),&spec);e!=nil{return "",e}
 if e:=validateWorkflow(spec.Steps);e!=nil{return "",e}
 root,e:=a.getRoot(spec.RootID);if e!=nil{return "",e}
 if filepath.Clean(root.Path)!=filepath.Clean(p.Path){return "",errors.New("projeto alterado")}
 result,e:=a.workflowStart(spec.RootID,spec.Steps)
 if e!=nil{return "",e}
 return fmt.Sprintf("Fluxo iniciado: %v",result),nil
}
func (a *App) workflowStart(rootID string,steps []string)(any,error){
 if e:=validateWorkflow(steps);e!=nil{return nil,e}
 if _,e:=a.getRoot(rootID);e!=nil{return nil,e}
 a.workflows.mu.Lock()
 if a.workflows.jobs==nil{a.workflows.jobs=map[string]*WorkflowTask{}}
 for _,j:=range a.workflows.jobs{if j.Status=="running"{a.workflows.mu.Unlock();return nil,errors.New("ja existe um fluxo em execucao")}}
 if len(a.workflows.jobs)>=20{
  oldest:="";var stamp time.Time
  for k,v:=range a.workflows.jobs{if oldest==""||v.Started.Before(stamp){oldest=k;stamp=v.Started}}
  delete(a.workflows.jobs,oldest)
 }
 ctx,cancel:=context.WithTimeout(context.Background(),8*time.Minute)
 job:=&WorkflowTask{ID:newID(),RootID:rootID,Steps:append([]string{},steps...),Status:"running",Started:time.Now(),Logs:[]string{"Fluxo iniciado"},cancel:cancel}
 a.workflows.jobs[job.ID]=job
 a.workflows.mu.Unlock()
 go a.workflowRun(ctx,job)
 return map[string]any{"id":job.ID,"status":"running","steps":job.Steps,"note":"Cada etapa e validada; o fluxo interrompe ao primeiro erro"},nil
}
func (a *App) workflowRun(ctx context.Context,job *WorkflowTask){
 defer func(){
  a.workflows.mu.Lock()
  if job.Status=="running"{job.Status="done"}
  job.Finished=time.Now();job.cancel=nil
  a.workflows.mu.Unlock()
 }()
 for i,step:=range job.Steps{
  if ctx.Err()!=nil {a.workflowStop(job,"cancelled",ctx.Err().Error());return}
  a.workflows.mu.Lock();job.Current=i+1;job.Logs=append(job.Logs,fmt.Sprintf("%d/%d Iniciando %s",i+1,len(job.Steps),step));a.workflows.mu.Unlock()
  result,e:=a.devJobStart(job.RootID,step,true)
  if e!=nil{a.workflowStop(job,"error",e.Error());return}
  data,ok:=result.(map[string]any);if !ok{a.workflowStop(job,"error","execucao retornou formato invalido");return}
  id,_:=data["id"].(string)
  if id==""{a.workflowStop(job,"error","job sem identificador");return}
  ticker:=time.NewTicker(150*time.Millisecond)
  settled:=false
  for !settled{
   select {
   case <-ctx.Done():
    _,_=a.devJobCancel(id);ticker.Stop();a.workflowStop(job,"cancelled",ctx.Err().Error());return
   case <-ticker.C:
    info,err:=a.devJobStatus(id)
    if err!=nil{ticker.Stop();a.workflowStop(job,"error",err.Error());return}
    result:=info.(DevJob)
    if result.Status=="running"{continue}
    ticker.Stop();settled=true
    a.workflows.mu.Lock()
    job.Logs=append(job.Logs,fmt.Sprintf("%s: %s %s",step,result.Status,clip(result.Output,1200)))
    if len(job.Logs)>30{job.Logs=job.Logs[len(job.Logs)-30:]}
    a.workflows.mu.Unlock()
    if result.Status!="done"{a.workflowStop(job,"error",fmt.Sprintf("%s falhou: %s",step,result.Error));return}
   }
  }
 }
}
func (a *App) workflowStop(job *WorkflowTask,status,detail string){
 a.workflows.mu.Lock();defer a.workflows.mu.Unlock()
 job.Status=status;job.Logs=append(job.Logs,clip(detail,900))
}
func (a *App) workflowStatus(id string)(any,error){
 a.workflows.mu.Lock();defer a.workflows.mu.Unlock()
 j:=a.workflows.jobs[id];if j==nil{return nil,errors.New("fluxo nao encontrado")}
 copy:=*j;copy.Steps=append([]string{},j.Steps...);copy.Logs=append([]string{},j.Logs...);copy.cancel=nil
 return copy,nil
}
func (a *App) workflowCancel(id string)(any,error){
 a.workflows.mu.Lock();defer a.workflows.mu.Unlock()
 j:=a.workflows.jobs[id];if j==nil||j.Status!="running"||j.cancel==nil{return nil,errors.New("fluxo nao esta em andamento")}
 j.cancel()
 return map[string]any{"id":id,"status":"cancel_requested"},nil
}
