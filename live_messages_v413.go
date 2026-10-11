package main

import (
 "errors"
 "fmt"
 "strings"
 "time"
 "unicode/utf8"
)

// Manual messages are explicitly written on the loopback dashboard protected
// by its existing CSRF token. They never require a second MCP or OpenAI API.
const maxPanelMessageBytes=2000
const panelMessageCooldown=3*time.Second

func validatePanelMessage(raw string)(string,error){
 msg:=strings.TrimSpace(raw)
 if msg==""{return "",errors.New("digite uma mensagem para o ChatGPT")}
 if !utf8.ValidString(msg){return "",errors.New("mensagem UTF-8 invalida")}
 if len(msg)>maxPanelMessageBytes{return "",fmt.Errorf("mensagem maior que %d bytes",maxPanelMessageBytes)}
 for _,r:=range msg{
  if r<32&&r!='\n'&&r!='\r'&&r!='\t'{return "",errors.New("caracteres de controle nao permitidos")}
  if r==127{return "",errors.New("caracteres de controle nao permitidos")}
 }
 return msg,nil
}
func (b *LiveLab) appendEventLocked(ev LiveEvent)(LiveEvent,error){
 seq:=b.seq+1
 if seq>999999{return LiveEvent{},errors.New("limite do journal atingido")}
 ev.ID=fmt.Sprintf("evt-%06d",seq)
 ev.Sequence=seq
 ev.CreatedAt=time.Now().UTC().Format(time.RFC3339Nano)
 events:=append(append([]LiveEvent(nil),b.events...),ev)
 if len(events)>liveMaxEvents {
  dropped:=events[:len(events)-liveMaxEvents]
  for _,old:=range dropped{
   if old.Kind=="PANEL_MESSAGE"&&old.Delivery=="queued"{
    return LiveEvent{},errors.New("fila de mensagens cheia: mantenha o Consecom Live aberto e aguarde entrega")
   }
  }
  events=events[len(events)-liveMaxEvents:]
 }
 if err:=b.saveLocked(seq,events);err!=nil{return LiveEvent{},err}
 b.seq=seq;b.events=events
 close(b.notify);b.notify=make(chan struct{})
 return ev,nil
}
func (b *LiveLab) enqueuePanelMessage(raw string)(LiveEvent,error){
 msg,err:=validatePanelMessage(raw);if err!=nil{return LiveEvent{},err}
 b.mu.Lock();defer b.mu.Unlock()
 if !b.lastPanelSent.IsZero()&&time.Since(b.lastPanelSent)<panelMessageCooldown{return LiveEvent{},errors.New("aguarde 3 segundos antes de enviar outra mensagem")}
 ev,err:=b.appendEventLocked(LiveEvent{Kind:"PANEL_MESSAGE",Message:msg,Delivery:"queued"})
 if err==nil {b.lastPanelSent=time.Now()}
 return ev,err
}
func (b *LiveLab) claimPanelMessage(id string)(bool,error){
 if !liveEventPattern.MatchString(id){return false,errors.New("identificador invalido")}
 b.mu.Lock();defer b.mu.Unlock()
 for i,ev:=range b.events{
  if ev.ID!=id{continue}
  if ev.Kind!="PANEL_MESSAGE"{return false,errors.New("evento nao e mensagem do painel")}
  if ev.Delivery!="queued"{return false,nil}
  next:=append([]LiveEvent(nil),b.events...)
  next[i].Delivery="claimed"
  next[i].ClaimedAt=time.Now().UTC().Format(time.RFC3339Nano)
  if err:=b.saveLocked(b.seq,next);err!=nil{return false,err}
  b.events=next
  return true,nil
 }
 return false,errors.New("mensagem nao encontrada")
}
func (b *LiveLab) retryPanelMessage(id string)(LiveEvent,error){
 if !liveEventPattern.MatchString(id){return LiveEvent{},errors.New("identificador invalido")}
 b.mu.Lock();defer b.mu.Unlock()
 for i,ev:=range b.events{
  if ev.ID!=id{continue}
  if ev.Kind!="PANEL_MESSAGE"{return LiveEvent{},errors.New("evento nao e mensagem do painel")}
  if ev.Delivery=="queued"{return LiveEvent{},errors.New("mensagem ainda na fila")}
  if ev.Delivery=="host_resolved"{return LiveEvent{},errors.New("o host aceitou a mensagem: nao reenviar automaticamente")}
  if ev.Delivery=="claimed"{
   at,err:=time.Parse(time.RFC3339Nano,ev.ClaimedAt)
   if err==nil&&time.Since(at)<2*time.Minute{return LiveEvent{},errors.New("mensagem em envio; aguarde 2 minutos antes de tentar novamente")}
  }
  if !b.lastPanelSent.IsZero()&&time.Since(b.lastPanelSent)<panelMessageCooldown{return LiveEvent{},errors.New("aguarde 3 segundos antes de reenviar")}
  // Supersede old ID, so the new widget session sees a fresh event and
  // exactly one claim can succeed, even with multiple widgets open.
  latest:=append([]LiveEvent(nil),b.events...)
  latest[i].Delivery="superseded"
  next:=LiveEvent{Kind:"PANEL_MESSAGE",Message:ev.Message,Delivery:"queued"}
  seq:=b.seq+1
  if seq>999999{return LiveEvent{},errors.New("limite do journal atingido")}
  next.ID=fmt.Sprintf("evt-%06d",seq)
  next.Sequence=seq
  next.CreatedAt=time.Now().UTC().Format(time.RFC3339Nano)
  latest=append(latest,next)
  if len(latest)>liveMaxEvents {latest=latest[len(latest)-liveMaxEvents:]}
  if err:=b.saveLocked(seq,latest);err!=nil{return LiveEvent{},err}
  b.seq=seq;b.events=latest;b.lastPanelSent=time.Now()
  close(b.notify);b.notify=make(chan struct{})
  return next,nil
 }
 return LiveEvent{},errors.New("mensagem nao encontrada")
}
func (b *LiveLab) panelOutbox()map[string]any{
 b.mu.Lock();defer b.mu.Unlock()
 messages:=make([]LiveEvent,0,12)
 for i:=len(b.events)-1;i>=0&&len(messages)<12;i--{
  if b.events[i].Kind=="PANEL_MESSAGE"{messages=append(messages,b.events[i])}
 }
 return map[string]any{
  "messages":messages,"count":len(messages),
  "note":"queued=aguardando widget; claimed=tentativa em andamento; host_resolved=host aceitou, nao prova leitura pelo modelo",
 }
}
