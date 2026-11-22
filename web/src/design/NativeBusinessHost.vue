<template>
 <Teleport to="body"><div v-for="(entry,index) in entries" :key="entry.id" class="native-action-host" :style="{zIndex:5000+index}" :data-native-action="entry.kind" role="dialog" aria-modal="true" tabindex="-1" :aria-label="entry.kind" @click.capture="captureLink" @keydown="key($event,entry)"><component :is="registry[entry.kind]" v-bind="entry.props" :Close="()=>close(entry.id)"/></div></Teleport>
 <NetworkWidgetDialog v-if="destination" :title="english?'Original destination':'原业务入口'" @close="destination=null"><div class="native-destination" data-native-destination><p>{{english?'This design preview keeps the original destination.':'设计预览保留以下原始业务入口。'}}</p><code>{{destination.href}}</code><dl v-if="destination.pkg"><dt>{{english?'Package check':'检查软件包'}}</dt><dd>{{destination.pkg}}</dd><dt>{{english?'Install package':'安装软件包'}}</dt><dd>{{destination.installPkg||destination.pkg}}</dd></dl><p>{{english?'Synthetic data only; no external service was opened.':'当前为示例数据，未访问设备或外部服务。'}}</p></div></NetworkWidgetDialog>
</template>
<script setup lang="ts">
import {markRaw,nextTick,onUnmounted,ref} from 'vue'
import appUtils from '../utils/app'
import {registerNativeActions} from './nativeActionBridge'
import NetworkWidgetDialog from './NetworkWidgetDialog.vue'
import Nas from '../components/action-nas/index.vue'
import Linkease from '../components/action-nas/linkease/index.vue'
import Unishare from '../components/action-nas/unishare/index.vue'
import Docker from '../components/action-docker/index.vue'
import Download from '../components/action-download/index.vue'
import Domain from '../components/action-domain/index.vue'
import Ddnsto from '../components/action-ddnsto/index.vue'
import Disk from '../components/action-disk/index.vue'
import Part from '../components/action-disk-part/index.vue'
import Format from '../components/action-disk-format/index.vue'
import Mount from '../components/action-disk-mount/index.vue'
import Confirm from '../components/action-confirm/index.vue'
type Entry={id:number;kind:string;props:Record<string,any>;opener:HTMLElement|null}
const registry:Record<string,any>=Object.fromEntries(Object.entries({'action-nas':Nas,'action-nas/linkease':Linkease,'action-nas/unishare':Unishare,'action-docker':Docker,'action-download':Download,'action-domain':Domain,'action-ddnsto':Ddnsto,'action-disk':Disk,'action-disk-part':Part,'action-disk-format':Format,'action-disk-mount':Mount,'action-confirm':Confirm}).map(([k,v])=>[k,markRaw(v)]))
const entries=ref<Entry[]>([]),destination=ref<{href:string;pkg?:string;installPkg?:string}|null>(null),english=window.vue_lang==='en'
let sequence=0,overflow='',lock:string|null=null,alive=true
function close(id:number){const entry=entries.value.find(e=>e.id===id);entries.value=entries.value.filter(e=>e.id!==id);void nextTick(()=>{if(entries.value.length){document.body.setAttribute('lock-scroll','true');const nodes=document.querySelectorAll<HTMLElement>('.native-action-host');nodes[nodes.length-1]?.focus({preventScroll:true})}else{document.body.style.overflow=overflow;if(lock===null)document.body.removeAttribute('lock-scroll');else document.body.setAttribute('lock-scroll',lock);if(entry?.opener?.isConnected)entry.opener.focus({preventScroll:true})}})}
const unregister=registerNativeActions((kind,props)=>{
 if(!entries.value.length){overflow=document.body.style.overflow;lock=document.body.getAttribute('lock-scroll')}
 const id=++sequence;entries.value.push({id,kind,props:kind==='action-docker'?{rootPath:'',...props}:props,opener:document.activeElement as HTMLElement})
 document.body.style.overflow='hidden'
 void nextTick(()=>{const element=document.querySelector<HTMLElement>(`[data-native-action="${kind}"]:last-of-type`);element?.focus({preventScroll:true})})
 return {Close:()=>close(id)}
})
function showDestination(value:{href:string;pkg?:string;installPkg?:string}){if(/^\/storage(?:\?|$)/.test(value.href)||/\/admin\/(system\/(diskman|mounts)|nas\/raid)/.test(value.href)){const url=new URL(value.href,location.origin);window.dispatchEvent(new CustomEvent('quickstart-storage-open',{detail:{tab:url.searchParams.get('tab')||(value.href.includes('/nas/raid')?'raid':'disks'),part:url.searchParams.get('part')}}));return}destination.value=value}
function captureLink(event:MouseEvent){const anchor=(event.target as Element).closest('a[href]');if(!anchor)return;event.preventDefault();event.stopPropagation();showDestination({href:anchor.getAttribute('href')||''})}
function external(event:Event){showDestination((event as CustomEvent).detail)}
window.addEventListener('quickstart-business-destination',external)
const original=appUtils.installAndGo
const installAndGo=async(pkg:string,app:string,href:string,installPkg?:string)=>{if(await appUtils.checkAndInstallApp(pkg,app,installPkg)){if(alive)showDestination({href,pkg,installPkg})}}
appUtils.installAndGo=installAndGo
function key(event:KeyboardEvent,entry:Entry,rootOverride?:HTMLElement){
 if(entry.id!==entries.value[entries.value.length-1]?.id)return
 const root=rootOverride||event.currentTarget as HTMLElement
 if(event.key==='Escape'){event.preventDefault();event.stopPropagation();const cancel=root.querySelector<HTMLElement>('.app-back,.action-footer .clear');if(cancel)cancel.click();else{entry.props.Cancel?.();entry.props.clear?.();close(entry.id)}}
 if(event.key==='Tab'){const nodes=Array.from(root.querySelectorAll<HTMLElement>('button:not(:disabled),input:not(:disabled),select:not(:disabled),a[href],[tabindex="0"]')).filter(n=>n.getClientRects().length),first=nodes[0],last=nodes[nodes.length-1];if(!first){event.preventDefault();root.focus()}else if(event.shiftKey&&(document.activeElement===first||document.activeElement===root)){event.preventDefault();last.focus()}else if(!event.shiftKey&&(document.activeElement===last||document.activeElement===root)){event.preventDefault();first.focus()}}
}
function globalEscape(event:KeyboardEvent){if(event.key!=='Escape'||!entries.value.length||destination.value)return;const nodes=document.querySelectorAll<HTMLElement>('.native-action-host'),root=nodes[nodes.length-1];if(root){event.stopImmediatePropagation();key(event,entries.value[entries.value.length-1],root)}}
window.addEventListener('keydown',globalEscape,true)
onUnmounted(()=>{alive=false;unregister();window.removeEventListener('quickstart-business-destination',external);window.removeEventListener('keydown',globalEscape,true);if(appUtils.installAndGo===installAndGo)appUtils.installAndGo=original;if(entries.value.length){document.body.style.overflow=overflow;if(lock===null)document.body.removeAttribute('lock-scroll');else document.body.setAttribute('lock-scroll',lock)}})
</script>
<style>
.native-action-host{position:fixed;inset:0;font:14px Arial,"Microsoft YaHei",sans-serif;outline:none;color:#333}.native-action-host #actioner{z-index:1;padding:16px;box-sizing:border-box}.native-action-host .bg{z-index:0}.native-action-host .action{max-width:100%;max-height:calc(100dvh - 32px);box-sizing:border-box;overflow:auto}.native-action-host .action h2.title{font-size:22px!important}.native-action-host input,.native-action-host select,.native-action-host button{font-family:inherit!important;font-size:14px!important;max-width:100%;box-sizing:border-box}.native-action-host .action .desc{font:14px/1.7 Arial,"Microsoft YaHei",sans-serif!important}.native-destination{font:14px/1.6 Arial,"Microsoft YaHei",sans-serif}.native-destination code{display:block;white-space:pre-wrap;overflow-wrap:anywhere}.native-destination dd{margin:4px 0 16px;overflow-wrap:anywhere}
body:has(.native-destination) .network-dialog-shade:has(.native-destination){z-index:7000}
.native-action-host .action-main{max-width:100%;max-height:calc(100dvh - 32px);box-sizing:border-box;overflow:auto}
@media(max-width:768px){.native-action-host .action{width:100%!important;padding:20px!important}.native-action-host .action .select-editable,.native-action-host .action .tips,.native-action-host .action .change{max-width:100%;width:100%}.native-action-host .action .finished{margin:20px}.native-action-host .action .btns{margin-top:24px}.native-action-host .action .action-header_title{font-size:18px!important}}
</style>
