import { Database } from "bun:sqlite";
type Row={seq:number;id:string;message:string;state:string;created_at:string;claimed_at:string|null;host_status:string|null;answer:string|null;answered_at:string|null;attempts:number;external_id:string|null};
const dbPath=Bun.env.QUEUE_DB_PATH||"/data/consecom-cloud.sqlite";
const db=new Database(dbPath,{create:true});
db.exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=FULL; PRAGMA busy_timeout=5000;");
db.exec("CREATE TABLE IF NOT EXISTS events (seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT UNIQUE, message TEXT NOT NULL, state TEXT NOT NULL DEFAULT 'queued', created_at TEXT NOT NULL, claimed_at TEXT, host_status TEXT, answer TEXT, answered_at TEXT, attempts INTEGER NOT NULL DEFAULT 0, external_id TEXT UNIQUE);CREATE INDEX IF NOT EXISTS events_pending_idx ON events(state,seq);");
let selfTest:{status:string,steps?:string[],error?:string}={status:"pending"};
db.exec("CREATE TABLE IF NOT EXISTS health_meta(k TEXT PRIMARY KEY,v INTEGER NOT NULL)");
db.query("INSERT INTO health_meta(k,v) VALUES('boots',1) ON CONFLICT(k) DO UPDATE SET v=v+1").run();
const bootCount=(db.query("SELECT v FROM health_meta WHERE k='boots'").get() as any).v;
const ts=()=>new Date().toISOString();

import { createHash, timingSafeEqual, randomBytes } from "node:crypto";
const issuer="https://consecom-mcp-relay-production.up.railway.app";
const resource=issuer+"/mcp";
const oauthScope="consecom:live";
db.exec("CREATE TABLE IF NOT EXISTS oauth_clients (client_id TEXT PRIMARY KEY, redirect_json TEXT NOT NULL, created_at INTEGER NOT NULL);CREATE TABLE IF NOT EXISTS oauth_flows (flow_id TEXT PRIMARY KEY, client_id TEXT NOT NULL, redirect_uri TEXT NOT NULL, code_challenge TEXT NOT NULL, state TEXT, resource TEXT NOT NULL, scope TEXT NOT NULL, expires_at INTEGER NOT NULL);CREATE TABLE IF NOT EXISTS oauth_codes (code_hash TEXT PRIMARY KEY, client_id TEXT NOT NULL, redirect_uri TEXT NOT NULL, challenge TEXT NOT NULL, resource TEXT NOT NULL, scope TEXT NOT NULL, expires_at INTEGER NOT NULL);CREATE TABLE IF NOT EXISTS oauth_tokens (access_hash TEXT PRIMARY KEY, refresh_hash TEXT UNIQUE NOT NULL, client_id TEXT NOT NULL, resource TEXT NOT NULL, scope TEXT NOT NULL, expires_at INTEGER NOT NULL, refresh_expires_at INTEGER NOT NULL);");
db.exec("CREATE TABLE IF NOT EXISTS oauth_consumed_flows (flow_hash TEXT PRIMARY KEY, consumed_at INTEGER NOT NULL)");
const secureToken=()=>randomBytes(32).toString("base64url");
const tokenHash=(x:string)=>createHash("sha256").update(x).digest("hex");
const safeCompare=(a:string,b:string)=>a.length===b.length&&timingSafeEqual(Buffer.from(a),Buffer.from(b));
const bounded=(value:string,max:number)=>typeof value==="string"&&value.length>0&&value.length<=max;
const escapeHtml=(s:string)=>s.replaceAll("&","&amp;").replaceAll("<","&lt;").replaceAll(">","&gt;").replaceAll('"',"&quot;").replaceAll("'","&#39;");
const oauthCors={"cache-control":"no-store","referrer-policy":"no-referrer","x-content-type-options":"nosniff"};
const ojson=(v:any,status=200,more:any={})=>new Response(JSON.stringify(v),{status,headers:{"content-type":"application/json",...oauthCors,...more}});
const challengeHeader={"WWW-Authenticate":'Bearer resource_metadata="'+issuer+'/.well-known/oauth-protected-resource", scope="'+oauthScope+'"'};
const unauthorized=()=>ojson({error:"unauthorized",detail:"Use OAuth no ChatGPT. Este servidor não aceita MCP anônimo."},401,challengeHeader);
const oauthValid=(r:Request)=>{
 const h=r.headers.get("authorization")||"";
 if(!h.startsWith("Bearer "))return false;
 const t=h.slice(7);
 if(!bounded(t,256))return false;
 // Keep existing administrative token solely for controlled diagnostics.
 if(auth(r,Bun.env.MCP_ACCESS_TOKEN))return true;
 const row=db.query("SELECT expires_at,resource,scope FROM oauth_tokens WHERE access_hash=?").get(tokenHash(t)) as any;
 return !!row&&row.expires_at>Date.now()&&row.resource===resource&&row.scope===oauthScope;
};
function consecomOAuthMetadata(){
 return {issuer,authorization_endpoint:issuer+"/oauth/authorize",token_endpoint:issuer+"/oauth/token",registration_endpoint:issuer+"/oauth/register",response_types_supported:["code"],grant_types_supported:["authorization_code","refresh_token"],code_challenge_methods_supported:["S256"],token_endpoint_auth_methods_supported:["none"],scopes_supported:[oauthScope],client_id_metadata_document_supported:false};
}
const registeredClient=(cid:string)=>db.query("SELECT * FROM oauth_clients WHERE client_id=?").get(cid) as any;
const validRedirect=(v:string)=>{try{const u=new URL(v);return u.protocol==="https:"&&["chatgpt.com","chat.openai.com"].includes(u.hostname)&&u.username===""&&u.password===""&&u.hash===""}catch{return false}};
const matchesRedirect=(cid:string,redirect:string)=>{
 const row=registeredClient(cid);if(!row)return false;
 try{return (JSON.parse(row.redirect_json) as string[]).includes(redirect)}catch{return false}
};
const oauthRate=new Map<string,{n:number,reset:number}>();
function limited(r:Request,kind:string,max:number,windowMs:number){
 const ip=(r.headers.get("x-forwarded-for")||"unknown").split(",")[0].trim().slice(0,64);
 const k=kind+":"+ip,now=Date.now();let x=oauthRate.get(k);
 if(!x||x.reset<now)x={n:0,reset:now+windowMs};
 x.n++;oauthRate.set(k,x);
 if(oauthRate.size>2500){for(const [key,v] of oauthRate)if(v.reset<now)oauthRate.delete(key)}
 return x.n>max;
}

