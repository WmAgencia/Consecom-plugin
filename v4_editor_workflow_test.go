package main

import(
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func TestV4CatalogAndProjects(t *testing.T){
 names:=map[string]bool{}
 for _,it:=range tools(){names[it.Name]=true}
 for _,n:=range []string{"codigo_buscar","codigo_ler_trecho","codigo_substituir_trecho","desktop_controles","desktop_controle_clicar","programas_criativos","video_inspecionar","fluxo_iniciar","fluxo_status","fluxo_cancelar"} {
  if !names[n]{t.Errorf("ferramenta ausente %s",n)}
 }
 if len(names)!=72{t.Fatalf("esperadas 72 ferramentas, recebidas %d",len(names))}
 a,e:=NewApp(filepath.Join(t.TempDir(),"config.json"));if e!=nil{t.Fatal(e)}
 if _,e=a.codeSearch("root-inexistente","uma busca");e==nil{t.Error("busca ignorou pasta autorizada")}
 if _,e=a.workflowRequest("root-inexistente",[]string{"git_status"});e==nil{t.Error("fluxo ignorou pasta autorizada")}
 if e=validateWorkflow([]string{"powershell"});e==nil{t.Error("comando shell arbitrario permitido")}
 if _,e=a.inspectVideo("root-inexistente","sample.mp4");e==nil{t.Error("video ignorou pasta autorizada")}
}
func TestV4EditorReadSearchAndConflict(t *testing.T){
 dir:=t.TempDir()
 if e:=os.WriteFile(filepath.Join(dir,"main.go"),[]byte("package main\nfunc hello(){println(\"original\")}\n"),0600);e!=nil{t.Fatal(e)}
 a,e:=NewApp(filepath.Join(t.TempDir(),"config.json"));if e!=nil{t.Fatal(e)}
 a.cfg.Roots=[]Root{{ID:"test-root",Name:"Projeto V4",Path:dir,Writable:true}}
 preview,e:=a.codeSnippet("test-root","main.go",1,3);if e!=nil{t.Fatal(e)}
 meta:=preview.(map[string]any);hash:=meta["sha256"].(string)
 if !strings.Contains(meta["content"].(string),"2 | func hello()"){t.Fatal("leitura por linha incorreta")}
 found,e:=a.codeSearch("test-root","original");if e!=nil{t.Fatal(e)}
 if len(found.(map[string]any)["matches"].([]map[string]any))!=1{t.Fatal("busca deveria encontrar exatamente uma linha")}
 req:=EditorEdit{RootID:"test-root",Path:"main.go",Old:"original",New:"alterado",ExpectedSHA:hash}
 result,e:=a.codeEditRequest(req);if e!=nil{t.Fatal(e)}
 if result.(map[string]any)["status"]!="aguardando_aprovacao_local"{t.Fatal("modo normal deveria pedir aprovacao")}
 b,_:=os.ReadFile(filepath.Join(dir,"main.go"));if strings.Contains(string(b),"alterado"){t.Fatal("arquivo mudou antes da aprovacao")}
 for _,p:=range a.pendings(){if _,e=a.pop(p["id"].(string));e!=nil{t.Fatal(e)}}
 if e=a.setMode("maximum","LIBERAR");e!=nil{t.Fatal(e)}
 result,e=a.codeEditRequest(req);if e!=nil{t.Fatal(e)}
 if result.(map[string]any)["status"]!="executada"{t.Fatal("modo maximo deveria editar direto")}
 b,_=os.ReadFile(filepath.Join(dir,"main.go"));if !strings.Contains(string(b),"alterado"){t.Fatal("edicao nao foi gravada")}
 if _,e=a.codeEditRequest(req);e==nil{t.Fatal("SHA256 antigo aceito")}
 if _,e=a.codeSnippet("test-root","../outro.go",1,5);e==nil{t.Fatal("path traversal permitido")}
 if len(a.pendings())!=0{t.Fatal("modo maximo criou aprovacao pendente")}
}
func TestV4WorkflowValidationAndCreativeList(t *testing.T){
 if e:=validateWorkflow([]string{"git_status","go_test"});e!=nil{t.Fatal(e)}
 if e:=validateWorkflow([]string{});e==nil{t.Fatal("fluxo vazio aceito")}
 if e:=validateWorkflow([]string{"git_status","git_status","git_status","git_status","git_status","git_status","git_status"});e==nil{t.Fatal("limite excedido")}
 val,e:=creativeTools();if e!=nil{t.Fatal(e)}
 toolsList:=val.(map[string]any)["programs"].([]CreativeProgram)
 if len(toolsList)<5{t.Fatalf("lista criativa incompleta: %d",len(toolsList))}
}
