<template>
    <div class="actioner-container">
        <div class="actioner-container_header">
            <span>{{ $gettext("域名配置向导") }}</span>
        </div>
        <div class="actioner-container_body">
            {{ msg }}
        </div>
    </div>
</template>
<script setup lang="ts">
import { onMounted, onUnmounted, PropType, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import request from "/@/request";
const props = defineProps({
    onSetup: {
        type: Function as PropType<(v?: string) => void>,
        required: true
    },
    token: {
        type: String,
        required: true
    },
    onDdnstoLocalConfig: {
        type: Function as PropType<(netaddr: string, routerId: string) => void>,
        required: true
    },
})
const msg = ref<any>($gettext("正在检测插件是否已启动..."))
const updateDdnstoToken = async (token: string) => {
    try {
        const res = await request.Guide.Ddnsto.POST({
            token: props.token
        })
        if (res?.data) {
            if (res.data.error) {
                msg.value = res.data.error
            }
            if ((res?.data?.success || 0) == 0) {
                checkDdnstoRunning()
            }
        }
    } catch (error) {
        msg.value = error
    }
}
updateDdnstoToken(props.token)
const interval = ref<any>()
const checkDdnstoRunning = () => {
    const tick = async () => {
        const res = await onDdnstoRunning()
        if (res === true) {
            getDdntoConfig()
            return
        }
        interval.value = window.setTimeout(tick, 2000)
    }
    tick()
}
const onDdnstoRunning = async () => {
    try {
        const res = await request.App.Check.POST({
            name: "ddnsto",
            checkRunning: true
        })
        if (res?.data) {
            if (res.data.error) {
                msg.value = res.data.error
            }
            const result = res.data.result
            if (result?.status == "running") {
                return true
            }
        }
    } catch (error) {
        msg.value = error
    }

    return false
}

onUnmounted(() => {
    if (interval.value) {
        clearInterval(interval.value)
    }
})

const getDdntoConfig = async () => {
    try {
        const res = await request.Guide.DdntoConfig.GET()
        if (res?.data) {
            if (res.data.error) {
                msg.value = res.data.error
            }
            if ((res?.data?.success || 0) == 0) {
                if (res.data.result) {
                    const result = res.data.result
                    props.onDdnstoLocalConfig(result.netAddr, result.deviceId)
                    props.onSetup("ddnsto-bind")
                }
            }
        }

    } catch (error) {
        msg.value = error
    }
}
</script>
<style lang="scss" scoped>
.actioner-container_body {
    display: flex;
    align-items: center;
    justify-content: center;
}
</style>
