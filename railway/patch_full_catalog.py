#!/usr/bin/env python3
"""Patch the existing Consecom Bun Railway Function source for the full Windows MCP catalog.

Usage:
  python railway/patch_full_catalog.py source-index.tsx patched-index.tsx

This is a source transformation, NOT a deployment command.
Inspect output and deploy the entire file with Railway Function's code editor/API.
No credentials are read or written by this script.
"""
from pathlib import Path
import sys

MARKER = "// CONSECOM_DESKTOP_DYNAMIC_CATALOG_V416"

def patch(source: str) -> str:
    if MARKER in source:
        return source

    def change(needle: str, replacement: str):
        nonlocal source
        count = source.count(needle)
        if count != 1:
            raise ValueError(f"Expected exactly one patch location, found {count}: {needle[:95]!r}")
        source = source.replace(needle, replacement)

    change(
        'const desktopWorkerId="owner-desktop";',
        '''// CONSECOM_DESKTOP_DYNAMIC_CATALOG_V416
db.exec("CREATE TABLE IF NOT EXISTS desktop_mcp_catalog (name TEXT PRIMARY KEY, ordinal INTEGER NOT NULL UNIQUE, spec_json TEXT NOT NULL);");
const desktopCatalogRows=()=>db.query("SELECT spec_json FROM desktop_mcp_catalog ORDER BY ordinal").all() as {spec_json:string}[];
const desktopCatalog=()=>desktopCatalogRows().map(row=>JSON.parse(row.spec_json));
const desktopCatalogCount=()=>Number((db.query("SELECT COUNT(*) n FROM desktop_mcp_catalog").get() as any).n);
const desktopCatalogHas=(name:string)=>typeof name==="string"&&!!db.query("SELECT name FROM desktop_mcp_catalog WHERE name=?").get(name);
const desktopWorkerId="owner-desktop";'''
    )
    change(
        "read_only_tools:[...desktopReadOnly],pending_jobs:",
        "read_only_tools:[...desktopReadOnly],registered_tool_count:desktopCatalogCount(),catalog_registered:desktopCatalogCount()>0,pending_jobs:"
    )
    change(
        'if(!desktopReadOnly.has(name))throw Error("Ferramenta nao autorizada pelo proxy de leitura");',
        'if(!desktopReadOnly.has(name)&&!desktopCatalogHas(name))throw Error("Ferramenta nao registrada no relay");'
    )
    change(
        ' if(Number(req.headers.get("content-length")||0)>90000)return json({error:"payload_too_large"},413);\n let a:any;try{const txt=await req.text();if(txt.length>90000)return json({error:"payload_too_large"},413);a=JSON.parse(txt)}catch{return json({error:"invalid_json"},400)}',
        ''' const maxWorkerBytes=path==="/desktop/worker/complete"?12582912:path==="/desktop/worker/register"?262144:90000;
 if(Number(req.headers.get("content-length")||0)>maxWorkerBytes)return json({error:"payload_too_large"},413);
 let a:any;try{const txt=await req.text();if(txt.length>maxWorkerBytes)return json({error:"payload_too_large"},413);a=JSON.parse(txt)}catch{return json({error:"invalid_json"},400)}'''
    )
    change(
        ''' if(path==="/desktop/worker/heartbeat"){
  db.query''',
        ''' if(path==="/desktop/worker/register"){
  if(!Array.isArray(a.tools)||a.tools.length<1||a.tools.length>160)return json({error:"invalid_catalog_size"},400);
  const seen=new Set<string>(),safe:any[]=[];
  for(const spec of a.tools){
   const name=spec?.name;
   if(typeof name!=="string"||! /^[a-z][a-z0-9_]{1,79}$/.test(name)||name.startsWith("consecom_")||seen.has(name))return json({error:"invalid_tool_name"},400);
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
  db.query'''
    )
    change(
        'if(result.length>1600000)return json({error:"result_too_large"},413);',
        'if(result.length>10000000)return json({error:"result_too_large"},413);'
    )
    change(
        'if(q.method==="tools/list")return rpc(q.id,{tools});',
        'if(q.method==="tools/list")return rpc(q.id,{tools:[...tools,...desktopCatalog()]});'
    )
    change(
        'else if(name==="consecom_pc_executar"){val=await desktopMcpExecute(arg.ferramenta,arg.argumentos)}',
        '''else if(name==="consecom_pc_executar"){val=await desktopMcpExecute(arg.ferramenta,arg.argumentos)}
   else if(desktopCatalogHas(name)){
    val=await desktopMcpExecute(name,arg);
    if(val?.ok&&name==="browser_capturar_tela"&&typeof val.resultado?.Data==="string"
       &&/^[A-Za-z0-9+/=]+$/.test(val.resultado.Data)&&val.resultado.Data.length<=10000000)
      return rpc(q.id,{content:[{type:"image",mimeType:"image/jpeg",data:val.resultado.Data}]});
   }'''
    )
    return source

if __name__=="__main__":
    if len(sys.argv)!=3:
        raise SystemExit("Usage: patch_full_catalog.py input-index.tsx output-index.tsx")
    src=Path(sys.argv[1]).read_text(encoding="utf-8")
    output=patch(src)
    Path(sys.argv[2]).write_text(output,encoding="utf-8")
    print(f"Patched Bun function: {len(output)} characters; old: {len(src)}. No secrets changed.")
