package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserSettingsAskByDefault(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	if p := a.getBrowserPreference(); p.Browser != "" || p.Profile != "" {
		t.Fatal("must ask by default")
	}
	if _, _, e := a.resolveBrowserChoice("", ""); e == nil || !strings.Contains(e.Error(), "SELECAO_NECESSARIA") {
		t.Fatalf("did not require choice: %v", e)
	}
}
func TestBrowserInvalidPreferenceNotSaved(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range []BrowserPreference{{Browser: "chrome", Profile: ""}, {Browser: "", Profile: "personal:Default"}, {Browser: "bad", Profile: "managed:chatgpt"}} {
		if a.setBrowserPreference(p) == nil {
			t.Fatalf("accepted invalid %+v", p)
		}
	}
	if e := a.setBrowserPreference(BrowserPreference{}); e != nil {
		t.Fatal(e)
	}
}
func TestBrowserURLRejectsUnsafeSchemes(t *testing.T) {
	for _, s := range []string{"javascript:alert(1)", "file:///windows/system32", "http://user:pass@example.com", "ftp://example.org", "http://", "https://example.org\nHost:attacker"} {
		if _, e := browserURL(s); e == nil {
			t.Errorf("accepted %q", s)
		}
	}
	if _, e := browserURL("https://example.org/test"); e != nil {
		t.Fatal(e)
	}
}
func TestBrowserToolListValid(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	r, ok := a.handleRPC([]byte(`{"jsonrpc":"2.0","id":12,"method":"tools/list"}`))
	if !ok {
		t.Fatal("missing")
	}
	b, _ := json.Marshal(r)
	if strings.Count(string(b), `"name":"browser_`) != 21 {
		t.Fatalf("expected 21 browser tools; listing has %d",strings.Count(string(b), `"name":"browser_`))
	}
}

// Regression: v2.4.0 advertised live:current in browser_opcoes but rejected
// it in browser_abrir and in the preference editor.
func TestLiveCurrentListedAndAccepted(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PROGRAMFILES", root)
	t.Setenv("PROGRAMFILES(X86)", "")
	t.Setenv("LOCALAPPDATA", t.TempDir())
	exe := filepath.Join(root, `Google\Chrome\Application`, "chrome.exe")
	if err := os.MkdirAll(filepath.Dir(exe), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("stub-executable"), 0600); err != nil {
		t.Fatal(err)
	}
	choices := discoverBrowsers()
	available := false
	for _, c := range choices {
		if c.ID == "chrome" {
			for _, p := range c.Profiles {
				if p.ID == "live:current" {
					available = true
				}
			}
		}
	}
	if !available {
		t.Fatal("live profile missing from options")
	}
	if !validBrowserProfile("chrome", "live:current") {
		t.Fatal("Chrome live profile advertised but not accepted")
	}
	if validBrowserProfile("edge", "live:current") {
		t.Fatal("Edge live profile incorrectly accepted")
	}
	a, err := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.setBrowserPreference(BrowserPreference{Browser: "chrome", Profile: "live:current"}); err != nil {
		t.Fatal(err)
	}
	browser, profile, err := a.resolveBrowserChoice("", "")
	if err != nil || browser != "chrome" || profile != "live:current" {
		t.Fatalf("got %q %q %v", browser, profile, err)
	}
}
