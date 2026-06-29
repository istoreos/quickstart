<template>
    <div>
        <Card :title="$gettext('网络连接和IP地址')" :showSettings="true">
            <template #settings>
                <div class="device-manage-btn" @click="goDeviceManagement">
                    <topologyIcon color="#3a3a3a" class="device-manage-icon" />
                    {{ $gettext('设备管理') }}
                </div>
            </template>
            <template #icon>
                <networkIcon color="#0a0a0a" class="icon networkIcon" />
            </template>
            <div class="info_content">
                <template v-if="info != null">
                    <!-- 网络连接正常 -->
                    <div class="status_box" v-if="info.networkInfo == 'netSuccess'">
                        <div class="status_name">
                            <successIcon color="#00a63e" class="icon" />
                            <span>{{ $gettext("网络连接正常") }}</span>
                        </div>
                        <div class="status_time">{{ stampForm(info.uptimeStamp) }}</div>
                    </div>
                    <!-- DNS错误 -->
                    <div class="status_box" v-else-if="info.networkInfo == 'dnsFailed'">
                        <div class="flex">
                            <div class="status_name">
                                <warnIcon style="width: 1.2rem;height: 1.2rem;margin-right: 4px;" />
                                <span>{{ $gettext("DNS错误") }}</span>
                            </div>
                            <div class="dns-btn">
                                <button class="btn-primary" @click="onDNS">{{ $gettext("DNS配置") }}</button>
                            </div>
                        </div>
                        <div class="status_time" style="background: #ffe2e2;color: #c10007;">
                            {{ stampForm(info.uptimeStamp) }}
                        </div>
                    </div>
                    <!-- 软件源错误 -->
                    <div class="status_box" v-else-if="info.networkInfo == 'softSourceFailed'">
                        <div class="flex">
                            <div class="status_name">
                                <softwareIcon color="#9810fa" style="width: 1.5rem;height: 1.5rem;margin-right: 4px;" />
                                <span>{{ $gettext("软件源错误") }}</span>
                            </div>
                            <div class="dns-btn">
                                <button class="btn-pink" @click="onSoftSource">{{ $gettext("软件源配置") }}</button>
                            </div>
                        </div>
                        <div class="status_time" style="background: #ffe2e2;color: #c10007;">{{
                            stampForm(info.uptimeStamp) }}</div>
                    </div>
                    <!-- 未连接互联网 -->
                    <div class="status_box" v-else-if="info.networkInfo == 'netFailed'">
                        <div class="status_name">
                            <notConnectedIcon style="width: 1.2rem;height: 1.2rem;margin-right: 4px;" />
                            <span>{{ $gettext("未联网") }}</span>
                        </div>
                        <div class="status_time" style="background: #ffe2e2;color: #c10007;">{{
                            stampForm(info.uptimeStamp) }}</div>
                    </div>
                    <!-- 检测中 -->
                    <div class="status_box" v-else>
                        <div class="status_name">
                            <wifiIcon color="#d08700" class="icon" />
                            <span>{{ $gettext("检测中...") }}</span>
                        </div>
                    </div>
                </template>
                <div class="ip_info">
                    <div class="ip_item">
                        <div>IPv4 （{{ info.defaultInterface }}）</div>
                        <div class="ip_action">
                            <div class="ip_tag">{{ getProto(info.proto || '') }}</div>
                            <div class="ip_tag" v-if="info.ipv4addr" style="cursor: pointer;"
                                @click="openAndHideIpv4">
                                <closeEyesIcon v-if="isShowIpv4" />
                                <openEyesIcon v-else />
                                {{ $gettext("已启用") }}
                            </div>
                        </div>
                    </div>
                    <div class="ip_address" v-if="info.ipv4addr">{{ isShowIpv4 ? info.ipv4addr : hiddenAddressMask }}</div>
                    <div class="ip_address" v-else>-</div>
                </div>
                <div class="ip_info">
                    <div class="ip_item">
                        <div>IPv6</div>
                        <div class="ip_action" v-if="info.ipv6addr">
                            <div class="ip_tag" style="cursor: pointer;" @click="openAndHideIpv6">
                                <closeEyesIcon v-if="isShowIpv6" />
                                <openEyesIcon v-else />
                                {{ $gettext("已启用") }}
                            </div>
                        </div>
                        <div class="ip_tag" v-else>{{ $gettext("未启用") }}</div>
                    </div>
                    <div class="ip_address" v-if="info.ipv6addr">{{ isShowIpv6 ? info.ipv6addr : hiddenAddressMask }}</div>
                    <div class="ip_address" v-else>-</div>
                </div>
                <div class="ip_info" style="margin-bottom: 0;">
                    <div class="ip_item">
                        <div>DNS（{{ getDnsProto(info.dnsProto) }}）</div>
                    </div>
                    <div class="ip_address" v-for="item in info.dnsList">{{ item }}</div>
                </div>

                <!-- <div class="line"></div>
                <div class="ip_item">
                    <div>{{ $gettext("连接设备数量") }}</div>
                    <div class="device">{{ deviceList?.devices?.length || 0 }} {{ $gettext("台设备") }}</div>
                </div>
                <div class="line1"></div>
                <div class="ip_info">
                    <div class="ip_item">
                        <div>{{ $gettext("今日数据") }}</div>
                        <div class="download">{{ $gettext("上传") }}/{{ $gettext("下载") }}</div>
                    </div>
                    <div class="ip_address">473 MB / 2.34GB</div>
                </div> -->

                <!-- <div class="line"></div>
                <div class="ip_item">
                    <div>{{ $gettext("网络延迟") }}</div>
                    <div class="delay">12ms</div>
                </div>
                <div class="ip_item">
                    <div>{{ $gettext("信号强度") }}</div>
                    <div class="device">优秀</div>
                </div> -->
            </div>
        </Card>
    </div>
