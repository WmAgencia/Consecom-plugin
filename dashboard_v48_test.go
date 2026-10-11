package main

import(
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func TestDashboardV48RealServerMetrics(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 a.startedAt=time.Now().Add(-489*time.Minute)
 status:=a.statusExtra()
 count,ok:=status["active_tools"].(int)
 if !ok||count!=len(tools())||count<=0{t.Fatalf("MCP count not from active catalog: %#v",status)}
 sec,ok:=status["uptime_seconds"].(int64)
 if !ok||sec<489*60||sec>489*60+3{t.Fatalf("invalid server uptime seconds: %#v",status)}
}

func TestDashboardV48ContextSourcesNotFakeOnline(t *testing.T){
 a,err:=NewApp(filepath.Join(t.TempDir(),"config.json"));if err!=nil{t.Fatal(err)}
 initial:=a.dashboardContext()
 if initial["source_count"]!=0||initial["note_count"]!=0{t.Fatal("new install must not invent sources")}
 _,err=a.contextRegisterSource("Claude Code","mcp-explicit");if err!=nil{t.Fatal(err)}
 info:=a.dashboardContext()
 if info["source_count"]!=1||info["note_count"]!=0{t.Fatalf("real context metadata lost: %#v",info)}
 sources:=info["sources"].([]map[string]string)
 if len(sources)!=1||sources[0]["label"]!="Claude Code"{t.Fatal("source mapping incorrect")}
 if _,exists:=sources[0]["connected"];exists{t.Fatal("must not represent registration as live online connection")}
}

func TestDashboardV48EmeraldAndNoMultiIDEOrBrowserPanel(t *testing.T){
 required:=[]string{
  "--bg:#061410", ".context-home", "id=\"homeContextSources\"",
  "id=\"homeSourceCount\"","id=\"homeNoteCount\"",
  "id=\"sideUptime\"","id=\"sideTools\"","id=\"toolsActive\"",
  "id=\"liveAgentsPanel\"","id=\"liveAgentCount\"",
  "formatUptimeSeconds(","activeAgentList()","renderContextHome(",
  "Ferramentas ativas:", "Tempo de atividade:",
  "onclick=\"page('context')\"","id=\"navContext\"",
  "Laboratório de Agentes","openModal('settings')",
 }
 for _,part:=range required{if !strings.Contains(uiHTML,part){t.Errorf("required UI element missing: %s",part)}}
 forbidden:=[]string{
  "Multi IDE", "Orquestração Multi-IDE", "<h2>Navegador e perfis</h2>",
  "id=\"navMultiIDE\"","id=\"previewCanvas\"",
  "mark('browserSummary'", "getElementById('previewWorld'",
 }
 for _,part:=range forbidden{if strings.Contains(uiHTML,part){t.Errorf("obsolete dashboard element present: %s",part)}}
 if strings.Count(uiHTML,`id="navContext"`)!=1{t.Fatal("Contexto is the single unified sidebar entry")}
}

func TestDashboardV48AgentFreshnessAndUptimeNode(t *testing.T){
 node,err:=exec.LookPath("node");if err!=nil{t.Skip("Node not available")}
 start:=strings.Index(uiHTML,"function activeAgentList(){")
 end:=strings.Index(uiHTML,"function renderAgents(){",start)
 if start<0||end<start{t.Fatal("agent filters missing")}
 source:=uiHTML[start:end]
 script:=`
 const current = new Date().toISOString();
 const stale = new Date(Date.now()-10*60000).toISOString();
 const state={agents:[
  {id:'a',status:'running',updated_at:current},
  {id:'b',status:'running',updated_at:stale},
  {id:'c',status:'done',updated_at:current},
  {id:'d',status:'waiting',updated_at:current},
 ]};
 `+source+`
 const active=activeAgentList();
 if(active.length!==1||active[0].id!=='a')throw Error('Agent view must show only fresh running clients');
 if(formatUptimeSeconds(489*60)!=='8h 09min')throw Error('489min should render 8h 09min');
 console.log('dashboard-agent-filter-ok');
 `
 file:=filepath.Join(t.TempDir(),"dashboard-v48-test.js")
 if err:=os.WriteFile(file,[]byte(script),0600);err!=nil{t.Fatal(err)}
 cmd:=exec.Command(node,file)
 out,err:=cmd.CombinedOutput()
 if err!=nil{t.Fatalf("dashboard JS rejected: %s: %v",string(out),err)}
}
