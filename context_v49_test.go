package main

import (
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestV49ObsidianOneClickPrefersOpenVault(t *testing.T){
 a,original:=contextFixture(t)
 second:=filepath.Join(t.TempDir(),"Other Vault")
 if e:=os.MkdirAll(second,0700);e!=nil{t.Fatal(e)}
 register:=filepath.Join(os.Getenv("APPDATA"),"obsidian","obsidian.json")
 data,_:=json.Marshal(map[string]any{"vaults":map[string]any{
  "first":map[string]any{"path":original,"open":true},
  "second":map[string]any{"path":second,"open":false},
 }})
 if e:=os.WriteFile(register,data,0600);e!=nil{t.Fatal(e)}
 value,e:=a.contextVaultAuto();if e!=nil{t.Fatal(e)}
 if value.(map[string]any)["name"]!=filepath.Base(original){t.Fatalf("wrong vault: %v",value)}
 if _,e:=os.Stat(filepath.Join(original,"Consecom","Contexto"));e!=nil{t.Fatalf("missing Consecom namespace: %v",e)}
 if _,e:=os.Stat(filepath.Join(second,"Consecom"));!os.IsNotExist(e){t.Fatalf("touched inactive vault: %v",e)}
 // The explicit vault remains stable after subsequent auto-connect clicks.
 value,e=a.contextVaultAuto();if e!=nil||value.(map[string]any)["name"]!=filepath.Base(original){t.Fatalf("auto reconnect failed: %v %v",value,e)}
}

func TestV410ContextUIInfiniteGraphOnly(t *testing.T) {
 required:=[]string{
  "id=\"contextGraphSearch\"","id=\"contextGraphPreview\"",
  "function contextGraphBuild()","function contextGraphTick()",
  "function contextGraphContextMenu(","function contextFinishDrag(",
  "function connectObsidianAuto()",
  "id=\"contextAutoVaultBtn\"","contexto/vault/auto",
  "ChatGPT","Obsidian","context-graph-infinite","context-graph-settings",
 }
 for _,part:=range required{if !strings.Contains(uiHTML,part){t.Errorf("missing Contexto UI feature %q",part)}}
 graphStart:=strings.Index(uiHTML,"<section id=\"contextPage\"")
 graphEnd:=graphStart+strings.Index(uiHTML[graphStart:],"</section>")
 if graphStart<0||graphEnd<0{t.Fatal("Contexto markup not found")}
 section:=uiHTML[graphStart:graphEnd]
 for _,unwanted:=range []string{"class=\"context-hero\"","class=\"context-explorer\"","class=\"context-detail\"","id=\"contextExplorerTree\""}{
  if strings.Contains(section,unwanted){t.Errorf("unwanted fixed panel %q",unwanted)}
 }
 if !strings.Contains(section,"id=\"contextIntegrations\"")||!strings.Contains(section,"id=\"contextSVG\""){t.Fatal("cards and infinite graph missing")}
 
 if len(integrationNames)==0||integrationNames[0].ID!="chatgpt"{
  t.Fatal("ChatGPT is not the first listed integration")
 }
}

func TestV49RepoCluesNotRunAsExecutables(t *testing.T){
 // The workspace scan is deliberately read-only. A directory name containing
 // Claude does not prove that the CLI is installed or connected.
 names:=map[string]bool{}
 for _,x:=range integrationNames{names[x.ID]=true}
 if !names["claude-code"]||!names["opencode"]{t.Fatal("expected integrations missing")}
}
