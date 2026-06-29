<template>
    <li class="webdav-item">
        <div class="webdav-item_name">
            <span>{{ $gettext("当前状态:") }}</span>
        </div>
        <div class="webdav-item_value">
            <span>{{ webdav?.path ? $gettext('已启用') : $gettext('未启用') }}</span>
        </div>
    </li>
    <li class="webdav-item" v-if="webdav?.path">
        <div class="webdav-item_name">
            <span>{{ $gettext("挂载路径:") }}</span>
        </div>
        <div class="webdav-item_value">
            <a target="_blank" :href="'/cgi-bin/luci/admin/services/linkease/file/?path=/root' + webdav?.path">{{ webdav?.path }}</a>
        </div>
    </li>
    <li class="webdav-item" v-if="webdav?.port">
        <div class="webdav-item_name">
            <span>{{ $gettext("服务路径:") }}</span>
        </div>
        <div class="webdav-item_value">
            <a :href="target" target="_blank" rel="noopener noreferrer">{{ target }}</a>
        </div>
    </li>
    <li class="webdav-item" v-if="webdav?.username">
        <div class="webdav-item_name">
            <span>{{ $gettext("账号:") }}</span>
        </div>
        <div class="webdav-item_value">
            <span>{{ webdav?.username }}</span>
        </div>
    </li>
</template>
<script setup lang="ts">
import { computed, PropType } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

const props = defineProps({
    webdav: {
        type: Object as PropType<NasServiceWebdavInfo>,
    }
})
const target = computed(() => {
    return `http://${location.hostname}:${props.webdav?.port}`
})
</script>
<style lang="scss" scoped>
li.webdav-item {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    margin: 1rem 0;
    .webdav-item_name {
        flex: 0 0 100%;
        max-width: 50%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding-right: 10px;
        color: var(--app-container_title-color);
    }

    .webdav-item_value {
        flex: 0 0 100%;
        max-width: 50%;
        padding-left: 10px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: var(--app-container_title-color);
    }
}
</style>