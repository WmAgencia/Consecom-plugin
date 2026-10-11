package main

import (
 "context"
 "encoding/json"
 "io"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "sync"
 "testing"
 "time"
)

func TestReverseJournalSurvivesRestart(t *testing.T){
 path:=filepath.Join(t.TempDir(),"state","events.json")
 b:=newBroker()
 if err:=b.restoreJournal(path);err!=nil{t.Fatal(err)}
 first,err:=b.emitWithError();if err!=nil{t.Fatal(err)}
 if first.Sequence!=1||first.ID!="evt-000001"{t.Fatalf("first %+v",first)}
 reloaded:=newBroker()
 if err:=reloaded.restoreJournal(path);err!=nil{t.Fatal(err)}
 state:=reloaded.journalState()
 if state["sequence"]!=uint64(1)||state["retained_events"]!=1{t.Fatalf("restart state %+v",state)}
 second,err:=reloaded.emitWithError();if err!=nil{t.Fatal(err)}
 if second.Sequence!=2||second.ID!="evt-000002"{t.Fatalf("second %+v",second)}
 if first.ID==second.ID{t.Fatal("replayed ID after restart")}
 info,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
 if runtime.GOOS!="windows"&&info.Mode().Perm()&0077!=0{t.Fatal("journal should be private to current user")}
}
func TestReverseWaitDetectsNewEventWithNoLostWakeup(t *testing.T){
 b:=newBroker()
 var wg sync.WaitGroup
 wg.Add(1)
 results:=make(chan eventWaitResult,1)
 go func(){defer wg.Done();results<-b.waitAfter(context.Background(),0,1700*time.Millisecond)}()
 time.Sleep(45*time.Millisecond)
 ev:=b.emit()
 got:=<-results
 wg.Wait()
 if got.Status!="event"||got.Event==nil||got.Event.ID!=ev.ID{
  t.Fatalf("lost wakeup or wrong event: %+v",got)
 }
 if got.WaitedMS>1000{t.Fatalf("slow in-process notification: %+v",got)}
}
func TestReverseWaitTimeoutAndNoDuplicate(t *testing.T){
 b:=newBroker()
 first:=b.emit()
 prior:=b.waitAfter(context.Background(),0,0)
 if prior.Status!="event"||prior.Event.ID!=first.ID{t.Fatalf("missing available event: %+v",prior)}
 start:=time.Now()
 got:=b.waitAfter(context.Background(),1,85*time.Millisecond)
 if got.Status!="timeout"||time.Since(start)>900*time.Millisecond{t.Fatalf("wrong timeout: %+v",got)}
 ctx,cancel:=context.WithCancel(context.Background())
 cancel()
 stop:=b.waitAfter(ctx,1,500*time.Millisecond)
 if stop.Status!="cancelled"{t.Fatalf("request cancellation not honored: %+v",stop)}
}
func TestReverseWaitRetentionGapIsReported(t *testing.T){
 b:=newBroker()
 for i:=0;i<maxEvents+7;i++{b.emit()}
 result:=b.waitAfter(context.Background(),0,0)
 if result.Status!="event"||!result.Gap||result.Event.Sequence!=8{t.Fatalf("expected retention gap: %+v",result)}
}
func TestReverseRequestArgumentValidation(t *testing.T){
 after,wait,err:=parseWaitArgs([]byte(`{"after_sequence":7,"wait_ms":155}`))
 if err!=nil||after!=7||wait!=155*time.Millisecond{t.Fatalf("incorrect JSON args after=%d wait=%s err=%v",after,wait,err)}
 for _,raw:=range []string{`{"after_sequence":-1}`,`{"wait_ms":12001}`,`{"wait_ms":"oops"}`}{
  if _,_,e:=parseWaitArgs([]byte(raw));e==nil{t.Errorf("accepted invalid args %s",raw)}
 }
}
func TestReverseHTTPMCPWaitAndEmit(t *testing.T){
 app:=testLiveApp(t)
 server:=httptest.NewServer(app.mux())
 defer server.Close()
 // Simulate ChatGPT's already-active MCP call, without using any model API.
 reqBody:=`{"jsonrpc":"2.0","id":18,"method":"tools/call","params":{"name":"consecom_live_aguardar_evento","arguments":{"after_sequence":0,"wait_ms":3500}}}`
 request,err:=http.NewRequest(http.MethodPost,server.URL+"/mcp",strings.NewReader(reqBody))
 if err!=nil{t.Fatal(err)}
 request.Header.Set("Content-Type","application/json")
 done:=make(chan []byte,1)
 errs:=make(chan error,1)
 started:=time.Now()
 go func(){
  client:=http.Client{Timeout:5*time.Second}
  response,e:=client.Do(request);if e!=nil{errs<-e;return}
  defer response.Body.Close()
  body,e:=io.ReadAll(response.Body);if e!=nil{errs<-e;return}
  done<-body
 }()
 time.Sleep(80*time.Millisecond)
 emitRequest,_:=http.NewRequest(http.MethodPost,server.URL+"/demo/emit",nil)
 emitRequest.Header.Set("Authorization","Bearer "+app.token)
 emitterResp,err:=http.DefaultClient.Do(emitRequest);if err!=nil{t.Fatal(err)}
 defer emitterResp.Body.Close()
 if emitterResp.StatusCode!=201{t.Fatalf("failed emitting event: %d",emitterResp.StatusCode)}
 var reply []byte
 select {
 case reply=<-done:
 case e:=<-errs:t.Fatal(e)
 case <-time.After(4500*time.Millisecond):t.Fatal("MCP long-poll tool did not wake")
 }
 var rpc struct {Result struct {StructuredContent map[string]json.RawMessage `json:"structuredContent"`} `json:"result"`}
 if err:=json.Unmarshal(reply,&rpc);err!=nil{t.Fatal(err)}
 var status string
 if err:=json.Unmarshal(rpc.Result.StructuredContent["status"],&status);err!=nil{t.Fatal(err)}
 var eventObj event
 if err:=json.Unmarshal(rpc.Result.StructuredContent["event"],&eventObj);err!=nil{t.Fatal(err)}
 if status!="event"||eventObj.ID!="evt-000001"{t.Fatalf("bad wake: %s",reply)}
 elapsed:=time.Since(started)
 t.Logf("MCP synthetic event wake after %dms (includes 80ms simulated emit delay)",elapsed.Milliseconds())
 if elapsed>3*time.Second{t.Fatalf("reverse transport too slow: %s",elapsed)}
}
func TestReverseJournalFailurePreventsEvent(t *testing.T){
 dir:=t.TempDir()
 blocker:=filepath.Join(dir,"events.json")
 b:=newBroker()
 if err:=b.restoreJournal(blocker);err!=nil{t.Fatal(err)}
 // Make the destination a directory AFTER registering its location.
 // Atomic rename must fail; no event is published or acknowledged.
 if err:=os.Mkdir(blocker,0700);err!=nil{t.Fatal(err)}
 if ev,err:=b.emitWithError();err==nil||ev.ID!=""{t.Fatal("event was emitted without durable write")}
 if b.count()!=0{t.Fatal("uncommitted event was visible")}
}
func TestReverseSSEReceivesExactEmittedEvent(t *testing.T){
 app:=testLiveApp(t)
 server:=httptest.NewServer(app.mux());defer server.Close()
 client:=http.Client{Timeout:2*time.Second}
 resp,err:=client.Get(server.URL+"/events?channel="+app.channel)
 if err!=nil{t.Fatal(err)}
 defer resp.Body.Close()
 if resp.StatusCode!=200{t.Fatalf("SSE status=%d",resp.StatusCode)}
 result:=make(chan string,1)
 go func(){
  out:=make([]byte,2048)
  // Initial : connected heartbeat, followed by an event frame.
  n,err:=resp.Body.Read(out)
  if err!=nil{result<-"read error: "+err.Error();return}
  data:=string(out[:n])
  if !strings.Contains(data,"event: consecom"){
   n,err=resp.Body.Read(out);if err!=nil{result<-"read error: "+err.Error();return}
   data+=string(out[:n])
  }
  result<-data
 }()
 time.Sleep(25*time.Millisecond)
 ev:=app.broker.emit()
 select{
 case data:=<-result:
  if !strings.Contains(data,"event: consecom")||!strings.Contains(data,ev.ID){t.Fatalf("SSE content mismatch: %q",data)}
 case <-time.After(1800*time.Millisecond):t.Fatal("SSE event not delivered")
 }
}

