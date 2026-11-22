<!-- THROWAWAY: disk-map / per-disk / usage-first variants on /storage. All changes are memory-only. -->
<template>
 <main class="sw" data-storage-workbench>
  <div class="sw-page" :inert="drawerOpen||!!modal">
  <div class="sw-demo"><span>产品交互稿 · 示例设备 · 不操作真实硬盘</span><label>场景<select aria-label="存储演示场景" :value="workbench.scene" :disabled="workbench.busy" @change="changeScene"><option value="normal">多盘日常使用</option><option value="raid">已有 RAID 镜像</option><option value="degraded">RAID 成员失联</option><option value="raid-unmounted">RAID 未挂载</option><option value="raid-resyncing">RAID 重同步中</option><option value="raid-missing">扫描恢复 RAID</option><option value="thin-full">应用池空间紧张</option></select></label><label><input type="checkbox" v-model="workbench.failNext" :disabled="workbench.busy" />演示下一项执行失败</label><button :disabled="workbench.busy" @click="reset">重新演示</button><ThemeSelector /></div>
  <nav class="sw-breadcrumb"><router-link to="/">首页</router-link><span>›</span><span>硬盘与存储</span></nav>
  <header class="sw-heading"><div><span class="sw-eyebrow">QUICKSTART / STORAGE</span><h1>硬盘与存储</h1><p>看清每块硬盘上的空间，保留现有文件，再按需创建分区、存储池或阵列。</p></div><button @click="tab='jobs'">任务记录 <small>{{workbench.jobs.length}}</small></button></header>
  <div class="sw-metrics"><div><span>物理硬盘</span><strong>{{workbench.drives.length}} <small>块</small></strong></div><div><span>普通文件存储</span><strong>{{fileParts.length}} <small>处</small></strong></div><div><span>存储池 / RAID</span><strong>{{workbench.pools.length}} <small>/ {{workbench.raids.length}}</small></strong></div><div :class="{alert:alerts.length}"><span>需要关注</span><strong>{{alerts.length}} <small>项</small></strong></div></div>
  <button v-for="alert in alerts" :key="alert.message" class="sw-alert" @click="tab=alert.tab"><span>!</span><strong>{{alert.message}}</strong><span>查看处理 ›</span></button>
  <nav class="sw-tabs" aria-label="存储工作区"><button v-for="item in tabs" :key="item.id" :class="{active:tab===item.id}" @click="tab=item.id">{{item.name}}</button></nav>
  <template v-if="tab==='disks'">
   <header class="sw-section-title"><div><h2>{{variant==='C'?'按用途查看存储':'全部硬盘与分区'}}</h2><p>布局变更先加入待执行列表，应用前可以撤销。</p></div><div class="sw-legend"><span v-for="kind in legend" :key="kind.kind"><i :class="kind.kind"></i>{{kind.label}}</span></div></header>
   <div v-if="variant==='C'" class="sw-usage-view"><section><h3>普通文件 · 沿用 EXT4</h3><button v-for="item in fileParts" :key="item.part.id" @click="select(item.part.id);variantTo('A')"><strong>{{item.part.name}}</strong><span>{{item.part.id}} · {{item.part.size}} GiB · {{item.part.mount?'已挂载':'未挂载'}}</span></button><p>直接访问和共享，不必创建应用卷。</p></section><section><h3>主机与应用存储</h3><button v-for="pool in workbench.pools" :key="pool.id" @click="poolId=pool.id;tab='pools'"><strong>{{pool.name}}</strong><span>{{pool.source}} → {{pool.vg}} → {{pool.volumes.length}} 个卷</span></button><p>主机磁盘由 LightHost 管理；应用卷由系统模块管理。</p></section><section><h3>可安排的空闲空间</h3><button v-for="item in freeParts" :key="item.part.id" @click="select(item.part.id);variantTo('A')"><strong>{{item.disk.id}} · {{item.part.size}} GiB</strong><span>未分配 · 可以创建分区或设置用途</span></button></section></div>
   <div v-else :class="['sw-workspace',{focused:variant==='B'}]">
    <aside v-if="variant==='B'" class="sw-drive-nav"><button v-for="drive in workbench.drives" :key="drive.id" :class="{active:selected?.disk.id===drive.id}" @click="select(drive.parts[0].id)"><strong>{{drive.id.toUpperCase()}} · {{drive.size}} GiB</strong><small>{{drive.model}} · {{drive.connected?'在线':'未连接'}}</small></button></aside>
    <section><StorageDiskMapPrototype :drives="variant==='B'&&selected?[selected.disk]:workbench.drives" :selected="partId" @select="select" /></section>

   </div>
   <section v-if="arrayDrives.length" class="sw-array-files"><h3>阵列文件存储</h3><p class="sw-note">阵列作为一份逻辑设备使用；挂载状态与 RAID 管理同步。</p><StorageDiskMapPrototype :drives="arrayDrives" :selected="partId" @select="select" /></section>
  </template>
  <template v-else-if="tab==='pools'">
   <header class="sw-section-title"><div><h2>存储池与卷</h2><p>从物理来源找到每个卷，分别查看容量上限、实际占用和管理归属。</p></div></header>
   <div class="sw-pool-layout"><aside class="sw-pool-nav"><button v-for="pool in workbench.pools" :key="pool.id" :class="{active:activePool?.id===pool.id}" @click="poolId=pool.id"><strong>{{pool.name}}</strong><small>{{pool.owner}} 管理 · {{pool.size}} GiB</small></button></aside><section v-if="activePool" class="sw-pool-detail"><header><div><span class="sw-eyebrow">{{activePool.owner}} 管理</span><h2>{{activePool.name}}</h2><button class="link" @click="select(activePool!.source);tab='disks'">来源：{{activePool.source}} › 查看硬盘位置</button></div><button v-if="activePool.owner==='系统'" class="primary" :disabled="poolTight||workbench.busy" @click="open('volume')">创建应用卷</button><button v-else @click="notify('主机和附加磁盘的创建、删除与恢复应在 LightHost 中完成；通用页面不直接修改 Incus 的卷。')">在 LightHost 中管理</button></header>
    <p v-if="poolTight" class="sw-note warning">数据或元数据空间紧张，暂不接收新分配。先清理由对应应用管理的恢复点，或评估扩容。</p>
    <div class="sw-pool-stats"><div><span>实际数据占用</span><strong>{{activePool.used}} / {{activePool.size}} GiB</strong><progress :value="activePool.used" :max="activePool.size"></progress></div><div><span>元数据占用</span><strong>{{activePool.metadata}}%</strong><progress :value="activePool.metadata" max="100"></progress></div><div><span>卷容量合计 / 尚可分配</span><strong>{{allocated}} / {{activePool.size-allocated}} GiB</strong><small>演示采用不超额分配</small></div></div>
    <p class="sw-note">底层区域另留约 {{activePool.reserve}} GiB 用于元数据与余量。实际占用包含卷写入和快照影响，不能用卷容量合计代替。</p>
    <div class="sw-table-wrap"><table><thead><tr><th>卷 / 用途</th><th>容量上限</th><th>实际占用</th><th>使用状态</th><th>操作</th></tr></thead><tbody><tr v-for="volume in activePool.volumes" :key="volume.id"><td><strong>{{volume.name}}</strong><small>{{volume.kind}} · {{volume.snapshots}} 个恢复点</small></td><td>{{volume.size}} GiB</td><td>{{volume.used}} GiB</td><td>{{volume.attached?'应用使用中':'未接入应用'}}</td><td><button :disabled="activePool.owner!=='系统'||poolTight||workbench.busy" @click="volumeId=volume.id;open('grow')">扩容</button><button :disabled="activePool.owner!=='系统'||volume.attached||!!volume.snapshots||workbench.busy" @click="volumeId=volume.id;open('deleteVolume')">删除</button></td></tr></tbody></table></div><p class="sw-note">使用中的卷及带恢复点的卷禁止直接删除。Docker 卷创建后还需要停机、迁移和接入；沙箱卷待启动与恢复验证后开放。</p><details><summary>技术信息与管理关系</summary><p>{{activePool.source}} → PV → {{activePool.vg}} → thin-pool → 逻辑卷。LightHost 与系统应用使用不同 VG，由各自管理者操作。</p></details>
   </section></div>
  </template>
  <section v-else-if="tab==='raid'" class="sw-raid-surface" data-native-raid>
   <ExistingRaidManagement :key="workbench.scene" />
  </section>
  <template v-else><header class="sw-section-title"><div><h2>任务记录</h2><p>逐项查看结果。失败会停止后续操作，已完成项目不会自动回滚。</p></div></header><div v-if="!workbench.jobs.length" class="sw-empty"><h2>还没有执行任务</h2><p>新建、删除等操作会先进入待执行列表。</p></div><article v-for="job in workbench.jobs" :key="job.id" class="sw-job"><span :class="job.status==='失败'?'sw-danger-text':'sw-ok'">{{job.status}}</span><div><strong>{{job.title}} · {{job.target}}</strong><p>{{job.detail}}</p></div></article></template>

  <aside v-if="workbench.changes.length" class="sw-pending"><div><strong>{{workbench.changes.length}} 项待执行操作</strong><small>尚未修改示例布局，可以撤销</small></div><button @click="workbench.changes=[]">全部撤销</button><button class="primary" @click="open('review')">检查并应用</button></aside>
  </div>
  <Transition name="sw-detail">
   <div v-if="drawerOpen&&selected" class="sw-drawer-shade" @click.self="closeDrawer" :inert="!!modal" :aria-hidden="modal?true:undefined">
    <aside class="sw-inspector" role="dialog" aria-modal="true" aria-labelledby="sw-detail-title" tabindex="-1" ref="detailDialog" data-storage-detail>
     <header class="sw-drawer-header"><span>分区与空间详情</span><button aria-label="关闭存储详情" @click="closeDrawer">×</button></header>
     <div class="sw-inspector-scroll"><span class="sw-eyebrow">当前选择 · {{selected.disk.id.toUpperCase()}}</span><h2 id="sw-detail-title">{{selected.part.name}}</h2><p class="sw-selection-id">{{selected.part.id}} · {{selected.part.size}} GiB</p><span class="sw-status" :class="selected.part.kind">{{kindLabel(selected.part.kind)}}</span><dl><dt>文件系统</dt><dd>{{selected.part.fs}}</dd><dt>实际占用</dt><dd>{{selected.part.kind==='files'?selected.part.used+' GiB':'由对应管理者统计'}}</dd><dt>挂载位置</dt><dd>{{selected.part.mount||'—'}}</dd><dt>管理归属</dt><dd>{{selected.part.owner||'普通文件 / 系统磁盘管理'}}</dd><dt>硬盘身份</dt><dd>{{selected.disk.serial}}</dd></dl>
     <p v-if="block" class="sw-note warning">{{block}}</p>
     <template v-if="selected.part.kind==='free'"><button class="primary" :disabled="workbench.busy||!selected.disk.connected" @click="open('partition')">创建分区</button><button :disabled="workbench.busy||!selected.disk.connected" @click="openSetup">设置存储用途</button><p class="sw-note">普通 EXT4 为默认选项，也可划给 LightHost 或应用存储。</p></template>
     <template v-else-if="selected.part.kind==='files'"><button class="primary" :disabled="workbench.busy" @click="mountPart(selected.part.id)">{{selected.part.mount?'卸载（演示）':'挂载并使用'}}</button><button v-if="selected.part.mount" @click="notify('演示：打开 '+selected!.part.mount+'，也可沿用原有共享和 Docker 目录设置。')">打开文件</button><div class="sw-secondary-actions"><button :disabled="!!block||workbench.busy" @click="open('format')">格式化</button><button class="danger" :disabled="!!block||workbench.busy" @click="open('delete')">删除分区</button></div><button :disabled="!!block||workbench.busy" @click="openSetup">重新规划用途</button></template>
     <button v-else-if="selected.part.kind==='hosts'||selected.part.kind==='apps'" class="primary" @click="showPoolSource(selected.part.id)">查看关联存储池</button><button v-else-if="selected.part.kind==='raid'" class="primary" @click="tab='raid'">查看 RAID 成员关系</button><button v-else disabled>系统分区受保护</button>
     <details><summary>理解分区与卷</summary><p>分区位于硬盘上；LVM 池中的逻辑卷位于另一层。删除分区会影响它上面的文件系统或全部卷。</p><p>调整已有分区大小、在线缩小文件系统不在本轮演示范围。</p></details>
    </div>
     <footer class="sw-drawer-footer"><button @click="closeDrawer">返回硬盘列表</button></footer>
    </aside>
   </div>
  </Transition>
  <p v-if="workbench.busy" class="sw-feedback">正在执行演示任务，请等待结果。</p><p v-else-if="workbench.feedback" class="sw-feedback" role="status">{{workbench.feedback}} <button aria-label="关闭提示" @click="workbench.feedback=''">×</button></p>

  <div v-if="modal" class="sw-shade" @click.self="close"><section :class="['sw-dialog',{wide:modal==='setup'}]" role="dialog" aria-modal="true" :aria-label="modalTitle" tabindex="-1" ref="dialog"><header><span>{{modalTitle}}</span><button aria-label="关闭操作" :disabled="setupBusy" @click="close">×</button></header>
   <div v-if="modal==='setup'&&setupTarget" class="lh-prototype sw-setup-content"><StorageSetupPrototype :target="setupTarget" :scene="setupScene" initial-purpose="files" :plan-only="true" @busy="setupBusy=$event" @complete="setupComplete" @close="close" @files="close" @volumes="close();tab='pools'" @continue="close" /></div>
   <template v-else-if="modal==='partition'&&selected"><h2>在未分配空间中创建分区</h2><p>{{selected.disk.id}} · 可用 {{selected.part.size}} GiB · 其他分区保留</p><label>分区名称<input v-model="name" aria-label="分区名称" placeholder="例如：共享文件" /></label><label>容量<input v-model.number="size" type="number" min="1" :max="selected.part.size" aria-label="分区容量" />GiB</label><div class="sw-plan-mini"><span :style="{flex:size}">新建 EXT4 {{size}} GiB</span><span :style="{flex:Math.max(0,selected.part.size-size)}">剩余 {{Math.max(0,selected.part.size-size)}} GiB</span></div><p class="sw-note">普通文件存储直接使用 EXT4。也可以返回“设置存储用途”，安排主机、应用及普通文件区。</p><footer><button @click="close">取消</button><button class="primary" :disabled="!validPartition" @click="submitPartition">加入待执行</button></footer></template>
   <template v-else-if="modal==='volume'||modal==='grow'"><h2>{{modal==='grow'?'扩容应用卷':'创建应用卷'}}</h2><p>{{activePool?.name}} · 尚可分配 {{activePool?activePool.size-allocated:0}} GiB</p><label v-if="modal==='volume'">用途<select v-model="volumeKind" aria-label="应用卷用途"><option>Docker 数据卷</option><option>LightHost 管理卷</option><option disabled>系统沙箱卷 · 待支持</option></select></label><label v-if="modal==='volume'">卷名称<input v-model="name" aria-label="卷名称" /></label><label>{{modal==='grow'?'扩容后的总容量':'容量上限'}}<input v-model.number="size" type="number" min="1" aria-label="卷容量" />GiB</label><p v-if="modal==='grow'" class="sw-note">当前 {{activeVolume?.size}} GiB。只支持增加容量，文件系统扩展也需要验证。</p><p class="sw-note">创建的是独立逻辑卷，内部使用 EXT4 保存文件；不会自动迁移已有应用数据。</p><footer><button @click="close">取消</button><button class="primary" :disabled="!validVolume" @click="submitVolume">加入待执行</button></footer></template>
   <template v-else-if="modal==='review'"><h2>检查并应用 {{workbench.changes.length}} 项操作</h2><p>以下是实际执行顺序。应用前会再次核对设备身份、占用和容量；这不是可自动回滚的事务。</p><ol class="sw-change-list"><li v-for="change in workbench.changes" :key="change.id"><strong>{{change.title}} · {{change.target}}</strong><p>{{change.impact}}</p><button @click="workbench.changes=workbench.changes.filter(x=>x.id!==change.id)">撤销此项</button></li></ol><p v-if="tokens.length" class="sw-note warning">含清空或删除操作。请核对目标与备份后继续。</p><label v-if="tokens.length">输入 {{tokens.join(', ')}} 确认目标<input v-model="typed" aria-label="输入受影响目标确认" /></label><label class="sw-check"><input type="checkbox" v-model="ack" />我已核对目标、影响范围和必要的备份。</label><footer><button @click="close">返回检查</button><button class="primary" :disabled="!canApply" @click="apply">应用操作（演示）</button></footer></template>
   <template v-else><h2>{{modalTitle}}</h2><p class="sw-note warning">{{deleteMessage}}</p><p>这一步只加入计划；最终应用时还需核对目标和清空范围。</p><footer><button @click="close">取消</button><button class="danger" @click="submitDelete">加入待执行</button></footer></template>
  </section></div>
  <PrototypeSwitcher v-if="!drawerOpen&&!modal" :variants="variants" />
 </main>
