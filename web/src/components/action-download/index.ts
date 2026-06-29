import { createApp, Plugin } from 'vue'
import compnonet from "./index.vue"
declare type ActionDownload = (props?: any) => {
    Close: () => void
}
const Action: ActionDownload = (props?: any) => {
    const el = document.createElement("div")
    document.body.appendChild(el)
    const vm = createApp(compnonet, {
        ...props,
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