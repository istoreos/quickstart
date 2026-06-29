import { createApp, Plugin } from 'vue'
import compnonet from "./index.vue"
declare module '@vue/runtime-core' {
    interface ComponentCustomProperties {
        $TTYD: ActionerTtyd
    }
}
declare type ActionerTtyd = (props: number) => {
    Close: () => void
}
const Actiom = <Plugin>{
    install(app, options) {
        const f: ActionerTtyd = (props) => {
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
        app.config.globalProperties.$TTYD = f
    }
}
export default Actiom