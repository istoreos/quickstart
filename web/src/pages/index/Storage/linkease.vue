<template>
    <ul class="app-container_linkease">
        <li class="linkease-item">
            <div class="linkease-item_name">
                <span>{{ $gettext("当前状态:") }}</span>
            </div>
            <div class="linkease-item_value">
                <span class="configure" v-if="linkease?.enabel">{{ $gettext("已配置") }}</span>
                <span class="configure enabel" @click="onSetting()" v-else>{{ $gettext("未配置") }}</span>
            </div>
        </li>
        <template v-if="linkease?.enabel">
            <li class="linkease-item" v-if="linkease?.port">
                <div class="linkease-item_name">
                    <span>{{ $gettext("服务地址:") }}</span>
                </div>
                <div class="linkease-item_value">
                    <a :href="target" target="_blank" rel="noopener noreferrer">{{ target }}</a>
                </div>
            </li>
        </template>
        <div>
            <a href="https://www.linkease.com" target="_blank">{{ $gettext("下载易有云客户端，随时随地相册备份、远程访问") }}</a>
        </div>
    </ul>
</template>
<script setup lang="ts">
import { computed, PropType } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import ActionNAS from "/@/components/action-nas"
const props = defineProps({
    linkease: {
        type: Object as PropType<NasServiceLinkeaseInfo>,
    }
})
const target = computed(() => {
    return `http://${location.hostname}:${props.linkease?.port}`
})
const onSetting = () => {
    ActionNAS({
        setup: 0
    })
}
</script>
<style lang="scss" scoped>
li.linkease-item {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    margin: 1rem 0;

    .linkease-item_name {
        flex: 0 0 100%;
        max-width: 50%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding-right: 10px;
        color: var(--app-container_title-color);
    }

    .linkease-item_value {
        flex: 0 0 100%;
        max-width: 50%;
        padding-left: 10px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: var(--app-container_title-color);

        .configure {
            color: #297ff3;
            padding: 3px;
        }

        .configure.enabel {
            cursor: pointer;
        }
    }
}

a {
    text-decoration: none;
    color: #297ff3;
}
</style>