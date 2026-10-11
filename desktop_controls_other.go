//go:build !windows
package main
import "errors"
func desktopChildControls(uint64)(any,error){return nil,errors.New("somente Windows")}
func desktopClickChild(uint64,uint64)(string,error){return "",errors.New("somente Windows")}
