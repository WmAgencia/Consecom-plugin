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
 Detail string `json:"detail"`
}

var integrationNames=[]struct{ID,Label,Executable string}{
 {"obsidian","Obsidian",""},
 {"claude-code","Claude Code","claude"},
 {"codex","Codex CLI","codex"},
 {"opencode","OpenCode","opencode"},
 {"cursor","Cursor",""},
 {"chatgpt","ChatGPT Chat",""},
}

func contextExecutable(command string)(string,bool){
 if command==""{return "",false}
 p,e:=exec.LookPath(command)
 if e!=nil{return "",false}
 // Never send shell scripts, .cmd or .bat into a privileged installer.
 if runtime.GOOS=="windows" && !strings.EqualFold(filepath.Ext(p),".exe"){return "",false}
 return p,true
}

func (a *App) contextIntegrations()(any,error){
 vaults,err:=obsidianVaults();if err!=nil{return nil,err}
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
 a.mu.Unlock()
 out:=make([]IntegrationStatus,0,len(integrationNames))
 for _,item:=range integrationNames{
  _,cli:=contextExecutable(item.Executable)
  desktop:=false
  for _,app:=range apps{
   n:=strings.ToLower(app.Name+" "+app.AppID)
   switch item.ID{
   case "obsidian":desktop=desktop||strings.Contains(n,"obsidian")
   case "claude-code":desktop=desktop||strings.Contains(n,"claude code")
   case "codex":desktop=desktop||strings.Contains(n,"codex")
   case "opencode":desktop=desktop||strings.Contains(n,"opencode")||strings.Contains(n,"open code")
   case "cursor":desktop=desktop||strings.Contains(n,"cursor")
   case "chatgpt":desktop=desktop||strings.Contains(n,"chatgpt")||strings.Contains(n,"openai.codex")
   }
  }
  detected:=cli||desktop
  if item.ID=="obsidian"{detected=detected||len(vaults)>0}
  supported:=item.ID=="obsidian"||item.ID=="claude-code"&&cli||item.ID=="codex"&&cli||item.ID=="opencode"&&cli
  cfg:=configured[item.ID]
  if item.ID=="obsidian"{cfg=selected!=""}
  observed:=false
  for _,s:=range sources{
   if strings.HasPrefix(s.Origin,"mcp-clientInfo")&&contextOrigin(s.Label)==contextOrigin(item.Label){observed=true;break}
  }
  detail:="Nao detectado neste Windows"
  if detected{detail="Detectado; ainda nao configurado"}
  if detected&&!supported{detail="Aplicativo detectado; conector CLI nao encontrado ou nao disponivel"}
  if cfg{detail="Configurado; reinicie o cliente e confirme a conexao MCP"}
  if observed{detail="Cliente MCP observado; identidade nao autenticada"}
  if item.ID=="obsidian"&&cfg{detail="Cofre selecionado; sincronizacao Consecom/Contexto ativa"}
  if item.ID=="obsidian"&&len(vaults)>1&&!cfg{detail="Mais de um cofre; escolha qual conectar no seletor acima"}
  if item.ID=="chatgpt"{supported=false;if detected{detail="ChatGPT Chat se conecta pelo aplicativo MCP do ChatGPT, nao por instalacao local"}}
  out=append(out,IntegrationStatus{ID:item.ID,Label:item.Label,Detected:detected,Supported:supported,Configured:cfg,Observed:observed,Detail:detail})
 }
 return map[string]any{"integrations":out},nil
}
func (a *App) contextMarkConfigured(id string)error{
 a.mu.Lock();defer a.mu.Unlock()
 if a.cfg.ContextConnections==nil{a.cfg.ContextConnections=map[string]bool{}}
 before:=a.cfg.ContextConnections[id]
 a.cfg.ContextConnections[id]=true
 if err:=a.saveLocked();err!=nil{if !before{delete(a.cfg.ContextConnections,id)};return err}
 return nil
}
func contextRunMCP(executable string, args ...string)error{
 ctx,cancel:=context.WithTimeout(context.Background(),12*time.Second)
 defer cancel()
 cmd:=exec.CommandContext(ctx,executable,args...)
 // Calls the official CLIs with a fixed, bounded argv. No shell or token values.
 cmd.Env=os.Environ()
 output,err:=cmd.CombinedOutput()
 if err!=nil {
  if ctx.Err()!=nil{return errors.New("o cliente nao respondeu dentro de 12 segundos")}
  return fmt.Errorf("o cliente MCP recusou a configuracao: %s",clip(strings.TrimSpace(string(output)),240))
 }
 return nil
}
func (a *App) contextConnectIntegration(id string)(any,error){
 switch id{
 case "obsidian":
  vaults,err:=obsidianVaults();if err!=nil{return nil,err}
  if len(vaults)!=1{return nil,errors.New("selecione o cofre desejado na lista Obsidian (exige exatamente um para conexao automatica)")}
  return a.contextVaultSelect(vaults[0]["id"].(string))
 case "claude-code","codex","opencode":
  var command string
  switch id{case "claude-code":command="claude";case "codex":command="codex";case "opencode":command="opencode"}
  cli,ok:=contextExecutable(command)
  if !ok{return nil,errors.New("CLI compatível nao detectada. Instale a CLI oficial e tente novamente")}
  binary,err:=os.Executable();if err!=nil{return nil,err}
  binary,err=filepath.Abs(binary);if err!=nil{return nil,err}
  if runtime.GOOS=="windows" && !strings.EqualFold(filepath.Base(binary),"MCP-PC.exe"){
   return nil,errors.New("a conexao so funciona na instalacao Windows oficial MCP-PC.exe")
  }
  if id=="opencode"{
   home,err:=os.UserHomeDir();if err!=nil{return nil,err}
   target:=filepath.Join(home,".config","opencode","opencode.json")
   if err=contextConfigureOpenCode(target,binary);err!=nil{return nil,err}
  }else{
   // Fail instead of changing an existing client entry named consecom.
   if err:=contextRunMCP(cli,"mcp","get","consecom");err==nil{
    return nil,errors.New("ja existe uma entrada consecom nesta CLI; confira sua configuracao antes de substituir")
   }
   args:=[]string{"mcp","add","consecom"}
   if id=="claude-code"{args=append(args,"--scope","user")}
   args=append(args,"--",binary,"--stdio")
   if err:=contextRunMCP(cli,args...);err!=nil{return nil,err}
   if err:=contextRunMCP(cli,"mcp","get","consecom");err!=nil{
    return nil,errors.New("cliente nao confirmou a configuracao MCP; verifique manualmente")
   }
  }
  if err:=a.contextMarkConfigured(id);err!=nil{return nil,err}
  label:=map[string]string{"claude-code":"Claude Code","codex":"Codex","opencode":"OpenCode"}[id]
  source,e:=a.contextRegisterSource(label,"setup-cli-unverified")
  if e!=nil{return nil,e}
  return map[string]any{"status":"configured","id":id,"verification":"configuracao registrada, handshake MCP ainda nao observado","source":source},nil
 default:return nil,errors.New("integracao nao suporta conexao automatica nesta versao")
 }
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
