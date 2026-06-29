import { createApp, Plugin } from 'vue'
import compnonet from "./index.vue"

declare type Actioner = (props?: {
    next?: () => void
    clear?: () => void
    continuer?: () => void
    nextTitle?: string
    clearTitle?: string
    title?: string
    content?: string
    continuerTitle?: string
}) => {
    Close: () => void
}
const Action: Actioner = (props?: any) => {
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