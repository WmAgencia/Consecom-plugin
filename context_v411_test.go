package main

import (
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)
func TestContextV411IndexCountsAndHiddenDirectories(t *testing.T){
 vault:=t.TempDir()
 for _,p:=range []string{"Projetos","Projetos/Subpasta",".obsidian",".git"}{
  if err:=os.MkdirAll(filepath.Join(vault,p),0700);err!=nil{t.Fatal(err)}
 }
 for _,p:=range []string{"Projetos/a.md","Projetos/Subpasta/b.md","Raiz.md",".obsidian/confidencial.md",".git/test.md","Projetos/notes.txt"}{
  if err:=os.WriteFile(filepath.Join(vault,p),[]byte("conteudo que nao deve ser indexado"),0600);err!=nil{t.Fatal(err)}
 }
 snap:=contextIndexCacheGet(vault)
 if snap.Status!="indexing"{t.Fatalf("unexpected initial status %q",snap.Status)}
 var done contextIndexSnapshot
 for i:=0;i<60;i++{
  time.Sleep(30*time.Millisecond)
  done=contextIndexCacheGet(vault)
  if done.Status=="ready"{break}
 }
 if done.Status!="ready"{t.Fatalf("index did not complete: %+v",done)}
 if done.Total!=3 || !done.Complete{t.Fatalf("expected 3 notes only: %+v",done)}
 if len(done.Files)!=3{t.Fatalf("expected 3 sample files, got %d",len(done.Files))}
 for _,f:=range done.Files{if strings.Contains(f.Path,".obsidian")||strings.Contains(f.Path,".git")||strings.Contains(f.Path,"confidencial"){t.Fatalf("hidden file indexed: %+v",f)}}
 found:=false
 for _,folder:=range done.Folders{if folder.Path=="Projetos"&&folder.Count==1{found=true}}
 if !found{t.Fatalf("expected project folder, got %+v",done.Folders)}
}
func TestV411UIApprovalSliderAndRealIndex(t *testing.T){
 for _,x:=range []string{
  "id=\"modeSlider\"","function commitModeSlider(",
  "id=\"contextIndexCount\"","function refreshContextIndex()",
  "contexto/indice","indexed-folder","indexed-file",
  "function contextIndexBadge(","exclusive-browser-choice",
 }{
  if !strings.Contains(uiHTML,x){t.Errorf("missing UI integration: %q",x)}
 }
 if strings.Contains(uiHTML,"id=\"modeAlways\" onclick=\"setMode('always')\"><span class=\"symbol\""){t.Fatal("large approval cards not removed")}
}
