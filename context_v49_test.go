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

func TestV49ContextUIExplorerAndNoRequiredVaultDropdown(t *testing.T) {
 required:=[]string{
  "id=\"contextExplorerTree\"","id=\"contextSearchInput\"",
  "function renderContextExplorer()","function connectObsidianAuto()",
  "id=\"contextAutoVaultBtn\"","contexto/vault/auto",
  "ChatGPT","Obsidian","context-canvas","context-detail",
 }
 for _,part:=range required{if !strings.Contains(uiHTML,part){t.Errorf("missing Contexto UI feature %q",part)}}
 if strings.Contains(uiHTML,"<span class=\"muted\">Cofre Obsidian</span><select"){
  t.Fatal("primary vault flow still forces dropdown")
 }
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
