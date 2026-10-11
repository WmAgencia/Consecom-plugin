//go:build windows

package main

import (
 "context"
 "encoding/base64"
 "encoding/json"
 "errors"
 "fmt"
 "os/exec"
 "strconv"
 "strings"
 "syscall"
 "time"
 "unicode/utf16"
)

// UI Automation is provided by the Windows accessibility layer. No third party
// binary, arbitrary user-supplied script or shell argument is executed.
type UIAElement struct{
 Name string `json:"name"`
 AutomationID string `json:"automation_id"`
 ControlType string `json:"control_type"`
 Enabled bool `json:"enabled"`
 X float64 `json:"x"`
 Y float64 `json:"y"`
 Width float64 `json:"width"`
 Height float64 `json:"height"`
}

const uiaEnumerateScript = `
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$parent=[System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]::new([int64]#HANDLE#))
if ($null -eq $parent) {throw 'Janela sem raiz UIA'}
$all=$parent.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)
$out=[System.Collections.Generic.List[object]]::new()
for($i=0;$i -lt $all.Count -and $out.Count -lt 90;$i++){
 try {
  $element=$all.Item($i)
  $props=$element.Current
  if ($props.IsPassword -or $props.IsOffscreen){continue}
  $r=$props.BoundingRectangle
  if ($r.Width -le 2 -or $r.Height -le 2){continue}
  $name=[string]$props.Name
  $id=[string]$props.AutomationId
  $ct=[string]$props.ControlType.ProgrammaticName
  if ($name.Length -gt 180){$name=$name.Substring(0,180)}
  if ($id.Length -gt 180){$id=$id.Substring(0,180)}
  if ($name.Length -eq 0 -and $id.Length -eq 0){continue}
  $out.Add([pscustomobject]@{
    name=$name;automation_id=$id;control_type=$ct
    enabled=[bool]$props.IsEnabled
    x=[math]::Round($r.X,1);y=[math]::Round($r.Y,1)
    width=[math]::Round($r.Width,1);height=[math]::Round($r.Height,1)
  })
 }catch{continue}
}
ConvertTo-Json -InputObject @($out.ToArray()) -Depth 3 -Compress
`

const uiaActionScript = `
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$expectedName=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('#NAME#'))
$expectedID=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('#AUTOMATION#'))
$expectedType=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('#TYPE#'))
$parent=[System.Windows.Automation.AutomationElement]::FromHandle([IntPtr]::new([int64]#HANDLE#))
if ($null -eq $parent){throw 'Janela invalida'}
$all=$parent.FindAll([System.Windows.Automation.TreeScope]::Descendants,[System.Windows.Automation.Condition]::TrueCondition)
$matches=[System.Collections.Generic.List[object]]::new()
for($i=0;$i -lt $all.Count -and $i -lt 1000;$i++){
 try{
  $e=$all.Item($i);$p=$e.Current
  if ($p.IsPassword -or $p.IsOffscreen -or -not $p.IsEnabled){continue}
  if ([string]$p.Name -ne $expectedName){continue}
  if ([string]$p.AutomationId -ne $expectedID){continue}
  if ([string]$p.ControlType.ProgrammaticName -ne $expectedType){continue}
  $matches.Add($e)
 }catch{continue}
}
if($matches.Count -ne 1){throw "Controle UIA ambiguo ou ausente ($($matches.Count))"}
$target=$matches[0]
if('#ACTION#' -eq 'focus'){
 $target.SetFocus()
}else{
 $pattern=$null
 if(-not $target.TryGetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern,[ref]$pattern)){throw 'Controle nao oferece InvokePattern'}
 $pattern.Invoke()
}
Write-Output 'OK'
`

func uiaPowerShell(ctx context.Context, script string)([]byte,error){
 encoded:=utf16.Encode([]rune(script))
 raw:=make([]byte,len(encoded)*2)
 for i,v:=range encoded{raw[2*i]=byte(v);raw[2*i+1]=byte(v>>8)}
 cmd:=exec.CommandContext(ctx,"powershell.exe","-NoLogo","-NoProfile","-NonInteractive","-EncodedCommand",base64.StdEncoding.EncodeToString(raw))
 // PowerShell executes only the fixed script built above; never interpolates
 // untrusted PowerShell source or lets the caller supply shell flags.
 out,err:=cmd.CombinedOutput()
 if len(out)>64000{return nil,errors.New("resposta UIA excede 64 KiB")}
 if err!=nil {
  if ctx.Err()!=nil{return nil,fmt.Errorf("UIA atingiu limite de tempo: %w",ctx.Err())}
  return nil,fmt.Errorf("Windows UI Automation: %s (%v)",clip(strings.TrimSpace(string(out)),600),err)
 }
 return out,nil
}

func desktopUIAElements(parent uint64)(any,error){
 if _,_,err:=desktopWindow(parent);err!=nil{return nil,err}
 ctx,cancel:=context.WithTimeout(context.Background(),12*time.Second);defer cancel()
 script:=strings.ReplaceAll(uiaEnumerateScript,"#HANDLE#",strconv.FormatUint(parent,10))
 out,err:=uiaPowerShell(ctx,script);if err!=nil{return nil,err}
 var elements []UIAElement
 if err=json.Unmarshal(out,&elements);err!=nil{return nil,fmt.Errorf("saida UIA invalida: %w",err)}
 if elements==nil{elements=[]UIAElement{}}
 return map[string]any{"window":parent,"elements":elements,"count":len(elements),"note":"Elementos da arvore UIA; campos de senha e elementos fora da tela sao omitidos"},nil
}

func desktopUIAAction(parent uint64,el UIAElement,action string)(string,error){
 if action!="focus"&&action!="invoke"{return "",errors.New("acao UIA nao permitida")}
 if el.Name==""&&el.AutomationID==""{return "",errors.New("identificador UIA vazio")}
 if len(el.Name)>180||len(el.AutomationID)>180||len(el.ControlType)>120||el.ControlType==""{return "",errors.New("seletor UIA invalido")}
 if _,_,err:=desktopWindow(parent);err!=nil{return "",err}
 if el.ControlType=="ControlType.Edit"&&action=="invoke"{return "",errors.New("campo Edit nao oferece invocacao; use foco")}
 script:=strings.NewReplacer(
  "#HANDLE#",strconv.FormatUint(parent,10),
  "#NAME#",base64.StdEncoding.EncodeToString([]byte(el.Name)),
  "#AUTOMATION#",base64.StdEncoding.EncodeToString([]byte(el.AutomationID)),
  "#TYPE#",base64.StdEncoding.EncodeToString([]byte(el.ControlType)),
  "#ACTION#",action,
 ).Replace(uiaActionScript)
 ctx,cancel:=context.WithTimeout(context.Background(),12*time.Second);defer cancel()
 _,err:=uiaPowerShell(ctx,script)
 if err!=nil{return "",err}
 return "UIA "+action+" concluido e confirmado pelo Windows",nil
}

var _=syscall.Errno(0)
