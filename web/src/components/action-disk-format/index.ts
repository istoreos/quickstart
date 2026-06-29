import { createApp, Plugin } from 'vue'
import compnonet from "./index.vue"
declare interface DiskFormatProps {
    action?: string,
    disk: NasDiskModel,
    mount?: MountPoint | null,
    Cancel: () => void
    Next: (rootPath: string) => void
}
export default (props: DiskFormatProps) => {
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