package main

import (
 "encoding/json"
 "errors"
 "fmt"
 "strings"
)

// Browser QA uses the Consecom-owned CDP target, never a user's unrelated
// tabs. It returns bounded structural findings, not inputs or cookies.
const browserQAAuditScript = `(()=>{
 const text=x=>String(x||'').trim().slice(0,110);
 const main=Array.from(document.querySelectorAll('h1')).slice(0,5).map(x=>text(x.textContent));
 const imgs=Array.from(document.images);
 const buttons=Array.from(document.querySelectorAll('button,[role="button"]'));
 const fields=Array.from(document.querySelectorAll('input:not([type="hidden"]),textarea,select'));
 const label=x=>Boolean(x.labels&&x.labels.length)||Boolean(x.getAttribute('aria-label'))||Boolean(x.getAttribute('aria-labelledby'))||Boolean(x.getAttribute('title'));
 const failures=imgs.filter(x=>x.complete&&x.naturalWidth===0).length;
 const unlabeledImages=imgs.filter(x=>!x.hasAttribute('alt')).length;
 const unlabeledButtons=buttons.filter(x=>!text(x.textContent)&&!x.getAttribute('aria-label')&&!x.getAttribute('title')).length;
 const unlabeledInputs=fields.filter(x=>!label(x)).length;
 const duplicates=new Set(),seen=new Set();
 for(const x of document.querySelectorAll('[id]')){if(seen.has(x.id))duplicates.add(x.id.slice(0,50));seen.add(x.id)}
 const q=(sel)=>document.querySelector(sel);
 const desc=q('meta[name="description"]');
 const vp=q('meta[name="viewport"]');
 const canon=q('link[rel="canonical"]');
 const sc=document.documentElement.scrollWidth, vw=window.innerWidth;
 const issues=[];
 if(!document.title.trim())issues.push('Titulo da pagina vazio');
 if(!desc||!desc.content.trim())issues.push('Meta description ausente');
 if(!vp)issues.push('Meta viewport ausente');
 if(!main.length)issues.push('H1 ausente');
 if(main.length>1)issues.push('Mais de um H1');
 if(failures)issues.push('Imagens quebradas: '+failures);
 if(unlabeledImages)issues.push('Imagens sem atributo alt: '+unlabeledImages);
 if(unlabeledButtons)issues.push('Botoes sem nome acessivel: '+unlabeledButtons);
 if(unlabeledInputs)issues.push('Campos sem rotulo: '+unlabeledInputs);
 if(duplicates.size)issues.push('IDs HTML duplicados: '+duplicates.size);
 if(sc>vw+3)issues.push('Overflow horizontal: '+(sc-vw)+' px');
 return {title:text(document.title),protocol:location.protocol,
  viewport:{width:vw,height:window.innerHeight,document_width:sc,horizontal_overflow:Math.max(0,sc-vw)},
  seo:{title_present:Boolean(document.title.trim()),description_present:Boolean(desc&&desc.content.trim()),viewport_present:Boolean(vp),canonical_present:Boolean(canon),h1_count:main.length},
  accessibility:{images:imgs.length,images_missing_alt:unlabeledImages,buttons_missing_label:unlabeledButtons,fields_missing_label:unlabeledInputs,duplicate_id_count:duplicates.size},
  broken_images:failures,issues:issues.slice(0,22),
  note:'Diagnostico estrutural DOM; nao substitui auditoria completa de acessibilidade, SEO ou Lighthouse.'};
})()`

var qaViewportSizes = map[string][2]int{"mobile":[2]int{390,844},"tablet":[2]int{768,1024},"desktop":[2]int{1365,768}}

func qaSafeViewport(v string)([2]int,error){
 if dimensions,ok:=qaViewportSizes[v];ok{return dimensions,nil}
 return [2]int{},errors.New("viewport permitido: mobile, tablet ou desktop")
}

func (a *App) browserQATool(name,viewport string)(any,error){
 a.browserControl.mu.Lock()
 defer a.browserControl.mu.Unlock()
 s:=a.browserControl.session
 if s==nil{return nil,errors.New("abra primeiro uma aba controlada pelo Consecom")}
 switch name{
 case "browser_qa_auditar":
  return cdpEvaluate(s,browserQAAuditScript)
 case "browser_qa_responsivo":
  dim,err:=qaSafeViewport(viewport);if err!=nil{return nil,err}
  // The viewport change only targets the currently owned tab. Always clear
  // the override, including when screenshot capture or JS inspection fails.
  if _,err:=cdpCall(s,"Emulation.setDeviceMetricsOverride",map[string]any{
   "width":dim[0],"height":dim[1],"deviceScaleFactor":1,"mobile":viewport=="mobile",
  });err!=nil{return nil,fmt.Errorf("nao foi possivel definir viewport: %w",err)}
  defer func(){_,_=cdpCall(s,"Emulation.clearDeviceMetricsOverride",map[string]any{})}()
  result,err:=cdpEvaluate(s,browserQAAuditScript)
  if err!=nil{return nil,err}
  screenshot,err:=browserScreenshotLocked(s)
  if err!=nil{return nil,err}
  return map[string]any{"viewport":viewport,"dimensions":dim,"audit":result,"screenshot":screenshot,
   "note":"Captura e auditoria na aba autorizada; dimensoes originais restauradas em seguida."},nil
 default:
  return nil,fmt.Errorf("ferramenta QA desconhecida: %s",strings.TrimSpace(name))
 }
}
