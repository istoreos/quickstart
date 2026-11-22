// THROWAWAY DESIGN: replace presentation in memory while retaining shipped RAID handlers.
// Production vite.config.ts does not import this file. No mdadm or API business logic lives here.
const diskIcon='<svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="3"/><path d="M3 14h18M7 17h.01M11 17h.01M8 8h8"/></svg>'
const heading=(title:string,subtitle:string)=>`<header class="rs-heading"><div><span class="rs-kicker">RAID / 存储管理</span><h2>${title}</h2><p>${subtitle}</p></div><button type="button" class="rs-close" aria-label="关闭 RAID 操作" :disabled="${title==='技术详情'?'false':'disabled'}" @click="onClose">×</button></header>`
const steps='<nav class="rs-steps" aria-label="操作步骤"><span :class="{active:rStep===1}"><i>1</i>选择配置</span><b>—</b><span :class="{active:rStep===2}"><i>2</i>核对并执行</span></nav>'
const pageTemplate=`<section class="rp" data-raid-presentation>
 <div class="rp-main" :inert="!!rSelected">
  <header class="rp-heading"><div><h2>RAID 阵列</h2><p>查看阵列的状态、硬盘成员和文件存储。</p></div><div class="rp-heading-actions"><button class="rp-secondary" :disabled="disabled||!!raidIssue()" @click="onRecoverRaid">扫描恢复</button><button class="primary" aria-label="创建阵列" :disabled="!!raidCreateIssue()" @click="onCreateRaid"><span aria-hidden="true">＋</span> 创建阵列</button></div></header>
  <p v-if="raidCreateIssue()" class="sw-note warning" role="status">{{raidCreateIssue()}} 已有阵列记录保留，不能据此判断当前阵列健康。</p><article v-for="item in raid.disksList" :key="item.path" class="rp-card" :data-array="item.name">
   <header><div class="rp-array-title"><span class="rp-array-icon">${diskIcon}</span><div><h3>{{item.name}} <small>{{rLevel(item.level)}}</small></h3><p>{{item.path}}</p></div></div><span class="rp-health" :class="rHealth(item)"><i></i>{{rStatus(item)}}</span></header>
   <div class="rp-card-content"><section class="rp-capacity"><span class="rp-label">阵列容量</span><strong>{{item.size||'—'}}</strong><template v-if="getRaidMountPoint(item).length"><div class="rp-usage"><i :style="{width:Math.max(0,Math.min(100,item.usage||0))+'%'}"></i></div><p>文件已用 {{item.used||'—'}} <span>{{Math.round(item.usage||0)}}%</span></p></template><p v-else class="rp-muted">挂载后可查看文件占用</p></section>
    <section class="rp-members"><span class="rp-label">硬盘成员 <small>{{item.members.length}} 块</small></span><div><button v-for="member in item.members" :key="member" @click="rLocate(member)"><span class="rp-member-icon">${diskIcon}</span><span><strong>{{member.replace('/dev/','')}}</strong><small>查看硬盘位置 ›</small></span></button></div></section>
    <section class="rp-files"><span class="rp-label">文件存储</span><template v-if="getRaidMountPoint(item).length"><strong v-for="file in item.childrens.filter(p=>p.mountPoint)" :key="file.path">{{file.mountPoint}}<small>{{file.filesystem?.toUpperCase()||'文件系统'}} · 已挂载</small></strong></template><template v-else><strong>尚未挂载<small>挂载后即可访问和共享文件</small></strong></template><button class="link" @click="rFiles(item)">{{getRaidMountPoint(item).length?'查看存储位置':'去挂载'}} ›</button></section>
   </div>
   <p v-if="item.status.includes('degraded')" class="rp-attention">阵列保护能力已降低。请核对硬盘连接，再通过“恢复成员”处理。</p><p v-else-if="item.rebuildStatus" class="rp-attention">{{rStatus(item)}} · {{item.rebuildStatus}}</p>
   <footer><span>{{rLevelDescription(item.level)}}</span><div><button v-if="item.status.includes('degraded')" class="primary" :disabled="!!raidIssue(item.level)" @click="onRecoverRaidItem(item)">恢复成员</button><button @click="rOpen(item)">查看与管理 <span>›</span></button></div></footer>
  </article>
  <div v-if="!raid.disksList.length" class="rp-empty"><span class="rp-array-icon">${diskIcon}</span><h3>还没有可见的 RAID 阵列</h3><p>具备系统能力及可用硬盘后才能创建。已有阵列未显示时，先识别已有阵列，再核对恢复影响。</p><button class="primary" :disabled="!!raidCreateIssue()" @click="onCreateRaid">创建阵列</button></div>
  <details class="rp-help"><summary>了解 RAID 与使用须知</summary><p>RAID 使用多块硬盘组成存储；保护能力取决于所选级别，不能代替独立备份。</p><p>已有 RAID 功能仍处于公测阶段，操作前请确认重要文件已有备份。系统重置或重启后阵列未显示时，请先只读识别；“扫描恢复”会改写配置并重启阵列服务，不是只读扫描。</p></details>
 </div>
 <Transition name="sw-detail"><div v-if="rSelected" class="rp-shade" @click.self="rClose"><aside class="rp-drawer" role="dialog" aria-modal="true" aria-label="RAID 阵列详情" tabindex="-1" ref="rDrawer">
  <header class="rp-drawer-header"><span>{{rView==='delete'?'删除阵列':'阵列详情'}}</span><button aria-label="关闭阵列详情" @click="rClose">×</button></header>
  <div class="rp-drawer-body"><span class="rs-kicker">{{rSelected.path}}</span><h2>{{rSelected.name}}</h2><span class="rp-health" :class="rHealth(rSelected)"><i></i>{{rStatus(rSelected)}} · {{rLevel(rSelected.level)}}</span>
   <template v-if="rView==='details'"><dl><dt>阵列容量</dt><dd>{{rSelected.size}}</dd><dt>成员数量</dt><dd>{{rSelected.members.length}} 块</dd><dt>挂载位置</dt><dd>{{rSelected.childrens.filter(p=>p.mountPoint).map(p=>p.mountPoint).join('、')||'未挂载'}}</dd><dt>系统状态</dt><dd>{{rSelected.status}}</dd></dl>
    <div class="rp-manage"><h3>管理阵列</h3><button :disabled="!!raidIssue(rSelected.level)||getDisabled(rSelected)" @click="onEidtRaidItem(rSelected)"><span><strong>扩充阵列</strong><small>添加硬盘成员，沿用现有扩充流程</small></span><b>›</b></button><button :disabled="!!raidIssue(rSelected.level)||getDisabled(rSelected)" @click="onRemoveRaidItem(rSelected)"><span><strong>移除成员</strong><small>选择从阵列中移除的成员</small></span><b>›</b></button><p v-if="raidIssue(rSelected.level)||getDisabled(rSelected)" class="rp-muted">{{raidIssue(rSelected.level)||rDisabledReason(rSelected)}}</p><button :disabled="!!raidIssue(rSelected.level)" @click="onRecoverRaidItem(rSelected)"><span><strong>恢复成员</strong><small>选择硬盘恢复阵列；与扩充操作不同</small></span><b>›</b></button><button @click="onRaidItemInfo(rSelected)"><span><strong>技术详情</strong><small>查看当前 RAID 设备的原始信息</small></span><b>›</b></button></div>
    <button class="rp-delete-link" :disabled="!!raidIssue(rSelected.level)" @click="rView='delete';rAccepted=false;rTyped=''">删除阵列</button>
   </template>
   <template v-else><div v-if="getRaidMountPoint(rSelected).length" class="rs-impact"><strong>先卸载文件系统</strong><p>此阵列仍有挂载的文件系统。请先确认没有应用占用并卸载，再返回删除。</p><button @click="rFiles(rSelected)">前往卸载 ›</button></div><template v-else><div class="rs-impact danger"><strong>删除后，阵列中的文件将无法正常访问</strong><p>将删除 {{rSelected.path}} 的阵列配置并处理现有成员。此操作可能导致数据丢失；删除成功后，再次组建阵列建议稍等几分钟。</p></div><label class="rs-field">输入阵列名称 {{rSelected.name}}<input v-model="rTyped" aria-label="确认阵列名称" autocomplete="off" /></label><label class="rs-check"><input type="checkbox" v-model="rAccepted" />我已核对阵列和成员，并确认重要文件已有备份。</label></template></template>
  </div>
  <footer class="rp-drawer-footer"><template v-if="rView==='delete'"><button @click="rView='details'" :disabled="rDeleting">返回详情</button><button v-if="!getRaidMountPoint(rSelected).length" class="danger" :disabled="!!raidIssue(rSelected.level)||!rAccepted||rTyped!==rSelected.name||rDeleting" @click="rDelete">{{rDeleting?'正在删除…':'确认删除'}}</button></template><button v-else @click="rClose">返回阵列列表</button></footer>
 </aside></div></Transition>
</section>`

