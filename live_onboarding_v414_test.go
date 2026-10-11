package main

import (
 "strings"
 "testing"
)

func TestConsecomSessionOnboardingHasCopyablePrompt(t *testing.T){
 expected:=[]string{
  "id=\"consecomOnboarding\"",
  "id=\"consecomSetupPrompt\"",
  "id=\"consecomSetupCopyStatus\"",
  "onclick=\"copyConsecomSetupPrompt()\"",
  "async function copyConsecomSetupPrompt()",
  "navigator.clipboard.writeText(prompt)",
  "consecom_live_abrir com modo=\"mensagens\"",
  "consecom_chat_responder",
  "consecom_chat_historico",
  "atualizacao_status",
  "Permitir envio automático",
  "não instala o programa",
 }
 for _,term:=range expected{
  if !strings.Contains(uiHTML,term){t.Errorf("missing onboarding text or control %q",term)}
 }
 for _,mustNotHave:=range []string{
  "runConsecomTenTests()",
  "liveBenchButton",
  "▶ Testar 10x",
 }{
  if strings.Contains(uiHTML,mustNotHave){t.Errorf("obsolete 10x control: %q",mustNotHave)}
 }
}

func TestConsecomSessionToolsAndWidgetStillRegistered(t *testing.T){
 tools:=liveToolSpecs()
 seen:=map[string]bool{}
 for _,spec:=range tools{seen[spec.Name]=true}
 for _,name:=range []string{
  "consecom_live_abrir",
  "consecom_live_aguardar_e_assumir",
  "consecom_chat_responder",
  "consecom_chat_historico",
 }{
  if !seen[name]{t.Fatalf("onboarding referenced missing MCP tool %s",name)}
 }
 if !strings.Contains(liveWidgetHTML,"Permitir envio automático"){
  t.Fatal("Live automatic receive control must stay available")
 }
}
