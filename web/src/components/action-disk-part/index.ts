import { createApp, Plugin } from 'vue'
import progress from "/@/components/progress/index.vue"
import compnonet from "./index.vue"
declare interface DiskManualProps {
    action?: string,
    disk: NasDiskModel,
    Cancel: () => void
    Next: (rootPath: string) => void
}
export default (props: DiskManualProps) => {
    const el = document.createElement("div")
    document.body.appendChild(el)
    const vm = createApp(compnonet, {
        ...props,
        Close: () => {
            Close()
        },
    })
    vm.component("progress-item", progress)
    vm.mount(el)
    const Close = () => {
        vm.unmount()
        el.remove()
    }
    return {
        Close
    }
}