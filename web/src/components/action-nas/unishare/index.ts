import { createApp } from 'vue'
import compnonet from "./index.vue"

declare interface NasUniShareProps {
    rootPath: string
}

export default (props: NasUniShareProps) => {
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