</template>

<script lang="ts" setup>
import openEyesIcon from "/@/components/svg/openEyes.vue"
import closeEyesIcon from "/@/components/svg/closeEyes.vue"
import networkIcon from "/@/components/svg/DNS.vue"
import topologyIcon from "/@/components/svg/topology.vue"
import successIcon from "/@/components/svg/success.vue"
import notConnectedIcon from "/@/components/svg/notConnected.vue"
import warnIcon from "/@/components/svg/warn.vue"
import softwareIcon from "/@/components/svg/software.vue"
import wifiIcon from '/@/components/svg/wifi.vue'
import Card from "../components/Card.vue"

import ActionDNS from "/@/components/action-dns"
import ActionSoftwareSource from "/@/components/action-softwaresource"
import { reactive, computed, ref, onUnmounted, watch } from 'vue';
import { useNetworkStore } from '/@/plugins/store'
import utils from '/@/utils'
import request from '/@/request';
import { useGettext } from '/@/plugins/i18n'
import { useRouter } from 'vue-router'
const { $gettext } = useGettext()
const networkStore = useNetworkStore()
const router = useRouter()

const hiddenAddressMask = '***************************************'

const useAutoHideToggle = () => {
    const isShow = ref(false)
    const left = ref(60)
    let timer: NodeJS.Timeout | null = null

    const clear = () => {
        if (timer !== null) {
            clearInterval(timer)
            timer = null
        }
    }
    const tick = () => {
        if (left.value > 0) left.value--
        else clear()
    }
    const start = () => {
        clear()
        timer = setInterval(tick, 1000)
    }
    watch(left, (newVal: number) => {
        if (newVal === 0) {
            isShow.value = false
        }
    })
    const toggle = () => {
        isShow.value = !isShow.value
        if (isShow.value) {
            left.value = 60
            start()
        } else {
            left.value = 60
            clear()
        }
    }

    onUnmounted(clear)

    return { isShow, toggle }
}

const { isShow: isShowIpv4, toggle: openAndHideIpv4 } = useAutoHideToggle()
const { isShow: isShowIpv6, toggle: openAndHideIpv6 } = useAutoHideToggle()

const info = computed(() => {
    // networkStore.status.networkInfo = 'dnsFailed'
    return networkStore.status
})

const onDNS = () => {
    ActionDNS()
}
const onSoftSource = () => {
    ActionSoftwareSource()
}

const deviceList = computed(() => networkStore.deviceList)

const porter = reactive<{
    portList: NetworkPort[],
    load: boolean
}>({
    portList: [],
    load: false
})

const getProto = (proto: string) => {
    switch (proto) {
        case "pppoe":
            return $gettext("拨号上网")
        case "static":
            return $gettext("静态网络")
        case "dhcp":
            return "DHCP"
    }
    return proto && proto.toUpperCase()
}

