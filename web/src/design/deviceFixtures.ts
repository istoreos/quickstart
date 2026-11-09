import { demo } from './fixtures'
import { inventory, profiles } from './devices'
import { validDeviceAlias } from '../pages/device/deviceProfile'
import { validDhcpHostname } from '../pages/device/devicePolicy'

const policies=new Map<string, any>()
let revision=1
export const speedCapability=()=>({'no-speed':'not_installed',disabled:'disabled','speed-error':'error',unsupported:'unsupported'}[demo.scenario]||'available')
export const routerContext=()=>({lan:'lan',topologyPosition:'lan_gateway_candidate',localAddress:'192.168.30.244',localDhcpConfigured:demo.authority==='local',localDhcpHealthy:true,dhcpAuthority:demo.authority==='local'?'local':demo.authority==='external'?'external_observed':'ambiguous',routeEditability:{editable:demo.authority==='local',reason:demo.authority==='local'?'local_dhcp_authority':demo.authority==='external'?'external_dhcp_authority':'multiple_dhcp_evidence'},upstreamGateways:['192.168.30.1'],externalDhcpServers:[],evidence:[]})
const targets=[{id:'default',name:'跟随网络默认',kind:'default',gateway:'192.168.30.1',supported:true},{id:'self',name:'本机路由',kind:'self',gateway:'192.168.30.244',supported:true},{id:'bypass',name:'旁路由',kind:'bypass',gateway:'192.168.30.244',supported:true},{id:'floating',name:'浮动网关',kind:'floating',gateway:'192.168.30.3',supported:false,reasons:['floating_gateway_disabled']}]
function policy(id:string){const d=inventory().find(item=>item.deviceId===id);return {deviceId:id,mac:d?.mac||'',currentIPv4:d?.addresses.current[0]?.address,static:{enabled:false,assignedIP:'',bindIP:false,hostname:'',tagName:''},speed:{enabled:false,uploadSpeed:20,downloadSpeed:100},access:{networkAccess:!id.endsWith(':06')},targetId:'default',...policies.get(id),capabilities:{static:{state:'available'},speed:{state:speedCapability()},access:{state:'available'}},rateLimit:{provider:'quickstart-native',executionNode:'local',state:'inactive',configured:false,loaded:false,verified:false,canApply:speedCapability()==='available',addressState:'stable_identity',ipv6State:'covered',offloadState:'unknown'},version:`demo-${revision}`}}
function network(id:string){const p=policy(id);return {deviceId:id,static:p.static,path:{targetId:p.targetId,effect:{desired:{targetId:p.targetId,static:p.static},applied:{targetId:p.targetId,state:'server_applied',configVersion:`demo-${revision}`},observed:{state:'pending_renewal'},needsAttention:p.targetId!=='default'}},targets,version:`demo-${revision}`}}
function profile(id:string){const d=inventory().find(item=>item.deviceId===id);if(!d)return {error:{code:'not_found',message:'Device not found in preview'}};return {profile:{deviceId:id,alias:d.displayName,originalHostname:d.hostname,scope:'persistent',classification:d.classification,manualBrand:d.classification?.brand,manualCategory:d.classification?.category,icon:d.icon},changed:false}}
const groups={version:'demo-groups-1',timezone:'UTC',groups:[],devicePolicies:{},global:{},effective:[],nextEvaluationAt:'2026-10-14T04:01:00Z'}
export function deviceFixture(key:string,method:string,url:URL,body:any):{status:number;data:unknown}|undefined{
 const id=String(body.deviceId||url.searchParams.get('deviceId')||'')
 const ok=(result:unknown)=>({status:200,data:{result,previewOnly:true}})
 const bad=(code:string,message:string)=>ok({error:{code,message}})
 if(method==='GET'){
   if(key==='/lanctrl/v2/router-context/')return ok(routerContext())
   if(key==='/lanctrl/v2/device-policy/')return ok({policy:policy(id)})
   if(key==='/lanctrl/v2/device-network-policy/')return ok({policy:network(id)})
   if(key==='/lanctrl/v2/device-profile/')return ok(profile(id))
   if(key==='/lanctrl/v2/traffic-insights/')return ok({uploadBytes:1000000,downloadBytes:10000000,buckets:[],bandix:{state:'not_installed'},legacyAccessReviewRequired:new URLSearchParams(location.search).get('legacy-quota')==='1'&&url.searchParams.get('network')==='lan1'})
   if(key==='/lanctrl/v2/device-groups/')return ok(groups)
   if(key==='/lanctrl/v2/advanced-network/')return ok({capabilities:{internet_access:{state:'available'},device_speed_limit:{state:speedCapability()},floating_gateway:{state:'disabled'},traffic_insights:{state:'available'}},events:[],webhook:{enabled:false,configured:false}})
   if(key==='/lanctrl/v2/gateway-targets/')return ok({targets,version:'demo-routes-1'})
   if(key==='/lanctrl/v2/lan-dhcp-settings/')return ok({settings:{enabled:demo.authority==='local',start:100,limit:100,leaseTime:'12h',defaultTargetId:'default'},pool:{start:'192.168.30.100',end:'192.168.30.199'},version:'demo-dhcp-1'})
   if(key==='/lanctrl/v2/migration/plan/')return ok({plan:{needsMigration:false,canApply:false,items:[],conflicts:[],version:'demo-migration-1'}})
   if(key==='/lanctrl/globalConfigs/')return ok({capabilities:{items:{internet_access:{state:'available'},device_speed_limit:{state:speedCapability()},floating_gateway:{state:'disabled'},traffic_insights:{state:'available'}}},network:{speedProvider:'quickstart-native'},speedLimit:{enabled:true,provider:'quickstart-native'},gateway:{}})
 }
 if(method==='POST'){
   if(key==='/lanctrl/v2/device-profile/'){
     if(!validDeviceAlias(body.alias||''))return bad('validation_failed','Invalid alias')
     if(body.action==='reset')profiles.set(id,{alias:profiles.get(id)?.alias,iconMode:'auto'})
     else profiles.set(id,{...profiles.get(id),...body})
     return ok({...profile(id),changed:true,transaction:{task:'profile',stage:'verify',status:'committed'}})
   }
   if(['/lanctrl/v2/device-network-policy/plan/','/lanctrl/v2/device-restrictions/plan/'].includes(key)){
     const net=key.includes('network-policy')
     if(net&&demo.authority!=='local')return bad('dhcp_authority_unavailable','Please configure on the address allocator')
     if(net&&!validDhcpHostname(body.static?.hostname||''))return bad('validation_failed','DHCP hostname must be ASCII')
     if(body.kind==='speed'&&speedCapability()!=='available')return bad('dependency_not_installed','Speed capability is unavailable')
     return ok({version:`demo-${revision}`,canApply:true,kind:body.kind,changes:[{kind:net?'internet_path':body.kind,description:'Synthetic preview only'}],reloadServices:net?['dnsmasq']:['firewall'],requiresRenewal:net,recoveryAction:'Restore previous example'})
   }
   if(['/lanctrl/v2/device-network-policy/apply/','/lanctrl/v2/device-restrictions/apply/'].includes(key)){
     if(body.expectedVersion!==`demo-${revision}`)return bad('conflict','Example version changed; preview again')
     const net=key.includes('network-policy')
     if(net&&demo.authority!=='local')return bad('dhcp_authority_unavailable','Please configure on the address allocator')
     const p=policy(id);policies.set(id,net?{...p,static:body.static,targetId:body.targetId}:{...p,[body.kind]:body[body.kind]});revision++
     return ok({policy:net?network(id):policy(id),changed:true,transaction:{task:net?'network':'restrictions',stage:'verify',status:'committed'}})
   }
 }
 return undefined
}
