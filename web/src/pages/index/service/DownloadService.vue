<template>
    <Card :title='$gettext("下载服务")' :showSettings="true" @footer-click="onSetting"
        style="width: 100%;height: 100%; display: block;" :is-settings-menu-open="isSettingsMenuOpen" @update:isSettingsMenuOpen="(val: boolean) => isSettingsMenuOpen = val">
        <template #icon>
            <downloadIcon color="#00a63e" class="icon" />
        </template>
        <template #settings>
            <div class="btn_settings" @click="onSetting">
                <downloadIcon color="#0a0a0a" class="icon1 downloadIcon" style="margin-right: 6px;" />
                <span>{{ $gettext('下载管理') }}</span>
                <div class="rotation" @click.stop="isSettingsMenuOpen = !isSettingsMenuOpen" v-if="Boolean(downloadService)">
                    <moreItem class="moreIcon" />
                </div>
            </div>
        </template>
        <template #settings-menu>
            <div><a @click="onClickAria2">{{ $gettext("Aria2高级配置") }}</a></div>
            <div><a @click="onClickqBittorrent">{{ $gettext("qBittorrent高级配置") }}</a></div>
            <div><a @click="onClickTransmission">{{ $gettext("Transmission高级配置") }}</a></div>
        </template>
        <div class="content">
            <div class="tab">
                <div class="item cloud" :class="{ active: active == 'aria2' }" @click="tabChange('aria2')">
                    <downloadIcon color="#f54900" class="icon2" />
                    <div class="title">Aria2</div>
                    <span v-if="downloadService?.aria2?.status == 'running'">{{ $gettext("已启用") }}</span>
                    <span v-else>{{ $gettext("未启用") }}</span>
                </div>
                <div class="item memory" :class="{ active: active == 'qbittorrent' }" @click="tabChange('qbittorrent')">
                    <downloadIcon color="#4a5565" class="icon2" />
                    <div class="title">qBittorrent</div>
                    <span v-if="downloadService?.qbittorrent?.status == 'running'">{{ $gettext("已启用") }}</span>
                    <span v-else>{{ $gettext("未启用") }}</span>
                </div>
                <div class="item network" :class="{ active: active == 'transmission' }"
                    @click="tabChange('transmission')">
                    <downloadIcon color="#009689" class="icon2" />
                    <div class="title">Transmission</div>
                    <span v-if="downloadService?.transmission?.status == 'running'">{{ $gettext("已启用") }}</span>
                    <span v-else>{{ $gettext("未启用") }}</span>
                </div>
            </div>
            <aria2-vue v-if="active == 'aria2'" :aria2="downloadService?.aria2"></aria2-vue>
            <qbittorrent-vue v-else-if="active == 'qbittorrent'" :qbittorrent="downloadService?.qbittorrent">
            </qbittorrent-vue>
            <transmission-vue v-else-if="active == 'transmission'" :transmission="downloadService?.transmission">
            </transmission-vue>
        </div>
    </Card>
</template>

<script lang="ts" setup>
import { ref } from "vue"
import Card from "../components/Card.vue"
import downloadIcon from "/@/components/svg/download1.vue"
import moreItem from "/@/components/svg/more.vue"
import request from '/@/request';
import aria2Vue from './aria2.vue';
import qbittorrentVue from './qbittorrent.vue';
import transmissionVue from './transmission.vue';
import ActionDownload from "/@/components/action-download"
import appUtils from "/@/utils/app";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

const isSettingsMenuOpen = ref(false)
type DownloadTab = "aria2" | "qbittorrent" | "transmission"
const active = ref<DownloadTab>("aria2")
const userSelectedTab = ref(false)
const tabChange = (index: DownloadTab) => {
    active.value = index
    userSelectedTab.value = true
}
const downloadService = ref<GuideDownloadServiceStatus>()
const setActiveByStatus = () => {
    if (userSelectedTab.value) return
    const priority: DownloadTab[] = ["aria2", "qbittorrent", "transmission"]
    const runningTab = priority.find((key) => downloadService.value?.[key]?.status === "running")
    active.value = runningTab || "aria2"
}

const getData = () => {
    request.Guide.DownloadService.Status.GET().then(res => {
        if (res?.data?.result) {
            const result = res.data.result
            downloadService.value = result
            setActiveByStatus()
        }
    })
}
setTimeout(getData, 800)
const disabled = ref<boolean>(false)
const showAria2Block = ref(false)
const MoreDevice = () => {
    showAria2Block.value = !showAria2Block.value
}
const onSetting = () => {
    //请求下载目录推荐安装位置接口
    request.Guide.DownloadPartition.List.GET().then(res => {
        let partitionList: string[] = []
        if (res?.data?.result?.partitionList) {
            partitionList = res.data.result.partitionList
        }
        ActionDownload({ services: downloadService.value, partitionList, defaultTab: active.value })
    })
}

const installAndGo = async (pkg: string, app: string, href: string) => {
    MoreDevice()
    appUtils.installAndGo(pkg, app, href)
}

// 检测是否安装了Aria2
const onClickAria2 = () => {
    installAndGo("app-meta-aria2", "Aria2", "/cgi-bin/luci/admin/services/aria2")
}

// 检测是否安装了qBittorrent
const onClickqBittorrent = () => {
    installAndGo("app-meta-qbittorrent", "qBittorrent", "/cgi-bin/luci/admin/nas/qBittorrent")
}

// 检测是否安装了Transmission
const onClickTransmission = () => {
    installAndGo("app-meta-transmission", "Transmission", "/cgi-bin/luci/admin/services/transmission")
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
    width: 1.5rem;
    height: 1.5rem;
    margin-bottom: 12px;
}

:deep(.downloadIcon) {
    path {
        fill: var(--app-container_title-color) !important;
    }
}

a {
    color: #1e1e1e;
    text-decoration: none;
    cursor: pointer;
    font-size: 14px;
    display: block;

}

.content {
    color: #333;
    margin-top: 20px;
    margin-bottom: 20px;
    font-weight: normal;

    .tab {
        display: flex;
        gap: 8px;

        .item {
            flex: 1;
            padding: 16px;
            display: flex;
            flex-direction: column;
            align-items: center;
            border-radius: 10px;
            cursor: pointer;
            border: 2px solid transparent;
            box-sizing: border-box;

            .title {
                margin-bottom: 8px;
            }

            >span {
                font-size: 12px;
            }
        }

        .active {
            border: 2px solid #6d6d6d;
        }

        .cloud {
            background-color: #fff7ed;
            color: #ca3500;
        }

        .memory {
            background-color: #f9fafb;
            color: #364153;
        }

        .network {
            background-color: #f0fdfa;
            color: #277881;
        }
    }
}

.btn_settings {
    position: relative;
    padding: 6px 34px 6px 18px;
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
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    .content {
        margin: 10px 0;
    }
}
</style>
