package main

// dashboardContext exposes only metadata needed by the local dashboard.
// A client declaring itself via MCP is a known source, not proof that its
// IDE is currently online or that cross-IDE task execution is available.
func (a *App) dashboardContext() map[string]any {
 a.mu.Lock()
 defer a.mu.Unlock()
 sources:=make([]map[string]string,0,len(a.cfg.ContextSources))
 for _,source:=range a.cfg.ContextSources {
  sources=append(sources,map[string]string{
   "id":source.ID,"label":source.Label,"origin":source.Origin,
  })
 }
 return map[string]any{
  "sources":sources,
  "source_count":len(a.cfg.ContextSources),
  "note_count":len(a.cfg.ContextNotes),
  "vault_selected":a.cfg.ContextVault!="",
 }
}
