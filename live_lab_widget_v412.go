package main

// Uses ChatGPT's authenticated existing MCP Apps bridge, no external endpoints,
// second plugin, API billing or automation of the user's browser.
const liveWidgetHTML = `<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<style>
:root{color-scheme:light dark}body{margin:0;padding:16px;font:14px/1.5 system-ui,sans-serif;background:#0f1721;color:#f3f8fc}
h2{font-size:17px;margin:0 0 8px}p{margin:6px 0 12px;color:#aebfd0}
section{border:1px solid #334757;border-radius:14px;background:#172532;padding:15px;margin-bottom:10px}
#status{background:#102f30;border:1px solid #326359;border-radius:9px;padding:10px;color:#a9eecb;white-space:pre-wrap;overflow-wrap:anywhere}
button{background:#207b64;color:#fff;border:1px solid #3d997d;border-radius:9px;padding:9px;cursor:pointer}
button:hover{background:#269b77}.small{font-size:11px;color:#91adbf}label{display:flex;gap:8px;align-items:center}
article{border-top:1px solid #2c4252;padding:9px 0}article strong{font-size:12px}article p{font-size:12px;margin:3px 0;white-space:pre-wrap;overflow-wrap:anywhere}
#events{max-height:260px;overflow:auto}
</style></head><body>
<section><h2>Consecom Live · chat bidirecional</h2>
<p>Escreva na Central de Controle do Consecom, no seu Windows. As mensagens chegam nesta conversa pelo mesmo MCP e a resposta do ChatGPT é devolvida ao painel Consecom.</p>
<label><input id="auto" type="checkbox" checked> Permitir envio automático das mensagens digitadas no painel</label>
<p id="status" role="status">Conectando...</p>
<p class="small">Sem API paga ou segunda conexão. A confirmação do host não garante que o modelo já leu; consulte esta conversa. Fechar ou suspender o ChatGPT interrompe a recepção.</p>
</section><section><h2>Caixa de entrada</h2><div id="events">Aguardando mensagens...</div></section>
<script>
(function(){
"use strict";
const box=document.getElementById("events"),status=document.getElementById("status"),automatic=document.getElementById("auto");
const seen=new Set();let cursor=0,active=true,attempts=0,failures=0;
const started=Date.now(),MAX_ATTEMPTS=50;
function notice(s){status.textContent=s}
function extract(response){
 if(!response)return null;
 if(response.structuredContent)return response.structuredContent;
 if(response.result&&response.result.structuredContent)return response.result.structuredContent;
 if(response.content){for(const p of response.content){if(p.type==="text"){try{return JSON.parse(p.text)}catch(_){}}}}
 return response;
}
function observe(ev,mode,result){
 const fn=window.openai&&window.openai.callTool;
 if(typeof fn!=="function")return;
 Promise.resolve(fn("consecom_live_reportar_host",{id:ev.id,mode:mode,status:result})).catch(function(){});
}
function ask(ev,manual){
 if(!ev||!/^evt-[0-9]{6}$/.test(ev.id))return;
 if(attempts>=MAX_ATTEMPTS){notice("Limite de envios nesta sessão. Reabra o componente.");return}
 if(!manual&&!automatic.checked)return;
 const fn=window.openai&&window.openai.sendFollowUpMessage,mode=manual?"manual":"auto";
 if(typeof fn!=="function"){notice("Host não disponibilizou o envio de mensagens.");observe(ev,mode,"unavailable");return}
 attempts++;
 let prompt;
 if(ev.kind==="PANEL_MESSAGE"){
  prompt="Mensagem do usuário enviada pelo chat Consecom. ID: "+ev.id+"\n\n"+
    String(ev.message||"")+
    "\n\nResponda ao usuário e publique sua resposta com consecom_chat_responder(id="+ev.id+", texto=RESPOSTA, tipo=final). Para tarefas longas, publique andamento durante a execução. Não simule resultados.";
 }else{
  prompt="Consecom Live LAB: analise o evento sintetico "+ev.id+" com consecom_live_ler_evento. Nao execute comandos nem altere arquivos; registre se este turno surgiu automaticamente.";
 }
 notice("Encaminhando "+ev.id+" ao ChatGPT...");
 let settled=false;
 setTimeout(function(){if(!settled)observe(ev,mode,"pending")},6000);
 try{
  // Start the host follow-up BEFORE telemetry: an extra MCP call must not delay the user message.
  const followUp=fn({prompt:prompt,scrollToBottom:false});
  observe(ev,mode,"attempt");
  Promise.resolve(followUp).then(function(result){
   settled=true;
   if(result&&result.isError){observe(ev,mode,"host_rejected");notice("O host recusou "+ev.id+".");return}
   observe(ev,mode,"host_resolved");
   notice(ev.id+" foi aceito pelo host. Confira a conversa.");
  }).catch(function(err){
   settled=true;observe(ev,mode,"host_rejected");
   notice("Envio bloqueado: "+String(err&&err.message||err).slice(0,150));
  });
 }catch(err){
  settled=true;observe(ev,mode,"host_rejected");
  notice("Envio bloqueado: "+String(err&&err.message||err).slice(0,150));
 }
}
async function claimAndSend(ev,manual){
 if(!manual&&!automatic.checked)return;
 if(attempts>=MAX_ATTEMPTS)return;
 const api=window.openai&&window.openai.callTool;
 if(typeof api!=="function"){notice("Ponte de ferramentas indisponível.");return}
 try{
  const result=extract(await api("consecom_live_assumir_mensagem",{id:ev.id}));
  if(result&&result.claimed===true){ask(ev,manual)}
  else notice(ev.id+" já foi assumida por outra janela ou enviada.");
 }catch(err){
  notice("Não foi possível reservar "+ev.id+": "+String(err&&err.message||err).slice(0,110));
 }
}
function show(ev){
 if(!ev||!/^evt-[0-9]{6}$/.test(ev.id)||seen.has(ev.id))return;
 seen.add(ev.id);
 if(box.textContent==="Aguardando mensagens...")box.textContent="";
 const item=document.createElement("article"),title=document.createElement("strong"),msg=document.createElement("p"),btn=document.createElement("button");
 title.textContent=ev.id+" · "+(ev.kind==="PANEL_MESSAGE"?"Mensagem do painel":ev.kind);
 msg.textContent=String(ev.message||"").slice(0,500);
 btn.textContent="Enviar ao ChatGPT";
 btn.addEventListener("click",function(){
  if(ev.kind==="PANEL_MESSAGE")void claimAndSend(ev,true);
  else ask(ev,true);
 });
 item.append(title,msg,btn);box.prepend(item);
 while(box.children.length>14)box.lastChild.remove();
 if(ev.kind==="PANEL_MESSAGE"){
  if(ev.delivery==="queued")void claimAndSend(ev,false);
 }else if(Date.parse(ev.created_at)>=started-1000){ask(ev,false)}
}
async function poll(){
 const api=window.openai&&window.openai.callTool;
 if(typeof api!=="function"){notice("ChatGPT não disponibilizou a ponte MCP do componente.");return}
 notice("Aguardando mensagens do painel Consecom...");
 while(active){
  try{
   const result=extract(await api("consecom_live_aguardar",{after_sequence:cursor,wait_ms:8000}));
   failures=0;
   if(result){
    const events=Array.isArray(result.events)?result.events:[];
    for(const ev of events){show(ev);cursor=Math.max(cursor,Number(ev.sequence)||0)}
    if(result.gap)notice("Alguns eventos antigos saíram do histórico. Confira a fila no painel.");
    if(result.sequence!==undefined&&!events.length)cursor=Math.max(cursor,Number(result.sequence)||0);
   }
  }catch(err){
   notice("Reconectando ao MCP: "+String(err&&err.message||err).slice(0,135));
   // Recover quickly from a transient MCP initialization conflict without a busy retry loop.
   failures=Math.min(failures+1,5);
   const retryMs=Math.min(2500,400*Math.pow(2,failures-1));
   await new Promise(resolve=>setTimeout(resolve,retryMs));
  }
 }
}
window.addEventListener("pagehide",function(){active=false},{once:true});
void poll();
})();
</script></body></html>`
