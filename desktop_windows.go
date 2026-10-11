//go:build windows

package main

import (
 "bytes"
 "encoding/base64"
 "errors"
 "fmt"
 "image"
 "image/color"
 "image/jpeg"
 "os"
 "os/exec"
 "strings"
 "syscall"
 "time"
 "unicode/utf16"
 "unsafe"
)
var deskUser32=syscall.NewLazyDLL("user32.dll")
var deskGDI=syscall.NewLazyDLL("gdi32.dll")
var enumWindows=deskUser32.NewProc("EnumWindows")
var isWindowVisible=deskUser32.NewProc("IsWindowVisible")
var getWindowTextLength=deskUser32.NewProc("GetWindowTextLengthW")
var getWindowText=deskUser32.NewProc("GetWindowTextW")
var getWindowProcess=deskUser32.NewProc("GetWindowThreadProcessId")
var getWindowRect=deskUser32.NewProc("GetWindowRect")
var setForeground=deskUser32.NewProc("SetForegroundWindow")
var getForeground=deskUser32.NewProc("GetForegroundWindow")
var showWindow=deskUser32.NewProc("ShowWindow")
var setCursorPos=deskUser32.NewProc("SetCursorPos")
var mouseEvent=deskUser32.NewProc("mouse_event")
var sendInput=deskUser32.NewProc("SendInput")
var getDC=deskUser32.NewProc("GetDC")
var releaseDC=deskUser32.NewProc("ReleaseDC")
var createCompatibleDC=deskGDI.NewProc("CreateCompatibleDC")
var deleteDC=deskGDI.NewProc("DeleteDC")
var createCompatibleBitmap=deskGDI.NewProc("CreateCompatibleBitmap")
var selectObject=deskGDI.NewProc("SelectObject")
var deleteObject=deskGDI.NewProc("DeleteObject")
var bitBlt=deskGDI.NewProc("BitBlt")
var getDIBits=deskGDI.NewProc("GetDIBits")
type desktopRect struct{Left,Top,Right,Bottom int32}
type bitmapInfoHeader struct{
 Size uint32
 Width int32
 Height int32
 Planes uint16
 BitCount uint16
 Compression uint32
 SizeImage uint32
 XPels int32
 YPels int32
 ClrUsed uint32
 ClrImportant uint32
}
type bitmapInfo struct{ Header bitmapInfoHeader; Colors [1]uint32 }
type desktopEntry struct{
 ID uint64 `json:"id"`
 Title string `json:"title"`
 Process string `json:"process"`
 PID uint32 `json:"pid"`
 Width int32 `json:"width"`
 Height int32 `json:"height"`
}
func desktopEntries()([]desktopEntry,error){
 list:=[]desktopEntry{}
 cb:=syscall.NewCallback(func(hwnd,lp uintptr)uintptr{
  visible,_,_:=isWindowVisible.Call(hwnd)
  if visible==0{return 1}
  n,_,_:=getWindowTextLength.Call(hwnd)
  if n==0||n>280{return 1}
  title:=make([]uint16,int(n)+1)
  result,_,_:=getWindowText.Call(hwnd,uintptr(unsafe.Pointer(&title[0])),uintptr(len(title)))
  if result==0{return 1}
  var pid uint32
  getWindowProcess.Call(hwnd,uintptr(unsafe.Pointer(&pid)))
  if pid<=4{return 1}
  proc,e:=processName(int(pid));if e!=nil{return 1}
  process:=strings.ToLower(proc)
  for _,blocked:=range []string{"lsass","winlogon","credentialuibroker","consent","securityhealth","taskmgr","logonui"} {
   if process==blocked{return 1}
  }
  var rect desktopRect
  getWindowRect.Call(hwnd,uintptr(unsafe.Pointer(&rect)))
  w,h:=rect.Right-rect.Left,rect.Bottom-rect.Top
  if w<100||h<60{return 1}
  list=append(list,desktopEntry{ID:uint64(hwnd),Title:strings.TrimSpace(syscall.UTF16ToString(title)),Process:proc,PID:pid,Width:w,Height:h})
  return 1
 })
 ok,_,err:=enumWindows.Call(cb,0)
 if ok==0 {return nil,fmt.Errorf("EnumWindows: %v",err)}
 if len(list)>60{list=list[:60]}
 return list,nil
}
func desktopWindow(id uint64)(desktopEntry,desktopRect,error){
 var rect desktopRect
 entries,e:=desktopEntries();if e!=nil{return desktopEntry{},rect,e}
 for _,entry:=range entries {
  if entry.ID==id{
   ok,_,err:=getWindowRect.Call(uintptr(id),uintptr(unsafe.Pointer(&rect)))
   if ok==0{return entry,rect,err}
   return entry,rect,nil
  }
 }
 return desktopEntry{},rect,errors.New("janela nao esta visivel ou esta protegida")
}
func desktopPerform(op string,a DesktopAction)(string,error){
 if op=="open_installed" {return launchRegisteredWindowsApp(a.AppID)}
 if op=="open_executable" {return launchPortableWindowsExe(a.Executable)}
 if op=="open" {
  binaries:=map[string]string{"notepad":"notepad.exe","calculator":"calc.exe","explorer":"explorer.exe","vscode":"code"}
  bin:=binaries[a.App];if bin==""{return "",errors.New("aplicativo nao autorizado")}
  path,e:=exec.LookPath(bin);if e!=nil{return "",e}
  cmd:=exec.Command(path)
  if e=cmd.Start();e!=nil{return "",e}
  return "Aplicativo autorizado iniciado: "+a.App,nil
 }
 _,rect,e:=desktopWindow(a.Window);if e!=nil{return "",e}
 showWindow.Call(uintptr(a.Window),9)
 setForeground.Call(uintptr(a.Window))
 time.Sleep(95*time.Millisecond)
 active,_,_:=getForeground.Call()
 if active!=uintptr(a.Window){return "",errors.New("Windows nao permitiu colocar a janela em primeiro plano; ative-a manualmente")}
 if op=="activate"{return "Janela ativada",nil}
 if op=="click"{
  if a.X<0||a.Y<0||int32(a.X)>=rect.Right-rect.Left||int32(a.Y)>=rect.Bottom-rect.Top {return "",errors.New("clique fora dos limites da janela")}
  ok,_,err:=setCursorPos.Call(uintptr(int32(a.X)+rect.Left),uintptr(int32(a.Y)+rect.Top))
  if ok==0{return "",err}
  mouseEvent.Call(0x0002,0,0,0,0);mouseEvent.Call(0x0004,0,0,0,0)
  return "Clique local aprovado executado",nil
 }
 if op=="type"{
  if len(a.Text)>1000||len(a.Text)==0||strings.IndexFunc(a.Text,func(r rune)bool{return r<32})>=0{return "",errors.New("texto invalido ou contem tecla especial")}
  // Unicode key events, without invoking shell or clipboard.
  for _,ch:=range utf16.Encode([]rune(a.Text)){
   active,_,_:=getForeground.Call()
   if active!=uintptr(a.Window){return "",errors.New("foco mudou durante a digitacao; entrada interrompida")}
   var keys [2]struct{ Type uint32; pad uint32; data [32]byte}
   keys[0].Type=1; keys[1].Type=1
   *(*uint16)(unsafe.Pointer(&keys[0].data[2]))=ch
   *(*uint16)(unsafe.Pointer(&keys[1].data[2]))=ch
   *(*uint32)(unsafe.Pointer(&keys[0].data[4]))=0x0004
   *(*uint32)(unsafe.Pointer(&keys[1].data[4]))=0x0004|0x0002
   count,_,err:=sendInput.Call(2,uintptr(unsafe.Pointer(&keys[0])),unsafe.Sizeof(keys[0]))
   if count!=2{return "",fmt.Errorf("SendInput: %v",err)}
  }
  return fmt.Sprintf("Digitados %d caracteres aprovados",len([]rune(a.Text))),nil
 }
 return "",errors.New("operacao desconhecida")
}
func desktopScreenshot(id uint64)([]byte,error){
 _,rect,e:=desktopWindow(id);if e!=nil{return nil,e}
 w,h:=rect.Right-rect.Left,rect.Bottom-rect.Top
 if w<=0||h<=0||int64(w)*int64(h)>6500000{return nil,errors.New("janela com dimensoes fora do limite")}
 screen,_,_:=getDC.Call(0)
 if screen==0{return nil,errors.New("DC da tela indisponivel")}
 defer releaseDC.Call(0,screen)
 mem,_,_:=createCompatibleDC.Call(screen);if mem==0{return nil,errors.New("DC de captura indisponivel")};defer deleteDC.Call(mem)
 bitmap,_,_:=createCompatibleBitmap.Call(screen,uintptr(w),uintptr(h));if bitmap==0{return nil,errors.New("bitmap indisponivel")};defer deleteObject.Call(bitmap)
 prev,_,_:=selectObject.Call(mem,bitmap);defer selectObject.Call(mem,prev)
 copied,_,_:=bitBlt.Call(mem,0,0,uintptr(w),uintptr(h),screen,uintptr(rect.Left),uintptr(rect.Top),0x00CC0020)
 if copied==0{return nil,errors.New("falha BitBlt; janela possivelmente protegida")}
 dib:=bitmapInfo{Header:bitmapInfoHeader{Size:40,Width:w,Height:-h,Planes:1,BitCount:32}}
 pixels:=make([]byte,int(w)*int(h)*4)
 got,_,_:=getDIBits.Call(mem,bitmap,0,uintptr(h),uintptr(unsafe.Pointer(&pixels[0])),uintptr(unsafe.Pointer(&dib)),0)
 if got==0{return nil,errors.New("falha GetDIBits")}
 dstW,dstH:=int(w),int(h)
 if dstW>1100 {dstH=dstH*1100/dstW;dstW=1100}
 if dstH>900 {dstW=dstW*900/dstH;dstH=900}
 img:=image.NewRGBA(image.Rect(0,0,dstW,dstH))
 for y:=0;y<dstH;y++{sy:=y*int(h)/dstH;for x:=0;x<dstW;x++{sx:=x*int(w)/dstW;off:=(sy*int(w)+sx)*4;img.SetRGBA(x,y,color.RGBA{R:pixels[off+2],G:pixels[off+1],B:pixels[off],A:255})}}
 var buf bytes.Buffer
 if e:=jpeg.Encode(&buf,img,&jpeg.Options{Quality:63});e!=nil{return nil,e}
 if buf.Len()>1400000{return nil,errors.New("captura acima do limite")}
 return buf.Bytes(),nil
}
func encodeBase64Capture(b []byte)string{return base64.StdEncoding.EncodeToString(b)}
var _ = os.ErrNotExist
