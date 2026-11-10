// Simulated service only; the external card and host loader are production code.
import {createFastnetHistoryFixture} from './fastnetHistoryFixture'
import defaults from '../../../backend/modules/homeconfig/defaults.json'
import contract from '../../../docs/contracts/fastnet-card.json'
import {configValue,type LayoutConfig} from '../home/configContract'
const params=new URLSearchParams(location.search),external=params.get('external'),scene=params.get('scene')||params.get('fastnet')||(external==='present'?'ready':'empty')
let installation='idle'
let installed=external==='present',revision=1,runs=0,installs=0,queries=0,active:any=null,lastTask:any=null
const copy=<T>(v:T):T=>JSON.parse(JSON.stringify(v)),initial=copy(contract.summary.latest) as any
const now=()=>new Date().toISOString()
if(scene==='stale')initial.measuredAt=new Date(Date.now()-3*86400000).toISOString()
if(scene==='partial'){initial.ipv6={status:'error',value:null,measuredAt:null,error:'probe_failed'}}
if(scene==='running')active={...copy(contract.task),id:'preview-run-active'}
if(scene==='failed')lastTask={...copy(contract.task),state:'failed',finishedAt:now()}
if(scene==='partial')lastTask={...copy(contract.task),state:'partial',issues:{ipv6:'probe_failed'},finishedAt:now()}
let latest:any=['empty','missing','install-error'].includes(scene)?null:initial
const historyFixture=createFastnetHistoryFixture(values=>{
 if(!latest)latest={runId:'',measuredAt:null,...Object.fromEntries(['download','upload','nat','ipv6'].map(k=>[k,{status:'not_tested',value:null,measuredAt:null,error:null}]))}
 for(const field of ['download','upload','nat','ipv6'])if(values[field])latest[field]={...values[field],error:null}
 if(values.download&&values.upload){latest.runId=values.download.runId;latest.measuredAt=values.upload.measuredAt}
},task=>{if(task.state==='running'){runs++;active=task}else{lastTask=task;active=null}})
const layout=configValue('layout',copy(external?contract.layoutV4:defaults.layout)) as LayoutConfig
if(external==='present'){
 layout.rows[0]={id:'network',template:'pair',height:320,heightMode:'chart',columns:[{weight:60,cards:['traffic'],split:50},{weight:40,cards:['connection'],split:50}]}
 layout.rows=layout.rows.filter(r=>r.id!=='external');layout.rows.splice(1,0,{id:'internet',template:'pair',height:280,heightMode:'content',columns:[{weight:50,cards:['plugin.fastnet'],split:50},{weight:50,cards:['networkInterface'],split:50}]})
}
if(params.get('form')==='summary'&&layout.widgetPreferences['plugin.fastnet'])layout.widgetPreferences['plugin.fastnet'].mode='summary'
const docs:Record<string,any>={layout,launcher:copy(defaults.launcher)},writes:any[]=[]
Object.defineProperty(window,'__quickstartCardFixture',{configurable:true,value:{fullAPI:(path:string,body:any)=>historyFixture.api(path.replace(/^\/apps\/fastnet/,''),body),get writes(){return copy(writes)},get stats(){return {runs,queries,installs}}}})
const manifest=()=>({...contract.manifest,...(window as any).__quickstartPreviewCardManifest,entry:params.get('entry')||(window as any).__quickstartPreviewCardManifest?.entry||contract.manifest.entry})
export async function installPreviewCard(id:string){if(id!=='plugin.fastnet')throw Error('No preview installer');installs++;await new Promise(r=>setTimeout(r,120));if(scene==='install-error'&&installs===1)throw Error('模拟下载失败，请重试');installed=true}
export function cardFixture(key:string,method:string,body:any):{status:number;data:any}|undefined{
 const prefix='/cgi-bin/luci/admin/services/fastnet/card/'
 if(key.startsWith(prefix)){
  const action=key.slice(prefix.length)
  if(scene==='permission')return {status:403,data:{error:'forbidden'}}
  if(scene==='unavailable')return {status:503,data:{error:'unavailable'}}
  if(action==='summary')return {status:200,data:{schemaVersion:1,instanceId:'preview-service',service:{state:scene==='stopped'||scene==='disabled'?'disabled':'ready'},latest:copy(latest),latestResults:historyFixture.summary().latestResults,historyCount:historyFixture.summary().historyCount,task:copy(active),lastTask:copy(lastTask)}}
  if(action==='run'&&method==='POST'){
   if(active)return {status:409,data:{error:'busy',task:copy(active)}}
   runs++;queries=0;active={id:'preview-run-'+runs,instanceId:'preview-service',state:'running',stage:'download',startedAt:now(),finishedAt:null,deadlineAt:new Date(Date.now()+120000).toISOString(),error:null};return {status:202,data:copy(active)}
  }
  if(action==='cancel'&&method==='POST'){if(active){lastTask={...active,state:'cancelled',finishedAt:now()};active=null}return {status:200,data:copy(lastTask)}}
  if(action==='task'){
   queries++
   if(scene==='poll-error')return {status:503,data:{error:'unavailable'}}
   if(active&&queries>=2){lastTask={...active,state:scene==='error'&&runs===1?'failed':'succeeded',finishedAt:now(),error:scene==='error'&&runs===1?'measurement_failed':null};active=null
    if(lastTask.state==='succeeded'){latest=copy(contract.summary.latest);latest.runId=lastTask.id;latest.measuredAt=now();for(const f of ['download','upload','nat','ipv6'])latest[f].measuredAt=latest.measuredAt;latest.download.value=951000000;latest.upload.value=49000000}
   }else if(active)active.stage='upload'
   return {status:200,data:copy(active||lastTask)}
  }
 }
 if(key==='/app/install/'&&body.name==='app-meta-fastnet'){installation='installing';void installPreviewCard('plugin.fastnet').then(()=>{installation='completed'}).catch(()=>{installation='failed'});return {status:200,data:{success:0,detail:'installing'}}}
 if(key==='/app/check/'&&body.name==='app-meta-fastnet')return {status:200,data:{result:{status:installed?'installed':'uninstalled',installation}}}
 if(key==='/home/v1/cards/'){
  if(external==='discovery-error')return {status:503,data:{error:'unavailable'}}
  return {status:200,data:{schemaVersion:1,cards:installed?[manifest()]:[],issues:external==='incompatible'?[{id:'plugin.fastnet',code:'unsupported_abi'}]:[]}}
 }
 const domain=key==='/home/v1/layout/'?'layout':key==='/home/v1/launcher/'?'launcher':''
 if(!domain)return
 const version=()=> '0123456789abcdef0123456789abcdef:'+revision
 if(method==='POST'){
  if(body.expectedVersion!==version())return {status:409,data:{error:'conflict'}}
  try{docs[domain]=configValue(domain,body.value);revision++;writes.push({domain,expectedVersion:body.expectedVersion,value:copy(docs[domain])});if(writes.length>32)writes.shift()}catch{return {status:422,data:{error:'invalid'}}}
 }
 return {status:200,data:{success:0,result:{state:'saved',version:version(),value:docs[domain]}}}
}
