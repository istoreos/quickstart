<template>
    <app>
        <div class="actioner-container">
            <div class="actioner-container_header">
                <ul class="tabs">
                    <li :class="{
                        'active cbi-tab': setup == 'info'
                    }" @click="onSetup('info')">
                        <a>{{ $gettext("设备信息") }}</a>
                    </li>
                    <li :class="{
                        'active cbi-tab': setup == 'attribute'
                    }" @click="onSetup('attribute')">
                        <a>{{ $gettext("属性") }}</a>
                    </li>
                    <li :class="{
                        'active cbi-tab': setup == 'log'
                    }" @click="onSetup('log')">
                        <a>{{ $gettext("自检日志") }}</a>
                    </li>
                    <li :class="{
                        'active cbi-tab': setup == 'extend'
                    }" @click="onSetup('extend')">
                        <a>{{ $gettext("扩展信息") }}</a>
                    </li>
                </ul>
            </div>
            <div class="actioner-container_body">
                <template v-if="setup == 'info'">
                    <table class="table">
                        <tr class="tr">
                            <td class="td left">{{ $gettext("设备") }}</td>
                            <td class="td left"> {{ disk.path }}</td>
                        </tr>
                        <tr class="tr">
                            <td class="td left">{{ $gettext("型号") }}</td>
                            <td class="td left"> {{ disk.model }}</td>
                        </tr>
                        <tr class="tr">
                            <td class="td left">{{ $gettext("序号") }}</td>
                            <td class="td left"> {{ disk.serial }}</td>
                        </tr>
                    </table>
                </template>
                <template v-else-if="setup == 'attribute'">
                    <textarea disabled :value="text.attribute"></textarea>

                </template>
                <template v-else-if="setup == 'log'">
                    <textarea disabled :value="text.log"></textarea>

                </template>
                <template v-else-if="setup == 'extend'">
                    <textarea disabled :value="text.extend"></textarea>
                </template>
            </div>
            <div class="actioner-container_footer">
                <div class="close" @click="onClose" :disabled="disabled">{{ $gettext("关闭") }}</div>
            </div>
        </div>
    </app>
</template>
<script setup lang="ts">
import { PropType, reactive, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import app from "./app.vue"
import request from "/@/request";
const props = defineProps({
    close: {
        type: Function,
        required: true
    },
    disk: {
        type: Object as PropType<SmartDiskInfo>,
        required: true
    }

})
const disabled = ref(false)

const setup = ref<"info" | "attribute" | "log" | "extend">("info")
const onSetup = (type: "info" | "attribute" | "log" | "extend") => {
    setup.value = type
    switch (type) {
        case "info":
            break;
        case "attribute":
            getAttribute()
            break;
        case "log":
            getLog();
            break;
        case "extend":
            getExtend()
            break;
    }
}
const onClose = () => {
    disabled.value = true
    props.close()
}
const text = reactive<{
    log: any
    attribute: any,
    extend: any
}>({
    log: "",
    attribute: "",
    extend: ""

})
const getLog = async () => {
    try {
        const res = await request.Smart.Test.Result.POST({
            type: "selftest",
            devicePath: props.disk.path || "",
        })
        if (res.data) {
            const { result, error } = res.data
            if (result && result.result) {
                text.log = result.result
            }
            if (error) {
                text.log = error
            }
        }
    } catch (error) {
        text.log = error
    }
}
const getAttribute = async () => {
    try {
        const res = await request.Smart.Attribute.Result.POST({
            devicePath: props.disk.path || "",
        })
        if (res.data) {
            const { result, error } = res.data
            if (result && result.result) {
                text.attribute = result.result
            }
            if (error) {
                text.attribute = error
            }
        }
    } catch (error) {
        text.attribute = error
    }
}
const getExtend = async () => {
    try {
        const res = await request.Smart.Extend.Result.POST({
            devicePath: props.disk.path || "",
        })
        if (res.data) {
            const { result, error } = res.data
            if (result && result.result) {
                text.extend = result.result
            }
            if (error) {
                text.extend = error
            }
        }
    } catch (error) {
        text.extend = error
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
