//go:build !windows

package main

import (
	"os/exec"
)

func platformHiddenInstaller(cmd *exec.Cmd) error { return nil }
