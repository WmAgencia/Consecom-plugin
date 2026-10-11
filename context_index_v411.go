package main

import (
 "errors"
 "io/fs"
 "os"
 "path/filepath"
 "sort"
 "strings"
 "sync"
 "time"
)

// Metadata-only, read-only index. No personal note contents or hidden trees.
type contextFolderIndex struct {
 Path string `json:"path"`
 Name string `json:"name"`
 Count int `json:"count"`
}
type contextFileIndex struct {
 Path string `json:"path"`
 Folder string `json:"folder"`
 Title string `json:"title"`
}
type contextIndexSnapshot struct {
 Status string `json:"status"`
 Total int `json:"total"`
 Scanned int `json:"scanned"`
 Folders []contextFolderIndex `json:"folders"`
 Files []contextFileIndex `json:"files"`
 Complete bool `json:"complete"`
 UpdatedAt time.Time `json:"updated_at,omitempty"`
 Source string `json:"source"`
}
var contextVaultIndexCache struct {
 sync.Mutex
 Vault string
 Expires time.Time
 Running bool
 Snapshot contextIndexSnapshot
}
func contextIndexCacheGet(vault string) contextIndexSnapshot {
 contextVaultIndexCache.Lock()
 defer contextVaultIndexCache.Unlock()
 if vault==""{return contextIndexSnapshot{Status:"disconnected",Source:"Sem cofre Obsidian autorizado"}}
 if vault!=contextVaultIndexCache.Vault{
  contextVaultIndexCache.Vault=vault
  contextVaultIndexCache.Expires=time.Time{}
  contextVaultIndexCache.Running=false
  contextVaultIndexCache.Snapshot=contextIndexSnapshot{Status:"indexing",Source:"Obsidian (nomes de arquivos Markdown)"}
 }
 if time.Now().After(contextVaultIndexCache.Expires)&&!contextVaultIndexCache.Running{
  contextVaultIndexCache.Running=true
  contextVaultIndexCache.Snapshot.Status="indexing"
  go scanContextVaultIndex(vault)
 }
 return contextVaultIndexCache.Snapshot
}
func (a *App) contextIndex() (any,error){
 a.mu.Lock();vault:=a.cfg.ContextVault;a.mu.Unlock()
 if vault!=""{
  if !filepath.IsAbs(vault)||isBroadOrDangerousRoot(vault){return nil,errors.New("cofre nao permitido")}
  permitted:=false
  vs,err:=obsidianVaults();if err!=nil{return nil,err}
  for _,v:=range vs{if p,ok:=v["path"].(string);ok&&strings.EqualFold(filepath.Clean(p),filepath.Clean(vault)){permitted=true;break}}
  if !permitted{return nil,errors.New("cofre nao encontrado no Obsidian")}
 }
 return contextIndexCacheGet(vault),nil
}
func scanContextVaultIndex(vault string){
 start:=time.Now()
 results:=contextIndexSnapshot{Status:"ready",Source:"Obsidian (metadados .md; nenhum conteudo lido)"}
 folderCounts:=map[string]int{}
 candidates:=[]contextFileIndex{}
 maxEntries:=45000
 maxFiles:=25000
 maxDuration:=8*time.Second
 skip:=map[string]bool{".obsidian":true,".git":true,".trash":true,".cache":true,"node_modules":true,".venv":true,"dist":true,"build":true}
 complete:=true
 err:=filepath.WalkDir(vault,func(path string,d fs.DirEntry,visitErr error)error{
  if visitErr!=nil{return nil}
  if results.Scanned>=maxEntries||results.Total>=maxFiles||time.Since(start)>maxDuration{
   complete=false
   return filepath.SkipAll
  }
  if path==vault{return nil}
  name:=strings.ToLower(d.Name())
  if d.Type()&os.ModeSymlink!=0 {if d.IsDir(){return filepath.SkipDir};return nil}
  if d.IsDir(){
   if skip[name]||strings.HasPrefix(name,".") {return filepath.SkipDir}
   return nil
  }
  results.Scanned++
  if !strings.HasSuffix(name,".md"){return nil}
  rel,er:=filepath.Rel(vault,path);if er!=nil||strings.HasPrefix(rel,".."){return nil}
  folder:=filepath.ToSlash(filepath.Dir(rel))
  if folder=="."{folder="Raiz"}
  results.Total++
  folderCounts[folder]++
  if len(candidates)<240 {
   candidates=append(candidates,contextFileIndex{
    Path:filepath.ToSlash(rel),Folder:folder,Title:strings.TrimSuffix(d.Name(),filepath.Ext(d.Name())),
   })
  }
  return nil
 })
 if err!=nil{results.Status="error"}
 results.Complete=complete&&err==nil
 results.UpdatedAt=time.Now().UTC()
 keys:=make([]string,0,len(folderCounts))
 for k:=range folderCounts{keys=append(keys,k)}
 sort.Slice(keys,func(i,j int)bool{
  if folderCounts[keys[i]]!=folderCounts[keys[j]]{return folderCounts[keys[i]]>folderCounts[keys[j]]}
  return keys[i]<keys[j]
 })
 if len(keys)>72{keys=keys[:72]}
 for _,k:=range keys{
  results.Folders=append(results.Folders,contextFolderIndex{Path:k,Name:filepath.Base(k),Count:folderCounts[k]})
 }
 results.Files=candidates
 contextVaultIndexCache.Lock()
 if contextVaultIndexCache.Vault==vault{
  contextVaultIndexCache.Snapshot=results
  contextVaultIndexCache.Expires=time.Now().Add(90*time.Second)
  contextVaultIndexCache.Running=false
 }
 contextVaultIndexCache.Unlock()
}
