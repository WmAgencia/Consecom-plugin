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


// Owner-only outbound Windows desktop bridge. No Cloudflare/OpenAI tunnel required.
// Do not enable this prototype for multiple customers: this instance has one owner token.
db.exec("CREATE TABLE IF NOT EXISTS desktop_workers(id TEXT PRIMARY KEY,last_seen INTEGER NOT NULL); CREATE TABLE IF NOT EXISTS desktop_jobs(id TEXT PRIMARY KEY,tool TEXT NOT NULL,args_json TEXT NOT NULL,state TEXT NOT NULL CHECK(state IN ('queued','claimed','answered','failed')),result_json TEXT,error TEXT,created_at INTEGER NOT NULL,claimed_at INTEGER,finished_at INTEGER); CREATE INDEX IF NOT EXISTS ix_desktop_jobs_state ON desktop_jobs(state,created_at);");
// CONSECOM_DESKTOP_DYNAMIC_CATALOG_V416
db.exec("CREATE TABLE IF NOT EXISTS desktop_mcp_catalog (name TEXT PRIMARY KEY, ordinal INTEGER NOT NULL UNIQUE, spec_json TEXT NOT NULL);");
const desktopCatalog=()=>((db.query("SELECT spec_json FROM desktop_mcp_catalog ORDER BY ordinal").all() as any[]).map(r=>JSON.parse(r.spec_json)));
const desktopCatalogCount=()=>Number((db.query("SELECT COUNT(*) n FROM desktop_mcp_catalog").get() as any).n);
const desktopCatalogHas=(name:string)=>typeof name==="string"&&!!db.query("SELECT name FROM desktop_mcp_catalog WHERE name=?").get(name);
const desktopWorkerId="owner-desktop";
const desktopReadOnly=new Set([
 "diagnostico_pc","listar_processos","pastas_autorizadas","politica_aprovacao",
 "dev_integracoes_status","dev_projeto_inspecionar","dev_servicos_inspecionar",
 "dev_deploy_preflight","listar_arquivos","ler_arquivo","terminal_restrito",
 "agentes_ativos","game_booster_estado"
]);
const desktopOnline=()=>Boolean((db.query("SELECT last_seen FROM desktop_workers WHERE id=?").get(desktopWorkerId) as any)?.last_seen>Date.now()-65000);
const desktopSummary=()=>({transport:"railway-https-outbound",tunnel_required:false,worker_connected:desktopOnline(),read_only_tools:[...desktopReadOnly],registered_tool_count:desktopCatalogCount(),catalog_registered:desktopCatalogCount()>0,pending_jobs:(db.query("SELECT COUNT(*) n FROM desktop_jobs WHERE state IN ('queued','claimed')").get() as any).n});
const desktopResult=(id:string)=>db.query("SELECT state,result_json,error FROM desktop_jobs WHERE id=?").get(id) as any;
async function desktopMcpExecute(name:string,args:any){
 if(!desktopReadOnly.has(name)&&!desktopCatalogHas(name))throw Error("Ferramenta nao registrada no relay");
 if(!desktopOnline())throw Error("Consecom Plugin Windows offline; abra o executavel e conecte ao Railway");
 const data=JSON.stringify(args??{});if(data.length>65000)throw Error("Argumentos excedem o limite");
 if((db.query("SELECT COUNT(*) n FROM desktop_jobs WHERE state IN ('queued','claimed')").get() as any).n>=12)throw Error("Fila do PC ocupada");
 const id="pcj_"+randomBytes(16).toString("hex");
 db.query("INSERT INTO desktop_jobs(id,tool,args_json,state,created_at) VALUES(?,?,?,'queued',?)").run(id,name,data,Date.now());
 const end=Date.now()+28000;
 while(Date.now()<end){
  const result=desktopResult(id);
  if(result?.state==="answered")return {ok:true,id,tool:name,resultado:JSON.parse(result.result_json)};
  if(result?.state==="failed")return {ok:false,id,tool:name,error:result.error};
  await Bun.sleep(300);
 }
 return {ok:false,id,tool:name,timeout:true,message:"O PC nao respondeu em 28s. A tarefa permanece registrada; nao sera executada duas vezes automaticamente."};
}
async function desktopWorkerHttp(req:Request,url:URL):Promise<Response|null>{
 const path=url.pathname;
 if(!path.startsWith("/desktop/worker/"))return null;
 if(!auth(req,Bun.env.CONSECOM_LOCAL_WORKER_TOKEN))return json({error:"unauthorized"},401);
 if(req.method!=="POST")return json({error:"method_not_allowed"},405);
 const maxWorkerBytes=path==="/desktop/worker/complete"?12582912:path==="/desktop/worker/register"?262144:90000;
 if(Number(req.headers.get("content-length")||0)>maxWorkerBytes)return json({error:"payload_too_large"},413);
 let a:any;try{const txt=await req.text();if(txt.length>maxWorkerBytes)return json({error:"payload_too_large"},413);a=JSON.parse(txt)}catch{return json({error:"invalid_json"},400)}
 if(a?.worker_id!==desktopWorkerId)return json({error:"wrong_worker"},404);
 if(path==="/desktop/worker/register"){
  if(!Array.isArray(a.tools)||a.tools.length<1||a.tools.length>160)return json({error:"invalid_catalog_size"},400);
  const seen=new Set<string>(),safe:any[]=[];
  for(const spec of a.tools){
   const name=spec?.name;
   if(typeof name!=="string"||!/^[a-z][a-z0-9_]{1,79}$/.test(name)||name.startsWith("consecom_")||seen.has(name))return json({error:"invalid_tool_name"},400);
   if(typeof spec.description!=="string"||!spec.description||spec.description.length>1500)return json({error:"invalid_description"},400);
   if(!spec.inputSchema||spec.inputSchema.type!=="object"||typeof spec.inputSchema.properties!=="object"||Array.isArray(spec.inputSchema.properties))return json({error:"invalid_schema"},400);
   if(!spec.annotations||typeof spec.annotations.readOnlyHint!=="boolean")return json({error:"invalid_annotations"},400);
   const normalized={name,description:spec.description,inputSchema:spec.inputSchema,annotations:spec.annotations};
   if(JSON.stringify(normalized).length>16000)return json({error:"tool_schema_too_large"},413);
   seen.add(name);safe.push(normalized);
  }
  db.transaction(()=>{
   db.query("DELETE FROM desktop_mcp_catalog").run();
   const put=db.query("INSERT INTO desktop_mcp_catalog(name,ordinal,spec_json) VALUES(?,?,?)");
   safe.forEach((spec,i)=>put.run(spec.name,i,JSON.stringify(spec)));
  })();
  return json({ok:true,tool_count:safe.length,catalog_registered:true});
 }
 if(path==="/desktop/worker/heartbeat"){
  db.query("INSERT INTO desktop_workers(id,last_seen) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET last_seen=excluded.last_seen").run(desktopWorkerId,Date.now());
  return json({ok:true,...desktopSummary()});
 }
 if(path==="/desktop/worker/poll"){
  const claimed=db.transaction(()=>{
   const row=db.query("SELECT id,tool,args_json FROM desktop_jobs WHERE state='queued' ORDER BY created_at LIMIT 1").get() as any;
   if(!row)return null;
   db.query("UPDATE desktop_jobs SET state='claimed',claimed_at=? WHERE id=? AND state='queued'").run(Date.now(),row.id);
   return {id:row.id,tool:row.tool,arguments:JSON.parse(row.args_json)};
  })();
  return json({job:claimed});
 }
 if(path==="/desktop/worker/complete"){
  if(!/^pcj_[0-9a-f]{32}$/.test(String(a.id||"")))return json({error:"bad_id"},400);
  const old=db.query("SELECT state FROM desktop_jobs WHERE id=?").get(a.id) as any;
  if(!old||old.state!=="claimed")return json({error:"not_claimed"},409);
  if(a.error){
   db.query("UPDATE desktop_jobs SET state='failed',error=?,finished_at=? WHERE id=?").run(String(a.error).slice(0,300),Date.now(),a.id);
   return json({ok:true});
  }
  const result=JSON.stringify(a.result??null);
  if(result.length>10000000)return json({error:"result_too_large"},413);
  db.query("UPDATE desktop_jobs SET state='answered',result_json=?,finished_at=? WHERE id=?").run(result,Date.now(),a.id);
  return json({ok:true});
 }
 if(path==="/desktop/worker/send"){
  const msg=a.message,external=a.external_id;
  if(typeof msg!=="string"||!msg.trim()||msg.length>2000||typeof external!=="string"||!/^pc-[0-9a-f]{12}-evt-[0-9]{6}$/.test(external))return json({error:"invalid_message"},400);
  const existing=db.query("SELECT id,state FROM events WHERE external_id=?").get(external) as any;
  if(existing)return json({id:existing.id,status:existing.state,duplicate:true},200);
  if((db.query("SELECT COUNT(*) c FROM events WHERE state<>'answered'").get() as any).c>=10000)return json({error:"queue_full"},429);
  const out=db.query("INSERT INTO events(message,state,created_at,external_id) VALUES(?,'queued',?,?)").run(msg.trim(),ts(),external);
  const id="evt-"+String(Number(out.lastInsertRowid)).padStart(6,"0");
  db.query("UPDATE events SET id=? WHERE seq=?").run(id,Number(out.lastInsertRowid));
  return json({id,status:"queued",persistent:true},202);
 }
 if(path==="/desktop/worker/reply"){
  const external=a.external_id;
  if(typeof external!=="string"||!/^pc-[0-9a-f]{12}-evt-[0-9]{6}$/.test(external))return json({error:"invalid_external_id"},400);
  const row=db.query("SELECT id,state,answer FROM events WHERE external_id=?").get(external) as any;
  return row?json({id:row.id,state:row.state,answer:row.state==="answered"?row.answer:null}):json({error:"not_found"},404);
 }
 return json({error:"not_found"},404);
}

