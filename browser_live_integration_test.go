package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveBrowserCDPWithRealChromium(t *testing.T) {
	if os.Getenv("CONSECOM_REAL_CDP_TEST") != "1" {
		t.Skip("opt-in")
	}
	executable, e := exec.LookPath("chromium")
	if e != nil {
		t.Skip("no chromium")
	}
	port, e := freeBrowserPort()
	if e != nil {
		t.Fatal(e)
	}
	root := filepath.Join(t.TempDir(), "chrome")
	cmd := exec.Command(executable, "--headless=new", "--no-sandbox", "--disable-gpu", "--no-first-run", "--no-default-browser-check", fmt.Sprintf("--remote-debugging-port=%d", port), "--remote-debugging-address=127.0.0.1", "--user-data-dir="+root, "about:blank")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	if e := waitBrowserReady(port, 10*time.Second); e != nil {
		t.Fatal(e)
	}
	// The raw browser websocket, not the page endpoint from /json/list:
	body, e := devtoolsHTTP(port, "/json/version", "GET")
	if e != nil {
		t.Fatal(e)
	}
	var meta struct {
		BrowserWS string `json:"webSocketDebuggerUrl"`
	}
	if e := json.Unmarshal(body, &meta); e != nil {
		t.Fatal(e)
	}
	if meta.BrowserWS == "" {
		t.Fatal("no browser websocket")
	}
	id, e := createLiveBrowserTarget(meta.BrowserWS)
	if e != nil {
		t.Fatal(e)
	}
	s := &BrowserSession{Browser: "chrome", Profile: "live:current", Target: id, BrowserWS: meta.BrowserWS}
	if _, e := browserNavigateLocked(s, "about:blank"); e != nil {
		t.Fatal(e)
	}
	if _, e := cdpEvaluate(s, "document.title='CONSECOM-LIVE-OK'"); e != nil {
		t.Fatal(e)
	}
	v, e := browserSnapshotLocked(s)
	if e != nil {
		t.Fatal(e)
	}
	snap := v.(map[string]any)
	if snap["title"] != "CONSECOM-LIVE-OK" {
		t.Fatalf("live snapshot: %v", snap)
	}
	if _, e := cdpBrowserCommand(meta.BrowserWS, "Target.closeTarget", map[string]any{"targetId": id}, ""); e != nil {
		t.Fatal(e)
	}
}

func TestLiveChromePortMetadata(t *testing.T) {
	base := t.TempDir()
	t.Setenv("LOCALAPPDATA", base)
	root := filepath.Join(base, "Google", "Chrome", "User Data")
	if e := os.MkdirAll(root, 0700); e != nil {
		t.Fatal(e)
	}
	if _, e := liveChromeWebSocket(); e == nil {
		t.Fatal("accepted missing consent")
	}
	if e := os.WriteFile(filepath.Join(root, "DevToolsActivePort"), []byte("9222\n/devtools/browser/test-id\n"), 0600); e != nil {
		t.Fatal(e)
	}
	ws, e := liveChromeWebSocket()
	if e != nil || ws != "ws://127.0.0.1:9222/devtools/browser/test-id" {
		t.Fatalf("%q %v", ws, e)
	}
	if e := os.WriteFile(filepath.Join(root, "DevToolsActivePort"), []byte("4444\n/devtools/browser/x?@foo\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := liveChromeWebSocket(); e == nil {
		t.Fatal("unsafe endpoint accepted")
	}
}
