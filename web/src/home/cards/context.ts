import {reactive,readonly,shallowRef,type Component} from 'vue'
import {getTheme,getThemePreference,subscribeTheme,type ThemeSnapshot} from '../../utils/theme'
import type {CardForm} from './contract'
export class CardRequestError extends Error{constructor(public status:number,message='card_request_failed'){super(message)}}
export function createCardContext(apiBase:string,locale:string,form:CardForm){
 const theme=reactive<ThemeSnapshot>({preference:getThemePreference(),effective:getTheme()}),display=reactive({form}),details=shallowRef<{component:Component;props:Record<string,unknown>;wide?:boolean;onClose?:()=>void}|null>(null)
 const unsubscribe=subscribeTheme(snapshot=>Object.assign(theme,snapshot)),requests=new Set<AbortController>()
 let disposed=false
 async function request(path:string,options:RequestInit={}){
  if(disposed)throw new Error('card_unmounted')
  if(!/^\/[A-Za-z0-9_-]+(?:\/[A-Za-z0-9_-]+)*\/?(?:\?[^#\x00-\x1f\x7f]*)?$/.test(path))throw new Error('invalid_card_path')
  const controller=new AbortController();requests.add(controller)
  const stop=()=>controller.abort();options.signal?.addEventListener('abort',stop,{once:true});if(options.signal?.aborted)stop()
  const timer=setTimeout(stop,8000)
  try{
   const headers=new Headers(options.headers);if(options.body)headers.set('Content-Type','application/json')
   // LuCI validates its session token. It is never forwarded to plugin code.
   const token=(window as any).L?.env?.token||(window as any).luci_token||(window as any).token
   let target=apiBase+path
   if(token&&options.method==='POST'){
    headers.set('X-CSRF-Token',String(token))
    const separator=target.indexOf('?'),pathname=separator<0?target:target.slice(0,separator),query=new URLSearchParams(separator<0?'':target.slice(separator+1))
    query.set('token',String(token));target=pathname+'?'+query.toString()
   }
   const response=await fetch(target,{...options,headers,signal:controller.signal,credentials:'same-origin',cache:'no-store',redirect:'manual'})
   if(response.type==='opaqueredirect')throw new CardRequestError(401)
   if(!response.ok)throw new CardRequestError(response.status)
   if(!response.headers.get('Content-Type')?.includes('application/json'))throw new Error('invalid_card_response')
   const raw=await response.text();if(new TextEncoder().encode(raw).length>8192)throw new Error('card_response_too_large')
   return JSON.parse(raw)
  }finally{clearTimeout(timer);requests.delete(controller);options.signal?.removeEventListener('abort',stop)}
 }
 function closeDetails(){const closing=details.value;details.value=null;closing?.onClose?.()}
 return {context:{theme:readonly(theme),get form(){return display.form},apiBase,locale,request,openDetails(component:Component,props:Record<string,unknown>={},options:{wide?:boolean;onClose?:()=>void}={}){if(!disposed)details.value={component,props,...options}}},details,closeDetails,setForm(form:CardForm){display.form=form},dispose(){disposed=true;unsubscribe();for(const r of requests)r.abort();requests.clear();details.value=null}}
}
