import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import DesignApp from './DesignApp.vue'
import HomePreview from './HomePreview.vue'
import DevicePreview from './DevicePreview.vue'
import MultiLanPreview from './MultiLanPreview.vue'
import DestinationPreview from './DestinationPreview.vue'
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
  const app = createApp(DesignApp)
  const components = { 'svg-menu':SvgMenu,'svg-system':SvgSystem,'svg-download':SvgDownload,'svg-store':SvgStore,'svg-info':SvgInfo,'svg-disk':SvgDisk,'svg-nav':SvgNav,'progress-item':Progress,'switch-box':SwitchBox,'svg-view-show':SvgShow,'svg-view-hidden':SvgHidden,'article-item':ArticleItem,'editable-select':EditableSelect }
  Object.entries(components).forEach(([name,component])=>app.component(name,component))
  app.use(installComponents).use(createPinia())
  const router=createRouter({ history:createWebHistory('/'),routes:[{path:'/',component:HomePreview},{path:'/devicemanagement',component:DevicePreview},{path:'/design/multi-lan',component:MultiLanPreview},{path:'/:pathMatch(.*)*',component:DestinationPreview}] })
  router.beforeEach((to,from)=>{
    const keys=['layout','variant','scenario','authority','theme','lang','tools','palette']
    const query={...to.query};let changed=false
    for(const key of keys)if(query[key]===undefined&&from.query[key]!==undefined){query[key]=from.query[key];changed=true}
    if(changed)return {path:to.path,query,replace:true}
    return true
  })
  app.use(router)
  await createI18n(app)
  app.mount('#app')
}
