<!-- Throwaway preview: plain EXT4 and preserving existing files are primary paths. -->
<template>
  <section class="lh-flow" data-storage-setup>
    <nav class="sp-step-strip" aria-label="设置步骤"><span v-for="(name,index) in steps" :key="name" :class="{active:stage===index+1,done:stage>index+1}"><b>{{stage>index+1?'✓':index+1}}</b>{{name}}</span></nav>
    <template v-if="stage===1">
      <h2 id="lh-flow-title">{{hasExistingFiles?'如何使用这块已有存储？':'这块空间准备怎么用？'}}</h2><p class="lh-flow-description">{{target.label}} · {{target.capacity}} GiB</p>
      <div class="lh-choices">
        <button class="lh-purpose" :class="{selected:mode==='files'}" :aria-pressed="mode==='files'" @click="mode='files'"><span class="lh-choice-icon" aria-hidden="true">▤</span><strong>{{hasExistingFiles?'保留文件并继续使用':'普通文件存储'}}</strong><small>{{hasExistingFiles?'继续使用现有 EXT4，不清空文件，不改变分区。':'用于文件共享、下载和应用数据目录，沿用原有磁盘用法。'}}</small><span class="lh-choice-detail">{{hasExistingFiles?'推荐 · 只处理挂载与访问':'EXT4 · 不需要应用存储组件'}}</span></button>
        <button class="lh-purpose" :class="{selected:mode==='custom'}" :aria-pressed="mode==='custom'" @click="mode='custom'"><span class="lh-choice-icon" aria-hidden="true">＋</span><strong>{{hasExistingFiles?'重新规划这块空间':'按用途划分空间'}}</strong><small>{{hasExistingFiles?'清空后，重新安排普通文件、LightHost 主机或应用存储。':'运行 LightHost、创建应用卷，也可以留一部分普通文件存储。'}}</small><span class="lh-choice-detail">{{hasExistingFiles?'会清空所选空间 · 请先备份':'按需设置 · 下一步分配容量'}}</span></button>
      </div>
      <template v-if="mode==='custom'"><h3 class="sp-subheading">需要哪些存储能力？</h3><div class="lh-choices sp-subchoices"><button v-for="item in purposes" :key="item.id" class="lh-purpose" :class="{selected:purpose===item.id}" :aria-pressed="purpose===item.id" @click="purpose=item.id"><strong>{{item.title}}</strong><small>{{item.description}}</small></button></div><p v-if="hasExistingFiles" class="lh-notice danger">这不是无损升级。所选空间中的现有文件会被清空；本流程不自动缩小已有文件系统。</p><p v-if="scene==='unsupported'" class="lh-notice">当前缺少应用存储组件。仍可以选择上面的普通文件存储。</p><details class="sp-technical"><summary>了解存储方式</summary><p>普通文件区直接使用 EXT4。LightHost 和系统应用分别使用独立 VG 与 thin-pool，应用卷内部也可以使用 EXT4。</p></details></template>
      <p v-else class="lh-notice">普通 EXT4 仍可用于共享文件、下载，以及兼容的应用数据目录；它本身不提供 LVM 快照。</p>
      <footer class="lh-flow-actions"><button @click="$emit('close')">取消</button><button class="primary" :disabled="mode==='custom'&&scene==='unsupported'" @click="stage=2">{{keepFiles?'查看挂载设置':mode==='files'?'查看文件存储方案':'分配容量'}}</button></footer>
    </template>
    <template v-else-if="stage===2">
      <h2 id="lh-flow-title">{{keepFiles?'保留现有文件存储':mode==='files'?'准备普通文件存储':'给不同用途分配空间'}}</h2><p class="lh-flow-description">{{target.label}} · {{target.capacity}} GiB</p>
      <template v-if="keepFiles"><dl class="lh-facts"><dt>文件系统</dt><dd>EXT4 · 已有工作文件约 24 GiB（示例）</dd><dt>本次操作</dt><dd>挂载并使用，不格式化，不改变分区</dd></dl><label class="lh-confirm-label">挂载目录<input v-model="mountPath" aria-label="挂载目录" /></label><p class="lh-notice">现有文件将保留。挂载目录须位于 /mnt 下，不能覆盖已有挂载点。</p></template>
      <template v-else>
        <template v-if="mode==='custom'">
          <section class="sp-file-reserve"><label class="lh-check"><input v-model="reserveFiles" type="checkbox" /><span><strong>也保留普通文件存储</strong><small>划出独立 EXT4 区，用于共享文件、下载和普通应用目录。</small></span></label><label v-if="reserveFiles" class="sp-volume-size">普通文件区<input v-model.number="fileShare" type="number" min="8" :max="maxFiles" aria-label="普通文件区容量" />GiB</label><p v-if="reserveFiles" class="lh-footnote">这里是重新划分后创建的普通文件区，不会自动保留原来的文件。</p></section>
          <template v-if="purpose==='both'"><div class="sp-split-values"><label>LightHost 主机存储<strong>{{hostShare}} <small>GiB</small></strong></label><label>应用存储<strong>{{managedCapacity-hostShare}} <small>GiB</small></strong></label></div><input v-model.number="hostShare" class="sp-range" type="range" min="24" :max="managedCapacity-24" step="1" aria-label="主机存储容量" /></template>
        </template>
        <div class="lh-layout-diagram"><div v-if="plan.files" class="files" :style="{flex:plan.files}"><strong>普通文件存储 · {{plan.files}} GiB</strong><span>EXT4 · 文件共享、下载、应用目录</span></div><div v-if="plan.hosts" class="pool" :style="{flex:plan.hosts}"><strong>主机存储 · {{plan.hosts}} GiB</strong><span>LightHost 主机磁盘与恢复点</span></div><div v-if="plan.apps" class="data" :style="{flex:plan.apps}"><strong>应用存储 · {{plan.apps}} GiB</strong><span>Docker 数据卷、管理卷等</span></div><div v-if="plan.management==='separate'" class="files" :style="{flex:plan.managementSize}"><strong>管理文件区 · {{plan.managementSize}} GiB</strong><span>LightHost 运行文件与数据库</span></div></div>
        <section v-if="mode==='custom'&&purpose==='hosts'" class="sp-management"><h3>LightHost 管理文件的位置</h3><label><input type="radio" v-model="management" value="existing" :disabled="scene==='no-management'" />使用已有管理存储 <small>{{scene==='no-management'?'未找到合适位置':'现有管理位置（示例）'}}</small></label><label v-if="reserveFiles"><input type="radio" v-model="management" value="files" />使用本次普通文件区中的独立目录</label><label><input type="radio" v-model="management" value="separate" />另留 {{managementSize}} GiB 管理文件区</label></section>
        <p v-if="mode==='custom'&&purpose==='both'" class="lh-notice">应用存储中将准备 {{managementSize}} GiB 的 LightHost 管理卷。普通文件区不会因此变成应用卷。</p>
        <p v-if="mode==='files'" class="lh-notice">完成后可直接挂载并使用，不必先创建应用卷。Docker 也可以通过原有设置选择这个目录，LVM 快照不覆盖此区域。</p>
        <dl class="lh-facts"><dt>本次影响</dt><dd>{{target.effect}}</dd><dt>保留内容</dt><dd>{{target.preserved}}</dd></dl>
        <p class="lh-footnote">容量均为演示值。正式预检需核算系统管理需求、元数据及余量。各区域已划分的容量不会自动互相借用。</p>
      </template>
      <p v-if="!validPlan" class="lh-notice danger">请核对容量或挂载目录，当前设置无法继续。</p>
      <footer class="lh-flow-actions"><button @click="stage=1">上一步</button><button class="primary" :disabled="!validPlan" @click="keepFiles?mountExisting():stage=3">{{keepFiles?(planOnly?'加入待执行':'挂载并使用（演示）'):'核对并继续'}}</button></footer>
    </template>
    <template v-else-if="stage===3">
      <h2 id="lh-flow-title">{{keepFiles?'确认挂载并使用':target.kind==='free'?'确认使用空闲空间':'确认清空所选空间'}}</h2>
      <p class="lh-notice" :class="{danger:!keepFiles&&target.kind!=='free'}">{{keepFiles?'仅挂载现有文件系统，不格式化、不重新分区，保留现有文件。':target.effect+'。'+(target.kind==='free'?'仅使用已核实的未分配范围。':'清空后不能通过撤销找回原文件。')}}</p>
      <dl class="lh-facts"><dt>目标</dt><dd>{{target.label}}</dd><dt>完成后</dt><dd>{{summary}}</dd><dt>保留内容</dt><dd>{{keepFiles?'所选空间的原文件及其他分区':target.preserved}}</dd></dl>
      <label v-if="!keepFiles" class="lh-confirm-label"><span>输入 {{target.token}} 确认目标</span><input v-model="typed" :placeholder="target.token" aria-label="输入目标名称确认" autocomplete="off" /></label>
      <label class="lh-check"><input v-model="ack" type="checkbox" /><span>{{keepFiles?'我确认挂载此文件系统，保留原文件。':target.kind==='free'?'我确认只使用未分配范围，保留系统分区。':'我已备份所需文件，确认清空上面这块空间。'}}</span></label>
      <footer class="lh-flow-actions"><button @click="stage=2">上一步</button><button :class="keepFiles||target.kind==='free'?'primary':'danger-button'" :disabled="!ack||(!keepFiles&&typed!==target.token)" @click="run">{{planOnly?'加入待执行':keepFiles?'挂载并使用（演示）':mode==='files'?'格式化为 EXT4 并使用（演示）':'准备存储（演示）'}}</button></footer>
    </template>
    <template v-else-if="stage===4"><h2 id="lh-flow-title">{{keepFiles?'正在挂载文件存储':'正在准备存储'}}</h2><ol class="lh-progress"><li v-for="(name,i) in progressSteps" :key="name" :class="{done:progress>i,current:progress===i}"><b>{{progress>i?'✓':progress===i?'◌':'·'}}</b><span>{{name}}</span></li></ol><p class="lh-notice">只演示进度，不执行真实磁盘操作。刷新会重置示例。</p></template>
    <template v-else-if="failed"><h2 id="lh-flow-title">存储准备未完成</h2><p class="lh-notice danger">布局已经修改，原文件不能自动恢复。本次停在创建存储步骤，不会自动重新清空。</p><button @click="diagnostic=true">查看处理建议</button><p v-if="diagnostic" class="lh-notice">保留现场，检查任务记录与存储组件；从已完成步骤继续。</p></template>
    <template v-else><h2 id="lh-flow-title">{{mode==='files'?'普通文件存储已就绪':'存储已准备好'}}</h2><p class="lh-flow-description">{{summary}}</p><div class="lh-success-icon">✓</div><p v-if="keepFiles" class="lh-notice">现有文件已保留，可以继续使用。</p><p v-if="plan.apps" class="lh-notice">应用卷按需创建；不会自动迁移已有 Docker 数据。</p><footer class="lh-flow-actions"><button @click="$emit('close')">查看存储总览</button><button v-if="plan.files" class="primary" @click="$emit('files')">打开文件（演示）</button><button v-else-if="plan.apps" class="primary" @click="$emit('volumes')">创建应用卷</button><button v-else class="primary" @click="$emit('continue')">继续设置 LightHost</button></footer></template>
  </section>
