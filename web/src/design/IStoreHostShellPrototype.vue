<!-- Preview facade of the existing host, based on supplied screenshots. Never imported by production. -->
<template>
 <div class="istore-host-prototype" :class="{'host-menu-open':opened}">
  <div v-if="mobile&&opened" class="host-menu-shade" @click="closeMenu()" aria-hidden="true"></div>
  <aside class="host-system-menu" :class="{'host-mobile-open':opened}" :role="mobile?'dialog':undefined" :aria-modal="mobile&&opened?true:undefined" aria-label="iStoreOS 系统菜单" :aria-hidden="mobile&&!opened?true:undefined" :inert="pageBlocked||(mobile&&!opened)" tabindex="-1" ref="menuRoot">
   <header class="host-menu-brand"><span>iStoreOS</span><button v-if="mobile" aria-label="关闭系统菜单" @click="closeMenu()">×</button></header>
   <label class="host-search"><span class="host-sr-only">搜索菜单</span><input v-model="search" type="search" placeholder="搜索菜单" aria-label="搜索菜单" /></label>
   <nav aria-label="系统功能">
    <template v-for="item in filteredItems" :key="item.id">
     <button :class="{selected:item.id==='home'&&route.path==='/',expanded:groupOpen(item.id),current:item.id==='storage'&&storagePage}" :aria-expanded="item.children?groupOpen(item.id):undefined" @click="chooseItem(item)"><svg viewBox="0 0 24 24" fill="none" aria-hidden="true" :style="{color:item.color}"><path :d="item.icon" /></svg><span>{{item.label}}</span><svg v-if="item.children" class="host-chevron" viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg></button>
     <div v-if="item.children&&groupOpen(item.id)" class="host-submenu"><button v-for="child in item.children" :key="child.path" :class="{selected:childSelected(child.path)}" :aria-current="childSelected(child.path)?'page':undefined" @click="go(child.path)">{{child.label}}</button></div>
    </template>
    <p v-if="!filteredItems.length" class="host-no-results">没有匹配的菜单</p>
    <button v-if="!search" class="host-logout" disabled title="设计预览不执行退出登录"><svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M10 4H4v16h6M9 12h12m-5-5 5 5-5 5" /></svg><span>退出</span></button>
   </nav>
  </aside>
  <div class="host-page-frame" :inert="mobile&&opened">
   <header class="host-topbar" :inert="pageBlocked" data-host-topbar>
    <div class="host-mobile-brand"><button aria-label="打开系统菜单" :aria-expanded="opened" :disabled="pageBlocked" @click="openMenu" ref="menuButton"><svg viewBox="0 0 24 24" fill="none" aria-hidden="true"><path d="M3 6h18M3 12h18M3 18h18" /></svg></button><span>iStoreOS</span></div>
    <div class="host-top-actions"><a href="https://www.istoreos.com/" target="_blank" rel="noopener noreferrer">iStoreOS官网</a><ThemeSelector /><button v-if="route.path==='/'" class="host-home-settings" aria-label="自定义首页" title="自定义首页" @click="customizeHome"><svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="1.7"/><rect x="14" y="3" width="7" height="7" rx="1.7"/><rect x="3" y="14" width="7" height="7" rx="1.7"/><rect x="14" y="14" width="7" height="7" rx="1.7"/></svg></button></div>
   </header>
   <slot />
  </div>
 </div>
