package main

import (
 "encoding/json"
 "errors"
 "os"
 "path/filepath"
 "sort"
 "time"
)

type WorkflowHistoryEntry struct {
 ID string `json:"id"`
 RootID string `json:"root_id"`
 Steps []string `json:"steps"`
 Current int `json:"current"`
 Status string `json:"status"`
 Started time.Time `json:"started"`
 Finished time.Time `json:"finished,omitempty"`
}

func workflowHistoryDir(a *App) string {return filepath.Join(filepath.Dir(a.path),"workflow-history")}

func (a *App) saveWorkflowSnapshot(j WorkflowTask) error {
 if !editorBackupID.MatchString(j.ID){return errors.New("id do fluxo invalido")}
 dir:=workflowHistoryDir(a)
 if err:=os.MkdirAll(dir,0700);err!=nil{return err}
 j.cancel=nil
 data,err:=json.MarshalIndent(j,"","  ");if err!=nil{return err}
 if len(data)>48000{return errors.New("registro de fluxo acima de 48 KiB")}
 f,err:=os.CreateTemp(dir,".workflow-*");if err!=nil{return err}
 temp:=f.Name()
 defer os.Remove(temp)
 if err=f.Chmod(0600);err!=nil{f.Close();return err}
 _,err=f.Write(data)
 if err==nil{err=f.Sync()}
 closeErr:=f.Close()
 if err==nil{err=closeErr}
 if err!=nil{return err}
 if err=replaceProjectFile(temp,filepath.Join(dir,j.ID+".json"));err!=nil{return err}
 // Cap disk history to 40 snapshots, keeping the most recently modified.
 files,err:=os.ReadDir(dir);if err!=nil{return nil}
 type item struct{name string;mod time.Time}
 records:=make([]item,0,len(files))
 for _,entry:=range files {
  if !editorBackupID.MatchString(trimWorkflowSuffix(entry.Name())) {continue}
  if !entry.Type().IsRegular(){continue}
  info,e:=entry.Info();if e==nil{records=append(records,item{entry.Name(),info.ModTime()})}
 }
 if len(records)>40{
  sort.Slice(records,func(i,k int)bool{return records[i].mod.After(records[k].mod)})
  for _,item:=range records[40:]{_ = os.Remove(filepath.Join(dir,item.name))}
 }
 return nil
}

func trimWorkflowSuffix(name string)string{
 if filepath.Ext(name)!=".json"{return ""}
 return name[:len(name)-5]
}

func (a *App) loadWorkflowSnapshot(id string)(WorkflowTask,error){
 var job WorkflowTask
 if !editorBackupID.MatchString(id){return job,errors.New("id do fluxo invalido")}
 path:=filepath.Join(workflowHistoryDir(a),id+".json")
 info,err:=os.Lstat(path);if err!=nil{return job,err}
 if !info.Mode().IsRegular()||info.Size()>48000{return job,errors.New("snapshot invalido")}
 data,err:=os.ReadFile(path);if err!=nil{return job,err}
 if err=json.Unmarshal(data,&job);err!=nil{return job,err}
 if job.ID!=id||len(job.Steps)>6||job.Current>6 {return job,errors.New("snapshot inconsistente")}
 job.cancel=nil
 // A process restart cannot resume a previously running job automatically.
 if job.Status=="running"{job.Status="interrupted";job.Logs=append(job.Logs,"Processo Consecom reiniciado; reexecute o fluxo manualmente apos revisar logs")}
 return job,nil
}

func (a *App) workflowListHistory()(any,error){
 a.workflows.mu.Lock()
 live:=map[string]WorkflowTask{}
 for id,j:=range a.workflows.jobs {copy:=*j;copy.cancel=nil;live[id]=copy}
 a.workflows.mu.Unlock()
 entries:=[]WorkflowHistoryEntry{}
 for _,job:=range live{entries=append(entries,workflowEntry(job))}
 dir:=workflowHistoryDir(a)
 files,err:=os.ReadDir(dir)
 if err!=nil&&!os.IsNotExist(err){return nil,err}
 for _,f:=range files{
  id:=trimWorkflowSuffix(f.Name())
  if !editorBackupID.MatchString(id){continue}
  if _,exists:=live[id];exists{continue}
  job,e:=a.loadWorkflowSnapshot(id)
  if e==nil {entries=append(entries,workflowEntry(job))}
 }
 sort.Slice(entries,func(i,j int)bool{return entries[i].Started.After(entries[j].Started)})
 if len(entries)>20{entries=entries[:20]}
 return map[string]any{"jobs":entries,"count":len(entries),"note":"Historico local; execucoes interrompidas pelo reinicio nao sao retomadas automaticamente"},nil
}

func workflowEntry(j WorkflowTask)WorkflowHistoryEntry{
 return WorkflowHistoryEntry{ID:j.ID,RootID:j.RootID,Steps:append([]string{},j.Steps...),Current:j.Current,Status:j.Status,Started:j.Started,Finished:j.Finished}
}

func (a *App) workflowCheckpoint(job *WorkflowTask){
 a.workflows.mu.Lock()
 cp:=*job
 cp.Steps=append([]string{},job.Steps...)
 cp.Logs=append([]string{},job.Logs...)
 cp.cancel=nil
 a.workflows.mu.Unlock()
 if err:=a.saveWorkflowSnapshot(cp);err!=nil{audit("workflow checkpoint %s: %v",cp.ID,err)}
}

