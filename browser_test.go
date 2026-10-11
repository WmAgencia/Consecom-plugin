package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserDefaultExclusiveWhenInstalled(t *testing.T) {
 root:=t.TempDir()
 t.Setenv("PROGRAMFILES",root)
 t.Setenv("PROGRAMFILES(X86)","")
 exe:=filepath.Join(root,`Google\Chrome\Application`,"chrome.exe")
 if err:=os.MkdirAll(filepath.Dir(exe),0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(exe,[]byte("stub"),0600);err!=nil{t.Fatal(err)}
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	if p:=a.getBrowserPreference();p.Browser!="chrome"||p.Profile!="managed:chatgpt"{t.Fatalf("expected exclusive Chrome: %+v",p)}
 if browser,profile,e:=a.resolveBrowserChoice("","");e!=nil||browser!="chrome"||profile!="managed:chatgpt"{t.Fatalf("resolved %q %q: %v",browser,profile,e)}
}
func TestBrowserInvalidPreferenceNotSaved(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range []BrowserPreference{{Browser: "", Profile: ""}, {Browser: "", Profile: "personal:Default"}, {Browser: "bad", Profile: "managed:chatgpt"}, {Browser:"chrome",Profile:"live:current"}} {
		if a.setBrowserPreference(p) == nil {
			t.Fatalf("accepted invalid %+v", p)
		}
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

// Preserve legacy recognition without permitting Chrome personal debugging.
func TestLiveProfileCannotBecomeDefault(t *testing.T){
 root:=t.TempDir()
 t.Setenv("PROGRAMFILES",root)
 t.Setenv("PROGRAMFILES(X86)","")
 exe:=filepath.Join(root,`Google\Chrome\Application`,"chrome.exe")
 if err:=os.MkdirAll(filepath.Dir(exe),0700);err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(exe,[]byte("stub"),0600);err!=nil{t.Fatal(err)}
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 if err:=a.setBrowserPreference(BrowserPreference{Browser:"chrome",Profile:"live:current"});err==nil{t.Fatal("accepted personal browser")}
 if _,_,err:=a.resolveBrowserChoice("chrome","live:current");err==nil{t.Fatal("attached personal Chrome")}
 options:=a.browserOptions()
 choices:=options["browsers"].([]BrowserChoice)
 for _,c:=range choices{for _,p:=range c.Profiles{if p.ID!="managed:chatgpt"{t.Fatalf("unexpected profile %q",p.ID)}}}
}
