import {computed,onMounted,onUnmounted,reactive,watch,type Ref} from 'vue'
import {discoverLANs, LANDiscoveryError} from '../pages/device/lanDiscovery'
import {buildDeviceListItems} from '../pages/device/deviceInventory'
import {useNetworkStore,useSystemStore} from '../plugins/store'
import type {WidgetId} from './layout/rowLayout'
import type {HomeData} from './widgetContext'

type Channel={id:string;widgets:WidgetId[];path:string;seconds:number;validate:(value:any)=>boolean}
const object=(v:any)=>!!v&&typeof v==='object'&&!Array.isArray(v)
const list=(key:string)=>(v:any)=>object(v)&&Array.isArray(v[key])&&v[key].length<=4096
const channels:Channel[]=[
 {id:'network',widgets:[],path:'/u/network/status/',seconds:5,validate:object},
 {id:'ports',widgets:['networkInterface'],path:'/network/port/list/',seconds:10,validate:list('ports')},
 {id:'traffic',widgets:['traffic'],path:'/u/network/statistics/',seconds:5,validate:list('items')},
 {id:'inventory',widgets:['devices'],path:'/lanctrl/v2/devices/',seconds:30,validate:list('devices')},
 {id:'routerContext',widgets:['devices'],path:'/lanctrl/v2/router-context/',seconds:30,validate:object},
 {id:'disks',widgets:['storage'],path:'/nas/disk/status/',seconds:60,validate:object},
 {id:'raid',widgets:['storage'],path:'/raid/list/',seconds:60,validate:object},
 {id:'sharing',widgets:['sharing'],path:'/u/nas/service/status/',seconds:60,validate:object},
 {id:'docker',widgets:['docker'],path:'/guide/docker/status/',seconds:60,validate:object},
 {id:'download',widgets:['downloads'],path:'/guide/download-service/status/',seconds:60,validate:object},
 {id:'remote',widgets:['remote'],path:'/u/guide/ddns/',seconds:3,validate:object},
 {id:'system',widgets:['system'],path:'/system/status/',seconds:5,validate:object},
 {id:'version',widgets:['system'],path:'/u/system/version/',seconds:0,validate:object},
]
// One owner per mounted homepage. Reordering does not recreate readers. Every
// request has a deadline; hide/unmount cancels requests and discards late data.
export function useHomeData(visible:Ref<WidgetId[]>){
 const data=reactive<HomeData>({}),errors=reactive<Record<string,string>>({}),updated=reactive<Record<string,number>>({})
 const pending=new Map<string,AbortController>(),timers=new Map<string,number>()
 const network=useNetworkStore(),system=useSystemStore();let active=false
 let discovery:ReturnType<typeof discoverLANs>|undefined
 let discoveredAt=0
 const wanted=(c:Channel)=>!c.widgets.length||c.widgets.some(id=>visible.value.includes(id))
 function clear(id:string){window.clearTimeout(timers.get(id));timers.delete(id);pending.get(id)?.abort();pending.delete(id)}
 async function load(c:Channel){
  if(!active||document.hidden||!wanted(c)||pending.has(c.id))return
  window.clearTimeout(timers.get(c.id));timers.delete(c.id)
  const controller=new AbortController();pending.set(c.id,controller);const deadline=window.setTimeout(()=>controller.abort(),8000)
  try{
   let path=c.path
   if(c.id==='inventory'||c.id==='routerContext'){
    if(Date.now()-discoveredAt>=30000){discovery=undefined;discoveredAt=Date.now()}
    const found=await (discovery ||= discoverLANs().catch(reason=>{discovery=undefined;throw reason}))
    data.lanNetworks=found.networks;data.lanLegacy=found.legacy
    const selected=found.networks.length===1?found.networks[0].networkId:'all'
    if(c.id==='routerContext'&&selected==='all'){if(active){data.routerContext={multiLAN:true};errors[c.id]='';updated[c.id]=Date.now()}return}
    if(!found.legacy)path+='?network='+encodeURIComponent(selected)
   }
   const response=await fetch('/cgi-bin/luci/istore'+path,{credentials:'same-origin',cache:'no-store',signal:controller.signal})
   if(!response.ok)throw new Error(response.status===401||response.status===403?'permission':'unavailable')
   const text=await response.text();if(text.length>2*1024*1024)throw new Error('invalid')
   const envelope=JSON.parse(text);if(envelope.error||envelope.success!==undefined&&envelope.success!==0||!c.validate(envelope.result))throw new Error(envelope.success===-1001?'permission':'invalid')
   if(!active||pending.get(c.id)!==controller||!wanted(c))return
   data[c.id]=c.id==='traffic'?{...envelope.result,items:envelope.result.items.slice(-60)}:envelope.result
   errors[c.id]='';updated[c.id]=Date.now()
   if(c.id==='network')network.updateNetworkStatus({...data.network!})
   if(c.id==='system')system.updateSystemStatus({...data.system!})
   if(c.id==='version')system.version={...data.version!}
  }catch(error){if(active&&pending.get(c.id)===controller)errors[c.id]=error instanceof LANDiscoveryError&&error.code==='permission'||error instanceof Error&&error.message==='permission'?'permission':'unavailable'}
  finally{window.clearTimeout(deadline);if(pending.get(c.id)===controller){pending.delete(c.id);if(active&&!document.hidden&&wanted(c)&&c.seconds)timers.set(c.id,window.setTimeout(()=>void load(c),c.seconds*1000))}}
 }
 function refresh(force=false){for(const c of channels){if(!wanted(c))clear(c.id);else if(force||!timers.has(c.id)&&!pending.has(c.id)&&(c.seconds>0||!updated[c.id]&&!errors[c.id]))void load(c)}}
 function visibility(){for(const c of channels)clear(c.id);if(!document.hidden)refresh(true)}
 function invalidate(id?:WidgetId){for(const c of channels)if(!id||c.widgets.includes(id))void load(c)}
 // A return anchor needs only the first outcome of the widgets above/in its
 // row. This does not gate rendering or wait for unrelated cards below it.
 function initialSettled(widgets:readonly string[]){return channels.every(c=>{
  const consumers=c.id==='network'?['traffic','connection','devices']:c.widgets
  return !consumers.some(id=>widgets.includes(id))||data[c.id]!==undefined||!!errors[c.id]
 })}
 watch(()=>[...visible.value].sort().join(','),()=>refresh())
 onMounted(()=>{active=true;refresh();document.addEventListener('visibilitychange',visibility)})
 onUnmounted(()=>{active=false;for(const c of channels)clear(c.id);document.removeEventListener('visibilitychange',visibility)})
 return {data,errors,updated,devices:computed(()=>buildDeviceListItems(data.inventory?.devices||[])),invalidate,initialSettled}
}
