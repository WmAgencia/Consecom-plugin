package main

import (
	"fmt"
	"strings"
)

// The official tunnel-client parses --mcp.command as a shell-style command
// line even on Windows. Raw Windows backslashes are consumed as escapes,
// turning C:\Users\... into C:Users.... Forward slashes work with Win32
// CreateProcess and are not escape characters for the command parser.
func buildTunnelCommand(exePath string) string {
	normalized := strings.ReplaceAll(exePath, `\`, `/`)
	return fmt.Sprintf("%q --stdio", normalized)
}