const schema=(p:any={},req:string[]=[])=>({type:"object",properties:p,required:req});
const tools=[
 {name:"consecom_live_abrir",description:"Abre Consecom Cloud Live nesta conversa. Requer widget ativo e consentimento do usuário. Opcionalmente dispara teste HTTPS.",inputSchema:schema({testar_recepcao:{type:"boolean"},mensagem_teste:{type:"string",maxLength:1500}}),_meta:{ui:{resourceUri:widgetURI},"openai/outputTemplate":widgetURI}},
 {name:"consecom_live_aguardar_e_assumir",description:"Espera ate 5 segundos e reserva eventos pendentes sem prazo de expiração.",inputSchema:schema({wait_ms:{type:"integer",minimum:0,maximum:5500}}),_meta:{"openai/widgetAccessible":true,ui:{visibility:["app"]}}},
 {name:"consecom_live_reportar_host",description:"Registra resultado de envio ao host para permitir recuperação segura.",inputSchema:schema({id:{type:"string"},status:{type:"string",enum:["attempt","host_resolved","host_rejected","unavailable"]}},["id","status"]),_meta:{"openai/widgetAccessible":true,ui:{visibility:["app"]}}},
 {name:"consecom_live_recuperar_pendente",description:"Reenfileira manualmente evento preso por mais de 2 minutos. Pode duplicar turno aceito pelo host; exige confirmação explícita.",inputSchema:schema({id:{type:"string"},confirmar_duplicacao:{type:"boolean"}},["id","confirmar_duplicacao"])},
 {name:"consecom_live_status",description:"Informa quantidades por estado, ID e idade da mensagem mais antiga.",inputSchema:schema()},
 {name:"consecom_chat_responder",description:"Registra resposta final para evento reservado, sem duplicar a resposta.",inputSchema:schema({id:{type:"string"},texto:{type:"string",maxLength:6000},tipo:{type:"string",enum:["final"]}},["id","texto","tipo"])},
 {name:"consecom_pc_status",description:"Mostra se o executavel Consecom Plugin no Windows esta conectado ao Railway, sem tunel.",inputSchema:schema()},
 {name:"consecom_pc_ferramentas",description:"Lista ferramentas de leitura e diagnostico autorizadas pelo Consecom remoto.",inputSchema:schema()},
 {name:"consecom_pc_executar",description:"Executa exclusivamente ferramentas locais de LEITURA aprovadas, via HTTPS pelo Railway. Alteracoes no computador nao sao permitidas nesta fase.",inputSchema:schema({ferramenta:{type:"string"},argumentos:{type:"object"}},["ferramenta"])}
];
const rpc=(id:any,result:any)=>json({jsonrpc:"2.0",id,result});
const rpcError=(id:any,code:number,message:string)=>json({jsonrpc:"2.0",id,error:{code,message}});
const get=(id:string)=>db.query("SELECT * FROM events WHERE id=?").get(id) as Row|null;
const counts=()=>db.query("SELECT state,COUNT(*) count FROM events GROUP BY state").all() as any[];
const summary=()=>({storage:"sqlite-volume",durable_queue:true,widget_requires_active_host:true,counts:counts(),oldest_queued:(db.query("SELECT created_at FROM events WHERE state='queued' ORDER BY seq LIMIT 1").get() as any)?.created_at||null,desktop:{...desktopSummary(),server_worker_token_present:!!Bun.env.CONSECOM_LOCAL_WORKER_TOKEN,server_worker_token_format_valid:/^cci_[A-Za-z0-9_-]{28,196}$/.test(Bun.env.CONSECOM_LOCAL_WORKER_TOKEN||"")}});
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

/** Protocol connectivity check only: this is NOT an inference model. */
const gwModel="consecom-connection-test";
const gwText="CONSECOM_CONNECTION_OK. Seu Claude Code alcançou e autenticou o Consecom Cloud no Railway. Esta resposta é uma verificação fixa do transporte, não é inteligência artificial. Ainda não há um modelo de inferência conectado ao endpoint.";
function gatewayCredential(req:Request){
 const expected=Bun.env.CONSECOM_GATEWAY_TEST_KEY||"";
 if(expected.length<40)return false;
 const h=req.headers.get("authorization")||"";
 const fromBearer=h.toLowerCase().startsWith("bearer ")?h.slice(7):"";
 const provided=fromBearer||req.headers.get("x-api-key")||"";
 return provided.length===expected.length && timingSafeEqual(Buffer.from(provided),Buffer.from(expected));
}
const anthropicError=(type:string,message:string,status:number)=>json({type:"error",error:{type,message}},status);
const gatewayMessage=(reqBody:any)=>{
 const id="msg_cc_test_"+randomBytes(10).toString("hex");
 return {id,type:"message",role:"assistant",model:reqBody.model===gatewayDesktopConnectionAlias?gatewayDesktopConnectionAlias:gwModel,content:[{type:"text",text:gwText}],stop_reason:"end_turn",stop_sequence:null,usage:{input_tokens:0,output_tokens:0,cache_creation_input_tokens:0,cache_read_input_tokens:0}};
};
const sseEvent=(name:string,body:any)=>"event: "+name+"\ndata: "+JSON.stringify(body)+"\n\n";

