package main

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "regexp"
 "time"
)

// Experimental synthetic-only event queue. No PC data or ChatGPT messages.
const maxReverseWait = 12000 * time.Millisecond
const maxJournalBytes = 128 << 10
var reverseEventID = regexp.MustCompile("^evt-[0-9]{6}$")

type eventJournal struct {
 Version int
 Sequence uint64
 Events []event
}
func (b *broker) restoreJournal(path string) error {
 if path==""{return errors.New("journal path is empty")}
 if !filepath.IsAbs(path){return errors.New("journal path must be absolute")}
 data,err:=os.ReadFile(path)
 if errors.Is(err,os.ErrNotExist){
  b.mu.Lock();b.journal=path;b.mu.Unlock()
  return nil
 }
 if err!=nil{return fmt.Errorf("open event journal: %w",err)}
 if len(data)>maxJournalBytes{return errors.New("event journal too large")}
 var record eventJournal
 if err:=json.Unmarshal(data,&record);err!=nil{return fmt.Errorf("invalid event journal: %w",err)}
 if record.Version!=1||len(record.Events)>maxEvents{return errors.New("event journal version or size invalid")}
 var previous uint64
 for _,ev:=range record.Events{
  if !reverseEventID.MatchString(ev.ID)||ev.Sequence<=previous||ev.Sequence>record.Sequence||len(ev.Message)>512{
   return errors.New("event journal contains invalid sequence")
  }
  previous=ev.Sequence
 }
 b.mu.Lock()
 defer b.mu.Unlock()
 if len(b.events)!=0||b.seq!=0{return errors.New("cannot restore journal after producing events")}
 b.seq=record.Sequence
 b.events=append([]event(nil),record.Events...)
 b.journal=path
 return nil
}
func saveEventJournal(path string,seq uint64,events []event) error {
 record:=eventJournal{Version:1,Sequence:seq,Events:events}
 raw,err:=json.Marshal(record);if err!=nil{return err}
 if len(raw)>maxJournalBytes{return errors.New("event journal exceeds limit")}
 folder:=filepath.Dir(path)
 if err:=os.MkdirAll(folder,0700);err!=nil{return err}
 out,err:=os.CreateTemp(folder,".live-events-*.tmp");if err!=nil{return err}
 defer os.Remove(out.Name())
 if err:=out.Chmod(0600);err!=nil{out.Close();return err}
 if _,err:=out.Write(raw);err!=nil{out.Close();return err}
 if err:=out.Sync();err!=nil{out.Close();return err}
 if err:=out.Close();err!=nil{return err}
 return os.Rename(out.Name(),path)
}
func (b *broker) emitWithError() (event,error) {
 b.mu.Lock()
 defer b.mu.Unlock()
 seq:=b.seq+1
 ev:=event{
  ID:fmt.Sprintf("evt-%06d",seq),
  Sequence:seq,
  Kind:"BUILD_FAILED",
  Message:"Erro de build de demonstracao: modulo ficticio ausente",
  CreatedAt:time.Now().UTC().Format(time.RFC3339Nano),
 }
 candidates:=append(append([]event(nil),b.events...),ev)
 if len(candidates)>maxEvents {candidates=candidates[len(candidates)-maxEvents:]}
 if b.journal!=""{
  if err:=saveEventJournal(b.journal,seq,candidates);err!=nil{return event{},err}
 }
 b.seq=seq
 b.events=candidates
 close(b.notify)
 b.notify=make(chan struct{})
 for ch:=range b.listeners{select{case ch<-ev:default:}}
 return ev,nil
}
type eventWaitResult struct {
 Status string
 Event *event
 AfterSequence uint64
 CurrentSequence uint64
 Gap bool
 WaitedMS int64
}
func (w eventWaitResult) asMap() map[string]any{
 result:=map[string]any{
  "status":w.Status,"after_sequence":w.AfterSequence,
  "current_sequence":w.CurrentSequence,"gap":w.Gap,
  "waited_ms":w.WaitedMS,
 }
 if w.Event!=nil {result["event"]=w.Event}
 return result
}
// No lost wake-ups: snapshot and notification channel are read under one lock.
// Tool-call timeout cannot autonomously start a new ChatGPT turn.
func (b *broker) waitAfter(ctx context.Context, after uint64, wait time.Duration) eventWaitResult {
 if wait<0{wait=0}
 if wait>maxReverseWait{wait=maxReverseWait}
 start:=time.Now()
 timer:=time.NewTimer(wait);defer timer.Stop()
 for {
  b.mu.Lock()
  seq:=b.seq
  var next *event
  var gap bool
  for i:=range b.events{
   if b.events[i].Sequence>after{
    val:=b.events[i]
    next=&val
    gap=val.Sequence>after+1
    break
   }
  }
  ch:=b.notify
  b.mu.Unlock()
  if next!=nil{return eventWaitResult{
   Status:"event",Event:next,AfterSequence:after,CurrentSequence:seq,Gap:gap,WaitedMS:time.Since(start).Milliseconds(),
  }}
  if wait==0{return eventWaitResult{Status:"timeout",AfterSequence:after,CurrentSequence:seq}}
  select{
  case <-ctx.Done():return eventWaitResult{Status:"cancelled",AfterSequence:after,CurrentSequence:seq,WaitedMS:time.Since(start).Milliseconds()}
  case <-timer.C:return eventWaitResult{Status:"timeout",AfterSequence:after,CurrentSequence:seq,WaitedMS:time.Since(start).Milliseconds()}
  case <-ch:
  }
 }
}
func defaultJournalLocation()(string,error){
 folder,err:=os.UserConfigDir();if err!=nil{return "",err}
 return filepath.Join(folder,"ConsecomLivePOC","events.json"),nil
}
func (b *broker) journalState() map[string]any {
 b.mu.Lock();defer b.mu.Unlock()
 first:=uint64(0)
 if len(b.events)>0{first=b.events[0].Sequence}
 return map[string]any{
  "status":"ready","mode":"synthetic-only","sequence":b.seq,
  "retained_events":len(b.events),"oldest_sequence":first,
  "persisted":b.journal!="",
 }
}
func parseWaitArgs(raw json.RawMessage)(uint64,time.Duration,error){
 var payload map[string]int64
 if len(raw)==0 {raw=[]byte("{}")}
 if err:=json.Unmarshal(raw,&payload);err!=nil{return 0,0,errors.New("JSON de espera invalido")}
 after,wait:=payload["after_sequence"],payload["wait_ms"]
 if after<0||wait<0||wait>12000{
  return 0,0,errors.New("after_sequence >=0 e wait_ms entre 0 e 12000")
 }
 return uint64(after),time.Duration(wait)*time.Millisecond,nil
}
