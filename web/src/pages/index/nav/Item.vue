<template>
    <div class="item_container">
        <div class="item" :style="{ backgroundColor: '#f3f7fd' }" @click="onNetIP">
            <intranetIcon color="#2b7fff" class="icon" />
            <span>{{ $gettext("内网配置") }}</span>
        </div>
        <div class="item" :style="{ backgroundColor: '#f4fbf7' }" @click="onTTYD" v-if="feature('ttyd')">
            <terminalIcon color="#00c850" class="icon" />
            <span>{{ $gettext("终端") }}</span>
        </div>
        <div class="item" :style="{ backgroundColor: '#f9f7fd' }" @click="onRenew" v-if="feature('ota')">
            <span class="app-update-button-more" @click.stop.prevent="toggleUpdateMenu" :title="$gettext('固件更新选项')">
                <MoreSvg></MoreSvg>
            </span>
            <downloadIcon color="#ad46ff" class="icon" />
            <span class="renew">
                <i v-if="useSystem.checkUpdate?.needUpdate"></i>
                {{ $gettext("固件更新") }}
                <span v-if="enableAutoCheckUpdateOri && useSystem.checkUpdate == null"
                    style="display: inline-block;margin-left: 4px;"><icon-loading size="1em"
                        color="currentColor" /></span>
            </span>
            <div class="app-update-button-menu" v-show="showUpdateMenu">
                <!-- 点击背景关闭按钮 -->
                <div class="menu_background" @click.stop.prevent="toggleUpdateMenu"></div>
                <ul @click.stop>
                    <li>
                        <switch-box v-model="enableAutoCheckUpdate">
                            <span class="app-update-menu-item"> {{ $gettext("自动检查更新") }}</span>
                        </switch-box>
                        <span v-if="enableAutoCheckUpdateBusy" class="app-update-menu-item-loading"><icon-loading
                                size="1em" color="currentColor" /></span>
                    </li>
                </ul>
            </div>
        </div>
        <div class="item" :class="{ 'disabled-style': !networkStatus?.proto }" :style="{ backgroundColor: '#f1fbfd' }"
            @click="onDNS">
            <DNSIcon color="#00b8db" class="icon" />
            <span class="dns_txt">
                {{ $gettext("DNS配置") }}
                <span v-if="!networkStatus?.proto" style="display: inline-block;margin-left: 4px;"><icon-loading
                        size="1em" color="currentColor" /></span>
            </span>
        </div>
        <div class="item" :style="{ backgroundColor: '#fbf5fa' }" @click="onSoftwareSource">
            <softwareIcon color="#f6339a" class="icon" />
            <span>{{ $gettext("软件源配置") }}</span>
        </div>
        <template v-if="feature('sandbox')">
            <div class="item" :style="{ backgroundColor: '#f9fafb' }" @click="onNotSandBox"
                v-if="sandBoxStatus?.status == 'unsupport'">
                <shieldIcon color="#cac9cd" class="icon" />
                <span>{{ $gettext("开启沙箱") }}</span>
            </div>
            <div class="item" :style="{ backgroundColor: '#fbf4f5' }" @click="onOpenSandBox"
                v-else-if="sandBoxStatus?.status == 'stopped'">
                <shieldIcon color="#fb2c36" class="icon" />
                <span>{{ $gettext("开启沙箱") }}</span>
            </div>
            <div class="item" :style="{ backgroundColor: '#dae8fd' }" @click="onCloseSandBox"
                v-else-if="sandBoxStatus?.status == 'running'">
                <shieldIcon color="#2b7fff" class="icon" />
                <span>{{ $gettext("沙箱已开启") }}</span>
            </div>
        </template>
        <div class="item" :style="{ backgroundColor: '#fcf7f2' }" @click="onLog">
            <logIcon color="#ff6900" class="icon" />
            <span>{{ $gettext("日志查看") }}</span>
        </div>
        <div class="item" :style="{ backgroundColor: '#eff5ff' }" @click="onsystem">
            <settingIcon color="#553afe" class="icon" />
            <span>{{ $gettext("系统维护") }}</span>
        </div>
    </div>
</template>

<script lang="ts" setup>
import intranetIcon from "/@/components/svg/intranet.vue"
import terminalIcon from "/@/components/svg/terminal.vue"
import downloadIcon from "/@/components/svg/download1.vue"
import DNSIcon from "/@/components/svg/DNS.vue"
import softwareIcon from "/@/components/svg/software.vue"
import shieldIcon from "/@/components/svg/shield.vue"
import logIcon from "/@/components/svg/log.vue"
import MoreSvg from "/@/components/svg/more.vue";
import settingIcon from "/@/components/svg/setting.vue"

import { computed, ref, watch } from 'vue';
import { useSystemStore, useNetworkStore } from '/@/plugins/store';
import { feature } from "/@/utils/features"
import { useGettext } from '/@/plugins/i18n'
import Toast from "/@/components/toast";
import request from '/@/request';
import ActionDNS from "/@/components/action-dns"
import ActionNetIp from "/@/components/action-netip"
import ActionSoftwareSource from "/@/components/action-softwaresource"
import ActionSandBox from "/@/components/action-sandbox"
import ActionSandBoxEnvironment from "/@/components/action-sandboxenvironment"

const { $gettext } = useGettext()
const enableAutoCheckUpdateOri = ref(true)
if (window.quickstart_configs?.update?.disable) {
    enableAutoCheckUpdateOri.value = false
}

