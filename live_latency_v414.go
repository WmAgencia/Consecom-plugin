package main

import (
 "errors"
 "sort"
 "strings"
 "time"
)

// All diagnostic data is stored with the existing local Live journal.
// No message text, credentials or external telemetry is sent.
func liveObservedAt(raw string)(string,error){
 now:=time.Now().UTC()
 if raw==""{return now.Format(time.RFC3339Nano),nil}
 t,err:=time.Parse(time.RFC3339Nano,raw)
 if err!=nil{return "",errors.New("observed_at deve ser RFC3339")}
 // Widget and server run on the same Windows PC. A wildly skewed/stale clock
 // is unreliable; fall back to server receipt time rather than fake precision.
 if d:=now.Sub(t);d>10*time.Minute||d< -10*time.Minute{
  return now.Format(time.RFC3339Nano),nil
 }
 return t.UTC().Format(time.RFC3339Nano),nil
}

func liveStageMS(from,to string)(int64,bool){
 if from==""||to==""{return 0,false}
 a,e:=time.Parse(time.RFC3339Nano,from);if e!=nil{return 0,false}
 b,e:=time.Parse(time.RFC3339Nano,to);if e!=nil||b.Before(a){return 0,false}
 return b.Sub(a).Milliseconds(),true
}

func (b *LiveLab) markPanelDisplayed(eventID,replyID,observedAt string)(bool,error){
 if !liveEventPattern.MatchString(eventID)||!strings.HasPrefix(replyID,"rsp-")||len(replyID)!=10{
  return false,errors.New("identificador invalido")
 }
 stamp,err:=liveObservedAt(observedAt);if err!=nil{return false,err}
 b.mu.Lock();defer b.mu.Unlock()
 valid:=false
 for _,r:=range b.replies{if r.ID==replyID&&r.EventID==eventID&&r.Kind=="final"{valid=true;break}}
 if !valid{return false,errors.New("resposta final nao encontrada")}
 for i,ev:=range b.events{
  if ev.ID!=eventID||ev.Kind!="PANEL_MESSAGE"{continue}
  if ev.PanelDisplayedAt!=""{return false,nil}
  next:=append([]LiveEvent(nil),b.events...)
  next[i].PanelDisplayedAt=stamp
  if err:=b.saveLocked(b.seq,next);err!=nil{return false,err}
  b.events=next
  return true,nil
 }
 return false,errors.New("mensagem original nao encontrada")
}

// A final tool response is the last model milestone, not host_resolved.
// Host acceptance may overlap model execution; phases must not be summed.
func (b *LiveLab) latencyReport()map[string]any{
 b.mu.Lock();defer b.mu.Unlock()
 finals:=make(map[string]LiveChatReply)
 for _,r:=range b.replies{if r.Kind=="final"{finals[r.EventID]=r}}
 samples:=make([]map[string]any,0,len(b.events))
 totals:=make([]int64,0)
 var largest string
 var largestMS int64
 for _,ev:=range b.events{
  if ev.Kind!="PANEL_MESSAGE"{continue}
  reply:=finals[ev.ID]
  stamps:=map[string]string{
   "sent_at":ev.CreatedAt,"claimed_at":ev.ClaimedAt,
   "followup_at":ev.FollowUpAt,"host_resolved_at":ev.HostResolvedAt,
   "reply_at":reply.CreatedAt,"panel_displayed_at":ev.PanelDisplayedAt,
  }
  periods:=map[string]int64{}
  add:=func(key,a,c string){if n,ok:=liveStageMS(a,c);ok{periods[key]=n}}
  add("queue_ms",ev.CreatedAt,ev.ClaimedAt)
  add("claim_to_followup_ms",ev.ClaimedAt,ev.FollowUpAt)
  add("followup_to_host_ms",ev.FollowUpAt,ev.HostResolvedAt)
  add("host_to_reply_ms",ev.HostResolvedAt,reply.CreatedAt)
  add("followup_to_reply_ms",ev.FollowUpAt,reply.CreatedAt)
  add("reply_to_display_ms",reply.CreatedAt,ev.PanelDisplayedAt)
  add("total_to_reply_ms",ev.CreatedAt,reply.CreatedAt)
  add("total_to_display_ms",ev.CreatedAt,ev.PanelDisplayedAt)
  if n,ok:=periods["total_to_reply_ms"];ok{totals=append(totals,n)}
  // Only compare non-overlapping known phases; the dispatch-to-reply period
  // includes both host scheduling and model processing, which cannot be split.
  bottleneck:="nao determinado"
  peak:=int64(-1)
  for _,candidate:=range []struct{key,label string}{
   {"queue_ms","fila_ate_widget"},
   {"claim_to_followup_ms","claim_ate_despacho"},
   {"followup_to_reply_ms","despacho_ate_resposta"},
   {"reply_to_display_ms","resposta_ate_painel"},
  }{
   if n,ok:=periods[candidate.key];ok&&n>peak{peak=n;bottleneck=candidate.label}
  }
  if peak>largestMS{largestMS=peak;largest=bottleneck}
  samples=append(samples,map[string]any{
   "event_id":ev.ID,"delivery":ev.Delivery,"timestamps":stamps,
   "durations":periods,"bottleneck":bottleneck,
  })
 }
 sort.Slice(totals,func(i,j int)bool{return totals[i]<totals[j]})
 summary:=map[string]any{
  "completed":len(totals),"observed":len(samples),
  "dominant_observed_phase":largest,
 }
 if len(totals)>0{
  var sum int64
  for _,v:=range totals{sum+=v}
  summary["mean_to_reply_ms"]=float64(sum)/float64(len(totals))
  summary["min_to_reply_ms"]=totals[0]
  summary["max_to_reply_ms"]=totals[len(totals)-1]
  if n:=len(totals);n%2==0{summary["median_to_reply_ms"]=float64(totals[n/2-1]+totals[n/2])/2
  }else{summary["median_to_reply_ms"]=float64(totals[n/2])}
 }
 return map[string]any{
  "version":2,"summary":summary,"messages":samples,
  "note":"Tempos reais de eventos, sem textos. host_resolved não indica fim do modelo; despacho→resposta mistura escalonamento do host e geração do ChatGPT. Campos ausentes não são estimados. panel_displayed_at é observado após renderização no navegador.",
 }
}
