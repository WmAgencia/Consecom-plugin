package main

import (
 "os"
 "encoding/json"
 "path/filepath"
 "regexp"
 "strings"
 "testing"
)

func TestV4EditorDiffPreviewAndRestore(t *testing.T){
 project:=t.TempDir()
 path:=filepath.Join(project,"example.go")
 original:="package main\nfunc main() {\n println(\"before\")\n}\n"
 if err:=os.WriteFile(path,[]byte(original),0600);err!=nil{t.Fatal(err)}
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 a.cfg.Roots=[]Root{{ID:"editor",Name:"Restore Test",Path:project,Writable:true}}
 req:=EditorEdit{RootID:"editor",Path:"example.go",ExpectedSHA:codeSHA([]byte(original)),Old:"before",New:"after"}
 preview,err:=a.codeDiffPreview(req);if err!=nil{t.Fatal(err)}
 d:=preview.(map[string]any)
 if !strings.Contains(d["diff"].(string),"- println(\"before\")")||!strings.Contains(d["diff"].(string),"+ println(\"after\")"){t.Fatalf("diff inesperado: %s",d["diff"])}
 if d["changed"]!=true||d["truncated"]!=false{t.Fatal("metadados da previa incorretos")}
 untouched,_:=os.ReadFile(path);if string(untouched)!=original{t.Fatal("preview modificou arquivo")}
 if err=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 result,err:=a.codeEditRequest(req);if err!=nil{t.Fatal(err)}
 desc:=result.(map[string]any)["resultado"].(string)
 match:=regexp.MustCompile(`backup: ([0-9a-f]{32})`).FindStringSubmatch(desc)
 if len(match)!=2{t.Fatalf("backup nao encontrado em: %s",desc)}
 id:=match[1]
 rec,back,err:=loadEditorBackup(a,id);if err!=nil{t.Fatal(err)}
 if string(back)!=original||rec.BeforeSHA!=req.ExpectedSHA||rec.Path!="example.go"{t.Fatal("backup inconsistente")}
 if _,err=a.codeDiffPreview(req);err==nil{t.Fatal("preview aceitou SHA obsoleto")}
 if err=os.WriteFile(path,[]byte("alteracao externa"),0600);err!=nil{t.Fatal(err)}
 if _,err=a.codeRestoreRequest(id);err==nil{t.Fatal("restauracao nao detectou conflito de edicao externa")}
 changed,_:=os.ReadFile(path);if string(changed)!="alteracao externa"{t.Fatal("conflito sobrescreveu arquivo")}
 expected:="package main\nfunc main() {\n println(\"after\")\n}\n"
 if err=os.WriteFile(path,[]byte(expected),0600);err!=nil{t.Fatal(err)}
 restored,err:=a.codeRestoreRequest(id);if err!=nil{t.Fatal(err)}
 if restored.(map[string]any)["status"]!="executada"{t.Fatal("modo maximo nao restaurou")}
 current,_:=os.ReadFile(path);if string(current)!=original{t.Fatal("restauracao incorreta")}
 if _,err=a.codeRestoreRequest(id);err==nil{t.Fatal("backup antigo aceito novamente apos restauracao")}
 entries,err:=os.ReadDir(editorBackupDirectory(a));if err!=nil{t.Fatal(err)}
 if len(entries)!=4{t.Fatalf("esperados 2 snapshots e 2 manifests; recebeu %d",len(entries))}
 if _,err=a.codeRestoreRequest("../../config.json");err==nil{t.Fatal("id malicioso aceito")}
}

func TestV4EditorRestoreLocalApproval(t *testing.T){
 project:=t.TempDir()
 path:=filepath.Join(project,"x.go")
 old:=[]byte("package demo\n// old\n")
 if err:=os.WriteFile(path,old,0600);err!=nil{t.Fatal(err)}
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 a.cfg.Roots=[]Root{{ID:"test",Name:"Test",Path:project,Writable:true}}
 req:=EditorEdit{RootID:"test",Path:"x.go",ExpectedSHA:codeSHA(old),Old:"old",New:"new"}
 p:=&Pending{Kind:"codeedit",Path:project}
 data,err:=json.Marshal(req);if err!=nil{t.Fatal(err)}
 p.Content=string(data)
 detail,err:=a.codeEditApproved(p);if err!=nil{t.Fatal(err)}
 matches:=regexp.MustCompile(`backup: ([0-9a-f]{32})`).FindStringSubmatch(detail)
 if len(matches)!=2{t.Fatalf("backup nao registrado: %s",detail)}
 r,err:=a.codeRestoreRequest(matches[1]);if err!=nil{t.Fatal(err)}
 if r.(map[string]any)["status"]!="aguardando_aprovacao_local"{t.Fatal("aprovação normal nao exigida")}
 still,_:=os.ReadFile(path);if !strings.Contains(string(still),"new"){t.Fatal("gravou antes de aprovar")}
 items:=a.pendings()
 if len(items)!=1{t.Fatalf("esperada uma aprovacao, houve %d",len(items))}
 if _,err=a.approve(items[0]["id"].(string));err!=nil{t.Fatal(err)}
 done,_:=os.ReadFile(path);if string(done)!=string(old){t.Fatal("aprovacao nao restaurou")}
}
