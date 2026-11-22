import { createApp } from 'vue'
import {initializeTheme,normalizeThemePreference,setThemePreference} from '../utils/theme'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import DesignApp from './DesignApp.vue'
import HomePreview from './HomePreview.vue'
import DevicePreview from './DevicePreview.vue'
import MultiLanPreview from './MultiLanPreview.vue'
import DestinationPreview from './DestinationPreview.vue'
import ExternalCardPreview from './ExternalCardPreview.vue'
import LighthostStoragePrototype from './LighthostStoragePrototype.vue'
import StorageWorkbenchPrototype from './StorageWorkbenchPrototype.vue'
import SmartManagement from '../pages/smart/index.vue'
import SmartSettings from '../pages/smart/index/index.vue'
import SmartTasks from '../pages/smart/task/index.vue'
import SmartLogs from '../pages/smart/log/index.vue'
import SmartActions from '../components/smart'
import './smartWorkbenchPreview.css'
import { createI18n } from '../plugins/i18n'
import installComponents from '../plugins/componnets'
import '../style/root.scss'
import '../App.vue' // retain the real app's global component styles
import './shell.css'
import SvgMenu from '../components/svg/menu.vue'
import SvgSystem from '../components/svg/system.vue'
import SvgDownload from '../components/svg/download.vue'
import SvgStore from '../components/svg/store.vue'
import SvgInfo from '../components/svg/info.vue'
import SvgDisk from '../components/svg/disk.vue'
import SvgNav from '../components/svg/nav.vue'
import Progress from '../components/progress/index.vue'
import SwitchBox from '../components/switch/index.vue'
import SvgShow from '../components/svg/view-show.vue'
import SvgHidden from '../components/svg/view.hidden.vue'
import ArticleItem from '../components/article/item.vue'
import EditableSelect from '../components/editable-select/index.vue'

export async function mount() {
  initializeTheme()
  const explicit=normalizeThemePreference(new URLSearchParams(location.search).get('theme'))
  if(explicit)setThemePreference(explicit)
  const app = createApp(DesignApp)
  const components = { 'svg-menu':SvgMenu,'svg-system':SvgSystem,'svg-download':SvgDownload,'svg-store':SvgStore,'svg-info':SvgInfo,'svg-disk':SvgDisk,'svg-nav':SvgNav,'progress-item':Progress,'switch-box':SwitchBox,'svg-view-show':SvgShow,'svg-view-hidden':SvgHidden,'article-item':ArticleItem,'editable-select':EditableSelect }
  Object.entries(components).forEach(([name,component])=>app.component(name,component))
  app.use(installComponents).use(SmartActions).use(createPinia())
  const router=createRouter({ history:createWebHistory('/'),routes:[{path:'/design/external-card',component:ExternalCardPreview},{path:'/',component:HomePreview},{path:'/devicemanagement',component:DevicePreview},{path:'/design/multi-lan',component:MultiLanPreview},{path:'/design/fastnet',redirect:to=>({path:'/',query:to.query})},{path:'/:pathMatch(.*)*',component:DestinationPreview}] })
  router.beforeEach((to,from)=>{
    const keys=['layout','variant','scenario','authority','theme','lang','tools','palette','presentation']
    const query={...to.query};let changed=false
    for(const key of keys)if(query[key]===undefined&&from.query[key]!==undefined){query[key]=from.query[key];changed=true}
    if(changed)return {path:to.path,query,replace:true}
    return true
  })
  router.addRoute({path:'/design/storage-setup',component:LighthostStoragePrototype})
  router.addRoute({path:'/storage',component:StorageWorkbenchPrototype})
  router.addRoute({path:'/raid',redirect:to=>({path:'/storage',query:{...to.query,tab:'raid'}})})
  router.addRoute({path:'/smart',component:SmartManagement,children:[{path:'',component:SmartSettings},{path:'task',component:SmartTasks},{path:'log',component:SmartLogs}]})
  router.addRoute({path:'/nas',redirect:to=>({path:'/storage',query:to.query})})
  window.addEventListener('quickstart-storage-open',(event:Event)=>{const detail=(event as CustomEvent).detail||{};void router.push({path:'/storage',query:{tab:detail.tab||'disks',part:detail.part,variant:'A'}})})
  app.use(router)
  await createI18n(app)
  app.mount('#app')
}
