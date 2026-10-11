package main

import (
 "errors"
 "os"
 "path/filepath"
 "testing"
 "time"
)

func TestV4WorkflowHistoryPersistsAcrossRestart(t *testing.T){
 base:=t.TempDir()
 config:=filepath.Join(base,"config.json")
 a,err:=NewApp(config);if err!=nil{t.Fatal(err)}
 job:=WorkflowTask{
  ID:newID(), RootID:"project", Steps:[]string{"git_status","go_test"},
  Current:1,Status:"running",Started:time.Now().UTC(),
  Logs:[]string{"1/2 Iniciando git_status"},
 }
 if err=a.saveWorkflowSnapshot(job);err!=nil{t.Fatal(err)}
 restarted,err:=NewApp(config);if err!=nil{t.Fatal(err)}
 loaded,err:=restarted.workflowStatus(job.ID);if err!=nil{t.Fatal(err)}
 saved:=loaded.(WorkflowTask)
 if saved.Status!="interrupted"{t.Fatalf("reinicio sem marcar interrupcao: %s",saved.Status)}
 if saved.Current!=1||len(saved.Logs)!=2{t.Fatalf("checkpoint perdido: %#v",saved)}
 listing,err:=restarted.workflowListHistory();if err!=nil{t.Fatal(err)}
 rows:=listing.(map[string]any)["jobs"].([]WorkflowHistoryEntry)
 if len(rows)!=1||rows[0].Status!="interrupted"||rows[0].ID!=job.ID{t.Fatalf("historico inconsistente: %#v",rows)}
 if _,err=restarted.workflowStatus("../../other");err==nil{t.Fatal("path traversal permitido")}
}

func TestV4WorkflowHistoryCompletedAndCorruptRecord(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 done:=WorkflowTask{ID:newID(),RootID:"dev",Status:"done",Current:2,Steps:[]string{"git_status","go_test"},Started:time.Now().Add(-time.Minute),Finished:time.Now(),Logs:[]string{"ok"}}
 if err=a.saveWorkflowSnapshot(done);err!=nil{t.Fatal(err)}
 loaded,err:=a.loadWorkflowSnapshot(done.ID);if err!=nil{t.Fatal(err)}
 if loaded.Status!="done"||len(loaded.Logs)!=1{t.Fatal("resultado persistido incorreto")}
 if err=os.WriteFile(filepath.Join(workflowHistoryDir(a),done.ID+".json"),[]byte("not json"),0600);err!=nil{t.Fatal(err)}
 if _,err=a.loadWorkflowSnapshot(done.ID);err==nil{t.Fatal("snapshot corrompido aceito")}
 if _,err=a.loadWorkflowSnapshot("00000000000000000000000000000000");!errors.Is(err,os.ErrNotExist){t.Fatalf("expected not-exist, got %v",err)}
}
