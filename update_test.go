package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpdateManifestAllowsOnlyPinnedGitHubRepository(t *testing.T) {
	m := UpdateManifest{Version: "4.1.1", URL: "https://github.com/WmAgencia/Consecom-plugin/releases/download/v4.1.1/Instalar_Consecom_MCP_v2_6_1.exe", SHA256: strings.Repeat("a", 64)}
	if e := validateUpdateManifest(officialUpdateRepo, m); e != nil {
		t.Fatal(e)
	}
	bad := []string{"http://github.com/wmagencia/consecom-mcp/releases/download/v4.1.1/update.exe", "https://attacker.com/installer.exe", "https://github.com/evil/evil/releases/download/v4.1.1/installer.exe", "https://github.com/WmAgencia/Consecom-plugin/releases/download/v2.9.9/installer.exe", "https://github.com/WmAgencia/Consecom-plugin/releases/download/v4.1.1/a.zip", "https://github.com/WmAgencia/Consecom-plugin/releases/download/v4.1.1/x.exe?redirect=hack"}
	for _, u := range bad {
		c := m
		c.URL = u
		if e := validateUpdateManifest(officialUpdateRepo, c); e == nil {
			t.Errorf("accepted dangerous update url %q", u)
		}
	}
	for _, v := range []string{"2.5.1-beta", "2.5", "2.5.0.2", "9999999999999999999999.0.0"} {
		c := m
		c.Version = v
		if e := validateUpdateManifest(officialUpdateRepo, c); e == nil {
			t.Errorf("accepted invalid version %q", v)
		}
	}
}
func TestUpdateChecksLocalPermissionsAndPersists(t *testing.T) {
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e := a.updateSettings(UpdateSettings{Repo: officialUpdateRepo, RemoteApproved: true}, ""); e == nil {
		t.Fatal("remote updates enabled without local phrase")
	}
	if e := a.updateSettings(UpdateSettings{Repo: officialUpdateRepo, RemoteApproved: true}, "CONFIO NAS ATUALIZACOES"); e != nil {
		t.Fatal(e)
	}
	if !a.updateStatus().RemoteApproved {
		t.Fatal("remote setting not saved")
	}
	b, e := NewApp(a.path)
	if e != nil {
		t.Fatal(e)
	}
	if !b.updateStatus().RemoteApproved {
		t.Fatal("remote setting not persisted")
	}
	if e := b.updateSettings(UpdateSettings{Repo: "evil/../repo"}, ""); e == nil {
		t.Fatal("unsafe github repo accepted")
	}
	if e := b.updateSettings(UpdateSettings{Repo: "", RemoteApproved: false}, ""); e != nil {
		t.Fatal(e)
	}
}
func TestUpdateSHA256RejectsTampering(t *testing.T) {
	b := []byte("installer bytes test")
	digest := sha256.Sum256(b)
	hash := hex.EncodeToString(digest[:])
	p := filepath.Join(t.TempDir(), "update.exe")
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
	if e := verifyFileSHA(p, hash); e != nil {
		t.Fatal(e)
	}
	if e := verifyFileSHA(p, strings.Repeat("f", 64)); e == nil {
		t.Fatal("accepted wrong hash")
	}
	if e := os.WriteFile(p, []byte("tampered"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := verifyFileSHA(p, hash); e == nil {
		t.Fatal("accepted modified package")
	}
}
func TestUpdateSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{{"4.1.1", "2.5.0", 1}, {"2.5.0", "2.5.0", 0}, {"2.4.9", "2.5.0", -1}, {"2.10.0", "2.9.99", 1}}
	for _, c := range cases {
		v, e := compareSemver(c.a, c.b)
		if e != nil || v != c.want {
			t.Fatalf("%s/%s got %d %v want %d", c.a, c.b, v, e, c.want)
		}
	}
	if _, e := compareSemver("2.5.dev", "2.5.0"); e == nil {
		t.Fatal("invalid version accepted")
	}
}
func TestUpdaterToolsExposed(t *testing.T) {
	seen := map[string]bool{}
	for _, tool := range tools() {
		seen[tool.Name] = true
	}
	for _, name := range []string{"atualizacao_status", "atualizacao_verificar", "atualizacao_solicitar"} {
		if !seen[name] {
			t.Fatalf("missing updater tool %s", name)
		}
	}
	if len(tools()) != 77 {
		t.Fatalf("expected 77 tools got %d", len(tools()))
	}
}

type updateRoundTripper func(*http.Request) (*http.Response, error)

func (f updateRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestUpdaterStagesVerifiedReleaseAndQueuesApproval(t *testing.T) {
	payload := []byte("fake installer for network flow")
	hash := sha256.Sum256(payload)
	sha := hex.EncodeToString(hash[:])
	repo := officialUpdateRepo
	manifest := UpdateManifest{Version: "4.1.1", URL: "https://github.com/WmAgencia/Consecom-plugin/releases/download/v4.1.1/Instalar_Consecom_2_6_1.exe", SHA256: sha}
	mb, _ := json.Marshal(manifest)
	old := updateHTTPClient
	updateHTTPClient = &http.Client{Timeout: 5 * time.Second, Transport: updateRoundTripper(func(r *http.Request) (*http.Response, error) {
		var data []byte
		if r.URL.Path == "/WmAgencia/Consecom-plugin/releases/latest/download/consecom-update.json" {
			data = mb
		} else if r.URL.String() == manifest.URL {
			data = payload
		} else {
			return nil, fmt.Errorf("unexpected URL %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), ContentLength: int64(len(data)), Header: make(http.Header)}, nil
	})}
	defer func() { updateHTTPClient = old }()
	a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e = a.updateSettings(UpdateSettings{Repo: repo}, ""); e != nil {
		t.Fatal(e)
	}
	check, e := a.updateCheck()
	if e != nil {
		t.Fatal(e)
	}
	if !check["update_available"].(bool) {
		t.Fatal("release not recognized")
	}
	res, e := a.updateRequest()
	if e != nil {
		t.Fatal(e)
	}
	if res["status"] != "aguardando_aprovacao_local" {
		t.Fatalf("unexpected update status: %v", res)
	}
	pending := a.pendings()
	if len(pending) != 1 {
		t.Fatalf("expected one approval, got %d", len(pending))
	}
	id := res["approval_id"].(string)
	p, e := a.pop(id)
	if e != nil {
		t.Fatal(e)
	}
	if p.Kind != "update" {
		t.Fatalf("invalid pending kind %q", p.Kind)
	}
	if e = verifyFileSHA(p.Path, sha); e != nil {
		t.Fatal(e)
	}
	// Exactly the bytes from the pinned repo are staged, never executable merely on request.
	if strings.HasSuffix(p.Path, ".exe") == false {
		t.Fatalf("invalid staging path %s", p.Path)
	}
}

func TestOfficialSourceWithoutConfiguration(t *testing.T) {
 a, e := NewApp(filepath.Join(t.TempDir(), "config.json"))
 if e != nil { t.Fatal(e) }
 got := a.updateStatus()
 if !got.Configured || got.Repo != officialUpdateRepo { t.Fatalf("repo nao fixo: %+v", got) }
 if e := a.updateSettings(UpdateSettings{Repo: "other/repo"}, ""); e == nil { t.Fatal("origem arbitraria aceita") }
 if e := a.updateSettings(UpdateSettings{RemoteApproved:true}, "CONFIO NAS ATUALIZACOES"); e != nil { t.Fatal(e) }
 if !a.updateStatus().RemoteApproved { t.Fatal("autorizacao remota nao salva") }
 b, e := NewApp(a.path); if e != nil { t.Fatal(e) }
 if !b.updateStatus().RemoteApproved { t.Fatal("permissao remota perdida") }
 b.mu.Lock(); b.cfg.Updates=UpdateSettings{Repo:"older/different", RemoteApproved:true}; b.mu.Unlock()
 if b.updateStatus().RemoteApproved { t.Fatal("autorizacao de outra origem reaproveitada") }
}
