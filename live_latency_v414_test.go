package main

import (
 "encoding/json"
 "net/http/httptest"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func TestLatencyMilestonesPersistAndAreIdempotent(t *testing.T){
 path:=filepath.Join(t.TempDir(),"live-lab-v412.json")
 b,err:=newLiveLab(path);if err!=nil{t.Fatal(err)}
 ev,err:=b.enqueuePanelMessage("segredo nao deve aparecer em metricas");if err!=nil{t.Fatal(err)}
 claimed,err:=b.claimPanelMessage(ev.ID);if err!=nil||!claimed{t.Fatal(err)}
 now:=time.Now().UTC().Format(time.RFC3339Nano)
 if err=b.report(ev.ID,"auto","attempt",now);err!=nil{t.Fatal(err)}
 if err=b.report(ev.ID,"auto","attempt",now);err!=nil{t.Fatal(err)}
 if err=b.report(ev.ID,"auto","host_resolved",time.Now().UTC().Format(time.RFC3339Nano));err!=nil{t.Fatal(err)}
 reply,err:=b.publishChatReply(ev.ID,"Resposta final","final","finish");if err!=nil{t.Fatal(err)}
 ok,err:=b.markPanelDisplayed(ev.ID,reply.ID,time.Now().UTC().Format(time.RFC3339Nano))
 if err!=nil||!ok{t.Fatalf("display: %v %v",ok,err)}
 ok,err=b.markPanelDisplayed(ev.ID,reply.ID,"")
 if err!=nil||ok{t.Fatalf("duplicate display changed record: %v %v",ok,err)}
 if _,err=b.markPanelDisplayed(ev.ID,"rsp-999999","");err==nil{t.Fatal("accepted another reply")}
 report:=b.latencyReport()
 samples:=report["messages"].([]map[string]any)
 if len(samples)!=1{t.Fatal(report)}
 ts:=samples[0]["timestamps"].(map[string]string)
 for _,k:=range []string{"sent_at","claimed_at","followup_at","host_resolved_at","reply_at","panel_displayed_at"}{
  if ts[k]==""{t.Fatalf("missing timestamp %s: %+v",k,ts)}
 }
 durations:=samples[0]["durations"].(map[string]int64)
 if _,ok:=durations["total_to_reply_ms"];!ok{t.Fatal(durations)}
 serialized,_:=json.Marshal(report)
 if strings.Contains(string(serialized),"segredo nao deve"){t.Fatal("diagnostics leak private message content")}
 reopened,err:=newLiveLab(path);if err!=nil{t.Fatal(err)}
 persisted:=reopened.latencyReport()["messages"].([]map[string]any)[0]["timestamps"].(map[string]string)
 if persisted["followup_at"]!=ts["followup_at"]||persisted["panel_displayed_at"]!=ts["panel_displayed_at"]{
  t.Fatal("milestones were not persisted")
 }
}

func TestLatencyReportsPartialAndClockSkew(t *testing.T){
 b,err:=newLiveLab(filepath.Join(t.TempDir(),"lab.json"));if err!=nil{t.Fatal(err)}
 ev,err:=b.enqueuePanelMessage("parcial");if err!=nil{t.Fatal(err)}
 data:=b.latencyReport()["messages"].([]map[string]any)
 if len(data)!=1||data[0]["bottleneck"]!="nao determinado"{t.Fatal(data)}
 if _,err=liveObservedAt("invalid");err==nil{t.Fatal("invalid timestamp accepted")}
 if _,ok:=liveStageMS("2026-01-01T00:00:03Z","2026-01-01T00:00:02Z");ok{t.Fatal("negative duration accepted")}
 if _,err=b.markPanelDisplayed(ev.ID,"rsp-000001","");err==nil{t.Fatal("display before final accepted")}
}

func TestLatencyEndpointIsAuthenticatedAndReadOnly(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 req:=httptest.NewRequest("GET","http://127.0.0.1:8765/api/live/latency",nil)
 rr:=httptest.NewRecorder();a.handler().ServeHTTP(rr,req)
 if rr.Code!=403{t.Fatalf("unauthorized HTTP %d",rr.Code)}
 req=httptest.NewRequest("GET","http://127.0.0.1:8765/api/live/latency",nil)
 req.Header.Set("X-Local-Key",a.csrf)
 rr=httptest.NewRecorder();a.handler().ServeHTTP(rr,req)
 if rr.Code!=200||!strings.Contains(rr.Body.String(),"dominant_observed_phase"){t.Fatalf("diagnostic endpoint HTTP %d %s",rr.Code,rr.Body.String())}
 req=httptest.NewRequest("POST","http://127.0.0.1:8765/api/live/latency",strings.NewReader("{}"))
 req.Header.Set("X-Local-Key",a.csrf)
 rr=httptest.NewRecorder();a.handler().ServeHTTP(rr,req)
 if rr.Code!=405{t.Fatalf("diagnostic POST was allowed %d",rr.Code)}
}
