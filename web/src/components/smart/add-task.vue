<template>
    <app>
        <div class="actioner-container">
            <div class="actioner-container_header">
                <span>
                    {{ $gettext("创建计划任务") }}
                </span>
            </div>
            <div class="actioner-container_body">
                <div class="cbi-value">
                    <label class="cbi-value-title">
                        {{ $gettext("磁盘") }}
                    </label>
                    <div class="cbi-value-field">
                        <div class="cbi-checkbox">
                            <select class="cbi-input-select" v-model.trim="task.devicePath">
                                <option value="">{{ $gettext("选择磁盘") }}</option>
                                <option :value="item.path" v-for="item in disks">
                                    {{ item.model }} [ {{ item.path }}，{{ item.sizeStr }} ] </option>
                            </select>
                        </div>
                    </div>
                </div>
                <div class="cbi-value">
                    <label class="cbi-value-title">
                        {{ $gettext("类型") }}
                    </label>
                    <div class="cbi-value-field">
                        <div class="cbi-checkbox">
                            <select class="cbi-input-select" v-model.trim="task.type">
                                <option value="short">{{ $gettext("短暂自检") }}</option>
                                <option value="long">{{ $gettext("长时自检") }}</option>
                                <option value="conveyance">{{ $gettext("传输时自检") }}</option>
                                <option value="offline">{{ $gettext("离线时自检") }}</option>
                            </select>
                        </div>
                    </div>
                </div>
                <div class="cbi-value">
                    <label class="cbi-value-title">
                        {{ $gettext("小时") }}
                    </label>
                    <div class="cbi-value-field">
                        <div class="cbi-checkbox">
                            <select class="cbi-input-select" v-model.trim="task.hour">
                                <option value="*">*</option>
                                <option :value="`${i}`" v-for="(item, i) in 24">{{ i }}</option>
                            </select>
                        </div>
                        <div class="cbi-value-description">
                            {{ $gettext("* 表示每小时") }}
                        </div>
                    </div>
                </div>
                <div class="cbi-value">
                    <label class="cbi-value-title">
                        {{ $gettext("天") }}
                    </label>
                    <div class="cbi-value-field">
                        <div class="cbi-checkbox">
                            <select class="cbi-input-select" v-model.trim="task.dayPerMonth">
                                <option value="*">*</option>
                                <option :value="`${item}`" v-for="(item) in 31">{{ item }}</option>
                            </select>
                        </div>
                        <div class="cbi-value-description">
                            {{ $gettext("* 表示每天") }}
                        </div>
                    </div>
                </div>
                <div class="cbi-value">
                    <label class="cbi-value-title">
                        {{ $gettext("月") }}
                    </label>
                    <div class="cbi-value-field">
                        <div class="cbi-checkbox">
                            <select class="cbi-input-select" v-model.trim="task.month">
                                <option value="*">*</option>
                                <option :value="`${item}`" v-for="(item, i) in 12">{{ item }}</option>
                            </select>
                        </div>
                        <div class="cbi-value-description">
                            {{ $gettext("* 表示每月") }}
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

import Toast from "../toast";
import app from "./app.vue"
import request from "/@/request";
const props = defineProps({
    close: {
        type: Function,
        required: true
    },
    config: {
        type: Object as PropType<PropsSmartConfig>,
        required: true
    },
    next: {
        type: Function as PropType<ActionSmartAddTaskPropsNext>,
        required: true
    }
})
const disabled = ref(false)
const task = reactive<SmartConfigTask>({
    type: "short",//检查类型
    devicePath: "", //磁盘路径
    month: "*", //月份,01 (January) to 12 (December)
    dayPerMonth: "*",//每月的第几天,01 to 31
    hour: "*",//每天的第几个小时,00 (midnight to just before 1 am) to 23 (11pm to just before midnight)
})
const disks = ref<SmartDisks>([])
const getData = async () => {
    try {
        const res = await request.Smart.List.GET()
        if (res.data) {
            const { result, error } = res.data
            if (result && result.disks) {
                disks.value = result.disks
            }
        }
    } catch (error) {
    }
}
getData();

const onClose = () => {
    disabled.value = true
    props.close()
}
const onNext = async () => {
    if (task.devicePath == "") {
        Toast.Warning($gettext("请选择磁盘"))
        return
    }
    disabled.value = true

    try {
        await props.next(task)
        onClose()
    } catch (error) {
    } finally {
        disabled.value = false
    }
}
</script>