</template>
<script setup lang="ts">
import {computed,nextTick,onMounted,onUnmounted,ref,watch} from 'vue'
import {useRoute,useRouter} from 'vue-router'
import ThemeSelector from '../home/ThemeSelector.vue'
type MenuItem={id:string;label:string;icon:string;color:string;path?:string;children?:{label:string;path:string}[]}
const route=useRoute(),router=useRouter(),opened=ref(false),mobile=ref(false),search=ref(''),expanded=ref<string[]>([]),pageBlocked=ref(false),menuRoot=ref<HTMLElement>(),menuButton=ref<HTMLButtonElement>()
const storagePage=computed(()=>route.path==='/storage'||route.path.startsWith('/smart'))
const items:MenuItem[]=[
 {id:'home',label:'首页',icon:'M3 3h18v18H3zM7 16V9M12 16V6M17 16v-5',color:'#6571e5',path:'/'},
 {id:'wizard',label:'网络向导',icon:'M12 3a7 7 0 0 0-4 13v3h8v-3a7 7 0 0 0-4-13ZM9 22h6M9 11l3 3 3-3',color:'#f17c68',path:'/design/system/network-wizard'},
 {id:'status',label:'状态',icon:'M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z',color:'#6774ef',children:[{label:'运行状态',path:'/design/system/status'}]},
 {id:'system',label:'系统',icon:'M9 3h6l1 4 4 2v6l-4 2-1 4H9l-1-4-4-2V9l4-2 1-4ZM15 12a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z',color:'#ed8262',children:[{label:'系统设置',path:'/design/system/settings'}]},
 {id:'store',label:'iStore',icon:'M4 8h16v13H4zM8 8V6a4 4 0 0 1 8 0v2',color:'#9274dc',path:'/design/system/istore'},
 {id:'docker',label:'Docker',icon:'M3 13h17c0 6-4 8-8 8s-8-2-9-8ZM6 9h3v3H6zM10 9h3v3h-3zM14 9h3v3h-3zM10 5h3v3h-3z',color:'#3ca2e9',children:[{label:'容器管理',path:'/design/system/docker'}]},
 {id:'services',label:'服务',icon:'M4 8h6v6H4zM14 3h6v6h-6zM14 15h6v6h-6zM10 11h4M12 6h2M12 18h2M12 6v12',color:'#51bfd2',children:[{label:'服务设置',path:'/design/system/services'}]},
 {id:'storage',label:'网络存储',icon:'M21 5c0 2-4 3-9 3S3 7 3 5s4-3 9-3 9 1 9 3ZM3 5v14c0 2 4 3 9 3s9-1 9-3V5M3 12c0 2 4 3 9 3s9-1 9-3',color:'#df80a7',children:[{label:'硬盘与存储',path:'/storage'},{label:'S.M.A.R.T.',path:'/smart'}]},
 {id:'network',label:'网络',icon:'M9 3h6v6H9zM3 16h6v5H3zM15 16h6v5h-6zM12 9v4M6 16v-3h12v3',color:'#a481e5',children:[{label:'局域网管理',path:'/design/multi-lan'}]},
]
const menuMatch=(value:string)=>value.toLowerCase().replace(/[.\s]/g,'').includes(search.value.toLowerCase().replace(/[.\s]/g,''))
const filteredItems=computed(()=>items.filter(item=>!search.value||menuMatch(item.label)||item.children?.some(child=>menuMatch(child.label))))
function childSelected(path:string){return route.path===path||(path==='/smart'&&route.path.startsWith('/smart/'))}
function groupOpen(id:string){return !!search.value||expanded.value.includes(id)}
function chooseItem(item:MenuItem){if(item.children){expanded.value=expanded.value.includes(item.id)?expanded.value.filter(id=>id!==item.id):[...expanded.value,item.id]}else if(item.path)void go(item.path)}
async function go(path:string){closeMenu(false);const changed=route.path!==path;if(changed)await router.push(path);await nextTick();if(changed)window.scrollTo({top:0,behavior:'auto'});(document.querySelector<HTMLElement>('.sw-heading h1,.host-page-frame main h1')||menuButton.value)?.focus({preventScroll:true})}
function openMenu(){if(!mobile.value||pageBlocked.value)return;document.querySelectorAll('.host-topbar details[open]').forEach(el=>el.removeAttribute('open'));opener=document.activeElement as HTMLElement;opened.value=true}
function closeMenu(restore=true){opened.value=false;search.value='';if(restore)void nextTick(()=>{if(opener?.isConnected)opener.focus({preventScroll:true})})}
async function customizeHome(){if(pageBlocked.value)return;if(route.path!=='/')await router.push('/');await nextTick();document.querySelector<HTMLButtonElement>('[data-home-settings]')?.click()}
let opener:HTMLElement|null=null,overflow:string|undefined,query:MediaQueryList|undefined,observer:MutationObserver|undefined,frame=0
function queryChanged(){mobile.value=!!query?.matches;if(!mobile.value)closeMenu(false)}
function key(event:KeyboardEvent){if(!mobile.value||!opened.value)return;if(event.key==='Escape'){event.preventDefault();event.stopImmediatePropagation();closeMenu();return}if(event.key!=='Tab'||!menuRoot.value)return;const elements=Array.from(menuRoot.value.querySelectorAll<HTMLElement>('button:not(:disabled),input,a[href]')).filter(e=>e.getClientRects().length);const first=elements[0],last=elements[elements.length-1];if(event.shiftKey&&(document.activeElement===first||document.activeElement===menuRoot.value)){event.preventDefault();last?.focus()}else if(!event.shiftKey&&(document.activeElement===last||document.activeElement===menuRoot.value)){event.preventDefault();first?.focus()}}
function inspectPage(){frame=0;pageBlocked.value=Array.from(document.querySelectorAll<HTMLElement>('[role="dialog"],#actioner,.direct-panel,.ordered-library')).some(el=>!el.closest('.host-system-menu')&&el.getClientRects().length>0&&el.getAttribute('aria-hidden')!=='true')||!!document.querySelector('.ordered-editing');if(pageBlocked.value&&opened.value)closeMenu(false)}
watch(opened,async value=>{if(value){overflow=document.body.style.overflow;document.body.style.overflow='hidden';await nextTick();menuRoot.value?.focus()}else if(overflow!==undefined){document.body.style.overflow=overflow;overflow=undefined}})
watch(()=>route.path,path=>{if((path==='/storage'||path.startsWith('/smart'))&&!expanded.value.includes('storage'))expanded.value.push('storage');if(path==='/design/multi-lan'&&!expanded.value.includes('network'))expanded.value.push('network');closeMenu(false)},{immediate:true})
onMounted(()=>{query=matchMedia('(max-width:760px)');queryChanged();query.addEventListener('change',queryChanged);document.addEventListener('keydown',key,true);observer=new MutationObserver(()=>{if(!frame)frame=requestAnimationFrame(inspectPage)});observer.observe(document.body,{subtree:true,childList:true,attributes:true,attributeFilter:['class','aria-hidden','open']});inspectPage()})
onUnmounted(()=>{query?.removeEventListener('change',queryChanged);observer?.disconnect();if(frame)cancelAnimationFrame(frame);document.removeEventListener('keydown',key,true);if(overflow!==undefined)document.body.style.overflow=overflow})
</script>
<style src="./istoreHostShellPrototype.css"></style>
