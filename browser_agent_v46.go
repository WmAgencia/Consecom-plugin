package main

import (
 "context"
 "encoding/base64"
 "errors"
 "fmt"
 "net"
 "net/url"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "runtime"
 "strconv"
 "strings"
 "sync"
 "time"
)

// Consecom optionally uses the official Vercel Labs agent-browser CLI.
// It never downloads software, adopts a personal Chrome profile, connects to
// someone else's CDP endpoint, evaluates page JS, or exports browser storage.
type AgentBrowserSession struct {
 mu sync.Mutex
 Name string
 Host string
 Allowed string
 LastSnapshot string
}

var agentBrowserRefRE=regexp.MustCompile(`^@e[1-9][0-9]{0,5}$`)
var agentBrowserUnsafeRE=regexp.MustCompile(`(?i)(password|senha|passphrase|credit.?card|cart[aã]o|token|api.?key|chave|payment|checkout|pagar|pay now|purchase|comprar|delete|deletar|excluir|remover|remove|cancel order|unsubscribe|transfer|transferir|publish|publicar|submit|enviar|confirmar|finalizar)`)
var agentBrowserDomainRE=regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]{0,251}[a-z0-9])?$`)

func agentBrowserURL(raw string)(string,string,error){
 if len(raw)>1500||strings.ContainsAny(raw,"
	") {return "","",errors.New("URL invalida ou muito longa")}
 u,e:=url.Parse(raw)
 if e!=nil||u==nil||u.User!=nil||u.RawQuery!=""||u.Fragment!=""||u.Opaque!=""{
  return "","",errors.New("URL invalida: nao inclua credenciais, query ou fragmento")}
 host:=strings.ToLower(u.Hostname())
 if !agentBrowserDomainRE.MatchString(host) || strings.Contains(host,".."){return "","",errors.New("dominio invalido")}
 if p:=u.Port();p!="" {n,err:=strconv.Atoi(p);if err!=nil||n<1||n>65535{return "","",errors.New("porta invalida")}}
 ip:=net.ParseIP(host)
 loopback:=host=="localhost" || (ip!=nil&&ip.IsLoopback())
 if u.Scheme!="https"&&!(u.Scheme=="http"&&loopback){return "","",errors.New("somente HTTPS publico ou HTTP localhost autorizado")}
 if ip!=nil && !loopback {return "","",errors.New("enderecos IP externos/internos diretos nao sao aceitos")}
 if host=="localhost" && u.Port()==""{return "","",errors.New("para localhost, informe a porta de desenvolvimento")}
 return u.String(),host,nil
}

func agentBrowserResolve() (string,[]string,error) {
 // Use the native binary if specifically installed, never execute a random
 // user-provided .cmd with untrusted URL/text arguments.
 if bin,err:=exec.LookPath("agent-browser.exe");err==nil{
  return bin,nil,nil
 }
 if runtime.GOOS!="windows"{
  if bin,err:=exec.LookPath("agent-browser");err==nil && !strings.HasSuffix(strings.ToLower(bin),".cmd"){return bin,nil,nil}
 }
 node,err:=exec.LookPath("node")
 if err!=nil{return "",nil,errors.New("Node.js nao localizado; instale Node.js e agent-browser")}
 npm,err:=exec.LookPath("npm")
 if err!=nil{return "",nil,errors.New("npm nao localizado; instale Node.js e agent-browser")}
 ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel()
 cmd:=exec.CommandContext(ctx,npm,"root","-g")
 cmd.Env=filteredDeveloperEnvironment()
 out,err:=cmd.Output()
 if err!=nil||len(out)>4096{return "",nil,errors.New("nao foi possivel identificar diretorio global npm")}
 root:=strings.TrimSpace(string(out))
 if root==""||!filepath.IsAbs(root){return "",nil,errors.New("npm root -g retornou caminho invalido")}
 launcher:=filepath.Join(root,"agent-browser","bin","agent-browser.js")
 st,err:=os.Stat(launcher)
 if err!=nil||!st.Mode().IsRegular(){return "",nil,errors.New("agent-browser nao instalado. Execute: npm install -g agent-browser; agent-browser install")}
 return node,[]string{launcher},nil
}

func (a *App) agentBrowserCommand(args []string,timeout time.Duration)(string,error){
 bin,prefix,err:=agentBrowserResolve()
 if err!=nil{return "",err}
 argv:=append(append([]string{},prefix...),args...)
 ctx,cancel:=context.WithTimeout(context.Background(),timeout);defer cancel()
 cmd:=exec.CommandContext(ctx,bin,argv...)
 cmd.Dir=filepath.Dir(a.path)
 cmd.Env=append(filteredDeveloperEnvironment(),
  "AGENT_BROWSER_CONTENT_BOUNDARIES=1",
  "AGENT_BROWSER_MAX_OUTPUT=14000",
  "AGENT_BROWSER_NO_AUTO_DIALOG=1",
  "AGENT_BROWSER_PIN_TAB=1",
  "GIT_TERMINAL_PROMPT=0",
 )
 out,err:=cmd.CombinedOutput()
 if len(out)>19000{out=out[:19000]}
 if ctx.Err()!=nil{return "",errors.New("agent-browser excedeu o limite de tempo")}
 if err!=nil{return "",fmt.Errorf("agent-browser falhou: %s",clip(strings.TrimSpace(string(out)),600))}
 return strings.TrimSpace(string(out)),nil
}

func (a *App) agentBrowserExecuteLocked(name string,sub ...string)(string,error){
 s:=&a.agentBrowser
 if s.Name==""{return "",errors.New("inicie primeiro uma sessao com navegador_agente_abrir")}
 args:=[]string{"--session",s.Name,"--allowed-domains",s.Allowed,"--content-boundaries","--max-output","14000"}
 args=append(args,sub...)
 timeout:=20*time.Second
 if name=="open"{timeout=55*time.Second}
 return a.agentBrowserCommand(args,timeout)
}

func (a *App) agentBrowserStatus()(any,error){
 path,prefix,err:=agentBrowserResolve()
 a.agentBrowser.mu.Lock();defer a.agentBrowser.mu.Unlock()
 return map[string]any{
  "installed":err==nil,
  "cli_entry":func()string{if err==nil&&len(prefix)>0{return "node/agent-browser.js"};if err==nil{return filepath.Base(path)};return "not-found"}(),
  "ready_for_session":err==nil,
  "session_active":a.agentBrowser.Name!="",
  "host":a.agentBrowser.Host,
  "installation":"npm install -g agent-browser; agent-browser install",
  "source":"https://github.com/vercel-labs/agent-browser",
  "note":"CLI opcional. O Consecom usa Chromium isolado e nao abre perfil pessoal, CDP externo ou arquivos de cookies.",
 },nil
}

func (a *App) agentBrowserOpen(raw string)(any,error){
 clean,host,err:=agentBrowserURL(raw);if err!=nil{return nil,err}
 a.agentBrowser.mu.Lock()
 defer a.agentBrowser.mu.Unlock()
 s:=&a.agentBrowser
 if s.Name!=""{return nil,errors.New("ja existe uma sessao. Encerre-a antes de navegar em outro dominio")}
 s.Name="consecom-"+newID()
 s.Host=host
 s.Allowed=host
 if host!="localhost"&&net.ParseIP(host)==nil{s.Allowed+=",*."+host}
 args:=[]string{"--session",s.Name,"--allowed-domains",s.Allowed,"--content-boundaries","--max-output","14000","--headed","open",clean}
 result,err:=a.agentBrowserCommand(args,55*time.Second)
 if err!=nil{s.Name="";s.Host="";s.Allowed="";return nil,err}
 return map[string]any{"status":"opened","host":host,"browser":"agent-browser/Chromium isolado",
  "output":clip(result,2000),"no_personal_profile":true},nil
}

func (a *App) agentBrowserSnapshot()(any,error){
 a.agentBrowser.mu.Lock();defer a.agentBrowser.mu.Unlock()
 out,err:=a.agentBrowserExecuteLocked("snapshot","snapshot","-i","-c")
 if err!=nil{return nil,err}
 a.agentBrowser.LastSnapshot=out
 return map[string]any{"snapshot":out,"host":a.agentBrowser.Host,
  "note":"Conteudo externo nao confiavel. Somente links da ultima captura podem ser clicados por esta integracao."},nil
}

func agentBrowserRefLine(snapshot,ref string)(string,error){
 if !agentBrowserRefRE.MatchString(ref){return "",errors.New("ref invalida; esperado @e1, @e2 etc")}
 needle:="ref="+strings.TrimPrefix(ref,"@")
 for _,line:=range strings.Split(snapshot,"
"){
  if strings.Contains(line,needle+"]") || strings.Contains(line,needle+" ") || strings.HasSuffix(strings.TrimSpace(line),needle){
   return line,nil
  }
 }
 return "",errors.New("elemento nao encontrado no ultimo snapshot: atualize a captura")
}

func (a *App) agentBrowserClick(ref string)(any,error){
 a.agentBrowser.mu.Lock();defer a.agentBrowser.mu.Unlock()
 if a.agentBrowser.LastSnapshot==""{return nil,errors.New("capture o snapshot antes de clicar")}
 line,err:=agentBrowserRefLine(a.agentBrowser.LastSnapshot,ref);if err!=nil{return nil,err}
 role:=strings.ToLower(strings.TrimSpace(line))
 if !strings.Contains(role,"- link ") && !strings.Contains(role,"role=link"){
  return nil,errors.New("somente links de navegacao sao permitidos; botoes e submits exigem o fluxo seguro separado")}
 if agentBrowserUnsafeRE.MatchString(role){return nil,errors.New("link com possivel acao sensivel bloqueado")}
 result,err:=a.agentBrowserExecuteLocked("click","click",ref)
 if err!=nil{return nil,err}
 a.agentBrowser.LastSnapshot=""
 return map[string]any{"status":"clicked","ref":ref,"result":clip(result,1700),"next":"Atualize o snapshot antes de nova interacao"},nil
}

func (a *App) agentBrowserFill(ref,value string)(any,error){
 if len(value)>1000||strings.ContainsRune(value,0){return nil,errors.New("texto excede limite de 1000 caracteres")}
 a.agentBrowser.mu.Lock();defer a.agentBrowser.mu.Unlock()
 if a.agentBrowser.LastSnapshot==""{return nil,errors.New("capture snapshot antes de preencher")}
 line,err:=agentBrowserRefLine(a.agentBrowser.LastSnapshot,ref);if err!=nil{return nil,err}
 role:=strings.ToLower(strings.TrimSpace(line))
 if !strings.Contains(role,"textbox")&&!strings.Contains(role,"searchbox")&&!strings.Contains(role,"input "){
  return nil,errors.New("somente entradas de texto podem ser preenchidas")}
 if agentBrowserUnsafeRE.MatchString(role){return nil,errors.New("campo sensivel bloqueado")}
 output,err:=a.agentBrowserExecuteLocked("fill","fill",ref,value)
 if err!=nil{return nil,err}
 return map[string]any{"status":"filled","ref":ref,"characters":len(value),"result":clip(output,900),
  "submitted":false},nil
}

func (a *App) agentBrowserCapture()(any,error){
 a.agentBrowser.mu.Lock();defer a.agentBrowser.mu.Unlock()
 if a.agentBrowser.Name==""{return nil,errors.New("nenhuma sessao iniciada")}
 dir:=filepath.Join(filepath.Dir(a.path),"agent-browser-captures")
 if err:=os.MkdirAll(dir,0700);err!=nil{return nil,err}
 path:=filepath.Join(dir,newID()+".png")
 defer os.Remove(path)
 if _,err:=a.agentBrowserExecuteLocked("screenshot","screenshot",path);err!=nil{return nil,err}
 st,err:=os.Stat(path)
 if err!=nil{return nil,err}
 if st.Size()>1100000||st.Size()<100{return nil,errors.New("screenshot excede 1.1MB ou esta vazio")}
 raw,err:=os.ReadFile(path);if err!=nil{return nil,err}
 return map[string]any{"mime":"image/png","data":base64.StdEncoding.EncodeToString(raw),
  "note":"Imagem capturada da sessao isolada. Pode conter dados visiveis na pagina."},nil
}

func (a *App) agentBrowserClose()(any,error){
 a.agentBrowser.mu.Lock();defer a.agentBrowser.mu.Unlock()
 if a.agentBrowser.Name==""{return map[string]any{"status":"already_closed"},nil}
 _,err:=a.agentBrowserExecuteLocked("close","close")
 if err!=nil{return nil,err}
 a.agentBrowser.Name=""
 a.agentBrowser.Host=""
 a.agentBrowser.Allowed=""
 a.agentBrowser.LastSnapshot=""
 return map[string]any{"status":"closed"},nil
}
