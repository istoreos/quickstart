// Design-only in-memory data. Never imported by the production entry.
export function createFastnetHistoryFixture(onLatest:(values:any)=>void,onNative:(task:any)=>void){
 const pending=new Map<string,any>(),history:any[]=[],latestResults:any={};let serial=0
 const now=()=>new Date().toISOString(),copy=(v:any)=>JSON.parse(JSON.stringify(v))
 const metric=(value:any,id:string,source='router')=>({status:'ok',value,runId:id,measuredAt:now(),source})
 function begin(kind:string,parameters:any={}){const run={id:'preview-history-'+(++serial),kind,parameters,state:'running',startedAt:now(),results:{},finishedAt:null};pending.set(run.id,run);return run}
 function merge(run:any,values:any){for(const [key,v]of Object.entries(values) as any){run.results[key]={...v,runId:run.id,measuredAt:now()};if(['ok','unavailable'].includes(v.status))latestResults[key]=run.results[key]}onLatest(copy(latestResults))}
 function finish(run:any,state=''){if(run.state!=='running')return run;run.state=state||'succeeded';run.finishedAt=now();pending.delete(run.id);history.unshift(run);if(history.length>32)history.pop();return run}
 const summary=()=>({schemaVersion:1,revision:String(serial),latestResults:copy(latestResults),activeRuns:copy([...pending.values()]),historyCount:history.length})
 async function api(path:string,body?:any):Promise<any>{
  const url=new URL(path,'http://preview.invalid'),action=url.pathname.split('/').pop()
  if(url.pathname.includes('/api/history/')){
   if(action==='begin')return copy(begin(body.kind,body.parameters))
   if(action==='summary')return summary()
   if(action==='list'){const page=Number(url.searchParams.get('page'))||0;return {items:copy(history.slice(page*8,page*8+8)),total:history.length,page,limit:32}}
   const run=pending.get(body?.id)||history.find(r=>r.id===(body?.id||url.searchParams.get('id')));if(!run)throw Error('Unknown preview run')
   if(action==='results')merge(run,Object.fromEntries(Object.entries(body.results).map(([k,v]:any)=>[k,{...v,source:'browser'}])))
   if(action==='finish')finish(run,body.state)
   return copy(run)
  }
  if(url.pathname.includes('/api/client-origin')||url.pathname.includes('/browser-origin'))return {isLoopback:false,isLocal:false}
  if(url.pathname.includes('test-usage'))return {used:0,limit:5,remaining:5}
  const kind=url.pathname.includes('/quick')?'quick':url.pathname.includes('/multi-stream')?'wan':url.pathname.includes('/nat')?'nat':url.pathname.includes('/ipv6')?'ipv6':url.pathname.includes('/homebox')?'upstream':'speedtest'
  const existing=pending.get(url.searchParams.get('runId')||''),run=existing||begin(kind)
  onNative({id:run.id,instanceId:'preview-service',state:'running',stage:'download',startedAt:run.startedAt,finishedAt:null})
  await new Promise(resolve=>setTimeout(resolve,700))
  const values:any={},data:any={downloadBps:951000000,uploadBps:49000000,pingMs:2,pingUs:2000,natType:'Full Cone',ipv6Info:'IPv6 可用',output:'NAT type: Full Cone',readiness:{ipv6Score:10,ipv4Score:10}}
  if(['quick','wan','speedtest'].includes(kind)){values.download=metric(data.downloadBps,run.id);values.upload=metric(data.uploadBps,run.id)}
  if(['quick','nat'].includes(kind))values.nat=metric(data.natType,run.id)
  if(['quick','ipv6'].includes(kind))values.ipv6=metric(data.ipv6Info,run.id)
  if(kind==='quick')values.ping=metric(2,run.id)
  if(kind==='upstream'){values.upstreamDownload=metric(data.downloadBps,run.id);values.upstreamUpload=metric(data.uploadBps,run.id)}
  merge(run,values);if(!existing)finish(run)
  onNative({...run,instanceId:'preview-service',state:'succeeded',stage:'ipv6'})
  return {...data,historyRunId:run.id}
 }
 return {api,summary}
}