const pageScript=`
import {watch as rWatch,nextTick as rNextTick,onMounted as rMounted} from 'vue'
import {raidIssue,raidCreateIssue} from '/src/design/storageCapabilitiesPrototype'
const rSelected=ref<Disksinfo|null>(null),rView=ref('details'),rAccepted=ref(false),rTyped=ref(''),rDeleting=ref(false),rDrawer=ref<HTMLElement>()
let rTrigger:HTMLElement|null=null,rOverflow:string|undefined
// The destructive acknowledgement is now rendered in the review panel instead of browser confirm().
const confirm=(_message:string)=>!!rSelected.value&&rAccepted.value&&rTyped.value===rSelected.value.name
const rLevel=(level:string)=>level==='jbod'||level==='linear'?'JBOD':level.replace('raid','RAID ')
const rHealth=(item:Disksinfo)=>raidIssue(item.level)?'neutral':/degraded|recover|resync|inactive/.test(item.status)?'attention':item.status.includes('clean')?'good':'neutral'
const rStatus=(item:Disksinfo)=>raidIssue(item.level)?'待重新核实':item.status.includes('degraded')?'已降级':item.status.includes('resync')?'同步中':item.status.includes('recover')?'恢复中':item.status.includes('inactive')?'未激活':item.status.includes('clean')?'正常':item.status||'状态待确认'
const rLevelDescription=(level:string)=>['raid0','jbod','linear'].includes(level)?'此级别不提供数据冗余':'冗余存储 · 仍需独立备份'
const rDisabledReason=(item:Disksinfo)=>['raid0','jbod'].includes(item.level)?'当前级别不支持此项成员变更。':item.status.includes('degraded')?'阵列已降级，扩充和移除暂不可用。':'阵列正在同步，扩充和移除暂不可用。'
function rOpen(item:Disksinfo){rTrigger=document.activeElement as HTMLElement;rView.value='details';rAccepted.value=false;rTyped.value='';rSelected.value=item}
function rClose(){if(!rDeleting.value)rSelected.value=null}
async function rFiles(item:Disksinfo){rClose();await rNextTick();window.dispatchEvent(new CustomEvent('quickstart-storage-open',{detail:{tab:'disks',part:item.name}}))}
async function rLocate(member:string){rClose();await rNextTick();window.dispatchEvent(new CustomEvent('quickstart-storage-open',{detail:{tab:'disks',part:member.replace('/dev/','').replace(/\\(.*?\\)/g,'')}}))}
async function rDelete(){if(!rSelected.value||raidIssue(rSelected.value.level)||!confirm(''))return;rDeleting.value=true;try{await onDeleteRaidItem(rSelected.value)}finally{rDeleting.value=false}}
rWatch(()=>raid.disksList,list=>{if(rSelected.value){const next=list.find(item=>item.path===rSelected.value!.path)||null;if(next&&(next.members.join('|')!==rSelected.value.members.join('|')||next.status!==rSelected.value.status)){rAccepted.value=false;rTyped.value=''}rSelected.value=next}})
rWatch(rSelected,async(value,old)=>{if(value&&!old){rOverflow=document.body.style.overflow;document.body.style.overflow='hidden';await rNextTick();rDrawer.value?.focus()}else if(!value&&old){if(rOverflow!==undefined){document.body.style.overflow=rOverflow;rOverflow=undefined}await rNextTick();if(rTrigger?.isConnected)rTrigger.focus({preventScroll:true})}})
function rKey(event:KeyboardEvent){if(!rSelected.value||document.querySelector('#actioner')||!rDrawer.value)return;if(event.key==='Escape'){event.preventDefault();rClose()}if(event.key!=='Tab')return;const controls=Array.from(rDrawer.value.querySelectorAll<HTMLElement>('button:not(:disabled),input:not(:disabled),summary')).filter(el=>el.offsetParent!==null);const first=controls[0],last=controls[controls.length-1];if(event.shiftKey&&(document.activeElement===first||document.activeElement===rDrawer.value)){event.preventDefault();last?.focus()}else if(!event.shiftKey&&(document.activeElement===last||document.activeElement===rDrawer.value)){event.preventDefault();first?.focus()}}
rMounted(()=>document.addEventListener('keydown',rKey));onBeforeUnmount(()=>{document.removeEventListener('keydown',rKey);if(rOverflow!==undefined)document.body.style.overflow=rOverflow})
`

