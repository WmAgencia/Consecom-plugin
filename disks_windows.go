//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var diskDLL = syscall.NewLazyDLL("kernel32.dll")
var getLogicalDrives = diskDLL.NewProc("GetLogicalDrives")
var getDiskFreeSpaceExW = diskDLL.NewProc("GetDiskFreeSpaceExW")

func diskList() []map[string]any {
	mask, _, _ := getLogicalDrives.Call()
	out := []map[string]any{}
	for i := 0; i < 26; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		drive := fmt.Sprintf("%c:\\", 'A'+i)
		var free, total, other uint64
		ptr, e := syscall.UTF16PtrFromString(drive)
		if e != nil {
			continue
		}
		ok, _, _ := getDiskFreeSpaceExW.Call(uintptr(unsafe.Pointer(ptr)), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&other)))
		if ok == 0 || total == 0 {
			continue
		}
		out = append(out, map[string]any{"label": fmt.Sprintf("%c:", 'A'+i), "total": total, "free": free, "used": total - free})
	}
	return out
}
