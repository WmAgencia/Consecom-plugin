//go:build !windows

package main

import ("errors";"encoding/base64")
func desktopEntries()([]desktopEntry,error){return nil,errors.New("desktop disponivel apenas no Windows")}
type desktopEntry struct { ID uint64; Title string; Process string }
func desktopPerform(_ string,_ DesktopAction)(string,error){return "",errors.New("desktop disponivel apenas no Windows")}
func desktopScreenshot(_ uint64)([]byte,error){return nil,errors.New("screenshot disponivel apenas no Windows")}
func encodeBase64Capture(b []byte)string{return base64.StdEncoding.EncodeToString(b)}
