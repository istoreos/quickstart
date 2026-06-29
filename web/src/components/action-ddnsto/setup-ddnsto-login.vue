<template>
    <div class="actioner-container">
        <div class="actioner-container_header">
            <span>{{ $gettext("域名配置向导") }}</span>
        </div>
        <div class="actioner-container_body ddnsto-login">
            <iframe src="https://www.kooldns.cn/bind/#/auth?send=1&source=openwrt&callback=*" />
        </div>
        <div class="actioner-container_footer">
            <div class="close" @click="onClose">{{ $gettext("取消") }}</div>
        </div>
    </div>
</template>
<script setup lang="ts">
import { onMounted, onUnmounted, PropType, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

const props = defineProps({
    onSetup: {
        type: Function as PropType<(v?: string) => void>,
        required: true
    },
    onDdnstoConfig: {
        type: Function as PropType<(sign: string, token: string) => void>,
        required: true
    }
})
const onClose = () => {
    props.onSetup()
}
const onCallback = (e: MessageEvent<{
    auth?: string
    sign?: string
    token?: string
}>) => {
    if (e.data.auth == "ddnsto") {
        const sign = e.data.sign
        const token = e.data.token
        if (sign && token) {
            removeEventListener("message", onCallback)
            props.onDdnstoConfig(sign, token)
            props.onSetup("ddnsto-run")
        }
    }
}
onMounted(() => {
    window.addEventListener("message", onCallback)
})
onUnmounted(() => {
    removeEventListener("message", onCallback)
})
</script>
<style lang="scss" scoped>
iframe {
    width: 100%;
    height: 100%;
    border: none;
}
</style>
