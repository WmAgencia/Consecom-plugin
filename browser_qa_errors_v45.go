package main

import (
 "errors"
)

// The recorder keeps only categories (not console arguments, user input,
// stack traces, full URLs or access tokens). It is installed in the Consecom
// target only, including after that tab's own navigations.
const browserQARecorderScript = `(()=>{
 if(window.__consecomQARecorderV1)return {active:true,already_installed:true};
 const entries=[];
 const add=(kind)=>{if(entries.length>=50)entries.shift();entries.push({kind:kind,time:Date.now()})};
 Object.defineProperty(window,'__consecomQARecorderV1',{configurable:false,enumerable:false,value:{entries}});
 window.addEventListener('error',()=>add('JavaScript error'));
 window.addEventListener('unhandledrejection',()=>add('Unhandled promise rejection'));
 const original=console.error;
 console.error=function(...args){add('console.error');return original.apply(this,args)};
 return {active:true,already_installed:false,note:'Captura apenas categorias; nao grava mensagem, argumentos, payloads nem cookies'};
})()`

const browserQARecorderReadScript = `(()=>{
 const recorder=window.__consecomQARecorderV1;
 if(!recorder)return {active:false,note:'Execute browser_qa_instrumentar antes de testar interacoes.'};
 const count={};
 for(const entry of recorder.entries){count[entry.kind]=(count[entry.kind]||0)+1}
 return {active:true,errors_total:recorder.entries.length,categories:count,
 note:'Apenas contagens de categorias coletadas na aba autorizada. Console e payloads nao sao exportados.'};
})()`

func browserQARecorder(s *BrowserSession,action string)(any,error){
 switch action{
 case "browser_qa_instrumentar":
  if _,err:=cdpCall(s,"Page.enable",map[string]any{});err!=nil{return nil,err}
  if _,err:=cdpCall(s,"Page.addScriptToEvaluateOnNewDocument",map[string]any{"source":browserQARecorderScript});err!=nil{return nil,err}
  return cdpEvaluate(s,browserQARecorderScript)
 case "browser_qa_erros":
  return cdpEvaluate(s,browserQARecorderReadScript)
 default:return nil,errors.New("acao de instrumentacao desconhecida")
 }
}
