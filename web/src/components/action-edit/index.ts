import { createApp, Plugin } from 'vue'
import compnonet from "./index.vue"
declare type ActionEdit = (props?: {
    e: "edit" | "add"
    name: FirewallType,
    inface: NetworkInterfaceInfo,
    next: (inface: NetworkInterfaceInfo) => void
}) => void
const Action: ActionEdit = (props) => {
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
}
export default Action


