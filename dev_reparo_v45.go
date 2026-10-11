package main

import (
 "encoding/json"
 "errors"
 "os"
 "path/filepath"
)

// The assistant is the reasoning/code-generation agent. The local MCP only
// selects bounded allowlisted checks, executes them, and returns checkpoints.
// Never run an arbitrary LLM-generated terminal command through this route.
func repairStepsForRoot(root Root)([]string,error){
 if projectFileExists(root.Path,"go.mod"){return []string{"go_test","go_build"},nil}
 rel,err:=safePath(root,"package.json",false)
 if err!=nil{return nil,err}
 st,err:=os.Stat(rel);if err!=nil{return nil,errors.New("projeto sem go.mod ou package.json")}
 if !st.Mode().IsRegular()||st.Size()>262144{return nil,errors.New("package.json muito grande ou irregular")}
 bytes,err:=os.ReadFile(rel);if err!=nil{return nil,err}
 var pkg struct{Scripts map[string]json.RawMessage `json:"scripts"`}
 if err=json.Unmarshal(bytes,&pkg);err!=nil{return nil,errors.New("package.json invalido")}
 steps:=[]string{}
 for _,s:=range []struct{key,action string}{{"lint","npm_lint"},{"test","npm_test"},{"build","npm_build"}}{
  if len(pkg.Scripts[s.key])>0{steps=append(steps,s.action)}
 }
 if len(steps)==0{return nil,errors.New("nenhum script lint, test ou build encontrado no projeto")}
 return steps,nil
}

func (a *App) repairPlan(rootID string)(any,error){
 root,err:=a.getRoot(rootID);if err!=nil{return nil,err}
 if isBroadOrDangerousRoot(root.Path){return nil,errors.New("pasta protegida")}
 if err=ensureNoLinks(root.Path);err!=nil{return nil,err}
 steps,err:=repairStepsForRoot(root);if err!=nil{return nil,err}
 return map[string]any{
  "root_id":root.ID,"project":root.Name,"steps":steps,
  "source":"scripts reais detectados ou go.mod",
  "strategy":"Execute dev_reparo_iniciar; consulte fluxo_status; o ChatGPT analisa as falhas, prepara um diff, aplica via git_patch_aplicar com rollback e repete as validacoes.",
  "automatic_code_generation":false,
  "note":"O Consecom nao gera correcoes sozinho sem ChatGPT. Testes executam scripts do proprio projeto autorizado.",
 },nil
}

func (a *App) repairStart(rootID string)(any,error){
 root,err:=a.getRoot(rootID);if err!=nil{return nil,err}
 if isBroadOrDangerousRoot(root.Path){return nil,errors.New("pasta protegida")}
 if err=ensureNoLinks(root.Path);err!=nil{return nil,err}
 steps,err:=repairStepsForRoot(root);if err!=nil{return nil,err}
 return a.workflowRequest(root.ID,steps)
}

// A patch-quality gate executes only scripts declared in the existing project.
// This is invoked by the existing gitPatchApproved (with rollback on failure).
// max 3 checks, each bounded by the standard approved test timeout.
func runQualityGate(root Root)(string,error){
 steps,err:=repairStepsForRoot(root);if err!=nil{return "",err}
 if projectFileExists(root.Path,"go.mod"){
  // Go project: the approved patch validator accepts Go tests as before.
  return runApprovedPatchTest(root,"go_test")
 }
 result:=""
 for _,step:=range steps{
  part,err:=runApprovedPatchTest(root,step)
  result+=step+": "+clip(part,600)+`\n`
  if err!=nil{return clip(result,1800),err}
 }
 return clip(result,1800),nil
}

var _ = filepath.Clean
