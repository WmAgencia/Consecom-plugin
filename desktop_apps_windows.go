//go:build windows

package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "sort"
 "sync"
 "strings"
 "time"
)

type InstalledWindowsApp struct {
 Name string `json:"name"`
 AppID string `json:"app_id"`
}

var startAppIDPattern = regexp.MustCompile(`^[A-Za-z0-9._!{}+\-]{1,260}$`)

// Get-StartApps reads the Windows Start-menu app registry, including packaged
// applications (UWP/MSIX) and registered classic desktop applications.
// The script is fixed: no user query, app name or path becomes PowerShell code.
const startAppsScript = `
$ErrorActionPreference='Stop'
$ProgressPreference='SilentlyContinue'
$WarningPreference='SilentlyContinue'
$apps=@(Get-StartApps | Select-Object -First 600)
$out=@(foreach($app in $apps){
 [pscustomobject]@{name=[string]$app.Name;app_id=[string]$app.AppID}
})
Write-Output ('CONSECOM_STARTAPPS_JSON='+ (ConvertTo-Json -InputObject $out -Compress -Depth 3))
`

var installedAppsCache struct {
 sync.Mutex
 Until time.Time
 Apps []InstalledWindowsApp
}
func enumerateWindowsApps() ([]InstalledWindowsApp,error) {
 installedAppsCache.Lock()
 if time.Now().Before(installedAppsCache.Until){v:=append([]InstalledWindowsApp(nil),installedAppsCache.Apps...);installedAppsCache.Unlock();return v,nil}
 installedAppsCache.Unlock()
 ctx,cancel:=context.WithTimeout(context.Background(),18*time.Second)
 defer cancel()
 out,err:=uiaPowerShell(ctx,startAppsScript)
 if err!=nil{return nil,fmt.Errorf("nao foi possivel consultar o menu Iniciar do Windows: %w",err)}
 jsonLine:=""
 for _,line:=range strings.Split(strings.ReplaceAll(string(out),"\r\n","\n"),"\n") {
  line=strings.TrimSpace(line)
  if strings.HasPrefix(line,"CONSECOM_STARTAPPS_JSON="){
   jsonLine=strings.TrimPrefix(line,"CONSECOM_STARTAPPS_JSON=")
   break
  }
 }
 if jsonLine==""{return nil,fmt.Errorf("a consulta ao menu Iniciar nao retornou JSON valido: %s",clip(strings.TrimSpace(string(out)),220))}
 var apps []InstalledWindowsApp
 if err=json.Unmarshal([]byte(jsonLine),&apps);err!=nil{return nil,fmt.Errorf("JSON do menu Iniciar invalido: %w",err)}
 unique:=map[string]bool{}
 clean:=make([]InstalledWindowsApp,0,len(apps))
 for _,app:=range apps{
  app.Name=strings.TrimSpace(app.Name)
  if len(app.Name)==0||len(app.Name)>200||!startAppIDPattern.MatchString(app.AppID){continue}
  id:=strings.ToLower(app.AppID)
  if unique[id]{continue}
  unique[id]=true
  clean=append(clean,app)
 }
 sort.Slice(clean,func(i,j int)bool{return strings.ToLower(clean[i].Name)<strings.ToLower(clean[j].Name)})
 installedAppsCache.Lock()
 installedAppsCache.Apps=append([]InstalledWindowsApp(nil),clean...)
 installedAppsCache.Until=time.Now().Add(3*time.Minute)
 installedAppsCache.Unlock()
 return clean,nil
}

// Search is bounded and read-only, and returns exact IDs for a follow-up
// launch request; fuzzy matching is never used to execute an ambiguous app.
func windowsAppsSearch(query string)(any,error) {
 query=strings.TrimSpace(query)
 if len(query)>100{return nil,errors.New("pesquisa com mais de 100 caracteres")}
 apps,err:=enumerateWindowsApps();if err!=nil{return nil,err}
 match:=make([]InstalledWindowsApp,0)
 for _,app:=range apps {
  if query!=""&&!strings.Contains(strings.ToLower(app.Name),strings.ToLower(query))&&
   !strings.Contains(strings.ToLower(app.AppID),strings.ToLower(query)){continue}
  match=append(match,app)
 }
 total:=len(match)
 if len(match)>120{match=match[:120]}
 return map[string]any{
  "programas":match, "total_encontrados":total, "exibidos":len(match),
  "fonte":"Menu Iniciar do Windows (Get-StartApps)",
  "dica":"Use o app_id exato em desktop_abrir_instalado; nao sao aceitos argumentos nem comandos.",
 },nil
}

