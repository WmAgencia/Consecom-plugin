import { Database } from "bun:sqlite";
type Row={seq:number;id:string;message:string;state:string;created_at:string;claimed_at:string|null;host_status:string|null;answer:string|null;answered_at:string|null;attempts:number;external_id:string|null};
const dbPath=Bun.env.QUEUE_DB_PATH||"/data/consecom-cloud.sqlite";
const db=new Database(dbPath,{create:true});
db.exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; PRAGMA busy_timeout=5000;");
db.exec("CREATE TABLE IF NOT EXISTS events (seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT UNIQUE, message TEXT NOT NULL, state TEXT NOT NULL DEFAULT 'queued', created_at TEXT NOT NULL, claimed_at TEXT, host_status TEXT, answer TEXT, answered_at TEXT, attempts INTEGER NOT NULL DEFAULT 0, external_id TEXT UNIQUE);CREATE INDEX IF NOT EXISTS events_pending_idx ON events(state,seq);");
let selfTest:{status:string,steps?:string[],error?:string}={status:"pending"};
const ts=()=>new Date().toISOString();
const widgetURI="ui://consecom/cloud/live";
const widgetHTML="<!doctype html><html lang=\"pt-BR\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><style>:root{color-scheme:light dark}body{font:14px/1.5 system-ui,sans-serif; padding:16px; margin:0; background:#101820;color:#f0f6fa}section{background:#1a2936;border:1px solid #334555;padding:14px;border-radius:14px;margin-bottom:12px}h3{margin:0 0 5px}p{margin:8px 0}.muted{color:#a8bfca;font-size:12px}button{background:#147e6b;border:0;border-radius:9px;padding:9px 12px;color:white;cursor:pointer}#status{padding:10px;background:#102d30;border-radius:9px;white-space:pre-wrap;overflow-wrap:anywhere}.entry{border-bottom:1px solid #344758;padding:8px 0;overflow-wrap:anywhere}.entry:last-child{border:0}</style></head><body><section><h3>Consecom Cloud Live</h3><p class=\"muted\">Transporte experimental via MCP. Fila persistente no Railway; a recepção automática exige que este componente permaneça ativo no ChatGPT.</p><label><input id=\"auto\" type=\"checkbox\" checked> Encaminhar mensagens automaticamente</label><p id=\"status\" role=\"status\">Iniciando conexão...</p><button id=\"retry\" type=\"button\">Tentar agora</button></section><section><h3>Eventos recentes</h3><div id=\"items\" class=\"muted\">Nenhum evento recebido ainda.</div></section><script>(function(){\"use strict\";var host=window.openai||{};var auto=document.getElementById(\"auto\"),status=document.getElementById(\"status\"),items=document.getElementById(\"items\"),retry=document.getElementById(\"retry\");var active=true,polling=false,errors=0;var seen=new Set();var delay=function(ms){return new Promise(function(resolve){setTimeout(resolve,ms)})};function unpack(x){if(x&&x.structuredContent)return x.structuredContent;if(x&&x.result&&x.result.structuredContent)return x.result.structuredContent;for(var part of ((x&&x.content)||[])){if(part.type===\"text\"){try{return JSON.parse(part.text)}catch(_){}}}return x||{}}function announce(s){status.textContent=s}function persist(){if(typeof host.setWidgetState===\"function\"){try{host.setWidgetState({automatic:auto.checked})}catch(_){}}}if(host.widgetState&&typeof host.widgetState.automatic===\"boolean\")auto.checked=host.widgetState.automatic;auto.addEventListener(\"change\",function(){persist();if(auto.checked){active=true;poll()}});function report(id,flag){if(typeof host.callTool!==\"function\")return;Promise.resolve(host.callTool(\"consecom_live_reportar_host\",{id:id,status:flag})).catch(function(){})}function show(e){if(seen.has(e.id))return;seen.add(e.id);if(items.textContent===\"Nenhum evento recebido ainda.\")items.textContent=\"\";var item=document.createElement(\"div\");item.className=\"entry\";item.textContent=e.id+\" · \"+e.message;items.prepend(item);while(items.children.length>18)items.lastChild.remove()}function dispatch(e){if(!auto.checked)return;if(typeof host.sendFollowUpMessage!==\"function\"){announce(\"Este host não disponibilizou sendFollowUpMessage. Mensagem reservada, aguardando reenvio seguro.\");report(e.id,\"unavailable\");return}var prompt=\"Mensagem externa recebida pelo Consecom Cloud. ID: \"+e.id+\"\\n\\n\"+e.message+\"\\n\\nResponda à mensagem e publique sua resposta usando a ferramenta MCP consecom_chat_responder com id=\"+e.id+\", texto=RESPOSTA e tipo=final. Não alegue ter executado comandos que não executou.\";announce(\"Enviando \"+e.id+\" ao ChatGPT...\");var submitted;try{submitted=host.sendFollowUpMessage({prompt:prompt,scrollToBottom:false})}catch(err){announce(\"Erro ao solicitar turno: \"+String(err));report(e.id,\"host_rejected\");return}report(e.id,\"attempt\");Promise.resolve(submitted).then(function(r){if(r&&r.isError){report(e.id,\"host_rejected\");announce(\"Host recusou \"+e.id);return}report(e.id,\"host_resolved\");announce(\"ChatGPT aceitou a solicitação de \"+e.id+\"; aguardando resposta publicada.\")}).catch(function(err){report(e.id,\"host_rejected\");announce(\"Falha na entrega: \"+String(err).slice(0,150))})}async function loop(){if(polling||!active)return;polling=true;try{while(active){try{host=window.openai||{};if(typeof host.callTool!==\"function\"){announce(\"Ponte MCP indisponível. Tentarei reconectar.\");await delay(1200);continue}if(!auto.checked){announce(\"Recepção automática pausada pelo usuário.\");await delay(1600);continue}if(typeof host.sendFollowUpMessage!==\"function\"){announce(\"Host sem envio de mensagens. Aguardando recurso disponível...\");await delay(1800);continue}var r=unpack(await host.callTool(\"consecom_live_aguardar_e_assumir\",{wait_ms:5000}));if(r&&r.isError)throw Error(\"Ferramenta recusada\");if(!active)break;errors=0;var owned=new Set(Array.isArray(r.claimed_ids)?r.claimed_ids:[]);for(var e of (Array.isArray(r.events)?r.events:[])){show(e);if(owned.has(e.id))dispatch(e)}if(!r.events||!r.events.length)announce(\"Conectado. Aguardando mensagens no Railway...\")}catch(err){errors=Math.min(errors+1,6);announce(\"Falha temporária, reconectando: \"+String(err).slice(0,110));await delay(Math.min(5000,300*Math.pow(2,errors)))}}}finally{polling=false;if(active)void loop()}}window.addEventListener(\"pagehide\",function(){active=false});window.addEventListener(\"pageshow\",function(){active=true;poll()});document.addEventListener(\"visibilitychange\",function(){if(!document.hidden&&active)poll()});retry.addEventListener(\"click\",function(){active=true;poll()});void loop()})();</script></body></html>";
const json=(v:any,status=200,headers:any={})=>new Response(JSON.stringify(v),{status,headers:{"content-type":"application/json","cache-control":"no-store",...headers}});
const auth=(r:Request,t:string|undefined)=>!!t&&t.length>=24&&r.headers.get("authorization")==="Bearer "+t;
const schema=(p:any={},req:string[]=[])=>({type:"object",properties:p,required:req});
const tools=[
 {name:"consecom_live_abrir",description:"Abre Consecom Cloud Live nesta conversa. Requer widget ativo e consentimento do usuário.",inputSchema:schema(),_meta:{ui:{resourceUri:widgetURI},"openai/outputTemplate":widgetURI}},
 {name:"consecom_live_aguardar_e_assumir",description:"Espera ate 5 segundos e reserva eventos pendentes sem prazo de expiração.",inputSchema:schema({wait_ms:{type:"integer",minimum:0,maximum:5500}}),_meta:{"openai/widgetAccessible":true,ui:{visibility:["app"]}}},
 {name:"consecom_live_reportar_host",description:"Registra resultado de envio ao host para permitir recuperação segura.",inputSchema:schema({id:{type:"string"},status:{type:"string",enum:["attempt","host_resolved","host_rejected","unavailable"]}},["id","status"]),_meta:{"openai/widgetAccessible":true,ui:{visibility:["app"]}}},
 {name:"consecom_live_status",description:"Informa quantidades por estado, ID e idade da mensagem mais antiga.",inputSchema:schema()},
 {name:"consecom_chat_responder",description:"Registra resposta final para evento reservado, sem duplicar a resposta.",inputSchema:schema({id:{type:"string"},texto:{type:"string",maxLength:6000},tipo:{type:"string",enum:["final"]}},["id","texto","tipo"])}
];
const rpc=(id:any,result:any)=>json({jsonrpc:"2.0",id,result});
const rpcError=(id:any,code:number,message:string)=>json({jsonrpc:"2.0",id,error:{code,message}});
const get=(id:string)=>db.query("SELECT * FROM events WHERE id=?").get(id) as Row|null;
const counts=()=>db.query("SELECT state,COUNT(*) count FROM events GROUP BY state").all() as any[];
const summary=()=>({storage:"sqlite-volume",durable_queue:true,widget_requires_active_host:true,counts:counts(),oldest_queued:(db.query("SELECT created_at FROM events WHERE state='queued' ORDER BY seq LIMIT 1").get() as any)?.created_at||null});
const pick=db.transaction(()=>{
 // An event claimed but not even attempted by the host can be recovered automatically
 // once an abnormal widget interruption leaves it unacknowledged for >=2 minutes.
 db.query("UPDATE events SET state='queued',claimed_at=NULL,host_status=NULL WHERE state='claimed' AND host_status IS NULL AND claimed_at < ?").run(new Date(Date.now()-120000).toISOString());
 const selected=db.query("SELECT * FROM events WHERE state='queued' ORDER BY seq LIMIT 5").all() as Row[];
 const now=ts();
 for(const e of selected)db.query("UPDATE events SET state='claimed',claimed_at=?,attempts=attempts+1 WHERE seq=? AND state='queued'").run(now,e.seq);
 return {events:selected.map(e=>({id:e.id,kind:"PANEL_MESSAGE",message:e.message,sequence:e.seq,delivery:"claimed",created_at:e.created_at})),claimed_ids:selected.map(e=>e.id)};
});
const toolResult=(data:any)=>({structuredContent:data,content:[{type:"text",text:JSON.stringify(data)}]});
async function getBatch(ms:number){
 const end=Date.now()+ms;
 let batch=pick();
 while(batch.events.length===0&&Date.now()<end){await Bun.sleep(Math.min(250,end-Date.now()));batch=pick()}
 return {...batch,...summary()};
}
async function handle(req:Request):Promise<Response>{
 const url=new URL(req.url),method=req.method;
 if(method==="GET"&&url.pathname==="/health"){try{db.query("SELECT 1").get();return json({status:"ok",service:"consecom-cloud-mcp",version:"0.2.0",storage:"sqlite-volume",self_test:selfTest})}catch{return json({status:"database_unavailable"},503)}}
 if(url.pathname==="/mcp"){
  if(!auth(req,Bun.env.MCP_ACCESS_TOKEN))return json({error:"unauthorized"},401);
  if(method==="GET")return json({error:"use Streamable HTTP POST for this prototype"},405);
  if(method!=="POST")return json({error:"method_not_allowed"},405);
  let q:any;try{q=await req.json()}catch{return json({error:"invalid_json"},400)}
  if(q?.jsonrpc!=="2.0"||typeof q.method!=="string")return rpcError(q?.id??null,-32600,"Invalid JSON-RPC request");
  if(q.method.startsWith("notifications/"))return new Response(null,{status:202});
  if(q.method==="initialize")return rpc(q.id,{protocolVersion:"2025-06-18",capabilities:{tools:{},resources:{}},serverInfo:{name:"consecom-cloud-mcp",version:"0.2.0"}});
  if(q.method==="ping")return rpc(q.id,{});
  if(q.method==="tools/list")return rpc(q.id,{tools});
  if(q.method==="resources/list")return rpc(q.id,{resources:[{uri:widgetURI,name:"Consecom Cloud Live",mimeType:"text/html;profile=mcp-app"}]});
  if(q.method==="resources/read")return q.params?.uri===widgetURI?rpc(q.id,{contents:[{uri:widgetURI,mimeType:"text/html;profile=mcp-app",text:widgetHTML}]}):rpcError(q.id,-32602,"Unknown resource");
  if(q.method!=="tools/call")return rpcError(q.id,-32601,"Unknown method");
  const name=q.params?.name,arg=q.params?.arguments||{};
  try{
   let val:any;
   if(name==="consecom_live_abrir"||name==="consecom_live_status")val=summary();
   else if(name==="consecom_live_aguardar_e_assumir")val=await getBatch(Math.max(0,Math.min(5500,Number(arg.wait_ms)||5000)));
   else if(name==="consecom_live_reportar_host"){
    if(typeof arg.id!=="string"||!["attempt","host_resolved","host_rejected","unavailable"].includes(arg.status))throw Error("invalid report");
    const e=get(arg.id);if(!e)throw Error("event not found");
    if(e.state==="claimed"){
      if(arg.status==="host_rejected"||arg.status==="unavailable"){
        // Definitive denial: requeue for next attempt. Never retry acknowledged host turns.
        if(e.host_status!=="host_resolved")db.query("UPDATE events SET state='queued',host_status=?,claimed_at=NULL WHERE id=? AND state='claimed'").run(arg.status,arg.id);
      }else db.query("UPDATE events SET host_status=? WHERE id=? AND state='claimed'").run(arg.status,arg.id);
    }
    val={ok:true,id:arg.id,status:get(arg.id)?.state,host_status:get(arg.id)?.host_status};
   }
   else if(name==="consecom_chat_responder"){
    if(typeof arg.id!=="string"||typeof arg.texto!=="string"||!arg.texto.trim()||arg.texto.length>6000||arg.tipo!=="final")throw Error("invalid reply");
    const e=get(arg.id);if(!e||e.state==="queued")throw Error("event not yet delivered");
    if(e.state==="answered")val={ok:true,id:e.id,already_answered:true,answer:e.answer};
    else{db.query("UPDATE events SET state='answered',answer=?,answered_at=? WHERE id=? AND state='claimed'").run(arg.texto.trim(),ts(),arg.id);val={ok:true,id:arg.id,state:get(arg.id)?.state}}
   }else return rpcError(q.id,-32601,"Unknown tool");
   return rpc(q.id,toolResult(val));
  }catch(e){return rpc(q.id,{isError:true,content:[{type:"text",text:String(e)}]})}
 }
 if(url.pathname==="/bridge/enqueue"&&method==="POST"){
  if(!auth(req,Bun.env.BRIDGE_INGEST_TOKEN))return json({error:"unauthorized"},401);
  let a:any;try{a=await req.json()}catch{return json({error:"invalid_json"},400)}
  const msg=a?.message,external=a?.external_id;
  if(typeof msg!=="string"||!msg.trim()||msg.length>2000)return json({error:"invalid_message"},400);
  if(external!==undefined&&(typeof external!=="string"||external.length>120||!/^[A-Za-z0-9_-]+$/.test(external)))return json({error:"invalid_external_id"},400);
  if(external){const duplicate=db.query("SELECT id,state FROM events WHERE external_id=?").get(external) as any;if(duplicate)return json({id:duplicate.id,status:duplicate.state,duplicate:true},200)}
  const n=(db.query("SELECT count(*) c FROM events WHERE state<>'answered'").get() as any).c;
  if(n>=10000)return json({error:"queue_full"},429);
  const result=db.query("INSERT INTO events(message,state,created_at,external_id) VALUES(?,'queued',?,?)").run(msg.trim(),ts(),external||null);
  const seq=Number(result.lastInsertRowid),id="evt-"+String(seq).padStart(6,"0");
  db.query("UPDATE events SET id=? WHERE seq=?").run(id,seq);
  return json({id,status:"queued",persistent:true},202);
 }
 if(url.pathname.startsWith("/bridge/result/")&&method==="GET"){
  if(!auth(req,Bun.env.BRIDGE_INGEST_TOKEN))return json({error:"unauthorized"},401);
  const e=get(url.pathname.slice("/bridge/result/".length));return e?json({id:e.id,status:e.state,answer:e.answer,created_at:e.created_at,answered_at:e.answered_at}):json({error:"not_found"},404);
 }
 if(url.pathname==="/bridge/status"&&method==="GET"){
  if(!auth(req,Bun.env.BRIDGE_INGEST_TOKEN))return json({error:"unauthorized"},401);
  return json(summary());
 }
 return json({error:"not_found"},404);
}

