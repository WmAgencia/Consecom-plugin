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
 "time"
)

// Identifiers are project REFERENCES only; this config intentionally does
// not hold OAuth tokens, private keys, provider credentials or .env contents.
type DevServiceBinding struct {
 RootID string `json:"root_id"`
 GitHubRepository string `json:"github_repository,omitempty"`
 VercelProjectID string `json:"vercel_project_id,omitempty"`
 VercelTeamID string `json:"vercel_team_id,omitempty"`
 SupabaseProjectRef string `json:"supabase_project_ref,omitempty"`
 RailwayProjectID string `json:"railway_project_id,omitempty"`
 Environment string `json:"environment"`
}
var bindingRepoPattern=regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}/[A-Za-z0-9_.-]{1,100}$`)
var bindingIDPattern=regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

func validateServiceBinding(b DevServiceBinding)error{
 if b.RootID==""{return errors.New("root_id obrigatorio")}
 if !bindingRepoPattern.MatchString(b.GitHubRepository)||strings.Contains(b.GitHubRepository,".."){
  return errors.New("repositorio GitHub deve seguir owner/repo")
 }
 for provider,id:=range map[string]string{
  "Vercel project":b.VercelProjectID,"Vercel team":b.VercelTeamID,
  "Supabase project":b.SupabaseProjectRef,"Railway project":b.RailwayProjectID,
 }{
  if id!=""&&!bindingIDPattern.MatchString(id){return fmt.Errorf("identificador %s invalido",provider)}
 }
 if b.Environment!="preview"&&b.Environment!="production"{return errors.New("ambiente permitido: preview ou production")}
 return nil
}

func (a *App) serviceBindingGet(rootID string)(any,error){
 if _,err:=a.getRoot(rootID);err!=nil{return nil,err}
 a.mu.Lock();defer a.mu.Unlock()
 for _,b:=range a.cfg.DevBindings{if b.RootID==rootID{
  return map[string]any{"binding":b,"provider_auth":"ChatGPT connectors only",
   "verification":"IDs declarados pelo usuario; validar com cada conector remoto antes do deploy"},nil
 }}
 return map[string]any{"root_id":rootID,"configured":false,
  "note":"Associe IDs explicitos dos projetos GitHub, Vercel, Supabase e Railway. Nenhum token sera copiado."},nil
}

func (a *App) serviceBindingSave(b DevServiceBinding,approved bool)(any,error){
 if err:=validateServiceBinding(b);err!=nil{return nil,err}
 root,err:=a.getRoot(b.RootID);if err!=nil{return nil,err}
 if isBroadOrDangerousRoot(root.Path){return nil,errors.New("projeto fora de escopo")}
 if !approved&&a.mode()!="maximum"{
  body,_:=json.Marshal(b)
  id,err:=a.queue(&Pending{Kind:"devbinding",Path:root.Path,Content:string(body),
   Description:"Associar projetos externos ao workspace "+root.Name+" (somente IDs; nao publica nada)"})
  if err!=nil{return nil,err}
  return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
 }
 a.mu.Lock();defer a.mu.Unlock()
 old:=append([]DevServiceBinding(nil),a.cfg.DevBindings...)
 for i,item:=range a.cfg.DevBindings{if item.RootID==b.RootID{
  a.cfg.DevBindings[i]=b
  if err=a.saveLocked();err!=nil{a.cfg.DevBindings=old;return nil,err}
  return map[string]any{"status":"associado","binding":b,"remote_modified":false},nil
 }}
 a.cfg.DevBindings=append(a.cfg.DevBindings,b)
 if err=a.saveLocked();err!=nil{a.cfg.DevBindings=old;return nil,err}
 return map[string]any{"status":"associado","binding":b,"remote_modified":false},nil
}

func (a *App) serviceBindingApproved(p *Pending)(string,error){
 var b DevServiceBinding
 if err:=json.Unmarshal([]byte(p.Content),&b);err!=nil{return "",err}
 root,err:=a.getRoot(b.RootID);if err!=nil{return "",err}
 if filepath.Clean(root.Path)!=filepath.Clean(p.Path){return "",errors.New("pasta foi alterada")}
 result,err:=a.serviceBindingSave(b,true);if err!=nil{return "",err}
 return fmt.Sprintf("Associacao local registrada: %v",result),nil
}

func gitWorktreeClean(path string)(bool,error){
 ctx,cancel:=context.WithTimeout(context.Background(),7*time.Second);defer cancel()
 // --porcelain emits paths that might themselves be secret names; return
 // only cleanliness and never surface the output.
 cmd:=exec.CommandContext(ctx,"git","status","--porcelain","--untracked-files=normal")
 cmd.Dir=path
 cmd.Env=append(filteredDeveloperEnvironment(),"GIT_TERMINAL_PROMPT=0","GIT_OPTIONAL_LOCKS=0")
 bytes,err:=cmd.Output()
 if err!=nil{return false,errors.New("nao foi possivel consultar git status no projeto")}
 if len(bytes)>400000{return false,errors.New("status Git grande demais; revise alteracoes locais")}
 return len(bytes)==0,nil
}

func (a *App) deployPreflight(rootID string)(any,error){
 root,err:=a.getRoot(rootID);if err!=nil{return nil,err}
 if err=ensureNoLinks(root.Path);err!=nil{return nil,err}
 info,err:=a.serviceBindingGet(rootID);if err!=nil{return nil,err}
 obj:=info.(map[string]any)
 b,ok:=obj["binding"].(DevServiceBinding);if !ok{return map[string]any{
  "ready":false,"root_id":rootID,"reason":"projetos remotos nao associados ao workspace",
  "remote_modified":false},nil}
 if !projectFileExists(root.Path,".git"){return map[string]any{
  "ready":false,"root_id":rootID,"reason":"projeto local nao possui repositorio Git",
  "binding":b,"remote_modified":false},nil}
 clean,err:=gitWorktreeClean(root.Path)
 if err!=nil{return nil,err}
 blockers:=[]string{}
 if !clean{blockers=append(blockers,"Repositorio com alteracoes locais; revise e commite antes de publicar")}
 if b.VercelProjectID==""&&b.RailwayProjectID==""{
  blockers=append(blockers,"Nenhum destino Vercel/Railway indicado")}
 if b.Environment=="production"{
  blockers=append(blockers,"Destino production requer revisao humana e verificacao de escopo no conector antes do deploy")}
 return map[string]any{
  "ready_for_provider_verification":len(blockers)==0,
  "ready_for_unattended_production_deploy":false,
  "root_id":rootID,"git_clean":clean,"binding":b,"blockers":blockers,
  "next":"Conferir IDs e acesso via conectores oficiais GitHub/Vercel/Supabase/Railway no ChatGPT; depois executar deploy autorizado nesses conectores.",
  "remote_modified":false,"secrets_read":false,
 },nil
}

var _=os.ErrNotExist
