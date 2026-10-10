package main

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// Tab navigation is strictly scoped to targets CREATED by Consecom.
// This never enumerates or commandeers the user's unrelated personal tabs.
func (a *App) browserNewTab(rawURL string) (any, error) {
	a.browserControl.mu.Lock()
	s := a.browserControl.session
	if s == nil {
		a.browserControl.mu.Unlock()
		return nil, errors.New("abra um navegador primeiro")
	}
	browser, profile := s.Browser, s.Profile
	a.browserControl.mu.Unlock()
	return a.browserOpen(rawURL, browser, profile)
}
func (a *App) browserTabs() (any, error) {
	a.browserControl.mu.Lock()
	defer a.browserControl.mu.Unlock()
	s := a.browserControl.session
	if s == nil {
		return map[string]any{"tabs": []any{}}, nil
	}
	tabs := []map[string]any{}
	for _, id := range s.OwnedTargets {
		item := map[string]any{"id": id, "active": id == s.Target}
		if s.BrowserWS == "" {
			b, e := devtoolsHTTP(s.Port, "/json/list", "GET")
			if e == nil {
				var pages []struct {
					ID    string `json:"id"`
					URL   string `json:"url"`
					Title string `json:"title"`
				}
				if json.Unmarshal(b, &pages) == nil {
					for _, p := range pages {
						if p.ID == id {
							item["url"] = p.URL
							item["title"] = p.Title
						}
					}
				}
			}
		}
		tabs = append(tabs, item)
	}
	return map[string]any{"tabs": tabs, "notice": "apenas abas criadas pelo Consecom"}, nil
}
func (a *App) browserSwitchTab(id string) (any, error) {
	a.browserControl.mu.Lock()
	defer a.browserControl.mu.Unlock()
	s := a.browserControl.session
	if s == nil {
		return nil, errors.New("sem sessao ativa")
	}
	valid := false
	for _, owned := range s.OwnedTargets {
		if owned == id {
			valid = true
		}
	}
	if !valid {
		return nil, errors.New("aba nao pertence ao Consecom")
	}
	if s.BrowserWS == "" {
		_, e := devtoolsHTTP(s.Port, "/json/activate/"+url.PathEscape(id), "GET")
		if e != nil {
			return nil, e
		}
	}
	s.Target = id
	return map[string]any{"active_target": id}, nil
}
func ownedTargetAdd(s *BrowserSession, id string) {
	if id == "" {
		return
	}
	for _, v := range s.OwnedTargets {
		if v == id {
			return
		}
	}
	if len(s.OwnedTargets) < 16 {
		s.OwnedTargets = append(s.OwnedTargets, id)
	}
}
func ownedTargetRemove(s *BrowserSession, id string) {
	kept := []string{}
	for _, v := range s.OwnedTargets {
		if v != id {
			kept = append(kept, v)
		}
	}
	s.OwnedTargets = kept
	if s.Target == id {
		s.Target = ""
		if len(kept) > 0 {
			s.Target = kept[len(kept)-1]
		}
	}
}
func sanitizeTabTarget(id string) string { return strings.TrimSpace(id) }