const createTemplate=`<section class="rs-form" role="dialog" aria-modal="true" aria-label="创建 RAID 阵列" tabindex="-1" ref="rForm">
 ${heading('创建 RAID 阵列','选择级别与成员，再核对本次操作的影响。')}
 <template v-if="status==='init'">${steps}<div class="rs-body">
  <template v-if="rStep===1"><div class="rs-section-heading"><h3>选择 RAID 级别</h3><span>当前：{{raidType.find(t=>t.name===selectReidType)?.title}}</span></div>
   <div class="rs-levels"><label v-for="item in raidType" :key="item.name" :class="{selected:selectReidType===item.name}"><input type="radio" name="raid-level" :value="item.name" v-model="selectReidType" :aria-label="item.title" :disabled="!!raidIssue(item.name)" /><span><strong>{{item.title}}</strong><small>至少 {{item.select}} 块硬盘</small><small v-if="raidIssue(item.name)">{{raidIssue(item.name)}}</small></span></label></div>
   <p v-if="raidCreateIssue(selectReidType)" class="sw-note warning" role="status">{{raidCreateIssue(selectReidType)}}</p><p class="rs-level-info">{{getRaidInfo(selectReidType)}}</p>
   <div class="rs-section-heading"><h3>选择硬盘成员</h3><span>已选 {{selectDisk.length}} 块 / 至少 {{rMinimum}} 块</span></div>
   <p v-if="raidConfig.loading" class="rs-empty-inline">正在检测可用硬盘…</p><div v-else-if="raidConfig.members.length" class="rs-disks"><label v-for="item in raidConfig.members" :key="item.path" :class="{selected:selectDisk.includes(item.path)}"><input type="checkbox" :value="item.path" v-model="selectDisk" :aria-label="item.name+' '+item.path" /><span class="rs-disk-icon">${diskIcon}</span><span class="rs-disk-name"><strong>{{item.model||item.name}}</strong><small>{{item.path}}</small></span><b>{{item.sizeStr}}</b></label></div><p v-else class="rs-empty-inline">没有可用的阵列成员。请先检查硬盘连接和当前用途。</p>
   <p class="rs-muted">成员由现有 RAID 服务提供。通过 USB 接入的设备不会在候选列表中显示。</p>
  </template>
  <template v-else><div class="rs-review-title"><span class="rs-disk-icon">${diskIcon}</span><div><h3>{{raidType.find(t=>t.name===selectReidType)?.title}}</h3><p>{{selectDisk.length}} 块硬盘组成新阵列</p></div></div><div class="rs-selected-list"><div v-for="item in raidConfig.members.filter(d=>selectDisk.includes(d.path))" :key="item.path"><span><strong>{{item.model}}</strong><small>{{item.path}}</small></span><b>{{item.sizeStr}}</b></div></div><div class="rs-impact"><strong>所选硬盘的所有分区将被清除</strong><p>将创建阵列，并沿用当前业务准备文件系统和挂载。其他硬盘保留。阵列实际容量以创建结果为准。</p></div><label class="rs-check"><input type="checkbox" v-model="rAccepted" />我已核对所选硬盘，理解清空范围，并确认重要文件已有备份。</label></template>
 </div><footer class="rs-footer"><button @click="rStep===2?rBack():onClose()" :disabled="disabled">{{rStep===2?'返回修改':'取消'}}</button><button v-if="rStep===1" class="primary" :disabled="!!raidCreateIssue(selectReidType)||disabled||raidConfig.loading||selectDisk.length<rMinimum" @click="rStep=2">核对配置 <span>›</span></button><button v-else class="primary" :disabled="!!raidCreateIssue(selectReidType)||disabled||!rAccepted" @click="onNext">创建阵列</button></footer></template>
 <div v-else class="rs-result"><template v-if="status==='loading'"><icon-loading :size="44" /><h3>正在创建阵列</h3><p>正在处理所选硬盘并准备文件存储，请等待操作结果。</p></template><template v-else-if="status==='success'"><span class="rs-result-icon">✓</span><h3>阵列已创建</h3><p>可以返回列表查看状态、成员和文件存储位置。</p><button class="primary" @click="onClose">查看阵列</button></template><template v-else><span class="rs-result-icon error">!</span><h3>创建未完成</h3><p role="alert">{{msg}}</p><div><button @click="status='init';rBack()">调整配置</button><button class="primary" :disabled="!!raidCreateIssue(selectReidType)||disabled" @click="onNext">重新创建</button></div></template></div>
</section>`
const createScript=`
import {computed as rComputed} from 'vue'
import {raidIssue,raidCreateIssue} from '/src/design/storageCapabilitiesPrototype'
const rStep=ref(1),rAccepted=ref(false)
const rMinimum=rComputed(()=>raidType.find(item=>item.name===selectReidType.value)?.select||2)
const confirm=(_message:string)=>!raidCreateIssue(selectReidType.value)&&rAccepted.value
function rBack(){rStep.value=1;rAccepted.value=false}
`