const gatewayBridgeModel="consecom-cloud-bridge-test";
const gatewayDesktopBridgeAlias="claude-consecom-cloud-bridge-test";
const gatewayDesktopConnectionAlias="claude-consecom-connection-test";
const gatewayBridgeWaitMs=42000;
const extractText=(part:any):string|null=>{
 if(typeof part==="string")return part;
 if(!Array.isArray(part))return null;
 if(part.some((p:any)=>p?.type!=="text"))return null;
 return part.map((p:any)=>p.text||"").join("\n");
};
const gatewayBridgeContent=(body:any):string|null=>{
 if(!Array.isArray(body.messages)||body.messages.length<1||body.messages.length>12)return null;
 const last=body.messages.at(-1);
 if(last?.role!=="user")return null;
 const t=extractText(last.content);
 if(!t||t.trim().length<1||t.length>1200)return null;
 return t.trim();
};
const gatewayBridgeReply=(id:string,answer:string,modelId:string=gatewayBridgeModel)=>({
 id:"msg_cc_bridge_"+id.replace(/[^A-Za-z0-9_-]/g,""),type:"message",role:"assistant",
 model:modelId,content:[{type:"text",text:answer}],
 stop_reason:"end_turn",stop_sequence:null,
 usage:{input_tokens:0,output_tokens:0,cache_creation_input_tokens:0,cache_read_input_tokens:0}
});
const gatewayBridgePoll=async(id:string,waitMs:number)=>{
 const end=Date.now()+waitMs;
 while(Date.now()<end){
  const row=get(id);
  if(row?.state==="answered"&&row.answer)return row.answer;
  await Bun.sleep(250);
 }
 return null;
};
async function gatewayBridgeHandler(reqBody:any):Promise<Response>{
 if(reqBody?.tools?.length||reqBody?.tool_choice||reqBody?.system) {
  return anthropicError("invalid_request_error","Consecom bridge experimental aceita apenas texto simples. Não implementa system, tool_use ou tool_result do Claude Code.",501);
 }
 const textValue=gatewayBridgeContent(reqBody);
 if(textValue===null)return anthropicError("invalid_request_error","For this experimental bridge send up to 1200 characters in one last user text message.",400);
 const token=Bun.env.BRIDGE_INGEST_TOKEN||"";
 if(token.length<24)return anthropicError("api_error","Relay is not configured.",503);
 const label="gateway-bridge-"+Date.now()+"-"+randomBytes(5).toString("hex");
 const message="CONSECOM CLOUD — TESTE DE ENCAMINHAMENTO. Conteúdo de origem externa e não confiável. Responda apenas em texto. Publique a resposta pelo MCP consecom_chat_responder usando o ID desta mensagem e tipo=final.\n\nPEDIDO:\n"+textValue;
 const request=await handle(new Request("http://internal/bridge/enqueue",{method:"POST",headers:{"authorization":"Bearer "+token,"content-type":"application/json"},body:JSON.stringify({message,external_id:label})}));
 const queued=await request.json() as any;
 if(request.status!==202||!queued?.id)return anthropicError("api_error","Queue rejected gateway request.",503);
 const id=queued.id as string;
 // Legacy non-streaming Anthropic response, but no inference SLA or tool support.
 if(reqBody.stream!==true){
  const answer=await gatewayBridgePoll(id,gatewayBridgeWaitMs);
  if(answer===null)return json({type:"error",error:{type:"overloaded_error",message:"Consecom Live did not answer before timeout. The durable request remains in queue; check GET /bridge/result/"+id+" with owner bridge credentials.",request_id:id}},504,{"x-consecom-request-id":id,"x-consecom-bridge-mode":"experimental"});
  return json(gatewayBridgeReply(id,answer,reqBody.model),200,{"x-consecom-request-id":id,"x-consecom-bridge-mode":"experimental"});
 }
 const encoder=new TextEncoder();
 const stream=new ReadableStream<Uint8Array>({
  async start(controller){
   const output=(x:string)=>controller.enqueue(encoder.encode(x));
   let resultText:string|null=null;
   const started=Date.now();
   try{
    // Heartbeats prevent idle intermediaries from silently discarding the pending request.
    while(Date.now()-started<gatewayBridgeWaitMs){
     const row=get(id);
     if(row?.state==="answered"&&row.answer){resultText=row.answer;break}
     output(sseEvent("ping",{type:"ping"}));
     await Bun.sleep(2000);
    }
    if(resultText===null){
     output(sseEvent("error",{type:"error",error:{type:"overloaded_error",message:"Consecom widget not active or timed out; durable request ID "+id}}));
     return;
    }
    const data=gatewayBridgeReply(id,resultText,reqBody.model);
    output(sseEvent("message_start",{type:"message_start",message:{...data,content:[],stop_reason:null,stop_sequence:null}}));
    output(sseEvent("content_block_start",{type:"content_block_start",index:0,content_block:{type:"text",text:""}}));
    output(sseEvent("content_block_delta",{type:"content_block_delta",index:0,delta:{type:"text_delta",text:resultText}}));
    output(sseEvent("content_block_stop",{type:"content_block_stop",index:0}));
    output(sseEvent("message_delta",{type:"message_delta",delta:{stop_reason:"end_turn",stop_sequence:null},usage:{output_tokens:0}}));
    output(sseEvent("message_stop",{type:"message_stop"}));
   }catch(_){output(sseEvent("error",{type:"error",error:{type:"api_error",message:"Relay interrupted."}}))}
   finally{controller.close()}
  }
 });
 return new Response(stream,{status:200,headers:{"content-type":"text/event-stream; charset=utf-8","cache-control":"no-cache, no-transform","x-consecom-request-id":id,"x-consecom-bridge-mode":"experimental"}});
}



