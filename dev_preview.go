package main

import (
 "bytes"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "net"
 "net/http"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strconv"
 "strings"
 "sync"
 "time"
)

type DevPreview struct {
 ID string `json:"id"`
 RootID string `json:"root_id"`
 Framework string `json:"framework"`
 URL string `json:"url"`
 Port int `json:"port"`
 Status string `json:"status"`
 Started time.Time `json:"started"`
 Finished time.Time `json:"finished,omitempty"`
 Error string `json:"error,omitempty"`
 cmd *exec.Cmd
 timer *time.Timer
}

type DevPreviewManager struct{
 mu sync.Mutex
 sessions map[string]*DevPreview
}

const devPreviewMaxDuration = 60*time.Minute

type devPreviewStartRequest struct {
 RootID string `json:"root_id"`
 Port int `json:"port"`
}

func readPreviewFramework(root Root)(string,error){
 path,err:=safePath(root,"package.json",false);if err!=nil{return "",err}
 b,err:=os.ReadFile(path);if err!=nil{return "",errors.New("package.json nao encontrado: crie um projeto Node primeiro")}
 if len(b)>262144{return "",errors.New("package.json excede 256 KiB")}
 var pkg struct {
  Scripts map[string]json.RawMessage `json:"scripts"`
  Dependencies map[string]json.RawMessage `json:"dependencies"`
  DevDependencies map[string]json.RawMessage `json:"devDependencies"`
 }
 if err=json.Unmarshal(b,&pkg);err!=nil{return "",errors.New("package.json invalido")}
 if len(pkg.Scripts["dev"])==0{return "",errors.New("script npm 'dev' ausente; nao executamos scripts arbitrarios")}
 deps:=map[string]bool{}
 for k:=range pkg.Dependencies{deps[k]=true}
 for k:=range pkg.DevDependencies{deps[k]=true}
 // Limit to known CLI syntaxes. No user-provided argument enters cmd.exe.
 switch {
 case deps["next"]:return "next",nil
 case deps["vite"]:return "vite",nil
 case deps["astro"]:return "astro",nil
 default:return "",errors.New("preview disponivel inicialmente para Next.js, Vite e Astro")
 }
}

func (a *App) previewStart(rootID string,port int,approved bool)(any,error){
 if port<1024||port>65535{return nil,errors.New("porta deve estar entre 1024 e 65535")}
 root,err:=a.getRoot(rootID);if err!=nil{return nil,err}
 if !root.Writable{return nil,errors.New("habilite escrita para este projeto no painel antes de executar o servidor dev")}
 if isBroadOrDangerousRoot(root.Path){return nil,errors.New("projeto fora do escopo seguro")}
 if err=ensureNoLinks(root.Path);err!=nil{return nil,err}
 framework,err:=readPreviewFramework(root);if err!=nil{return nil,err}
 if !projectFileExists(root.Path,"node_modules"){return nil,errors.New("node_modules ausente; instale as dependencias do projeto antes")}
 if _,err=exec.LookPath("npm");err!=nil{return nil,errors.New("npm nao instalado ou indisponivel no PATH")}
 if !approved&&a.mode()!="maximum"{
  b,_:=json.Marshal(devPreviewStartRequest{RootID:rootID,Port:port})
  id,err:=a.queue(&Pending{Kind:"devpreview",Path:root.Path,Content:string(b),Description:fmt.Sprintf("Iniciar npm run dev em %s, apenas em 127.0.0.1:%d (maximo 60 minutos)",root.Name,port)})
  if err!=nil{return nil,err}
  return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
 }
 a.devPreviews.mu.Lock()
 if a.devPreviews.sessions==nil{a.devPreviews.sessions=map[string]*DevPreview{}}
 running:=0
 for _,s:=range a.devPreviews.sessions{
  if s.Status=="running"{
   running++
   if s.RootID==rootID||s.Port==port{a.devPreviews.mu.Unlock();return nil,errors.New("projeto ou porta ja tem preview ativo")}
  }
 }
 if running>=2{a.devPreviews.mu.Unlock();return nil,errors.New("limite de 2 previews simultaneos")}
 listener,err:=net.Listen("tcp",net.JoinHostPort("127.0.0.1",strconv.Itoa(port)))
 if err!=nil{a.devPreviews.mu.Unlock();return nil,fmt.Errorf("porta local ocupada: %w",err)}
 listener.Close()
 args:="npm run dev -- "
 if framework=="next"{args+="--hostname 127.0.0.1"}else{args+="--host 127.0.0.1"}
 args+=" --port "+strconv.Itoa(port)
 var cmd *exec.Cmd
 if runtime.GOOS=="windows"{
  cmd=exec.Command("cmd.exe","/D","/S","/C",args)
 }else{
  cmd=exec.Command("npm","run","dev","--",strings.TrimSpace(strings.Split(args,"-- ")[1]))
  // Non-Windows is not the shipping product; preserve argument separation.
  opt:=[]string{"run","dev","--"}
  if framework=="next"{opt=append(opt,"--hostname")}else{opt=append(opt,"--host")}
  opt=append(opt,"127.0.0.1","--port",strconv.Itoa(port))
  cmd=exec.Command("npm",opt...)
 }
 cmd.Dir=root.Path
 cmd.Env=append(filteredDeveloperEnvironment(),"PORT="+strconv.Itoa(port),"HOST=127.0.0.1","HOSTNAME=127.0.0.1","BROWSER=none","GIT_TERMINAL_PROMPT=0")
 // Avoid surfacing arbitrary npm output; scripts may print project secrets.
 cmd.Stdout=io.Discard
 cmd.Stderr=io.Discard
 if err=cmd.Start();err!=nil{a.devPreviews.mu.Unlock();return nil,fmt.Errorf("falha ao iniciar servidor: %w",err)}
 preview:=&DevPreview{ID:newID(),RootID:rootID,Framework:framework,URL:fmt.Sprintf("http://127.0.0.1:%d",port),Port:port,Started:time.Now(),Status:"running",cmd:cmd}
 a.devPreviews.sessions[preview.ID]=preview
 a.devPreviews.mu.Unlock()
 preview.timer=time.AfterFunc(devPreviewMaxDuration,func(){_,_=a.previewStop(preview.ID)})
 go func(){
  err:=cmd.Wait()
  a.devPreviews.mu.Lock()
  defer a.devPreviews.mu.Unlock()
  if preview.timer!=nil{preview.timer.Stop()}
  if preview.Status=="running"{
   preview.Status="exited"
   if err!=nil{preview.Error="processo terminou com erro; execute npm run dev localmente para obter diagnostico detalhado"}
  }
  preview.Finished=time.Now()
 }()
 return a.previewStatus(preview.ID)
}