function memberTemplate(mode:'edit'|'remove'|'recover'){
 const title=mode==='edit'?'扩充阵列':mode==='remove'?'移除成员':'恢复成员',description=mode==='edit'?'选择添加到阵列的硬盘。':mode==='remove'?'选择从阵列中移除的成员。':'选择用于恢复阵列的硬盘。'
 const members=mode==='remove'?`<div class="rs-disks"><label v-for="item in raid.members" :key="item" :class="{selected:selectDisk===item}"><input type="radio" name="raid-member" :value="item" v-model="selectDisk" :aria-label="item" /><span class="rs-disk-icon">${diskIcon}</span><span class="rs-disk-name"><strong>{{item.replace('/dev/','')}}</strong><small>当前阵列成员</small></span></label></div>`:`<p v-if="raidConfig.loading" class="rs-empty-inline">正在检测可用硬盘…</p><div v-else-if="raidConfig.members.length" class="rs-disks"><label v-for="item in raidConfig.members" :key="item.path" :class="{selected:selectDisk===item.path}"><input type="radio" name="raid-member" :value="item.path" v-model="selectDisk" :aria-label="item.name+' '+item.path" /><span class="rs-disk-icon">${diskIcon}</span><span class="rs-disk-name"><strong>{{item.model||item.name}}</strong><small>{{item.path}}</small></span><b>{{item.sizeStr}}</b></label></div><p v-else class="rs-empty-inline">没有可用的阵列成员。</p>`
 const impact=mode==='edit'?'所选硬盘将加入阵列并投入使用。扩充阵列不等于文件系统容量已自动增加。':mode==='remove'?'所选成员将从阵列中移除。移除不是缩容，现有阵列容量不会据此自动减小。':'所选硬盘将按现有恢复流程加入阵列；成员已充足时可能成为预备盘。原有分区可能被清除，操作完成也不代表重建已结束。'
 return `<section class="rs-form" role="dialog" aria-modal="true" aria-label="${title}" tabindex="-1" ref="rForm">${heading(title,description)}<p v-if="raidIssue(raid.level)" class="sw-note warning" role="status">{{raidIssue(raid.level)}}</p>${steps}<div class="rs-body"><div class="rs-array-context"><span class="rs-disk-icon">${diskIcon}</span><div><strong>{{raid.name}}</strong><small>{{raid.path}} · {{raid.level.toUpperCase()}} · {{raid.size}}</small></div></div><template v-if="rStep===1"><div class="rs-section-heading"><h3>${description}</h3><span>单选一块硬盘</span></div>${members}</template><template v-else><dl class="rs-review-fields"><dt>操作</dt><dd>${title}</dd><dt>目标阵列</dt><dd>{{raid.path}}</dd><dt>所选硬盘</dt><dd>{{selectDisk}}</dd></dl><div class="rs-impact"><strong>请核对本次变更</strong><p>${impact}</p></div><label class="rs-check"><input type="checkbox" v-model="rAccepted" />我已核对阵列、硬盘及操作影响。</label></template></div><footer class="rs-footer"><button :disabled="disabled" @click="rStep===2?rBack():onClose()">{{rStep===2?'返回修改':'取消'}}</button><button v-if="rStep===1" class="primary" :disabled="!!raidIssue(raid.level)||disabled||!selectDisk" @click="rStep=2">核对变更 <span>›</span></button><button v-else class="primary" :disabled="!!raidIssue(raid.level)||disabled||!rAccepted" @click="onNext">${mode==='recover'?'开始恢复':mode==='remove'?'确认移除':'确认扩充'}</button></footer></section>`
}
const memberScript=`import {raidIssue} from '/src/design/storageCapabilitiesPrototype';const rStep=ref(1),rAccepted=ref(false);function rBack(){rStep.value=1;rAccepted.value=false}`
const infoTemplate=`<section class="rs-form" role="dialog" aria-modal="true" aria-label="RAID 技术详情" tabindex="-1" ref="rForm">${heading('技术详情','查看服务返回的原始设备信息。')}<div class="rs-body"><p class="rs-muted">{{raid.name}} · {{raid.path}}</p><textarea class="rs-detail-text" readonly :value="info" aria-label="RAID 原始详情"></textarea></div><footer class="rs-footer"><button @click="onClose">关闭</button></footer></section>`
const actionTemplate=`<action-component :type="2"><div class="action-main raid-action-host"><setup-create :Close="Close" :success="success" v-if="setup==='create'"/><setup-info v-else-if="setup==='info'&&raid" :Close="Close" :raid="raid"/><setup-edit v-else-if="setup==='edit'&&raid" :Close="Close" :raid="raid" :success="success"/><setup-remove v-else-if="setup==='remove'&&raid" :Close="Close" :raid="raid" :success="success"/><setup-recover v-else-if="setup==='recover'&&raid" :Close="Close" :raid="raid" :success="success"/></div></action-component>`
const confirmationTemplate=`<action-component :Close="Close" :type="1"><section class="rs-confirm" role="dialog" aria-modal="true" aria-label="RAID 操作确认"><span class="rs-kicker">RAID / 操作确认</span><h2>{{title||'确认本次操作'}}</h2><p>{{content}}</p><footer class="rs-footer"><button v-if="clear" @click="onClear">{{clearTitle||'取消'}}</button><button class="primary" @click="onNext">{{nextTitle||'确认并继续'}}</button><button v-if="continuer" @click="onContinue">{{continuerTitle||'继续保存'}}</button></footer></section></action-component>`

