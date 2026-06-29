import { createApp, Plugin } from 'vue'
import compnonet from "./index.vue"
import installComponents from "/@/plugins/componnets"
const Action: ActionRaid = (props) => {
    const el = document.createElement("div")
    document.body.appendChild(el)
    const vm = createApp(compnonet, {
        ...props,
        Close: () => {
            Close()
        },
    })
    vm.use(installComponents) //注册全局组件
    vm.mount(el)
    const Close = () => {
        vm.unmount()
        el.remove()
    }
    return {
        Close
    }
}
export default Action