<!-- Design only: maintenance drawer, memory examples, no real disk or notification operations. -->
<template>
 <div class="sc-shade" @click.self="$emit('close')">
  <section class="sc-drawer" role="dialog" aria-modal="true" :aria-label="title" tabindex="-1" ref="root">
   <header><div><small>硬盘与存储</small><h2>{{title}}</h2></div><button aria-label="关闭维护详情" @click="$emit('close')">×</button></header>
   <div class="sc-body">
    <template v-if="panel==='health'">
     <h3>{{care.aliases[target]||disk?.model||target}} <small>{{disk?.id}}</small></h3>
     <dl><dt>当前连接</dt><dd>{{disk?.connected?'在线':'未连接'}}</dd><dt>物理身份</dt><dd>{{disk?.serial||'未知'}} · {{disk?.identityReliable===false?'身份不可靠，禁用写入':'示例稳定身份'}}</dd><dt>连接位置</dt><dd>{{disk?.id?.startsWith('md')?'阵列逻辑设备':'SATA / 演示插槽 '+(workbench.drives.findIndex(d=>d.id===target)+1)}}；本设备不提供定位灯</dd><dt>健康状态</dt><dd>{{health.state}}</dd><dt>上次测试</dt><dd>{{health.result}}</dd><dt>维护计划</dt><dd>{{health.schedule}}</dd></dl>
     <p class="sw-note">SMART 自检与文件内容校验不同。休眠、不支持和读取失败都不能显示成健康通过。</p>
     <p v-if="maintenanceBusy()" class="sw-note warning">正在进行存储维护，暂不启动另一项重 I/O 测试。可在任务记录查看。</p>
     <button class="primary" :disabled="!disk?.connected||maintenanceBusy()||workbench.readState!=='fresh'" @click="startSmart">运行短测试（演示）</button>
     <div class="sc-field"><label for="smart-plan">定期测试</label><select id="smart-plan" v-model="schedule"><option>未设置</option><option>每周日 03:00 · 短测试</option><option>每月首日 03:00 · 长测试</option></select><button @click="health.schedule=schedule;message='维护计划已保存（演示）；冲突时延后并记录原因。'">保存计划</button></div>
     <details><summary>模拟健康状态</summary><select aria-label="健康状态示例" v-model="health.state"><option>未知</option><option>休眠</option><option>不支持</option><option>尚未测试</option><option>异常</option></select></details>
     <button @click="$emit('navigate','jobs')">查看测试结果与任务</button><p class="sc-muted"><router-link to="/smart">S.M.A.R.T. 全局设置、计划与日志 ›</router-link></p>
    </template>
    <template v-else-if="panel==='capacity'">
     <h3>{{pool?.name}}</h3><p class="sc-muted">{{pool?.owner}} 管理 · 以下容量来自示例采样</p>
     <dl><dt>数据占用</dt><dd>{{pool?.used}} / {{pool?.size}} GiB</dd><dt>元数据</dt><dd>{{pool?.metadata}}% 已使用</dd><dt>VG 空闲</dt><dd>尚未采样，不能按预留量推算</dd><dt>布局预留</dt><dd>{{pool?.reserve}} GiB，包含元数据与应急余量</dd><dt>自动扩容</dt><dd>未验证监控与 VG 余量，不承诺自动扩容</dd><dt>满池策略</dt><dd>需目标机验证；空间紧张时暂停新建卷与恢复点</dd></dl>
     <h3>先处理什么？</h3><ol><li>在归属应用中查看恢复点与备份，再选择清理。</li><li>删除文件后重新采样，确认 discard 是否归还空间；快照可能继续持有旧块。</li><li>确认有可扩空间和受支持布局后，再执行预检。</li></ol>
     <div class="sc-resource" v-for="v in pool?.volumes" :key="v.id"><strong>{{v.name}}</strong><span>{{v.snapshots}} 个恢复点 · {{v.attached?'使用中':'未接入'}}</span><button @click="$emit('open','snapshots',v.id)">查看恢复点归属</button></div>
     <button disabled>扩容此存储池 · 等待布局资格验证</button><p class="sc-muted">当前原型不提供任意扩池按钮。不能在线扩容时先备份，再迁移到新存储。</p>
    </template>
    <template v-else-if="panel==='readiness'">
     <h3>{{appName}} · {{appState}}</h3><dl><dt>目标卷</dt><dd>{{volume?.name||'宿主 Docker 数据'}}</dd><dt>挂载目录</dt><dd>{{appName==='LightHost 管理文件'?'/mnt/lighthost-data':'/mnt/docker-data'}}</dd><dt>预期身份</dt><dd>{{target==='docker'?care.expectedUUID:'DEMO-FS-'+target}}</dd><dt>开机检查</dt><dd>阵列组装 → LVM 激活 → UUID 挂载核对 → 应用启动</dd></dl>
     <p class="sw-note warning" v-if="appState!=='就绪'">目标文件系统尚未就绪，{{appName}} 保持停止。目录存在不能证明正确磁盘已经挂载；不回落到系统可写区。</p>
     <p v-else class="sw-note">示例已核对目标文件系统身份与可写状态，可启动应用。真实实现仍需目标机重启与缺盘验证。</p>
     <label class="sc-field">模拟挂载结果<select aria-label="挂载检查示例" v-model="appState"><option>未接入</option><option>等待存储</option><option>挂载失败</option><option>就绪</option></select></label><button @click="message='已重新核对示例状态；真实系统须核对 mountinfo、UUID、读写和容量，不能只检测目录。'">重新检查（演示）</button>
     <details><summary>接入和迁移的影响</summary><p>宿主 Docker 只有一个当前 data-root。先停服务、复制校验、切换、启动检查；验证前保留旧数据。切换后两边都有新写入时，不能直接回退。</p><p>LightHost 管理文件也须通过开机挂载守卫。LightHost 主机卷在 Incus 中管理；沙箱 LVM 尚未开放。</p></details>
    </template>
    <template v-else-if="panel==='discovery'">
     <p class="sw-note">先只读识别已有成员，不修改配置，不组装阵列，不重启服务。原“扫描恢复”仍是独立的有影响操作。</p>
     <button class="primary" @click="care.discoveryState='found';message='只读识别完成（示例），未调用 Autofix。'">识别已有阵列（演示）</button>
     <div v-if="care.discoveryState==='found'" class="sc-resource"><h3>家庭文件镜像 · RAID 1</h3><dl><dt>阵列 UUID</dt><dd>DEMO-MD-001</dd><dt>识别成员</dt><dd>sdd1、sde1 · 序列号末位 004 / 005</dd><dt>缺失成员</dt><dd>无；识别不等于可直接使用</dd><dt>配置状态</dt><dd>尚未由本机接管</dd></dl><label><input type="checkbox" v-model="conflict" />模拟 UUID / 旧成员事件冲突</label><p v-if="conflict" class="sw-note warning">存在成员身份冲突。暂不能恢复；先导出诊断资料并核实来源，不强制组装。</p><button :disabled="conflict" @click="discoveryReview=true">核对恢复影响</button><template v-if="discoveryReview&&!conflict"><p class="sw-note warning">恢复会写入选定阵列配置并尝试组装，可能影响关联文件服务。新发现接口在生产接入前仍待实现；不会自动调用旧全量 Autofix。</p><button disabled>执行选定恢复 · 待新接口接入</button></template></div>
    </template>
    <template v-else-if="panel==='maintenance'">
     <h3>{{array?.name||'RAID 阵列'}}</h3><p class="sc-muted">状态：{{array?.status||'未发现'}} · 上次一致性检查：{{lastMdCheck}} · 下次：{{care.mdSchedule}}</p>
     <p class="sw-note">检查用于发现阵列层的一致性问题，不等于文件内容校验。不同 RAID 级别的 check 可能有不同写入行为；修复需单独确认，不能套用 ZFS 的自修复承诺。</p>
     <p v-if="maintenanceBusy()" class="sw-note warning">存在重建、同步或其他维护任务，请先等待完成。</p>
     <button class="primary" :disabled="!array||maintenanceBusy()||workbench.readState!=='fresh'" @click="startCheck">开始一致性检查（演示）</button><button disabled>修复 · 需诊断结果与专项流程</button>
     <label class="sc-field">检查计划<select aria-label="阵列检查计划" v-model="care.mdSchedule"><option>未设置</option><option>每月首个周日 04:00</option></select></label><button @click="message='阵列检查计划已保存（演示）；调度需避开重建与 SMART 长测试。'">保存检查计划</button>
     <button @click="$emit('open','recovery',array?.id||'md0')">核对换盘与重新加入</button><button @click="$emit('navigate','jobs')">查看维护结果</button>
    </template>
    <template v-else-if="panel==='recovery'">
     <h3>恢复阵列成员</h3><label class="sc-choice"><input type="radio" value="old" v-model="care.recoveryChoice" />原盘重新连接</label><label class="sc-choice"><input type="radio" value="new" v-model="care.recoveryChoice" />使用新硬盘恢复</label>
     <p class="sw-note" v-if="care.recoveryChoice==='old'">先核对阵列 UUID、成员角色、事件与扇区。不能因盘符相同就直接重新加入；冲突或过旧成员需诊断，原有 Recover 不保证原盘无损。</p><p class="sw-note warning" v-else>新盘可能被清空。核对型号、序列号、实际可用扇区、扇区大小、只读与占用后，才能进入现有恢复流程。</p>
     <dl><dt>目标阵列</dt><dd>{{target}} · DEMO-MD-001</dd><dt>候选新盘</dt><dd>sdf · DEMO-NAS-006 · 1024 GiB</dd><dt>容量资格</dt><dd>需实际扇区比较，标称同容量不代表可替换</dd></dl><p v-if="care.recoveryChoice==='old'" class="sc-muted">原盘专用重新加入流程待后端接入；此处不提供强制恢复。</p><button v-else @click="$emit('navigate','raid')">在现有 RAID 管理中恢复</button>
    </template>
    <template v-else-if="panel==='events'">
     <p class="sc-muted">相同资源和事件类型合并显示。标记已读不会修复问题。</p>
     <div class="sc-resource" v-for="alert in careAlerts" :key="alert.id"><strong>{{alert.title}}</strong><span>{{care.read.includes(alert.id)?'已读 · 问题仍在':'待处理'}}</span><button @click="care.read.includes(alert.id)||care.read.push(alert.id)">标记已读</button><button @click="$emit('open',alert.panel,alert.target)">定位与处理</button></div><p v-if="!careAlerts.length" class="sw-note">当前示例没有未解决的存储告警。</p>
     <details v-if="care.events.some(e=>e.resolvedAt)"><summary>已恢复的提醒</summary><div v-for="event in care.events.filter(e=>e.resolvedAt)" :key="event.id+':'+event.episode" class="sc-resource"><strong>{{event.title}}</strong><span>发现：{{event.firstSeen}} · 已恢复：{{event.resolvedAt}}</span></div></details><h3>通知设置</h3><p class="sc-muted">演示站内状态与外发测试，不发送真实消息。生产离线通知需要服务端持久投递。</p><label class="sc-field">通知渠道<select aria-label="通知渠道" v-model="care.channel"><option>站内提醒</option><option>外发 Webhook · 演示配置</option></select></label><label><input type="checkbox" v-model="care.deliveryFailure" />模拟通知发送失败</label><button :disabled="care.notificationState==='sending'" @click="testNotice">发送测试（演示）</button><p role="status">{{care.notificationTest}}</p>
    </template>
    <template v-else-if="panel==='task'">
     <template v-if="task"><h3>{{task.title}} · {{task.state}}</h3><p class="sc-muted">{{task.target}} · {{task.started}}</p><p v-if="care.queryInterrupted" class="sw-note warning">任务查询中断；保留上次记录，不判定服务端失败。</p><ol class="sc-stages"><li v-for="stage in task.stages" :key="stage.name"><strong>{{stage.name}} <span>{{stage.state}}</span></strong><p>{{stage.detail}}</p></li></ol><p class="sw-note">{{task.next}}</p><div v-if="task.state==='执行中'" class="sc-actions"><button @click="finishCareTask(task,'完成')">模拟完成</button><button @click="finishCareTask(task,'失败')">模拟失败</button><button v-if="task.title==='SMART 短测试'" @click="finishCareTask(task,'已中止')">中止示例测试</button></div><button @click="care.queryInterrupted=!care.queryInterrupted">{{care.queryInterrupted?'恢复任务查询（演示）':'模拟查询中断'}}</button></template><p v-else>找不到此任务，请返回任务记录。</p>
     <p class="sc-muted">记录在本次原型会话内共享，刷新会重置。生产需服务端持久化、请求去重与重启核对。</p>
    </template>
    <template v-else-if="panel==='snapshots'">
     <h3>{{volume?.name||'受管理卷'}} · 恢复点与备份</h3><dl><dt>管理归属</dt><dd>{{volumePool?.owner||'对应应用'}}；在归属模块操作</dd><dt>恢复点数量</dt><dd>{{volume?.snapshots||0}} 个示例恢复点</dd><dt>覆盖范围</dt><dd>当前逻辑卷；外部 bind mount 与其他磁盘不自动包含</dd><dt>一致性</dt><dd>未核实应用静默，不能宣称数据库一致</dd><dt>保留策略</dt><dd>待归属模块配置；不能自动删除用户恢复点</dd><dt>异地备份</dt><dd>尚未验证 · 快照不是备份</dd><dt>恢复验证</dt><dd>待恢复到独立位置后检查应用和文件</dd></dl><p class="sw-note">备份部分目录时，由归属模块提供只读视图，再交给 backuprepo。VM 磁盘需识别来宾文件系统，不能直接作为目录。</p><p class="sc-muted">只读备份视图必须有租约、到期回收和异常解冻；本原型不创建视图、不删除恢复点。</p><button disabled>管理恢复点 · 待归属模块接入</button><button disabled>目录备份 · 待只读视图接口</button>
    </template>
    <template v-else>
     <h3>设备名称与支持资料</h3><label class="sc-field">用途别名<input aria-label="设备用途别名" v-model="alias" maxlength="32" placeholder="例如：客厅文件盘" /></label><button @click="care.aliases[target]=alias.trim();message='用途别名已保存（演示），不改变 UUID、设备路径或挂载配置。'">保存别名</button><p class="sw-note">诊断仅导出版本标记、脱敏设备身份、容量、状态和任务阶段，不包含用户文件、账号或密钥。</p><button @click="download">导出示例诊断资料</button>
    </template>
    <p v-if="message" role="status" class="sw-note">{{message}}</p>
   </div>
   <footer><span>示例数据 · 不执行真实操作</span><button @click="$emit('close')">返回</button></footer>
  </section>
 </div>
