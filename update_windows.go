//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func platformHiddenInstaller(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	return nil
}
