//go:build !windows

package main

import "os"

// Windows remains the shipping target; non-Windows preview process cleanup
// stops the spawned process. Descendant process isolation is a later milestone.
func stopPreviewProcessTree(proc *os.Process){
 if proc!=nil{_ = proc.Kill()}
}
