package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "time"
 "unicode/utf8"
)

type EditorBackupRecord struct {
 ID string `json:"id"`
 RootID string `json:"root_id"`
 Path string `json:"path"`
 BeforeSHA string `json:"before_sha256"`
 AfterSHA string `json:"after_sha256"`
 Created time.Time `json:"created"`
}

var editorBackupID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func editorBackupDirectory(a *App) string {
 return filepath.Join(filepath.Dir(a.path), "editor-backups")
}

// Save both the original bytes and a bounded, verifiable restoration manifest.
// If metadata creation fails, no project file is modified.
func saveEditorBackup(a *App, rootID, rel string, before []byte, afterSHA string) (string,error) {
 dir:=editorBackupDirectory(a)
 if err:=os.MkdirAll(dir,0700);err!=nil{return "",err}
 id:=newID()
 record:=EditorBackupRecord{ID:id,RootID:rootID,Path:rel,BeforeSHA:codeSHA(before),AfterSHA:afterSHA,Created:time.Now().UTC()}
 data,err:=json.Marshal(record);if err!=nil{return "",err}
 backupPath:=filepath.Join(dir,id+".bak")
 f,err:=os.OpenFile(backupPath,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600);if err!=nil{return "",err}
 _,err=f.Write(before)
 if err==nil{err=f.Sync()}
 closeErr:=f.Close()
 if err==nil{err=closeErr}
 if err!=nil{os.Remove(backupPath);return "",err}
 metaPath:=filepath.Join(dir,id+".json")
 m,err:=os.OpenFile(metaPath,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600)
 if err!=nil{os.Remove(backupPath);return "",err}
 _,err=m.Write(data)
 if err==nil{err=m.Sync()}
 closeErr=m.Close()
 if err==nil{err=closeErr}
 if err!=nil{os.Remove(metaPath);os.Remove(backupPath);return "",err}
 return id,nil
}

func loadEditorBackup(a *App,id string)(EditorBackupRecord,[]byte,error){
 var r EditorBackupRecord
 if !editorBackupID.MatchString(id){return r,nil,errors.New("id de backup invalido")}
 dir:=editorBackupDirectory(a)
 metaPath:=filepath.Join(dir,id+".json")
 st,err:=os.Lstat(metaPath)
 if err!=nil{return r,nil,err}
 if !st.Mode().IsRegular()||st.Size()>4096{return r,nil,errors.New("metadados de backup invalidos")}
 data,err:=os.ReadFile(metaPath);if err!=nil{return r,nil,err}
 if err=json.Unmarshal(data,&r);err!=nil{return r,nil,err}
 if r.ID!=id||!validSha.MatchString(r.BeforeSHA)||!validSha.MatchString(r.AfterSHA)||r.RootID==""||r.Path==""{
  return r,nil,errors.New("metadados de backup inconsistentes")
 }
 backupPath:=filepath.Join(dir,id+".bak")
 st,err=os.Lstat(backupPath);if err!=nil{return r,nil,err}
 if !st.Mode().IsRegular()||st.Size()>1024*1024{return r,nil,errors.New("backup nao e arquivo regular ate 1 MiB")}
 before,err:=os.ReadFile(backupPath);if err!=nil{return r,nil,err}
 if codeSHA(before)!=r.BeforeSHA{return r,nil,errors.New("checksum do backup divergente")}
 return r,before,nil
}

func (a *App) codeDiffPreview(req EditorEdit)(any,error){
 root,e:=a.getRoot(req.RootID);if e!=nil{return nil,e}
 if len(req.Old)<1||len(req.Old)>28000||len(req.New)>28000||!utf8EditorText(req.Old)||!utf8EditorText(req.New){return nil,errors.New("texto invalido ou acima de 28 KiB")}
 _,before,e:=codeReadChecked(root,req.Path);if e!=nil{return nil,e}
 if codeSHA(before)!=req.ExpectedSHA{return nil,errors.New("SHA256 desatualizado; leia o arquivo novamente")}
 if strings.Count(string(before),req.Old)!=1{return nil,errors.New("trecho deve aparecer exatamente uma vez")}
 after:=strings.Replace(string(before),req.Old,req.New,1)
 if len(after)>1024*1024{return nil,errors.New("arquivo resultante excede 1 MiB")}
 diff,truncated:=editorUnifiedPreview(req.Path,string(before),after)
 return map[string]any{"path":req.Path,"before_sha256":codeSHA(before),"after_sha256":codeSHA([]byte(after)),"changed":string(before)!=after,"diff":diff,"truncated":truncated},nil
}