</template>
<script setup lang="ts">
import {computed,onUnmounted,ref,watch} from 'vue'
import {purposes,type Purpose,type StoragePlan,type StorageTarget} from './storagePurposePrototype'
const props=defineProps<{target:StorageTarget;scene:string;initialPurpose?:Purpose;planOnly?:boolean}>()
const emit=defineEmits<{(e:'close'):void;(e:'busy',value:boolean):void;(e:'complete',plan:StoragePlan):void;(e:'volumes'):void;(e:'files'):void;(e:'continue'):void}>()
const mode=ref<'files'|'custom'>(!props.initialPurpose||props.initialPurpose==='files'?'files':'custom'),purpose=ref<Purpose>(props.initialPurpose==='files'||!props.initialPurpose?'both':props.initialPurpose),stage=ref(props.scene==='ordinary'&&props.target.kind!=='free'&&(!props.initialPurpose||props.initialPurpose==='files')?2:1),hostShare=ref(Math.floor(props.target.capacity*.625/4)*4),management=ref<'existing'|'files'|'separate'>(props.scene==='no-management'?'separate':'existing'),typed=ref(''),ack=ref(false),progress=ref(0),failed=ref(false),diagnostic=ref(false),reserveFiles=ref(false),fileShare=ref(24),mountPath=ref('/mnt/'+props.target.token.replace('-free','-data'))
const hasExistingFiles=computed(()=>props.scene==='ordinary'&&props.target.kind!=='free')
const managementSize=16,keepFiles=computed(()=>mode.value==='files'&&hasExistingFiles.value),steps=computed(()=>keepFiles.value?['选择方式','挂载位置','确认设置','挂载使用']:['选择方式','安排空间','确认范围','准备存储'])
const maxFiles=computed(()=>props.target.capacity-(purpose.value==='both'?48:24)-(purpose.value==='hosts'&&management.value==='separate'?managementSize:0))
const managedCapacity=computed(()=>props.target.capacity-(reserveFiles.value?fileShare.value:0))
watch([managedCapacity,purpose],()=>{hostShare.value=Math.max(24,Math.min(hostShare.value,managedCapacity.value-24))})
watch(reserveFiles,value=>{if(!value&&management.value==='files')management.value=props.scene==='no-management'?'separate':'existing'})
const plan=computed<StoragePlan>(()=>mode.value==='files'?{purpose:'files',hosts:0,apps:0,files:props.target.capacity,management:'existing',managementSize:0,preserveExisting:keepFiles.value,fileMount:mountPath.value}:{purpose:purpose.value,hosts:purpose.value==='apps'?0:purpose.value==='both'?hostShare.value:managedCapacity.value-(management.value==='separate'?managementSize:0),apps:purpose.value==='hosts'?0:purpose.value==='both'?managedCapacity.value-hostShare.value:managedCapacity.value,files:reserveFiles.value?fileShare.value:0,management:purpose.value==='both'?'application':purpose.value==='apps'?'existing':management.value==='files'?'existing':management.value,managementSize:purpose.value==='apps'?0:managementSize,fileMount:reserveFiles.value?mountPath.value:undefined,managementInFiles:purpose.value==='hosts'&&management.value==='files'})
const validPlan=computed(()=>keepFiles.value?/^\/mnt\/[A-Za-z0-9_-]+$/.test(mountPath.value):mode.value==='files'||((!reserveFiles.value||(Number.isInteger(fileShare.value)&&fileShare.value>=8&&fileShare.value<=maxFiles.value))&&plan.value.hosts>=0&&plan.value.apps>=0))
const summary=computed(()=>[plan.value.files?`普通文件存储 ${plan.value.files} GiB · EXT4`:'',plan.value.hosts?`主机存储 ${plan.value.hosts} GiB`:'',plan.value.apps?`应用存储 ${plan.value.apps} GiB`:'',plan.value.management==='separate'?`管理文件区 ${managementSize} GiB`:'',keepFiles.value?mountPath.value:''].filter(Boolean).join('，'))
const progressSteps=computed(()=>keepFiles.value?['核对文件系统与挂载位置','挂载并验证访问']:mode.value==='files'?['核对目标与保留范围','准备 EXT4 文件系统','挂载并验证访问']:['再次核对目标与保留范围','划分所选空间','创建所需存储','准备管理文件位置','验证存储状态'])
let timer:ReturnType<typeof setInterval>|undefined
function mountExisting(){if(!keepFiles.value||!validPlan.value)return;ack.value=true;run()}
function run(){if(!validPlan.value||!ack.value||(!keepFiles.value&&typed.value!==props.target.token)||(mode.value==='custom'&&props.scene==='unsupported'))return;if(props.planOnly){emit('complete',{...plan.value});return}stage.value=4;emit('busy',true);timer=setInterval(()=>{progress.value++;if(props.scene==='failure'&&mode.value==='custom'&&progress.value===2){clearInterval(timer);failed.value=true;stage.value=5;emit('busy',false);return}if(progress.value===progressSteps.value.length){clearInterval(timer);emit('complete',{...plan.value});stage.value=5;emit('busy',false)}},550)}
onUnmounted(()=>{if(timer)clearInterval(timer)})
</script>
