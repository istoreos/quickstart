<template>
 <HomeLayout :bootstrapping="!initialReady" :config="config.layout.value" :data="model.data" :errors="model.errors" :theme="theme" :favorite-ids="config.launcher.value.favorites" :can-edit="config.layoutEditable.value" :external-busy="config.state.launcher.saving" :persist="saveLayout" @editing="layoutEditing" @visible="previewIds=$event" @navigate="navigate" @device="openDevice">
  <template #status><div v-for="domain in domains" :key="domain" class="home-config-state" :data-config-domain="domain" role="status" v-show="initialReady&&!dismissed[domain]&&!config.state[domain].loading&&(config.state[domain].error||config.state[domain].snapshot?.state!=='saved')">
   <span class="home-config-message">{{statusText(domain)}}</span><div class="home-config-actions"><button type="button" v-if="!config.state[domain].loading&&!config.state[domain].saving" data-retry-config @click="config.read(domain)">{{t('重新读取','Reload')}}</button><button type="button" v-if="!editing&&['backup','damaged'].includes(config.state[domain].snapshot?.state||'')" :disabled="config.state[domain].saving" @click="recover(domain)">{{config.state[domain].snapshot?.state==='backup'?t('恢复有效备份','Restore valid backup'):t('保留损坏文件并恢复默认','Keep damaged file and restore defaults')}}</button></div><button type="button" class="home-config-close" data-dismiss-config-notice :aria-label="domain==='layout'?t('关闭首页布局提示','Dismiss homepage layout notice'):t('关闭应用工具提示','Dismiss apps and tools notice')" @click="dismissed[domain]=true"><svg viewBox="0 0 20 20" aria-hidden="true"><path d="m5 5 10 10M15 5 5 15"/></svg></button>
  </div></template>
  <template #widget="slot"><HomeWidget v-bind="slot" :data="model.data" :errors="model.errors" :devices="model.devices.value" :theme="theme" contained @navigate="navigate" @device="openDevice" @retry="model.invalidate(slot.id)"/></template>
  <template #launcher><HomeApplications collapsed host-only :fallback-entries="[]" :network="model.data.network" :return-ready="returnReady" @navigate="navigate"/></template>
 </HomeLayout>
