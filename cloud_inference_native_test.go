package main

import (
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func TestCloudModelAlias(t *testing.T){
 cases:=map[string]string{
  "qwen3:4b":"claude-local-qwen3-4b",
  "gemma3:12b":"claude-local-gemma3-12b",
  "":"",
  ":":"",
 }
 for input,want:=range cases{
  if got:=cloudModelAlias(input);got!=want{t.Errorf("alias(%q)=%q want=%q",input,got,want)}
 }
}
func TestCloudSettingsProtectedFromStatus(t *testing.T){
 app,err:=NewApp(t.TempDir()+"/config.json")
 if err!=nil{t.Fatal(err)}
 key:="cci_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_abcdef"
 if err=app.configureCloudInference(CloudInferenceSettings{Enabled:true,WorkerToken:key});err!=nil{t.Fatal(err)}
 status:=app.cloudInferenceStatus()
 packed,_:=json.Marshal(status)
 if strings.Contains(string(packed),key){t.Fatal("private worker token leaked in status")}
 if status["token_configured"]!=true||status["enabled"]!=true{t.Fatalf("bad status: %v",status)}
 app2,err:=NewApp(app.path)
 if err!=nil{t.Fatal(err)}
 if app2.cloudInferenceSettings().WorkerToken!=key{t.Fatal("worker token did not persist")}
 if err=app2.configureCloudInference(CloudInferenceSettings{Enabled:false});err!=nil{t.Fatal(err)}
 if !validCloudWorkerToken(app2.cloudInferenceSettings().WorkerToken){t.Fatal("blank token wrongly erased the saved token")}
}
func TestCloudStatusRequiresLocalKey(t *testing.T){
 app,err:=NewApp(t.TempDir()+"/config.json")
 if err!=nil{t.Fatal(err)}
 handler:=app.handler()
 request:=httptest.NewRequest("GET","http://127.0.0.1:8765/api/cloud-inference/status",nil)
 missing:=httptest.NewRecorder()
 handler.ServeHTTP(missing,request)
 if missing.Code!=http.StatusForbidden{t.Fatalf("status without local key = %d",missing.Code)}
 request.Header.Set("X-Local-Key",app.csrf)
 allowed:=httptest.NewRecorder()
 handler.ServeHTTP(allowed,request)
 if allowed.Code!=http.StatusOK{t.Fatalf("status with local key = %d",allowed.Code)}
 if strings.Contains(allowed.Body.String(),"worker_token"){t.Fatal("worker token name present in status output")}
}