// Optional local inference backend (Ollama Anthropic-compatible) via an outbound-only worker.
// This does not access ChatGPT consumer sessions or extract outputs from the ChatGPT UI.
db.exec("CREATE TABLE IF NOT EXISTS local_inference_workers (worker_id TEXT PRIMARY KEY,models_json TEXT NOT NULL,seen_at INTEGER NOT NULL);CREATE TABLE IF NOT EXISTS local_inference_jobs(id TEXT PRIMARY KEY,worker_id TEXT NOT NULL,model_id TEXT NOT NULL,request_json TEXT NOT NULL,state TEXT NOT NULL CHECK(state IN ('queued','claimed','answered','failed')),result_json TEXT,error TEXT,created_at INTEGER NOT NULL,leased_at INTEGER,finished_at INTEGER);CREATE INDEX IF NOT EXISTS ix_local_jobs_state ON local_inference_jobs(worker_id,state,created_at);");
const workerId="owner-local";
const localAuth=(req:Request)=>auth(req,Bun.env.CONSECOM_LOCAL_WORKER_TOKEN);
const localModels=()=>{
 const rows=db.query("SELECT models_json FROM local_inference_workers WHERE worker_id=? AND seen_at>?").all(workerId,Date.now()-90000) as any[];
 if(!rows.length)return [] as any[];
 try{
  const a=JSON.parse(rows[0].models_json);
  if(!Array.isArray(a))return [];
  return a.filter((m:any)=>typeof m?.id==="string"&&typeof m?.ollama_id==="string"&&m.id.startsWith("claude-local-")).slice(0,24).map((m:any)=>({
   id:m.id,ollama_id:m.ollama_id,type:"model",display_name:m.display_name||m.ollama_id,created_at:"2026-10-11T00:00:00Z",
   lifecycle:"active",capabilities:{tool_use:true},max_input_tokens:null,max_tokens:null
  }));
 }catch{return [] as any[]}
};
const activeLocalModel=(id:string)=>localModels().find(m=>m.id===id)||null;
const localJob=db.transaction((modelId:string,body:any)=>{
 const id="li_"+randomBytes(16).toString("hex");
 db.query("INSERT INTO local_inference_jobs(id,worker_id,model_id,request_json,state,created_at) VALUES(?,?,?,?,'queued',?)").run(id,workerId,modelId,JSON.stringify(body),Date.now());
 db.query("DELETE FROM local_inference_jobs WHERE finished_at IS NOT NULL AND finished_at<?").run(Date.now()-86400000);
 return id;
});
const localJobResult=(id:string)=>db.query("SELECT state,result_json,error FROM local_inference_jobs WHERE id=?").get(id) as any;
async function localAwait(id:string,limitMs:number){
 const end=Date.now()+limitMs;
 while(Date.now()<end){
  const row=localJobResult(id);
  if(row?.state==="answered"||row?.state==="failed")return row;
  await Bun.sleep(500);
 }
 return null;
}
function responseToSSE(message:any):string{
 const full={...message,type:"message",role:"assistant",content:[],stop_reason:null,stop_sequence:null};
 let out=sseEvent("message_start",{type:"message_start",message:full});
 for(const [index,block] of (message.content||[]).entries()){
  let opening:any;
  let delta:any=null;
  if(block?.type==="text"){opening={type:"text",text:""};delta={type:"text_delta",text:block.text||""}}
  else if(block?.type==="tool_use"){opening={type:"tool_use",id:block.id,name:block.name,input:{}};delta={type:"input_json_delta",partial_json:JSON.stringify(block.input||{})}}
  else if(block?.type==="thinking"){opening={type:"thinking",thinking:""};delta={type:"thinking_delta",thinking:block.thinking||""}}
  else continue;
  out+=sseEvent("content_block_start",{type:"content_block_start",index,content_block:opening});
  if(delta)out+=sseEvent("content_block_delta",{type:"content_block_delta",index,delta});
  out+=sseEvent("content_block_stop",{type:"content_block_stop",index});
 }
 out+=sseEvent("message_delta",{type:"message_delta",delta:{stop_reason:message.stop_reason||"end_turn",stop_sequence:message.stop_sequence||null},usage:{output_tokens:message.usage?.output_tokens||0}});
 return out+sseEvent("message_stop",{type:"message_stop"});
}
async function localInferenceResponse(body:any):Promise<Response>{
 const mdl=activeLocalModel(body.model);
 if(!mdl)return anthropicError("not_found_error","O modelo local não está conectado. Inicie o worker Ollama.",404);
 const pending=(db.query("SELECT COUNT(*) n FROM local_inference_jobs WHERE state IN ('queued','claimed')").get() as any).n;
 if(pending>=8)return anthropicError("overloaded_error","Fila de inferência local cheia.",529);
 const id=localJob(body.model,body);
 if(body.stream!==true){
  const row=await localAwait(id,55000);
  if(!row)return json({type:"error",error:{type:"overloaded_error",message:"Modelo local não respondeu no prazo; verifique o worker."},request_id:id},504,{"x-consecom-request-id":id});
  if(row.state==="failed")return anthropicError("api_error",row.error||"Worker local indisponível",502);
  return json(JSON.parse(row.result_json),200,{"x-consecom-model-status":"local-inference","x-consecom-request-id":id});
 }
 const encoder=new TextEncoder();
 const stream=new ReadableStream({
  async start(controller){
   try{
    const end=Date.now()+120000;let row:any=null;
    while(Date.now()<end){
     row=localJobResult(id);if(row?.state==="answered"||row?.state==="failed")break;
     controller.enqueue(encoder.encode(sseEvent("ping",{type:"ping"})));
     await Bun.sleep(2000);
    }
    if(!row||row.state!=="answered"){
     controller.enqueue(encoder.encode(sseEvent("error",{type:"error",error:{type:"api_error",message:row?.error||"Tempo de resposta local excedido."}})));
    }else controller.enqueue(encoder.encode(responseToSSE(JSON.parse(row.result_json))));
   }catch{controller.enqueue(encoder.encode(sseEvent("error",{type:"error",error:{type:"api_error",message:"Erro no processamento local."}})))}
   finally{controller.close()}
  }
 });
 return new Response(stream,{status:200,headers:{"content-type":"text/event-stream; charset=utf-8","cache-control":"no-store","x-consecom-model-status":"local-inference","x-consecom-request-id":id}});
}
async function localWorkerAPI(req:Request,url:URL):Promise<Response|null>{
 const path=url.pathname,method=req.method;
 if(!path.startsWith("/worker/inference/"))return null;
 if(!localAuth(req))return json({error:"unauthorized"},401);
 if(method!=="POST")return json({error:"method_not_allowed"},405);
 const bytes=Number(req.headers.get("content-length")||0);if(bytes>2200000)return json({error:"payload_too_large"},413);
 let body:any;try{const raw=await req.text();if(raw.length>2200000)return json({error:"payload_too_large"},413);body=JSON.parse(raw)}catch{return json({error:"invalid_json"},400)}
 if(body.worker_id!==workerId)return json({error:"worker_not_found"},404);
 if(path==="/worker/inference/register"){
  if(!Array.isArray(body.models)||body.models.length>24)return json({error:"invalid_models"},400);
  const list=body.models.filter((m:any)=>typeof m?.id==="string"&&/^claude-local-[A-Za-z0-9._-]{2,100}$/.test(m.id)&&typeof m.ollama_id==="string"&&m.ollama_id.length<130).map((m:any)=>({id:m.id,ollama_id:m.ollama_id,display_name:String(m.display_name||m.ollama_id).slice(0,130)}));
  db.query("INSERT INTO local_inference_workers(worker_id,models_json,seen_at) VALUES(?,?,?) ON CONFLICT(worker_id) DO UPDATE SET models_json=excluded.models_json,seen_at=excluded.seen_at").run(workerId,JSON.stringify(list),Date.now());
  registryCache=null;
  return json({ok:true,models:list.length},200);
 }
 if(path==="/worker/inference/poll"){
  db.query("UPDATE local_inference_jobs SET state='queued',leased_at=NULL WHERE worker_id=? AND state='claimed' AND leased_at<?").run(workerId,Date.now()-120000);
  const claim=db.transaction(()=>{
   const row=db.query("SELECT id,request_json,model_id FROM local_inference_jobs WHERE worker_id=? AND state='queued' ORDER BY created_at LIMIT 1").get(workerId) as any;
   if(!row)return null;
   db.query("UPDATE local_inference_jobs SET state='claimed',leased_at=? WHERE id=? AND state='queued'").run(Date.now(),row.id);
   return {id:row.id,model_id:row.model_id,request:JSON.parse(row.request_json)};
  })();
  return json({job:claim});
 }
 if(path==="/worker/inference/complete"){
  if(!/^li_[0-9a-f]{32}$/.test(body.id||""))return json({error:"invalid_id"},400);
  const row=db.query("SELECT id,model_id,state FROM local_inference_jobs WHERE id=? AND worker_id=?").get(body.id,workerId) as any;
  if(!row||row.state!=="claimed")return json({error:"not_claimed"},409);
  if(body.error){db.query("UPDATE local_inference_jobs SET state='failed',error=?,finished_at=? WHERE id=?").run(String(body.error).slice(0,240),Date.now(),body.id);return json({ok:true})}
  const result=body.response;
  if(!result||result.type!=="message"||result.role!=="assistant"||!Array.isArray(result.content)||typeof result.stop_reason!=="string")return json({error:"invalid_anthropic_message"},422);
  result.model=row.model_id;
  const encoded=JSON.stringify(result);if(encoded.length>1600000)return json({error:"response_too_large"},413);
  db.query("UPDATE local_inference_jobs SET state='answered',result_json=?,finished_at=? WHERE id=? AND state='claimed'").run(encoded,Date.now(),body.id);
  return json({ok:true});
 }
 return json({error:"not_found"},404);
}

const gatewayModelInfos=[
 {id:gatewayDesktopConnectionAlias,type:"model",display_name:"Consecom – teste de conexão (sem IA)",created_at:"2026-10-11T00:00:00Z",capabilities:null,max_input_tokens:null,max_tokens:null,lifecycle:"active"},
 {id:gatewayDesktopBridgeAlias,type:"model",display_name:"Consecom – ponte de texto experimental (não é Claude)",created_at:"2026-10-11T00:00:00Z",capabilities:null,max_input_tokens:null,max_tokens:null,lifecycle:"active"},
 {id:gwModel,type:"model",display_name:"Consecom – diagnóstico legado",created_at:"2026-10-11T00:00:00Z",capabilities:null,max_input_tokens:null,max_tokens:null,lifecycle:"active"},
 {id:gatewayBridgeModel,type:"model",display_name:"Consecom – ponte legado",created_at:"2026-10-11T00:00:00Z",capabilities:null,max_input_tokens:null,max_tokens:null,lifecycle:"active"}
];