func utf8EditorText(s string)bool{
 return !strings.ContainsRune(s,0)&&utf8.ValidString(s)
}

// A single contiguous line hunk, with up to 3 lines of context.
// This is for review, not a substitute for a Git patch apply operation.
func editorUnifiedPreview(path,before,after string)(string,bool){
 a:=strings.Split(before,"\n");b:=strings.Split(after,"\n")
 prefix:=0
 for prefix<len(a)&&prefix<len(b)&&a[prefix]==b[prefix]{prefix++}
 suffix:=0
 for suffix<len(a)-prefix&&suffix<len(b)-prefix&&a[len(a)-1-suffix]==b[len(b)-1-suffix]{suffix++}
 if prefix==len(a)&&prefix==len(b){return "",false}
 start:=prefix-3;if start<0{start=0}
 endA:=len(a)-suffix+3;if endA>len(a){endA=len(a)}
 endB:=len(b)-suffix+3;if endB>len(b){endB=len(b)}
 var sb strings.Builder
 fmt.Fprintf(&sb,"--- a/%s\n+++ b/%s\n@@ -%d,%d +%d,%d @@\n",path,path,start+1,endA-start,start+1,endB-start)
 for _,line:=range a[start:prefix]{sb.WriteString(" "+line+"\n")}
 for _,line:=range a[prefix:len(a)-suffix]{sb.WriteString("-"+line+"\n")}
 for _,line:=range b[prefix:len(b)-suffix]{sb.WriteString("+"+line+"\n")}
 for _,line:=range a[len(a)-suffix:endA]{sb.WriteString(" "+line+"\n")}
 s:=sb.String()
 if len(s)>24000{return s[:24000],true}
 return s,false
}

func (a *App) codeRestoreRequest(id string)(any,error){
 record,_,err:=loadEditorBackup(a,id);if err!=nil{return nil,err}
 root,err:=a.getRoot(record.RootID);if err!=nil{return nil,err}
 if !root.Writable{return nil,errors.New("escrita do projeto desabilitada")}
 _,current,err:=codeReadChecked(root,record.Path);if err!=nil{return nil,err}
 if codeSHA(current)!=record.AfterSHA{return nil,errors.New("arquivo mudou desde o backup; restauracao recusada")}
 p:=&Pending{Kind:"coderestore",Path:root.Path,Content:id,Description:"Restaurar backup "+id+" do arquivo "+record.Path+" (com verificacao SHA256)"}
 if a.mode()=="maximum"{
  result,err:=a.codeRestoreApproved(p);if err!=nil{return nil,err}
  return map[string]any{"status":"executada","resultado":result},nil
 }
 approvalID,err:=a.queue(p);if err!=nil{return nil,err}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":approvalID},nil
}

func (a *App) codeRestoreApproved(p *Pending)(string,error){
 editorMu.Lock();defer editorMu.Unlock()
 r,backup,err:=loadEditorBackup(a,p.Content);if err!=nil{return "",err}
 root,err:=a.getRoot(r.RootID);if err!=nil{return "",err}
 if !root.Writable||filepath.Clean(root.Path)!=filepath.Clean(p.Path){return "",errors.New("escopo ou permissao do projeto mudou")}
 path,current,err:=codeReadChecked(root,r.Path);if err!=nil{return "",err}
 if codeSHA(current)!=r.AfterSHA{return "",errors.New("conflito SHA256; arquivo nao alterado")}
 if _,err=saveEditorBackup(a,r.RootID,r.Path,current,r.BeforeSHA);err!=nil{return "",fmt.Errorf("backup preventivo de restauracao falhou: %w",err)}
 tmp:=path+".consecom-"+newID()+".tmp"
 f,err:=os.OpenFile(tmp,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600);if err!=nil{return "",err}
 _,err=f.Write(backup);if err==nil{err=f.Sync()}
 closeErr:=f.Close();if err==nil{err=closeErr}
 if err!=nil{os.Remove(tmp);return "",err}
 if err=replaceProjectFile(tmp,path);err!=nil{os.Remove(tmp);return "",fmt.Errorf("restauracao nao aplicada: %w",err)}
 return fmt.Sprintf("Arquivo restaurado: %s; sha256: %s",r.Path,r.BeforeSHA),nil
}