function formScript(busy='disabled.value'){return `
import {onMounted as rFormMounted,onBeforeUnmount as rFormUnmounted,nextTick as rFormTick} from 'vue'
const rForm=ref<HTMLElement>();let rFormTrigger:HTMLElement|null=null
function rFormKey(event:KeyboardEvent){if(!rForm.value)return;if(event.key==='Escape'){event.preventDefault();event.stopPropagation();if(!${busy})onClose();return}if(event.key!=='Tab')return;const controls=Array.from(rForm.value.querySelectorAll<HTMLElement>('button:not(:disabled),input:not(:disabled),textarea,summary')).filter(el=>el.offsetParent!==null);const first=controls[0],last=controls[controls.length-1];if(event.shiftKey&&(document.activeElement===first||document.activeElement===rForm.value)){event.preventDefault();last?.focus()}else if(!event.shiftKey&&(document.activeElement===last||document.activeElement===rForm.value)){event.preventDefault();first?.focus()}}
rFormMounted(async()=>{rFormTrigger=document.activeElement as HTMLElement;await rFormTick();rForm.value?.focus();document.addEventListener('keydown',rFormKey)});rFormUnmounted(()=>{document.removeEventListener('keydown',rFormKey);if(rFormTrigger?.isConnected)rFormTrigger.focus({preventScroll:true})})
`}

