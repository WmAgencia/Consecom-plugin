package main

// This UI uses the authenticated MCP Apps bridge only; it never fetches a
// second public endpoint, copies credentials or automates chat.openai.com.
const liveWidgetHTML = `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<style>
:root{color-scheme:light dark}body{margin:0;padding:16px;font:14px/1.5 system-ui,sans-serif;background:#0f1721;color:#f3f8fc}
h2{font-size:17px;margin:0 0 8px}p{margin:6px 0 12px;color:#aebfd0}
section{border:1px solid #334757;border-radius:14px;background:#172532;padding:15px;margin-bottom:10px}
#status{background:#102f30;border:1px solid #326359;border-radius:9px;padding:10px;color:#a9eecb;white-space:pre-wrap;overflow-wrap:anywhere}
button{background:#207b64;color:#fff;border:1px solid #3d997d;border-radius:9px;padding:9px;cursor:pointer}
button:hover{background:#269b77}.small{font-size:11px;color:#91adbf}label{display:flex;gap:8px;align-items:center}
article{border-top:1px solid #2c4252;padding:9px 0}article strong{font-size:12px}article p{font-size:11px;margin:3px 0}
#events{max-height:220px;overflow:auto}
</style></head><body>
<section><h2>Consecom Live · integrado</h2>
<p>Um único MCP. Eventos sintéticos gerados no Windows; o ChatGPT recebe por suas ferramentas já conectadas.</p>
<label><input id="auto" type="checkbox" checked> Testar continuação automática sem clique</label>
<p id="status" role="status">Inicializando componente...</p>
<p class="small">Não há API paga, segundo plugin, automação do navegador ou acesso adicional aos arquivos. O ChatGPT pode bloquear mensagens sem gesto do usuário.</p>
</section><section><h2>Eventos de teste</h2><div id="events">Aguardando...</div></section>
<script>
(function(){
"use strict";
const box=document.getElementById("events"),status=document.getElementById("status"),automatic=document.getElementById("auto");
const seen=new Set();
let cursor=0,active=true,attempts=0;
const started=Date.now();
const MAX_ATTEMPTS=9;
function message(text){status.textContent=text}
function extract(response){
  if(!response)return null;
  if(response.structuredContent)return response.structuredContent;
  if(response.result&&response.result.structuredContent)return response.result.structuredContent;
  if(response.content){
    for(const p of response.content){
      if(p.type==="text"){try{return JSON.parse(p.text)}catch(_){}}
    }
  }
  return response;
}
function observe(ev,mode,result){
  const api=window.openai&&window.openai.callTool;
  if(typeof api!=="function")return;
  // Best effort. The reporting call cannot prove that another model turn started.
  Promise.resolve(api("consecom_live_reportar_host",{id:ev.id,mode:mode,status:result})).catch(function(){});
}
function ask(ev,manual){
  if(!ev||!/^evt-[0-9]{6}$/.test(ev.id))return;
  if(!manual&&(!automatic.checked||attempts>=MAX_ATTEMPTS))return;
  const fn=window.openai&&window.openai.sendFollowUpMessage;
  const mode=manual?"manual":"auto";
  if(typeof fn!=="function"){message("Host não expôs sendFollowUpMessage.");observe(ev,mode,"unavailable");return}
  attempts++;
  observe(ev,mode,"attempt");
  const prompt="Consecom Live LAB: analise o evento sintetico "+ev.id+
    " com consecom_live_ler_evento. Nao execute comandos nem altere arquivos; registre se este turno surgiu automaticamente.";
  message((manual?"Solicitação manual":"Tentativa automática")+" enviada ao host para "+ev.id+"...");
  let settled=false;
  setTimeout(function(){if(!settled)observe(ev,mode,"pending")},6000);
  try{
    Promise.resolve(fn({prompt:prompt,scrollToBottom:false})).then(function(result){
      settled=true;
      if(result&&result.isError){observe(ev,mode,"host_rejected");message("Host rejeitou a mensagem de "+ev.id);return}
      observe(ev,mode,"host_resolved");
      message("Host respondeu à tentativa de "+ev.id+". Verifique se houve um novo turno REAL.");
    }).catch(function(err){
      settled=true;observe(ev,mode,"host_rejected");
      message("Host bloqueou a continuação: "+String(err&&err.message||err).slice(0,180));
    });
  }catch(err){
    settled=true;observe(ev,mode,"host_rejected");
    message("Host bloqueou a continuação: "+String(err&&err.message||err).slice(0,180));
  }
}
function show(ev){
  if(!ev||!/^evt-[0-9]{6}$/.test(ev.id)||seen.has(ev.id))return;
  seen.add(ev.id);
  if(box.textContent==="Aguardando...")box.textContent="";
  const item=document.createElement("article"),title=document.createElement("strong"),msg=document.createElement("p"),btn=document.createElement("button");
  title.textContent=ev.id+" · "+ev.kind;
  msg.textContent=String(ev.message||"").slice(0,250);
  btn.textContent="Pedir análise manualmente";
  btn.addEventListener("click",function(){ask(ev,true)});
  item.append(title,msg,btn);box.prepend(item);
  while(box.children.length>12)box.lastChild.remove();
  if(Date.parse(ev.created_at)>=started-1000)ask(ev,false);
}
async function poll(){
  const api=window.openai&&window.openai.callTool;
  if(typeof api!=="function"){message("ChatGPT não expôs a ponte de ferramentas do componente.");return}
  message("MCP único conectado. Aguardando eventos de teste...");
  while(active&&Date.now()-started<6*60*1000){
    try{
      const result=extract(await api("consecom_live_aguardar",{after_sequence:cursor,wait_ms:8000}));
      if(result){
        const events=Array.isArray(result.events)?result.events:[];
        for(const ev of events){
          show(ev);cursor=Math.max(cursor,Number(ev.sequence)||0);
        }
        if(result.gap){message("Alguns eventos antigos expiraram do histórico de demonstração.")}
        if(result.sequence!==undefined&&!events.length)cursor=Math.max(cursor,Number(result.sequence)||0);
      }
    }catch(error){
      message("Consulta pelo MCP indisponível: "+String(error&&error.message||error).slice(0,160));
      await new Promise(resolve=>setTimeout(resolve,3000));
    }
  }
  if(active)message("Janela de teste encerrada. Reabra o Consecom Live para nova rodada.");
}
window.addEventListener("pagehide",function(){active=false},{once:true});
void poll();
})();
</script></body></html>`
