// Public v1 card contract. No imports from host state or business modules.
export const CARD_ABI = 'quickstart-card-v1'
export const externalId = /^plugin\.[a-z][a-z0-9-]{0,47}$/
export type CardForm = 'standard' | 'summary'
export type CardManifest = {
 schemaVersion:1; id:string; title:string; glyph:string; abi:typeof CARD_ABI; version:string
 entry:string; apiBase:string
 forms:Partial<Record<CardForm,{minWidth:number;minHeight:number}>>; stacked:CardForm[]
}
const plain=(v:any)=>v!==null&&typeof v==='object'&&!Array.isArray(v)
const exact=(v:any,keys:string[])=>plain(v)&&Object.keys(v).length===keys.length&&keys.every(k=>Object.prototype.hasOwnProperty.call(v,k))
const bounded=(v:any,min:number,max:number)=>Number.isInteger(v)&&v>=min&&v<=max
export function cardManifest(input:unknown):CardManifest {
 const v:any=input
 const bad=()=>{throw new Error('invalid_card_manifest')}
 if(!exact(v,['schemaVersion','id','title','glyph','abi','version','entry','apiBase','forms','stacked']))bad()
 if(v.schemaVersion!==1||v.abi!==CARD_ABI||typeof v.id!=='string'||!externalId.test(v.id))bad()
 if(typeof v.title!=='string'||!v.title.trim()||Array.from(v.title).length>64||/[\x00-\x1f\x7f]/.test(v.title))bad()
 if(typeof v.version!=='string'||!/^[A-Za-z0-9.+-]{1,64}$/.test(v.version)||typeof v.glyph!=='string'||!/^[a-z-]{1,32}$/.test(v.glyph))bad()
 if(typeof v.entry!=='string'||!/^\/luci-static\/[a-z][a-z0-9-]*\/(?:[a-zA-Z0-9_-]+\/)*[a-zA-Z0-9_-]+\.[a-f0-9]{8,64}\.js$/.test(v.entry))bad()
 if(typeof v.apiBase!=='string'||!/^\/cgi-bin\/luci\/admin\/services\/[a-z][a-z0-9-]*\/card$/.test(v.apiBase))bad()
 if(!plain(v.forms)||!Object.keys(v.forms).length||Object.keys(v.forms).some(k=>!['standard','summary'].includes(k)))bad()
 for(const size of Object.values(v.forms) as any[])if(!exact(size,['minWidth','minHeight'])||!bounded(size.minWidth,240,1200)||!bounded(size.minHeight,100,1600))bad()
 if(!Array.isArray(v.stacked)||new Set(v.stacked).size!==v.stacked.length||v.stacked.some((f:any)=>f!=='summary'||!v.forms[f]))bad()
 if(new TextEncoder().encode(JSON.stringify(v)).length>8192)bad()
 return JSON.parse(JSON.stringify(v))
}
// A duplicate is ambiguous: reject every occurrence, independent of file order.
export function cardManifests(input:unknown):CardManifest[]{
 if(!Array.isArray(input)||input.length>32||new TextEncoder().encode(JSON.stringify(input)).length>65536)throw new Error('invalid_card_registry')
 const counts=new Map<string,number>()
 for(const v of input)if(plain(v)&&typeof v.id==='string')counts.set(v.id,(counts.get(v.id)||0)+1)
 return input.flatMap(v=>{try{const m=cardManifest(v);return counts.get(m.id)===1?[m]:[]}catch{return []}})
}
export type TaskState='idle'|'running'|'cancel_requested'|'succeeded'|'partial'|'failed'|'cancelled'|'interrupted'
const transitions:Record<TaskState,TaskState[]>={idle:['running'],running:['cancel_requested','succeeded','partial','failed','interrupted'],cancel_requested:['cancelled','interrupted'],succeeded:[],partial:[],failed:[],cancelled:[],interrupted:[]}
export function taskTransition(from:TaskState,to:TaskState){return transitions[from]?.includes(to)??false}
export type Measurement<T>={status:'not_tested'|'ok'|'unavailable'|'error';value:T|null;measuredAt:string|null;error:string|null;runId?:string}
export type CardTask={id:string;instanceId:string;state:TaskState;stage:string;startedAt:string;finishedAt:string|null;deadlineAt:string;error:string|null;issues?:Record<string,string>}
export type CardResult={runId:string;measuredAt:string|null;download:Measurement<number>;upload:Measurement<number>;nat:Measurement<string>;ipv6:Measurement<string>;bandwidth?:{runId:string;measuredAt:string;downloadBps:number;uploadBps:number}}
export type CardSummary={schemaVersion:1;instanceId:string;service:{state:'ready'|'disabled'|'unavailable';persistenceError?:string};latest:CardResult|null;task:CardTask|null;lastTask?:CardTask}
