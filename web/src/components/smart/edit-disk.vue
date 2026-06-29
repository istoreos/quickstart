<template>
    <app>
        <div class="actioner-container">
            <div class="actioner-container_header">
                <span>
                    S.M.A.R.T. » {{ $gettext("设备") }} » {{ disk.path }}
                </span>
            </div>
            <div class="actioner-container_body">
                <div class="cbi-value">
                    <label class="cbi-value-title">
                        {{ $gettext("磁盘") }}
                    </label>
                    <div class="cbi-value-field">
                        <div class="cbi-value-description">
                            {{ disk.model }} [ {{ disk.path }}，{{ disk.sizeStr }} ]
                        </div>
                    </div>
                </div>
                <div class="cbi-value">
                    <label class="cbi-value-title">
                        {{ $gettext("温度监测（差异）") }}
                    </label>
                    <div class="cbi-value-field">
                        <div class="cbi-checkbox">
                            <select class="cbi-input-select" v-model.number="config.tmpDiff">
                                <option :value="-1">{{ $gettext("使用全局配置") }}</option>
                                <option :value="0">{{ $gettext("禁用") }}</option>
                                <option :value="item" v-for="item in 20">{{ item }}°C</option>
                            </select>
                        </div>
                        <div class="cbi-value-description">
                            {{ $gettext("自上次报告以来温度变化至少 N 度，则需报告.") }}
                        </div>
                    </div>
                </div>
                <div class="cbi-value">
                    <label class="cbi-value-title">
                        {{ $gettext("温度监测（最大）") }}
                    </label>
                    <div class="cbi-value-field">
                        <div class="cbi-checkbox">
                            <select class="cbi-input-select" v-model.number="config.tmpMax">
                                <option :value="-1">{{ $gettext("使用全局配置") }}</option>
                                <option :value="0">{{ $gettext("禁用") }}</option>
                                <option :value="item * 5" v-for="item in 20">{{ item * 5 }}°C</option>
                            </select>
                        </div>
                        <div class="cbi-value-description">
                            {{ $gettext("如果温度大于或等于 N 摄氏度则报告.") }}
                        </div>
                    </div>
                </div>
            </div>
            <div class="actioner-container_footer">
                <button class="close" @click="onClose" :disabled="disabled">{{ $gettext("取消") }}</button>
                <button class="next" @click="onNext" :disabled="disabled">{{ $gettext("保存") }}</button>
            </div>
        </div>
    </app>
</template>
<script setup lang="ts">
import { PropType, reactive, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import app from "./app.vue"
const props = defineProps({
    close: {
        type: Function,
        required: true
    },
    disk: {
        type: Object as PropType<SmartDiskInfo>,
        required: true
    },
    device: {
        type: Object as PropType<ActionSmartEditDiskPropsDevice>,
    },
    next: {
        type: Function as PropType<ActionSmartEditDiskPropsNext>,
        required: true
    }
})
console.log(props.device);

const disabled = ref(false)
const config = reactive({
    tmpDiff: props.device?.tmpDiff || 0,
    tmpMax: props.device?.tmpMax || 0,
    devicePath: props.device?.devicePath || ""
})
const onClose = () => {
    disabled.value = true
    props.close()
}
const onNext = async () => {
    disabled.value = true
    try {
        await props.next({
            tmpDiff: config.tmpDiff,
            tmpMax: config.tmpMax,
            devicePath: config.devicePath
        })
        disabled.value = false
        onClose()
    } catch (error) {
    }
}
</script>
