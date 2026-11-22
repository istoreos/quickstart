// DESIGN ONLY: capability samples, never probes this container or installs packages.
import {reactive} from 'vue'
import type {StoragePlan, Purpose} from './storagePurposePrototype'
export type Feature='partition'|'ext4'|'ext4grow'|'mount'|'lvm'|'thin'|'raid'|'smart'|'lighthost'
export type CapabilityState='ready'|'missing'|'unsupported'|'checking'|'unknown'|'error'|'forbidden'
export type Capability={state:CapabilityState;reason:string;next:string}
export const featureNames:Record<Feature,string>={partition:'分区管理',ext4:'EXT4 格式化',ext4grow:'EXT4 扩容',mount:'文件挂载',lvm:'LVM 卷管理',thin:'精简存储与快照',raid:'RAID 阵列',smart:'硬盘健康检测',lighthost:'LightHost 存储接入'}
export const stateNames:Record<CapabilityState,string>={ready:'可用',missing:'缺少组件',unsupported:'系统不支持',checking:'正在检测',unknown:'尚未确认',error:'检测失败',forbidden:'无操作权限'}
export const capabilityProfiles=[{id:'ready',label:'存储能力齐全'},{id:'basic',label:'仅普通文件存储'},{id:'no-lvm',label:'缺少 LVM 工具'},{id:'no-thin',label:'内核不支持 thin-pool'},{id:'no-raid-tools',label:'缺少 RAID 管理工具'},{id:'no-grow',label:'缺少 EXT4 扩容工具'},{id:'no-raid',label:'系统不支持 RAID'},{id:'raid-limited',label:'仅支持部分 RAID 级别'},{id:'no-lighthost',label:'LightHost 未安装'},{id:'no-smart',label:'缺少 SMART 组件'},{id:'no-ext4',label:'缺少 EXT4 格式化工具'},{id:'checking',label:'正在检测系统能力'},{id:'unknown',label:'旧后端未提供能力信息'},{id:'error',label:'能力检测失败'},{id:'forbidden',label:'当前账号只读'}]
const ready=():Record<Feature,Capability>=>Object.fromEntries(Object.keys(featureNames).map(key=>[key,{state:'ready',reason:'示例已确认具备所需能力',next:''}])) as Record<Feature,Capability>
export const capabilities=reactive({profile:'ready',features:ready(),levels:{} as Record<string,Capability>,sampledAt:new Date().toLocaleString('zh-CN')})
export function setCapabilityProfile(profile:string){
 profile=capabilityProfiles.some(p=>p.id===profile)?profile:'unknown';capabilities.profile=profile;capabilities.features=ready();capabilities.levels={};capabilities.sampledAt=new Date().toLocaleString('zh-CN')
 const set=(feature:Feature,state:CapabilityState,reason:string,next:string)=>{capabilities.features[feature]={state,reason,next}}
 if(['basic','no-lvm'].includes(profile))set('lvm','missing','未检测到 LVM 管理工具，不能创建或修改 LVM 卷。','请检查当前固件的软件包支持；补齐兼容组件后重新检测。')
 if(profile==='no-raid-tools')set('raid','missing','未检测到可用的 mdadm 管理工具，不能提交阵列变更。','检查固件中兼容的阵列管理组件后重新检测。')
 if(profile==='no-grow')set('ext4grow','missing','缺少 EXT4 扩容工具，无法完成卷与文件系统的一起扩容。','补齐扩容工具后重新检测；新建 EXT4 的能力单独判断。')
 if(profile==='no-thin')set('thin','unsupported','当前内核不支持 dm-thin-pool，LVM 工具可用也不能创建精简存储。','请使用提供相应内核能力的兼容固件；普通文件存储仍可使用。')
 if(['basic','no-raid'].includes(profile))set('raid','unsupported','当前系统缺少可用的 MD RAID 内核能力。','请确认固件同时提供阵列驱动与管理工具；接入更多硬盘不能补齐系统能力。')
 if(profile==='raid-limited')for(const level of ['raid5','raid6','raid10'])capabilities.levels[level]={state:'unsupported',reason:'当前系统未提供此 RAID 级别所需的内核能力。',next:'可选择本机支持的级别，或使用兼容固件后重新检测。'}
 if(profile==='no-lighthost')set('lighthost','missing','LightHost 尚未安装，不能准备由它管理的主机存储。','安装并完成 LightHost 初始化后重新检测；系统应用卷不受此项影响。')
 if(profile==='no-smart')set('smart','missing','未检测到硬盘健康管理组件，不能提交自检或保存监测计划。','检查固件的 SMART 组件支持；单块硬盘和 USB 转接器还需分别确认。')
 if(profile==='no-ext4')set('ext4','missing','缺少 EXT4 格式化工具，不能创建新的文件系统。','补齐兼容工具后重新检测；已挂载文件仍可访问。')
 if(['checking','unknown','error','forbidden'].includes(profile))for(const key of Object.keys(featureNames) as Feature[])set(key,profile as CapabilityState,profile==='checking'?'正在读取系统能力，暂不提交新的存储操作。':profile==='unknown'?'后端未提供可信的能力结果，不能按可用处理。':profile==='error'?'未能取得本机能力结果；这不代表系统一定不支持。':'当前账号没有存储管理权限。',profile==='forbidden'?'请使用具有存储管理权限的账号。':profile==='unknown'?'请检查后端版本和能力接口。':'检查服务连接后重新检测。')
}
export function featureIssue(feature:Feature){const c=capabilities.features[feature];return c.state==='ready'?'':featureNames[feature]+' · '+stateNames[c.state]+'：'+c.reason}
export function requireFeatures(features:Feature[]){return features.map(featureIssue).find(Boolean)||''}
export function purposeIssue(purpose:Purpose){return purpose==='files'?'':requireFeatures(purpose==='apps'?['lvm','thin']:['lvm','thin','lighthost'])}
export function planFeatures(plan:StoragePlan):Feature[]{return plan.preserveExisting?['mount']:['partition',...(plan.files||plan.management==='separate'?['ext4','mount'] as Feature[]:[]),...(plan.hosts||plan.apps?['lvm','thin'] as Feature[]:[]),...(plan.hosts?['lighthost'] as Feature[]:[]),...(plan.apps?['ext4'] as Feature[]:[])]}
export function raidIssue(level?:string){if(level&&/^\d+$/.test(level))level='raid'+level;return requireFeatures(['raid'])||(level&&capabilities.levels[level]?.state!=='ready'&&capabilities.levels[level]?capabilities.levels[level].reason:'')}
export function raidCreateIssue(level?:string){return raidIssue(level)||requireFeatures(['partition','ext4','mount'])}
// A retry keeps the same simulated evidence. Never turn a failure into a successful probe by timeout.
export function recheckCapabilities(){setCapabilityProfile(capabilities.profile)}
