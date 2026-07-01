<template>
    <Card :title='$gettext("存储服务")' style="width: 100%;height: 100%; display: block;" :is-settings-menu-open="isSettingsMenuOpen" @update:isSettingsMenuOpen="(val: boolean) => isSettingsMenuOpen = val">
        <template #icon>
            <shareIcon color="#4f39f6" class="icon" />
        </template>
        <template #settings>
            <div class="btn_settings" @click="onSetting">
                <settingIcon color="#0a0a0a" class="icon1 settings-icon" style="margin-right: 6px;" />
                <span>{{ $gettext('配置存储服务') }}</span>
                <div class="rotation" @click.stop="isSettingsMenuOpen = !isSettingsMenuOpen">
                    <moreItem class="moreIcon" />
                </div>
            </div>
        </template>
        <template #settings-menu>
            <div><a href="/cgi-bin/luci/admin/nas/unishare">{{ $gettext("统一共享高级配置") }}</a></div>
        </template>
        <div class="content">
            <div class="tab">
                <div class="item cloud" :class="{ active: activeIndex == 0 }" @click="tabChange(0)">
                    <cloudIcon color="#155dfc" class="icon2" />
                    <div class="title">{{ $gettext("易有云") }}</div>
                    <span v-if="service?.linkease?.enabel">{{ $gettext("已配置") }}</span>
                    <span v-else>{{ $gettext("未配置") }}</span>
                </div>
                <div class="item memory" :class="{ active: activeIndex == 1 }" @click="tabChange(1)">
                    <memoryIcon color="#0bab47" class="icon2" />
                    <div class="title">SAMBA</div>
                    <span>{{ service?.sambas?.length ? $gettext('已启用') : $gettext('未启用') }}</span>
                </div>
                <div class="item network" :class="{ active: activeIndex == 2 }" @click="tabChange(2)">
                    <networkIcon color="#9810fa" class="icon2" />
                    <div class="title">WEBDAV</div>
                    <span>{{ service?.webdav?.path ? $gettext('已启用') : $gettext('未启用') }}</span>
                </div>
            </div>
            <linkease-vue v-if="activeIndex == 0" :linkease="service?.linkease"></linkease-vue>
            <samba-vue v-else-if="activeIndex == 1" :sambas="service?.sambas"></samba-vue>
            <webdav-vue v-else-if="activeIndex == 2" :webdav="service?.webdav"></webdav-vue>
        </div>
    </Card>
</template>

<script lang="ts" setup>
import { ref } from "vue"
import Card from "../components/Card.vue"
import shareIcon from "/@/components/svg/share.vue"
import memoryIcon from "/@/components/svg/memory.vue"
import networkIcon from "/@/components/svg/DNS.vue"
import cloudIcon from "/@/components/svg/cloud.vue"
import settingIcon from "/@/components/svg/setting.vue"
import moreItem from "/@/components/svg/more.vue"

import { useGettext } from '/@/plugins/i18n'
import request from '/@/request';
import sambaVue from './samba.vue';
import webdavVue from './webdav.vue';
import linkeaseVue from './linkease.vue';
import ActionNAS from "/@/components/action-nas"
import { useNasStore } from '/@/plugins/store';

const { $gettext } = useGettext()
const service = ref<NasServiceStatus>()
const nasStore = useNasStore()

const isSettingsMenuOpen = ref(false)

const getData = () => {
    request.Nas.Service.Status.GET().then(res => {
        if (res?.data?.result) {
            const result = res.data.result
            service.value = result
            setActiveByStatus()
            if (result.webdav) {
                nasStore.webdav = result.webdav
            }
        }
    })
}
getData()
const onSetting = () => {
    ActionNAS({
        setup: 0
    })
}

const activeIndex = ref(0)
const hasUserSelectedTab = ref(false)
const setActiveByStatus = () => {
    if (hasUserSelectedTab.value) return
    const states = [
        Boolean(service.value?.linkease?.enabel),
        Boolean(service.value?.sambas?.length),
        Boolean(service.value?.webdav?.path),
    ]
    const activeTabIndex = states.findIndex(Boolean)
    activeIndex.value = activeTabIndex === -1 ? 0 : activeTabIndex
}
const tabChange = (index: number) => {
    activeIndex.value = index
    hasUserSelectedTab.value = true
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

.settings-icon :deep(svg),
.settings-icon :deep(g),
.settings-icon :deep(path),
.settings-icon :deep(circle),
.settings-icon :deep(rect),
.settings-icon :deep(line),
.settings-icon :deep(polyline),
.settings-icon :deep(polygon) {
  fill:   var(--app-container_title-color) !important;
  stroke: var(--app-container_title-color) !important;
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
    margin-top: 10px;
    margin-bottom: 10px;
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
            background-color: #eff6ff;
            color: #1447e6;
        }

        .memory {
            background-color: #dbfce7;
            color: #008236;
        }

        .network {
            background-color: #faf5ff;
            color: #8200db;
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
