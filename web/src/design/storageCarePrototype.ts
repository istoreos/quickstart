// Design only. All maintenance, events and tasks are in-memory examples; no backend writes.
import {computed, reactive, watch} from 'vue'
import {workbench, nextId, type Job} from './storageWorkbenchPrototype'

export type CarePanel = 'health'|'capacity'|'readiness'|'discovery'|'maintenance'|'recovery'|'events'|'task'|'snapshots'|'support'
export type CareTask = {id:number; title:string; target:string; state:string; stages:{name:string;state:string;detail:string}[]; started:string; next:string}
const now=()=>new Date().toLocaleString('zh-CN',{hour12:false})
export type CareEvent={id:string;title:string;panel:CarePanel;target:string;firstSeen:string;resolvedAt:string;episode:number}
export const care=reactive({
  mdSchedule:'未设置',scene:'normal', queryInterrupted:false, sampledAt:now(), aliases:{} as Record<string,string>,
  health:{} as Record<string,{state:string; result:string; schedule:string}>,
  events:[] as CareEvent[],tasks:[] as CareTask[], read:[] as string[], notices:[] as string[],
  channel:'站内提醒', notificationTest:'', notificationState:'idle', deliveryFailure:false,
  applicationStates:{} as Record<string,string>,applicationState:'就绪', expectedUUID:'DEMO-FS-DOCKER',
  discoveryState:'idle', recoveryChoice:'new',
})
export function resetCare(scene='normal'){
  care.applicationStates={};care.mdSchedule='未设置';care.scene=scene;care.sampledAt=now();care.queryInterrupted=scene==='read-error';care.tasks=[];care.events=[];care.read=[];care.notices=[];care.health={};care.aliases={};care.discoveryState='idle';care.notificationTest='';care.notificationState='idle';care.deliveryFailure=false;care.channel='站内提醒';
  care.applicationState=scene==='mount-wait'?'等待存储':scene==='mount-failed'?'挂载失败':'就绪'
  if(scene==='partial-grow')care.tasks.push({id:nextId(),title:'扩充家庭文件阵列',target:'md0',state:'部分完成',started:now(),next:'成员已加入，请先重新读取阵列状态；不要重新添加或清空成员。文件系统扩展需阵列调整完成后单独预检。',stages:[{name:'加入成员',state:'完成',detail:'已识别新成员 /dev/sdf1'},{name:'调整阵列',state:'失败',detail:'示例：mdadm grow 退出码 1，保留现有阵列状态。'},{name:'扩展 EXT4',state:'未执行',detail:'设备 2048 GiB；文件系统仍为 1024 GiB。'}]})
  if(scene==='resume-task')care.tasks.push({id:nextId(),title:'阵列重建',target:'md0',state:'执行中',started:now(),next:'此为已提交任务的示例，可离开本页后返回查看；演示进度不随时间伪造增长。',stages:[{name:'提交成员恢复',state:'完成',detail:'示例成员已提交'},{name:'内核重建',state:'执行中',detail:'示例采样：358 / 1024 GiB，剩余时间未知。实际实现读取 sync_completed。'},{name:'核对健康与容量',state:'等待',detail:'重建完成后重新读取'}]})
}
export const careAlerts=computed(()=>[
  ...Object.entries(care.health).filter(([,h])=>['异常','失败'].includes(h.state)).map(([id])=>({id:'health:'+id,title:id+' 健康检查需要处理',panel:'health' as CarePanel,target:id})),
  ...workbench.drives.filter(d=>!d.connected).map(d=>({id:'offline:'+d.serial,title:d.id+' 未连接',panel:'health' as CarePanel,target:d.id})),
  ...workbench.raids.filter(r=>r.status!=='正常').map(r=>({id:'raid:'+r.id,title:r.name+'：'+r.status,panel:'maintenance' as CarePanel,target:r.id})),
  ...workbench.pools.filter(p=>p.used/p.size>=.9||p.metadata>=90).map(p=>({id:'pool:'+p.id,title:p.name+'：'+(p.metadata>=90?'元数据空间紧张':'数据空间紧张'),panel:'capacity' as CarePanel,target:p.id})),
  ...(workbench.readState!=='fresh'?[{id:'inventory-read',title:'存储状态读取失败，上次数据已过期',panel:'events' as CarePanel,target:''}]:[]),
  ...(care.applicationState!=='就绪'?[{id:'mount:docker',title:'宿主 Docker：'+care.applicationState,panel:'readiness' as CarePanel,target:'docker'}]:[]),
])
watch(careAlerts,alerts=>{
 const active=new Set(alerts.map(a=>a.id))
 for(const alert of alerts){const previous=care.events.find(e=>e.id===alert.id&&!e.resolvedAt);if(previous){previous.title=alert.title}else {care.read=care.read.filter(id=>id!==alert.id);care.events.unshift({...alert,firstSeen:now(),resolvedAt:'',episode:nextId()})}}
 for(const event of care.events)if(!active.has(event.id)&&!event.resolvedAt)event.resolvedAt=now()
},{immediate:true})
export function healthFor(id:string){return care.health[id]??(care.health[id]={state:care.scene==='health-unknown'?'未知':'尚未测试',result:'没有近期测试结果；在线和温度正常不等于健康检查通过。',schedule:'未设置'})}
export function maintenanceBusy(){return workbench.busy||workbench.raids.some(r=>r.status==='重建中'||r.nativeStatus?.includes('resync'))||care.tasks.some(t=>t.state==='执行中')}
export function startCareTask(title:string,target:string,steps:string[]){
  if(maintenanceBusy())return false
  const task:CareTask={id:nextId(),title,target,state:'执行中',started:now(),next:'等待示例结果；可关闭详情后继续查看任务。',stages:steps.map((name,i)=>({name,state:i?'等待':'执行中',detail:i?'尚未开始':'已提交演示任务'}))}
  care.tasks.unshift(task);return true
}
export function finishCareTask(task:CareTask,outcome:'完成'|'失败'|'已中止'){
  if(task.state!=='执行中')return
  task.state=outcome;let stopped=false
  task.stages.forEach(s=>{s.state=stopped?'未执行':outcome;s.detail=stopped?'前序未完成，未继续执行':outcome==='完成'?'示例结果已读取':outcome==='失败'?'示例命令失败：保留输出和退出码，不解释为正在运行':'用户中止示例测试';if(outcome!=='完成')stopped=true})
  task.next=outcome==='完成'?'已保存演示结果。请按计划继续关注，不代表全部文件内容经过验证。':'核对最新状态后再操作，不自动重放已执行步骤。'
  if(task.title==='SMART 短测试'){const h=healthFor(task.target);h.state=outcome;h.result=now()+' · '+(outcome==='完成'?'示例测试通过，未发现本次自检错误':outcome+'；查看任务阶段和原始结果。')}
}
export function jobAsTask(job:Job):CareTask{return {id:job.id,title:job.title,target:job.target,state:job.status,started:job.at||'本次会话',next:job.status==='完成'?'查看更新后的资源；文件区请挂载后使用，应用卷请完成接入。':'核对实际布局，不自动重试整批操作。',stages:[{name:job.title,state:job.status,detail:job.detail}]}}
export function exportCareDiagnostics(){
  // Explicit allowlist. Never collect account data, host files, credentials or command environment.
  return JSON.stringify({prototype:true,generatedAt:now(),readState:workbench.readState,sampledAt:care.sampledAt,devices:workbench.drives.map(d=>({path:d.id,identity:d.serial.replace(/.{3}$/,'***'),sizeGiB:d.size,connected:d.connected,parts:d.parts.map(p=>({id:p.id,kind:p.kind,sizeGiB:p.size}))})),pools:workbench.pools.map(p=>({id:p.id,owner:p.owner,dataGiB:p.used,metadataPercent:p.metadata})),tasks:care.tasks.map(t=>({id:t.id,state:t.state,stages:t.stages.map(s=>({name:s.name,state:s.state}))}))},null,2)
}
