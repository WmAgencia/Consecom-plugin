package main

import (
 "crypto/subtle"
 "encoding/json"
 "net/http"
 "strings"
 "time"
)

// Host observations record only synthetic event IDs and API acceptance/errors.
// They CANNOT independently determine if ChatGPT actually began a new turn.
type hostObservation struct{
 EventID string
 Mode string
 Status string
 Error string
 Time string
}
func (a *app) channelAllowed(r *http.Request) bool{
 provided:=r.URL.Query().Get("channel")
 return len(provided)==len(a.channel)&&subtle.ConstantTimeCompare([]byte(provided),[]byte(a.channel))==1
}
func (a *app) recordObservation(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Access-Control-Allow-Origin","*")
 w.Header().Set("Access-Control-Allow-Methods","POST, OPTIONS")
 w.Header().Set("Access-Control-Allow-Headers","Content-Type")
 w.Header().Set("Cache-Control","no-store")
 if r.Method=="OPTIONS"{w.WriteHeader(http.StatusNoContent);return}
 if r.Method!="POST"{http.Error(w,"method not allowed",405);return}
 if !a.channelAllowed(r){http.Error(w,"forbidden",403);return}
 var p struct{EventID,Mode,Status,Error string}
 dec:=json.NewDecoder(http.MaxBytesReader(w,r.Body,1200))
 dec.DisallowUnknownFields()
 if err:=dec.Decode(&p);err!=nil{http.Error(w,"bad observation",400);return}
 validStatus:=map[string]bool{"attempt":true,"host_resolved":true,"host_rejected":true,"unavailable":true,"pending":true}
 validMode:=p.Mode=="auto"||p.Mode=="manual"
 if !validStatus[p.Status]||!validMode||!reverseEventID.MatchString(p.EventID) || len(p.Error)>180{
  http.Error(w,"invalid observation",400);return
 }
 p.Error=strings.TrimSpace(p.Error)
 ob:=hostObservation{EventID:p.EventID,Mode:p.Mode,Status:p.Status,Error:p.Error,Time:time.Now().UTC().Format(time.RFC3339Nano)}
 a.broker.mu.Lock()
 a.broker.observations=append(a.broker.observations,ob)
 if len(a.broker.observations)>70{a.broker.observations=a.broker.observations[len(a.broker.observations)-70:]}
 a.broker.mu.Unlock()
 jsonResponse(w,200,map[string]any{"ok":true})
}
func (a *app) listObservations(w http.ResponseWriter,r *http.Request){
 w.Header().Set("Access-Control-Allow-Origin","*")
 if r.Method!="GET"{http.Error(w,"method not allowed",405);return}
 if !a.channelAllowed(r){http.Error(w,"forbidden",403);return}
 a.broker.mu.Lock()
 items:=append([]hostObservation(nil),a.broker.observations...)
 a.broker.mu.Unlock()
 jsonResponse(w,200,map[string]any{"entries":items,"count":len(items),
  "limitation":"host_resolved means the API promise resolved, NOT that a ChatGPT model turn began"})
}
