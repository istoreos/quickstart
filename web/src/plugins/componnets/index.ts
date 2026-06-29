import { App } from "vue";
import IconLoading from "/@/components/icons/loading.vue"
import IconSuccess from "/@/components/icons/success.vue"
import IconError from "/@/components/icons/error.vue"
import HelpVue from "/@/components/help/index.vue"

declare module '@vue/runtime-core' {
    export interface GlobalComponents {
        IconLoading: typeof IconLoading,
        IconSuccess: typeof IconSuccess,
        IconError: typeof IconError
        GlHelp: typeof HelpVue,
    }
}
// 注册全局组件
export default {
    install: (app: App) => {
        app.component("icon-loading", IconLoading)
        app.component("icon-success", IconSuccess)
        app.component("icon-error", IconError)
        app.component("GlHelp", HelpVue)
    }
}