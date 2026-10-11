//go:build !windows
package main
import "os"
func replaceProjectFile(tmp,target string)error{return os.Rename(tmp,target)}