const getDnsProto = (dnsProto?: string) => {
    switch (dnsProto) {
        case "manual":
            return $gettext("手动配置")
        case "auto":
            return $gettext("自动获取")
        default:
            return ""
    }
}

const getPortList = () => {
    (porter.load && document.hidden ? Promise.resolve() : request.Network.PortList.GET().then(res => {
        if (res?.data) {
            const { result } = res?.data
            if (result) {
                porter.portList = result.ports || []
            }
        }
    })).finally(() => {
        porter.load = true
        setTimeout(getPortList, 10000)
    })
}
getPortList()
const stampForm = utils.stampForm

const goDeviceManagement = () => {
    router.push("/devicemanagement")
}

</script>

<style lang="scss" scoped>
.icon {
    width: 1.5rem;
    height: 1.5rem;
}

:deep(.networkIcon) {
    path {
        fill: var(--app-container_title-color) !important;
    }
}

.flex {
    display: flex;
    align-items: center;
}

.info_content {
    margin: 12px 0 4px;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: space-between;

    .status_box {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding-bottom: 12px;
        border-bottom: 1px solid var(--border-color);
        font-size: 14px;
        line-height: 1;
        margin-bottom: 6px;

        .status_name {
            display: flex;
            align-items: center;
            line-height: 1;

            .icon {
                width: 1rem;
                height: 1rem;
                margin-right: 6px;
            }
        }

        .status_time {
            padding: 4px 8px;
            background: #dbfce7;
            color: #008236;
            border-radius: 4px;
        }
    }

    .ip_item {
        display: flex;
        justify-content: space-between;
        align-items: center;

        .ip_action {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .ip_tag {
            padding: 3px 6px;
            border-radius: 6px;
            border: 1px solid #d1d5db;
            font-size: 12px;
            line-height: 1;
            display: flex;
            align-items: center;

            >svg {
                width: 1.3rem;
                height: 1.3rem;
                vertical-align: middle;
                margin-right: 4px;
            }
        }

        .device {
            font-size: 14px;
            color: #155dfc;
        }

        .delay {
            font-size: 14px;
            color: #00a663;
        }

        .download {
            font-size: 14px;
            color: var(--item-label_key-span-color);
        }
    }

    .line {
        width: 100%;
        height: 1px;
        background: var(--btn-border-color);
        margin-bottom: 20px;
    }

    .line1 {
        width: 100%;
        height: 1px;
        background: var(--btn-border-color);
        margin: 20px 0;
    }

    .ip_item:nth-last-child(1) {
        margin-top: 20px;
    }

    .ip_info {
        // margin: 24px 0 20px;
        min-height: 60px;
        display: flex;
        flex-direction: column;
        justify-content: flex-end;
        // font-size: 14px;

        .ip_address {
            color: var(--item-label_key-span-color);
            margin-top: 10px;
        }
    }
}

.device-manage-btn {
    border: 1px solid var(--btn-border-color);
    border-radius: 6px;
    background: var(--card-bg-color);
    color: var(--app-container_title-color);
    padding: 8px 10px 6px;
    line-height: 1;
    font-size: 12px;
    cursor: pointer;
    transition: background 0.2s, transform 0.1s;
    display: inline-flex;
    align-items: center;
    gap: 6px;

    &:hover {
        background: var(--btn-border-hover-color);
    }

    &:active {
        transform: scale(0.97);
    }
}

.device-manage-icon {
    width: 14px;
    height: 14px;
    display: inline-block;
}


.btn-primary {
    background-color: #00b8db;
    color: white;
    border: none;
    padding: 3px 16px;
    border-radius: 8px;
    font-size: 14px;
    cursor: pointer;
    transition: background 0.2s, transform 0.1s;
    margin-left: 6px;
}

.btn-primary:hover {
    background-color: #26a7c7;

}

.btn-primary:active {
    transform: scale(0.95);
}

.btn-pink {
    background-color: #f751a9;
    color: white;
    border: none;
    padding: 3px 12px;
    border-radius: 8px;
    font-size: 14px;
    cursor: pointer;
    transition: background 0.2s, transform 0.1s;
    margin-left: 6px;
}

.btn-pink:hover {
    background-color: #e60076;
}

.btn-pink:active {
    transform: scale(0.95);
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    .flex {
        flex-direction: column;
        align-items: flex-start;
    }
}
</style>
