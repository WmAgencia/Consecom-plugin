package main

import (
 "os"
 "os/exec"
 "path/filepath"
 "testing"
 "time"
)

func TestMaximumApprovalDesktopSkipsQueue(t *testing.T) {
 a,e:=NewApp(filepath.Join(t.TempDir(),"config.json"))
 if e!=nil{t.Fatal(e)}
 // Invalid window is harmless: the point is ensuring a real execution attempt, not a pending approval.
 if _,e=a.desktopRequest("activate",DesktopAction{Window:999999999});e!=nil{t.Fatal(e)}
 if len(a.pendings())!=1{t.Fatal("default mode should queue desktop action")}
 for _,p:=range a.pendings(){_,_ =a.pop(p["id"].(string))}
 if e=a.setMode("maximum","LIBERAR");e!=nil{t.Fatal(e)}
 if _,e=a.desktopRequest("activate",DesktopAction{Window:999999999});e==nil{
  t.Fatal("invalid window cannot be activated")
 }
 if n:=len(a.pendings());n!=0{t.Fatalf("maximum mode must not queue desktop actions: %d",n)}
 if _,e=a.desktopRequest("open",DesktopAction{App:"cmd"});e==nil{
  t.Fatal("maximum mode bypassed application allowlist")
 }
}

func TestMaximumApprovalGitAndDevJob(t *testing.T){
 if _,e:=exec.LookPath("git");e!=nil{t.Skip("git not installed")}
 dir:=t.TempDir()
 out,e:=exec.Command("git","-C",dir,"init","-q").CombinedOutput()
 if e!=nil{t.Fatalf("git init: %v %s",e,out)}
 a,e:=NewApp(filepath.Join(t.TempDir(),"config.json"));if e!=nil{t.Fatal(e)}
 a.cfg.Roots=[]Root{{ID:"safe-root",Name:"test",Path:dir,Writable:true}}
 result,e:=a.gitBranchRequest("safe-root","test-mode-maximum")
 if e!=nil{t.Fatal(e)}
 if result.(map[string]any)["status"]!="aguardando_aprovacao_local"{t.Fatalf("mode normal should queue: %#v",result)}
 for _,p:=range a.pendings(){_,_ =a.pop(p["id"].(string))}
 if e=a.setMode("maximum","LIBERAR");e!=nil{t.Fatal(e)}
 result,e=a.gitBranchRequest("safe-root","test-mode-maximum")
 if e!=nil{t.Fatalf("git branch in maximum mode: %v",e)}
 if result.(map[string]any)["status"]!="executada"{t.Fatalf("git branch was not executed: %#v",result)}
 if len(a.pendings())!=0{t.Fatal("git branch queued despite maximum")}
 job,e:=a.devJobStart("safe-root","git_status",false)
 if e!=nil{t.Fatal(e)}
 id,ok:=job.(map[string]any)["id"].(string)
 if !ok||id==""{t.Fatalf("job was queued: %#v",job)}
 deadline:=time.Now().Add(5*time.Second)
 for time.Now().Before(deadline) {
  detail,e:=a.devJobStatus(id);if e!=nil{t.Fatal(e)}
  if j:=detail.(DevJob);j.Status!="running"{
   if j.Status!="done"{t.Fatalf("git status failed: %#v",j)}
   break
  }
  time.Sleep(40*time.Millisecond)
 }
 if len(a.pendings())!=0{t.Fatal("maximum mode created a pending approval")}
 if _,e=a.devJobStart("safe-root","shell",false);e==nil{t.Fatal("arbitrary shell not blocked")}
 if _,e=a.gitBranchRequest("not-authorized","invalid-test");e==nil{t.Fatal("git escaped authorized project")}
 if _,e=os.Stat(filepath.Join(dir,".git"));e!=nil{t.Fatal(e)}
}
