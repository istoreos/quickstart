<template>
 <Teleport to="body">
 <div v-show="open" ref="overlay" class="feature-overlay device-feature-surface" @click.self="close" data-feature-overlay>
  <section ref="panel" class="feature-drawer" role="dialog" aria-modal="true" :aria-label="t('功能组件','Feature components')" tabindex="-1" data-feature-components>
   <div class="feature-heading"><div><h2>{{t('功能组件','Feature components')}}</h2><p>{{t('按需准备组件，再选择需要的功能。','Prepare components, then choose how to use them.')}}</p></div><button class="feature-close" type="button" @click="close" :aria-label="t('关闭功能组件','Close feature components')" data-close-components>×</button></div>
   <div class="feature-tools"><span>{{t('安装与启用不会自动切换当前限速方式。','Installing or enabling does not switch the selected limit mode.')}}</span><button type="button" :disabled="reading" @click="read" data-check-components>{{reading?t('正在检查…','Checking…'):t('重新检查','Check again')}}</button></div>
   <p v-if="notice" class="feature-feedback" role="status"><span>{{notice}}</span><button type="button" @click="notice=''" :aria-label="t('关闭提示','Dismiss message')">×</button></p>
   <p v-if="error" class="feature-feedback warning" role="status">{{error}}</p>
   <p v-if="reading&&!catalog" class="feature-loading" role="status">{{t('正在读取组件状态…','Reading component status…')}}</p>
   <div v-if="catalog" class="feature-list" :aria-busy="reading">
    <article v-for="item in catalog.components" :key="item.id" class="feature-item" :class="{target:focusId===item.id}" :data-feature-component="item.id" tabindex="-1">
     <div class="feature-item-heading"><div><h3><HomeGlyph :name="item.id==='floatip'?'network':'activity'"/>{{name(item.id)}}</h3><div class="feature-tags"><span v-if="item.id==='quickstart-native'" class="recommend">{{t('推荐','Recommended')}}</span><span v-if="item.selected">{{t('当前使用','Currently selected')}}</span></div></div><span class="feature-state" :class="item.status.state">{{status(item)}}</span></div>
     <p>{{description(item.id)}}</p>
     <p v-if="item.status.reason" class="feature-reason">{{reason(item.status.reason)}}</p>
     <div class="feature-actions">
      <button v-if="componentAction(item)" class="primary" type="button" :disabled="reading||Boolean(busy)||Boolean(error)" @click="preview(item)" :data-prepare-component="item.id">{{componentAction(item)==='enable'?t('查看并启用','Review and enable'):t('查看并安装','Review and install')}}</button>
      <button v-else-if="item.operation" type="button" :disabled="reading" @click="read">{{t('检查状态','Check status')}}</button>
      <button v-if="canConfigure(item)" type="button" :disabled="Boolean(busy)||reading||Boolean(error)" @click="configure(item)" :data-configure-component="item.id">{{item.provider&&!item.selected?t('预览切换','Review switch'):t('查看设置','View settings')}}</button>
     </div>
     <section v-if="plan&&plan.capabilityKey===item.capabilityKey" class="feature-confirm" :aria-label="t('确认组件操作','Confirm component action')" data-component-confirm>
      <h4>{{plan.action==='install'?t('安装组件','Install component'):t('启用组件','Enable component')}}</h4>
      <dl><div><dt>{{t('组件','Component')}}</dt><dd>{{item.package}}</dd></div><div v-if="plan.action==='install'"><dt>{{t('最低可用空间','Minimum free space')}}</dt><dd>{{Math.ceil((plan.requiredFreeBytes||0)/1048576)}} MB</dd></div></dl>
      <p>{{plan.action==='install'?t('安装会更新软件源，包脚本可能启动组件服务。当前执行方式和未提交设置会保留。','Installation updates package sources; package scripts may start the component. The selected mode and unsaved settings are retained.'):t('将启动组件并设置开机启用，可能恢复该组件已有规则。当前限速方式的切换仍需另行确认。','Starts the component and enables it at boot; existing component rules may resume. Switching the selected limit mode needs a separate confirmation.')}}</p>
      <p v-if="plan.operationScope === 'system' || (plan.affectedNetworks?.length || 0) > 1">{{t('此操作影响局域网：','Affected LANs: ')}}{{plan.affectedNetworks?.join(', ') || t('全部局域网','All LANs')}}</p>
      <div class="feature-confirm-actions"><button type="button" :disabled="Boolean(busy)" @click="plan=undefined">{{t('取消','Cancel')}}</button><button class="primary" type="button" :disabled="Boolean(busy)" @click="apply" data-confirm-component>{{busy?t('正在处理…','Working…'):plan.action==='install'?t('确认安装','Confirm installation'):t('确认启用','Confirm enabling')}}</button></div>
     </section>
    </article>
   </div>
   <p v-if="busy" class="feature-progress" role="status">{{t('操作已提交。关闭面板不会取消操作，请勿重复提交。','Action submitted. Closing this panel does not cancel it; do not submit it again.')}}</p>
   <p v-if="allowConfigure===false" class="feature-progress">{{t('执行方式和网关配置可在“局域网设置”中调整。','Change the limit mode and gateway configuration in LAN settings.')}}</p>
  </section>
 </div>
 </Teleport>