// Registry of REAL upstream models. Only authenticated upstreams explicitly configured
// in Railway may be queried or routed. The ChatGPT subscription is NOT an upstream API.
type GatewayProvider={name:string,base_url:string,key_env:string,auth_scheme:"x-api-key"|"bearer"};
type GatewayRoute={provider:GatewayProvider,upstreamId:string};
const providerConfig=():GatewayProvider[]=>{
 let items:any[]=[];
 try{const parsed=JSON.parse(Bun.env.CONSECOM_UPSTREAM_PROVIDERS_JSON||"[]");if(Array.isArray(parsed))items=parsed.slice(0,8)}catch{return []}
 return items.filter(v=>{
  if(!v||typeof v!=="object"||typeof v.name!=="string"||typeof v.base_url!=="string"||typeof v.key_env!=="string")return false;
  if(v.name.length>70||!/^CONSECOM_UPSTREAM_[A-Z0-9_]+$/.test(v.key_env))return false;
  if(v.auth_scheme!=="x-api-key"&&v.auth_scheme!=="bearer")return false;
  try{const u=new URL(v.base_url);return u.protocol==="https:"&&!u.username&&!u.password&&!u.hash&&!u.search&&u.hostname.length>4&&u.hostname!=="localhost"}catch{return false}
 }).map(v=>({name:v.name,base_url:v.base_url.replace(/\/+$/,""),key_env:v.key_env,auth_scheme:v.auth_scheme}));
};
const upstreamHeaders=(def:GatewayProvider)=>{
 const secret=Bun.env[def.key_env]||"";
 if(secret.length<12)return null;
 return {header:def.auth_scheme==="bearer"?"authorization":"x-api-key",value:def.auth_scheme==="bearer"?"Bearer "+secret:secret};
};
type GatewayRegistry={until:number,models:any[],routes:Map<string,GatewayRoute>,providers:{name:string,status:string,model_count:number}[]};
let registryCache:GatewayRegistry|null=null;
const normalizeUpstreamModel=(entry:any)=>{
 if(!entry||typeof entry!=="object"||typeof entry.id!=="string"||!/^[-A-Za-z0-9_./:]{2,150}$/.test(entry.id))return null;
 return {type:"model",id:entry.id,display_name:typeof entry.display_name==="string"?entry.display_name.slice(0,120):entry.id,
  created_at:typeof entry.created_at==="string"?entry.created_at:"1970-01-01T00:00:00Z",
  capabilities:entry.capabilities&&typeof entry.capabilities==="object"?entry.capabilities:null,
  max_input_tokens:Number.isSafeInteger(entry.max_input_tokens)&&entry.max_input_tokens>0?entry.max_input_tokens:null,
  max_tokens:Number.isSafeInteger(entry.max_tokens)&&entry.max_tokens>0?entry.max_tokens:null,
  lifecycle:["active","deprecated","retired"].includes(entry.lifecycle)?entry.lifecycle:"active",
  deprecated_at:typeof entry.deprecated_at==="string"?entry.deprecated_at:null};
};
async function availableModelRegistry(force=false):Promise<GatewayRegistry>{
 if(!force&&registryCache&&Date.now()<registryCache.until)return registryCache;
 const models:any[]=[...gatewayModelInfos],routes=new Map<string,GatewayRoute>(),providers:GatewayRegistry["providers"]=[];
 const used=new Set<string>(models.map(x=>x.id));
 for(const m of localModels()){if(!used.has(m.id)){models.push(m);used.add(m.id)}}
 for(const def of providerConfig()){
  const cred=upstreamHeaders(def);
  if(!cred){providers.push({name:def.name,status:"missing_credential",model_count:0});continue}
  let after="",found=0,ok=false;
  try{
   for(let page=0;page<10;page++){
    const endpoint=new URL(def.base_url+"/v1/models");
    endpoint.searchParams.set("limit","100");
    if(after)endpoint.searchParams.set("after_id",after);
    const response=await fetch(endpoint.toString(),{headers:{[cred.header]:cred.value,"anthropic-version":"2023-06-01","accept":"application/json"},redirect:"error",signal:AbortSignal.timeout(8000)});
    if(!response.ok)throw Error("HTTP "+response.status);
    const body=await response.json() as any;
    if(!Array.isArray(body.data))throw Error("bad models response");
    for(const raw of body.data){
     const entry=normalizeUpstreamModel(raw);
     if(!entry||used.has(entry.id)||entry.lifecycle==="retired")continue;
     used.add(entry.id);models.push(entry);routes.set(entry.id,{provider:def,upstreamId:entry.id});found++;
    }
    ok=true;
    if(!body.has_more||!body.last_id||body.last_id===after)break;
    after=body.last_id;
   }
   providers.push({name:def.name,status:ok?"reachable":"unknown",model_count:found});
  }catch(err){providers.push({name:def.name,status:"discovery_error:"+String(err).slice(0,65),model_count:found})}
 }
 // Prefer real inference models over diagnostic transport-only entries.
 const experimental=new Set([gwModel,gatewayBridgeModel,gatewayDesktopConnectionAlias,gatewayDesktopBridgeAlias]);
 models.sort((x,y)=>Number(experimental.has(x.id))-Number(experimental.has(y.id)));
 registryCache={until:Date.now()+90000,models,routes,providers};
 return registryCache;
}
async function upstreamMessage(req:Request,body:any):Promise<Response|null>{
 if(!body||typeof body.model!=="string")return null;
 const registry=await availableModelRegistry();
 const route=registry.routes.get(body.model);if(!route)return null;
 const credential=upstreamHeaders(route.provider);
 if(!credential)return anthropicError("authentication_error","Upstream credential missing",503);
 const headers:Record<string,string>={"content-type":"application/json","accept":"application/json, text/event-stream","anthropic-version":req.headers.get("anthropic-version")||"2023-06-01",[credential.header]:credential.value};
 const beta=req.headers.get("anthropic-beta");if(beta)headers["anthropic-beta"]=beta;
 try{
  const upstream=await fetch(route.provider.base_url+"/v1/messages",{method:"POST",headers,body:JSON.stringify({...body,model:route.upstreamId}),redirect:"error",signal:AbortSignal.timeout(180000)});
  const contentType=upstream.headers.get("content-type")||"application/json";
  // Stream bytes unchanged, including tool_use/tool_result.
  return new Response(upstream.body,{status:upstream.status,headers:{"content-type":contentType,"cache-control":"no-store","x-consecom-model-route":route.provider.name}});
 }catch{return anthropicError("api_error","Configured inference provider is unavailable.",502)}
}
const paginateModels=(items:any[],url:URL)=>{
 let list=items.filter(x=>x.lifecycle!=="retired");
 const lifecycle=url.searchParams.getAll("lifecycle");
 if(lifecycle.length)list=list.filter(x=>lifecycle.includes(x.lifecycle));
 const after=url.searchParams.get("after_id"),before=url.searchParams.get("before_id");
 if(after){const i=list.findIndex(x=>x.id===after);if(i<0)return null;list=list.slice(i+1)}
 if(before){const i=list.findIndex(x=>x.id===before);if(i<0)return null;list=list.slice(0,i)}
 const requested=Number(url.searchParams.get("limit")||20);
 if(!Number.isInteger(requested)||requested<1||requested>1000)return null;
 const page=list.slice(0,requested);
 return {data:page,has_more:list.length>page.length,first_id:page[0]?.id||null,last_id:page.at(-1)?.id||null};
};