const showUpdateMenu = ref(false)
const enableAutoCheckUpdate = ref(enableAutoCheckUpdateOri.value)
const enableAutoCheckUpdateBusy = ref(false)

const useSystem = useSystemStore()

const networkStore = useNetworkStore()
const networkStatus = computed(() => {
    return networkStore.status
})

const sandBoxStatus = ref<NasGetSandbox>()
const onDNS = () => {
    ActionDNS()
}

watch(enableAutoCheckUpdate, (value) => {
    enableAutoCheckUpdateBusy.value = true
    request.System.AutoCheckUpdate.POST({ enable: value }).catch(e => {
        Toast.Warning(e)
    }).finally(() => {
        enableAutoCheckUpdateBusy.value = false
    })
})

const onRenew = () => {
    window.location.href = '/cgi-bin/luci/admin/system/ota'
}

const onLog = () => {
    window.location.href = '/cgi-bin/luci/admin/status/logs'
}

const onsystem = () => {
    window.location.href = '/cgi-bin/luci/admin/store/pages/maintance'
}

const onOpenSandBox = () => {
    ActionSandBox()
}

const onCloseSandBox = () => {
    ActionSandBoxEnvironment()
}

const onNotSandBox = () => {
    alert($gettext("该固件不支持沙箱模式"))
}

const onNetIP = () => {
    ActionNetIp()
}
const onSoftwareSource = () => {
    ActionSoftwareSource()
}
const onTTYD = () => {
    window.open(`${window.quickstart_configs?.ttyd?.ssl ? "https" : "http"}://${window.location.hostname}:${window.quickstart_configs?.ttyd?.port || 7681}/`, "_blank")
}

const toggleUpdateMenu = () => {
    showUpdateMenu.value = !showUpdateMenu.value
}

if (enableAutoCheckUpdateOri.value) {
    setTimeout(() => {
        useSystem.requestCheckUpdate()
    }, 1100)
}

if (feature('sandbox')) {
    const getSandBox = () => {
        request.Nas.GetSandbox.GET().then(res => {
            if (res?.data) {
                if ((res?.data?.success || 0) == 0) {
                    if (res?.data?.result) {
                        sandBoxStatus.value = res.data.result
                    }
                } else if (res?.data?.error) {
                    alert(res.data.error)
                }
            }
        }).catch(error => Toast.Warning(error))
    }
    getSandBox()
}

</script>

<style lang="scss" scoped>
// .icon {
//     width: 1.5rem;
//     height: 1.5rem;
//     margin-bottom: 12px;
// }
:deep(.icon) {
    width: 1.5rem;
    height: 1.5rem;
    margin-bottom: 12px;
    display: inline-block;
    /* 确保占位 */
    flex: 0 0 auto;
}

button {
    margin: 0 !important;
}

button.item:disabled {
    opacity: 1;
}

button.item:disabled svg,
button.item:disabled .icon {
    opacity: 1 !important;
    filter: none !important;
    color: #00b8db !important;
    stroke: #00b8db !important;
    fill: #00b8db !important;
}

.item_container {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 16px;
    width: 100%;
    padding-bottom: 4px;
    box-sizing: border-box;

    &::-webkit-scrollbar {
        height: 6px;
    }

    &::-webkit-scrollbar-thumb {
        background: #ccc;
        border-radius: 3px;
    }

    .item {
        position: relative;
        padding: 16px 12px;
        min-width: 180px;
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        font-size: 14px;
        border-radius: 8px;
        box-sizing: border-box;
        border: 1px solid #e5e7eb;
        cursor: pointer;
        color: var(--card-txt-color);
    }

    .app-update-button-more {
        position: absolute;
        top: 4px;
        right: 4px;
    }

    .menu_background {
        position: fixed;
        left: 0;
        top: 0;
        right: 0;
        bottom: 0;
    }

    .renew {
        display: flex;
        align-items: center;

        i {
            display: inline-block;
            padding: 3px;
            background-color: #f00;
            border-radius: 50%;
            margin-right: 4px;
        }
    }

    .dns_txt {
        display: flex;
        align-items: center;
        line-height: 1;
        color: #32325d;
    }

    .disabled-style {
        opacity: 0.6;
        cursor: not-allowed;
        pointer-events: none;
        background-color: #e0e0e0;
    }

    .app-update-button-menu {
        position: absolute;
        z-index: 999;
        width: 30%;
        right: 0;
        top: 0;

        ul {
            background-color: #fff;
            box-shadow: 0 0 10px 1px #373f6924;
            padding: 6px 0px;
            border-radius: 6px;
            top: -45px;
            right: 0;
            text-align: center;
            position: absolute;
            word-break: keep-all;

            li {
                cursor: pointer;
                font-size: 16px;
                line-height: 1em;
                color: #1e1e1e;
                padding: 0 5px;
                position: relative;

                .app-update-menu-item {
                    padding: 5px 2px;
                    white-space: nowrap;
                }

                .app-update-menu-item-loading {
                    display: flex;
                    justify-content: center;
                    align-items: center;
                    position: absolute;
                    width: 100%;
                    height: 100%;
                    top: 0;
                    left: 0;

                    background-color: #fffc;
                }
            }
        }
    }
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    .item_container {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        width: 100%;
        padding: 0;
        box-sizing: border-box;
        justify-content: space-between;

        .item {
            width: 48%;
            min-width: 120px;
            flex-shrink: 0;
        }
    }
}
</style>