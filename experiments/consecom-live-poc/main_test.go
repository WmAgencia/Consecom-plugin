package main

import (
 "encoding/json"
 "io"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func testLiveApp(t *testing.T) *app {
 t.Helper()
 a,err:=newApp(strings.Repeat("a",32),"http://127.0.0.1:8793")
 if err!=nil {t.Fatal(err)}
 return a
}
func TestLiveAutoFollowupEnabledAndSafe(t *testing.T) {
 a:=testLiveApp(t)
 html,err:=a.renderWidget()
 if err!=nil{t.Fatal(err)}
 required:=[]string{
  "id=\"auto\" checked", "window.openai.sendFollowUpMessage",
  "new EventSource(", "requestTurn(ev,false)", "MAX_ATTEMPTS = 10",
  "MIN_INTERVAL_MS = 30000", "created<startedAt", "ev.id",
 }
 for _,part:=range required {if !strings.Contains(html,part){t.Errorf("widget missing %q",part)}}
 if strings.Contains(html,"document.querySelector('button').click()"){t.Fatal("do not simulate clicks")}
}
func TestLiveBrokerEventsSyntheticOnly(t *testing.T) {
 a:=testLiveApp(t)
 e:=a.broker.emit()
 if e.Kind!="BUILD_FAILED" || !strings.Contains(e.Message,"ficticio") {t.Fatalf("unsafe event %+v",e)}
 if ev,ok:=a.broker.get(e.ID);!ok||ev.ID!=e.ID {t.Fatal("event could not be looked up")}
 if a.broker.count()!=1 {t.Fatal("broker count mismatch")}
}
func TestLiveEndpointsProtected(t *testing.T){
 a:=testLiveApp(t)
 req:=httptest.NewRequest(http.MethodPost,"/demo/emit",nil)
 w:=httptest.NewRecorder()
 a.mux().ServeHTTP(w,req)
 if w.Code!=401 {t.Fatalf("emit accepted without token: %d",w.Code)}
 req=httptest.NewRequest(http.MethodGet,"/events?channel=bad",nil)
 w=httptest.NewRecorder()
 a.mux().ServeHTTP(w,req)
 if w.Code!=403 {t.Fatalf("SSE accepted wrong channel: %d",w.Code)}
}
func TestLiveMCPWidgetResource(t *testing.T){
 a:=testLiveApp(t)
 input:=`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"ui://consecom/live-poc-v1.html"}}`
 req:=httptest.NewRequest(http.MethodPost,"/mcp",strings.NewReader(input))
 req.Header.Set("Authorization","Bearer "+a.token)
 req.Header.Set("Content-Type","application/json")
 w:=httptest.NewRecorder()
 a.mux().ServeHTTP(w,req)
 if w.Code!=200 {t.Fatalf("status %d",w.Code)}
 b,_:=io.ReadAll(w.Body)
 var payload struct{Result struct{Contents []struct{Text string `json:"text"`} `json:"contents"`}`json:"result"`}
 if err:=json.Unmarshal(b,&payload);err!=nil{t.Fatal(err)}
 if len(payload.Result.Contents)!=1 || !strings.Contains(payload.Result.Contents[0].Text,"Solicitar continuação automaticamente") {t.Fatalf("widget missing: %s",b)}
}

func TestLiveReadOnlyMCPDoesNotRequireStaticToken(t *testing.T) {
 a:=testLiveApp(t)
 req:=httptest.NewRequest(http.MethodPost,"/mcp",strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`))
 req.Header.Set("Content-Type","application/json")
 w:=httptest.NewRecorder()
 a.mux().ServeHTTP(w,req)
 if w.Code!=200 {t.Fatalf("anonymous synthetic-only MCP rejected: %d",w.Code)}
 body:=w.Body.String()
 if !strings.Contains(body,"consecom_live_abrir") || strings.Contains(body,"consecom_live_emitir_teste") {t.Fatalf("unexpected tools: %s",body)}
}
