// DESIGN ONLY: responses for the shipped RAID components, never disk commands.
// Keep production request shapes and error handling; this adapter only updates example inventory.
import {workbench,raidDrive,type Raid,type Drive} from './storageWorkbenchPrototype'
let activated=false
const result=(value:unknown={})=>({status:200,data:{success:0,result:value,previewOnly:true}})
const error=(message:string)=>({status:200,data:{error:message,previewOnly:true}})
const diskId=(path:string)=>path.replace(/^\/dev\//,'').replace(/\d+$/,'')
const candidates=()=>workbench.drives.filter(d=>d.connected&&!d.parts.some(p=>p.protected||p.mount||['hosts','apps','raid'].includes(p.kind)))
const capacity=(drives:Drive[],level:string)=>{const n=drives.length,min=Math.min(...drives.map(d=>d.size));return level==='jbod'||level==='linear'?drives.reduce((s,d)=>s+d.size,0):min*(level==='raid1'?1:level==='raid5'?n-1:level==='raid6'?n-2:level==='raid10'?n/2:n)}
const useMember=(disk:Drive,raid:Raid)=>{disk.parts=[{id:disk.id+'1',name:'RAID 成员',start:0,size:disk.size,kind:'raid',fs:'linux_raid_member',used:0,mount:'',owner:raid.id}]}
function serialize(raid:Raid){
 const file=raidDrive(raid).parts[0],level=raid.level.startsWith('raid')||raid.level==='linear'||raid.level==='jbod'?raid.level:'raid'+raid.level
 return {name:raid.id,path:'/dev/'+raid.id,venderModel:'示例 RAID',active:'active',status:raid.nativeStatus||(raid.status==='正常'?'clean':raid.status==='降级'?'clean, degraded':'clean, recovering'),level,members:raid.members.map(id=>'/dev/'+id+'1'),total:raid.size+' GiB',size:raid.size+' GiB',sizeInt:String(raid.size*1024**3),used:raid.used+' GiB',usage:raid.used/raid.size*100,tranName:'md',partLabelType:'LOOP',rebuildStatus:raid.status==='重建中'?'recovery = '+raid.progress+'% (示例，非真实进度)':'',childrens:[{name:raid.id,path:'/dev/'+raid.id,filesystem:'ext4',uuid:'DEMO-FS-'+raid.id,mountPoint:file.mount,sizeInt:String(file.size*1024**3),total:file.size+' GiB',used:raid.used+' GiB',usage:raid.used/file.size*100,isSystemRoot:false,isDockerRoot:false,isReadOnly:false}]}
}
export function raidWorkbenchFixture(key:string,method:string,body:Record<string,any>){
 if(location.pathname==='/storage'||location.pathname==='/nas')activated=true
 if(!activated||!key.startsWith('/raid/'))return
 if(key==='/raid/list/'&&method==='GET')return result({disks:workbench.raids.map(serialize)})
 if(key==='/raid/create/list/'&&method==='GET')return result({members:candidates().map(d=>({name:d.id,path:'/dev/'+d.id,model:d.model,sizeStr:d.size+' GiB'}))})
 if(method!=='POST')return
 const raid=workbench.raids.find(r=>'/dev/'+r.id===body.path)
 if(key==='/raid/detail/')return raid?result({detail:`${body.path}:\n  Version : 1.2\n  Raid Level : ${serialize(raid).level}\n  Array Size : ${raid.size} GiB\n  State : ${serialize(raid).status}\n  Members : ${serialize(raid).members.join(' ')}\n\n产品交互示例；未读取真实 mdadm。`}):error('获取raid详情失败')
 if(workbench.failNext){workbench.failNext=false;return error('示例：RAID 操作失败；原操作组件显示此错误，设备未变化。')}
 if(key==='/raid/autofix/'){
  // The missing-array scene keeps member signatures so the original scan action has visible results.
  if(!workbench.raids.length&&workbench.drives.some(d=>d.parts.some(p=>p.owner==='md0'&&p.kind==='raid'))){workbench.raids.push({id:'md0',name:'家庭文件镜像',level:'1',members:['sdd','sde'],size:1024,used:368,status:'正常',progress:100,mount:'/mnt/md0'})}
  return result()
 }
 if(key==='/raid/create/'){
  const level=String(body.level),min=level==='raid5'?3:['raid6','raid10'].includes(level)?4:2
  // Mirror the current backend: the UI says jbod, but NormalizeLevel accepts linear instead.
  if(!['linear','raid0','raid1','raid5','raid6','raid10'].includes(level))return error('不支持的RAID级别')
  const ids=(Array.isArray(body.devicePaths)?body.devicePaths:[]).map(diskId)
  const drives=ids.map((id:string)=>candidates().find(d=>d.id===id))
  if(ids.length<min)return error('没有足够的成员设备')
  if(new Set(ids).size!==ids.length||drives.some((d:Drive|undefined)=>!d))return error('无效或重复的成员设备')
  let index=0;while(workbench.raids.some(r=>r.id==='md'+index))index++
  const created:Raid={id:'md'+index,name:'md'+index,level,members:ids,size:capacity(drives as Drive[],level),used:0,status:'正常',progress:100,mount:'/mnt/md'+index}
  // RaidPostCreate formats and mounts the new md device through the NAS lifecycle service.
  drives.forEach(d=>{if(d)useMember(d,created)});workbench.raids.push(created)
  return result(serialize(created).childrens[0])
 }
 if(!raid)return error('raid not found')
 if(key==='/raid/delete/'){
  if(raidDrive(raid).parts[0].mount)return error('删除 RAID 设备之前请先卸载文件系统')
  for(const id of raid.members){const d=workbench.drives.find(d=>d.id===id);if(d)d.parts=[{id:id+'-free',name:'未分配',start:0,size:d.size,kind:'free',fs:'—',used:0,mount:''}]}
  workbench.raids=workbench.raids.filter(r=>r!==raid);return result()
 }
 const member=diskId(String(body.memberPath||'')),disk=workbench.drives.find(d=>d.id===member)
 if(key==='/raid/remove/'){
  if(!raid.members.includes(member))return error('删除成员失败')
  raid.members=raid.members.filter(id=>id!==member);raid.status='降级';raid.nativeStatus=undefined;return result()
 }
 if(key==='/raid/add/'||key==='/raid/recover/'){
  if(!disk||!candidates().some(d=>d.id===member))return error('成员设备不可用')
  raid.filesystemSize??=raid.size
  raid.members.push(member);useMember(disk,raid)
  if(key==='/raid/add/')raid.size=capacity(workbench.drives.filter(d=>raid.members.includes(d.id)),serialize(raid).level)
  // Recovery adds a member; it is not an array-size change or an automatic file-system resize.
  raid.status='重建中';raid.nativeStatus=key==='/raid/add/'?'clean, resyncing(PENDING)':'clean, degraded, recovering';raid.progress=35
  return result()
 }
 return error('Preview RAID fixture is not defined for '+key)
}
