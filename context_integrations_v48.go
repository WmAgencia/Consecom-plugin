package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "time"
)

// IntegrationStatus separates program detection, configuration and a real MCP
// handshake. A user-selected entry is never displayed as a verified client.
type IntegrationStatus struct {
 ID string `json:"id"`
 Label string `json:"label"`
 Detected bool `json:"detected"`
 Supported bool `json:"supported"`
 Configured bool `json:"configured"`
 Observed bool `json:"observed"`
 WorkspaceHint string `json:"workspace_hint,omitempty"`
 Ready bool `json:"ready"`
 Detail string `json:"detail"`
}

var integrationNames=[]struct{ID,Label,Executable string}{
 {"chatgpt","ChatGPT Chat",""},
 {"obsidian","Obsidian",""},
 {"claude-code","Claude Code","claude"},
 {"opencode","OpenCode","opencode"},
 {"codex","Codex CLI","codex"},
 {"cursor","Cursor",""},
}

func contextExecutable(command string)(string,bool){
 cli,ok:=contextCLI(command);return cli.Binary,ok
}
func (a *App) contextIntegrations()(any,error){
 vaults,err:=obsidianVaults();if err!=nil{return nil,err}
 clues:=contextWorkspaceClues()
 var apps []InstalledWindowsApp
 if runtime.GOOS=="windows" {
  if v,e:=windowsAppsSearch("");e==nil{
   if record,ok:=v.(map[string]any);ok{apps,_=record["programas"].([]InstalledWindowsApp)}
  }
 }
 a.mu.Lock()
 selected:=a.cfg.ContextVault
 configured:=make(map[string]bool,len(a.cfg.ContextConnections))
 for k,v:=range a.cfg.ContextConnections{configured[k]=v}
 sources:=append([]ContextSource(nil),a.cfg.ContextSources...)
 lastMCP:=a.lastMCP
 a.mu.Unlock()
 tunnel:=tunnelStatus(a)
 tunnelReady:=tunnel["status"]=="pronto"
 mcpRecent:=!lastMCP.IsZero()&&time.Since(lastMCP)<3*time.Minute
 out:=make([]IntegrationStatus,0,len(integrationNames))
 for _,item:=range integrationNames{
  _,cli:=contextExecutable(item.Executable)
  desktop:=false
  for _,app:=range apps{
   n:=strings.ToLower(app.Name+" "+app.AppID)
   switch item.ID{
   case "obsidian": desktop=desktop||strings.Contains(n,"obsidian")
   case "claude-code":desktop=desktop||strings.Contains(n,"claude code")
   case "codex":desktop=desktop||strings.Contains(n,"codex")
   case "opencode":desktop=desktop||strings.Contains(n,"opencode")||strings.Contains(n,"open code")
   case "cursor":desktop=desktop||strings.Contains(n,"cursor")
   case "chatgpt":desktop=desktop||strings.Contains(n,"chatgpt")
   }
  }
  hint:=clues[item.ID]
  detected:=cli||desktop||hint!=""
  supported:=cli&&(item.ID=="claude-code"||item.ID=="codex")
  if item.ID=="opencode"{supported=detected}
  if item.ID=="obsidian"{detected=detected||len(vaults)>0;supported=len(vaults)>0}
  cfg:=configured[item.ID]
  if item.ID=="obsidian"{cfg=selected!=""}
  observed:=false
  for _,source:=range sources{
   if strings.HasPrefix(source.Origin,"mcp-clientInfo")&&contextOrigin(source.Label)==contextOrigin(item.Label){observed=true;break}
  }
  ready:=false
  detail:="Aplicativo nao detectado"
  if detected { detail="Aplicativo detectado; pronto para configurar" }
  if hint!=""&&!cli{detail="Pasta de projeto identificada; CLI ainda nao localizada"}
  if cfg{detail="Configuracao MCP salva; reinicie o cliente para validar"}
  if observed{detail="Cliente MCP observado anteriormente; identidade nao autenticada"}
  if item.ID=="obsidian"{
   if len(vaults)>0{detail="Cofre detectado; clique para conectar automaticamente"}
   if cfg{detail="Cofre sincronizado com Consecom/Contexto";ready=true}
  }
  if item.ID=="opencode"&&cfg{detail="Configuracao local MCP criada; reinicie o OpenCode"}
  if item.ID=="chatgpt"{
   detected=tunnel["installed"]==true||tunnel["configured"]==true||tunnelReady
   cfg=tunnelReady
   observed=mcpRecent
   supported=false
   ready=tunnelReady
   switch {
   case mcpRecent: detail="ChatGPT utilizou o MCP recentemente; tunel pronto"
   case tunnelReady: detail="Tunel pronto; aguardando a primeira chamada do ChatGPT"
   case tunnel["configured"]==true:detail="Tunel configurado; aguardando conexao"
   default: detail="Conecte o tunel na Central de Controle"
   }
  }
  out=append(out,IntegrationStatus{ID:item.ID,Label:item.Label,Detected:detected,Supported:supported,Configured:cfg,Observed:observed,Ready:ready,WorkspaceHint:hint,Detail:detail})
 }
 return map[string]any{"integrations":out,"search":"PATH, locais oficiais, menu Iniciar e nomes de pastas em C:/D: com busca limitada"},nil
}
func (a *App) contextMarkConfigured(id string)error{
 a.mu.Lock();defer a.mu.Unlock()
 if a.cfg.ContextConnections==nil{a.cfg.ContextConnections=map[string]bool{}}
 before:=a.cfg.ContextConnections[id]
 a.cfg.ContextConnections[id]=true
 if err:=a.saveLocked();err!=nil{if !before{delete(a.cfg.ContextConnections,id)};return err}
 return nil
}
func contextRunMCPWithCLI(cli contextCLIInfo,args ...string)error{
 ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second)
 defer cancel()
 argv:=append(append([]string{},cli.Prefix...),args...)
 cmd:=exec.CommandContext(ctx,cli.Binary,argv...)
 cmd.Env=os.Environ()
 output,err:=cmd.CombinedOutput()
 if err!=nil{
  if ctx.Err()!=nil{return errors.New("CLI expirou em 20 segundos; nenhuma confirmacao de conexao foi registrada")}
  return fmt.Errorf("CLI nao configurou o MCP: %s",clip(strings.TrimSpace(string(output)),320))
 }
 return nil
}
func (a *App) contextConnectIntegration(id string)(any,error){
 switch id {
 case "obsidian":
  return a.contextVaultAuto()
 case "claude-code","codex","opencode":
  var command string
  switch id{case "claude-code":command="claude";case "codex":command="codex";case "opencode":command="opencode"}
  binary,err:=os.Executable();if err!=nil{return nil,err}
  binary,err=filepath.Abs(binary);if err!=nil{return nil,err}
  if runtime.GOOS=="windows" && !strings.EqualFold(filepath.Base(binary),"MCP-PC.exe"){
   return nil,errors.New("use a instalacao oficial MCP-PC.exe para configurar ferramentas locais")
  }
  if id=="opencode"{
   home,err:=os.UserHomeDir();if err!=nil{return nil,err}
   target:=filepath.Join(home,".config","opencode","opencode.json")
   if err=contextConfigureOpenCode(target,binary);err!=nil{return nil,err}
  }else{
   cli,ok:=contextCLI(command)
   if !ok{return nil,errors.New("CLI nao encontrada nos locais padrao; pastas de projetos nao executam comandos MCP")}
   args:=[]string{"mcp","add"}
   if id=="claude-code"{args=append(args,"--scope","user")}
   args=append(args,"consecom","--",binary,"--stdio")
   if err:=contextRunMCPWithCLI(cli,args...);err!=nil{return nil,err}
  }
  if err:=a.contextMarkConfigured(id);err!=nil{return nil,err}
  label:=map[string]string{"claude-code":"Claude Code","codex":"Codex","opencode":"OpenCode"}[id]
  source,e:=a.contextRegisterSource(label,"setup-cli-unverified")
  if e!=nil{return nil,e}
  return map[string]any{"status":"configured","id":id,"verification":"Configurado. Valide o cliente no proximo handshake MCP.","source":source},nil
 default:return nil,errors.New("integracao nao oferece conexao automatica")
 }
}
func (a *App) contextIndexIntegration(id string)(any,error){
 names:=map[string]string{"claude-code":"Claude Code","opencode":"OpenCode","codex":"Codex","cursor":"Cursor"}
 label,ok:=names[id];if !ok{return nil,errors.New("tipo de fonte desconhecido")}
 clues:=contextWorkspaceClues()
 if clues[id]==""{return nil,errors.New("nenhum repositorio identificado para essa fonte")}
 return a.contextRegisterSource(label,"local-directory-unverified")
}
// OpenCode's global JSON format is merged instead of replacing other entries.
// Existing non-JSON or conflicting Consecom entries remain untouched.
func contextConfigureOpenCode(path,binary string)error{
 original,e:=os.ReadFile(path)
 if e!=nil&&!errors.Is(e,os.ErrNotExist){return e}
 if len(original)>1<<20{return errors.New("configuracao OpenCode maior que 1 MiB")}
 root:=map[string]any{}
 if len(strings.TrimSpace(string(original)))>0{
  if err:=json.Unmarshal(original,&root);err!=nil{return errors.New("configuracao OpenCode contem comentarios ou JSON invalido; nao alterada para proteger seus dados")}
 }
 mcp,ok:=root["mcp"].(map[string]any)
 if !ok{if root["mcp"]!=nil{return errors.New("campo mcp existente invalido")};mcp=map[string]any{}}
 if old,exists:=mcp["consecom"];exists{
  entry,ok:=old.(map[string]any)
  if !ok{return errors.New("entrada Consecom ja existente e diferente")}
  cmd,ok:=entry["command"].([]any)
  if !ok||len(cmd)!=2||cmd[0]!=binary||cmd[1]!="--stdio"{
   return errors.New("entrada Consecom ja existente; nao substituida")}
  return nil
 }
 mcp["consecom"]=map[string]any{"type":"local","command":[]string{binary,"--stdio"},"enabled":true}
 root["mcp"]=mcp
 output,err:=json.MarshalIndent(root,"","  ");if err!=nil{return err}
 if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil{return err}
 if len(original)>0{
  backup:=path+".consecom-backup"
  if err:=writeExclusive(backup,string(original));err!=nil{
   return fmt.Errorf("nao foi possivel salvar backup antes da configuracao: %w",err)
  }
 }
 tmp,err:=os.CreateTemp(filepath.Dir(path),".consecom-opencode-*.tmp")
 if err!=nil{return err}
 defer os.Remove(tmp.Name())
 if err=tmp.Chmod(0600);err!=nil{tmp.Close();return err}
 if _,err=tmp.Write(output);err!=nil{tmp.Close();return err}
 if err=tmp.Close();err!=nil{return err}
 if err=os.Rename(tmp.Name(),path);err!=nil{return err}
 return nil
}
