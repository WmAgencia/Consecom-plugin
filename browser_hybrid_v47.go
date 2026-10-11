package main

import "errors"

// Hybrid routing prefers the accessibility-first engine for isolated,
// non-authenticated web navigation. Native CDP is the fallback when the
// optional CLI is absent and a trusted browser/profile was selected.
func (a *App) hybridBrowserStatus()(any,error){
 _,_,e:=agentBrowserResolve()
 active:=a.browserStatus()
 engine:="agent-browser"
 reason:="CLI instalada: prefere refs de acessibilidade e sessoes isoladas"
 if e!=nil{
  engine="consecom-cdp"
  reason="agent-browser ausente: navegador nativo CDP continua disponivel"
 }
 return map[string]any{
  "primary_engine":engine,"fallback_engine":"consecom-cdp",
  "agent_browser_installed":e==nil,"native_browser":active,
  "policy":"Agent-browser primeiro para acessibilidade em paginas comuns; Consecom CDP para perfis autorizados, QA nativo e fallback. Sem troca silenciosa de perfil pessoal.",
  "reason":reason,
 },nil
}
func (a *App) hybridBrowserOpen(raw string)(any,error){
 // Validate both pathways before choosing an engine.
 clean,_,err:=agentBrowserURL(raw);if err!=nil{return nil,err}
 _,_,installedErr:=agentBrowserResolve()
 if installedErr==nil{
  result,err:=a.agentBrowserOpen(clean)
  if err!=nil{return nil,err}
  return map[string]any{"engine":"agent-browser","result":result},nil
 }
 pref:=a.getBrowserPreference()
 if pref.Browser==""||pref.Profile==""{
  return nil,errors.New("Agent-browser nao instalado e nenhum navegador/perfil padrao definido. Instale a CLI ou escolha Chrome/Edge e perfil no painel.")
 }
 result,err:=a.browserOpen(clean,pref.Browser,pref.Profile)
 if err!=nil{return nil,err}
 return map[string]any{"engine":"consecom-cdp","result":result},nil
}
