package main

import (
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "path/filepath"
 "strings"
 "testing"
)

func TestDesktopRailwayReadOnlyPolicy(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"))
 if err!=nil {t.Fatal(err)}
 for _,name:=range []string{"diagnostico_pc","pastas_autorizadas","listar_arquivos"}{
  result,err:=desktopExecuteReadonly(a,desktopJob{Tool:name,Arguments:json.RawMessage("{}")})
  if err!=nil&&name!="listar_arquivos"{t.Fatalf("read-only %s failed: %v",name,err)}
  _=result
 }
 for _,name:=range []string{"solicitar_gravacao_arquivo","gravar_arquivo_pc","excluir_arquivo_pc","nao_existe"}{
  if _,err:=desktopExecuteReadonly(a,desktopJob{Tool:name,Arguments:json.RawMessage("{}")});err==nil{t.Fatalf("remote write tool %s was allowed",name)}
 }
}

func TestDesktopCloudBridgeSettingsAndPanelSecurity(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"))
 if err!=nil {t.Fatal(err)}
 settings:=a.desktopRelayStatus()
 if settings["enabled"]!=false||settings["connected"]!=false||settings["tunnel_required"]!=false{t.Fatalf("unexpected initial status: %#v",settings)}
 token:="cci_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890_abcdef"
 if err=a.configureCloudInference(CloudInferenceSettings{Enabled:true,WorkerToken:token});err!=nil{t.Fatal(err)}
 request:=httptest.NewRequest("GET","http://127.0.0.1:8765/api/status",nil)
 request.Header.Set("X-Local-Key",a.csrf)
 recorder:=httptest.NewRecorder();a.handler().ServeHTTP(recorder,request)
 if recorder.Code!=200 {t.Fatalf("status HTTP %d",recorder.Code)}
 if strings.Contains(recorder.Body.String(),token){t.Fatal("desktop status leaked worker credential")}
 r:=httptest.NewRequest("POST","http://127.0.0.1:8765/api/tunnel/connect",strings.NewReader("{}"))
 r.Header.Set("X-Local-Key",a.csrf)
 r.Header.Set("Content-Type","application/json")
 w:=httptest.NewRecorder();a.handler().ServeHTTP(w,r)
 if w.Code!=http.StatusGone {t.Fatalf("legacy tunnel endpoint must return 410, got %d",w.Code)}
}

func TestDesktopCloudUIReplacesTunnelPairing(t *testing.T){
 if !strings.Contains(uiHTML,"Conectar ao ChatGPT pelo Railway") {t.Fatal("missing Railway pairing UI")}
 if strings.Contains(uiHTML,"href=\"/cloud-inference\"") {t.Fatal("separate Consecom Cloud sidebar must be removed")}
 if strings.Contains(uiHTML,"'tunnel/connect'")||strings.Contains(uiHTML,"'tunnel/disconnect'") {t.Fatal("old tunnel JS actions still active")}
}

func TestDesktopOutboxKeepsOriginStable(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"))
 if err!=nil{t.Fatal(err)}
 if a.live==nil{t.Fatal("missing live journal")}
 ev,err:=a.live.enqueuePanelMessage("Teste com Railway")
 if err!=nil{t.Fatal(err)}
 expected:=a.desktopExternalID(ev.ID)
 if !strings.HasPrefix(expected,"pc-")||!strings.HasSuffix(expected,ev.ID)||len(expected)!=26{t.Fatalf("unexpected origin %s",expected)}
 pending:=a.live.desktopPendingMessages()
 if len(pending)!=1||pending[0].ID!=ev.ID{t.Fatal("pending outbox not ready")}
 if _,err=a.live.claimPanelMessage(ev.ID);err!=nil{t.Fatal(err)}
 if _,err=a.live.publishChatReply(ev.ID,"Railway recebeu","final","railway-"+ev.ID);err!=nil{t.Fatal(err)}
 if len(a.live.desktopPendingMessages())!=0{t.Fatal("answered events must not remain in outgoing queue")}
}
