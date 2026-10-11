package main

// The test attempts a host-mediated follow-up automatically when a NEW
// synthetic SSE event arrives. This is not a click simulator and cannot
// bypass a ChatGPT host rejection or user-consent requirement.
const widgetHTML = `<!doctype html>
<html lang="pt-BR">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<style>
:root{color-scheme:dark}body{margin:0;padding:20px;background:#091b19;color:#e9fff6;font:14px system-ui,sans-serif}
section{border:1px solid #245747;border-radius:16px;padding:16px;margin-bottom:13px;background:#102b25}
h2{margin:0 0 8px;font-size:20px}p{line-height:1.5}#events{list-style:none;padding:0;margin:12px 0}
li{border-bottom:1px solid #245747;padding:12px 0}small{color:#aad3c2}
label{display:flex;gap:9px;align-items:center;font-weight:600}
button{border:1px solid #348f71;border-radius:10px;background:#13533f;color:white;padding:9px;cursor:pointer}
#status{overflow-wrap:anywhere}
</style></head>
<body>
<section><h2>Consecom Live — Teste experimental</h2>
<p>Eventos simulados por SSE. Ao chegar um evento NOVO, o componente tenta solicitar automaticamente um turno ao ChatGPT pela interface MCP Apps.</p>
<label><input type="checkbox" id="auto" checked> Solicitar continuação automaticamente</label>
<p id="status" role="status">Conectando ao canal...</p>
<small>O ChatGPT pode rejeitar a solicitação ou exigir ação humana. O experimento não simula cliques nem ignora confirmações.</small>
</section>
<section><h2>Eventos</h2><ul id="events"></ul></section>
<script>
(function(){
  "use strict";
  const origin = {{printf "%q" .Origin}};
  const channel = {{printf "%q" .Channel}};
  const enabled = document.getElementById("auto");
  const status = document.getElementById("status");
  const list = document.getElementById("events");
  const seen = new Set();
  const startedAt = Date.now();
  let lastAttempt = 0;
  let attempts = 0;
  let latest = null;
  const MAX_ATTEMPTS = 10, MIN_INTERVAL_MS = 30000;

  function report(msg) { status.textContent = msg; }
  function observe(ev, mode, outcome, detail) {
    // Best-effort telemetry only. It never contains chat text or private data.
    if (!ev || !/^evt-[0-9]{6}$/.test(ev.id)) return;
    const payload = JSON.stringify({EventID:ev.id,Mode:mode,Status:outcome,Error:String(detail||"").slice(0,180)});
    try {
      void fetch(origin+"/demo/widget-report?channel="+encodeURIComponent(channel), {
        method:"POST",headers:{"Content-Type":"application/json"},body:payload,
        mode:"cors",credentials:"omit",keepalive:false
      }).catch(function(){});
    } catch (_err) {}
  }
  async function requestTurn(ev, manual) {
    if (!ev || !ev.id || !/^evt-[0-9]{6}$/.test(ev.id)) return;
    if (!manual) {
      if (!enabled.checked) return;
      if (attempts >= MAX_ATTEMPTS) { report("Limite de 10 solicitações por sessão; use o botão manual."); return; }
      if (Date.now()-lastAttempt < MIN_INTERVAL_MS) { report("Evento recebido, aguardando intervalo de segurança."); return; }
    }
    const fn = window.openai && window.openai.sendFollowUpMessage;
    if (typeof fn !== "function") { report("ChatGPT não disponibilizou sendFollowUpMessage neste componente."); return; }
    lastAttempt = Date.now();
    attempts++;
    const mode=manual?"manual":"auto";
    observe(ev,mode,"attempt","");
    let settled=false;
    // A pending promise is not proof of wake-up.
    setTimeout(()=>{if(!settled)observe(ev,mode,"pending","No host result within 5000ms")},5000);
    const prompt = "Consecom Live POC: analise somente o evento sintético "+ev.id+
      " usando a ferramenta consecom_live_ler_evento do app. Este é um experimento e não deve alterar arquivos.";
    report("Solicitando análise de "+ev.id+" ao ChatGPT...");
    try {
      const response = await fn({prompt:prompt,scrollToBottom:false});
      settled=true;
      if (response && response.isError) {
        observe(ev,mode,"host_rejected","Host returned isError");
        report("O ChatGPT rejeitou a solicitação para "+ev.id); return;
      }
      observe(ev,mode,"host_resolved","");
      report("Solicitação entregue ao host para "+ev.id+". Verifique se um novo turno realmente começou.");
    } catch (e) {
      settled=true;
      observe(ev,mode,"host_rejected",String((e&&e.message)||e).slice(0,180));
      report("Solicitação não aceita: "+String((e && e.message)||e).slice(0,180));
    }
  }
  function onEvent(ev) {
    if (!ev || !ev.id || seen.has(ev.id)) return;
    seen.add(ev.id); latest = ev;
    const li=document.createElement("li");
    const title=document.createElement("strong");
    title.textContent=ev.id+" · "+(ev.kind||"EVENTO");
    const desc=document.createElement("p");
    desc.textContent=String(ev.message||"").slice(0,400);
    const btn=document.createElement("button");
    btn.textContent="Tentar análise manualmente";
    btn.addEventListener("click",()=>requestTurn(ev,true));
    li.append(title,desc,btn);list.prepend(li);
    while(list.children.length>20)list.lastChild.remove();
    // Cached SSE history is displayed but never silently re-submitted.
    const created=Date.parse(ev.created_at);
    if (!Number.isFinite(created) || created<startedAt) return;
    void requestTurn(ev,false);
  }
  enabled.addEventListener("change",()=>{
    report(enabled.checked?"Experimento automático ligado. Aguardando novo evento.":"Experimento automático desligado.");
  });
  if (typeof EventSource!=="function") {report("SSE não disponível neste ambiente.");return;}
  const stream=new EventSource(origin+"/events?channel="+encodeURIComponent(channel));
  stream.addEventListener("open",()=>report("SSE conectado. Automático ligado; aguardando evento novo."));
  stream.addEventListener("consecom",e=>{
    try{onEvent(JSON.parse(e.data));}catch(err){report("Evento inválido recebido.");}
  });
  stream.addEventListener("error",()=>report("SSE desconectado; o navegador tentará reconectar."));
  window.addEventListener("pagehide",()=>stream.close(),{once:true});
})();
</script>
</body></html>`
