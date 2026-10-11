package main

import (
 "path/filepath"
 "strings"
 "testing"
)

func TestV3ToolCatalog(t *testing.T){
 required:=[]string{"dev_job_iniciar","dev_job_status","dev_job_cancelar","git_patch_validar","git_patch_aplicar","git_patch_reverter","git_branch_criar","desktop_janelas","desktop_abrir","desktop_ativar","desktop_clicar","desktop_digitar","desktop_capturar","desktop_ultima_captura","browser_habilitar_downloads","browser_downloads","browser_upload_arquivo","browser_confirmar_acao","browser_download_copiar"}
 found:=map[string]bool{}
 for _,x:=range tools(){found[x.Name]=true}
 if len(tools())!=78{t.Fatalf("78 ferramentas esperadas, recebeu %d",len(tools()))}
 for _,name:=range required{if !found[name]{t.Errorf("ferramenta ausente %s",name)}}
 if found["tarefas_listar"]||found["tarefas_responder"]{t.Error("ferramentas de conversa antigas retornaram")}
}
func TestV3LocalApprovalAndInvalidActions(t *testing.T){
 a,e:=NewApp(filepath.Join(t.TempDir(),"config.json"));if e!=nil{t.Fatal(e)}
 if _,e=a.devJobStart("sem_acesso","npm_test",false);e==nil{t.Fatal("terminal aceitou projeto sem acesso")}
 if _,e=a.devJobStart("sem_acesso","powershell",false);e==nil{t.Fatal("terminal aceitou comando arbitrario")}
 if _,e=a.desktopRequest("open",DesktopAction{App:"cmd"});e==nil{t.Fatal("desktop aceitou aplicativo fora da lista")}
 result,e:=a.desktopRequest("open",DesktopAction{App:"notepad"});if e!=nil{t.Fatal(e)}
 queued,ok:=result.(map[string]any)
 if !ok||queued["status"]!="aguardando_aprovacao_local"{t.Fatalf("acao desktop executou sem aprovacao: %#v",result)}
 if len(a.pendings())!=1{t.Fatal("a aprovacao nao apareceu no painel")}
 if _,e=a.desktopRequest("type",DesktopAction{Window:1,Text:"seguro\nnao"});e==nil{t.Fatal("digitacao com linha adicional foi permitida")}
 if _,e=a.browserUploadRequest("sem_acesso","x.txt","input[type=file]");e==nil{t.Fatal("upload aceitou pasta sem permissao")}
 if _,e=a.gitBranchRequest("sem_acesso","main");e==nil{t.Fatal("Git aceitou projeto sem permissao")}
}
func TestV3PatchRejectsTraversal(t *testing.T){
 root:=Root{Name:"test",Path:t.TempDir(),Writable:true}
 bad:="diff --git a/../secret.txt b/../secret.txt\n--- a/../secret.txt\n+++ b/../secret.txt\n@@ -1 +1 @@\n-old\n+new\n"
 if e:=validateProjectPatch(root,bad);e==nil{t.Fatal("path traversal aceito em patch")}
 if e:=validateProjectPatch(root,"not a patch");e==nil{t.Fatal("arquivo arbitrario aceito como patch")}
 if !strings.HasPrefix(bad,"diff --git"){t.Fatal("fixture invalida")}
}
