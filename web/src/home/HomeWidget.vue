<template>
  <div class="layout-widget" ref="element" :inert="editing" :class="['widget-'+id,{narrow:width<450,medium:width<650,short:!natural&&height<310}]" :data-layout-card="id" :data-natural-card="natural||undefined" :data-presentation="resolved.id" :data-chart-stretch="id==='traffic'&&chartStretch||undefined">
    <template v-if="id==='traffic'"><NetworkFlow v-if="nativeVisible" :key="theme"/><div class="traffic-header-actions widget-action-group"><WidgetPrimaryAction icon="activity" class="layout-speed" data-home-speed @click="speedTest" :disabled="speedBusy" :aria-busy="speedBusy" :aria-label="t('网络测速','Network speed test')">{{t('测速','Test')}}</WidgetPrimaryAction></div><span v-if="speedBusy" class="speed-announcement" role="status">{{t('正在打开FastNet','Opening FastNet')}}</span><div v-if="speedError" class="speed-error" role="status" data-speed-error><span>{{launcherActivation.status==='timeout'?t('打开测速超时，请重试。','Opening speed test timed out. Please retry.'):launcherActivation.status==='permission'?t('无访问权限，请重新登录后重试。','No permission. Sign in and retry.'):t('暂时无法打开测速，请重试。','Unable to open speed test. Please retry.')}}</span><div><button @click="speedTest" data-speed-retry>{{t('重试','Retry')}}</button><button @click="dismissSpeed" data-speed-dismiss :aria-label="t('关闭提示','Dismiss')">×</button></div></div><span v-if="errors.traffic" class="layout-data-notice">{{t('流量采样暂不可用','Traffic sampling unavailable')}}</span></template>
    <NetworkConnectionCard v-else-if="id==='connection'" :data="data.network" :error="errors.network" :summary="resolved.id==='summary'" @navigate="$emit('navigate',$event)" @retry="$emit('retry')"/>
    <NetworkInterfaceCard :legacy-summary="resolved.legacy" v-else-if="id==='networkInterface'" :data="data.ports" :error="errors.ports" :summary="resolved.id==='summary'" @navigate="$emit('navigate',$event)" @retry="$emit('retry')"/>
    <HomeApplications v-else-if="id==='applications'" :collapsed="false" :fallback-entries="[]" :contained="contained" :preference="preference" :natural="natural" :network="data.network" @navigate="$emit('navigate',$event)"/>
    <BusinessCard v-else-if="['storage','sharing','docker','downloads','remote','system'].includes(id)" :id="id" :data="data" :errors="errors" :summary="resolved.id==='summary'" @retry="$emit('retry')"/>
    <template v-else-if="id.startsWith('plugin.')"><CardShell v-if="externalManifest&&externalManifest.forms[resolved.id as 'standard'|'summary']" :key="externalManifest.entry" :manifest="externalManifest" :form="resolved.id==='summary'?'summary':'standard'" :rediscover="rediscover"/><section v-else class="qs-card-placeholder" data-missing-plugin role="status"><h2>{{externalName}}</h2><p>{{externalManifest?t('此形态暂不可用，请在显示设置中调整','This form is unavailable. Change display settings.'):registry.state.value==='unavailable'?t('无法确认插件状态，请重试读取','Plugin status unavailable; reload to retry'):needsHostUpdate(id)?t('此卡片需要升级 Quickstart，原位置已保留','Update Quickstart to display this card. Position retained.'):t('插件未安装或需要升级，原位置已保留','Plugin missing or needs updating. Position retained.')}}</p><button @click="registry.discover(true)">{{t('重新读取','Reload')}}</button></section></template>
    <DeviceWidget v-else :data="data" :errors="errors" :devices="devices" :summary="resolved.id==='summary'" :summary-limit="id==='devices'?resolved.capacity:undefined" :disk-limit="contained?4:undefined" contextual-actions @navigate="$emit('navigate',$event)" @device="$emit('device',$event)" @retry="$emit('retry')"/>
  </div>