</template>
<script setup lang="ts">
import {computed,nextTick,onMounted,onUnmounted,ref,watch} from 'vue'
import {useRoute,useRouter} from 'vue-router'
import ThemeSelector from '../home/ThemeSelector.vue'
import PrototypeSwitcher from './PrototypeSwitcher.vue'
import StorageDiskMapPrototype from './StorageDiskMapPrototype.vue'
// Reuse shipped RAID business handlers and actions; styling and transport are design-only.
import ExistingRaidManagement from '../pages/raid/index/index.vue'
import './raidWorkbenchPreview.css'
import StorageSetupPrototype from './StorageSetupPrototype.vue'
import type {StoragePlan,StorageTarget} from './storagePurposePrototype'
import {workbench,findPart,reason,queueCreate,queueDelete,queueFormat,mountPart,queueSetup,queueVolume,queueGrow,queueDeleteVolume,applyChanges,resetWorkbench,raidDrive,type PartKind} from './storageWorkbenchPrototype'
const route=useRoute(),router=useRouter(),tab=ref(String(route.query.tab||'disks')),partId=ref(String(route.query.part||'sdb-free')),poolId=ref('apps'),volumeId=ref('')
if(route.query.scene&&['raid','degraded','raid-unmounted','raid-resyncing','raid-missing','thin-full'].includes(String(route.query.scene)))resetWorkbench(String(route.query.scene));
const drawerOpen=ref(false),detailDialog=ref<HTMLElement>()
let detailTrigger:HTMLElement|null=null,previousOverflow:string|undefined
const variant=computed(()=>['A','B','C'].includes(String(route.query.variant))?String(route.query.variant):'A'),variants=[{key:'A',name:'多盘布局（推荐）'},{key:'B',name:'逐盘管理'},{key:'C',name:'按用途查看'}]
const tabs=[{id:'disks',name:'硬盘与分区'},{id:'pools',name:'存储池与卷'},{id:'raid',name:'RAID 阵列'},{id:'jobs',name:'任务记录'}]
const legend=[{kind:'files',label:'普通文件'},{kind:'hosts',label:'主机存储'},{kind:'apps',label:'应用存储'},{kind:'raid',label:'RAID 成员'},{kind:'free',label:'未分配'}]
const selected=computed(()=>findPart(partId.value)),block=computed(()=>selected.value?reason(selected.value.part):''),fileParts=computed(()=>workbench.drives.flatMap(disk=>disk.parts.filter(p=>p.kind==='files').map(part=>({disk,part})))),freeParts=computed(()=>workbench.drives.flatMap(disk=>disk.parts.filter(p=>p.kind==='free').map(part=>({disk,part}))))
const arrayDrives=computed(()=>workbench.raids.map(raidDrive))

