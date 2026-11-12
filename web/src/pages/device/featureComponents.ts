import type {CapabilityState} from './deviceCapabilities'

export const componentDefinitions = [
 {id:'quickstart-native',key:'device_speed_limit',pkg:'quickstart-netpolicy',provider:'quickstart-native',zh:'智能设备限速',en:'Smart device limits',descriptionZh:'按设备身份管理 IPv4 与 IPv6 限速；是否就绪以组件检查为准。',descriptionEn:'Device-based IPv4 and IPv6 limits; readiness is checked separately.'},
 {id:'eqos',key:'compat_speed_limit',pkg:'luci-app-eqos',provider:'eqos',zh:'兼容限速',en:'Compatible limits',descriptionZh:'按固定 IPv4 执行，适合使用现有兼容限速设置。',descriptionEn:'Limits by stable IPv4 address using the existing compatible settings.'},
 {id:'floatip',key:'floating_gateway',pkg:'app-meta-floatip',provider:'',zh:'浮动网关',en:'Failover gateway',descriptionZh:'提供主备网关入口；节点职责、虚拟地址与实际持有者在设置中确认。',descriptionEn:'A shared gateway entry; configure node roles and verify the virtual IP holder in settings.'},
] as const
export type ComponentId=typeof componentDefinitions[number]['id']
export type FeatureComponent={id:ComponentId;capabilityKey:string;package:string;provider?:string;selected:boolean;status:{state:CapabilityState;reason?:string;actions?:{kind:string}[]};operation?:'running'|'pending'}
export type ComponentCatalog={schemaVersion:1;currentProvider:string;components:FeatureComponent[]}
export type ComponentPlan={capabilityKey:string;action:'install'|'enable';current:{state:CapabilityState};target:string;canApply:boolean;requiresConfirmation:boolean;requiredFreeBytes?:number;availableFreeBytes?:number;draftToken?:string;version?:string;networkId?:string;operationScope?:string;affectedNetworks?:string[];error?:{code:string;message?:string}}
export class ComponentError extends Error{constructor(public code:string){super(code)}}
const object=(v:any)=>v&&typeof v==='object'&&!Array.isArray(v)
const states=['available','disabled','not_installed','unsupported','error']
export function parseComponentCatalog(body:any):ComponentCatalog{
 if(!object(body)||body.schemaVersion!==1||typeof body.currentProvider!=='string'||!Array.isArray(body.components)||body.components.length!==3)throw new ComponentError('invalid')
 for(const definition of componentDefinitions){
  const items=body.components.filter((item:any)=>item?.id===definition.id),item=items[0]
  if(items.length!==1||item.capabilityKey!==definition.key||item.package!==definition.pkg||item.provider!==definition.provider&&!(definition.provider===''&&item.provider===undefined)||typeof item.selected!=='boolean'||!object(item.status)||!states.includes(item.status.state)||item.operation!==undefined&&!['running','pending'].includes(item.operation))throw new ComponentError('invalid')
  if(item.status.actions!==undefined&&(!Array.isArray(item.status.actions)||item.status.actions.some((a:any)=>!object(a)||!['install','enable','retry'].includes(a.kind))))throw new ComponentError('invalid')
 }
 return body
}
export function componentAction(item:FeatureComponent):'install'|'enable'|undefined{
 if(item.operation)return
 const kind=item.status.state==='not_installed'?'install':item.id==='quickstart-native'&&item.status.state==='disabled'?'enable':undefined
 return kind&&item.status.actions?.some(action=>action.kind===kind)?kind:undefined
}
export function createComponentClient(transport:typeof fetch=fetch, networkId?:string){
 async function exchange(path:string,body?:unknown,write=false,signal?:AbortSignal){
  const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),write?45000:10000)
  const abort=()=>controller.abort();if(signal?.aborted)abort();signal?.addEventListener('abort',abort,{once:true})
  try{
   const response=await transport('/cgi-bin/luci/istore/lanctrl/v2/'+path+(networkId?'?network='+encodeURIComponent(networkId):''),{method:body?'POST':'GET',credentials:'same-origin',cache:'no-store',signal:controller.signal,headers:body?{'Content-Type':'application/json'}:undefined,body:body?JSON.stringify(body):undefined})
   if(!response.ok)throw new ComponentError(response.status===401||response.status===403?'forbidden':response.status===404?'unsupported':'unavailable')
   const text=await response.text();if(text.length>65536)throw new ComponentError('invalid')
   let envelope:any;try{envelope=JSON.parse(text)}catch{throw new ComponentError('invalid')}
   if(!object(envelope)||envelope.error||envelope.success!==undefined&&envelope.success!==0)throw new ComponentError(envelope?.success===-1001?'forbidden':'unavailable')
   if(!object(envelope.result))throw new ComponentError('invalid')
   return envelope.result
  }catch(error){if(write)throw new ComponentError(error instanceof ComponentError&&error.code==='forbidden'?'forbidden':'uncertain');if(error instanceof ComponentError)throw error;throw new ComponentError('unavailable')}
  finally{clearTimeout(timer);signal?.removeEventListener('abort',abort)}
 }
 return {
  async read(signal?:AbortSignal){return parseComponentCatalog(await exchange('feature-components/',undefined,false,signal))},
  async plan(item:FeatureComponent,action:'install'|'enable',signal?:AbortSignal){
   const result=await exchange('capability-action/plan/',{capabilityKey:item.capabilityKey,action,draftToken:'components:'+item.id},false,signal)
   const plan=result.plan
   if(!object(plan)||plan.capabilityKey!==item.capabilityKey||plan.action!==action||typeof plan.canApply!=='boolean'||plan.requiresConfirmation!==true)throw new ComponentError('invalid')
   if(plan.error)throw new ComponentError(plan.error.code||'unavailable')
   if(!object(plan.current)||plan.current.state!==(action==='install'?'not_installed':'disabled')||plan.target!==(action==='install'?item.package:item.capabilityKey)||!plan.canApply||action==='install'&&(!Number.isFinite(plan.requiredFreeBytes)||plan.requiredFreeBytes<0||!Number.isFinite(plan.availableFreeBytes)||plan.availableFreeBytes<plan.requiredFreeBytes))throw new ComponentError('invalid')
   return plan as ComponentPlan
  },
  async apply(plan:ComponentPlan){
   const result=await exchange('capability-action/apply/',{capabilityKey:plan.capabilityKey,action:plan.action,draftToken:plan.draftToken,expectedState:plan.current.state,expectedVersion:plan.version,confirm:true},true)
   if(result.error)throw new ComponentError(result.error.code||'uncertain')
   if(typeof result.changed!=='boolean'||result.changed!==true||!object(result.current)||!states.includes(result.current.state))throw new ComponentError('uncertain')
   return result
  },
 }
}
