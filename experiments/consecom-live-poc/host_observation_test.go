package main

import (
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"
)

func TestHostObservationsOnlySyntheticChannelScoped(t *testing.T) {
 app:=testLiveApp(t)
 ts:=httptest.NewServer(app.mux());defer ts.Close()
 request,err:=http.NewRequest("POST",ts.URL+"/demo/widget-report?channel=wrong",strings.NewReader(`{"EventID":"evt-000001","Mode":"auto","Status":"host_rejected","Error":"no gesture"}`))
 if err!=nil{t.Fatal(err)}
 request.Header.Set("Content-Type","application/json")
 response,err:=http.DefaultClient.Do(request);if err!=nil{t.Fatal(err)}
 response.Body.Close()
 if response.StatusCode!=403{t.Fatalf("unknown channel accepted: %d",response.StatusCode)}
 req,err:=http.NewRequest("POST",ts.URL+"/demo/widget-report?channel="+app.channel,strings.NewReader(`{"EventID":"evt-000001","Mode":"auto","Status":"host_rejected","Error":"Method sendFollowUpMessage called without synchronous user event"}`))
 if err!=nil{t.Fatal(err)}
 req.Header.Set("Content-Type","application/json")
 resp,err:=http.DefaultClient.Do(req);if err!=nil{t.Fatal(err)}
 resp.Body.Close()
 if resp.StatusCode!=200{t.Fatalf("observation not stored: %d",resp.StatusCode)}
 read,err:=http.Get(ts.URL+"/demo/widget-observations?channel="+app.channel);if err!=nil{t.Fatal(err)}
 defer read.Body.Close()
 buf:=make([]byte,2000)
 n,_:=read.Body.Read(buf)
 result:=string(buf[:n])
 if !strings.Contains(result,"host_rejected")||!strings.Contains(result,"evt-000001"){t.Fatalf("missing host rejection: %s",result)}
 if !strings.Contains(result,"NOT that a ChatGPT model turn began"){t.Fatal("missing explicit host result caveat")}
}
