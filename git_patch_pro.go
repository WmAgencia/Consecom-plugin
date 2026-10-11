package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "strings"
 "sync"
 "time"
)
var gitChangeMu sync.Mutex
var branchNameSafe=regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]{0,65}$`)
type patchPayload struct{
 Patch string `json:"patch"`
 Test string `json:"test"`
}
type patchRecord struct{
 ID string `json:"id"`
 Root string `json:"root"`
 Patch string `json:"patch"`
}
func validateProjectPatch(root Root, patch string)error{
 if !root.Writable{return errors.New("habilite escrita na pasta pelo painel")}
 if len(patch)<15||len(patch)>30000||strings.ContainsRune(patch,0)||!strings.HasPrefix(patch,"diff --git ") {
  return errors.New("patch git invalido ou acima de 30KiB")
 }
 if strings.Contains(patch,"GIT binary patch")||strings.Contains(patch,"Binary files ")||strings.Contains(patch,""){
  return errors.New("somente patches de texto UTF-8 com LF")
 }
 seen:=0
 for _,line:=range strings.Split(patch,"
"){
  var path string
  switch {
  case strings.HasPrefix(line,"diff --git a/"):
   fields:=strings.Fields(line)
   if len(fields)!=4 || !strings.HasPrefix(fields[2],"a/")||!strings.HasPrefix(fields[3],"b/")||fields[2][2:]!=fields[3][2:] {return errors.New("cabecalho de patch invalido")}
   path=fields[2][2:];seen++
  case strings.HasPrefix(line,"--- a/"):path=strings.TrimPrefix(line,"--- a/")
  case strings.HasPrefix(line,"+++ b/"):path=strings.TrimPrefix(line,"+++ b/")
  case strings.HasPrefix(line,"rename ")||strings.HasPrefix(line,"copy ")||strings.HasPrefix(line,"old mode")||strings.HasPrefix(line,"new mode")||strings.HasPrefix(line,"new file mode 120000"):
   return errors.New("renomeacoes e alteracoes de permissao nao permitidas")
  default: continue
  }
  if _,e:=safePath(root,path,true);e!=nil{return fmt.Errorf("arquivo %s: %w",path,e)}
 }
 if seen==0||seen>12{return errors.New("o patch deve conter de 1 a 12 arquivos")}
 return nil
}
func gitRunWithInput(root Root, input string, argv ...string)(string,error){
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel()
 cmd:=exec.CommandContext(ctx,"git",argv...)
 cmd.Dir=root.Path;cmd.Env=append(filteredDeveloperEnvironment(),"GIT_PAGER=cat","GIT_TERMINAL_PROMPT=0","NO_COLOR=1")
 cmd.Stdin=strings.NewReader(input)
 b,e:=cmd.CombinedOutput()
 if len(b)>24000 {b=b[:24000]}
 if ctx.Err()!=nil{return string(b),ctx.Err()}
 if e!=nil{return string(b),fmt.Errorf("%s: %w",strings.TrimSpace(string(b)),e)}
 return string(b),nil
}
func (a *App) gitPatchCheck(rootID,patch string)(any,error){
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 if e=validateProjectPatch(root,patch);e!=nil{return nil,e}
 out,e:=gitRunWithInput(root,patch,"apply","--check","--")
 if e!=nil{return nil,e}
 return map[string]any{"status":"valido","project":root.Name,"message":"Patch aplicavel sem alterar arquivos","output":out},nil
}
func (a *App) gitPatchRequest(rootID,patch,test string)(any,error){
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 if _,e=a.gitPatchCheck(rootID,patch);e!=nil{return nil,e}
 if test!=""&&test!="none"&&test!="go_test"&&test!="npm_test"&&test!="npm_lint" {return nil,errors.New("teste permitido: none,go_test,npm_test,npm_lint")}
 data,_:=json.Marshal(patchPayload{Patch:patch,Test:test})
 id,e:=a.queue(&Pending{Kind:"gitpatch",Path:root.Path,Content:string(data),Description:"Aplicar patch Git em "+root.Name+" e testar "+test+" (ate 12 arquivos; rollback automatico se falhar)"})
 if e!=nil{return nil,e}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) gitPatchApproved(p *Pending)(string,error){
 var payload patchPayload
 if e:=json.Unmarshal([]byte(p.Content),&payload);e!=nil{return "",e}
 var root Root;found:=false
 for _,r:=range a.roots(){if filepath.Clean(r.Path)==filepath.Clean(p.Path){root=r;found=true;break}}
 if !found{return "",errors.New("pasta de projeto nao autorizada")}
 if e:=validateProjectPatch(root,payload.Patch);e!=nil{return "",e}
 gitChangeMu.Lock();defer gitChangeMu.Unlock()
 if _,e:=gitRunWithInput(root,payload.Patch,"apply","--check","--");e!=nil{return "",e}
 if _,e:=gitRunWithInput(root,payload.Patch,"apply","--");e!=nil{return "",e}
 rollback:=func()error{_,e:=gitRunWithInput(root,payload.Patch,"apply","-R","--check","--");if e!=nil{return e};_,e=gitRunWithInput(root,payload.Patch,"apply","-R","--");return e}
 if payload.Test!=""&&payload.Test!="none"{
  argv:=longDevActions[payload.Test]
  _=argv
  var out string
  var e error
  // Executa a lista fixa de comandos previamente autorizados.
  out,e=runApprovedPatchTest(root,payload.Test)
  if e!=nil{
   if rb:=rollback();rb!=nil{return "",fmt.Errorf("teste falhou: %v; reversao automatica tambem falhou: %v",e,rb)}
   return "",fmt.Errorf("teste falhou e patch foi revertido: %v; %s",e,clip(out,500))
  }
 }
 id:=newID()
 rec:=patchRecord{ID:id,Root:root.Path,Patch:payload.Patch}
 dat,_:=json.Marshal(rec)
 folder:=filepath.Join(filepath.Dir(a.path),"git-patches")
 if err:=os.MkdirAll(folder,0700);err!=nil{return "",fmt.Errorf("patch aplicado; falha ao criar registro de rollback: %w",err)}
 if err:=os.WriteFile(filepath.Join(folder,id+".json"),dat,0600);err!=nil{return "",fmt.Errorf("patch aplicado; falha ao salvar rollback: %w",err)}
 return "Patch aplicado; backup reversivel ID "+id,nil
}
func runApprovedPatchTest(root Root,act string)(string,error){
 argv,ok:=longDevActions[act];if !ok||!jobNeedsApproval(act){return "",errors.New("teste nao autorizado")}
 ctx,cancel:=context.WithTimeout(context.Background(),95*time.Second);defer cancel()
 var cmd *exec.Cmd
 if strings.HasPrefix(act,"npm_") {cmd=exec.CommandContext(ctx,"cmd.exe","/D","/S","/C","npm run "+argv[2])}else{cmd=exec.CommandContext(ctx,argv[0],argv[1:]...)}
 cmd.Dir=root.Path;cmd.Env=filteredDeveloperEnvironment()
 b,e:=cmd.CombinedOutput();if len(b)>12000{b=b[:12000]}
 if ctx.Err()!=nil{return string(b),ctx.Err()}
 return string(b),e
}
func (a *App) gitPatchRollbackRequest(id string)(any,error){
 if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(id){return nil,errors.New("id de patch invalido")}
 rec,e:=a.readPatchRecord(id);if e!=nil{return nil,e}
 allowed:=false;for _,r:=range a.roots(){if filepath.Clean(r.Path)==filepath.Clean(rec.Root)&&r.Writable{allowed=true}}
 if !allowed{return nil,errors.New("pasta nao autorizada para reverter")}
 approval,e:=a.queue(&Pending{Kind:"gitrollback",Action:id,Path:rec.Root,Description:"Reverter apenas as alteracoes do patch Consecom "+id+" em "+rec.Root})
 if e!=nil{return nil,e};return map[string]any{"status":"aguardando_aprovacao_local","approval_id":approval},nil
}
func (a *App) readPatchRecord(id string)(patchRecord,error){
 var rec patchRecord
 if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(id){return rec,errors.New("id invalido")}
 b,e:=os.ReadFile(filepath.Join(filepath.Dir(a.path),"git-patches",id+".json"))
 if e!=nil{return rec,e}
 if e=json.Unmarshal(b,&rec);e!=nil{return rec,e}
 if rec.ID!=id{return rec,errors.New("registro inconsistente")}
 return rec,nil
}
func (a *App) gitPatchRollbackApproved(p *Pending)(string,error){
 gitChangeMu.Lock();defer gitChangeMu.Unlock()
 rec,e:=a.readPatchRecord(p.Action);if e!=nil{return "",e}
 if filepath.Clean(rec.Root)!=filepath.Clean(p.Path){return "",errors.New("origem mudou")}
 var root Root;found:=false;for _,r:=range a.roots(){if filepath.Clean(r.Path)==filepath.Clean(rec.Root)&&r.Writable{root=r;found=true;break}}
 if !found{return "",errors.New("pasta revogada")}
 if e:=validateProjectPatch(root,rec.Patch);e!=nil{return "",e}
 if _,e=gitRunWithInput(root,rec.Patch,"apply","-R","--check","--");e!=nil{return "",e}
 if _,e=gitRunWithInput(root,rec.Patch,"apply","-R","--");e!=nil{return "",e}
 _=os.Remove(filepath.Join(filepath.Dir(a.path),"git-patches",p.Action+".json"))
 return "Patch "+p.Action+" revertido",nil
}
func (a *App) gitBranchRequest(rootID,branch string)(any,error){
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 if !root.Writable{return nil,errors.New("escrita desativada no projeto")}
 if !branchNameSafe.MatchString(branch)||strings.Contains(branch,"..")||strings.HasSuffix(branch,".")||strings.HasSuffix(branch,".lock")||strings.Contains(branch,"//"){return nil,errors.New("nome de branch invalido")}
 id,e:=a.queue(&Pending{Kind:"gitbranch",Path:root.Path,Action:branch,Description:"Criar e trocar para branch Git "+branch+" no projeto "+root.Name})
 if e!=nil{return nil,e};return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) gitBranchApproved(p *Pending)(string,error){
 var root Root;ok:=false;for _,r:=range a.roots(){if filepath.Clean(r.Path)==filepath.Clean(p.Path)&&r.Writable{root=r;ok=true;break}}
 if !ok||!branchNameSafe.MatchString(p.Action){return "",errors.New("projeto ou branch invalida")}
 gitChangeMu.Lock();defer gitChangeMu.Unlock()
 out,e:=gitRunWithInput(root,"","switch","-c",p.Action)
 return out,e
}
