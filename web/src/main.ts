import { createApp } from 'vue'
import App from './App.vue'
import "/@/style/root.scss"

import SvgMenu from "/@/components/svg/menu.vue"
import SvgSytem from "/@/components/svg/system.vue"
import SvgDownload from "/@/components/svg/download.vue"
import SvgStore from "/@/components/svg/store.vue"
import SvgInfo from "/@/components/svg/info.vue"
import SvgDisk from "/@/components/svg/disk.vue"
import SvgNav from "/@/components/svg/nav.vue"
import progress from "/@/components/progress/index.vue"
import SwitchBox from "/@/components/switch/index.vue"
import SvgViewShow from "/@/components/svg/view-show.vue"
import SvgViewHidden from "/@/components/svg/view.hidden.vue"
import ArticleItem from "/@/components/article/item.vue"
import EditableSelect from "/@/components/editable-select/index.vue"

import installComponents from "/@/plugins/componnets"
import { createI18n } from '/@/plugins/i18n'

const app = createApp(App)
app.component("svg-menu", SvgMenu)
app.component("svg-system", SvgSytem)
app.component("svg-download", SvgDownload)
app.component("svg-store", SvgStore)
app.component("svg-info", SvgInfo)
app.component("svg-disk", SvgDisk)
app.component("svg-nav", SvgNav)
app.component("progress-item", progress)
app.component("svg-view-show", SvgViewShow)
app.component("svg-view-hidden", SvgViewHidden)
app.component("article-item", ArticleItem)
app.component("switch-box", SwitchBox)
app.component("editable-select", EditableSelect)
app.use(installComponents) //注册全局组件
import { createPinia } from 'pinia'
import { router } from "/@/plugins/router"
app.use(router)
app.use(createPinia())
createI18n(app)
    .finally(()=>app.mount('#app'))
