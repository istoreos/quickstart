<template>
    <a :href="url" target="_blank" :title="$gettext('跳转教程')">
        <svg t="1684144670421" class="icon" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg"
            p-id="4343">
            <path
                d="M512 74.666667c241.066667 0 437.333333 196.266667 437.333333 437.333333S753.066667 949.333333 512 949.333333 74.666667 753.066667 74.666667 512 270.933333 74.666667 512 74.666667zM512 704c-23.466667 0-42.666667 19.2-42.666667 42.666667s19.2 42.666667 42.666667 42.666666 42.666667-19.2 42.666667-42.666666-19.2-42.666667-42.666667-42.666667z m0-458.666667c-76.8 0-138.666667 61.866667-138.666667 138.666667 0 17.066667 14.933333 32 32 32s32-14.933333 32-32c0-40.533333 34.133333-74.666667 74.666667-74.666667s74.666667 34.133333 74.666667 74.666667c0 2.133333 0 6.4-2.133334 10.666667-6.4 14.933333-19.2 32-40.533333 51.2-10.666667 10.666667-21.333333 19.2-34.133333 27.733333-2.133333 2.133333-6.4 4.266667-8.533334 6.4l-6.4 4.266667c-8.533333 6.4-14.933333 17.066667-14.933333 27.733333v108.8c2.133333 17.066667 14.933333 29.866667 32 29.866667h2.133333c17.066667-2.133333 29.866667-14.933333 29.866667-32v-89.6l12.8-10.666667c10.666667-8.533333 19.2-17.066667 29.866667-25.6 27.733333-25.6 46.933333-49.066667 57.6-74.666667 4.266667-10.666667 6.4-23.466667 6.4-34.133333 0-76.8-61.866667-138.666667-138.666667-138.666667z"
                fill="#666666" p-id="4344"></path>
        </svg>
    </a>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

type HelpTyep =
    "disk" | //磁盘信息
    "store" |//存储服务
    "docker" | //docker
    "download" | //下载服务
    "ddns" | //远程域名
    "network-interface" //网络接口
const props = defineProps<{
    type: HelpTyep
}>()
const url = computed(() => {
    switch (props.type) {
        case "disk":
            return "https://www.linkease.com/rd/8myYAEVA/"
        case "store":
            return "https://www.linkease.com/rd/1F58VUTT/"
        case "docker":
            return "https://www.linkease.com/rd/2Q28MDtf/"
        case "download":
            return "https://www.linkease.com/rd/1tJo1KX-/"
        case "ddns":
            return "https://www.linkease.com/rd/3yFiX5-X/"
        case "network-interface":
            return "https://www.linkease.com/rd/3ca51a3G/"
    }
})
</script>
<style lang="scss" scoped>
a {
    margin-left: 4px;
    width: 20px;
    vertical-align: middle;

    svg {
        width: 22px;
        height: 22px;
    }

    &:hover {
        svg {
            path {
                fill: #3688FF;
            }
        }
    }
}
</style>