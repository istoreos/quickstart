<template>
    <div class="actioner-container">
        <div class="actioner-container_header">
            <span>{{ $gettext("域名配置向导") }}</span>
        </div>
        <div class="actioner-container_body">
            <span>{{ msg }}</span>
        </div>
        <div class="actioner-container_footer">
            <template v-if="status">
                <div class="close" @click="onClose">{{ $gettext("取消") }}</div>
                <div class="next" @click="onNext">{{ $gettext("确定") }}</div>
            </template>
        </div>
    </div>
</template>
<script setup lang="ts">
import { onMounted, onUnmounted, PropType, ref } from "vue";
import Toast from "../toast";
import request from "/@/request";
import appUtils from "/@/utils/app";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

const props = defineProps({
    onSetup: {
        type: Function as PropType<(v?: string) => void>,
        required: true
    },
})
const onClose = () => {
    props.onSetup()
}
const onNext = async () => {
    if (disabled.value) {
        return
    }
    disabled.value = true
    const load = Toast.Loading($gettext("安装中..."))
    try {
        const res = await appUtils.installApp("app-meta-ddnsto", 30)
        if (res) {
            props.onSetup('ddnsto-login')
            return
        } else {
            msg.value = $gettext("安装失败")
        }
    } catch (err) {
        msg.value = err
    } finally {
        load.Close()
    }
    disabled.value = false
}
const msg = ref<any>($gettext("正在检测中..."))
const status = ref(false)
const disabled = ref(false)
const checkIsInstallDdnsto = async () => {
    try {
        const res = await request.App.Check.POST({
            name: "ddnsto"
        })
        if (res?.data) {
            const { result, error } = res.data
            if (error) {
                msg.value = error
                return
            }
            if (result) {
                if (result.status == "installed") {
                    props.onSetup('ddnsto-login')
                    return
                }
                if (result.status == "uninstalled") {
                    msg.value = $gettext("需要安装DDNSTO插件，点击“确定”开始安装")
                }
            }
        }
    } catch (error) {
        msg.value = error
    }
    status.value = true
}

checkIsInstallDdnsto()

</script>
<style lang="scss" scoped>
.actioner-container_body {
    display: flex;
    align-items: center;
    justify-content: center;
}
</style>
