<template>
    <Card :title='$gettext("网络接口状态")' :showSettings="true" :is-settings-menu-open="isSettingsMenuOpen"
        @update:isSettingsMenuOpen="(val: boolean) => isSettingsMenuOpen = val">
        <template #icon>
            <interfaceIcon color="#0a0a0a" class="icon interfaceIcon" />
        </template>
        <template #settings>
            <div class="btn_settings" @click="goTo">
                <settingIcon color="#0a0a0a" class="icon1 interfaceIcon" style="margin-right: 6px;" />
                <span>{{ $gettext('配置网络接口') }}</span>
                <!-- <div class="rotation" @click.stop="isSettingsMenuOpen = !isSettingsMenuOpen">
                    <moreItem class="moreIcon" />
                </div> -->
            </div>
        </template>
        <template #settings-menu>
            <div>
                <div v-for="item in porter.portList" :key="item.name" class="row">
                    <input type="checkbox" :value="item.name" :checked="checkedList.includes(item.name)"
                        @change="toggleCheck(item.name, $event)" />
                    <span>{{ item.name }}</span>
                </div>
            </div>
        </template>
        <div class="content">
            <div class="item" v-for="item in porter.portList" v-if="porter.load" ref="el"
                @mouseenter="onMouseenter(item, $event)" @mouseleave="onMouseleave">
                <!-- <div class="item" v-for="item in portList" v-if="porter.load" ref="el" @click="goNetwork"> -->
                <div class="icon_box" :style="{ backgroundColor: item.linkState == 'DOWN' ? '#f3f4f6' : '#dbfce7' }">
                    <interfaceIcon :color="item.linkState == 'DOWN' ? '#99a1af' : '#00a63e'" class="icon2" />
                </div>
                <div>
                    <div class="name">
                        {{ item.name }} {{ item.interfaceNames ?
                            `(${item.interfaceNames.join(",").toLocaleUpperCase()})` : ""
                        }}
                        <!-- <div class="speed" v-if="item.linkSpeed">{{ item.linkSpeed }}</div> -->
                        <div class="speed" v-if="item.linkState == 'DOWN'" style="background: #f3f4f6;color: #4a5565;">
                            {{
                                $gettext("已断开") }}</div>
                    </div>
                    <div style="display: flex;align-items: center;">
                        <div class="status" v-if="item.linkState == 'DOWN'">{{ $gettext("未连接") }}</div>
                        <div class="status" v-else>{{ $gettext("已连接") }}</div>
                        <div class="speed" v-if="item.linkSpeed" style="margin-left: 6px;">{{ item.linkSpeed }}</div>
                    </div>
                </div>
            </div>
        </div>
    </Card>
</template>

<script lang="ts" setup>
import Card from "../components/Card.vue"
import interfaceIcon from "/@/components/svg/interface.vue"
import settingIcon from "/@/components/svg/setting.vue"
import moreItem from "/@/components/svg/more.vue"

import { reactive, ref, watch } from 'vue';
import request from '/@/request';
import Toast from '/@/components/toast';
import { useAppStore } from '/@/plugins/store';
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()
import { useRouter } from 'vue-router'
const router = useRouter()
const isSettingsMenuOpen = ref(false)
const appStore = useAppStore()
const porter = reactive<{
    portList: NetworkPort[],
    load: boolean
}>({
    portList: [],
    load: false
})

const goTo = () => {
    router.push('/interfaceconfig')
}
const goNetwork = () => {
    window.location.href = '/cgi-bin/luci/admin/network/network'
}

// === 封装存储和读取函数 ===
const saveChecked = (key: string, data: string[]) => {
    localStorage.setItem(key, JSON.stringify(data))
}
const loadChecked = (key: string): string[] => {
    const str = localStorage.getItem(key)
    try {
        return str ? JSON.parse(str) : []
    } catch {
        return []
    }
}


// === 本地存储 key ===
const STORAGE_KEY = "checkedPorts"
// 初始默认至少勾选一个，从本地读取
const checkedList = ref<string[]>(loadChecked(STORAGE_KEY))
const portList = ref<NetworkPort[]>([])
const getPortList = () => {
    (porter.load && document.hidden ? Promise.resolve() : request.Network.PortList.GET().then(res => {
        if (res?.data) {
            const { result } = res?.data
            if (result) {
                porter.portList = result.ports || []
                portList.value = result.ports || []

                // 如果本地存储为空，默认勾选全部
                // if (checkedList.value.length === 0) {
                //     porter.portList.forEach(item => {
                //         checkedList.value.push(item.name)
                //     })
                //     saveChecked(STORAGE_KEY, checkedList.value)
                //     portList.value = result.ports || []
                // } else {
                //     portList.value = porter.portList.filter(item => checkedList.value.includes(item.name))
                // }
            }
        }
    })).finally(() => {
        porter.load = true
        setTimeout(getPortList, 10000)
    })
}
getPortList()

