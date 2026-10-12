// Consecom local inference worker — Node.js 20+
// Executes Anthropic-compatible LLM calls on the user's own local Ollama server.
// Outbound-only HTTPS, no shell/desktop commands and no ChatGPT subscription extraction.
const gateway=(process.env.CONSECOM_BASE_URL||"https://consecom-mcp-relay-production.up.railway.app").replace(/\/+$/,"");
const token=process.env.CONSECOM_LOCAL_WORKER_TOKEN;
const ollama="http://127.0.0.1:11434";
const worker_id="owner-local";
if(!token||token.length<30){console.error("Missing CONSECOM_LOCAL_WORKER_TOKEN; copy from Railway service Variables.");process.exit(2)}
if(!gateway.startsWith("https://")){console.error("CONSECOM_BASE_URL must use HTTPS");process.exit(2)}
const sleep=ms=>new Promise(r=>setTimeout(r,ms));
const req=async(path,data)=>{
 const r=await fetch(gateway+path,{method:"POST",headers:{"content-type":"application/json","authorization":"Bearer "+token},body:JSON.stringify({...data,worker_id}),signal:AbortSignal.timeout(25000)});
 if(!r.ok)throw Error(path+" HTTP "+r.status+" "+(await r.text()).slice(0,150));
 return r.json();
};
let models=new Map();let lastRegistry=0;
async function refresh(){
 const r=await fetch(ollama+"/api/tags",{signal:AbortSignal.timeout(10000)});
 if(!r.ok)throw Error("Ollama /api/tags HTTP "+r.status);
 const data=await r.json();if(!Array.isArray(data.models))throw Error("Invalid Ollama model list");
 const next=new Map();
 for(const m of data.models.slice(0,24)){
  const name=m.name||m.model;
  if(typeof name!=="string"||name.length>130)continue;
  const alias="claude-local-"+name.toLowerCase().replace(/[^a-z0-9._-]+/g,"-").replace(/^-+|-+$/g,"");
  if(alias.length>115||next.has(alias))continue;
  next.set(alias,name);
 }
 const list=[...next].map(([id,ollama_id])=>({id,ollama_id,display_name:"Modelo local: "+ollama_id}));
 const result=await req("/worker/inference/register",{models:list});
 models=next;lastRegistry=Date.now();
 console.log("Consecom local: "+result.models+" modelos registrados: "+list.map(x=>x.id).join(", "));
}
async function runJob(job){
 const raw=job.request||{};
 const ollamaName=models.get(job.model_id);
 if(!ollamaName)throw Error("Modelo não disponível localmente: "+job.model_id);
 const body={...raw,model:ollamaName,stream:false};
 // Ollama implements the Anthropic Messages subset directly, including tool use/results.
 const res=await fetch(ollama+"/v1/messages",{method:"POST",headers:{"content-type":"application/json","x-api-key":"ollama","anthropic-version":"2023-06-01"},body:JSON.stringify(body),signal:AbortSignal.timeout(113000)});
 if(!res.ok)throw Error("Ollama /v1/messages HTTP "+res.status+": "+(await res.text()).slice(0,250));
 const answer=await res.json();
 if(answer?.type!=="message"||answer?.role!=="assistant"||!Array.isArray(answer.content))throw Error("Resposta Ollama não é mensagem Anthropic válida");
 return answer;
}
console.log("Worker de inferência Consecom iniciado. Gateway:",gateway);
while(true){
 try{
  if(Date.now()-lastRegistry>25000)await refresh();
  if(models.size===0){console.log("Nenhum modelo Ollama instalado. Execute: ollama pull qwen3:4b");await sleep(12000);continue}
  const response=await req("/worker/inference/poll",{});
  const job=response.job;
  if(!job){await sleep(1400);continue}
  console.log("Inferência recebida:",job.id,job.model_id);
  try{
   const answer=await runJob(job);
   await req("/worker/inference/complete",{id:job.id,response:answer});
   console.log("Inferência concluída:",job.id,answer.stop_reason);
  }catch(error){
   console.error("Modelo falhou:",job.id,String(error));
   try{await req("/worker/inference/complete",{id:job.id,error:String(error).slice(0,240)})}catch(inner){console.error("Falha ao comunicar erro:",String(inner))}
  }
 }catch(error){console.error("Gateway/Ollama offline:",String(error));await sleep(5000)}
}
