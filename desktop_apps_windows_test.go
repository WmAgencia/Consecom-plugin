//go:build windows

package main

import (
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestDesktopLauncherRejectsCommandsAndUnregisteredIDs(t *testing.T){
 for _,id:=range []string{"", "cmd /c whoami", "../notepad", "bad\\id", "https://example.test", "a; b", strings.Repeat("a",261)}{
  if startAppIDPattern.MatchString(id){t.Fatalf("unsafe app ID accepted: %q",id)}
 }
 if !startAppIDPattern.MatchString("Microsoft.WindowsAlarms_8wekyb3d8bbwe!App"){
  t.Fatal("Windows Clock's expected AUMID format was rejected")
 }
}

func TestPortableExecutableIsLocalArgumentFreeAndProtected(t *testing.T){
 dir:=t.TempDir()
 good:=filepath.Join(dir,"test-app.exe")
 if err:=os.WriteFile(good,[]byte("fake exe, never launched"),0600);err!=nil{t.Fatal(err)}
 if got,err:=validatePortableWindowsExe(good);err!=nil||got!=filepath.Clean(good){t.Fatalf("safe local file rejected: %q %v",got,err)}
 for _,bad:=range []string{"", "cmd /c calc", `\\\\server\\share\\evil.exe`, filepath.Join(dir,"script.cmd"), filepath.Join(dir,"nothing.exe")}{
  if _,err:=validatePortableWindowsExe(bad);err==nil{t.Errorf("unsafe or missing executable accepted: %q",bad)}
 }
 if windir:=os.Getenv("WINDIR");windir!=""{
  if _,err:=validatePortableWindowsExe(filepath.Join(windir,"System32","cmd.exe"));err==nil{
   t.Fatal("protected Windows executable path accepted")
  }
 }
 if _,err:=validatePortableWindowsExe(filepath.Join(dir,"data.ps1"));err==nil{t.Fatal("PowerShell scripts allowed")}
}

func TestLauncherToolsAndMaximumApprovalValidation(t *testing.T){
 expected:=map[string]bool{"desktop_aplicativos":false,"desktop_abrir_instalado":true,"desktop_abrir_executavel":true}
 for _,tool:=range tools(){
  requireMutating,ok:=expected[tool.Name];if !ok{continue}
  if tool.Annotations["readOnlyHint"]!=!requireMutating{t.Fatalf("wrong readOnlyHint for %s",tool.Name)}
  delete(expected,tool.Name)
 }
 if len(expected)!=0{t.Fatalf("missing launcher tools: %v",expected)}
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 if err=a.setMode("maximum","LIBERAR");err!=nil{t.Fatal(err)}
 requests:=[]struct{name string; args map[string]any}{
  {"desktop_abrir_instalado",map[string]any{"app_id":"bad;id"}},
  {"desktop_abrir_executavel",map[string]any{"caminho_absoluto":"not-a-path"}},
 }
 for _,test:=range requests{
  b,_:=json.Marshal(test.args)
  if _,err=a.dispatch(test.name,b);err==nil{t.Errorf("unsafe request accepted: %s",test.name)}
 }
 if len(a.pendings())!=0{t.Fatal("invalid launches created approval requests in maximum mode")}
}

func TestWindowsStartAppDiscoveryReadOnly(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 apps,err:=windowsAppsSearch("relogio")
 if err!=nil{t.Logf("Start menu unavailable in headless CI: %v",err);return}
 response:=apps.(map[string]any)
 if response["total_encontrados"].(int)<0{t.Fatal("negative result count")}
 if len(a.pendings())!=0{t.Fatal("read-only search queued approvals")}
}
