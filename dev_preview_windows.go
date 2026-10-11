//go:build windows

package main

import(
 "os"
 "os/exec"
 "path/filepath"
 "strconv"
)

// taskkill /T terminates the entire npm/cmd/node tree rather than orphaning
// the local dev server when the parent cmd.exe ends.
func stopPreviewProcessTree(proc *os.Process){
 if proc==nil{return}
 windir:=os.Getenv("WINDIR")
 if windir!=""{
  killer:=filepath.Join(windir,"System32","taskkill.exe")
  if err:=exec.Command(killer,"/PID",strconv.Itoa(proc.Pid),"/T","/F").Run();err==nil{return}
 }
 _=proc.Kill()
}
