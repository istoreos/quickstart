import { createApp, Plugin } from 'vue'
import compnonet from "./index.vue"
declare type ActionDns = (props?: any) => {
    Close: () => void
}
const Action: ActionDns = () => {
    const el = document.createElement("div")
    document.body.appendChild(el)
    const vm = createApp(compnonet, {
        Close: () => {
            Close()
        },
    })
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