package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "sort"
 "strings"
)

// The Consecom desktop MCP and the cloud connectors installed in ChatGPT are
// separate trust domains. Tokens are never copied from ChatGPT into Windows.
// These diagnostic tools make local development readiness visible without
// acquiring credentials, reading secret files or invoking provider commands.
type LocalDevCLI struct {
 Provider string `json:"provider"`
 Executable string `json:"executable"`
 Installed bool `json:"installed"`
}

func localDevStackStatus(a *App) (any,error) {
 specs:=[]struct{provider,command string}{
  {"Git", "git"},
  {"GitHub", "gh"},
  {"Node.js", "node"},
  {"npm", "npm"},
  {"Vercel", "vercel"},
  {"Supabase", "supabase"},
  {"Railway", "railway"},
  {"Go", "go"},
 }
 list:=make([]LocalDevCLI,0,len(specs))
 for _,spec:=range specs {
  _,err:=exec.LookPath(spec.command)
  list=append(list,LocalDevCLI{Provider:spec.provider,Executable:spec.command,Installed:err==nil})
 }
 roots:=a.roots()
 return map[string]any{
  "local_clis":list,
  "authorized_project_count":len(roots),
  "cloud_connectors":[]string{"GitHub","Vercel","Supabase","Railway"},
  "cloud_connection_note":"As conexoes cloud do ChatGPT sao gerenciadas nos respectivos conectores e nao compartilham tokens com o Consecom; disponibilidade local de CLI nao prova login.",
  "usage":"Use os conectores cloud no ChatGPT para GitHub/Vercel/Supabase/Railway. Use Consecom para editar e testar pastas de projeto cadastradas no painel.",
  "secrets":"Nenhuma credencial ou conteudo de .env foi lido.",
 },nil
}

type webProjectManifest struct {
 Name string `json:"name"`
 Scripts map[string]json.RawMessage `json:"scripts"`
 Dependencies map[string]json.RawMessage `json:"dependencies"`
 DevDependencies map[string]json.RawMessage `json:"devDependencies"`
}

func projectFileExists(root,relative string)bool{
 path:=filepath.Join(root,relative)
 if err:=ensureNoLinks(path);err!=nil{return false}
 st,err:=os.Lstat(path)
 return err==nil&&(st.Mode().IsRegular()||st.IsDir())
}

func webFramework(deps map[string]bool,files map[string]bool)string{
 switch {
 case deps["next"] || files["next.config.js"] || files["next.config.mjs"] || files["next.config.ts"]:return "Next.js"
 case deps["astro"]:return "Astro"
 case deps["nuxt"]:return "Nuxt"
 case deps["@sveltejs/kit"]:return "SvelteKit"
 case deps["@remix-run/react"]:return "Remix"
 case deps["vite"]:return "Vite"
 case deps["react"]:return "React"
 case deps["vue"]:return "Vue"
 case files["index.html"]:return "HTML estatico"
 default:return "Nao identificado"
 }
}

func (a *App) devProjectInspect(rootID string)(any,error){
 root,err:=a.getRoot(rootID);if err!=nil{return nil,err}
 if isBroadOrDangerousRoot(root.Path){return nil,errors.New("pasta de projeto muito abrangente ou protegida")}
 if err=ensureNoLinks(root.Path);err!=nil{return nil,err}
 st,err:=os.Stat(root.Path);if err!=nil||!st.IsDir(){return nil,errors.New("pasta de projeto nao esta acessivel")}
 paths:=[]string{
  "package.json","package-lock.json","pnpm-lock.yaml","yarn.lock","bun.lockb",
  "index.html","src","app","pages","public","next.config.js","next.config.mjs","next.config.ts",
  "vercel.json","railway.json","railway.toml","Dockerfile","supabase/config.toml",
  ".git","node_modules",
 }
 files:=map[string]bool{}
 for _,path:=range paths {files[path]=projectFileExists(root.Path,path)}
 info:=webProjectManifest{}
 if files["package.json"]{
  target,err:=safePath(root,"package.json",false);if err!=nil{return nil,err}
  stat,err:=os.Stat(target);if err!=nil{return nil,err}
  if !stat.Mode().IsRegular()||stat.Size()>262144{return nil,errors.New("package.json irregular ou acima de 256 KiB")}
  raw,err:=os.ReadFile(target);if err!=nil{return nil,err}
  if err=json.Unmarshal(raw,&info);err!=nil{return nil,fmt.Errorf("package.json invalido: %w",err)}
 }
 scriptNames:=make([]string,0,len(info.Scripts))
 for s:=range info.Scripts {
  if len(s)>0&&len(s)<100{scriptNames=append(scriptNames,s)}
 }
 sort.Strings(scriptNames)
 if len(scriptNames)>60{scriptNames=scriptNames[:60]}
 deps:=map[string]bool{}
 for d:=range info.Dependencies{deps[d]=true}
 for d:=range info.DevDependencies{deps[d]=true}
 safeName:=strings.TrimSpace(info.Name)
 if len(safeName)>100{safeName=safeName[:100]}
 return map[string]any{
  "root_id":root.ID,
  "root_name":root.Name,
  "writable":root.Writable,
  "package_name":safeName,
  "framework":webFramework(deps,files),
  "scripts":scriptNames,
  "detected_files":files,
  "is_git_repository":files[".git"],
  "vercel_configured":files["vercel.json"],
  "supabase_project_configured":files["supabase/config.toml"],
  "railway_configured":files["railway.json"]||files["railway.toml"]||files["Dockerfile"],
  "next_actions":[]string{"Teste o projeto local com dev_job_iniciar ou fluxo_iniciar.","Use o conector GitHub para versionar e Vercel/Supabase/Railway para infraestrutura remota.","Para ler/escrever este projeto, a pasta deve estar cadastrada e com escrita habilitada no painel."},
  "privacy_note":"O Consecom nao leu arquivos .env, credenciais, .git/config, tokens ou conteudos dos scripts do package.json.",
 },nil
}
