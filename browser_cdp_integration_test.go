package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Opt-in real Chromium integration test. Runs headless browser on localhost;
// does not access the user's Windows profile or the public internet.
func TestCDPWithRealChromium(t *testing.T) {
	if os.Getenv("CONSECOM_REAL_CDP_TEST") != "1" {
		t.Skip("opt-in: set CONSECOM_REAL_CDP_TEST=1")
	}
	executable, e := exec.LookPath("chromium")
	if e != nil {
		t.Skip("chromium not installed")
	}
	pageHTML := `<!doctype html><title>Teste Consecom CDP</title><h1>Teste real de navegador</h1><input id="q" type="text"/><button id="safe" onclick="document.getElementById('output').textContent='OK ' + document.getElementById('q').value">Pesquisar</button><button id="danger" type="submit">Enviar</button><p id="output"></p>`
	pageURL := "about:blank"
	port, e := freeBrowserPort()
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(executable, "--headless=new", "--no-sandbox", "--disable-gpu", "--no-first-run", "--no-default-browser-check", fmt.Sprintf("--remote-debugging-port=%d", port), "--remote-debugging-address=127.0.0.1", "--user-data-dir="+filepath.Join(t.TempDir(), "chromium"), "about:blank")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if e := cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	if e := waitBrowserReady(port, 12*time.Second); e != nil {
		t.Fatal(e)
	}
	target, e := newBrowserTarget(port)
	if e != nil {
		t.Fatal(e)
	}
	session := &BrowserSession{Port: port, Target: target}
	_, e = browserNavigateLocked(session, pageURL)
	if e != nil {
		t.Fatal(e)
	}
	// Enterprise browser policy blocks data: and localhost in this environment;
	// inject a controlled test DOM into about:blank via the private CDP test.
	_, e = cdpEvaluate(session, `document.body.innerHTML=`+jsonJS(pageHTML)+`;document.title='Teste Consecom CDP';true`)
	if e != nil {
		t.Fatal(e)
	}
	v, e := browserSnapshotLocked(session)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(fmt.Sprint(v), "Teste real de navegador") {
		t.Fatalf("page not read: %v", v)
	}
	if v, e = browserTypeLocked(session, "#q", "Sorocaba"); e != nil {
		t.Fatal(e)
	}
	v, e = browserClickLocked(session, "#safe")
	if e != nil {
		t.Fatal(e)
	}
	page, e := browserSnapshotLocked(session)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(fmt.Sprint(page), "OK Sorocaba") {
		t.Fatalf("click did not work: %v", page)
	}
	blocked, e := browserClickLocked(session, "#danger")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(fmt.Sprint(blocked), "acao sensivel") {
		t.Fatalf("submit was not blocked: %v", blocked)
	}
	shot, e := browserScreenshotLocked(session)
	if e != nil {
		t.Fatal(e)
	}
	if len(shot.(BrowserScreenshot).Data) < 2500 {
		t.Fatal("screenshot missing")
	}
	t.Logf("Success! Loaded local page, typed, clicked, blocked submit, screenshot (%d chars)", len(shot.(BrowserScreenshot).Data))
}