const gatewayCORS=(response:Response)=>{
 const headers=new Headers(response.headers);
 headers.set("Access-Control-Allow-Origin","*");
 headers.set("Access-Control-Allow-Methods","GET, POST, OPTIONS");
 headers.set("Access-Control-Allow-Headers","x-api-key, authorization, anthropic-version, anthropic-beta, content-type, accept");
 headers.set("Access-Control-Expose-Headers","x-consecom-request-id, x-consecom-model-status");
 headers.set("Access-Control-Max-Age","600");
 return new Response(response.body,{status:response.status,headers});
};
const gatewayDiagnostic=async(req:Request)=>{const registry=await availableModelRegistry();return json({
 type:"gateway_diagnostic",status:"reachable_and_authenticated",auth_scheme:"x-api-key",
 provider:"Consecom Cloud gateway",inference_available:(registry.routes.size>0||localModels().length>0),local_worker_online:localModels().length>0,local_model_count:localModels().length,
 message:"O gateway autentica requisicoes e transporta texto pelo MCP. Nao fornece uma LLM nativa nem tool_use/tool_result para Claude Desktop.",
 available_models:registry.models.map((m:any)=>m.id),
 discovery_compatible_aliases:[gatewayDesktopConnectionAlias,gatewayDesktopBridgeAlias],
 test_model:gwModel,bridge_model:gatewayBridgeModel,
 models_endpoint:"/v1/models",messages_endpoint:"/v1/messages",
 caveat:"A ponte ChatGPT nao substitui uma API de inferencia. Modelos de provedores autenticamente conectados sao repassados a montante.",
 discovered_upstream_models:registry.routes.size,providers:registry.providers,
 configured_upstream_count:providerConfig().length,
 model_ids:registry.models.map((m:any)=>m.id)
},200,{"cache-control":"private, no-store"});};

async function consecomTestGateway(req:Request,url:URL):Promise<Response|null>{
 const method=req.method,path=url.pathname;
 if(path!=="/v1/messages"&&path!=="/v1/models"&&!path.startsWith("/v1/models/")&&path!=="/v1/messages/count_tokens"&&path!=="/gateway/diagnostics")return null;
 if(method==="OPTIONS")return new Response(null,{status:204});
 if(!gatewayCredential(req))return anthropicError("authentication_error","Consecom Cloud gateway: chave ausente ou inválida.",401);
 if(path==="/gateway/diagnostics"){
  if(method!=="GET")return anthropicError("invalid_request_error","Method not allowed",405);
  return await gatewayDiagnostic(req);
 }
 if(path==="/v1/models"){
  if(method!=="GET")return anthropicError("invalid_request_error","Method not allowed",405);
  const registry=await availableModelRegistry();const page=paginateModels(registry.models,url);return page?json(page):anthropicError("invalid_request_error","Invalid model page cursor or limit",400);
 }
 if(path.startsWith("/v1/models/")){
  if(method!=="GET")return anthropicError("invalid_request_error","Method not allowed",405);
  const mid=decodeURIComponent(path.slice("/v1/models/".length));
  const registry=await availableModelRegistry();const model=registry.models.find((m:any)=>m.id===mid);
  return model?json(model):anthropicError("not_found_error","Modelo nao disponivel neste gateway de teste. Nenhuma inferencia real configurada.",404);
 }
 if(method!=="POST")return anthropicError("invalid_request_error","Method not allowed",405);
 const cl=Number(req.headers.get("content-length")||0);
 if(cl>131072)return anthropicError("request_too_large","Payload exceeds 128 KiB connectivity-test limit.",413);
 let body:any;try{const raw=await req.text();if(raw.length>131072)return anthropicError("request_too_large","Payload exceeds limit",413);body=JSON.parse(raw)}catch{return anthropicError("invalid_request_error","Invalid JSON body",400)}
 if(!body||typeof body!=="object"||Array.isArray(body))return anthropicError("invalid_request_error","Invalid Messages API request",400);
 if(path==="/v1/messages/count_tokens"){
  // Count endpoint must not invent model token usage. Explicitly unsupported.
  return anthropicError("invalid_request_error","Token counting requires a genuine inference model; this is connectivity-only.",501);
 }
 if(activeLocalModel(body.model))return localInferenceResponse(body);
 const upstream=await upstreamMessage(req,body);if(upstream)return upstream;
 if(body.model===gatewayBridgeModel||body.model===gatewayDesktopBridgeAlias)return gatewayBridgeHandler(body);
 if(body.model!==gwModel&&body.model!==gatewayDesktopConnectionAlias)return anthropicError("not_found_error","No inference model is configured; available routes are diagnostic only.",404);
 if(!Array.isArray(body.messages))return anthropicError("invalid_request_error","messages must be an array",400);
 if(body.tools&&Array.isArray(body.tools)&&body.tools.length>0)return anthropicError("invalid_request_error","This connection test does not support Claude Code tool_use or tool_result. It is not an LLM.",501);
 const data=gatewayMessage(body);
 if(body.stream!==true)return json(data,200,{"x-consecom-model-status":"connectivity-test-only"});
 const start={...data,content:[],stop_reason:null,stop_sequence:null,usage:{input_tokens:0,output_tokens:0}};
 const events=[
  sseEvent("message_start",{type:"message_start",message:start}),
  sseEvent("content_block_start",{type:"content_block_start",index:0,content_block:{type:"text",text:""}}),
  sseEvent("ping",{type:"ping"}),
  sseEvent("content_block_delta",{type:"content_block_delta",index:0,delta:{type:"text_delta",text:gwText}}),
  sseEvent("content_block_stop",{type:"content_block_stop",index:0}),
  sseEvent("message_delta",{type:"message_delta",delta:{stop_reason:"end_turn",stop_sequence:null},usage:{output_tokens:0}}),
  sseEvent("message_stop",{type:"message_stop"})
 ];
 return new Response(events.join(""),{status:200,headers:{"content-type":"text/event-stream; charset=utf-8","cache-control":"no-cache, no-transform","connection":"keep-alive","x-consecom-model-status":"connectivity-test-only"}});
}

