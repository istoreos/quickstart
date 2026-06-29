import { createApp, onUnmounted, ref } from "vue"
import Compnonet from "./index.vue"
const Values = new Map()
const ToastComponent = (props: any) => {
    const app = createApp(Compnonet, {
        ...props,
        Close: () => {
            Close()
        },
    })
    const el: Element = document.createElement("div")
    document.body.append(el)
    app.mount(el)
    const Close = () => {
        el.remove()
        if (Values.get(app._uid)) {
            Values.delete(app._uid)
        }
    }
    if (props.type == "loading") {
        Values.set(app._uid, {
            Close: Close
        })
    }
    if (props?.duration == 0) {
    } else if (props?.duration > 0) {
        setTimeout(() => {
            Close()
        }, props?.duration)
    } else {
        setTimeout(() => {
            Close()
        }, 3000)
    }
    return {
        Close,

    }
}
const Toast = (props: any) => {
    return ToastComponent(props)
}
Toast.Loading = (msg: string, countdown?: number) => {
    return ToastComponent({
        type: "loading",
        message: msg || "加载中...",
        duration: 0,
        countdown: countdown || 0,
    })
}
Toast.Success = (msg: string) => {
    return ToastComponent({
        type: "success",
        message: msg,
    })
}
Toast.Error = (msg: string) => {
    return ToastComponent({
        type: "error",
        message: msg,
        duration: 0,
    })
}
Toast.Warning = (msg: string) => {
    return ToastComponent({
        type: "warning",
        message: msg
    })
}
Toast.Message = (msg: string) => {
    return ToastComponent({
        message: msg,
    })
}
Toast.Clear = () => {
    Values.forEach((value, key) => {
        value.Close()
        Values.delete(key)
    })
}
export default Toast