func (a *App) previewApproved(p *Pending)(string,error){
 var req devPreviewStartRequest
 if err:=json.Unmarshal([]byte(p.Content),&req);err!=nil{return "",err}
 root,err:=a.getRoot(req.RootID);if err!=nil{return "",err}
 if filepath.Clean(root.Path)!=filepath.Clean(p.Path){return "",errors.New("projeto autorizado mudou")}
 r,err:=a.previewStart(req.RootID,req.Port,true)
 if err!=nil{return "",err}
 return fmt.Sprintf("Preview iniciado: %v",r),nil
}

func publicPreview(s *DevPreview)map[string]any{
 return map[string]any{"id":s.ID,"root_id":s.RootID,"framework":s.Framework,"url":s.URL,"port":s.Port,"status":s.Status,"started":s.Started,"finished":s.Finished,"error":s.Error,"max_duration_minutes":60}
}

func (a *App) previewStatus(id string)(any,error){
 a.devPreviews.mu.Lock();defer a.devPreviews.mu.Unlock()
 p:=a.devPreviews.sessions[id];if p==nil{return nil,errors.New("preview nao encontrado")}
 return publicPreview(p),nil
}

func (a *App) previewList()(any,error){
 a.devPreviews.mu.Lock();defer a.devPreviews.mu.Unlock()
 list:=[]map[string]any{}
 for _,p:=range a.devPreviews.sessions{list=append(list,publicPreview(p))}
 return list,nil
}

func (a *App) previewStop(id string)(any,error){
 a.devPreviews.mu.Lock()
 p:=a.devPreviews.sessions[id]
 if p==nil{a.devPreviews.mu.Unlock();return nil,errors.New("preview nao encontrado")}
 if p.Status!="running"{a.devPreviews.mu.Unlock();return nil,errors.New("preview ja encerrado")}
 p.Status="stopping"
 if p.timer!=nil{p.timer.Stop()}
 proc:=p.cmd.Process
 a.devPreviews.mu.Unlock()
 if proc!=nil{stopPreviewProcessTree(proc)}
 a.devPreviews.mu.Lock()
 p.Status="stopped"
 p.Finished=time.Now()
 a.devPreviews.mu.Unlock()
 return map[string]any{"id":id,"status":"stopped"},nil
}

func (a *App) previewCheck(id string)(any,error){
 a.devPreviews.mu.Lock()
 s:=a.devPreviews.sessions[id]
 if s==nil{a.devPreviews.mu.Unlock();return nil,errors.New("preview nao encontrado")}
 if s.Status!="running"{a.devPreviews.mu.Unlock();return nil,errors.New("preview nao esta em execucao")}
 localURL:=s.URL
 a.devPreviews.mu.Unlock()
 cli:=http.Client{Timeout:4*time.Second,CheckRedirect:func(req *http.Request,via []*http.Request)error{return http.ErrUseLastResponse}}
 resp,err:=cli.Get(localURL)
 if err!=nil{return map[string]any{"ready":false,"url":localURL,"note":"Servidor ainda nao responde no loopback; aguarde ou confira o script dev no projeto."},nil}
 defer resp.Body.Close()
 body,err:=io.ReadAll(io.LimitReader(resp.Body,1024*1024))
 if err!=nil{return nil,err}
 lower:=strings.ToLower(string(body))
 getTag:=func(tag string)bool{return strings.Contains(lower,"<"+tag+" ")||strings.Contains(lower,"<"+tag+">")}
 html:=strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")),"text/html")
 return map[string]any{
  "ready":resp.StatusCode>=200&&resp.StatusCode<400,"url":localURL,
  "http_status":resp.StatusCode,"content_type":resp.Header.Get("Content-Type"),
  "html_detected":html,"title_present":getTag("title"),
  "viewport_meta_present":strings.Contains(lower,"name=\"viewport\"")||strings.Contains(lower,"name='viewport'"),
  "description_meta_present":strings.Contains(lower,"name=\"description\"")||strings.Contains(lower,"name='description'"),
  "h1_present":getTag("h1"),
  "checked_bytes":len(body),
  "note":"Auditoria HTTP basica; nao substitui Lighthouse, Playwright ou teste completo de SEO, desempenho e acessibilidade.",
 },nil
}

var _ = bytes.MinRead