</template>
<script setup lang="ts">
import {nextTick,onBeforeUnmount,ref,watch} from 'vue'
import HomeGlyph from '/@/home/HomeGlyph.vue'
import '../deviceVisualTokens.css'
import { useLANContext } from '../lanContext'
import {componentDefinitions,componentAction,createComponentClient,ComponentError,type ComponentId,type FeatureComponent,type ComponentCatalog,type ComponentPlan} from '../featureComponents'
const props=withDefaults(defineProps<{open:boolean;focusId?:ComponentId;allowConfigure?:boolean}>(),{allowConfigure:true})
const emit=defineEmits<{(event:'close'):void;(event:'updated'):void;(event:'busy',value:boolean):void;(event:'configure',id:ComponentId):void}>()
const t=(zh:string,en:string)=>window.vue_lang==='en'?en:zh
const lan=useLANContext()
const client=createComponentClient(fetch,lan.legacy?undefined:lan.networkId),catalog=ref<ComponentCatalog>(),reading=ref(false),error=ref(''),notice=ref(''),busy=ref(''),plan=ref<ComponentPlan>()
const overlay=ref<HTMLElement>(),panel=ref<HTMLElement>()
let disposed=false,generation=0,planGeneration=0,locked=false,readController:AbortController|undefined,planController:AbortController|undefined,previousFocus:HTMLElement|null=null,previousOverflow='',inertNodes:{node:HTMLElement;value:boolean}[]=[]
const definition=(id:ComponentId)=>componentDefinitions.find(item=>item.id===id)!
const name=(id:ComponentId)=>{const d=definition(id);return t(d.zh,d.en)}
const description=(id:ComponentId)=>{const d=definition(id);return t(d.descriptionZh,d.descriptionEn)}
const reason=(value:string)=>({dependency_not_installed:t('尚未安装此组件。','This component is not installed.'),service_stopped:t('组件已安装，尚未运行。','Installed; the component is not running.'),engine_unavailable:t('服务未就绪，请检查运行情况。','The service is not ready; check its status.'),interface_not_supported:t('组件版本与当前接口不兼容。','The component version is incompatible.'),kernel_not_supported:t('当前系统内核不支持。','The current kernel is unsupported.'),status_unavailable:t('暂时无法确认状态，请重新检查。','Status is unavailable; check again.')} as Record<string,string>)[value]||t('当前不可用，请检查组件或固件。','Unavailable; check the component and firmware.')
const status=(item:FeatureComponent)=>item.operation?t(item.operation==='running'?'正在处理':'结果待确认',item.operation==='running'?'Working':'Outcome unconfirmed'):({available:item.id==='floatip'?t('已启用','Enabled'):t('已就绪','Ready'),disabled:t('未启用','Not enabled'),not_installed:t('未安装','Not installed'),unsupported:t('不兼容','Unsupported'),error:t('需要检查','Needs attention')} as Record<string,string>)[item.status.state]
const message=(value:unknown)=>{
 const code=value instanceof ComponentError?value.code:'unavailable'
 return ({forbidden:t('登录已失效或没有操作权限，请重新登录。','Session expired or access denied. Sign in again.'),unsupported:t('当前后端不支持组件管理，请更新后端后重试。','This backend does not support component management. Update it and retry.'),invalid:t('组件信息不完整，未执行操作，请重新检查。','Component information is invalid. Check again before proceeding.'),uncertain:t('操作结果尚未确认。请先检查状态，不要重复安装或启用。','Outcome unconfirmed. Check status before installing or enabling again.'),component_busy:t('另一个组件操作正在进行，请检查状态。','Another component action is running. Check its status.'),component_pending:t('组件操作结果待确认，请检查状态。','Component outcome is pending. Check its status.'),install_pending:t('安装结果待确认，请检查状态。','Installation outcome is pending. Check its status.'),enable_pending:t('启用结果待确认，请检查状态。','Enabling outcome is pending. Check its status.'),insufficient_space:t('可用空间不足，无法安装组件。','There is not enough free storage to install this component.'),conflict:t('组件状态已变化，请重新检查后确认。','Component state changed. Check it again before confirming.')} as Record<string,string>)[code]||t('操作未完成，请检查软件源、组件和固件后重试。','The action could not finish. Check sources, component and firmware, then retry.')
}
async function read(notify:unknown=false){if(reading.value)return;const current=++generation;readController=new AbortController();reading.value=true;error.value='';try{const result=await client.read(readController.signal);if(disposed||current!==generation)return;catalog.value=result;if(notify)emit('updated')}catch(value){if(!disposed&&current===generation)error.value=message(value)}finally{if(!disposed&&current===generation)reading.value=false}}
async function preview(item:FeatureComponent){if(busy.value||reading.value||error.value)return;const action=componentAction(item);if(!action)return;const current=++planGeneration;planController=new AbortController();busy.value='plan';plan.value=undefined;notice.value='';try{const next=await client.plan(item,action,planController.signal);if(!disposed&&props.open&&current===planGeneration)plan.value=next}catch(value){if(!disposed&&current===planGeneration){notice.value=message(value);void read()}}finally{if(!disposed)busy.value=''}}
async function apply(){if(!plan.value||busy.value)return;const submitted=plan.value;busy.value=submitted.action;notice.value='';try{await client.apply(submitted);if(!disposed)notice.value=t('操作已完成，请以重新检查的组件状态为准。','Action finished. Check the updated component state.')}catch(value){if(!disposed)notice.value=message(value)}finally{if(!disposed){busy.value='';plan.value=undefined;await read(true)}}}
const canConfigure=(item:FeatureComponent)=>props.allowConfigure!==false&&!item.operation&&(item.status.state==='available'||item.id==='floatip'&&item.status.state==='disabled')
function configure(item:FeatureComponent){emit('configure',item.id);close()}
function close(){planGeneration++;planController?.abort();readController?.abort();generation++;reading.value=false;plan.value=undefined;emit('close')}
function keydown(event:KeyboardEvent){if(!props.open)return;if(event.key==='Escape'){event.preventDefault();event.stopImmediatePropagation();close()}else if(event.key==='Tab'){
 const controls=Array.from(panel.value!.querySelectorAll<HTMLElement>('button:not(:disabled),[href],input,select,[tabindex="0"]')).filter(node=>node.getClientRects().length>0&&getComputedStyle(node).visibility!=='hidden')
 const first=controls[0],last=controls[controls.length-1];if(!first)return
 if(event.shiftKey&&(document.activeElement===first||document.activeElement===panel.value)){event.preventDefault();last.focus()}else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus()}
}}
function unlock(){if(!locked)return;locked=false;document.removeEventListener('keydown',keydown,true);for(const item of inertNodes){if(item.value)item.node.setAttribute('inert','');else item.node.removeAttribute('inert')}inertNodes=[];document.body.style.overflow=previousOverflow;previousFocus?.isConnected&&previousFocus.focus({preventScroll:true})}
watch(busy,value=>emit('busy',Boolean(value)))
watch(()=>props.open,async open=>{
 if(!open){unlock();return}
 locked=true;previousFocus=document.activeElement as HTMLElement;previousOverflow=document.body.style.overflow;document.body.style.overflow='hidden';document.addEventListener('keydown',keydown,true)
 await nextTick();if(disposed||!props.open)return
 let node=overlay.value
 while(node&&node.parentElement){for(const sibling of Array.from(node.parentElement.children))if(sibling!==node&&sibling instanceof HTMLElement&&!['SCRIPT','STYLE'].includes(sibling.tagName)){inertNodes.push({node:sibling,value:sibling.hasAttribute('inert')});sibling.setAttribute('inert','')}if(node.parentElement===document.body)break;node=node.parentElement}
 panel.value?.focus({preventScroll:true});await read();await nextTick();if(!disposed&&props.open&&props.focusId){const target=panel.value?.querySelector<HTMLElement>('[data-feature-component="'+props.focusId+'"]');target?.focus({preventScroll:true});target?.scrollIntoView({block:'nearest'})}
},{immediate:true})
onBeforeUnmount(()=>{disposed=true;generation++;readController?.abort();planController?.abort();if(props.open)unlock()})
</script>
<style scoped>
.feature-drawer .feature-heading h2,.feature-drawer .feature-confirm h4{padding:0!important;box-shadow:none!important;border:0;background:transparent!important;color:var(--device-text);line-height:1.4}
.feature-overlay{position:fixed;inset:0;z-index:2500;display:flex;justify-content:flex-end;background:#0006;color:var(--device-text);font-size:14px}.feature-drawer{box-sizing:border-box;width:min(560px,100%);height:100%;overflow:auto;overscroll-behavior:contain;background:var(--device-panel);padding:20px;box-shadow:-8px 0 28px #0003}.feature-heading{display:flex;justify-content:space-between;align-items:flex-start;gap:12px}.feature-heading h2{margin:0;font-size:20px;color:var(--device-text)}.feature-heading p,.feature-item>p{margin:8px 0;color:var(--device-muted);line-height:1.6;font-size:13px}.feature-drawer button{box-sizing:border-box;min-width:44px;min-height:44px;padding:8px 12px;border:1px solid var(--device-line);border-radius:7px;background:transparent;color:var(--device-accent);font:inherit;cursor:pointer}.feature-drawer button:disabled{opacity:.55;cursor:default}.feature-drawer button:focus-visible,.feature-item:focus-visible{outline:2px solid var(--device-accent);outline-offset:2px}.feature-drawer .feature-close{flex:none;border:0;color:var(--device-muted);font-size:24px;padding:4px}.feature-tools{display:flex;gap:12px;align-items:center;margin:16px 0}.feature-tools>span{flex:1;color:var(--device-muted);font-size:12px;line-height:1.6}.feature-tools>button{flex:none}.feature-list{display:grid;gap:12px}.feature-item{min-width:0;border:1px solid var(--device-line);border-radius:10px;padding:16px}.feature-item.target{border-color:var(--device-accent)}.feature-item-heading{display:flex;justify-content:space-between;gap:12px;align-items:flex-start}.feature-item h3{display:flex;gap:7px;align-items:center;font-size:16px;margin:0;color:var(--device-text);background:none;text-align:left;padding:0}.feature-item h3 svg{width:18px;height:18px;color:var(--device-accent);flex:none}.feature-tags{display:flex;gap:6px;margin-top:6px;flex-wrap:wrap}.feature-tags span{font-size:11px;color:var(--device-muted)}.feature-tags .recommend{color:var(--device-accent)}.feature-state{flex:none;font-size:12px;color:var(--device-muted);padding-top:3px}.feature-state.available{color:var(--device-good)}.feature-state.not_installed,.feature-state.error,.feature-state.unsupported{color:var(--device-warning)}.feature-actions{display:flex;gap:8px;flex-wrap:wrap;margin-top:12px}.feature-drawer .primary{color:var(--device-primary-text);background:var(--device-primary-bg);border-color:var(--device-primary-bg)}.feature-confirm{margin-top:14px;padding:14px;background:var(--device-soft);border:1px solid var(--device-line);border-radius:8px}.feature-confirm h4{font-size:14px;margin:0 0 10px}.feature-confirm dl{margin:0;display:grid;gap:8px}.feature-confirm dl>div{display:grid;grid-template-columns:auto minmax(0,1fr);gap:12px;font-size:12px}.feature-confirm dt{color:var(--device-muted)}.feature-confirm dd{margin:0;overflow-wrap:anywhere}.feature-confirm p,.feature-progress{font-size:12px;color:var(--device-muted);line-height:1.7}.feature-confirm-actions{display:flex;justify-content:flex-end;gap:8px;flex-wrap:wrap}.feature-feedback{display:flex;align-items:center;gap:8px;padding:8px 10px;background:var(--device-soft);color:var(--device-muted);border-radius:7px;font-size:13px;line-height:1.6}.feature-feedback>span{flex:1}.feature-feedback>button{border:0;flex:none}.feature-feedback.warning{color:var(--device-warning);background:var(--device-warning-soft)}.feature-loading{font-size:13px;color:var(--device-muted)}@media(max-width:520px){.feature-drawer{padding:16px}.feature-item{padding:13px}.feature-item-heading{gap:8px;flex-wrap:wrap}.feature-tools{align-items:flex-start}.feature-tools>button{padding:8px}.feature-confirm{padding:12px}.feature-confirm-actions>button{flex:1}.feature-confirm dl>div{grid-template-columns:1fr;gap:4px}}
</style>
