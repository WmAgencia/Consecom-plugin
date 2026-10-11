package main

import(
 "encoding/json"
 "errors"
 "fmt"
 "strconv"
)
type DesktopElementAction struct {
 Parent uint64 `json:"parent"`
 Child uint64 `json:"child"`
}
func (a *App) desktopElementClick(parentText,childText string)(any,error){
 parent,e:=strconv.ParseUint(parentText,10,64);if e!=nil||parent==0{return nil,errors.New("id de janela invalido")}
 child,e:=strconv.ParseUint(childText,10,64);if e!=nil||child==0{return nil,errors.New("id de controle invalido")}
 // Validate the Win32 parent-child relationship before requesting confirmation.
 controls,e:=desktopChildControls(parent);if e!=nil{return nil,e}
 m,ok:=controls.(map[string]any);if !ok{return nil,errors.New("lista de controles invalida")}
 match:=false
 for _,c:=range m["controls"].([]WinControl){if c.ID==child && c.Enabled {match=true;break}}
 if !match{return nil,errors.New("controle nao encontrado ou desabilitado")}
 b,_:=json.Marshal(DesktopElementAction{Parent:parent,Child:child})
 p:=&Pending{Kind:"winchild",Content:string(b),Description:fmt.Sprintf("Clicar controle %d da janela %d",child,parent)}
 if a.mode()=="maximum"{
  result,e:=a.approveDesktopElement(p);if e!=nil{return nil,e}
  return map[string]any{"status":"executada","resultado":result},nil
 }
 id,e:=a.queue(p);if e!=nil{return nil,e}
 return map[string]any{"status":"aguardando_aprovacao_local","approval_id":id},nil
}
func (a *App) approveDesktopElement(p *Pending)(string,error){
 var req DesktopElementAction
 if e:=json.Unmarshal([]byte(p.Content),&req);e!=nil{return "",e}
 return desktopClickChild(req.Parent,req.Child)
}
