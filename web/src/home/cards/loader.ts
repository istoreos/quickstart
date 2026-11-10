import type {Component} from 'vue'
import {cardManifest,type CardManifest} from './contract'
export type CardModule={apiVersion:1;default:Component}
export type CardImporter=(url:string)=>Promise<CardModule>
const remote:CardImporter=url=>import(/* @vite-ignore */ url)
// ESM stays in the browser module cache; this cache also merges concurrent callers.
export function createCardLoader(importer:CardImporter=remote,deadline=15000){
 const pending=new Map<string,Promise<Component>>()
 const failures=new Map<string,number>()
 return function load(input:CardManifest):Promise<Component>{
  const manifest=cardManifest(input),key=manifest.entry
  const cached=pending.get(key);if(cached)return cached
  const attempt=failures.get(key)||0
  if(attempt>3)return Promise.reject(new Error('card_reload_required'))
  let timer:ReturnType<typeof setTimeout>
  const timeout=new Promise<never>((_,reject)=>{timer=setTimeout(()=>reject(new Error('card_load_timeout')),deadline)})
  // Browsers cache failed ESM evaluations too. A bounded, deterministic retry
  // URL permits explicit recovery; successful versions retain their Promise.
  const url=attempt?key+'?qs-card-retry='+attempt:key
  const result=Promise.race([Promise.resolve().then(()=>importer(url)),timeout]).then(module=>{
   const c:any=module.default
   if(module.apiVersion!==1||!c||!(typeof c==='function'||typeof c==='object'&&(typeof c.setup==='function'||typeof c.render==='function')))throw new Error('card_export_incompatible')
   return module.default
  }).catch(error=>{pending.delete(key);failures.set(key,attempt+1);throw error}).finally(()=>clearTimeout(timer))
  pending.set(key,result);return result
 }
}
export const loadCard=createCardLoader()