const oauthFlowError=(used:boolean)=>new Response('<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Consecom Cloud — autorização</title><style>body{display:grid;min-height:100vh;place-items:center;margin:0;padding:20px;box-sizing:border-box;background:#0f172a;color:#e2e8f0;font:15px/1.6 system-ui}main{max-width:440px;padding:28px;background:#1e293b;border:1px solid #475569;border-radius:16px}h2{margin:0 0 8px}p{color:#cbd5e1}a{color:#7dd3fc}strong{color:#fff}</style></head><body><main><h2>'+(used?'Autorização já utilizada':'Autorização expirada')+'</h2><p>'+(used?'O Consecom recebeu e aceitou o primeiro envio deste formulário. Por segurança, ele só pode ser utilizado uma vez.':'O prazo de segurança desta tentativa de autorização terminou.')+'</p><p><strong>Agora:</strong> retorne à aba do ChatGPT e confira se o plugin ficou conectado. Caso contrário, inicie uma nova conexão OAuth no ChatGPT. Não use o botão Voltar nem reenvie este formulário.</p><p><a href="https://chatgpt.com">Voltar ao ChatGPT</a></p></main></body></html>',{status:used?409:410,headers:{"content-type":"text/html; charset=utf-8","cache-control":"no-store","referrer-policy":"no-referrer","x-content-type-options":"nosniff","content-security-policy":"default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'"}});