func validRegisteredApp(id string) (InstalledWindowsApp,error) {
 if !startAppIDPattern.MatchString(id){return InstalledWindowsApp{},errors.New("identificador de aplicativo invalido")}
 apps,err:=enumerateWindowsApps();if err!=nil{return InstalledWindowsApp{},err}
 for _,app:=range apps{
  if strings.EqualFold(app.AppID,id){return app,nil}
 }
 return InstalledWindowsApp{},errors.New("aplicativo nao encontrado no menu Iniciar; consulte desktop_aplicativos")
}

func launchRegisteredWindowsApp(id string)(string,error){
 app,err:=validRegisteredApp(id);if err!=nil{return "",err}
 windir:=os.Getenv("WINDIR")
 if windir==""{return "",errors.New("diretorio do Windows indisponivel")}
 explorer:=filepath.Join(windir,"explorer.exe")
 if st,err:=os.Stat(explorer);err!=nil||!st.Mode().IsRegular(){return "",errors.New("Explorer oficial do Windows indisponivel")}
 // A single Shell Namespace argument, never a cmd.exe/PowerShell command.
 // No arbitrary executable, URL, arguments, protocol or elevated verb is used.
 cmd:=exec.Command(explorer,`shell:AppsFolder\`+app.AppID)
 if err=cmd.Start();err!=nil{return "",err}
 _=cmd.Process.Release()
 return "Solicitacao de abertura enviada ao Windows para "+app.Name,nil
}

func validatePortableWindowsExe(path string)(string,error){
 if len(path)>1024||!filepath.IsAbs(path)||strings.HasPrefix(path,`\\`){
  return "",errors.New("informe caminho absoluto de executavel local, sem rede")
 }
 clean:=filepath.Clean(path)
 if !strings.EqualFold(filepath.Ext(clean),".exe"){
  return "",errors.New("somente programas .exe; nao executa .bat, .cmd, .ps1, atalhos ou scripts")
 }
 // System executables, credential directories and Consecom internals must not
 // become a generic command-execution tunnel.
 for _,root:=range []string{os.Getenv("WINDIR"),os.Getenv("SYSTEMROOT"),os.Getenv("PROGRAMDATA")}{
  if root!=""&&isWithinDirectory(clean,root){return "",errors.New("executaveis de area do sistema nao podem ser abertos por caminho; use um aplicativo registrado")}
 }
 if local:=os.Getenv("LOCALAPPDATA");local!=""&&isWithinDirectory(clean,filepath.Join(local,"ConsecomMCP")){
  return "",errors.New("area privada do Consecom protegida")
 }
 if err:=ensureNoLinks(clean);err!=nil{return "",err}
 info,err:=os.Lstat(clean);if err!=nil{return "",err}
 if !info.Mode().IsRegular()||info.Size()==0||info.Size()>2<<30{
  return "",errors.New("executavel irregular, vazio ou acima do limite")
 }
 return clean,nil
}

func isWithinDirectory(path,root string)bool{
 path=filepath.Clean(path);root=filepath.Clean(root)
 rel,err:=filepath.Rel(root,path)
 return err==nil&&(rel=="."||rel!=".."&&!strings.HasPrefix(rel,".."+string(os.PathSeparator)))
}

func launchPortableWindowsExe(path string)(string,error){
 path,err:=validatePortableWindowsExe(path);if err!=nil{return "",err}
 cmd:=exec.Command(path)
 cmd.Dir=filepath.Dir(path)
 if err=cmd.Start();err!=nil{return "",err}
 _=cmd.Process.Release()
 return "Executavel iniciado sem parametros: "+filepath.Base(path),nil
}
