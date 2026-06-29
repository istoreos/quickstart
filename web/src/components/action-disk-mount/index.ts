import { createApp, Plugin } from 'vue'
import compnonet from "./index.vue"
declare interface DiskMountProps {
    action?: string,
    disk: NasDiskModel,
    mount?: MountPoint,
    Cancel: () => void
    Next: (rootPath: string) => void
}
export default (props: DiskMountProps) => {
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