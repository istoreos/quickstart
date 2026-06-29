<template>
    <div id="page">
        <h2 name="content">{{ $gettext("网口配置") }}</h2>
        <div class="container">
            <div class="table-wrapper">
                <!-- 表头 -->
                <div class="table-header">
                    <div class="header-cell device-col" v-for="item in porter.devices" :key="item.name">
                        <div class="interface-device-flex">
                            <item-vue :item="item" />
                        </div>
                    </div>
                    <div class="header-cell spacer-col"></div>
                    <div class="header-cell"></div>
                    <div class="header-cell action-col"></div>
                </div>

                <!-- 表体 -->
                <div class="table-body">
                    <!-- LAN接口行 -->
                    <template v-for="(item, i) in interfaces.lan" :key="i">
                        <div class="table-row">
                            <div class="table-cell device-col" v-for="device in porter.devices" :key="device.name">
                                <input type="checkbox" :name="device.name" :value="device.name"
                                    v-model="item.deviceNames" @input="onCheckboxChange($event, i)">
                            </div>
                            <div class="table-cell spacer-col"></div>
                            <div class="table-cell name-col">
                                <b>{{ item.name }}</b>
                            </div>
                            <div class="table-cell action-col">
                                <button class="btn cbi-button cbi-button-apply" :title="$gettext('编辑')"
                                    @click="AddInterfaces('lan', i)">
                                    {{ $gettext("编辑") }}
                                </button>
                                <template v-if="i !== 0">
                                    <button class="cbi-button cbi-button-remove" @click="DelInterfaces('lan', i)">
                                        {{ $gettext("删除") }}
                                    </button>
                                </template>
                            </div>
                        </div>
                    </template>

                    <!-- 添加LAN按钮行 -->
                    <div class="table-row add-row" @click="AddInterfaces('lan')">
                        <div class="table-cell device-col" v-for="item in porter.devices" :key="item.name"></div>
                        <div class="table-cell spacer-col"></div>
                        <div class="table-cell name-col"></div>
                        <div class="table-cell action-col">
                            <AddSvg class="icon" />
                        </div>
                    </div>

                    <!-- WAN接口行 -->
                    <template v-for="(item, i) in interfaces.wan" :key="i">
                        <div class="table-row">
                            <div class="table-cell device-col" v-for="device in porter.devices" :key="device.name">
                                <input type="checkbox" :name="device.name" :value="device.name"
                                    v-model="item.deviceNames" @input="onCheckboxWan($event, i)">
                            </div>
                            <div class="table-cell spacer-col"></div>
                            <div class="table-cell name-col">
                                <b>{{ item.name }}</b>
                            </div>
                            <div class="table-cell action-col">
                                <button class="btn cbi-button cbi-button-apply" :title="$gettext('编辑')"
                                    @click="AddInterfaces('wan', i)">
                                    {{ $gettext("编辑") }}
                                </button>
                                <template v-if="i !== 0">
                                    <button class="cbi-button cbi-button-remove" @click="DelInterfaces('wan', i)">
                                        {{ $gettext("删除") }}
                                    </button>
                                </template>
                            </div>
                        </div>
                    </template>

                    <!-- 添加WAN按钮行 -->
                    <div class="table-row add-row" @click="AddInterfaces('wan')">
                        <div class="table-cell device-col" v-for="item in porter.devices" :key="item.name"></div>
                        <div class="table-cell spacer-col"></div>
                        <div class="table-cell name-col"></div>
                        <div class="table-cell action-col">
                            <AddSvg class="icon" />
                        </div>
                    </div>
                </div>
            </div>
        </div>
        <div class="cbi-page-actions control-group">
                <input class="btn cbi-button cbi-button-apply" type="button" :value="$gettext('保存并应用')"
                    @click="ClickSave" :disabled="disabled">
            </div>
    </div>
</template>

<script setup lang="ts">
// 保持原有script部分完全不变
import { ref, PropType, reactive } from 'vue';
import { useGettext, formatNumber } from '/@/plugins/i18n'
const { $gettext, $ngettext } = useGettext()

