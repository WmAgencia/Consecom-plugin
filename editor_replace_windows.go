//go:build windows
package main
import ("syscall";"unsafe")
func replaceProjectFile(tmp,target string)error {
 kernel:=syscall.NewLazyDLL("kernel32.dll")
 move:=kernel.NewProc("MoveFileExW")
 a,e:=syscall.UTF16PtrFromString(tmp);if e!=nil{return e}
 b,e:=syscall.UTF16PtrFromString(target);if e!=nil{return e}
 ok,_,err:=move.Call(uintptr(unsafe.Pointer(a)),uintptr(unsafe.Pointer(b)),uintptr(1|8))
 if ok==0{return err}
 return nil
}