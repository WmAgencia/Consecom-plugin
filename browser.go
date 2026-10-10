package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Browser preferences are deliberately separate from approval mode. An empty
// browser/profile means ASK, even if the user enabled maximum file permissions.
type BrowserPreference struct {
	Browser string `json:"browser"`
	Profile string `json:"profile"`
}
type BrowserChoice struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Path      string           `json:"-"`
	Installed bool             `json:"installed"`
	Profiles  []BrowserProfile `json:"profiles"`
}
type BrowserProfile struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Notice string `json:"notice,omitempty"`
}
type BrowserSession struct {
	Browser      string
	Profile      string
	Port         int
	Target       string
	DataDir      string
	PID          int
	BrowserWS    string
	OwnedTargets []string
}
type BrowserControl struct {
	mu      sync.Mutex
	session *BrowserSession
}

var allowedProfileDir = regexp.MustCompile(`^(Default|Profile [1-9][0-9]{0,3})$`)

func findBrowserExe(browser string) string {
	var sub, exe string
	switch browser {
	case "chrome":
		sub, exe = `Google\Chrome\Application`, "chrome.exe"
	case "edge":
		sub, exe = `Microsoft\Edge\Application`, "msedge.exe"
	default:
		return ""
	}
	for _, base := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)"), os.Getenv("LOCALAPPDATA")} {
		if base == "" {
			continue
		}
		p := filepath.Join(base, sub, exe)
		if s, e := os.Stat(p); e == nil && !s.IsDir() {
			return p
		}
	}
	return ""
}
func browserDataRoot(browser string) string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return ""
	}
	if browser == "chrome" {
		return filepath.Join(local, "Google", "Chrome", "User Data")
	}
	if browser == "edge" {
		return filepath.Join(local, "Microsoft", "Edge", "User Data")
	}
	return ""
}
func listBrowserProfiles(browser string) []BrowserProfile {
	profiles := []BrowserProfile{{ID: "managed:chatgpt", Name: "ChatGPT (exclusivo)", Kind: "managed", Notice: "Sessão própria; seus logins não são compartilhados com o navegador pessoal."}}
	dataRoot := browserDataRoot(browser)
	if dataRoot == "" {
		return profiles
	}
	b, e := os.ReadFile(filepath.Join(dataRoot, "Local State"))
	if e != nil {
		return profiles
	}
	var doc struct {
		Profile struct {
			InfoCache map[string]struct {
				Name string `json:"name"`
			} `json:"info_cache"`
		} `json:"profile"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return profiles
	}
	keys := []string{"Default"}
	for i := 1; i < 101; i++ {
		keys = append(keys, fmt.Sprintf("Profile %d", i))
	}
	for _, dir := range keys {
		p, ok := doc.Profile.InfoCache[dir]
		if !ok || !allowedProfileDir.MatchString(dir) {
			continue
		}
		if s, e := os.Stat(filepath.Join(dataRoot, dir)); e != nil || !s.IsDir() {
			continue
		}
		name := strings.TrimSpace(p.Name)
		if name == "" {
			name = dir
		}
		if len(name) > 90 {
			name = name[:90]
		}
		profiles = append(profiles, BrowserProfile{ID: "personal:" + dir, Name: name + " (sessão isolada)", Kind: "personal-isolated", Notice: "A seleção cria uma sessão independente identificada por este perfil. Não copia cookies ou senhas; faça login uma vez se necessário."})
	}
	return profiles
}
func discoverBrowsers() []BrowserChoice {
	a := []BrowserChoice{}
	for _, v := range []struct{ id, name string }{{"chrome", "Google Chrome"}, {"edge", "Microsoft Edge"}} {
		exe := findBrowserExe(v.id)
		profiles := listBrowserProfiles(v.id)
		if v.id == "chrome" {
			profiles = append(profiles, BrowserProfile{ID: "live:current", Name: "Chrome JA ABERTO — sessao atual conectada", Kind: "live-existing", Notice: "Usa os logins da sessao aberta. Requer Chrome 144+, Depuracao Remota habilitada em chrome://inspect/#remote-debugging e aprovacao no Chrome. Se voce usa varios perfis, conecta o perfil padrao. Nao copia senhas ou cookies."})
		}
		a = append(a, BrowserChoice{ID: v.id, Name: v.name, Path: exe, Installed: exe != "", Profiles: profiles})
	}
	return a
}
func validBrowserProfile(browser, profile string) bool {
	// Validate against the EXACT same catalog returned by browser_opcoes.
	// v2.4.0 mistakenly used listBrowserProfiles(), which omits the
	// Chrome live:current option appended by discoverBrowsers().
	for _, choice := range discoverBrowsers() {
		if choice.ID != browser || !choice.Installed {
			continue
		}
		for _, p := range choice.Profiles {
			if p.ID == profile {
				return true
			}
		}
	}
	return false
}
func (a *App) getBrowserPreference() BrowserPreference {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.Browser
}
func (a *App) setBrowserPreference(b BrowserPreference) error {
	if (b.Browser == "") != (b.Profile == "") {
		return errors.New("selecione navegador e perfil, ou deixe os dois vazios para perguntar sempre")
	}
	if b.Browser != "" && !validBrowserProfile(b.Browser, b.Profile) {
		return errors.New("navegador ou perfil indisponivel; atualize a lista")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	old := a.cfg.Browser
	a.cfg.Browser = b
	if e := a.saveLocked(); e != nil {
		a.cfg.Browser = old
		return e
	}
	return nil
}
func (a *App) browserOptions() map[string]any {
	p := a.getBrowserPreference()
	return map[string]any{"default": p, "browsers": discoverBrowsers(), "requires_choice": p.Browser == "" || p.Profile == "", "note": "Perfis personal:* sao isolados. Chrome live:current anexa a sessao real JA ABERTA com autorizacao local do Chrome (144+), podendo usar logins existentes; nao copia arquivos de cookies nem senhas. Chrome pode exigir aprovacao presencial a cada nova conexao."}
}
func (a *App) resolveBrowserChoice(browser, profile string) (string, string, error) {
	pref := a.getBrowserPreference()
	if browser == "" {
		browser = pref.Browser
	}
	if profile == "" && browser == pref.Browser {
		profile = pref.Profile
	}
	if browser == "" || profile == "" {
		return "", "", errors.New("SELECAO_NECESSARIA: pergunte ao usuario qual navegador (Chrome ou Edge) e qual perfil deseja usar; use browser_opcoes para mostrar alternativas. Nao escolha por conta propria")
	}
	if !validBrowserProfile(browser, profile) {
		return "", "", errors.New("navegador/perfil nao encontrado; use browser_opcoes e pergunte novamente")
	}
	return browser, profile, nil
}
func browserURL(raw string) (string, error) {
	u, e := url.Parse(raw)
	if e != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return "", errors.New("URL invalida: informe http:// ou https:// sem credenciais embutidas")
	}
	if len(raw) > 2048 || strings.ContainsAny(raw, "\r\n\t") {
		return "", errors.New("URL invalida ou muito longa")
	}
	return raw, nil
}
func safeBrowserUserDir(browser, profile string) (string, error) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		local = os.TempDir()
	}
	sum := sha256.Sum256([]byte(browser + "\x00" + profile))
	dir := filepath.Join(local, "ConsecomMCP", "BrowserProfiles", browser+"-"+hex.EncodeToString(sum[:8]))
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	return dir, nil
}
func freeBrowserPort() (int, error) {
	l, e := net.Listen("tcp4", "127.0.0.1:0")
	if e != nil {
		return 0, e
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
func devtoolsHTTP(port int, path, method string) ([]byte, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, e := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	if e != nil {
		return nil, e
	}
	resp, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DevTools retornou HTTP %d", resp.StatusCode)
	}
	var max int64 = 2 * 1024 * 1024
	return ioReadLimit(resp.Body, max)
}
func (a *App) browserOpen(rawURL, browser, profile string) (any, error) {
	normalized, e := browserURL(rawURL)
	if e != nil {
		return nil, e
	}
	browser, profile, e = a.resolveBrowserChoice(browser, profile)
	if e != nil {
		return a.browserOptions(), e
	}
	a.browserControl.mu.Lock()
	defer a.browserControl.mu.Unlock()
	sess := a.browserControl.session
	if profile == "live:current" {
		// Never start another Chrome process or read cookies: ask the running
		// browser to authorize a DevTools attachment to its live profile.
		if sess == nil || sess.Browser != browser || sess.Profile != profile || sess.BrowserWS == "" {
			ws, er := liveChromeWebSocket()
			if er != nil {
				return nil, er
			}
			sess = &BrowserSession{Browser: browser, Profile: profile, BrowserWS: ws}
			a.browserControl.session = sess
		}
		id, er := createLiveBrowserTarget(sess.BrowserWS)
		if er != nil {
			return nil, er
		}
		sess.Target = id
		ownedTargetAdd(sess, id)
		return browserNavigateLocked(sess, normalized)
	}
	if sess == nil || sess.Browser != browser || sess.Profile != profile || !browserPortOK(sess.Port) {
		dir, er := safeBrowserUserDir(browser, profile)
		if er != nil {
			return nil, er
		}
		port, er := freeBrowserPort()
		if er != nil {
			return nil, er
		}
		exe := findBrowserExe(browser)
		pid, er := launchBrowser(exe, dir, port)
		if er != nil {
			return nil, er
		}
		if er = waitBrowserReady(port, 7*time.Second); er != nil {
			return nil, fmt.Errorf("navegador nao iniciou controle: %w (feche instancias antigas do perfil controlado e tente novamente)", er)
		}
		sess = &BrowserSession{Browser: browser, Profile: profile, Port: port, PID: pid, DataDir: dir}
		a.browserControl.session = sess
	}
	target, er := newBrowserTarget(sess.Port)
	if er != nil {
		return nil, er
	}
	sess.Target = target
	ownedTargetAdd(sess, target)
	return browserNavigateLocked(sess, normalized)
}
func waitBrowserReady(port int, max time.Duration) error {
	end := time.Now().Add(max)
	for time.Now().Before(end) {
		if browserPortOK(port) {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return errors.New("tempo esgotado aguardando porta DevTools local")
}
func browserPortOK(port int) bool {
	if port <= 0 {
		return false
	}
	_, e := devtoolsHTTP(port, "/json/version", "GET")
	return e == nil
}
func newBrowserTarget(port int) (string, error) {
	b, e := devtoolsHTTP(port, "/json/new?about:blank", "PUT")
	if e != nil {
		return "", e
	}
	var x struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if e = json.Unmarshal(b, &x); e != nil {
		return "", e
	}
	if x.ID == "" || x.Type != "page" {
		return "", errors.New("aba DevTools invalida")
	}
	return x.ID, nil
}
func browserCurrent(sess *BrowserSession) (string, error) {
	if sess == nil || sess.Target == "" {
		return "", errors.New("nenhuma aba controlada; abra uma pagina com browser_abrir")
	}
	b, e := devtoolsHTTP(sess.Port, "/json/list", "GET")
	if e != nil {
		return "", e
	}
	var x []struct {
		ID   string `json:"id"`
		WS   string `json:"webSocketDebuggerUrl"`
		Type string `json:"type"`
	}
	if json.Unmarshal(b, &x) != nil {
		return "", errors.New("lista DevTools invalida")
	}
	for _, t := range x {
		if t.ID == sess.Target && t.Type == "page" && strings.HasPrefix(t.WS, fmt.Sprintf("ws://127.0.0.1:%d/", sess.Port)) {
			return t.WS, nil
		}
	}
	return "", errors.New("aba controlada foi fechada; abra uma nova pagina")
}
func (a *App) browserCommand(name string, args map[string]json.RawMessage) (any, error) {
	if name == "browser_opcoes" {
		return a.browserOptions(), nil
	}
	if name == "browser_abas" {
		return a.browserTabs()
	}
	if name == "browser_nova_aba" {
		url, e := argStr(args, "url")
		if e != nil {
			return nil, e
		}
		return a.browserNewTab(url)
	}
	if name == "browser_ativar_aba" {
		id, e := argStr(args, "id")
		if e != nil {
			return nil, e
		}
		return a.browserSwitchTab(id)
	}
	if name == "browser_abrir" {
		url, e := argStr(args, "url")
		if e != nil {
			return nil, e
		}
		return a.browserOpen(url, optionalString(args, "navegador"), optionalString(args, "perfil"))
	}
	a.browserControl.mu.Lock()
	defer a.browserControl.mu.Unlock()
	s := a.browserControl.session
	if s == nil {
		return nil, errors.New("nenhum navegador controlado; execute browser_abrir primeiro")
	}
	switch name {
	case "browser_navegar":
		raw, e := argStr(args, "url")
		if e != nil {
			return nil, e
		}
		url, e := browserURL(raw)
		if e != nil {
			return nil, e
		}
		return browserNavigateLocked(s, url)
	case "browser_ler_pagina":
		return browserSnapshotLocked(s)
	case "browser_clicar":
		sel, e := argStr(args, "seletor")
		if e != nil {
			return nil, e
		}
		return browserClickLocked(s, sel)
	case "browser_digitar":
		sel, e := argStr(args, "seletor")
		if e != nil {
			return nil, e
		}
		v, e := argStr(args, "texto")
		if e != nil {
			return nil, e
		}
		return browserTypeLocked(s, sel, v)
	case "browser_rolar":
		pixels, er := optionalInt(args, "pixels", 650)
		if er != nil {
			return nil, er
		}
		return browserScrollLocked(s, pixels)
	case "browser_capturar_tela":
		return browserScreenshotLocked(s)
	case "browser_fechar_aba":
		var e error
		if s.BrowserWS != "" {
			_, e = cdpBrowserCommand(s.BrowserWS, "Target.closeTarget", map[string]any{"targetId": s.Target}, "")
		} else {
			_, e = devtoolsHTTP(s.Port, "/json/close/"+url.PathEscape(s.Target), "GET")
		}
		if e != nil {
			return nil, e
		}
		ownedTargetRemove(s, s.Target)
		return map[string]any{"status": "aba_fechada"}, nil
	default:
		return nil, errors.New("operacao de navegador desconhecida")
	}
}
func optionalString(m map[string]json.RawMessage, key string) string {
	var s string
	_ = json.Unmarshal(m[key], &s)
	return s
}
func optionalInt(m map[string]json.RawMessage, key string, def int) (int, error) {
	if len(m[key]) == 0 {
		return def, nil
	}
	var x int
	if e := json.Unmarshal(m[key], &x); e != nil {
		return 0, e
	}
	if x > 2500 || x < -2500 {
		return 0, errors.New("rolagem limitada a 2500 pixels")
	}
	return x, nil
}
func (a *App) browserStatus() map[string]any {
	p := a.getBrowserPreference()
	a.browserControl.mu.Lock()
	defer a.browserControl.mu.Unlock()
	x := map[string]any{"default": p, "active": false}
	if s := a.browserControl.session; s != nil && s.Target != "" {
		x["active"] = true
		x["browser"] = s.Browser
		x["profile"] = s.Profile
	}
	return x
}

// BrowserScreenshot is rendered as an MCP image content item (not dumped into text).
type BrowserScreenshot struct{ Data string }

var _ = sync.Mutex{} // standalone browser lock type avoids sharing App file permissions lock
