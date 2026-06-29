<template>
    <ul class="app-container_transmission">
        <li class="transmission-item">
            <div class="transmission-item_name">
                <span>{{ $gettext("当前状态:") }}</span>
            </div>
            <div class="transmission-item_value">
                <span class="configure" v-if="transmission?.status == 'running'">{{ $gettext("已启动") }}</span>
                <span class="configure enabel" v-else>{{ $gettext("未启动") }}</span>
            </div>
        </li>
        <template v-if="transmission?.status == 'running'">
            <li class="transmission-item">
                <div class="transmission-item_name">
                    <span>{{ $gettext("下载目录:") }}</span>
                </div>

                <div class="transmission-item_value">
                    <a target="_blank" :href="'/cgi-bin/luci/admin/services/linkease/file/?path=/root' + transmission?.downloadPath">{{ transmission?.downloadPath }}</a>
                </div>
            </li>

            <li class="transmission-item">
                <div class="transmission-item_name">
                    <span>{{ $gettext("网络地址:") }}</span>
                </div>

                <div class="transmission-item_value">
                    <a :href="target" target="_blank" rel="noopener noreferrer">{{ target }}</a>
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
    transmission: {
        type: Object as PropType<GuideDownloadServiceTransmissionInfo>,
    }
})
const target = computed(() => {
    return `http://${location.hostname}${props.transmission?.webPath}`
})
const onSetting = () => {
    ActionNAS({
        setup: 0
    })
}
</script>
<style lang="scss" scoped>
li.transmission-item {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    margin: 1rem 0;

    .transmission-item_name {
        flex: 0 0 100%;
        max-width: 50%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding-right: 10px;
        color: var(--app-container_title-color);
    }

    .transmission-item_value {
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