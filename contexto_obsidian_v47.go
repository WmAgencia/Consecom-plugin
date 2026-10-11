package main

import (
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "regexp"
 "sort"
 "strings"
 "time"
)

// Contexto uses plain Markdown plus Obsidian wiki-links. It never indexes
// personal notes in the vault and only reads/writes Consecom/Contexto.
type ContextSource struct{
 ID string `json:"id"`
 Label string `json:"label"`
 Origin string `json:"origin"`
 Created time.Time `json:"created"`
}
type ContextNote struct{
 ID string `json:"id"`
 SourceID string `json:"source_id"`
 Title string `json:"title"`
 Filename string `json:"filename"`
 Links []string `json:"links,omitempty"`
 Created time.Time `json:"created"`
}
type obsidianVaultRecord struct{
 Path string `json:"path"`
 Open bool `json:"open"`
}
var contextValidID=regexp.MustCompile(`^[a-z0-9]{32}$`)
func contextHasBadControl(s string)bool{
 for _,r:=range s{if r<32 && r!=9 && r!=10 && r!=13{return true}}
 return false
}

func contextName(name string)(string,error){
 name=strings.TrimSpace(name)
 if name==""||len([]rune(name))>90||contextHasBadControl(name)||strings.ContainsAny(name,string(rune(10))+string(rune(13))){
  return "",errors.New("titulo ou fonte vazia, longa ou com caracteres de controle")
 }
 return name,nil
}
func contextOrigin(raw string)string{
 v:=strings.ToLower(strings.TrimSpace(raw))
 switch {
 case strings.Contains(v,"claude")||strings.Contains(v,"cloud code"): return "Claude Code"
 case strings.Contains(v,"opencode")||strings.Contains(v,"open code"):return "OpenCode"
 case strings.Contains(v,"chatgpt")||strings.Contains(v,"openai chat"):return "ChatGPT"
 case strings.Contains(v,"codex"):return "Codex"
 case strings.Contains(v,"cursor"):return "Cursor"
 default:return strings.TrimSpace(raw)
 }
}
func contextSlug(s string)string{
 s=strings.ToLower(strings.TrimSpace(s))
 var out []rune
 dash:=false
 for _,r:=range s{
  if (r>='a'&&r<='z')||(r>='0'&&r<='9'){out=append(out,r);dash=false
  }else if !dash{out=append(out,'-');dash=true}
  if len(out)>=44{break}
 }
 v:=strings.Trim(string(out),"-")
 if v==""{return "contexto"};return v
}
func obsidianVaults()([]map[string]any,error){
 dir:=os.Getenv("APPDATA")
 if dir==""{return []map[string]any{},nil}
 file:=filepath.Join(dir,"obsidian","obsidian.json")
 bytes,err:=os.ReadFile(file)
 if errors.Is(err,os.ErrNotExist){return []map[string]any{},nil}
 if err!=nil{return nil,err}
 if len(bytes)>1048576{return nil,errors.New("registro Obsidian muito grande")}
 var data struct{Vaults map[string]obsidianVaultRecord `json:"vaults"`}
 if err=json.Unmarshal(bytes,&data);err!=nil{return nil,errors.New("registro do Obsidian invalido")}
 out:=[]map[string]any{}
 for _,v:=range data.Vaults{
  path:=filepath.Clean(v.Path)
  if !filepath.IsAbs(path)||isBroadOrDangerousRoot(path)||ensureNoLinks(path)!=nil{continue}
  st,err:=os.Stat(path);if err!=nil||!st.IsDir(){continue}
  h:=sha256.Sum256([]byte(strings.ToLower(path)))
  out=append(out,map[string]any{"id":hex.EncodeToString(h[:12]),"name":filepath.Base(path),"path":path,"open":v.Open})
 }
 sort.Slice(out,func(i,j int)bool{return out[i]["name"].(string)<out[j]["name"].(string)})
 return out,nil
}
func (a *App) contextObserveMCPClient(raw string){
 normalized:=strings.ToLower(strings.TrimSpace(raw))
 recognized:=false
 for _,part:=range []string{"claude","cloud code","opencode","open code","chatgpt","openai","codex","cursor"}{
  if strings.Contains(normalized,part){recognized=true;break}
 }
 if recognized{_,_=a.contextRegisterSource(contextOrigin(raw),"mcp-clientInfo-unverified")}
}
func syncContextSources(vault string,sources []ContextSource)error{
 dir,err:=contextDirectory(vault);if err!=nil{return err}
 for _,source:=range sources{
  if !contextValidID.MatchString(source.ID){return errors.New("ID de fonte invalido")}
  file:=filepath.Join(dir,"Fonte-"+contextSlug(source.Label)+"-"+source.ID[:8]+".md")
  if err:=ensureNoLinks(file);err!=nil{return err}
  if _,err:=os.Stat(file);err==nil{continue}else if !errors.Is(err,os.ErrNotExist){return err}
  content:=fmt.Sprintf(`# %s

Fonte MCP declarada em %s.

> Identidade informada pelo cliente; nao autenticada.
`,source.Label,source.Created.Format(time.RFC3339))
  if err:=writeExclusive(file,content);err!=nil{return err}
 }
 return nil
}
func (a *App) contextVaultSelect(id string)(any,error){
 vs,err:=obsidianVaults();if err!=nil{return nil,err}
 for _,v:=range vs{
  if v["id"]!=id{continue}
  path:=v["path"].(string)
  // Selection is only exposed by the CSRF-protected local panel, not MCP.
  a.mu.Lock()
  sources:=append([]ContextSource(nil),a.cfg.ContextSources...)
  a.mu.Unlock()
  if err:=syncContextSources(path,sources);err!=nil{return nil,err}
  a.mu.Lock();defer a.mu.Unlock()
  old:=a.cfg.ContextVault
  a.cfg.ContextVault=path
  if err:=a.saveLocked();err!=nil{a.cfg.ContextVault=old;return nil,err}
  return map[string]any{"selected":true,"name":v["name"],"sync_mode":"Consecom/Contexto only"},nil
 }
 return nil,errors.New("cofre nao reconhecido: selecione um dos cofres detectados")
}
func (a *App) contextVaultPath()(string,error){
 a.mu.Lock();root:=a.cfg.ContextVault;a.mu.Unlock()
 if root==""{return "",errors.New("selecione um cofre Obsidian no painel Contexto antes de sincronizar")}
 vs,err:=obsidianVaults();if err!=nil{return "",err}
 matched:=false
 for _,v:=range vs{if filepath.Clean(v["path"].(string))==filepath.Clean(root){matched=true;break}}
 if !matched{return "",errors.New("cofre Obsidian nao esta mais disponivel no registro local")}
 if ensureNoLinks(root)!=nil{return "",errors.New("cofre contem links nao permitidos")}
 return root,nil
}
func contextDirectory(vault string)(string,error){
 sub:=filepath.Join(vault,"Consecom","Contexto")
 if err:=ensureNoLinks(vault);err!=nil{return "",err}
 if err:=ensureNoLinks(filepath.Join(vault,"Consecom"));err!=nil{return "",err}
 if err:=ensureNoLinks(sub);err!=nil{return "",err}
 if err:=os.MkdirAll(sub,0700);err!=nil{return "",err}
 return sub,nil
}
func writeExclusive(path,body string)error{
 if err:=ensureNoLinks(path);err!=nil{return err}
 file,err:=os.OpenFile(path,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600)
 if err!=nil{return err}
 _,err=file.WriteString(body)
 closeErr:=file.Close()
 if err!=nil{_ = os.Remove(path);return err}
 if closeErr!=nil{_ = os.Remove(path);return closeErr}
 return nil
}
func (a *App) contextRegisterSource(label string,origin string)(any,error){
 label=contextOrigin(label)
 var err error
 label,err=contextName(label);if err!=nil{return nil,err}
 if origin==""{origin="mcp-declared"}
 if len(origin)>40{origin=origin[:40]}
 a.mu.Lock()
 for _,s:=range a.cfg.ContextSources{if strings.EqualFold(s.Label,label){
  a.mu.Unlock()
  return map[string]any{"source":s,"existing":true,"identity_verified":false},nil
 }}
 if len(a.cfg.ContextSources)>=40{a.mu.Unlock();return nil,errors.New("limite de 40 fontes atingido")}
 source:=ContextSource{ID:newID(),Label:label,Origin:origin,Created:time.Now().UTC()}
 a.cfg.ContextSources=append(a.cfg.ContextSources,source)
 if err=a.saveLocked();err!=nil{a.cfg.ContextSources=a.cfg.ContextSources[:len(a.cfg.ContextSources)-1];a.mu.Unlock();return nil,err}
 a.mu.Unlock()
 // If the user has selected an Obsidian vault, register the source as a .md
 // file. Do not read any existing notes or create a vault automatically.
 if vault,err:=a.contextVaultPath();err==nil{
  if syncErr:=syncContextSources(vault,[]ContextSource{source});syncErr!=nil{
    return map[string]any{"source":source,"existing":false,"identity_verified":false,
      "sync_error":"Fonte cadastrada, mas o arquivo Obsidian falhou: "+syncErr.Error()},nil
  }
 }
 return map[string]any{"source":source,"existing":false,"identity_verified":false},nil
}
func (a *App) contextGraph()(any,error){
 a.mu.Lock()
 sources:=append([]ContextSource{},a.cfg.ContextSources...)
 notes:=append([]ContextNote{},a.cfg.ContextNotes...)
 selected:=a.cfg.ContextVault
 a.mu.Unlock()
 name:=""
 if selected!=""{name=filepath.Base(selected)}
 return map[string]any{"sources":sources,"notes":notes,"vault_selected":selected!="",
  "vault_name":name,"note_count":len(notes),"source_count":len(sources),
  "limits":"Somente notas criadas pelo Consecom. Historico total do ChatGPT nao e acessivel pelo MCP automaticamente."},nil
}
func (a *App) contextStatus()(any,error){
 vs,err:=obsidianVaults();if err!=nil{return nil,err}
 graph,err:=a.contextGraph();if err!=nil{return nil,err}
 index,indexErr:=a.contextIndex()
 if indexErr!=nil{index=map[string]any{"status":"error","source":indexErr.Error()}}
 return map[string]any{"detected_vaults":vs,"graph":graph,"index":index,"obsidian_detected":len(vs)>0,
  "notes_sync_requires_selection":true,
  "history_import":"Para reunir conversas antigas do ChatGPT, use um export fornecido pelo usuario ou resumos explicitamente enviados via MCP."},nil
}
func (a *App) contextSaveNote(sourceID,title,body string,links []string)(any,error){
 title,err:=contextName(title);if err!=nil{return nil,err}
 if len(body)>24000||len(strings.TrimSpace(body))==0||contextHasBadControl(body){return nil,errors.New("nota vazia, invalida ou maior que 24 KB")}
 if len(links)>12{return nil,errors.New("limite de 12 referencias por nota")}
 vault,err:=a.contextVaultPath();if err!=nil{return nil,err}
 a.mu.Lock()
 var source *ContextSource
 for _,s:=range a.cfg.ContextSources{if s.ID==sourceID{t:=s;source=&t;break}}
 if source==nil{a.mu.Unlock();return nil,errors.New("fonte inexistente; registre a fonte primeiro")}
 if len(a.cfg.ContextNotes)>=2000{a.mu.Unlock();return nil,errors.New("limite de 2000 notas")}
 existing:=map[string]ContextNote{}
 for _,n:=range a.cfg.ContextNotes{existing[n.ID]=n}
 visited:=map[string]bool{}
 for _,l:=range links{
  if !contextValidID.MatchString(l)||existing[l].ID==""{a.mu.Unlock();return nil,errors.New("referencia a nota inexistente")}
  if visited[l]{a.mu.Unlock();return nil,errors.New("referencia duplicada")}
  visited[l]=true
 }
 id:=newID()
 meta:=ContextNote{ID:id,SourceID:source.ID,Title:title,Filename:contextSlug(title)+"-"+id[:8]+".md",
  Links:append([]string{},links...),Created:time.Now().UTC()}
 a.mu.Unlock()
 dir,err:=contextDirectory(vault);if err!=nil{return nil,err}
 // Note filenames never come from user paths. No existing note is overwritten.
 newline:=string(rune(10))
 output:="# "+title+newline+newline+body+newline+newline+"---"+newline+"Origem: [[Fonte-"+contextSlug(source.Label)+"-"+source.ID[:8]+"]]"+newline
 for _,target:=range links{n:=existing[target];output+="Relacionado: [["+strings.TrimSuffix(n.Filename,".md")+"]]"+newline}
 target:=filepath.Join(dir,meta.Filename)
 if err:=writeExclusive(target,output);err!=nil{return nil,err}
 a.mu.Lock()
 a.cfg.ContextNotes=append(a.cfg.ContextNotes,meta)
 if err:=a.saveLocked();err!=nil{
  a.cfg.ContextNotes=a.cfg.ContextNotes[:len(a.cfg.ContextNotes)-1];a.mu.Unlock()
  _=os.Remove(target)
  return nil,err
 }
 a.mu.Unlock()
 return map[string]any{"note":meta,"saved_in_obsidian":true,"folder":"Consecom/Contexto"},nil
}
func (a *App) contextReadNote(id string)(any,error){
 if !contextValidID.MatchString(id){return nil,errors.New("id de nota invalido")}
 vault,err:=a.contextVaultPath();if err!=nil{return nil,err}
 a.mu.Lock()
 var found *ContextNote
 for _,n:=range a.cfg.ContextNotes{if n.ID==id{v:=n;found=&v;break}}
 a.mu.Unlock()
 if found==nil{return nil,errors.New("nota nao cadastrada no Consecom")}
 dir:=filepath.Join(vault,"Consecom","Contexto")
 if ensureNoLinks(dir)!=nil{return nil,errors.New("pasta contem links simbolicos")}
 target:=filepath.Join(dir,found.Filename)
 if err:=ensureNoLinks(target);err!=nil{return nil,err}
 st,err:=os.Stat(target);if err!=nil{return nil,err}
 if !st.Mode().IsRegular()||st.Size()>28000{return nil,errors.New("nota irregular ou grande demais")}
 b,err:=os.ReadFile(target);if err!=nil{return nil,err}
 return map[string]any{"note":found,"content":string(b)},nil
}