</template>
<script setup lang="ts">
import {computed,onMounted,onUnmounted,reactive,ref,watch} from 'vue'
import {useRouter} from 'vue-router'
import {provide} from 'vue'
import {cardInstallerKey} from '/@/home/cards/installContext'
import {useCardRegistry} from '/@/home/cards/useCardRegistry'
import HomeWidget from '/@/home/HomeWidget.vue'
import HomeApplications from '/@/home/HomeApplications.vue'
// Load the shared shell after its views, preserving the accepted override order.
import HomeLayout from '/@/home/HomeLayout.vue'
import {useHomeConfig} from '/@/home/useHomeConfig'
import {useHomeData} from '/@/home/useHomeData'
import {provideLauncher} from '/@/home/launcherContext'
import type {WidgetId} from '/@/home/layout/rowLayout'
import {visibleIds} from '/@/home/layout/orderedLayout'
import {getTheme,subscribeTheme} from '/@/utils/theme'
import type {ConfigDomain,LayoutConfig} from '/@/home/configContract'
provide(cardInstallerKey,{inspect:(id,signal)=>import('/@/home/cards/installClient').then(m=>m.cardInstaller.inspect(id,signal)),install:(id,signal)=>import('/@/home/cards/installClient').then(m=>m.cardInstaller.install(id,signal))})
const previewIds=ref<WidgetId[]|null>(null)
const config=useHomeConfig(),launcher=provideLauncher(config),model=useHomeData(computed(()=>previewIds.value||visibleIds(config.layout.value.rows)))
// Only the two bounded homepage settings gate the first layout. Business data
// and the installed catalogue keep their independent, parallel loading owners.
// Latch this once: explicit rereads must not replace the page with a skeleton.
const initialReady=ref(false)
watch(()=>!config.state.layout.loading&&!config.state.launcher.loading,ready=>{if(ready)initialReady.value=true},{immediate:true})
const router=useRouter(),editing=ref(false),theme=ref(getTheme()),domains:ConfigDomain[]=['layout','launcher']
// Dismiss only this message instance. Read errors and save guards stay owned
// by useHomeConfig; no browser storage hides a fresh read or a new error.
const dismissed=reactive<Record<ConfigDomain,boolean>>({layout:false,launcher:false})
for(const domain of domains)watch(()=>[config.state[domain].loading,config.state[domain].error,config.state[domain].snapshot?.state,config.state[domain].snapshot?.version],()=>{dismissed[domain]=false})
const t=(zh:string,en:string)=>window.vue_lang==='en'?en:zh
function statusText(domain:ConfigDomain){const s=config.state[domain];return (domain==='layout'?t('首页布局：','Homepage: '):t('应用工具：','Apps & tools: '))+(s.loading?t('正在校验路由器配置…','Checking router configuration…'):s.error||({backup:t('原文件异常，临时使用有效备份；请恢复后编辑。','Original file invalid. Showing valid backup; restore before editing.'),damaged:t('配置损坏，临时显示默认内容，未覆盖原文件。','Configuration damaged. Showing defaults without replacing the file.'),future:t('配置来自更新版本，当前页面只读。','Configuration requires a newer version. This page is read only.'),missing:t('配置尚未初始化，请重试。','Configuration not initialized. Retry.')} as Record<string,string>)[s.snapshot?.state||'']||'')}
function recover(domain:ConfigDomain){if(window.confirm(t('将保留损坏文件，恢复此项配置；其他设置不变。确定继续？','Keep the damaged file and restore this setting? Other settings are unchanged.')))void config.recover(domain)}
function layoutEditing(value:boolean){editing.value=value;if(value)config.begin('layout')}
const saveLayout=(value:LayoutConfig)=>config.save('layout',value)
function returnReady(widgets:readonly string[]){return !config.state.layout.loading&&model.initialSettled(widgets)}
function navigate(path:string){if(editing.value)return;if(path.startsWith('/cgi-bin/'))window.location.assign(path);else void router.push(path)}
function openDevice(id:string){if(!editing.value)void router.push({path:'/devicemanagement',query:{device:id}})}
let unsubscribeTheme=()=>{}
onMounted(()=>{void useCardRegistry().discover();unsubscribeTheme=subscribeTheme(snapshot=>theme.value=snapshot.effective)})
onUnmounted(()=>unsubscribeTheme())
</script>
<style scoped>
.home-config-state{max-width:1640px;margin:12px auto;padding:4px 6px 4px 14px;border:1px solid var(--c-line);border-radius:8px;color:var(--c-muted);font-size:13px;display:flex;align-items:center;gap:8px}.home-config-message{flex:1;min-width:0;line-height:1.6;overflow-wrap:anywhere}.home-config-actions{display:flex;align-items:center;flex-wrap:wrap;gap:4px}.home-config-state button{min-height:44px;min-width:44px;padding:8px;background:none;border:0;border-radius:6px;color:var(--c-accent);font:inherit;cursor:pointer}.home-config-state button:focus-visible{outline:2px solid var(--c-accent);outline-offset:1px}.home-config-state .home-config-close{flex:none;color:var(--c-muted);display:grid;place-items:center}.home-config-close:hover{background:var(--c-soft)}.home-config-close svg{width:18px;height:18px;fill:none;stroke:currentColor;stroke-width:1.5;stroke-linecap:round}@media(max-width:600px){.home-config-state{display:grid;grid-template-columns:minmax(0,1fr) 44px;gap:0 4px}.home-config-actions{grid-column:1;grid-row:2}.home-config-close{grid-column:2;grid-row:1;align-self:start}}
</style>
