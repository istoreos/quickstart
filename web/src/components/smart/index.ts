import { App, createApp, createVNode, Plugin, render } from 'vue'
import editDisk from "./edit-disk.vue"
import addTask from "./add-task.vue"
import testTask from "./test-task.vue"
import diskInfo from "./disk-info.vue"
import installComponents from "/@/plugins/componnets"
let commentApp: App | null = null
export default {
    install(app: App) {
        commentApp = app
    }
}

export const ActionSmartEditDisk: ActionSmartEditDisk = (props) => {
    const el = document.createElement("div")
    document.body.appendChild(el)
    const vm = createVNode(editDisk, {
        ...props,
        close: () => {
            close()
        }
    });
    const close = () => {
        el.remove()
    }
    // 捆绑主实例
    if (commentApp) {
        vm.appContext = commentApp._context
    }
    render(vm, el);
}

export const ActionSmartAddTask: ActionSmartAddTask = (props) => {
    const el = document.createElement("div")
    document.body.appendChild(el)
    const vm = createVNode(addTask, {
        ...props,
        close: () => {
            close()
        }
    });
    const close = () => {
        el.remove()
    }
    // 捆绑主实例
    if (commentApp) {
        vm.appContext = commentApp._context
    }
    render(vm, el);
}

export const ActionSmartTestTask: ActionSmartTestTask = (props) => {
    const el = document.createElement("div")
    document.body.appendChild(el)
    const vm = createVNode(testTask, {
        ...props,
        close: () => {
            close()
        }
    });
    const close = () => {
        el.remove()
    }
    // 捆绑主实例
    if (commentApp) {
        vm.appContext = commentApp._context
    }
    render(vm, el);
}

export const ActionSmartDiskInfo: ActionSmartDiskInfo = (props) => {
    const el = document.createElement("div")
    document.body.appendChild(el)
    const vm = createVNode(diskInfo, {
        ...props,
        close: () => {
            close()
        }
    });
    const close = () => {
        el.remove()
    }
    // 捆绑主实例
    if (commentApp) {
        vm.appContext = commentApp._context
    }
    render(vm, el);
}