const presentations:Record<string,{template:string;script?:string}>={
 'pages/raid/index/index.vue':{template:pageTemplate,script:pageScript},
 'components/action-raid/index.vue':{template:actionTemplate},
 'components/action-raid/setup-create.vue':{template:createTemplate,script:createScript+formScript()},
 'components/action-raid/setup-edit.vue':{template:memberTemplate('edit'),script:memberScript+formScript()},
 'components/action-raid/setup-remove.vue':{template:memberTemplate('remove'),script:memberScript+formScript()},
 'components/action-raid/setup-recover.vue':{template:memberTemplate('recover'),script:memberScript+formScript()},
 'components/action-raid/setup-info.vue':{template:infoTemplate,script:formScript('false')},
 'components/action-confirm/index.vue':{template:confirmationTemplate},
}
export function raidPresentation(code:string,relative:string){
 const presentation=presentations[relative];if(!presentation)return
 const scriptStart=code.indexOf('<script'),templateEnd=code.lastIndexOf('</template>',scriptStart)
 let output='<template>'+presentation.template+'</template>'+code.slice(templateEnd+11)
 if(presentation.script)output=output.replace('</script>',presentation.script+'\n</script>')
 return output.replace(/<style\b[^>]*>[\s\S]*?<\/style>/g,'')
}
