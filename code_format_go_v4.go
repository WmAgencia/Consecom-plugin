package main

import (
 "bytes"
 "encoding/json"
 "errors"
 "fmt"
 "go/format"
 "os"
 "path/filepath"
 "strings"
)

type FormatGoSpec struct{
 RootID string `json:"root_id"`
 Path string `json:"path"`
 ExpectedSHA string `json:"expected_sha256"`
}

func (a *App) codeFormatGoRequest(spec FormatGoSpec)(any,error){
 root,err:=a.getRoot(spec.RootID);if err!=nil{return nil,err}
 if !root.Writable{return nil,errors.New("projeto sem permissao de escrita")}
 if !strings.EqualFold(filepath.Ext(spec.Path),".go"){return nil,errors.New("gofmt permitido somente em arquivos .go")}
 _,before,err:=codeReadChecked(root,spec.Path);if err!=nil{return nil,err}
 if codeSHA(before)!=spec.ExpectedSHA{return nil,errors.New("SHA256 desatualizado")}
 after,err:=format.Source(before);if err!=nil{return nil,fmt.Errorf("gofmt recusou arquivo com sintaxe invalida: %w",err)}
 if bytes.Equal(before,after){return map[string]any{"status":"sem_alteracoes","sha256":spec.ExpectedSHA},nil}
 data,err:=json.Marshal(spec);if err!=nil{return nil,err}
 pending:=&Pending{Kind:"codeformat",Path:root.Path,Content:string(data),Description:"Formatar Go em "+spec.Path+" com gofmt, SHA256 e backup"}
 if a.mode()=="maximum"{
  detail,err:=a.codeFormatGoApproved(pending);if err!=nil{return nil,err}
  return map[string]any{"status":"executada","resultado":detail},nil
 }
 id,err:=a.queue(pending);if err!=nil{return nil,err}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}

func (a *App) codeFormatGoApproved(p *Pending)(string,error){
 var spec FormatGoSpec
 if err:=json.Unmarshal([]byte(p.Content),&spec);err!=nil{return "",err}
 editorMu.Lock();defer editorMu.Unlock()
 root,err:=a.getRoot(spec.RootID);if err!=nil{return "",err}
 if !root.Writable||filepath.Clean(root.Path)!=filepath.Clean(p.Path){return "",errors.New("permissao de projeto alterada")}
 if !strings.EqualFold(filepath.Ext(spec.Path),".go"){return "",errors.New("somente arquivos Go")}
 path,before,err:=codeReadChecked(root,spec.Path);if err!=nil{return "",err}
 if codeSHA(before)!=spec.ExpectedSHA{return "",errors.New("arquivo mudou; formatacao cancelada")}
 after,err:=format.Source(before);if err!=nil{return "",err}
 if bytes.Equal(before,after){return "Arquivo ja formatado",nil}
 backupID,err:=saveEditorBackup(a,spec.RootID,spec.Path,before,codeSHA(after));if err!=nil{return "",err}
 temp:=path+".consecom-"+newID()+".tmp"
 fd,err:=os.OpenFile(temp,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600);if err!=nil{return "",err}
 _,err=fd.Write(after)
 if err==nil{err=fd.Sync()}
 closeErr:=fd.Close()
 if err==nil{err=closeErr}
 if err!=nil{os.Remove(temp);return "",err}
 if err=replaceProjectFile(temp,path);err!=nil{os.Remove(temp);return "",err}
 return fmt.Sprintf("Go formatado: %s; sha256: %s; backup: %s",spec.Path,codeSHA(after),backupID),nil
}
