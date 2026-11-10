<template>
  <aside class="design-system-reserve" data-system-menu-reserve aria-label="系统菜单占位，不属于设计范围">
    <strong>系统菜单</strong><span>保持原样</span>
    <p>仅预留宿主空间<br>不是菜单设计</p>
    <small>菜单项、顺序、样式及折叠行为均由系统提供。</small>
  </aside>
  <div class="design-content-region" data-quickstart-content>
  <aside class="design-tools" v-if="legacy && tools">
    <strong>Quickstart Vue 设计预览 · 示例数据</strong>
    <label>布局 <select aria-label="布局" :value="route.query.layout || 'baseline'" @change="change('layout',$event)"><option value="baseline">当前首页</option><option value="below-network">设备摘要：网络区下方（推荐）</option><option value="above-network">设备摘要：网络区上方</option></select></label>
    <label>场景 <select aria-label="场景" :value="demo.scenario" @change="change('scenario',$event)"><option v-for="item in scenarios" :key="item[0]" :value="item[0]">{{item[1]}}</option></select></label>
    <label>地址分配 <select aria-label="地址分配" :value="demo.authority" @change="change('authority',$event)"><option value="external">外部分配</option><option value="local">本机分配</option><option value="unknown">尚不明确</option></select></label>
    <ThemeSelector/><button @click="language">{{english?'中文':'English'}}</button><button @click="tools=false">收起评审工具</button>
    <span>真实 Vue/CSS · 模拟 LuCI 外壳 · 操作仅作用于示例</span>
  </aside>
  <button class="design-reopen" v-else-if="legacy && route.query.tools!=='0'" @click="tools=true">设计工具</button>
  <details v-if="!legacy&&tools&&route.path==='/'" class="card-preview-tools home-semantic-colors" data-card-preview-tools><summary>设计预览 · FastNet</summary><label>卡片场景<select aria-label="FastNet 场景" :value="previewScene" @change="changeCardScene"><option v-for="scene in cardScenes" :key="scene[0]" :value="scene[0]">{{scene[1]}}</option></select></label><p>模拟接口，不执行真实安装或测速。</p><button @click="tools=false">收起工具</button></details>
  <button v-if="!legacy&&!tools&&route.query.tools!=='0'" class="card-preview-reopen" @click="tools=true">设计工具</button>
  <div class="qs-product" :class="{'studio-shell':!legacy && route.path==='/'}"><App /></div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import App from '../App.vue'
import ThemeSelector from '../home/ThemeSelector.vue'
import { demo } from './fixtures'
import { useNetworkStore } from '../plugins/store'
const tools=ref(new URLSearchParams(location.search).get('tools')!=='0')
const route=useRoute(),router=useRouter()
const legacy=computed(()=>['baseline','above-network','below-network'].includes(String(route.query.layout||'studio')))
const english=ref(window.vue_lang==='en')
const cardScenes=[['uninstalled','未安装'],['ready','上次结果'],['empty','尚未测速'],['stale','旧结果'],['running','运行中'],['failed','测速失败'],['partial','部分检测异常'],['disabled','服务未启用'],['permission','登录/权限失败'],['poll-error','查询连续失败'],['install-error','安装失败'],['discovery-error','发现失败'],['incompatible','需要升级 Quickstart']]
const previewScene=computed(()=>route.query.external==='present'?String(route.query.scene||'ready'):String(route.query.scene||route.query.external||'uninstalled'))
function changeCardScene(event:Event){const scene=(event.target as HTMLSelectElement).value,url=new URL(location.href);url.searchParams.delete('theme');url.searchParams.delete('fastnet');if(['uninstalled','install-error'].includes(scene)){url.searchParams.delete('external');url.searchParams.set('scene',scene==='uninstalled'?'empty':scene)}else if(['incompatible','discovery-error'].includes(scene)){url.searchParams.set('external',scene);url.searchParams.delete('scene')}else{url.searchParams.set('external','present');url.searchParams.set('scene',scene)}location.assign(url.href)}
const scenarios=[['normal','正常'],['many','128 台设备'],['empty','没有设备'],['loading','首次加载'],['error','设备 API 失败'],['stale','过期快照'],['old','旧后端'],['permission','登录/权限失败'],['no-speed','服务未安装'],['disabled','服务未启用'],['speed-error','服务异常'],['unsupported','固件不支持']]
const change=(key:string,event:Event)=>{const value=(event.target as HTMLSelectElement).value;demo.recovered=false;router.replace({query:{...route.query,[key]:value,device:undefined}})}
const language=()=>{location.href=router.resolve({path:route.path,query:{...route.query,lang:english.value?'zh-cn':'en'}}).href}
watch(()=>route.query,()=>{demo.layout=String(route.query.layout||'studio');demo.authority=String(route.query.authority||'external');demo.scenario=String(route.query.scenario||'normal');if(legacy.value)useNetworkStore().requestDeviceList()})
</script>

<style scoped>
.card-preview-tools,.card-preview-reopen{position:fixed;bottom:12px;right:12px;z-index:120;background:var(--qs-card-panel);color:var(--qs-card-text);border:1px solid var(--qs-card-line);border-radius:8px;padding:8px 12px;font-size:12px}.card-preview-tools summary{cursor:pointer}.card-preview-tools label{display:grid;gap:6px;margin-top:12px}.card-preview-tools select{max-width:240px;background:var(--qs-card-panel);color:var(--qs-card-text);padding:6px;border:1px solid var(--qs-card-line)}.card-preview-tools p{color:var(--qs-card-muted);font-size:11px}.card-preview-tools button{background:var(--qs-card-soft);color:inherit;border:1px solid var(--qs-card-line);border-radius:4px}
</style>