async function startupSelfTest(){
 const steps:string[]=[];
 const bridge=Bun.env.BRIDGE_INGEST_TOKEN,mcp=Bun.env.MCP_ACCESS_TOKEN;
 if(!bridge||!mcp){selfTest={status:"failed",error:"missing service auth tokens"};return}
 let testId="";
 try{
  const integrity=db.query("PRAGMA integrity_check").get() as any;
  if(!integrity||!Object.values(integrity).includes("ok"))throw Error("SQLite integrity_check failed");
  steps.push("sqlite-integrity");
  const unauthorized=await handle(new Request("http://local/mcp",{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({jsonrpc:"2.0",id:1,method:"initialize"})}));
  if(unauthorized.status!==401)throw Error("authentication rejection failed");
  steps.push("mcp-unauthorized-401");
  const rpcHeaders={authorization:"Bearer "+mcp,"content-type":"application/json"};
  const initialize=await handle(new Request("http://local/mcp",{method:"POST",headers:rpcHeaders,body:JSON.stringify({jsonrpc:"2.0",id:2,method:"initialize"})}));
  const initJson=await initialize.json() as any;
  if(initialize.status!==200||initJson.result?.serverInfo?.name!=="consecom-cloud-mcp")throw Error("MCP initialize failed");
  steps.push("mcp-initialize");
  const list=await handle(new Request("http://local/mcp",{method:"POST",headers:rpcHeaders,body:JSON.stringify({jsonrpc:"2.0",id:3,method:"tools/list"})}));
  const listBody=await list.json() as any;
  if(!listBody.result?.tools?.some((t:any)=>t.name==="consecom_chat_responder"))throw Error("MCP tools/list failed");
  steps.push("mcp-tools-list");
  const currentPending=(db.query("SELECT COUNT(*) n FROM events WHERE state='queued'").get() as any).n;
  if(currentPending>0){selfTest={status:"passed-with-queue-skip",steps};return}
  const ext="selftest-"+Date.now();
  const ingestHeaders={authorization:"Bearer "+bridge,"content-type":"application/json"};
  const enqueued=await handle(new Request("http://local/bridge/enqueue",{method:"POST",headers:ingestHeaders,body:JSON.stringify({message:"[Consecom Cloud internal test]",external_id:ext})}));
  const enq=await enqueued.json() as any;
  if(enqueued.status!==202||!enq.id)throw Error("enqueue failed");
  testId=enq.id;steps.push("bridge-enqueue");
  const getCall=(id:number,name:string,args:any)=>handle(new Request("http://local/mcp",{method:"POST",headers:rpcHeaders,body:JSON.stringify({jsonrpc:"2.0",id,method:"tools/call",params:{name,arguments:args}})}));
  const claimResp=await getCall(4,"consecom_live_aguardar_e_assumir",{wait_ms:1});
  const claimed=await claimResp.json() as any;
  if(!claimed.result?.structuredContent?.claimed_ids?.includes(testId))throw Error("atomic claim failed");
  steps.push("mcp-atomic-claim");
  const replyResp=await getCall(5,"consecom_chat_responder",{id:testId,texto:"Teste OK",tipo:"final"});
  const replied=await replyResp.json() as any;
  if(!replied.result?.structuredContent?.ok)throw Error("reply publish failed");
  steps.push("mcp-response");
  const resultResp=await handle(new Request("http://local/bridge/result/"+testId,{headers:ingestHeaders}));
  const result=await resultResp.json() as any;
  if(result.status!=="answered"||result.answer!=="Teste OK")throw Error("result lookup failed");
  steps.push("bridge-result");
  selfTest={status:"passed",steps};
 }catch(err){selfTest={status:"failed",steps,error:String(err)}}
 finally{if(testId)db.query("DELETE FROM events WHERE id=?").run(testId)}
}
void startupSelfTest();

export default {port:Number(Bun.env.PORT||8080),async fetch(req:Request){try{return await handle(req)}catch(e){console.error("cloud MCP request error:",e);return json({error:"internal_error"},500)}}};
