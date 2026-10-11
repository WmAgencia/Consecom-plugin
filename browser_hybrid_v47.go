package main

import "errors"

// The selected Chrome/Edge managed profile is the only default browser route.
// Agent-browser remains available through explicit agent-browser tools; it
// never silently overrides the user's exclusive managed browser choice.
func (a *App) hybridBrowserStatus()(any,error){
 _,_,agentErr:=agentBrowserResolve()
 return map[string]any{
  "primary_engine":"consecom-cdp",
  "fallback_engine":"none",
  "agent_browser_installed":agentErr==nil,
  "native_browser":a.browserStatus(),
  "policy":"Chrome ou Edge com perfil Consecom exclusivo; agent-browser apenas em ferramentas explicitas.",
  "reason":"O navegador selecionado no painel e persistido e usado por padrao.",
 },nil
}
func (a *App) hybridBrowserOpen(raw string)(any,error){
 clean,_,err:=agentBrowserURL(raw);if err!=nil{return nil,err}
 pref:=a.getBrowserPreference()
 if pref.Browser==""||pref.Profile==""{
  return nil,errors.New("Nenhum navegador Chrome/Edge disponivel; instale um para usar a sessao exclusiva Consecom")
 }
 result,err:=a.browserOpen(clean,pref.Browser,pref.Profile)
 if err!=nil{return nil,err}
 return map[string]any{"engine":"consecom-cdp","result":result},nil
}
