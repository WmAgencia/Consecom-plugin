//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

func launchBrowser(exe, dir string, port int) (int, error) {
	cmd := exec.Command(exe, "--no-first-run", "--no-default-browser-check", "--remote-debugging-address=127.0.0.1", fmt.Sprintf("--remote-debugging-port=%d", port), "--user-data-dir="+dir, "about:blank")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if e := cmd.Start(); e != nil {
		return 0, e
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}
