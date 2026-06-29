<template>
    <ul class="app-container_qbittorrent">
        <li class="qbittorrent-item">
            <div class="qbittorrent-item_name">
                <span>{{ $gettext("当前状态:") }}</span>
            </div>
            <div class="qbittorrent-item_value">
                <span class="configure" v-if="qbittorrent?.status == 'running'">{{ $gettext("已启动") }}</span>
                <span class="configure enabel" v-else>{{ $gettext("未启动") }}</span>
            </div>
        </li>
        <template v-if="qbittorrent?.status == 'running'">
            <li class="qbittorrent-item">
                <div class="qbittorrent-item_name">
                    <span>{{ $gettext("下载目录:") }}</span>
                </div>

                <div class="qbittorrent-item_value">
                    <a target="_blank" :href="'/cgi-bin/luci/admin/services/linkease/file/?path=/root' + qbittorrent?.downloadPath">{{ qbittorrent?.downloadPath }}</a>
                </div>
            </li>

            <li class="qbittorrent-item">
                <div class="qbittorrent-item_name">
                    <span>{{ $gettext("网络地址:") }}</span>
                </div>

                <div class="qbittorrent-item_value">
                    <a :href="target" target="_blank" rel="noopener noreferrer">{{ target }}</a>
                </div>
            </li>
            <li class="qbittorrent-item">
                <div class="qbittorrent-item_name right">
                    <span>{{ $gettext("默认用户名：") }}admin</span>
                </div>

                <div class="qbittorrent-item_value">
                    <span>{{ $gettext("默认密码：") }}adminadmin</span>
                </div>
            </li>
        </template>
    </ul>
</template>
<script setup lang="ts">
import { computed, PropType } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import ActionNAS from "/@/components/action-nas"
const props = defineProps({
    qbittorrent: {
        type: Object as PropType<GuideDownloadServiceqBittorrentInfo>,
    }
})
const target = computed(() => {
    return `http://${location.hostname}${props.qbittorrent?.webPath}`
})
const onSetting = () => {
    ActionNAS({
        setup: 0
    })
}
</script>
<style lang="scss" scoped>
li.qbittorrent-item {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    margin: 1rem 0;

    .qbittorrent-item_name {
        flex: 0 0 100%;
        max-width: 50%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding-right: 10px;
        color: var(--app-container_title-color);
    }

    .qbittorrent-item_value {
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

a {
    text-decoration: none;
    color: #297ff3;
}
</style>