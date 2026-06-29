import { createRouter, createWebHistory } from "vue-router"
import IndexPage from "/@/pages/index/index.vue"
import NetworkPage from "/@/pages/network/index.vue"
import NetwokHomePage from "/@/pages/network/index/index.vue"
import NetwokPppoePage from "/@/pages/network/pppoe/index.vue"
import NetwokDhcpPage from "/@/pages/network/dhcp/index.vue"
import NetwokGatewayPage from "/@/pages/network/gateway/index.vue"
import RaidPage from "/@/pages/raid/index/index.vue"
import SmartPage from "/@/pages/smart/index.vue"
import SmartPageIndex from "/@/pages/smart/index/index.vue"
import SmartPageTask from "/@/pages/smart/task/index.vue"
import SmartPageLog from "/@/pages/smart/log/index.vue"
import InterfaceConfigPage from "/@/pages/interfaceconfig/index.vue"
import DeviceManagement from "/@/pages/device/index.vue"
import NetworkSpeedTest from "/@/pages/networkSpeedTest/index.vue"
import QuickwifiPage from "/@/pages/quickwifi/index.vue"
const getVueBase = (): string => {
    return window.vue_base || "/cgi-bin/luci/admin/quickstart"
}
const router = createRouter({
    history: createWebHistory(getVueBase()),
    routes: [
        {
            name: "IndexPage",
            path: "/",
            meta: {
                title: "控制台"
            },
            component: IndexPage,
        },
        {
            name: "NetworkPage",
            path: "/network",
            meta: {
                title: "网络设置向导"
            },
            component: NetworkPage,
            children: [
                {
                    path: "",
                    component: NetwokHomePage,
                },
                {
                    path: "pppoe",
                    component: NetwokPppoePage,
                },
                {
                    path: "dhcp",
                    component: NetwokDhcpPage,
                },
                {
                    path: "gateway",
                    component: NetwokGatewayPage,
                }
            ]
        },
        {
            path: "/quickwifi",
            component: QuickwifiPage
        },
        {
            name: "RaidPage",
            path: "/raid",
            meta: {
                title: "raid向导"
            },
            component: RaidPage,
        },
        {
            name: "SmartPage",
            path: "/smart",
            meta: {
                title: "smart检测"
            },
            component: SmartPage,
            children: [
                {
                    path: "",
                    component: SmartPageIndex
                },
                {
                    path: "task",
                    component: SmartPageTask
                },
                {
                    path: "log",
                    component: SmartPageLog
                }
            ]
        },
        {
            path: "/interfaceconfig",
            component: InterfaceConfigPage
        },
        {
            path: "/devicemanagement",
            component: DeviceManagement
        },
        {
            path: "/networkSpeedTest",
            component: NetworkSpeedTest
        }
    ],
})
router.beforeEach((to, form) => {
    if (to.meta.title) {
        // document.title = `${to.meta.title}`
    }
    return true
})
export {
    router,
    getVueBase,
}
