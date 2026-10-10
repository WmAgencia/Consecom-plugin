package main

import (
	"strings"
	"testing"
)

func TestBuildTunnelCommandWindowsPath(t *testing.T) {
	got := buildTunnelCommand(`C:\Users\junin\AppData\Local\ConsecomMCP\bin\MCP-PC.exe`)
	want := `"C:/Users/junin/AppData/Local/ConsecomMCP/bin/MCP-PC.exe" --stdio`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if strings.Contains(got, `\`) {
		t.Fatalf("backslashes must not reach shlex parser: %q", got)
	}
}

func TestBuildTunnelCommandSpaces(t *testing.T) {
	got := buildTunnelCommand(`C:\Users\Nome Com Espaco\AppData\Local\ConsecomMCP\bin\MCP-PC.exe`)
	want := `"C:/Users/Nome Com Espaco/AppData/Local/ConsecomMCP/bin/MCP-PC.exe" --stdio`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
