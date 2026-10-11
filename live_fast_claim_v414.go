package main

import (
 "context"
 "errors"
 "time"
)

const liveBridgeFreshFor = 15*time.Second

// The heartbeat is an observation of an actual MCP tool invocation, not a
// promise that ChatGPT will run a widget when the host suspends it.
func (b *LiveLab) markLivePoll(automatic bool){
 b.mu.Lock()
 now:=time.Now()
 b.lastLivePoll=now
 if automatic {b.lastLiveAutoPoll=now}
 b.mu.Unlock()
}
func (b *LiveLab) connectionStatusLocked() map[string]any {
 now:=time.Now()
 result:=map[string]any{
  "status":"disconnected",
  "automatic_ready":false,
  "threshold_ms":liveBridgeFreshFor.Milliseconds(),
  "note":"Apenas observa chamadas MCP recentes. Nao garante que o host manterá o widget em execução.",
 }
 if !b.lastLivePoll.IsZero(){
  age:=now.Sub(b.lastLivePoll)
  result["last_poll_age_ms"]=age.Milliseconds()
  result["last_poll_at"]=b.lastLivePoll.UTC().Format(time.RFC3339Nano)
  if age<=liveBridgeFreshFor {result["status"]="passive"}
 }
 if !b.lastLiveAutoPoll.IsZero(){
  age:=now.Sub(b.lastLiveAutoPoll)
  result["last_auto_poll_age_ms"]=age.Milliseconds()
  if age<=liveBridgeFreshFor {
   result["status"]="ready"
   result["automatic_ready"]=true
  }
 }
 return result
}
func (b *LiveLab) connectionStatus()map[string]any{
 b.mu.Lock();defer b.mu.Unlock()
 return b.connectionStatusLocked()
}

// An old poll cursor must never strand messages created while auto-send was
// switched off. On every fast poll reclaim QUEUED only (never CLAIMED or
// host_resolved), independent of the caller's after_sequence.
func (b *LiveLab) claimQueuedBatch(max int)([]LiveEvent,error){
 b.mu.Lock();defer b.mu.Unlock()
 updated:=append([]LiveEvent(nil),b.events...)
 claimed:=make([]LiveEvent,0,max)
 stamp:=time.Now().UTC().Format(time.RFC3339Nano)
 for i:=range updated{
  if len(claimed)>=max{break}
  if updated[i].Kind!="PANEL_MESSAGE"||updated[i].Delivery!="queued"{continue}
  updated[i].Delivery="claimed"
  updated[i].ClaimedAt=stamp
  claimed=append(claimed,updated[i])
 }
 if len(claimed)==0{return claimed,nil}
 if err:=b.saveLocked(b.seq,updated);err!=nil{return nil,err}
 b.events=updated
 return claimed,nil
}

// One MCP round trip waits, then atomically claims every queued PANEL_MESSAGE
// *before* the response returns to the widget. Only returned claimed_ids may
// be submitted to sendFollowUpMessage; other widgets can never dispatch them.
// The legacy separate claim tool stays available for old MCP Apps resources.
func (b *LiveLab) waitAndClaim(ctx context.Context, after uint64, ms int)(map[string]any,error){
 b.markLivePoll(true)
 // Recover older queued messages immediately, even when after_sequence has
 // already moved past them in a previously passive widget.
 pending,err:=b.claimQueuedBatch(8)
 if err!=nil{return nil,errors.New("falha ao persistir fila pendente: "+err.Error())}
 if len(pending)>0{
  ids:=make([]string,0,len(pending))
  for _,ev:=range pending{ids=append(ids,ev.ID)}
  b.mu.Lock();sequence:=b.seq;b.mu.Unlock()
  return map[string]any{"events":pending,"claimed_ids":ids,"sequence":sequence,
   "automatic_ready":true,"waited_ms":int64(0),"gap":false},nil
 }
 view:=b.wait(ctx,after,ms)
 events,ok:=view["events"].([]LiveEvent)
 if !ok{return view,nil}
 claimedIDs:=make([]string,0,len(events))
 for i:=range events{
  ev:=events[i]
  if ev.Kind!="PANEL_MESSAGE"||ev.Delivery!="queued"{continue}
  claimed,err:=b.claimPanelMessage(ev.ID)
  if err!=nil{return nil,errors.New("falha ao persistir reivindicacao: "+err.Error())}
  if claimed{claimedIDs=append(claimedIDs,ev.ID)}
  if current,exists:=b.get(ev.ID);exists{events[i]=current}
 }
 view["events"]=events
 view["claimed_ids"]=claimedIDs
 view["automatic_ready"]=true
 return view,nil
}