const oauthHtml=(flowId:string,error?:string)=>new Response('<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Autorizar Consecom Cloud</title><style>body{margin:0;padding:22px;display:grid;min-height:100vh;place-items:center;font:15px/1.6 system-ui;background:radial-gradient(ellipse at 70% 20%,#153b43,#091522 70%);color:#e8f4f4}main{width:min(100%,440px);padding:32px;border:1px solid #3c6067;border-radius:22px;background:#182a35;box-shadow:0 22px 75px #0006}.brand{font-size:11px;letter-spacing:.12em;color:#83e0c3;font-weight:800;margin-bottom:13px}input,button{font:inherit;padding:12px;border-radius:8px;box-sizing:border-box;width:100%}button{background:#57d3ab;border:0;color:#09231e;cursor:pointer;font-weight:750;margin-top:14px}button:hover{background:#85e5c5}input{background:#0a1923;color:white;border:1px solid #607b83;outline:0}input:focus{border-color:#71debb;box-shadow:0 0 0 3px #50cda531}p{line-height:1.5;color:#cbd5e1}.error{color:#fca5a5}</style></head><body><main><div class="brand">CONSECOM CLOUD · ACESSO SEGURO</div><h2>Conecte seu ChatGPT</h2><p>Utilize sua chave Consecom para autorizar o ChatGPT. Confirme o endereço <b>consecom-mcp-relay-production.up.railway.app</b>.</p>'+ (error?'<p class="error">'+escapeHtml(error)+'</p>':'')+'<form method="post" action="/oauth/authorize"><input type="hidden" name="flow_id" value="'+escapeHtml(flowId)+'"><label for="password">Chave Consecom (API Key)</label><input id="password" name="password" type="password" autocomplete="off" required minlength="16" maxlength="150"><button type="submit">Conectar ao ChatGPT</button></form><p style="font-size:12px">Sua chave é confidencial e não é a senha do ChatGPT. Não a compartilhe.</p></main></body></html>',{status:200,headers:{"content-type":"text/html; charset=utf-8","content-security-policy":"default-src 'none'; style-src 'unsafe-inline'; form-action 'self' https://chatgpt.com https://chat.openai.com; frame-ancestors 'none'; base-uri 'none'","x-frame-options":"DENY",...oauthCors}});
async function handleOAuth(req:Request,u:URL):Promise<Response|null>{
 const path=u.pathname,m=req.method,now=Date.now();
 if(m==="GET"&&(path==="/.well-known/oauth-authorization-server"||path==="/.well-known/openid-configuration"))return ojson(consecomOAuthMetadata());
 if(m==="GET"&&(path==="/.well-known/oauth-protected-resource"||path==="/.well-known/oauth-protected-resource/mcp"))return ojson({resource,authorization_servers:[issuer],scopes_supported:[oauthScope],bearer_methods_supported:["header"]});
 if(path==="/oauth/register"&&m==="POST"){
  if(limited(req,"dcr",30,3600000))return ojson({error:"rate_limited"},429);
  let a:any;try{a=await req.json()}catch{return ojson({error:"invalid_client_metadata"},400)}
  if(!Array.isArray(a.redirect_uris)||a.redirect_uris.length<1||a.redirect_uris.length>5||a.redirect_uris.some((x:any)=>typeof x!=="string"||!validRedirect(x)))return ojson({error:"invalid_redirect_uri"},400);
  if(a.token_endpoint_auth_method&&a.token_endpoint_auth_method!=="none")return ojson({error:"invalid_client_metadata"},400);
  const n=(db.query("SELECT COUNT(*) n FROM oauth_clients").get() as any).n;
  if(n>=1000)return ojson({error:"registration_closed"},429);
  const cid="cc_"+secureToken();
  db.query("INSERT INTO oauth_clients (client_id,redirect_json,created_at) VALUES(?,?,?)").run(cid,JSON.stringify(a.redirect_uris),now);
  return ojson({client_id:cid,client_id_issued_at:Math.floor(now/1000),redirect_uris:a.redirect_uris,token_endpoint_auth_method:"none",grant_types:["authorization_code","refresh_token"],response_types:["code"]},201);
 }
 if(path==="/oauth/authorize"&&m==="GET"){
  if(limited(req,"authorization_start",60,3600000))return ojson({error:"rate_limited"},429);
  const cid=u.searchParams.get("client_id")||"",ruri=u.searchParams.get("redirect_uri")||"",challenge=u.searchParams.get("code_challenge")||"",state=u.searchParams.get("state")||"",scope=u.searchParams.get("scope")||oauthScope,requestResource=u.searchParams.get("resource")||resource;
  if(u.searchParams.get("response_type")!=="code"||u.searchParams.get("code_challenge_method")!=="S256"||!matchesRedirect(cid,ruri)||!/^[-A-Za-z0-9_]{43}$/.test(challenge)||!bounded(state,400)||scope!==oauthScope||requestResource!==resource)return ojson({error:"invalid_request",detail:"Client, callback, PKCE, scope or resource invalid."},400);
  const flow="flow_"+secureToken(); db.query("INSERT INTO oauth_flows(flow_id,client_id,redirect_uri,code_challenge,state,resource,scope,expires_at) VALUES(?,?,?,?,?,?,?,?)").run(flow,cid,ruri,challenge,state,resource,scope,now+600000);
  return oauthHtml(flow);
 }
 if(path==="/oauth/authorize"&&m==="POST"){
  if(limited(req,"authorization_submit",12,600000))return ojson({error:"too_many_attempts"},429);
  let f:FormData;try{f=await req.formData()}catch{return ojson({error:"invalid_request"},400)}
  const flowId=String(f.get("flow_id")||""),secret=String(f.get("password")||"");
  if(!flowId.startsWith("flow_")||flowId.length>100)return ojson({error:"invalid_flow"},400);
  const flow=db.query("SELECT * FROM oauth_flows WHERE flow_id=?").get(flowId) as any;
  if(!flow||flow.expires_at<now){const used=!!db.query("SELECT 1 FROM oauth_consumed_flows WHERE flow_hash=?").get(tokenHash(flowId));return oauthFlowError(used)}
  const expected=Bun.env.OAUTH_PAIRING_CODE||"";
  if(!expected||expected.length<16||!safeCompare(secret,expected))return oauthHtml(flowId,"Senha incorreta. Confira a variável OAUTH_PAIRING_CODE no Railway.");
  const code=secureToken();
  db.transaction(()=>{
   db.query("DELETE FROM oauth_flows WHERE flow_id=?").run(flowId);
   db.query("INSERT INTO oauth_consumed_flows(flow_hash,consumed_at) VALUES(?,?)").run(tokenHash(flowId),now);
   db.query("DELETE FROM oauth_consumed_flows WHERE consumed_at < ?").run(now-86400000);
   db.query("INSERT INTO oauth_codes(code_hash,client_id,redirect_uri,challenge,resource,scope,expires_at) VALUES(?,?,?,?,?,?,?)").run(tokenHash(code),flow.client_id,flow.redirect_uri,flow.code_challenge,resource,flow.scope,now+120000);
  })();
  const cb=new URL(flow.redirect_uri);cb.searchParams.set("code",code);cb.searchParams.set("state",flow.state);console.info("consecom-oauth-redirect",JSON.stringify({host:cb.hostname,path:cb.pathname,has_code:cb.searchParams.has("code"),has_state:cb.searchParams.has("state")}));return new Response(null,{status:303,headers:{"Location":cb.toString(),...oauthCors}});
 }
 if(path==="/oauth/token"&&m==="POST"){
  if(limited(req,"token",120,3600000))return ojson({error:"rate_limited"},429);
  let form:URLSearchParams;try{form=new URLSearchParams(await req.text())}catch{return ojson({error:"invalid_request"},400)}
  const grant=form.get("grant_type"),cid=form.get("client_id")||"";
  if(!registeredClient(cid))return ojson({error:"invalid_client"},401);
  const targetResource=form.get("resource")||resource;
  if(targetResource!==resource)return ojson({error:"invalid_target"},400);
  if(grant==="authorization_code"){
   const code=form.get("code")||"",verifier=form.get("code_verifier")||"",redirect=form.get("redirect_uri")||"";
   if(!bounded(code,256)||!/^[-._~A-Za-z0-9]{43,128}$/.test(verifier)||!matchesRedirect(cid,redirect))return ojson({error:"invalid_grant"},400);
   const row=db.query("SELECT * FROM oauth_codes WHERE code_hash=?").get(tokenHash(code)) as any;
   if(!row||row.expires_at<now||row.client_id!==cid||row.redirect_uri!==redirect||row.resource!==resource)return ojson({error:"invalid_grant"},400);
   const digest=createHash("sha256").update(verifier).digest("base64url");
   if(!safeCompare(digest,row.challenge))return ojson({error:"invalid_grant"},400);
   return db.transaction(()=>{
    const consumed=db.query("DELETE FROM oauth_codes WHERE code_hash=?").run(tokenHash(code));
    if(consumed.changes!==1)return ojson({error:"invalid_grant"},400);
    return issueOAuthTokens(cid);
   })();
  }
  if(grant==="refresh_token"){
   const refresh=form.get("refresh_token")||"";
   if(!bounded(refresh,256))return ojson({error:"invalid_grant"},400);
   const row=db.query("SELECT * FROM oauth_tokens WHERE refresh_hash=?").get(tokenHash(refresh)) as any;
   if(!row||row.client_id!==cid||row.resource!==resource||row.refresh_expires_at<now)return ojson({error:"invalid_grant"},400);
   return db.transaction(()=>{
    const removed=db.query("DELETE FROM oauth_tokens WHERE refresh_hash=?").run(tokenHash(refresh));
    if(removed.changes!==1)return ojson({error:"invalid_grant"},400);
    return issueOAuthTokens(cid);
   })();
  }
  return ojson({error:"unsupported_grant_type"},400);
 }
 return null;
}
function issueOAuthTokens(clientId:string){
 const a=secureToken(),r=secureToken(),now=Date.now();
 db.query("INSERT INTO oauth_tokens(access_hash,refresh_hash,client_id,resource,scope,expires_at,refresh_expires_at) VALUES(?,?,?,?,?,?,?)").run(tokenHash(a),tokenHash(r),clientId,resource,oauthScope,now+8*3600000,now+30*86400000);
 return ojson({access_token:a,refresh_token:r,token_type:"Bearer",expires_in:8*3600,scope:oauthScope});
}

