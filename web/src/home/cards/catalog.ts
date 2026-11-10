import type {CardManifest} from './contract'
// Available metadata ships with Quickstart. No executable URLs or iStore changes.
export const installableCards=[{id:'plugin.fastnet',zh:'外网测速',en:'Internet speed',glyph:'activity',description:'检测外网速率、NAT 和 IPv6',package:'app-meta-fastnet',minWidth:280,minHeight:260,singleHeight:280}] as const
const registered=new Map<string,CardManifest>()
export function registeredCard(id:string){return registered.get(id)}
export function setRegisteredCards(cards:CardManifest[]){registered.clear();for(const c of cards)registered.set(c.id,c)}
export function externalWidget(id:`plugin.${string}`){
 const known=installableCards.find(c=>c.id===id),m=registered.get(id),size=m?.forms.standard||m?.forms.summary
 return {id,zh:m?.title||known?.zh||'插件模块',en:known?.en||m?.title||'Plugin module',glyph:m?.glyph||known?.glyph||'grid',minWidth:size?.minWidth||known?.minWidth||240,minHeight:size?.minHeight||known?.minHeight||260,singleHeight:size?.minHeight||known?.singleHeight||280,extra:true}
}

const registrationIssues=new Map<string,string>()
export function setCardIssues(issues:unknown){registrationIssues.clear();if(!Array.isArray(issues))return;for(const item of issues.slice(0,32))if(item&&/^plugin\.[a-z][a-z0-9-]{0,47}$/.test(item.id)&&['unsupported_abi','unsupported_protocol'].includes(item.code))registrationIssues.set(item.id,item.code)}
export function needsHostUpdate(id:string){return registrationIssues.has(id)}
