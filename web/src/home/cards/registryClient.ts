import {cardManifests,type CardManifest} from './contract'
import {setRegisteredCards,setCardIssues} from './catalog'
export function createCardRegistry(transport:typeof fetch=fetch){
 let pending:Promise<CardManifest[]>|undefined,cached:CardManifest[]|undefined
 return {read(force=false):Promise<CardManifest[]>{
  if(pending)return pending
  if(cached&&!force)return Promise.resolve(cached)
  const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),8000)
  pending=(async()=>{
   const res=await transport('/cgi-bin/luci/istore/home/v1/cards/',{credentials:'same-origin',signal:controller.signal,cache:'no-store'})
   if(!res.ok||!res.headers.get('Content-Type')?.includes('application/json'))throw Error('card_discovery_unavailable')
   const text=await res.text();if(new TextEncoder().encode(text).length>65536)throw Error('card_registry_too_large')
   const value=JSON.parse(text);if(value.schemaVersion!==1)throw Error('card_registry_incompatible')
   cached=cardManifests(value.cards);setRegisteredCards(cached);setCardIssues(value.issues);return cached
  })().finally(()=>{clearTimeout(timer);pending=undefined})
  return pending
 }}
}
export const cardRegistry=createCardRegistry()
