<template>
    <ul class="app-container_aria2">
        <li class="aria2-item">
            <div class="aria2-item_name">
                <span>{{ $gettext("当前状态:") }}</span>
            </div>
            <div class="aria2-item_value">
                <span class="configure" v-if="aria2?.status == 'running'">{{ $gettext("已启动") }}</span>
                <span class="configure enabel" v-else>{{ $gettext("未启动") }}</span>
            </div>
        </li>
        <template v-if="aria2?.status == 'running'">
            <li class="aria2-item">
                <div class="aria2-item_name">
                    <span>{{ $gettext("下载目录:") }}</span>
                </div>

                <div class="aria2-item_value">
                    <a target="_blank" :href="'/cgi-bin/luci/admin/services/linkease/file/?path=/root' + aria2?.downloadPath">{{ aria2?.downloadPath }}</a>
                </div>
            </li>

            <li class="aria2-item">
                <div class="aria2-item_name">
                    <span>{{ $gettext("网络地址:") }}</span>
                </div>

                <div class="aria2-item_value">
                    <a :href="target" target="_blank" rel="noopener noreferrer">{{ target }}</a>
                </div>
            </li>

            <li class="aria2-item">
                <div class="aria2-item_name right">
                    <span>{{ $gettext("认证失败？") }}</span>
                </div>

                <div class="aria2-item_value">
                    <a :href="ariaNgAutoConf" target="_blank" rel="noopener noreferrer">{{ $gettext("点此自动配置 AriaNg") }}</a>
                </div>
            </li>
        </template>
        <div class="use-url_app">
            <a href="https://doc.linkease.com/zh/guide/linkease_app/tutorial.html#%E8%BF%9C%E7%A8%8B%E4%B8%8B%E8%BD%BD"
                target="_blank">{{ $gettext("使用易有云APP，随时随地远程下载") }}</a>
        </div>
    </ul>
</template>
<script setup lang="ts">
import { computed, PropType } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import ActionNAS from "/@/components/action-nas"
const props = defineProps({
    aria2: {
        type: Object as PropType<GuideDownloadServiceAria2Info>,
    }
})
const target = computed(() => {
    // ariang served by luci http server, so location.origin
    return `${location.origin}${props.aria2?.webPath}`
})
const ariaNgAutoConf = computed(() => {
    let token = props.aria2?.rpcToken
    if (token) {
        token = encodeURIComponent(btoa(token))
    } else {
        token = ""
    }
    const hostname = encodeURIComponent(location.hostname)
    return `${location.origin}${props.aria2?.webPath}/#!/settings/rpc/set/http/${hostname}/${props.aria2?.rpcPort}/jsonrpc/${token}`
})
const onSetting = () => {
    ActionNAS({
        setup: 0
    })
}

</script>
<style lang="scss" scoped>
li.aria2-item {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    margin: 1rem 0;

    .aria2-item_name {
        flex: 0 0 100%;
        max-width: 50%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding-right: 10px;
        color: var(--app-container_title-color);
    }

    .aria2-item_value {
        flex: 0 0 100%;
        max-width: 50%;
        padding-left: 10px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;

        .configure {
            color: #297ff3;
            padding: 3px;
        }

        .configure.enabel {
            color: #888;
        }
    }
}

.use-url_app {
    a {
        text-decoration: none;
        color: #297ff3;
    }

    padding-bottom: 14px;
}
</style>