</template>
<script setup lang="ts">
import WidgetPrimaryAction from './WidgetPrimaryAction.vue'
import {needsHostUpdate} from './cards/catalog'
import CardShell from './cards/CardShell.vue'
import {useCardRegistry} from './cards/useCardRegistry'
import {widget} from './layout/rowLayout'
import {computed,onMounted,onUnmounted,provide,ref} from 'vue'
import type {DeviceListItem} from '../pages/device/deviceInventory'
import type {WidgetId} from './layout/rowLayout'
import NetworkFlow from '../pages/index/network/NetworkFlow.vue'
import DeviceWidget from './DeviceWidget.vue'
import {launcherActivation,requestLauncherApp,cancelLauncherApp} from './launcherBrowserHost'
import {homeApplications} from './homeApplications'
import {widgetContext} from './widgetContext'
import HomeApplications from './HomeApplications.vue'
import NetworkConnectionCard from './NetworkConnectionCard.vue'
import NetworkInterfaceCard from './NetworkInterfaceCard.vue'
import BusinessCard from './BusinessCard.vue'
import {resolvePresentation,type WidgetPreference} from './layout/widgetPresentation'
const props=defineProps<{id:WidgetId;data:Record<string,any>;errors:Record<string,string>;devices:DeviceListItem[];theme:string;contained?:boolean;preference?:WidgetPreference;natural?:boolean;role?:'ordinary'|'stacked';chartStretch?:boolean;editing?:boolean}>()
const registry=useCardRegistry(),externalManifest=computed(()=>registry.cards.value.find(c=>c.id===props.id)),externalName=computed(()=>window.vue_lang==='en'?widget(props.id).en:widget(props.id).zh)
async function rediscover(){return (await registry.discover(true)).find(c=>c.id===props.id)}
provide(widgetContext,{data:props.data,bounded:()=>true,details:()=>{},invalidate:()=>emit('retry')})
const speedBusy=computed(()=>launcherActivation.source==='traffic'&&launcherActivation.status==='loading'),speedError=computed(()=>launcherActivation.source==='traffic'&&!['idle','loading'].includes(launcherActivation.status))
function speedTest(){requestLauncherApp(homeApplications.find(a=>a.id==='fastnet')!,'traffic')}
function dismissSpeed(){cancelLauncherApp();element.value?.querySelector<HTMLElement>('[data-home-speed]')?.focus({preventScroll:true})}
const emit=defineEmits<{(e:'navigate',path:string):void;(e:'device',id:string):void;(e:'retry'):void}>()
const english=window.vue_lang==='en',t=(zh:string,en:string)=>english?en:zh
const element=ref<HTMLElement|null>(null),width=ref(600),height=ref(320)
const resolved=computed(()=>resolvePresentation(props.id,props.preference||{mode:'auto',iconRows:0},{width:width.value,height:props.natural?100000:height.value,role:props.role||'ordinary',favoriteCount:4,mobile:width.value<650,natural:props.natural}))
// Design container owns visibility. The reused native chart has delayed setup;
// unmounting while hidden also prevents a not-yet-started initial sample.
const nativeVisible=ref(!document.hidden),visibility=()=>nativeVisible.value=!document.hidden
let observer:ResizeObserver|undefined
onMounted(()=>{if(props.id==='traffic')document.addEventListener('visibilitychange',visibility);observer=new ResizeObserver(entries=>{width.value=entries[0].contentRect.width;height.value=entries[0].contentRect.height});if(element.value)observer.observe(element.value)})
onUnmounted(()=>{observer?.disconnect();document.removeEventListener('visibilitychange',visibility)})
</script>
<style src="./layoutPreviewCard.css" scoped></style>
<style scoped>
[data-home-speed][aria-busy=true]{opacity:.65;cursor:progress}.speed-announcement{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}.speed-error{position:absolute;top:54px;right:14px;z-index:30;max-width:calc(100% - 28px);box-sizing:border-box;padding:12px;border:1px solid var(--c-line);border-radius:8px;background:var(--c-panel);color:var(--c-text);box-shadow:0 8px 24px #0002;font:14px/1.5 Arial}.speed-error>div{display:flex;justify-content:flex-end;gap:8px;margin-top:8px}.speed-error button{font:inherit;color:inherit;background:var(--c-soft);border:1px solid var(--c-line);border-radius:4px;min-height:32px;padding:4px 10px;cursor:pointer}@media(pointer:coarse){.speed-error button{min-height:44px;min-width:44px}}
</style>