watch(checkedList, (newVal) => {
    portList.value = porter.portList.filter(item => newVal.includes(item.name))
    console.log(newVal, 'newVal');
    saveChecked(STORAGE_KEY, newVal)
})

const toggleCheck = (name: string, event: any) => {
    if (event.target.checked) {
        // 勾选时加入
        if (!checkedList.value.includes(name)) {
            checkedList.value = [...checkedList.value, name]
        }
    } else {
        // 取消勾选时，判断是否至少保留一个
        if (checkedList.value.length > 1) {
            checkedList.value = checkedList.value.filter(x => x !== name)
        } else {
            Toast.Warning($gettext("至少保留一个网络接口！"))
            // 强制保持选中
            event.target.checked = true
        }
    }
}

const el = ref<HTMLElement | null>(null)
// 鼠标移入事件
const onMouseenter = (item: any, event: MouseEvent) => {
    appStore.portitemStyle.show = true
    const target = event?.target as HTMLElement
    if (target) {
        const { left, top } = target.getBoundingClientRect()
        appStore.portitemStyle.left = left
        appStore.portitemStyle.top = top
    }
    appStore.portitemStyle.portitem = item

}
// 鼠标移出事件
const onMouseleave = () => {
    appStore.portitemStyle.show = false
}

</script>

<style lang="scss" scoped>
.icon {
    width: 1.3rem;
    height: 1.3rem;
}

.icon1 {
    width: 1rem;
    height: 1rem;
}

.icon2 {
    width: 1rem;
    height: 1rem;
}

:deep(.interfaceIcon) {
    path {
        fill: var(--app-container_title-color) !important;
    }
}

:deep(.footer-btn) {
    margin-top: 6px;
}


.content {
    margin-top: 6px;
    padding-bottom: 16px;
    min-height: 30px;
    display: flex;
    overflow-x: auto;
    overflow-y: hidden;
    -webkit-overflow-scrolling: touch;
    scrollbar-gutter: stable both-edges;
    scrollbar-width: thin;
    scrollbar-color: rgba(0, 0, 0, .35) transparent;

    &::-webkit-scrollbar {
        height: 6px;
    }

    &::-webkit-scrollbar-thumb {
        background: #ccc;
        border-radius: 3px;
    }

    .item {
        position: relative;
        display: inline-flex;
        align-items: center;
        padding-right: 20px;
        margin-right: 16px;
        cursor: pointer;

        &::before {
            content: "";
            display: inline-block;
            position: absolute;
            right: 0;
            top: 50%;
            transform: translateY(-50%);
            width: 1px !important;
            height: 60%;
            background: #e0e0e0;
        }

        // 让最后一个 item 去掉分隔线
        &:last-child::before {
            content: none;
        }
    }

    .icon_box {
        display: inline-flex;
        justify-content: center;
        align-items: center;
        width: 30px;
        height: 30px;
        border-radius: 8px;
        background: #dbfce7;
        margin-right: 12px;
    }

    .name {
        display: flex;
        align-items: center;
        margin-bottom: 6px;
    }

    .speed {
        padding: 4px 6px;
        background: #dbfce7;
        font-size: 12px;
        border-radius: 6px;
        line-height: 1;
        color: #008236;
        margin-left: 8px;
    }

    .status {
        font-size: 12px;
        color: #6a7282;
    }
}
.content > * { flex: 0 0 auto; }
:deep(.content::-webkit-scrollbar) { height: 8px; }
:deep(.content::-webkit-scrollbar-thumb) { border-radius: 4px; background: rgba(0,0,0,.35); }
:deep(.content::-webkit-scrollbar-track) { background: transparent; }

.btn_settings {
    position: relative;
    padding: 6px 18px;
    border-radius: 4px;
    border: 1px solid var(--btn-border-color);
    line-height: 1;
    display: flex;
    align-items: center;
}

.rotation {
    position: absolute;
    right: 2px;
    top: 50%;
    height: 100%;
    transform: translateY(-50%);
    border-left: 1px solid var(--btn-border-color);
    display: flex;
    align-items: center;

    .moreIcon {
        transform: rotate(90deg);
    }
}

.row input[type="checkbox"] {
    vertical-align: middle;
    margin: 0;
}

.row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0;
    margin: 6px 0;
    display: flex;
    align-items: center;

    >input {
        margin-right: 6px !important;
        margin-top: 0;
    }
}

:deep(.dropdown-menu div:hover) {
    background: transparent !important;
}

:deep(.dropdown-menu) {
    padding: 8px 0;

    >div {
        padding: 0;
    }
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {}
</style>