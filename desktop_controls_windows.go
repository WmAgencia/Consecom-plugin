//go:build windows
package main

import (
 "errors"
 "fmt"
 "strings"
 "syscall"
 "unsafe"
)
var enumChildWindows=deskUser32.NewProc("EnumChildWindows")
var getClassName=deskUser32.NewProc("GetClassNameW")
var isWindowEnabled=deskUser32.NewProc("IsWindowEnabled")
var getAncestor=deskUser32.NewProc("GetAncestor")
type WinControl struct{
 ID uint64 `json:"id"`
 Class string `json:"class"`
 Label string `json:"label"`
 Enabled bool `json:"enabled"`
 X int32 `json:"x"`
 Y int32 `json:"y"`
 Width int32 `json:"width"`
 Height int32 `json:"height"`
}
func desktopChildControls(parent uint64)(any,error){
 _,rootRect,e:=desktopWindow(parent);if e!=nil{return nil,e}
 list:=[]WinControl{}
 callback:=syscall.NewCallback(func(hwnd,lp uintptr)uintptr{
  if len(list)>=90{return 0}
  visible,_,_:=isWindowVisible.Call(hwnd);if visible==0{return 1}
  class:=make([]uint16,140)
  n,_,_:=getClassName.Call(hwnd,uintptr(unsafe.Pointer(&class[0])),uintptr(len(class)))
  if n==0{return 1}
  title:=make([]uint16,180)
  _,_,_=getWindowText.Call(hwnd,uintptr(unsafe.Pointer(&title[0])),uintptr(len(title)))
  var rect desktopRect
  ok,_,_:=getWindowRect.Call(hwnd,uintptr(unsafe.Pointer(&rect)))
  if ok==0||rect.Right-rect.Left<4||rect.Bottom-rect.Top<4{return 1}
  enabled,_,_:=isWindowEnabled.Call(hwnd)
  cls:=syscall.UTF16ToString(class)
  label:=strings.TrimSpace(syscall.UTF16ToString(title))
  if len(label)>140{label=label[:140]}
  list=append(list,WinControl{ID:uint64(hwnd),Class:cls,Label:label,Enabled:enabled!=0,X:rect.Left-rootRect.Left,Y:rect.Top-rootRect.Top,Width:rect.Right-rect.Left,Height:rect.Bottom-rect.Top})
  return 1
 })
 ok,_,e:=enumChildWindows.Call(uintptr(parent),callback,0)
 if ok==0 && len(list)==0 {return nil,fmt.Errorf("EnumChildWindows: %v",e)}
 return map[string]any{"window":parent,"controls":list,"count":len(list),"note":"Controles Win32; aplicativos modernos podem exigir UI Automation em versao futura"},nil
}
func desktopClickChild(parent,child uint64)(string,error){
 if child==0||child==parent{return "",errors.New("controle invalido")}
 _,rootRect,e:=desktopWindow(parent);if e!=nil{return "",e}
 ancestor,_,_:=getAncestor.Call(uintptr(child),2)
 if ancestor!=uintptr(parent){return "",errors.New("controle nao pertence a janela autorizada")}
 visible,_,_:=isWindowVisible.Call(uintptr(child))
 enabled,_,_:=isWindowEnabled.Call(uintptr(child))
 if visible==0||enabled==0{return "",errors.New("controle invisivel ou desabilitado")}
 var rect desktopRect
 ok,_,_:=getWindowRect.Call(uintptr(child),uintptr(unsafe.Pointer(&rect)))
 if ok==0||rect.Right<=rect.Left||rect.Bottom<=rect.Top{return "",errors.New("retangulo invalido")}
 x:=int(rect.Left+(rect.Right-rect.Left)/2-rootRect.Left)
 y:=int(rect.Top+(rect.Bottom-rect.Top)/2-rootRect.Top)
 return desktopPerform("click",DesktopAction{Window:parent,X:x,Y:y})
}