</template>
<script setup lang="ts">
import {computed,nextTick,onMounted,onUnmounted,ref,watch} from 'vue'
import {workbench} from './storageWorkbenchPrototype'
import {care,careAlerts,healthFor,maintenanceBusy,startCareTask,finishCareTask,jobAsTask,exportCareDiagnostics,type CarePanel} from './storageCarePrototype'
const props=defineProps<{panel:CarePanel;target:string}>(),emit=defineEmits<{(e:'close'):void;(e:'navigate',tab:string):void;(e:'open',panel:CarePanel,target:string):void}>()
const titles:Record<CarePanel,string>={health:'健康与物理身份',capacity:'处理存储空间',readiness:'应用存储就绪检查',discovery:'识别已有阵列',maintenance:'阵列健康与维护',recovery:'换盘与重新加入',events:'存储提醒与通知',task:'任务详情',snapshots:'恢复点与备份',support:'名称与诊断资料'}
const title=computed(()=>titles[props.panel]),root=ref<HTMLElement>(),message=ref(''),schedule=ref('未设置'),alias=ref(''),conflict=ref(false),discoveryReview=ref(false)
const disk=computed(()=>workbench.drives.find(d=>d.id===props.target)),health=computed(()=>healthFor(props.target)),pool=computed(()=>workbench.pools.find(p=>p.id===props.target)||workbench.pools[1]),array=computed(()=>workbench.raids.find(r=>r.id===props.target)||workbench.raids[0])
const volumePool=computed(()=>workbench.pools.find(p=>p.volumes.some(v=>v.id===props.target))),volume=computed(()=>volumePool.value?.volumes.find(v=>v.id===props.target))
const appName=computed(()=>volume.value?.kind==='LightHost 管理卷'?'LightHost 管理文件':'宿主 Docker')
const appState=computed({get:()=>props.target==='docker'?care.applicationState:care.applicationStates[props.target]||(volume.value?.attached?'就绪':'未接入'),set:(value:string)=>{if(props.target==='docker')care.applicationState=value;else care.applicationStates[props.target]=value}})
const lastMdCheck=computed(()=>{const t=care.tasks.find(t=>t.title==='阵列一致性检查'&&t.target===(array.value?.id||props.target)&&t.state!=='执行中');return t?t.started+' · '+t.state:'暂无结果'})
const task=computed(()=>care.tasks.find(t=>String(t.id)===props.target)||(()=>{const j=workbench.jobs.find(j=>String(j.id)===props.target);return j?jobAsTask(j):undefined})())
function startSmart(){if(startCareTask('SMART 短测试',props.target,['提交自检','读取自检结果'])){health.value.state='测试中';emit('open','task',String(care.tasks[0].id))}}
function startCheck(){if(startCareTask('阵列一致性检查',array.value?.id||props.target,['启动检查','读取内核检查状态','保存检查结果']))emit('open','task',String(care.tasks[0].id))}
let testTimer:ReturnType<typeof setTimeout>|undefined
function testNotice(){care.notificationState='sending';care.notificationTest='正在发送示例测试…';testTimer=setTimeout(()=>{care.notificationState=care.deliveryFailure?'failed':'sent';care.notificationTest=care.deliveryFailure?'测试发送失败（演示），请检查渠道配置后重试。':'测试消息已送达（演示）；真实外发未启用。'},300)}
function download(){const url=URL.createObjectURL(new Blob([exportCareDiagnostics()],{type:'application/json'})),a=document.createElement('a');a.href=url;a.download='quickstart-storage-prototype-diagnostics.json';a.click();URL.revokeObjectURL(url);message.value='示例诊断已导出，设备身份已脱敏。'}
function keydown(e:KeyboardEvent){if(e.key==='Escape'){e.preventDefault();e.stopImmediatePropagation();emit('close');return}if(e.key!=='Tab'||!root.value)return;const els=Array.from(root.value.querySelectorAll<HTMLElement>('button:not(:disabled),input,select,summary,a[href]')).filter(e=>e.offsetParent!==null),first=els[0],last=els[els.length-1];if(e.shiftKey&&(document.activeElement===first||document.activeElement===root.value)){e.preventDefault();last?.focus()}else if(!e.shiftKey&&(document.activeElement===last||document.activeElement===root.value)){e.preventDefault();first?.focus()}}
let trigger:HTMLElement|null=null,overflow=''
watch(()=>[props.panel,props.target],async()=>{message.value='';alias.value=care.aliases[props.target]||'';schedule.value=health.value.schedule;discoveryReview.value=false;await nextTick();root.value?.focus()},{immediate:true})
onMounted(async()=>{trigger=document.activeElement as HTMLElement;overflow=document.body.style.overflow;document.body.style.overflow='hidden';document.addEventListener('keydown',keydown,true);await nextTick();root.value?.focus()})
onUnmounted(()=>{if(testTimer){clearTimeout(testTimer);if(care.notificationState==='sending'){care.notificationState='idle';care.notificationTest='测试查询已离开，请重新测试。'}}document.body.style.overflow=overflow;document.removeEventListener('keydown',keydown,true);if(trigger?.isConnected)trigger.focus({preventScroll:true})})
</script>
<style src="./storageCarePrototype.css"></style>
