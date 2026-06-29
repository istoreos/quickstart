<template>
    <button class="btn cbi-button cbi-button-apply" @click="addTask()">{{ $gettext("新建") }}</button>
    <div class="cbi-section cbi-tblsection" id="cbi-nfs-mount">
        <table class="table cbi-section-table">
            <thead>
                <tr class="tr cbi-section-table-titles anonymous">
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("设备") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("类型") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("调度") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("功能") }}</th>
                </tr>
            </thead>
            <tbody>
                <tr class="tr cbi-section-table-row" v-for="(item, i) in config.tasks">
                    <td class="td cbi-value-field">
                        <b>
                            {{ item.devicePath }}
                        </b>
                    </td>
                    <td class="td cbi-value-field">
                        <b>
                            {{ getType(item.type) }}
                        </b>
                    </td>
                    <td class="td cbi-value-field">
                        <b>{{ item.month }}/{{ item.dayPerMonth }}/{{ item.hour }}</b>
                    </td>
                    <td class="td cbi-value-field">
                        <button class="btn cbi-button cbi-button-apply" :title="$gettext('调试')" @click="testTask(item)">{{ $gettext("预览") }}</button>
                        <button class="cbi-button cbi-button-remove" :title="$gettext('删除')" @click="delTask(i)">{{ $gettext("删除") }}</button>
                    </td>
                </tr>
            </tbody>
        </table>
    </div>
</template>
<script setup lang="ts">
import { PropType, reactive } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import { ActionSmartAddTask, ActionSmartTestTask } from "/@/components/smart"
const props = defineProps({
    config: {
        type: Object as PropType<PropsSmartConfig>,
        required: true,
    },
    saveData: {
        type: Function as PropType<(data: ResponseSmartConfig) => Promise<void>>,
        required: true,
    }
})
const config = reactive(props.config)
const getType = (e?: string) => {
    switch (e) {
        case "short":
            return $gettext("短暂自检")
        case "long":
            return $gettext("长时自检")
        case "conveyance":
            return $gettext("传输时自检")
        case "offline":
            return $gettext("离线时自检")
        default:
            return $gettext("未知")
    }
}
const addTask = () => {
    ActionSmartAddTask({
        config: props.config,
        disks: [],
        next: async (task) => {
            await props.saveData({
                tasks: [...config.tasks, task],
                global: props.config.global,
                devices: props.config.devices,
            })
            config.tasks = props.config.tasks || []

        }
    })
}
const delTask = async (i: number) => {
    const tasks = [...config.tasks]
    tasks.splice(i, 1)
    await props.saveData({
        tasks,
        global: props.config.global,
        devices: props.config.devices,
    })
    config.tasks = props.config.tasks || []
}
const testTask = (item: SmartConfigTask) => {
    ActionSmartTestTask({
        task: item,
    })
}
</script>
<style lang="scss" scoped>
</style>