async function handle(req:Request):Promise<Response>{
 const url=new URL(req.url),method=req.method;
 const oauthResp=await handleOAuth(req,url);if(oauthResp)return oauthResp;
 const desktop=await desktopWorkerHttp(req,url);if(desktop)return desktop;
 const worker=await localWorkerAPI(req,url);if(worker)return worker;
 const gateway=await consecomTestGateway(req,url);if(gateway)return gatewayCORS(gateway);
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
  if(q.method==="tools/list")return rpc(q.id,{tools:[...tools,...desktopCatalog()]});
  if(q.method==="resources/list")return rpc(q.id,{resources:[{uri:widgetURI,name:"Consecom Cloud Live",mimeType:"text/html;profile=mcp-app"}]});
  if(q.method==="resources/read")return q.params?.uri===widgetURI?rpc(q.id,{contents:[{uri:widgetURI,mimeType:"text/html;profile=mcp-app",text:widgetHTML}]}):rpcError(q.id,-32602,"Unknown resource");
  if(q.method!=="tools/call")return rpcError(q.id,-32601,"Unknown method");
  const name=q.params?.name,arg=q.params?.arguments||{};
  try{
   let val:any;
   if(name==="consecom_pc_status"){val=desktopSummary()}
   else if(name==="consecom_pc_ferramentas"){val={...desktopSummary(),note:"Ferramentas remotas somente leitura; acesso aos arquivos limitado a pastas autorizadas no Windows."}}
   else if(name==="consecom_pc_executar"){val=await desktopMcpExecute(arg.ferramenta,arg.argumentos)}
   else if(desktopCatalogHas(name)){
    val=await desktopMcpExecute(name,arg);
    if(val?.ok&&name==="browser_capturar_tela"&&typeof val.resultado?.Data==="string"
       &&/^[A-Za-z0-9+/=]+$/.test(val.resultado.Data)&&val.resultado.Data.length<=10000000)
      return rpc(q.id,{content:[{type:"image",mimeType:"image/jpeg",data:val.resultado.Data}]});
   }
   else if(name==="consecom_live_abrir"&&arg.testar_recepcao===true){
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
 let desktopTestJobId="";
 let desktopPriorSeen:number|null=null;
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

  const denyDesktop=await handle(new Request("http://local/desktop/worker/heartbeat",{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({worker_id:desktopWorkerId})}));
  if(denyDesktop.status!==401)throw Error("desktop-worker unauthenticated request not rejected");
  steps.push("desktop-worker-auth-401");
  const desktopToken=Bun.env.CONSECOM_LOCAL_WORKER_TOKEN||"";
  if(desktopToken.length<30)throw Error("desktop-worker secret is not configured");
  const dH={authorization:"Bearer "+desktopToken,"content-type":"application/json"};
  desktopPriorSeen=(db.query("SELECT last_seen FROM desktop_workers WHERE id=?").get(desktopWorkerId) as any)?.last_seen??null;
  const ping=await handle(new Request("http://local/desktop/worker/heartbeat",{method:"POST",headers:dH,body:JSON.stringify({worker_id:desktopWorkerId})}));
  if(ping.status!==200)throw Error("desktop-worker heartbeat failed");
  steps.push("desktop-worker-heartbeat");
  const queuedPC=await handle(new Request("http://local/desktop/worker/send",{method:"POST",headers:dH,body:JSON.stringify({worker_id:desktopWorkerId,message:"[Desktop cloud relay self-test]",external_id:"pc-000000000000-evt-999999"})}));
  const pcMessage=await queuedPC.json() as any;
  if(queuedPC.status!==202||!pcMessage.id)throw Error("desktop outbound message enqueue failed");
  db.query("DELETE FROM events WHERE id=?").run(pcMessage.id);
  steps.push("desktop-worker-send");
  const fakeRequest=handle(new Request("http://local/mcp",{method:"POST",headers:rpcHeaders,body:JSON.stringify({jsonrpc:"2.0",id:98,method:"tools/call",params:{name:"consecom_pc_executar",arguments:{ferramenta:"diagnostico_pc",argumentos:{}}}})}));
  let pendingPC:any=null;
  for(let n=0;n<45;n++){
   const poll=await handle(new Request("http://local/desktop/worker/poll",{method:"POST",headers:dH,body:JSON.stringify({worker_id:desktopWorkerId})}));
   const parsed=await poll.json() as any;if(parsed.job){pendingPC=parsed.job;break}
   await Bun.sleep(25);
  }
  if(!pendingPC?.id||pendingPC.tool!=="diagnostico_pc")throw Error("desktop MCP job was not claimed");
  desktopTestJobId=pendingPC.id;
  steps.push("desktop-mcp-job-claim");
  const finish=await handle(new Request("http://local/desktop/worker/complete",{method:"POST",headers:dH,body:JSON.stringify({worker_id:desktopWorkerId,id:desktopTestJobId,result:{prova:"LOCAL_PC_ROUNDTRIP_OK"}})}));
  if(finish.status!==200)throw Error("desktop job completion rejected");
  const final=await fakeRequest;const roundtrip=await final.json() as any;
  if(!roundtrip.result?.structuredContent?.ok||roundtrip.result?.structuredContent?.resultado?.prova!=="LOCAL_PC_ROUNDTRIP_OK")throw Error("desktop MCP call did not complete roundtrip");
  steps.push("desktop-mcp-roundtrip");

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

  // A working HTTP protocol smoke-test must not be misrepresented as model inference.
  const authTestHeader={authorization:"Bearer "+(Bun.env.CONSECOM_GATEWAY_TEST_KEY||""),"content-type":"application/json"};
  const testReq={model:gwModel,max_tokens:100,messages:[{role:"user",content:"banana"}]};
  const denied=await handle(new Request("http://local/v1/messages",{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify(testReq)}));
  if(denied.status!==401)throw Error("Anthropic test gateway must reject missing key");
  steps.push("gateway-401");
  const apiKeyProbe=await handle(new Request("http://local/v1/messages",{method:"POST",headers:{"x-api-key":Bun.env.CONSECOM_GATEWAY_TEST_KEY||"","content-type":"application/json"},body:JSON.stringify(testReq)}));
  const probeBody=await apiKeyProbe.json() as any;
  if(apiKeyProbe.status!==200||probeBody.model!==gwModel||!probeBody.content?.[0]?.text?.includes("CONSECOM_CONNECTION_OK"))throw Error("Gateway x-api-key header authentication failed");
  steps.push("gateway-x-api-key");

  const optionsResp=await handle(new Request("http://local/v1/messages",{method:"OPTIONS",headers:{"origin":"null","access-control-request-method":"POST","access-control-request-headers":"x-api-key,content-type,anthropic-version"}}));
  if(optionsResp.status!==204||optionsResp.headers.get("access-control-allow-origin")!=="*"||!(optionsResp.headers.get("access-control-allow-headers")||"").includes("x-api-key"))throw Error("CORS preflight for native desktop gateway failed");
  steps.push("gateway-cors-preflight");
  const modelsResp=await handle(new Request("http://local/v1/models",{method:"GET",headers:{"x-api-key":Bun.env.CONSECOM_GATEWAY_TEST_KEY||""}}));
  const modelsBody=await modelsResp.json() as any;
  if(modelsResp.status!==200||modelsBody?.data?.length!==4||modelsBody.data?.some((m:any)=>m.type!=="model"))throw Error("Gateway models discovery failed");
  steps.push("gateway-models-list");
  if(!modelsBody.data.some((m:any)=>m.id===gatewayDesktopBridgeAlias)||!modelsBody.data.some((m:any)=>m.id===gatewayDesktopConnectionAlias))throw Error("Claude Desktop model aliases missing");
  steps.push("gateway-desktop-discovery-aliases");
  const singleResp=await handle(new Request("http://local/v1/models/"+gwModel,{method:"GET",headers:{"x-api-key":Bun.env.CONSECOM_GATEWAY_TEST_KEY||""}}));
  const single=await singleResp.json() as any;
  if(singleResp.status!==200||single.id!==gwModel)throw Error("Gateway model retrieve failed");
  steps.push("gateway-model-details");
  const aliasMsg=await handle(new Request("http://local/v1/messages",{method:"POST",headers:{"x-api-key":Bun.env.CONSECOM_GATEWAY_TEST_KEY||"","content-type":"application/json"},body:JSON.stringify({...testReq,model:gatewayDesktopConnectionAlias})}));
  const aliasPayload=await aliasMsg.json() as any;
  if(aliasMsg.status!==200||aliasPayload.model!==gatewayDesktopConnectionAlias||!aliasPayload.content?.[0]?.text?.startsWith("CONSECOM_CONNECTION_OK"))throw Error("Claude Desktop alias response mismatch");
  steps.push("gateway-desktop-alias-message");
  const diagResp=await handle(new Request("http://local/gateway/diagnostics",{method:"GET",headers:{"x-api-key":Bun.env.CONSECOM_GATEWAY_TEST_KEY||""}}));
  const diag=await diagResp.json() as any;
  if(diagResp.status!==200||diag.inference_available!==(localModels().length>0||(await availableModelRegistry()).routes.size>0)||diag.auth_scheme!=="x-api-key")throw Error("Gateway diagnostics must match real inference availability");
  steps.push("gateway-no-inference-diagnostics");
  const registry=await availableModelRegistry(true);
  if(registry.models.length<4||registry.routes.size!==0||registry.providers.length!==0)throw Error("Default registry must only include diagnostic models when no upstream is configured");
  const norm=normalizeUpstreamModel({id:"test/known-model",display_name:"Test",created_at:"2026-01-01T00:00:00Z",max_input_tokens:200000,max_tokens:64000});
  if(!norm||norm.max_input_tokens!==200000||norm.type!=="model")throw Error("Model metadata normalization failed");
  const page=paginateModels(registry.models,new URL("http://local/v1/models?limit=2"));
  if(!page||page.data.length!==2||page.has_more!==true)throw Error("Model list pagination failed");
  steps.push("gateway-dynamic-catalog-normalization");

  // Exercise the outbound-worker protocol without executing any local commands.
  const wc=Bun.env.CONSECOM_LOCAL_WORKER_TOKEN||"";
  if(wc.length<30)throw Error("Worker token missing");
  const wrkHeaders={"authorization":"Bearer "+wc,"content-type":"application/json"};
  const wbody=(extra:any)=>JSON.stringify({worker_id:workerId,...extra});
  const noWorkerAuth=await handle(new Request("http://local/worker/inference/poll",{method:"POST",headers:{"content-type":"application/json"},body:wbody({})}));
  if(noWorkerAuth.status!==401)throw Error("Worker auth missing 401");steps.push("local-worker-auth-401");
  const existing=db.query("SELECT * FROM local_inference_workers WHERE worker_id=?").get(workerId) as any;
  const testModelId="claude-local-railway-testonly";
  let localId="";
  if(!existing||existing.seen_at<Date.now()-90000){
   try{
    const wr=await handle(new Request("http://local/worker/inference/register",{method:"POST",headers:wrkHeaders,body:wbody({models:[{id:testModelId,ollama_id:"testonly",display_name:"Local Test"}]})}));
    if(wr.status!==200)throw Error("Worker registration failed");
    steps.push("local-worker-register");
    const discovered=await availableModelRegistry(true);
    if(!discovered.models.some((m:any)=>m.id===testModelId))throw Error("Local model not discovered");
    steps.push("local-worker-model-discovery");
    const promise=handle(new Request("http://local/v1/messages",{method:"POST",headers:authTestHeader,body:JSON.stringify({model:testModelId,max_tokens:100,tools:[{name:"Bash",input_schema:{type:"object",properties:{command:{type:"string"}}}}],messages:[{role:"user",content:"Use Bash"}]})}));
    let claim:any=null;
    for(let attempt=0;attempt<30;attempt++){
     const poll=await handle(new Request("http://local/worker/inference/poll",{method:"POST",headers:wrkHeaders,body:wbody({})}));
     const body=await poll.json() as any;
     if(body.job){claim=body.job;break}
     await Bun.sleep(40);
    }
    if(!claim?.id)throw Error("Worker polling did not claim queued job");
    localId=claim.id;steps.push("local-worker-job-claim");
    const mock={id:"msg_local_test",type:"message",role:"assistant",model:"testonly",content:[{type:"tool_use",id:"toolu_local_test",name:"Bash",input:{command:"echo Consecom"}}],stop_reason:"tool_use",stop_sequence:null,usage:{input_tokens:10,output_tokens:20}};
    const complete=await handle(new Request("http://local/worker/inference/complete",{method:"POST",headers:wrkHeaders,body:wbody({id:localId,response:mock})}));
    if(complete.status!==200)throw Error("Worker completion rejected");steps.push("local-worker-tool-result");
    const response=await promise,output=await response.json() as any;
    if(response.status!==200||output.model!==testModelId||output.stop_reason!=="tool_use"||output.content?.[0]?.name!=="Bash")throw Error("Gateway did not relay tool_use response");
    const streamed=responseToSSE(output);
    if(!streamed.includes("input_json_delta")||!streamed.includes("toolu_local_test")||!streamed.includes("message_stop"))throw Error("tool_use streaming encoder incomplete");
    steps.push("local-worker-tool-use-roundtrip");
   }finally{
    if(localId)db.query("DELETE FROM local_inference_jobs WHERE id=?").run(localId);
    if(existing)db.query("INSERT INTO local_inference_workers(worker_id,models_json,seen_at) VALUES(?,?,?) ON CONFLICT(worker_id) DO UPDATE SET models_json=excluded.models_json,seen_at=excluded.seen_at").run(existing.worker_id,existing.models_json,existing.seen_at);
    else db.query("DELETE FROM local_inference_workers WHERE worker_id=?").run(workerId);
    registryCache=null;
   }
  }else steps.push("local-worker-mock-skipped-active-client");


  const accepted=await handle(new Request("http://local/v1/messages",{method:"POST",headers:authTestHeader,body:JSON.stringify(testReq)}));
  const acceptedBody=await accepted.json() as any;
  if(accepted.status!==200||acceptedBody.type!=="message"||acceptedBody.model!==gwModel||!acceptedBody.content?.[0]?.text?.startsWith("CONSECOM_CONNECTION_OK"))throw Error("Anthropic static protocol test failed");
  steps.push("gateway-nonstream-static");
  const streamed=await handle(new Request("http://local/v1/messages",{method:"POST",headers:authTestHeader,body:JSON.stringify({...testReq,stream:true})}));
  const streamPayload=await streamed.text();
  if(streamed.status!==200||!streamed.headers.get("content-type")?.startsWith("text/event-stream")||!streamPayload.includes("event: message_stop")||!streamPayload.includes("CONSECOM_CONNECTION_OK"))throw Error("Anthropic test SSE protocol failed");
  steps.push("gateway-sse-static");
  const unsupported=await handle(new Request("http://local/v1/messages",{method:"POST",headers:authTestHeader,body:JSON.stringify({...testReq,model:"claude-sonnet"})}));
  if(unsupported.status!==404)throw Error("Unknown real model must be rejected");
  steps.push("gateway-reject-fake-inference");

  // Gateway -> same durable Cloud queue -> MCP claim -> ChatGPT-style response -> gateway result.
  const bridgeInput={model:gatewayBridgeModel,max_tokens:64,messages:[{role:"user",content:"bridge-diagnostic"}],stream:false};
  let queuedId="";
  try{
   const pendingResponse=handle(new Request("http://local/v1/messages",{method:"POST",headers:authTestHeader,body:JSON.stringify(bridgeInput)}));
   for(let i=0;i<24;i++){
    const found=db.query("SELECT id FROM events WHERE external_id LIKE 'gateway-bridge-%' AND state='queued' ORDER BY seq DESC LIMIT 1").get() as any;
    if(found?.id){queuedId=found.id;break}
    await Bun.sleep(50);
   }
   if(!queuedId)throw Error("gateway did not enqueue into Consecom Cloud");
   steps.push("gateway-cloud-enqueue");
   const claimedResp=await getCall(88,"consecom_live_aguardar_e_assumir",{wait_ms:1});
   const claimJson=await claimedResp.json() as any;
   if(!claimJson.result?.structuredContent?.claimed_ids?.includes(queuedId))throw Error("Gateway message not available to MCP widget");
   steps.push("gateway-cloud-mcp-claim");
   const replyResp=await getCall(89,"consecom_chat_responder",{id:queuedId,texto:"BRIDGE_INTERNAL_ROUNDTRIP_OK",tipo:"final"});
   const replyData=await replyResp.json() as any;
   if(!replyData.result?.structuredContent?.ok)throw Error("MCP response not persisted");
   const gatewayReply=await pendingResponse;
   const gatewayData=await gatewayReply.json() as any;
   if(gatewayReply.status!==200||gatewayData.model!==gatewayBridgeModel||gatewayData.content?.[0]?.text!=="BRIDGE_INTERNAL_ROUNDTRIP_OK")throw Error("Gateway did not return Cloud MCP response");
   steps.push("gateway-cloud-response-roundtrip");
  }finally{if(queuedId)db.query("DELETE FROM events WHERE id=?").run(queuedId)}




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
 finally{
  if(testId)db.query("DELETE FROM events WHERE id=?").run(testId);
  if(desktopTestJobId)db.query("DELETE FROM desktop_jobs WHERE id=?").run(desktopTestJobId);
  if(desktopPriorSeen===null)db.query("DELETE FROM desktop_workers WHERE id=?").run(desktopWorkerId);
  else db.query("UPDATE desktop_workers SET last_seen=? WHERE id=?").run(desktopPriorSeen,desktopWorkerId);
 }
}
void startupSelfTest().then(()=>console.info("consecom-cloud-selftest",JSON.stringify(selfTest))).catch(err=>console.error("consecom-cloud-selftest-error",String(err)));

export default {port:Number(Bun.env.PORT||8080),async fetch(req:Request){try{return await handle(req)}catch(e){console.error("cloud MCP request error:",e);return json({error:"internal_error"},500)}}};
