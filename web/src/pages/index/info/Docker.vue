<template>
    <Card title="Docker" :showSettings="true" @footer-click="onSetting" style="width: 100%;height: 100%; display: block;" :is-settings-menu-open="isSettingsMenuOpen" @update:isSettingsMenuOpen="(val: boolean) => isSettingsMenuOpen = val">
        <template #icon>
            <dockerIcon color="#155dfc" class="icon" />
        </template>
        <template #settings>
            <div class="btn_settings" @click="onSetting">
                <dockerIcon color="#0a0a0a" class="icon1 dockerIcon" style="margin-right: 6px;" />
                <span>{{ $gettext('Docker迁移') }}</span>
                <div class="rotation" @click.stop="isSettingsMenuOpen = !isSettingsMenuOpen" v-if="docker?.status === 'running'">
                    <moreItem class="moreIcon" />
                </div>
            </div>
        </template>
        <template #settings-menu v-if="docker?.status === 'running'">
            <div><a href="/cgi-bin/luci/admin/docker/overview">{{ $gettext("Docker高级配置") }}</a></div>
        </template>
        <div class="content" v-if="load">
            <statusVue :docker="docker" />
        </div>
        <div v-else class="content" style="display: flex;justify-content: center;">
            <icon-loading :size="40" color="currentColor" />
        </div>
    </Card>
</template>

<script lang="ts" setup>
import Card from "../components/Card.vue"
import dockerIcon from "/@/components/svg/docker.vue"
import statusVue from './status.vue';
import moreItem from "/@/components/svg/more.vue"

import { ref } from "vue";
import request from '/@/request';
import ActionDocker from "/@/components/action-docker";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

const load = ref(false)
const docker = ref<GuideDockerStatus>()

const isSettingsMenuOpen = ref(false)

const onSetting = () => {
    ActionDocker({
        setup: 0
    })
}
const getData = () => {
    request.Guide.DockerStatus.GET().then(res => {
        if (res?.data?.result) {
            const result = res.data.result
            docker.value = result
        }
    }).finally(() => {
        load.value = true
    })
}
setTimeout(getData, 1100)
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

:deep(.dockerIcon) {
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
    margin-top: 10px;
    margin-bottom: 10px;
    font-weight: normal;
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