import request from '/@/request';
import itemVue from "/@/components/interfacer/item.vue"
import AddSvg from "/@/components/svg/add.vue"
import ActionEdit from "/@/components/action-edit"
import Toast from '/@/components/toast';
const porter = reactive<{
    devices: NetworkPortInfo[]
    interfaces: NetworkInterfaceInfo[]
}>({
    devices: [],
    interfaces: []
})
const disabled = ref(false)
const interfaces = reactive({
    lan: [] as NetworkInterfaceInfo[],
    wan: [] as NetworkInterfaceInfo[],
})
// 网络接口配置
const getData = () => {
    request.Network.GetInterfaceConfig.GET().then(res => {
        if (res.data) {
            const { result } = res.data
            if (result) {
                porter.devices = result.devices || []
                porter.interfaces = result.interfaces || []
                for (let i = 0; i < porter.interfaces.length; i++) {
                    if (porter.interfaces[i].firewallType == "wan") {
                        interfaces.wan.push(porter.interfaces[i])
                    } else if (porter.interfaces[i].firewallType == "lan") {
                        interfaces.lan.push(porter.interfaces[i])
                    }
                }
            }
        }

    })
}
getData()
const DelInterfaces = (name: FirewallType, i: number) => {
    if (name == "wan") {
        interfaces.wan.splice(i, 1)
    } else if (name == "lan") {
        interfaces.lan.splice(i, 1)
    }
}
const AddInterfaces = (name: FirewallType, i?: number) => {
    if (i == null) {
        let index = name == "wan" ? interfaces.wan.length : interfaces.lan.length
        if (index == 6 && name == "wan") {
            index++
        }
        ActionEdit({
            e: "add",
            name: name,
            inface: {
                name: name + `${index}`,
                proto: "dhcp",
                ipv4Addr: "",
                ipv6Addr: "",
                portName: "",
                deviceNames: [],
                ports: [],
                firewallType: name,
            },
            next: (inface) => {
                if (name == "wan") {
                    interfaces.wan.push(inface)
                } else {
                    interfaces.lan.push(inface)
                }
                Toast.Message($gettext("请在保存以后前往'网络-接口'页面配置接口详细参数"))
            }

        })
    } else {
        ActionEdit({
            e: "edit",
            name: name,
            inface: name == "wan" ? interfaces.wan[i] : interfaces.lan[i],
            next: (inface) => {
                if (name == "wan") {
                    interfaces.wan[i] = inface
                } else {
                    interfaces.lan[i] = inface
                }
            }
        })
    }
}
const indexOf = (ary: string[] | null, val: string) => {
    return ary ? ary.indexOf(val) : -1
}
const onCheckboxChange = (e: Event, index: number) => {
    const target = e.target as HTMLInputElement
    const value = target.value
    for (let i = 0; i < interfaces.wan.length; i++) {
        const n = indexOf(interfaces.wan[i].deviceNames, value)
        if (n != -1) {
            interfaces.wan[i].deviceNames.splice(n, 1)
        }
    }
    for (let i = 0; i < interfaces.lan.length; i++) {
        if (i != index) {
            const n = indexOf(interfaces.lan[i].deviceNames, value)
            if (n != -1) {
                interfaces.lan[i].deviceNames.splice(n, 1)
            }
        }
    }
    const n = indexOf(interfaces.lan[index].deviceNames, value)
    if (n != -1) {
        interfaces.lan[index].deviceNames.splice(n, 1)
    } else {
        if (interfaces.lan[index].deviceNames === null) {
            interfaces.lan[index].deviceNames = []
        }
        interfaces.lan[index].deviceNames.push(value)
    }
}
const onCheckboxWan = (e: Event, index: number) => {
    const target = e.target as HTMLInputElement
    const value = target.value
    for (let i = 0; i < interfaces.wan.length; i++) {
        if (i != index) {
            const n = indexOf(interfaces.wan[i].deviceNames, value)
            if (n != -1) {
                interfaces.wan[i].deviceNames.splice(n, 1)
            }
        }
    }
    for (let i = 0; i < interfaces.lan.length; i++) {
        const n = indexOf(interfaces.lan[i].deviceNames, value)
        if (n != -1) {
            interfaces.lan[i].deviceNames.splice(n, 1)
        }
    }
    interfaces.wan[index].deviceNames = [value]

}
const ClickSave = async () => {
    disabled.value = true
    const configs: NetworkInterfaceConfig[] = []
    for (let i = 0; i < interfaces.wan.length; i++) {
        const item = interfaces.wan[i]
        configs.push({
            name: item.name,
            proto: item.proto,
            devices: item.deviceNames || [],
            firewallType: item.firewallType
        })
    }
    for (let i = 0; i < interfaces.lan.length; i++) {
        const item = interfaces.lan[i]
        if ("lan" === item.name && (!item.deviceNames || item.deviceNames.length == 0)
            && !confirm($gettext("LAN口未关联任何物理网口，可能导致路由器失联，是否继续操作？"))) {
            disabled.value = false
            return
        }
        configs.push({
            name: item.name,
            proto: item.proto,
            devices: item.deviceNames || [],
            firewallType: item.firewallType
        })
    }
    const load = Toast.Loading($gettext("保存中..."))
    try {
        const res = await request.Network.POSTInterfaceConfig.POST({
            configs: configs
        })
        if (res.data) {
            const { success, result, error } = res.data
            if (error) {
                throw error
            }
            if ((success || 0) == 0) {
                Toast.Success($gettext("配置成功"))
            }
        }
    } catch (error) {
        Toast.Error(`${error}`)
    } finally {
        load.Close()
        disabled.value = false
    }

}
</script>

