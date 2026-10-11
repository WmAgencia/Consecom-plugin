package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "io/fs"
 "os"
 "path/filepath"
 "sort"
 "strings"
 "sync"
 "unicode/utf8"
)
var editorMu sync.Mutex
type EditorEdit struct {
 RootID string `json:"root_id"`
 Path string `json:"path"`
 ExpectedSHA string `json:"expected_sha256"`
 Old string `json:"old"`
 New string `json:"new"`
}
func codeSHA(b []byte)string{hash:=sha256.Sum256(b);return hex.EncodeToString(hash[:])}
func codeReadChecked(root Root, rel string)(string,[]byte,error){
 path,e:=safePath(root,rel,false);if e!=nil{return "",nil,e}
 st,e:=os.Lstat(path);if e!=nil{return "",nil,e}
 if !st.Mode().IsRegular()||st.Size()>1024*1024{return "",nil,errors.New("arquivo deve ser texto regular menor que 1 MiB")}
 if !permittedExt[strings.ToLower(filepath.Ext(path))]{return "",nil,errors.New("extensao nao autorizada")}
 b,e:=os.ReadFile(path);if e!=nil{return "",nil,e}
 if !utf8.Valid(b)||strings.ContainsRune(string(b),0){return "",nil,errors.New("arquivo nao e UTF-8 valido")}
 return path,b,nil
}
func (a *App) codeSnippet(rootID,rel string,start,count int)(any,error){
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 _,b,e:=codeReadChecked(root,rel);if e!=nil{return nil,e}
 if start<1||count<1||count>160{return nil,errors.New("intervalo de linhas invalido (maximo 160)")}
 lines:=strings.Split(string(b),string(rune(10)))
 if start>len(lines){return nil,errors.New("inicio alem do fim do arquivo")}
 last:=start+count-1
 if last>len(lines){last=len(lines)}
 var out strings.Builder
 for i:=start-1;i<last;i++{fmt.Fprintln(&out,fmt.Sprintf("%d | %s",i+1,lines[i]))}
 return map[string]any{"path":rel,"sha256":codeSHA(b),"line_count":len(lines),"start":start,"end":last,"content":out.String()},nil
}
func (a *App) codeSearch(rootID,term string)(any,error){
 root,e:=a.getRoot(rootID);if e!=nil{return nil,e}
 if len(term)<2||len(term)>100||strings.IndexFunc(term,func(r rune)bool{return r<32})>=0{return nil,errors.New("busca de 2 a 100 caracteres em uma linha")}
 if e=ensureNoLinks(root.Path);e!=nil{return nil,e}
 hits:=[]map[string]any{};scanned:=0
 stop:=errors.New("limite de busca atingido")
 e=filepath.WalkDir(root.Path,func(path string,entry fs.DirEntry,walkErr error)error{
  if walkErr!=nil{return nil}
  rel,err:=filepath.Rel(root.Path,path);if err!=nil{return nil}
  if rel=="."{return nil}
  if entry.Type()&(os.ModeSymlink|os.ModeIrregular)!=0{return nil}
  if entry.IsDir(){
   if blockedPart(entry.Name())||strings.HasPrefix(entry.Name(),".")||entry.Name()=="dist"||entry.Name()=="build" {return filepath.SkipDir}
   return nil
  }
  if scanned>=250||len(hits)>=70{return stop}
  if !entry.Type().IsRegular()||!permittedExt[strings.ToLower(filepath.Ext(path))]{return nil}
  if _,err=safePath(root,rel,false);err!=nil{return nil}
  info,err:=entry.Info();if err!=nil||info.Size()>200000{return nil}
  scanned++
  b,err:=os.ReadFile(path);if err!=nil||!utf8.Valid(b){return nil}
  for n,line:=range strings.Split(string(b),string(rune(10))){
   if len(hits)>=70{return stop}
   if strings.Contains(strings.ToLower(line),strings.ToLower(term)){hits=append(hits,map[string]any{"path":rel,"line":n+1,"preview":clip(strings.TrimSpace(line),180)})}
  }
  return nil
 })
 if e!=nil&&e!=stop{return nil,e}
 sort.SliceStable(hits,func(i,j int)bool{return fmt.Sprint(hits[i]["path"])<fmt.Sprint(hits[j]["path"])})
 return map[string]any{"root":root.Name,"term":term,"scanned_files":scanned,"matches":hits,"truncated":len(hits)>=70||scanned>=250},nil
}
func (a *App) codeEditRequest(req EditorEdit)(any,error){
 root,e:=a.getRoot(req.RootID);if e!=nil{return nil,e}
 if !root.Writable{return nil,errors.New("escrita desabilitada para o projeto")}
 if len(req.Old)<1||len(req.Old)>28000||len(req.New)>28000||!utf8.ValidString(req.Old)||!utf8.ValidString(req.New)||strings.ContainsRune(req.New,0){return nil,errors.New("edicao excede limite ou possui texto invalido")}
 _,b,e:=codeReadChecked(root,req.Path);if e!=nil{return nil,e}
 if codeSHA(b)!=req.ExpectedSHA{return nil,errors.New("o arquivo mudou desde a leitura; obtenha SHA256 atualizado")}
 if strings.Count(string(b),req.Old)!=1{return nil,errors.New("trecho original precisa aparecer exatamente uma vez")}
 if len(b)-len(req.Old)+len(req.New)>1024*1024{return nil,errors.New("resultado excederia 1 MiB")}
 data,e:=json.Marshal(req);if e!=nil{return nil,e}
 p:=&Pending{Kind:"codeedit",Path:root.Path,Content:string(data),Description:"Alterar trecho em "+req.Path+" no projeto "+root.Name+" (SHA256 verificado e backup)"}
 if a.mode()=="maximum"{
  result,e:=a.codeEditApproved(p);if e!=nil{return nil,e}
  return map[string]any{"status":"executada","resultado":result},nil
 }
 id,e:=a.queue(p);if e!=nil{return nil,e}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) codeEditApproved(p *Pending)(string,error){
 var req EditorEdit
 if e:=json.Unmarshal([]byte(p.Content),&req);e!=nil{return "",e}
 editorMu.Lock();defer editorMu.Unlock()
 root,e:=a.getRoot(req.RootID);if e!=nil{return "",e}
 if filepath.Clean(root.Path)!=filepath.Clean(p.Path)||!root.Writable{return "",errors.New("a permissao de projeto foi alterada")}
 path,b,e:=codeReadChecked(root,req.Path);if e!=nil{return "",e}
 if codeSHA(b)!=req.ExpectedSHA{return "",errors.New("arquivo mudou; alteracao cancelada")}
 if strings.Count(string(b),req.Old)!=1{return "",errors.New("trecho original nao e unico")}
 result:=strings.Replace(string(b),req.Old,req.New,1)
 if len(result)>1024*1024||!utf8.ValidString(result){return "",errors.New("resultado invalido")}
 // Backup preserva o conteudo original antes de qualquer escrita.
 backupDir:=filepath.Join(filepath.Dir(a.path),"editor-backups")
 if e=os.MkdirAll(backupDir,0700);e!=nil{return "",e}
 backupID:=newID()
 if e=os.WriteFile(filepath.Join(backupDir,backupID+".bak"),b,0600);e!=nil{return "",e}
 // Sidecar temporario usa acesso exclusivo e e removido em falhas.
 tmp:=path+".consecom-"+newID()+".tmp"
 fd,e:=os.OpenFile(tmp,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if e!=nil{return "",e}
 if _,e=fd.Write([]byte(result));e!=nil{fd.Close();os.Remove(tmp);return "",e}
 if e=fd.Sync();e!=nil{fd.Close();os.Remove(tmp);return "",e}
 if e=fd.Close();e!=nil{os.Remove(tmp);return "",e}
 // On Windows replacement may require removal. Avoid deleting the original.
 if e=replaceProjectFile(tmp,path);e!=nil{os.Remove(tmp);return "",fmt.Errorf("backup %s guardado; substituicao falhou sem apagar original: %w",backupID,e)}
 return fmt.Sprintf("Arquivo editado: %s; sha256: %s; backup: %s",req.Path,codeSHA([]byte(result)),backupID),nil
}