const widgetURI="ui://consecom/cloud/live";
const widgetHTML="<!doctype html><html lang=\"pt-BR\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"><style>:root{color-scheme:light dark}body{font:14px/1.5 system-ui,sans-serif; padding:16px; margin:0; background:#101820;color:#f0f6fa}section{background:#1a2936;border:1px solid #334555;padding:14px;border-radius:14px;margin-bottom:12px}h3{margin:0 0 5px}p{margin:8px 0}.muted{color:#a8bfca;font-size:12px}button{background:#147e6b;border:0;border-radius:9px;padding:9px 12px;color:white;cursor:pointer}#status{padding:10px;background:#102d30;border-radius:9px;white-space:pre-wrap;overflow-wrap:anywhere}.entry{border-bottom:1px solid #344758;padding:8px 0;overflow-wrap:anywhere}.entry:last-child{border:0}</style></head><body><section><h3>Consecom Cloud Live</h3><p class=\"muted\">Transporte experimental via MCP. Fila persistente no Railway; a recepção automática exige que este componente permaneça ativo no ChatGPT.</p><label><input id=\"auto\" type=\"checkbox\" checked> Encaminhar mensagens automaticamente</label><p id=\"status\" role=\"status\">Iniciando conexão...</p><button id=\"retry\" type=\"button\">Tentar agora</button></section><section><h3>Eventos recentes</h3><div id=\"items\" class=\"muted\">Nenhum evento recebido ainda.</div></section><script>(function(){\"use strict\";var host=window.openai||{};var auto=document.getElementById(\"auto\"),status=document.getElementById(\"status\"),items=document.getElementById(\"items\"),retry=document.getElementById(\"retry\");var active=true,polling=false,errors=0;var seen=new Set();var delay=function(ms){return new Promise(function(resolve){setTimeout(resolve,ms)})};function unpack(x){if(x&&x.structuredContent)return x.structuredContent;if(x&&x.result&&x.result.structuredContent)return x.result.structuredContent;for(var part of ((x&&x.content)||[])){if(part.type===\"text\"){try{return JSON.parse(part.text)}catch(_){}}}return x||{}}function announce(s){status.textContent=s}function persist(){if(typeof host.setWidgetState===\"function\"){try{host.setWidgetState({automatic:auto.checked})}catch(_){}}}if(host.widgetState&&typeof host.widgetState.automatic===\"boolean\")auto.checked=host.widgetState.automatic;auto.addEventListener(\"change\",function(){persist();if(auto.checked){active=true;poll()}});function report(id,flag){if(typeof host.callTool!==\"function\")return;Promise.resolve(host.callTool(\"consecom_live_reportar_host\",{id:id,status:flag})).catch(function(){})}function show(e){if(seen.has(e.id))return;seen.add(e.id);if(items.textContent===\"Nenhum evento recebido ainda.\")items.textContent=\"\";var item=document.createElement(\"div\");item.className=\"entry\";item.textContent=e.id+\" · \"+e.message;items.prepend(item);while(items.children.length>18)items.lastChild.remove()}function dispatch(e){if(!auto.checked)return;if(typeof host.sendFollowUpMessage!==\"function\"){announce(\"Este host não disponibilizou sendFollowUpMessage. Mensagem reservada, aguardando reenvio seguro.\");report(e.id,\"unavailable\");return}var prompt=\"Mensagem externa recebida pelo Consecom Cloud. ID: \"+e.id+\"\\n\\n\"+e.message+\"\\n\\nResponda à mensagem e publique sua resposta usando a ferramenta MCP consecom_chat_responder com id=\"+e.id+\", texto=RESPOSTA e tipo=final. Não alegue ter executado comandos que não executou.\";announce(\"Enviando \"+e.id+\" ao ChatGPT...\");var submitted;try{submitted=host.sendFollowUpMessage({prompt:prompt,scrollToBottom:false})}catch(err){announce(\"Erro ao solicitar turno: \"+String(err));report(e.id,\"host_rejected\");return}report(e.id,\"attempt\");Promise.resolve(submitted).then(function(r){if(r&&r.isError){report(e.id,\"host_rejected\");announce(\"Host recusou \"+e.id);return}report(e.id,\"host_resolved\");announce(\"ChatGPT aceitou a solicitação de \"+e.id+\"; aguardando resposta publicada.\")}).catch(function(err){report(e.id,\"host_rejected\");announce(\"Falha na entrega: \"+String(err).slice(0,150))})}async function loop(){if(polling||!active)return;polling=true;try{while(active){try{host=window.openai||{};if(typeof host.callTool!==\"function\"){announce(\"Ponte MCP indisponível. Tentarei reconectar.\");await delay(1200);continue}if(!auto.checked){announce(\"Recepção automática pausada pelo usuário.\");await delay(1600);continue}if(typeof host.sendFollowUpMessage!==\"function\"){announce(\"Host sem envio de mensagens. Aguardando recurso disponível...\");await delay(1800);continue}var r=unpack(await host.callTool(\"consecom_live_aguardar_e_assumir\",{wait_ms:5000}));if(r&&r.isError)throw Error(\"Ferramenta recusada\");if(!active)break;errors=0;var owned=new Set(Array.isArray(r.claimed_ids)?r.claimed_ids:[]);for(var e of (Array.isArray(r.events)?r.events:[])){show(e);if(owned.has(e.id))dispatch(e)}if(!r.events||!r.events.length)announce(\"Conectado. Aguardando mensagens no Railway...\")}catch(err){errors=Math.min(errors+1,6);announce(\"Falha temporária, reconectando: \"+String(err).slice(0,110));await delay(Math.min(5000,300*Math.pow(2,errors)))}}}finally{polling=false;if(active)void loop()}}window.addEventListener(\"pagehide\",function(){active=false});window.addEventListener(\"pageshow\",function(){active=true;poll()});document.addEventListener(\"visibilitychange\",function(){if(!document.hidden&&active)poll()});retry.addEventListener(\"click\",function(){active=true;poll()});void loop()})();</script></body></html>";
const json=(v:any,status=200,headers:any={})=>new Response(JSON.stringify(v),{status,headers:{"content-type":"application/json","cache-control":"no-store",...headers}});
const auth=(r:Request,t:string|undefined)=>!!t&&t.length>=24&&r.headers.get("authorization")==="Bearer "+t;

// Prototype-only private binary artifact transfer. Not multi-tenant: never use this with multiple customers.
const artifactMaxBytes=8*1024*1024;
const artifactQuotaBytes=96*1024*1024;
const artifactMime=new Set(["application/pdf","image/png","image/jpeg","text/plain","application/zip"]);
db.exec("CREATE TABLE IF NOT EXISTS cloud_artifacts (id TEXT PRIMARY KEY,file_name TEXT NOT NULL,mime TEXT NOT NULL,body BLOB NOT NULL,byte_size INTEGER NOT NULL,sha256 TEXT NOT NULL,created_at TEXT NOT NULL); CREATE INDEX IF NOT EXISTS cloud_artifacts_created_idx ON cloud_artifacts(created_at);");
const artifactUsage=()=>db.query("SELECT COUNT(*) count,COALESCE(SUM(byte_size),0) used_bytes FROM cloud_artifacts").get() as {count:number,used_bytes:number};
const safeFileName=(raw:string)=>{
 const name=raw.replace(/[^a-zA-Z0-9._-]+/g,"_").slice(0,110).replace(/^\.+/,"");
 return name||"artifact.bin";
};
async function artifactHttp(req:Request,url:URL):Promise<Response|null>{
 const path=url.pathname,method=req.method;
 if(path!=="/bridge/artifacts"&&!path.startsWith("/bridge/artifacts/"))return null;
 if(!auth(req,Bun.env.BRIDGE_INGEST_TOKEN))return json({error:"unauthorized"},401);
 if(path==="/bridge/artifacts"){
  if(method!=="POST")return json({error:"method_not_allowed"},405);
  const mime=(req.headers.get("content-type")||"").split(";")[0].trim().toLowerCase();
  if(!artifactMime.has(mime))return json({error:"unsupported_media_type"},415);
  const contentLen=Number(req.headers.get("content-length")||0);
  if(contentLen>artifactMaxBytes)return json({error:"artifact_too_large",max_bytes:artifactMaxBytes},413);
  let data:Uint8Array;try{data=new Uint8Array(await req.arrayBuffer())}catch{return json({error:"invalid_upload"},400)}
  if(data.length<1||data.length>artifactMaxBytes)return json({error:"invalid_size",max_bytes:artifactMaxBytes},413);
  if(mime==="application/pdf"&&Buffer.from(data.subarray(0,5)).toString("ascii")!=="%PDF-")return json({error:"invalid_pdf_header"},422);
  const usage=artifactUsage();
  if(usage.used_bytes+data.length>artifactQuotaBytes)return json({error:"storage_quota_reached",quota_bytes:artifactQuotaBytes},507);
  const fileName=safeFileName(req.headers.get("x-file-name")||("artifact."+({["application/pdf"]:"pdf",["image/png"]:"png",["image/jpeg"]:"jpg",["text/plain"]:"txt",["application/zip"]:"zip"} as any)[mime]));
  const artifactId="art_"+randomBytes(16).toString("hex");
  const hash=createHash("sha256").update(data).digest("hex");
  db.query("INSERT INTO cloud_artifacts (id,file_name,mime,body,byte_size,sha256,created_at) VALUES(?,?,?,?,?,?,?)").run(artifactId,fileName,mime,Buffer.from(data),data.length,hash,ts());
  return json({id:artifactId,file_name:fileName,mime,bytes:data.length,sha256:hash,download_path:"/bridge/artifacts/"+artifactId,requires_authorization:true,storage:"private"},201);
 }
 const suffix=path.slice("/bridge/artifacts/".length);
 const isMeta=suffix.endsWith("/meta"),id=isMeta?suffix.slice(0,-5):suffix;
 if(!/^art_[a-f0-9]{32}$/.test(id))return json({error:"not_found"},404);
 if(method==="DELETE"&&!isMeta){
  const gone=db.query("DELETE FROM cloud_artifacts WHERE id=?").run(id);
  return gone.changes===1?json({id,deleted:true}):json({error:"not_found"},404);
 }
 if(method!=="GET")return json({error:"method_not_allowed"},405);
 const row=db.query("SELECT * FROM cloud_artifacts WHERE id=?").get(id) as any;
 if(!row)return json({error:"not_found"},404);
 const meta={id:row.id,file_name:row.file_name,mime:row.mime,bytes:row.byte_size,sha256:row.sha256,created_at:row.created_at};
 if(isMeta)return json(meta);
 return new Response(Buffer.from(row.body),{status:200,headers:{"content-type":row.mime,"content-disposition":'attachment; filename="'+safeFileName(row.file_name)+'"',"content-length":String(row.byte_size),"cache-control":"private, no-store","x-content-type-options":"nosniff","x-artifact-sha256":row.sha256}});
}

const schema=(p:any={},req:string[]=[])=>({type:"object",properties:p,required:req});
const tools=[
 {name:"consecom_live_abrir",description:"Abre Consecom Cloud Live nesta conversa. Requer widget ativo e consentimento do usuário. Opcionalmente dispara teste HTTPS.",inputSchema:schema({testar_recepcao:{type:"boolean"},mensagem_teste:{type:"string",maxLength:1500}}),_meta:{ui:{resourceUri:widgetURI},"openai/outputTemplate":widgetURI}},
 {name:"consecom_live_aguardar_e_assumir",description:"Espera ate 5 segundos e reserva eventos pendentes sem prazo de expiração.",inputSchema:schema({wait_ms:{type:"integer",minimum:0,maximum:5500}}),_meta:{"openai/widgetAccessible":true,ui:{visibility:["app"]}}},
 {name:"consecom_live_reportar_host",description:"Registra resultado de envio ao host para permitir recuperação segura.",inputSchema:schema({id:{type:"string"},status:{type:"string",enum:["attempt","host_resolved","host_rejected","unavailable"]}},["id","status"]),_meta:{"openai/widgetAccessible":true,ui:{visibility:["app"]}}},
 {name:"consecom_live_recuperar_pendente",description:"Reenfileira manualmente evento preso por mais de 2 minutos. Pode duplicar turno aceito pelo host; exige confirmação explícita.",inputSchema:schema({id:{type:"string"},confirmar_duplicacao:{type:"boolean"}},["id","confirmar_duplicacao"])},
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
 const oauthResp=await handleOAuth(req,url);if(oauthResp)return oauthResp;
 const fileResponse=await artifactHttp(req,url);if(fileResponse)return fileResponse;
 if(method==="GET"&&url.pathname==="/health"){try{db.query("SELECT 1").get();return json({status:"ok",service:"consecom-cloud-mcp",version:"0.2.0",storage:"sqlite-volume",boots_from_persistent_disk:bootCount,self_test:selfTest})}catch{return json({status:"database_unavailable"},503)}}
 if(url.pathname==="/mcp"){
  if(!oauthValid(req))return unauthorized();
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
   if(name==="consecom_live_abrir"&&arg.testar_recepcao===true){
    if(!Bun.env.BRIDGE_INGEST_TOKEN)throw Error("bridge token unavailable");
    const msg=typeof arg.mensagem_teste==="string"&&arg.mensagem_teste.trim().length>0&&arg.mensagem_teste.length<=1500?arg.mensagem_teste.trim():"TESTE CONSECOM CLOUD. Responda e publique pelo MCP.";
    const external_id="cloud-live-check-"+Date.now()+"-"+randomBytes(4).toString("hex");
    const rr=await fetch(issuer+"/bridge/enqueue",{method:"POST",headers:{"Content-Type":"application/json","Authorization":"Bearer "+Bun.env.BRIDGE_INGEST_TOKEN},body:JSON.stringify({message:msg,external_id}),signal:AbortSignal.timeout(12000)});
    const b=await rr.json() as any;
    if(rr.status!==202||!b.id)throw Error("HTTPS bridge enqueue failed with HTTP "+rr.status);
    val={...summary(),test:{id:b.id,message:msg,ingress:"public HTTPS /bridge/enqueue",status:b.status}};
   }else if(name==="consecom_live_abrir"||name==="consecom_live_status")val=summary();
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
   else if(name==="consecom_live_recuperar_pendente"){
    if(typeof arg.id!=="string"||arg.confirmar_duplicacao!==true)throw Error("explicit duplicate-risk confirmation required");
    const evt=get(arg.id);
    if(!evt||evt.state!=="claimed"||!evt.claimed_at)throw Error("event not pending in claimed state");
    if(Date.now()-Date.parse(evt.claimed_at)<120000)throw Error("wait at least 2 minutes before manual recovery");
    db.query("UPDATE events SET state='queued',claimed_at=NULL,host_status='manual_requeue' WHERE id=? AND state='claimed'").run(arg.id);
    val={ok:true,id:arg.id,state:get(arg.id)?.state,warning:"previous ChatGPT turn may have executed; duplicated response is possible"};
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

  // Private artifact storage round-trip; no public download without the bridge secret.
  const pdfBytes=Buffer.from("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF\n","utf8");
  const downloadAuth={authorization:"Bearer "+bridge};
  const noAuth=await handle(new Request("http://local/bridge/artifacts",{method:"POST",headers:{"content-type":"application/pdf","x-file-name":"banana.pdf"},body:pdfBytes}));
  if(noAuth.status!==401)throw Error("Artifact POST should require authentication");
  steps.push("artifacts-auth-401");
  const upload=await handle(new Request("http://local/bridge/artifacts",{method:"POST",headers:{...downloadAuth,"content-type":"application/pdf","x-file-name":"banana.pdf"},body:pdfBytes}));
  const info=await upload.json() as any;
  if(upload.status!==201||!info.id||info.bytes!==pdfBytes.length)throw Error("Artifact upload failed");
  const artifactId=info.id;steps.push("artifacts-upload");
  try{
   const metadata=await handle(new Request("http://local/bridge/artifacts/"+artifactId+"/meta",{headers:downloadAuth}));
   const met=await metadata.json() as any;
   if(metadata.status!==200||met.sha256!==createHash("sha256").update(pdfBytes).digest("hex"))throw Error("Artifact metadata sha mismatch");
   steps.push("artifacts-metadata");
   const download=await handle(new Request("http://local/bridge/artifacts/"+artifactId,{headers:downloadAuth}));
   const roundtrip=Buffer.from(await download.arrayBuffer());
   if(download.status!==200||!roundtrip.equals(pdfBytes))throw Error("Artifact byte roundtrip failed");
   steps.push("artifacts-download-integrity");
   const unauthorizedGet=await handle(new Request("http://local/bridge/artifacts/"+artifactId));
   if(unauthorizedGet.status!==401)throw Error("Artifact GET must reject anonymous clients");
   steps.push("artifacts-private-download");
  }finally{
   const del=await handle(new Request("http://local/bridge/artifacts/"+artifactId,{method:"DELETE",headers:downloadAuth}));
   if(del.status!==200)throw Error("Artifact cleanup failed");
  }
  steps.push("artifacts-delete");


  // Exercise OAuth DCR, PKCE login, token authorization and refresh against live handlers.
  let selfCid="";
  try{
   const verifier=randomBytes(32).toString("base64url");
   const challenge=createHash("sha256").update(verifier).digest("base64url");
   const callback="https://chatgpt.com/oauth/callback/consecom-selftest";
   const reg=await handle(new Request("http://local/oauth/register",{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({redirect_uris:[callback],token_endpoint_auth_method:"none"})}));
   const regBody=await reg.json() as any;
   if(reg.status!==201||!regBody.client_id)throw Error("oauth-register failed");
   selfCid=regBody.client_id;steps.push("oauth-dcr");
   const authUrl=new URL("http://local/oauth/authorize");
   const params={response_type:"code",client_id:selfCid,redirect_uri:callback,state:"consecom-selftest",code_challenge:challenge,code_challenge_method:"S256",resource,scope:oauthScope};
   Object.entries(params).forEach(([key,value])=>authUrl.searchParams.set(key,value));
   const landing=await handle(new Request(authUrl));
   const html=await landing.text(),fid=/name="flow_id" value="(flow_[^"]+)"/.exec(html)?.[1];
   if(landing.status!==200||!fid)throw Error("oauth-authorize form failed");
   const csp=landing.headers.get("content-security-policy")||"";
   if(!csp.includes("form-action 'self' https://chatgpt.com https://chat.openai.com"))throw Error("OAuth callback CSP blocked");
   steps.push("oauth-chatgpt-csp-callback");
   const login=await handle(new Request("http://local/oauth/authorize",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:new URLSearchParams({flow_id:fid,password:Bun.env.OAUTH_PAIRING_CODE||""})}));
   const location=login.headers.get("Location");
   if(login.status!==303||!location)throw Error("oauth-consent failed");
   const duplicatedLogin=await handle(new Request("http://local/oauth/authorize",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:new URLSearchParams({flow_id:fid,password:Bun.env.OAUTH_PAIRING_CODE||""})}));
   if(duplicatedLogin.status!==409||(await duplicatedLogin.text()).includes("Autorização já utilizada")===false)throw Error("OAuth replay-handling failed");
   steps.push("oauth-replay-friendly-error");
   const code=new URL(location).searchParams.get("code");
   if(!code)throw Error("oauth-code missing");
   steps.push("oauth-consent");
   const exchange=await handle(new Request("http://local/oauth/token",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:new URLSearchParams({grant_type:"authorization_code",client_id:selfCid,redirect_uri:callback,code,code_verifier:verifier,resource})}));
   const credentials=await exchange.json() as any;
   if(exchange.status!==200||!credentials.access_token||!credentials.refresh_token)throw Error("oauth-token exchange failed");
   const list=await handle(new Request("http://local/mcp",{method:"POST",headers:{authorization:"Bearer "+credentials.access_token,"content-type":"application/json"},body:JSON.stringify({jsonrpc:"2.0",id:77,method:"tools/list"})}));
   const listRes=await list.json() as any;
   if(list.status!==200||!listRes.result?.tools?.length)throw Error("oauth-MCP authorization failed");
   steps.push("oauth-mcp-access");
   const rotated=await handle(new Request("http://local/oauth/token",{method:"POST",headers:{"content-type":"application/x-www-form-urlencoded"},body:new URLSearchParams({grant_type:"refresh_token",client_id:selfCid,refresh_token:credentials.refresh_token,resource})}));
   const rotatedBody=await rotated.json() as any;
   if(rotated.status!==200||!rotatedBody.access_token)throw Error("oauth-refresh failed");
   if(oauthValid(new Request("http://local/mcp",{headers:{authorization:"Bearer "+credentials.access_token}})))throw Error("old OAuth access token not revoked after refresh");
   steps.push("oauth-refresh-rotation");
  }finally{if(selfCid){db.query("DELETE FROM oauth_clients WHERE client_id=?").run(selfCid);db.query("DELETE FROM oauth_tokens WHERE client_id=?").run(selfCid);db.query("DELETE FROM oauth_flows WHERE client_id=?").run(selfCid);db.query("DELETE FROM oauth_codes WHERE client_id=?").run(selfCid)}}
  selfTest={status:"passed",steps};
 }catch(err){selfTest={status:"failed",steps,error:String(err)}}
 finally{if(testId)db.query("DELETE FROM events WHERE id=?").run(testId)}
}
void startupSelfTest().then(()=>console.info("consecom-cloud-selftest",JSON.stringify(selfTest))).catch(err=>console.error("consecom-cloud-selftest-error",String(err)));

export default {port:Number(Bun.env.PORT||8080),async fetch(req:Request){try{return await handle(req)}catch(e){console.error("cloud MCP request error:",e);return json({error:"internal_error"},500)}}};
