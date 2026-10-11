//go:build !windows
package main
import "errors"
type WinControl struct{
 ID uint64
 Class string
 Label string
 Enabled bool
 X int32
 Y int32
 Width int32
 Height int32
}
func desktopChildControls(uint64)(any,error){return nil,errors.New("somente Windows")}
func desktopClickChild(uint64,uint64)(string,error){return "",errors.New("somente Windows")}
