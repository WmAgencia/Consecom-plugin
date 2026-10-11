package main

import(
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "regexp"
 "strings"
)
type browserApproval struct{
 Selector string `json:"selector"`
 URL string `json:"url"`
 Target string `json:"target"`
 Label string `json:"label"`
}
var browserFilename=regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._ -]{0,130}$`)
func (a *App) browserDownloadDir()(string,error){
 dir:=filepath.Join(filepath.Dir(a.path),"downloads")
 if e:=os.MkdirAll(dir,0700);e!=nil{return "",e}
 return dir,nil
}
func (a *App) browserEnableDownloads()(any,error){
 a.browserControl.mu.Lock();defer a.browserControl.mu.Unlock()
 s:=a.browserControl.session
 if s==nil{return nil,errors.New("abra navegador primeiro")}
 dir,e:=a.browserDownloadDir();if e!=nil{return nil,e}
 _,e=cdpCall(s,"Page.setDownloadBehavior",map[string]any{"behavior":"allow","downloadPath":dir})
 if e!=nil{return nil,e}
 return map[string]any{"status":"downloads_habilitados","scope":"diretorio privado do Consecom","note":"arquivos so sao copiados para projetos com aprovacao local"},nil
}
func (a *App) browserDownloads()(any,error){
 dir,e:=a.browserDownloadDir();if e!=nil{return nil,e}
 files,e:=os.ReadDir(dir);if e!=nil{return nil,e}
 list:=[]map[string]any{}
 for _,entry:=range files {
  if len(list)>=50{break}
  if !browserFilename.MatchString(entry.Name())||entry.IsDir()||strings.HasSuffix(entry.Name(),".crdownload"){continue}
  stat,e:=entry.Info();if e!=nil||!stat.Mode().IsRegular(){continue}
  list=append(list,map[string]any{"name":entry.Name(),"size_bytes":stat.Size(),"modified_at":stat.ModTime()})
 }
 return map[string]any{"files":list},nil
}
func browserUploadAllowed(ext string)bool{
 switch strings.ToLower(ext){
 case ".png",".jpg",".jpeg",".webp",".pdf",".csv",".txt",".md",".json",".mp4",".zip",".svg":return true
 }
 return false
}
func (a *App) browserUploadRequest(rootID,relative,selector string)(any,error){
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 path,e:=safePath(root,relative,false);if e!=nil{return nil,e}
 fi,e:=os.Stat(path);if e!=nil{return nil,e}
 if !fi.Mode().IsRegular()||fi.Size()>20<<20||!browserUploadAllowed(filepath.Ext(path)){return nil,errors.New("arquivo nao autorizado ou acima de 20 MB")}
 if len(selector)<1||len(selector)>300{return nil,errors.New("seletor CSS invalido")}
 a.browserControl.mu.Lock()
 s:=a.browserControl.session
 if s==nil {a.browserControl.mu.Unlock();return nil,errors.New("navegador nao conectado")}
 snapshot,e:=cdpEvaluate(s,"location.href")
 target:=s.Target
 a.browserControl.mu.Unlock()
 if e!=nil{return nil,e}
 page,ok:=snapshot.(string);if !ok{return nil,errors.New("url invalida")}
 approval:=browserApproval{Selector:selector,URL:page,Target:target}
 content,_:=json.Marshal(approval)
 id,e:=a.queue(&Pending{Kind:"browserupload",Path:path,Content:string(content),Description:"Upload de "+filepath.Base(path)+" ("+fmt.Sprint(fi.Size())+" bytes) no site "+page+"; confirme no PC"})
 if e!=nil{return nil,e}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) browserSensitiveRequest(selector string)(any,error){
 if len(selector)<1||len(selector)>300{return nil,errors.New("seletor invalido")}
 a.browserControl.mu.Lock();s:=a.browserControl.session
 if s==nil{a.browserControl.mu.Unlock();return nil,errors.New("navegador nao conectado")}
 info,e:=cdpEvaluate(s,`(()=>{let el=document.querySelector(`+jsonJS(selector)+`);if(!el)return null;return {url:location.href,text:(el.innerText||el.value||el.getAttribute('aria-label')||'').slice(0,160),type:(el.getAttribute('type')||'').toLowerCase(),tag:el.tagName.toLowerCase()}})()`)
 target:=s.Target;a.browserControl.mu.Unlock()
 if e!=nil{return nil,e}
 m,ok:=info.(map[string]any);if !ok{return nil,errors.New("botao nao encontrado")}
 if m["type"]=="password"||m["tag"]=="input"&&m["type"]=="file" {return nil,errors.New("elemento protegido")}
 label,_:=m["text"].(string);page,_:=m["url"].(string)
 if page==""{return nil,errors.New("pagina indisponivel")}
 approval:=browserApproval{Selector:selector,URL:page,Target:target,Label:label}
 b,_:=json.Marshal(approval)
 id,e:=a.queue(&Pending{Kind:"browserconfirm",Content:string(b),Description:"Confirmar clique que pode enviar/publicar/excluir em "+page+": "+clip(label,90)})
 if e!=nil{return nil,e}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) approveBrowserTransfer(p *Pending)(string,error){
 var info browserApproval
 if e:=json.Unmarshal([]byte(p.Content),&info);e!=nil{return "",e}
 a.browserControl.mu.Lock();defer a.browserControl.mu.Unlock()
 s:=a.browserControl.session
 if s==nil||s.Target!=info.Target {return "",errors.New("aba mudou; solicite aprovacao novamente")}
 urlVal,e:=cdpEvaluate(s,"location.href");if e!=nil{return "",e}
 if urlVal!=info.URL{return "",errors.New("url mudou desde a aprovacao")}
 if p.Kind=="browserupload"{
  fi,e:=os.Stat(p.Path);if e!=nil{return "",e}
  if !fi.Mode().IsRegular()||fi.Size()>20<<20||!browserUploadAllowed(filepath.Ext(p.Path)){return "",errors.New("arquivo deixou de ser permitido")}
  code:=`(()=>{const x=document.querySelector(`+jsonJS(info.Selector)+`);return !!x&&x.matches('input[type="file"]')})()`
  valid,e:=cdpEvaluate(s,code);if e!=nil||valid!=true{return "",errors.New("input de arquivo nao encontrado")}
  doc,e:=cdpCall(s,"DOM.getDocument",map[string]any{});if e!=nil{return "",e}
  var d struct{Root struct{NodeID int `json:"nodeId"`} `json:"root"`}
  if e=json.Unmarshal(doc,&d);e!=nil{return "",e}
  q,e:=cdpCall(s,"DOM.querySelector",map[string]any{"nodeId":d.Root.NodeID,"selector":info.Selector});if e!=nil{return "",e}
  var node struct{NodeID int `json:"nodeId"`};if e=json.Unmarshal(q,&node);e!=nil{return "",e}
  if node.NodeID<=0{return "",errors.New("campo de upload nao encontrado")}
  _,e=cdpCall(s,"DOM.setFileInputFiles",map[string]any{"files":[]string{p.Path},"nodeId":node.NodeID})
  if e!=nil{return "",e}
  return "Arquivo selecionado no campo aprovado. O envio final exige aprovacao separada.",nil
 }
 if p.Kind=="browserconfirm" {
  code:=`(()=>{const x=document.querySelector(`+jsonJS(info.Selector)+`);if(!x)return {ok:false};const txt=(x.innerText||x.value||x.getAttribute('aria-label')||'').slice(0,160);if(txt!==`+jsonJS(info.Label)+`)return {ok:false,reason:'rotulo alterado'};if(x.matches('input[type="password"],input[type="file"]'))return {ok:false};x.click();return {ok:true}})()`
  got,e:=cdpEvaluate(s,code);if e!=nil{return "",e}
  obj,ok:=got.(map[string]any)
  if !ok||obj["ok"]!=true{return "",errors.New("elemento mudou; clique cancelado")}
  return "Clique sensivel aprovado localmente e executado",nil
 }
 return "",errors.New("operacao invalida")
}
func (a *App) browserDownloadCopyRequest(name,rootID,relative string)(any,error){
 if !browserFilename.MatchString(name){return nil,errors.New("nome de download invalido")}
 dir,e:=a.browserDownloadDir();if e!=nil{return nil,e}
 file:=filepath.Join(dir,name)
 stat,e:=os.Lstat(file);if e!=nil{return nil,e}
 if !stat.Mode().IsRegular()||stat.Size()>20<<20||!browserUploadAllowed(filepath.Ext(name)){return nil,errors.New("download nao permitido ou acima de 20 MB")}
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 dest,e:=safePath(root,relative,true);if e!=nil{return nil,e}
 data,_:=json.Marshal(map[string]string{"source":file,"destination":dest,"root":root.Path})
 id,e:=a.queue(&Pending{Kind:"browsercopy",Content:string(data),Description:"Copiar download "+name+" para "+dest+" sem substituir arquivo existente"})
 if e!=nil{return nil,e}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) approveBrowserCopy(p *Pending)(string,error){
 var v map[string]string
 if e:=json.Unmarshal([]byte(p.Content),&v);e!=nil{return "",e}
 rootOK:=false
 for _,r:=range a.roots(){if filepath.Clean(r.Path)==filepath.Clean(v["root"])&&r.Writable {if _,e:=safePath(r,strings.TrimPrefix(strings.TrimPrefix(v["destination"],r.Path),string(os.PathSeparator)),true);e==nil{rootOK=true}}}
 if !rootOK{return "",errors.New("destino deixou de ser permitido")}
 dir,e:=a.browserDownloadDir();if e!=nil{return "",e}
 if filepath.Dir(v["source"])!=dir||!browserFilename.MatchString(filepath.Base(v["source"])){return "",errors.New("origem invalida")}
 f,e:=os.Open(v["source"]);if e!=nil{return "",e};defer f.Close()
 st,e:=f.Stat();if e!=nil||!st.Mode().IsRegular()||st.Size()>20<<20{return "",errors.New("download nao e arquivo permitido")}
 out,e:=os.OpenFile(v["destination"],os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if e!=nil{return "",e};defer out.Close()
 _,e=io.Copy(out,io.LimitReader(f,20<<20+1))
 if e!=nil{os.Remove(v["destination"]);return "",e}
 return "Download copiado com sucesso",nil
}
