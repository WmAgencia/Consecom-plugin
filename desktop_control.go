package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strconv"
 "strings"
 "sync"
 "time"
)
type DesktopAction struct {
 Window uint64 `json:"window"`
 X int `json:"x"`
 Y int `json:"y"`
 Text string `json:"text"`
 App string `json:"app"`
}
var captureMu sync.Mutex
var latestCapture string
var captureAt time.Time
func (a *App) desktopRequest(kind string, action DesktopAction)(any,error){
 switch kind {
 case "open":
  if action.App!="notepad"&&action.App!="calculator"&&action.App!="explorer"&&action.App!="vscode" {return nil,errors.New("aplicativo fora da lista")}
 case "activate","click","type","capture":
  if action.Window==0 {return nil,errors.New("identificador da janela obrigatorio")}
  if kind=="click"&&(action.X<0||action.Y<0||action.X>5000||action.Y>5000){return nil,errors.New("coordenadas invalidas")}
  if kind=="type"&&(len(action.Text)<1||len(action.Text)>1000||strings.IndexFunc(action.Text,func(r rune)bool{return r<32})>=0){return nil,errors.New("texto invalido")}
 default:return nil,errors.New("acao nao permitida")
 }
 data,_:=json.Marshal(action)
 desc:="Controle visual do Windows: "+kind
 if kind=="type" {desc+=fmt.Sprintf(" (%d caracteres; texto: %q)",len([]rune(action.Text)),clip(action.Text,180))}
 if kind=="open" {desc+=" "+action.App}
 if kind!="open" {desc+=fmt.Sprintf(" na janela %d",action.Window)}
 p:=&Pending{Kind:"desktop",Action:kind,Content:string(data),Description:desc}
 if a.mode()=="maximum" {
  result,e:=a.approveDesktop(p)
  if e!=nil{return nil,e}
  return map[string]any{"status":"executada","resultado":result},nil
 }
 p.Description+=" — exige aprovacao local"
 id,e:=a.queue(p)
 if e!=nil{return nil,e}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) approveDesktop(p *Pending)(string,error){
 var action DesktopAction
 if e:=json.Unmarshal([]byte(p.Content),&action);e!=nil{return "",e}
 if p.Action=="capture" {
  data,e:=desktopScreenshot(action.Window)
  if e!=nil{return "",e}
  folder:=filepath.Join(filepath.Dir(a.path),"captures")
  if e=os.MkdirAll(folder,0700);e!=nil{return "",e}
  path:=filepath.Join(folder,newID()+".jpg")
  if e=os.WriteFile(path,data,0600);e!=nil{return "",e}
  captureMu.Lock();old:=latestCapture;latestCapture=path;captureAt=time.Now();captureMu.Unlock()
  if old!=""&&old!=path {_=os.Remove(old)}
  return "Captura autorizada. Use desktop_ultima_captura para visualizar por ate 5 minutos.",nil
 }
 return desktopPerform(p.Action,action)
}
func desktopLatestCapture()(any,error){
 captureMu.Lock();p,t:=latestCapture,captureAt;captureMu.Unlock()
 if p==""||time.Since(t)>5*time.Minute {return nil,errors.New("nenhuma captura autorizada recentemente")}
 data,e:=os.ReadFile(p);if e!=nil{return nil,e}
 if len(data)>1500000 {return nil,errors.New("imagem muito grande")}
 return map[string]any{"media_type":"image/jpeg","encoding":"base64","data":encodeBase64Capture(data),"captured_at":t.Format(time.RFC3339)},nil
}
func desktopArg(args map[string]json.RawMessage)(DesktopAction,error){
 var out DesktopAction
 if raw:=args["window"];len(raw)>0 {
  var value string
  if e:=json.Unmarshal(raw,&value);e==nil {n,e:=strconv.ParseUint(value,10,64);if e!=nil{return out,e};out.Window=n}else{
   var n uint64;if e:=json.Unmarshal(raw,&n);e!=nil{return out,e};out.Window=n
  }
 }
 out.App=optionalString(args,"app");out.Text=optionalString(args,"texto")
 if len(args["x"])>0 {if e:=json.Unmarshal(args["x"],&out.X);e!=nil{return out,e}}
 if len(args["y"])>0 {if e:=json.Unmarshal(args["y"],&out.Y);e!=nil{return out,e}}
 return out,nil
}