<style lang="scss" scoped>
:deep(.app-container_status-label_bg) {
    margin: 8px 0;
    flex: 0 0 170px;
    height: 80px;
    justify-content: start;
}

:deep(.interface-device-flex) {
    justify-content: start;
}

.container {
    width: 100%;
    overflow-x: auto;
    /* 整体横向滚动 */

    .table-wrapper {
        min-width: 1280px;
        /* 最小宽度防止内容过窄 */
        width: max-content;
        /* 内容宽度撑开 */

        .table-body {
            display: flex;
            flex-direction: column;
            min-width: 100%;
        }

        .table-header {
            display: flex;
            // font-weight: bold;
            border-bottom: 2px solid #e5e7eb;
            background-color: #f8fafc;
            padding-left: 10px;
        }

        .header-cell {}

        .table-row {
            display: flex;
            min-width: 100%;
            align-items: center;
            border-bottom: 1px solid #e5e7eb;
            transition: background-color 0.2s;

            &:hover {
                background-color: #f3f4f6;
            }
        }

        .add-row {
            cursor: pointer;

            &:hover {
                background-color: #f0f9ff;
            }
        }

        .table-cell {
            padding: 12px 16px;
            box-sizing: border-box;
            display: flex;
            justify-content: center;
        }

        .device-col {
            flex: 0 0 200px;
            /* 固定设备列宽度 */
            min-width: 200px;
        }

        .spacer-col:first-of-type {
            flex: 0 0 10px;
            /* 第一个间隔列 */
        }

        .spacer-col:last-of-type {
            flex: 0 0 32px;
            /* 第二个间隔列 */
        }

        .name-col {
            flex: 0 0 150px;
            /* 接口名称列宽度 */
            min-width: 150px;
            text-align: left;
        }

        .action-col {
            flex: 0 0 auto;
            /* 操作列自适应 */
            text-align: right;
            min-width: 160px;
            /* 操作列最小宽度 */
        }

        .icon {
            width: 48px;
            height: 100%;
            cursor: pointer;
        }

        .interface-device-flex {
            display: flex;
            justify-content: center;
            width: 100%;
        }
    }

    .cbi-page-actions {
        margin-top: 20px;
        display: flex;
        justify-content: flex-end;
    }

    // 移动端适配
    @media (max-width: 768px) {
        :deep(.app-container_status-label_bg) {
            margin: 8px 0;
            flex: 0 0 80px;
            width: 120px;
            height: 80px;
            justify-content: start;
        }

        .table-wrapper {
            min-width: 100%;
            /* 增大最小宽度适应小屏幕 */
        }

        .table-cell {
            padding: 8px 12px;
        }

        .device-col {
            flex: 0 0 120px !important;
            /* 固定设备列宽度 */
            min-width: 120px !important;
            margin-right: 16px;
        }

        .name-col {
            flex: 0 0 80px !important;
            /* 缩小接口名称列宽度 */
            min-width: 80px !important;
        }

        .action-col {
            min-width: 120px;
            /* 缩小操作列宽度 */
        }

        .interface-device-flex {
            flex-direction: column;
            gap: 4px;
        }
    }
}
</style>