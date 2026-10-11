package main

import(
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func contextFixture(t *testing.T)(*App,string){
 t.Helper()
 appdata:=t.TempDir()
 t.Setenv("APPDATA",appdata)
 vault:=filepath.Join(t.TempDir(),"My Notes")
 if err:=os.MkdirAll(vault,0700);err!=nil{t.Fatal(err)}
 obsDir:=filepath.Join(appdata,"obsidian")
 if err:=os.MkdirAll(obsDir,0700);err!=nil{t.Fatal(err)}
 bytes,err:=json.Marshal(map[string]any{"vaults":map[string]any{
  "vaultid":{"path":vault,"open":true},
 }})
 if err!=nil{t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(obsDir,"obsidian.json"),bytes,0600);err!=nil{t.Fatal(err)}
 app,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 return app,vault
}

func TestContextVaultChoiceSourceBackfillAndObsidianSync(t *testing.T){
 a,vault:=contextFixture(t)
 _,err:=a.contextSaveNote("bad","Minha ideia","texto",nil)
 if err==nil{t.Fatal("notes before choosing Obsidian vault must fail")}
 v,err:=a.contextRegisterSource("Claude Code","mcp-explicit")
 if err!=nil{t.Fatal(err)}
 s:=v.(map[string]any)["source"].(ContextSource)
 if s.Label!="Claude Code"{t.Fatalf("wrong canonical name: %s",s.Label)}
 if _,err=a.contextRegisterSource("cloud code","mcp-explicit");err!=nil{t.Fatal(err)}
 if result,_:=a.contextGraph();result.(map[string]any)["source_count"]!=1{t.Fatal("source registration not idempotent")}
 vals,err:=obsidianVaults();if err!=nil||len(vals)!=1{t.Fatalf("vault discovery failure: %v %v",vals,err)}
 if _,err=a.contextVaultSelect("unknown-vault");err==nil{t.Fatal("unregistered vault accepted")}
 res,err:=a.contextVaultSelect(vals[0]["id"].(string));if err!=nil{t.Fatal(err)}
 if res.(map[string]any)["selected"]!=true{t.Fatal("vault select failed")}
 base:=filepath.Join(vault,"Consecom","Contexto")
 sourceFile:=filepath.Join(base,"Fonte-claude-code-"+s.ID[:8]+".md")
 if _,err:=os.Stat(sourceFile);err!=nil{t.Fatalf("source Obsidian note missing: %v",err)}
 saved,err:=a.contextSaveNote(s.ID,"Projeto Alpha","Uma nova ideia de automacao",nil)
 if err!=nil{t.Fatal(err)}
 note:=saved.(map[string]any)["note"].(ContextNote)
 body,err:=os.ReadFile(filepath.Join(base,note.Filename));if err!=nil{t.Fatal(err)}
 if !strings.Contains(string(body),"[[Fonte-claude-code-"){t.Fatal("Obsidian backlink missing")}
 if !strings.Contains(string(body),"Uma nova ideia de automacao"){t.Fatal("user approved note text missing")}
 read,err:=a.contextReadNote(note.ID);if err!=nil{t.Fatal(err)}
 if read.(map[string]any)["content"]!=string(body){t.Fatal("read content mismatch")}
 saved2,err:=a.contextSaveNote(s.ID,"Projeto Beta","Relacionada ao Alpha",[]string{note.ID})
 if err!=nil{t.Fatal(err)}
 link:=saved2.(map[string]any)["note"].(ContextNote)
 linked,err:=a.contextReadNote(link.ID);if err!=nil{t.Fatal(err)}
 if !strings.Contains(linked.(map[string]any)["content"],strings.TrimSuffix(note.Filename,".md")){t.Fatal("note relation backlink missing")}
 if _,err:=a.contextSaveNote(s.ID,"X","Y",[]string{"unregistered"});err==nil{t.Fatal("missing relation accepted")}
 if _,err:=a.contextReadNote("abcdef");err==nil{t.Fatal("invalid note id accepted")}
 if _,err:=a.contextReadNote(strings.Repeat("a",32));err==nil{t.Fatal("unregistered note read accepted")}
 // Reboot persists graph metadata, not personal vault content.
 reload,err:=NewApp(a.path);if err!=nil{t.Fatal(err)}
 graph,err:=reload.contextGraph();if err!=nil{t.Fatal(err)}
 if graph.(map[string]any)["note_count"]!=2{t.Fatal("graph missing notes after restart")}
}

func TestContextMCPSourceObservationAndNoAutomaticChatExport(t *testing.T){
 a,_:=contextFixture(t)
 for _,name:=range []string{"claude-code","OpenCode MCP Client","ChatGPT","unrecognized-random-client"}{
  req:=map[string]any{"jsonrpc":"2.0","id":1,"method":"initialize","params":map[string]any{
   "clientInfo":map[string]any{"name":name,"version":"1.0"}}}
  b,_:=json.Marshal(req)
  if _,ok:=a.handleRPC(b);!ok{t.Fatal("initialize rejected")}
 }
 graph,err:=a.contextGraph();if err!=nil{t.Fatal(err)}
 sources:=graph.(map[string]any)["sources"].([]ContextSource)
 if len(sources)!=3{t.Fatalf("only known clients should register: %v",sources)}
 for _,s:=range sources{if s.Origin!="mcp-clientInfo-unverified"{t.Fatal("MCP source must be marked unverified")}}
 if graph.(map[string]any)["note_count"]!=0{t.Fatal("unexpected imported conversation content")}
}

func TestContextInputsCannotEscapeScopedFolder(t *testing.T){
 a,vault:=contextFixture(t)
 if _,err:=contextName("bad"+string(rune(10))+"name");err==nil{t.Fatal("multiline source accepted")}
 if _,err:=contextName(strings.Repeat("a",92));err==nil{t.Fatal("oversized name accepted")}
 v,err:=a.contextRegisterSource("OpenCode","mcp-explicit");if err!=nil{t.Fatal(err)}
 id:=v.(map[string]any)["source"].(ContextSource).ID
 vs,err:=obsidianVaults();if err!=nil{t.Fatal(err)}
 if _,err=a.contextVaultSelect(vs[0]["id"].(string));err!=nil{t.Fatal(err)}
 saved,err:=a.contextSaveNote(id,"../secret.md","Conteudo protegido",nil)
 if err!=nil{t.Fatal(err)}
 n:=saved.(map[string]any)["note"].(ContextNote)
 if strings.Contains(n.Filename,"..")||strings.Contains(n.Filename,"/"){t.Fatalf("unsafe generated filename %q",n.Filename)}
 if _,err:=os.Stat(filepath.Join(vault,"secret.md"));!os.IsNotExist(err){t.Fatal("note escaped authorized folder")}
 if _,err:=a.contextSaveNote(id,"A","",nil);err==nil{t.Fatal("empty content accepted")}
 if _,err:=a.contextSaveNote(id,"A",strings.Repeat("x",24001),nil);err==nil{t.Fatal("oversized content accepted")}
}

func TestHybridBrowserRouteRequiresSafeURL(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 if _,err=a.hybridBrowserOpen("file:///C:/secret");err==nil{t.Fatal("unsafe URL accepted")}
 if _,err=a.hybridBrowserOpen("http://169.254.1.1:3000");err==nil{t.Fatal("internal IP accepted")}
 st,err:=a.hybridBrowserStatus();if err!=nil{t.Fatal(err)}
 if st.(map[string]any)["fallback_engine"]!="consecom-cdp"{t.Fatal("native fallback missing")}
}
func TestContextToolsAreDeclared(t *testing.T){
 required:=map[string]bool{
  "contexto_status":true,"contexto_registrar_fonte":false,
  "contexto_grafo":true,"contexto_salvar_nota":false,
  "contexto_ler_nota":true,"navegador_hibrido_status":true,"navegador_hibrido_abrir":false,
 }
 for _,tool:=range tools(){
  readonly,ok:=required[tool.Name];if !ok{continue}
  if tool.Annotations["readOnlyHint"]!=readonly{t.Errorf("annotation incorrect %s",tool.Name)}
  delete(required,tool.Name)
 }
 if len(required)>0{t.Fatalf("missing tools: %v",required)}
}
