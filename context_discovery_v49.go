package main

import (
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "sync"
 "time"
)

// Only official-looking, fixed installation locations may be executed.
// Project folders on C: / D: are detected as *context*, never executed.
type contextCLIInfo struct { Binary string; Prefix []string }
func contextFindFile(paths ...string) string {
 for _,p:=range paths { if p=="" {continue}; if st,e:=os.Stat(p);e==nil&&st.Mode().IsRegular(){return p} }
 return ""
}
func contextCLI(command string)(contextCLIInfo,bool) {
 if runtime.GOOS!="windows" {p,e:=exec.LookPath(command);return contextCLIInfo{Binary:p},e==nil}
 home,_:=os.UserHomeDir()
 local,appdata:=os.Getenv("LOCALAPPDATA"),os.Getenv("APPDATA")
 paths:=[]string{}
 if p,e:=exec.LookPath(command+".exe");e==nil{paths=append(paths,p)}
 switch command{
 case "claude":
  paths=append(paths,filepath.Join(home,".local","bin","claude.exe"),
   filepath.Join(local,"Microsoft","WinGet","Links","claude.exe"),
   filepath.Join(local,"Programs","claude","claude.exe"))
 case "codex":
  paths=append(paths,filepath.Join(home,".local","bin","codex.exe"),
   filepath.Join(local,"Microsoft","WinGet","Links","codex.exe"))
 case "opencode":
  paths=append(paths,filepath.Join(home,".opencode","bin","opencode.exe"),
   filepath.Join(local,"Microsoft","WinGet","Links","opencode.exe"),
   filepath.Join(local,"Programs","opencode","opencode.exe"))
 }
 if binary:=contextFindFile(paths...);binary!=""{return contextCLIInfo{Binary:binary},true}
 node,e:=exec.LookPath("node.exe")
 if e!=nil { node=contextFindFile(filepath.Join(os.Getenv("ProgramFiles"),"nodejs","node.exe")) }
 if node==""{return contextCLIInfo{},false}
 var relative string
 switch command {
 case "claude": relative=filepath.Join("@anthropic-ai","claude-code","cli.js")
 case "codex": relative=filepath.Join("@openai","codex","bin","codex.js")
 case "opencode": relative=filepath.Join("opencode-ai","bin","opencode.js")
 default: return contextCLIInfo{},false
 }
 bases:=[]string{filepath.Join(appdata,"npm","node_modules"),filepath.Join(home,".npm-global","lib","node_modules"),
  filepath.Join(home,"AppData","Roaming","npm","node_modules"), filepath.Join(os.Getenv("ProgramFiles"),"nodejs","node_modules"),
  filepath.Join("C:\\","node_modules"),filepath.Join("D:\\","node_modules")}
 for _,base:=range bases{
  if script:=contextFindFile(filepath.Join(base,relative));script!="" {
   return contextCLIInfo{Binary:node,Prefix:[]string{script}},true
  }
 }
 return contextCLIInfo{},false
}

// Disk discovery reads bounded directory *names only*, without reading
// repositories, source code, credentials, file contents or entire disks.
type contextClue struct { ID string; Location string }
var contextClueCache struct{
 sync.Mutex
 Until time.Time
 Values map[string]string
}
func contextWorkspaceClues() map[string]string {
 contextClueCache.Lock()
 if time.Now().Before(contextClueCache.Until){
  out:=map[string]string{};for k,v:=range contextClueCache.Values{out[k]=v}
  contextClueCache.Unlock();return out
 }
 contextClueCache.Unlock()
 result:=map[string]string{}
 deadline:=time.Now().Add(1800*time.Millisecond)
 count:=0
 skip:=map[string]bool{"windows":true,"program files":true,"program files (x86)":true,
  "programdata":true,"system volume information":true,"$recycle.bin":true,
  "node_modules":true,".git":true,"appdata":true,".cache":true,".next":true,
  ".venv":true,"dist":true,"build":true}
 type entry struct{path string;level int}
 queue:=[]entry{}
 for _,root:=range []string{os.Getenv("USERPROFILE"),"C:\\","D:\\",
  "C:\\Projetos","D:\\Projetos","C:\\Projects","D:\\Projects",
  "C:\\Repos","D:\\Repos","C:\\GitHub","D:\\GitHub",
  "D:\\Desenvolvimento","D:\\Dev","C:\\Dev"}{
  if root!="" {queue=append(queue,entry{root,0})}
 }
 seen:=map[string]bool{}
 for len(queue)>0 && count<1800 && time.Now().Before(deadline) {
  cur:=queue[0];queue=queue[1:]
  key:=strings.ToLower(filepath.Clean(cur.path))
  if seen[key]{continue};seen[key]=true
  dirs,err:=os.ReadDir(cur.path);if err!=nil{continue}
  for _,d:=range dirs {
   if count>=1800||time.Now().After(deadline){break}
   if !d.IsDir() || d.Type()&os.ModeSymlink!=0{continue}
   count++
   name:=strings.ToLower(d.Name())
   normalized:=strings.NewReplacer("-","","_",""," ","","." ,"").Replace(name)
   id:=""
   switch {
   case strings.Contains(normalized,"claudecode")||strings.Contains(normalized,"cloudcode"):id="claude-code"
   case strings.Contains(normalized,"opencode"):id="opencode"
   case strings.Contains(normalized,"codex"):id="codex"
   }
   if id!=""&&result[id]==""{result[id]=filepath.Join(cur.path,d.Name())}
   if cur.level<2 && !skip[name] && !strings.HasPrefix(name,".") {
    queue=append(queue,entry{filepath.Join(cur.path,d.Name()),cur.level+1})
   }
  }
 }
 contextClueCache.Lock();contextClueCache.Values=result;contextClueCache.Until=time.Now().Add(3*time.Minute);contextClueCache.Unlock()
 return result
}

func (a *App) contextVaultAuto()(any,error){
 vaults,err:=obsidianVaults();if err!=nil{return nil,err}
 if len(vaults)==0{return nil,os.ErrNotExist}
 a.mu.Lock();selected:=a.cfg.ContextVault;a.mu.Unlock()
 // Prefer the already-connected vault, then the sole open vault, then a
 // deterministic first existing vault. Never overwrite personal notes.
 for _,v:=range vaults {if v["path"]==selected{return a.contextVaultSelect(v["id"].(string))}}
 open:=[]map[string]any{}
 for _,v:=range vaults {if v["open"]==true{open=append(open,v)}}
 chosen:=vaults[0]
 if len(open)==1{chosen=open[0]}
 return a.contextVaultSelect(chosen["id"].(string))
}
