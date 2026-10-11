package main

import (
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "testing"
)

func TestV4GoFormatterApprovalAndBackup(t *testing.T){
 dir:=t.TempDir()
 original:="package main\nfunc main(){ println(\"ok\") }\n"
 if err:=os.WriteFile(filepath.Join(dir,"main.go"),[]byte(original),0600);err!=nil{t.Fatal(err)}
 app,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 app.cfg.Roots=[]Root{{ID:"root",Name:"Go Test",Path:dir,Writable:true}}
 spec:=FormatGoSpec{RootID:"root",Path:"main.go",ExpectedSHA:codeSHA([]byte(original))}
 queued,err:=app.codeFormatGoRequest(spec);if err!=nil{t.Fatal(err)}
 if queued.(map[string]any)["status"]!="aguardando_aprovacao_local"{t.Fatal("no approval queued")}
 before,_:=os.ReadFile(filepath.Join(dir,"main.go"));if string(before)!=original{t.Fatal("modified before approval")}
 approvals:=app.pendings();if len(approvals)!=1{t.Fatal("missing pending approval")}
 result,err:=app.approve(approvals[0]["id"].(string));if err!=nil{t.Fatal(err)}
 backup:=regexp.MustCompile(`backup: ([0-9a-f]{32})`).FindStringSubmatch(result)
 if len(backup)!=2{t.Fatalf("backup missing from result: %s",result)}
 after,_:=os.ReadFile(filepath.Join(dir,"main.go"))
 if string(after)==original||!strings.Contains(string(after),"func main() {"){t.Fatal("gofmt did not change output")}
 rec,b,err:=loadEditorBackup(app,backup[1]);if err!=nil{t.Fatal(err)}
 if string(b)!=original||rec.AfterSHA!=codeSHA(after){t.Fatal("backup or output hash mismatch")}
 if _,err=app.codeFormatGoRequest(spec);err==nil{t.Fatal("stale SHA accepted")}
 again,err:=app.codeFormatGoRequest(FormatGoSpec{RootID:"root",Path:"main.go",ExpectedSHA:codeSHA(after)})
 if err!=nil{t.Fatal(err)}
 if again.(map[string]any)["status"]!="sem_alteracoes"{t.Fatal("gofmt not idempotent")}
}

func TestV4GoFormatterRejectsInvalidInputs(t *testing.T){
 dir:=t.TempDir()
 app,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 app.cfg.Roots=[]Root{{ID:"root",Name:"Test",Path:dir,Writable:true}}
 if _,err=app.codeFormatGoRequest(FormatGoSpec{RootID:"missing",Path:"x.go",ExpectedSHA:"bad"});err==nil{t.Fatal("unknown root allowed")}
 if err=os.WriteFile(filepath.Join(dir,"bad.go"),[]byte("package main\nfunc main(\n"),0600);err!=nil{t.Fatal(err)}
 if _,err=app.codeFormatGoRequest(FormatGoSpec{RootID:"root",Path:"bad.go",ExpectedSHA:codeSHA([]byte("package main\nfunc main(\n"))});err==nil{t.Fatal("invalid syntax allowed")}
 if err=os.WriteFile(filepath.Join(dir,"x.txt"),[]byte("hello"),0600);err!=nil{t.Fatal(err)}
 if _,err=app.codeFormatGoRequest(FormatGoSpec{RootID:"root",Path:"x.txt",ExpectedSHA:codeSHA([]byte("hello"))});err==nil{t.Fatal("non Go file allowed")}
}
