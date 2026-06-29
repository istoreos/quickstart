<template>
    <app>
        <div class="actioner-container">
            <div class="actioner-container_header">
                <span>
                    {{ $gettext("运行调试") }}
                </span>
            </div>
            <div class="actioner-container_body">
                <textarea :value="resultText + `\n` + logText + '\n' + loadText" disabled></textarea>
            </div>
            <div class="actioner-container_footer">
                <div class="close" @click="onClose" :disabled="disabled">{{ $gettext("关闭") }}</div>
                <div class="next" @click="onTest" :disabled="disabled" v-if="!disabled">{{ $gettext("运行") }}</div>
            </div>
        </div>
    </app>
</template>
<script setup lang="ts">
import { onBeforeUnmount, PropType, reactive, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import app from "./app.vue"
import request from "/@/request";
import utils from "/@/utils"

const props = defineProps({
    close: {
        type: Function,
        required: true
    },
    task: {
        type: Object as PropType<SmartConfigTask>,
        required: true
    }
})
const disabled = ref(false)
const loadText = ref<any>("")
const resultText = ref<any>("")
const logText = ref<any>("")
const getData = async () => {
    loadText.value += "."
    try {
        const res = await request.Smart.Test.Result.POST({
            type: "selftest",
            devicePath: props.task.devicePath || "",
        })
        if (res.data) {
            const { result, error } = res.data
            if (result && result.result) {
                logText.value = result.result
            }
            if (error) {
                logText.value = error
            }
        }
    } catch (error) {
        if (error) {
            logText.value = error
        }
    }
}
const cancelGetData = utils.easyInterval(getData, 5000)

onBeforeUnmount(() => {
    cancelGetData()
})
const onClose = () => {
    disabled.value = true
    cancelGetData()
    props.close()

}
const onTest = async () => {
    disabled.value = true
    try {
        const res = await request.Smart.Test.POST({
            type: props.task.type || "short",
            devicePath: props.task.devicePath || "",
        })
        if (res.data) {
            const { success, error, result } = res.data
            if (error) {
                resultText.value = error
            }
            if (result && result.result) {
                resultText.value = result.result
            }
        }
    } catch (error) {
        resultText.value = error
    } finally {
    }

}
</script>
<style lang="scss" scoped>
textarea {
    display: block;
    width: 100%;
    height: 400px;
    padding: 1rem;
    font-size: 14px;
    resize: none;
    border: none;
    background-color: #1e1e1e;
    color: #fff;
}
</style>
