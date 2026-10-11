package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "sort"
 "strings"
 "time"
 "unicode/utf8"
)

// Responses are written by the ChatGPT model through the existing authenticated
// MCP tools. The Windows dashboard ONLY reads this locally persisted transcript.
const maxChatReplyBytes=6000
const maxChatReplies=120
const maxChatReplyFileBytes=850000

type LiveChatReply struct {
 ID string `json:"id"`
 EventID string `json:"event_id"`
 Text string `json:"text"`
 Kind string `json:"kind"`
 CreatedAt string `json:"created_at"`
 DedupeKey string `json:"dedupe_key,omitempty"`
}
type liveReplyFile struct {
 Version int `json:"version"`
 Sequence uint64 `json:"sequence"`
 Replies []LiveChatReply `json:"replies"`
}
type LiveChatEntry struct {
 ID string `json:"id"`
 EventID string `json:"event_id"`
 Role string `json:"role"`
 Kind string `json:"kind"`
 Text string `json:"text"`
 CreatedAt string `json:"created_at"`
 Status string `json:"status"`
 DispatchedAt string `json:"dispatched_at,omitempty"`
 ClaimedAt string `json:"claimed_at,omitempty"`
}

func (b *LiveLab) loadChatReplies()error{
 b.replyPath=filepath.Join(filepath.Dir(b.journal),"live-chat-replies.json")
 raw,err:=os.ReadFile(b.replyPath)
 if errors.Is(err,os.ErrNotExist){return nil}
 if err!=nil{return err}
 if len(raw)>maxChatReplyFileBytes{return errors.New("arquivo de respostas Consecom excede limite")}
 var data liveReplyFile
 if err=json.Unmarshal(raw,&data);err!=nil{return fmt.Errorf("respostas Consecom invalidas: %w",err)}
 if data.Version!=1||len(data.Replies)>maxChatReplies{return errors.New("formato invalido de respostas Consecom")}
 var last uint64
 for _,r:=range data.Replies{
  if !liveEventPattern.MatchString(r.EventID)||!strings.HasPrefix(r.ID,"rsp-")||r.Kind!="andamento"&&r.Kind!="final"||!utf8.ValidString(r.Text)||len(r.Text)>maxChatReplyBytes{
   return errors.New("resposta Consecom invalida")
  }
  var n uint64
  if _,err=fmt.Sscanf(r.ID,"rsp-%06d",&n);err!=nil||n<=last||n>data.Sequence{return errors.New("ordem de respostas invalida")}
  last=n
 }
 b.replySeq=data.Sequence
 b.replies=append([]LiveChatReply(nil),data.Replies...)
 return nil
}
func (b *LiveLab) saveChatRepliesLocked(seq uint64,replies []LiveChatReply)error{
 raw,err:=json.Marshal(liveReplyFile{Version:1,Sequence:seq,Replies:replies});if err!=nil{return err}
 if len(raw)>maxChatReplyFileBytes{return errors.New("limite de armazenamento do chat atingido")}
 dir:=filepath.Dir(b.replyPath)
 if err=os.MkdirAll(dir,0700);err!=nil{return err}
 file,err:=os.CreateTemp(dir,"chat-*.tmp");if err!=nil{return err}
 defer os.Remove(file.Name())
 if err=file.Chmod(0600);err!=nil{file.Close();return err}
 if _,err=file.Write(raw);err!=nil{file.Close();return err}
 if err=file.Close();err!=nil{return err}
 return os.Rename(file.Name(),b.replyPath)
}
func validateChatReply(text,kind string)error{
 if kind!="final"&&kind!="andamento"{return errors.New("tipo deve ser andamento ou final")}
 if text==""||strings.TrimSpace(text)==""{return errors.New("texto da resposta vazio")}
 if !utf8.ValidString(text)||len(text)>maxChatReplyBytes{return errors.New("resposta deve ser UTF-8 de ate 6000 bytes")}
 for _,r:=range text{if (r<32&&r!=10&&r!=9&&r!=13)||r==127{return errors.New("resposta contem controle invalido")}}
 return nil
}
func (b *LiveLab) publishChatReply(eventID,text,kind,key string)(LiveChatReply,error){
 if !liveEventPattern.MatchString(eventID){return LiveChatReply{},errors.New("id de mensagem invalido")}
 text=strings.TrimSpace(text)
 if err:=validateChatReply(text,kind);err!=nil{return LiveChatReply{},err}
 if len(key)>80{return LiveChatReply{},errors.New("chave de idempotencia excede limite")}
 for _,c:=range key{if (c<'a'||c>'z')&&(c<'A'||c>'Z')&&(c<'0'||c>'9')&&c!='-'&&c!='_'{return LiveChatReply{},errors.New("chave de idempotencia invalida")}}
 b.mu.Lock();defer b.mu.Unlock()
 var found *LiveEvent
 for i:=range b.events{if b.events[i].ID==eventID&&b.events[i].Kind=="PANEL_MESSAGE"{found=&b.events[i];break}}
 if found==nil{return LiveChatReply{},errors.New("mensagem do painel nao encontrada")}
 if found.Delivery=="queued"||found.Delivery=="superseded"{
  return LiveChatReply{},errors.New("mensagem ainda nao entregue ao ChatGPT")
 }
 if key!=""{
  for _,reply:=range b.replies{if reply.EventID==eventID&&reply.DedupeKey==key{return reply,nil}}
 }
 if kind=="final"{
  for _,reply:=range b.replies{if reply.EventID==eventID&&reply.Kind=="final"{return LiveChatReply{},errors.New("resposta final ja publicada para esta mensagem")}}
 }
 seq:=b.replySeq+1
 if seq>999999{return LiveChatReply{},errors.New("limite de respostas atingido")}
 reply:=LiveChatReply{
  ID:fmt.Sprintf("rsp-%06d",seq),EventID:eventID,Text:text,Kind:kind,
  CreatedAt:time.Now().UTC().Format(time.RFC3339Nano),DedupeKey:key,
 }
 newReplies:=append(append([]LiveChatReply(nil),b.replies...),reply)
 if len(newReplies)>maxChatReplies{newReplies=newReplies[len(newReplies)-maxChatReplies:]}
 if err:=b.saveChatRepliesLocked(seq,newReplies);err!=nil{return LiveChatReply{},err}
 b.replySeq=seq;b.replies=newReplies
 if kind=="final"&&found.Delivery!="answered"{
  next:=append([]LiveEvent(nil),b.events...)
  for i:=range next{if next[i].ID==eventID{next[i].Delivery="answered";break}}
  // The response is already safely stored. If this status write fails,
  // keep the persisted reply, and the UI derives answered from that reply.
  if err:=b.saveLocked(b.seq,next);err==nil{b.events=next}
 }
 return reply,nil
}
func (b *LiveLab) chatHistory()map[string]any{
 b.mu.Lock();defer b.mu.Unlock()
 records:=make([]LiveChatEntry,0,len(b.events)+len(b.replies))
 answered:=map[string]bool{}
 for _,reply:=range b.replies{if reply.Kind=="final"{answered[reply.EventID]=true}}
 for _,ev:=range b.events{
  if ev.Kind!="PANEL_MESSAGE"{continue}
  status:=ev.Delivery
  if answered[ev.ID]{status="answered"}
  records=append(records,LiveChatEntry{ID:ev.ID,EventID:ev.ID,Role:"user",Kind:"message",Text:ev.Message,CreatedAt:ev.CreatedAt,Status:status,DispatchedAt:ev.FollowUpAt,ClaimedAt:ev.ClaimedAt})
 }
 for _,reply:=range b.replies{
  records=append(records,LiveChatEntry{ID:reply.ID,EventID:reply.EventID,Role:"assistant",Kind:reply.Kind,Text:reply.Text,CreatedAt:reply.CreatedAt,Status:"published"})
 }
 sort.SliceStable(records,func(i,j int)bool{return records[i].CreatedAt<records[j].CreatedAt})
 return map[string]any{
  "messages":records,"total":len(records),"reply_limit_bytes":maxChatReplyBytes,
  "connection":b.connectionStatusLocked(),
  "bridge":"ChatGPT Chat via existing MCP Apps widget; no API billing",
  "note":"Messages appear when the ChatGPT model calls consecom_chat_responder; model replies are not scraped automatically.",
 }
}