const activePool=computed(()=>workbench.pools.find(p=>p.id===poolId.value)||workbench.pools[0]),activeVolume=computed(()=>activePool.value?.volumes.find(v=>v.id===volumeId.value))
const allocated=computed(()=>activePool.value?.volumes.reduce((s,v)=>s+v.size,0)||0),poolTight=computed(()=>!!activePool.value&&(activePool.value.used/activePool.value.size>=.9||activePool.value.metadata>=90))
const alerts=computed(()=>[...workbench.raids.filter(r=>r.status!=='正常').map(r=>({message:r.name+'：'+r.status,tab:'raid'})),...workbench.pools.filter(p=>p.used/p.size>=.9||p.metadata>=90).map(p=>({message:p.name+'：数据或元数据空间紧张',tab:'pools'}))])
type Modal='partition'|'volume'|'grow'|'delete'|'format'|'deleteVolume'|'review'|'setup'|''
const modal=ref<Modal>(''),name=ref(''),size=ref(16),volumeKind=ref('Docker 数据卷'),typed=ref(''),ack=ref(false),dialog=ref<HTMLElement>(),setupBusy=ref(false),setupTarget=ref<StorageTarget>()
const setupScene=computed(()=>selected.value?.part.kind==='files'?'ordinary':'new')
const modalTitle=computed(()=>({partition:'创建分区',volume:'创建应用卷',grow:'扩容应用卷',delete:'删除分区',format:'格式化分区',deleteVolume:'删除应用卷',review:'应用待执行操作',setup:'设置存储用途','':''}[modal.value]))
const deleteMessage=computed(()=>modal.value==='deleteVolume'?`删除「${activeVolume.value?.name}」及其中的全部数据。`:`${modal.value==='format'?'格式化':'删除'} ${selected.value?.part.id}「${selected.value?.part.name}」，其中的文件将无法正常访问。`)
const tokens=computed(()=>[...new Set(workbench.changes.map(c=>c.token).filter((t):t is string=>!!t))]),canApply=computed(()=>!!workbench.changes.length&&ack.value&&(!tokens.value.length||typed.value===tokens.value.join(', ')))
const validPartition=computed(()=>!!selected.value&&Number.isFinite(size.value)&&size.value>=1&&size.value<=selected.value.part.size),validVolume=computed(()=>!!activePool.value&&Number.isFinite(size.value)&&size.value>=1&&(modal.value==='grow'?!!activeVolume.value&&size.value>activeVolume.value.size&&size.value-activeVolume.value.size<=activePool.value.size-allocated.value:size.value<=activePool.value.size-allocated.value))
const drive=(id:string)=>workbench.drives.find(d=>d.id===id),kindLabel=(kind:PartKind)=>({system:'系统保护',files:'普通文件存储',free:'未分配',hosts:'LightHost 管理',apps:'系统应用存储',raid:'RAID 成员',raw:'未格式化'}[kind])
function notify(message:string){workbench.feedback=message}
function select(id:string){const disk=drive(id);partId.value=disk?disk.parts[0].id:id;if(!selected.value)return;if(!drawerOpen.value)detailTrigger=document.activeElement as HTMLElement;drawerOpen.value=true}
function closeDrawer(){drawerOpen.value=false}
function variantTo(value:string){router.replace({query:{...route.query,variant:value}})}
function changeScene(e:Event){resetWorkbench((e.target as HTMLSelectElement).value);partId.value='sdb-free'}
function reset(){resetWorkbench(workbench.scene);partId.value='sdb-free';modal.value=''}
function open(kind:Modal){if(workbench.busy)return;name.value='';size.value=kind==='grow'?(activeVolume.value?.size||0)+4:16;typed.value='';ack.value=false;modal.value=kind}
function close(){if(!setupBusy.value)modal.value=''}
function finishPlanning(){close();closeDrawer()}
function openSetup(){if(!selected.value||block.value)return;const p=selected.value.part;setupTarget.value={id:p.id,label:selected.value.disk.id+' / '+p.name,capacity:p.size,kind:p.kind==='free'?'free':'partition',token:p.id,effect:p.kind==='free'?'仅使用所选未分配范围，新建分区':'清空并重新规划所选分区覆盖范围',preserved:'同盘其他分区及其他硬盘保留'};open('setup')}
function setupComplete(plan:StoragePlan){if(!setupTarget.value)return;const id=setupTarget.value.id;modal.value='';setupBusy.value=false;if(queueSetup(id,plan))closeDrawer()}
function showPoolSource(id:string){const p=workbench.pools.find(x=>x.source===id);if(p)poolId.value=p.id;tab.value='pools'}
function submitPartition(){if(validPartition.value&&queueCreate(partId.value,name.value,size.value))finishPlanning()}
function submitVolume(){const p=activePool.value;if(!p||!validVolume.value)return;const ok=modal.value==='grow'&&activeVolume.value?queueGrow(p,activeVolume.value,size.value):queueVolume(p,name.value,size.value,volumeKind.value);if(ok)finishPlanning()}
function submitDelete(){let ok=false;if(modal.value==='delete')ok=queueDelete(partId.value);else if(modal.value==='format')ok=queueFormat(partId.value);else if(modal.value==='deleteVolume'&&activePool.value&&activeVolume.value)ok=queueDeleteVolume(activePool.value,activeVolume.value);if(ok)finishPlanning()}
async function apply(){if(!canApply.value)return;modal.value='';tab.value='jobs';await applyChanges()}
function keydown(e:KeyboardEvent){
 const root=modal.value?dialog.value:drawerOpen.value?detailDialog.value:undefined
 if(!root)return
 if(e.key==='Escape'){e.preventDefault();if(modal.value)close();else closeDrawer();return}
 if(e.key!=='Tab')return
 const controls=Array.from(root.querySelectorAll<HTMLElement>('button:not(:disabled),input:not(:disabled),select:not(:disabled),summary,a[href]')).filter(el=>el.offsetParent!==null)
 const first=controls[0],last=controls[controls.length-1]
 if(!first){e.preventDefault();root.focus();return}
 if(e.shiftKey&&(document.activeElement===first||document.activeElement===root||!root.contains(document.activeElement))){e.preventDefault();last.focus()}
 else if(!e.shiftKey&&(document.activeElement===last||document.activeElement===root||!root.contains(document.activeElement))){e.preventDefault();first.focus()}
}
watch(modal,async value=>{await nextTick();if(value)dialog.value?.focus();else if(drawerOpen.value)detailDialog.value?.focus()})
watch(drawerOpen,async value=>{await nextTick();if(value)detailDialog.value?.focus();else if(detailTrigger?.isConnected)detailTrigger.focus({preventScroll:true})})
watch(()=>drawerOpen.value||!!modal.value,open=>{if(open){previousOverflow=document.body.style.overflow;document.body.style.overflow='hidden'}else if(previousOverflow!==undefined){document.body.style.overflow=previousOverflow;previousOverflow=undefined}})
watch(tab,value=>{if(value!=='disks')closeDrawer()})
watch(()=>route.query.tab,value=>{if(tabs.some(t=>t.id===value))tab.value=String(value)})
function locateStorage(event:Event){const detail=(event as CustomEvent).detail;if(detail?.part){tab.value='disks';select(String(detail.part))}}
onMounted(()=>{window.addEventListener('quickstart-storage-open',locateStorage);if(route.query.part)select(String(route.query.part));document.addEventListener('keydown',keydown)})
onUnmounted(()=>{window.removeEventListener('quickstart-storage-open',locateStorage);document.removeEventListener('keydown',keydown);if(previousOverflow!==undefined)document.body.style.overflow=previousOverflow})
</script>
<style src="./storageWorkbenchPrototype.css"></style>
