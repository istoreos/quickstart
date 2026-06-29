<template>
    <div class="actioner-container">
        <div class="actioner-container_header">
            <span>{{ $gettext("域名配置向导") }}</span>
        </div>
        <div class="actioner-container_body ddnsto-bind">
            <iframe :src="addrURL" />
        </div>
    </div>
</template>
<script setup lang="ts">
import request from "/@/request";
import { computed, onMounted, onUnmounted, PropType, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

const props = defineProps({
    onSetup: {
        type: Function as PropType<(v?: string) => void>,
        required: true
    },
    config: {
        type: Object as PropType<{
            domain: string,
            sign: string
            token: string
            routerId: string
            netaddr: string
        }>,
        required: true
    },
    domain: {
        type: String,
        required: true
    }
})
const emit = defineEmits([
    "update:domain"
])

const addrURL = computed(() => {
    const { domain, token, sign, routerId, netaddr } = props.config
    const a = encodeURIComponent(domain)
    const b = encodeURIComponent(netaddr)
    return `https://www.kooldns.cn/bind/#/domain?domain=${a}&sign=${sign}&token=${token}&routerId=${routerId}&netaddr=${b}`
})
const onCallback = (e: MessageEvent<{
    auth?: string,
    success?: number,
    prefix?: string, // ddnsto 设置的远程域名 前缀
    url?: string,        // ddnsto 设置的远程域名
}>) => {
    if (e.data) {
        const { auth, url } = e.data
        if (auth === "ddnsto" && url) {
            DdnstoAddress(url)
        }
    }
}

const DdnstoAddress = async (url: string) => {
    try {
        const res = await request.Guide.DdnstoAddress.POST({
            address: url
        })
        if (res?.data) {
            if ((res?.data?.success || 0) == 0) {
                emit("update:domain", url)
                props.onSetup("ddnsto-save")
            }
        }

    } catch {

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
