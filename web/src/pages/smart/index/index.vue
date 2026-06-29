<template>
    <fieldset class="cbi-section">
        <div class="cbi-value">
            <label class="cbi-value-title">
                {{ $gettext("启用") }}
            </label>
            <div class="cbi-value-field">
                <div class="cbi-checkbox">
                    <input type="checkbox" v-model="config.global.enable" :value="!config.global.enable">
                </div>
            </div>
        </div>
        <!-- <div class="cbi-value">
            <label class="cbi-value-title">
                检测周期（秒）
            </label>
            <div class="cbi-value-field">
                <div class="cbi-checkbox">
                    <input type="text" :value="1800" disabled>
                </div>
                <div class="cbi-value-description">
                    将磁盘检测的间隔时间设置为N秒，记录S.M.A.R.T.的错误和属性的变化。
                </div>
            </div>
        </div> -->
        <div class="cbi-value">
            <label class="cbi-value-title">
                {{ $gettext("电源模式") }}
            </label>
            <div class="cbi-value-field">
                <div class="cbi-checkbox">
                    <select class="cbi-input-select" v-model.trim="config.global.powermode">
                        <option value="never">{{ $gettext("总是") }}</option>
                        <option value="sleep">{{ $gettext("睡眠") }}</option>
                        <option value="standby">{{ $gettext("待机") }}</option>
                        <option value="idle">{{ $gettext("闲置") }}</option>
                    </select>
                </div>
                <div class="cbi-value-description">
                    <span> {{ $gettext("测试时磁盘会转动，请选择合适的模式来控制磁盘转动。") }}</span><br />
                    <span>* {{ $gettext("总是-无论是什么功耗模式下都测试(检查)磁盘，当检查时，这可能会使停转的磁盘开始转动。") }}</span><br />
                    <span>* {{ $gettext("睡眠-处于睡眠模式下不检查设备。") }}</span><br />
                    <span>* {{ $gettext("待机-处于待机和睡眠模式下不检查设备。此模式下磁盘一般不旋转，如果你不想每次检查都转动磁盘，那么这个模式比较适合。") }}</span><br />
                    <span>* {{ $gettext("闲置-处于待机、睡眠、闲置模式下不检查设备，在闲置状态下，大多数磁盘还在转动，所以这可能不适合你。") }}</span>
                </div>
            </div>
        </div>
        <div class="cbi-value">
            <label class="cbi-value-title">
                {{ $gettext("温度监测（差异）") }}
            </label>
            <div class="cbi-value-field">
                <div class="cbi-checkbox">
                    <select class="cbi-input-select" v-model.number="config.global.tmpDiff">
                        <option :value="0">{{ $gettext("禁用") }}</option>
                        <option :value="item" v-for="item in 15">{{ item }}°C</option>
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
                    <select class="cbi-input-select" v-model.number="config.global.tmpMax">
                        <option :value="0">{{ $gettext("禁用") }}</option>
                        <option :value="item * 5" v-for="item in 20">{{ item * 5 }}°C</option>
                    </select>
                </div>
                <div class="cbi-value-description">
                    {{ $gettext("如果温度大于或等于 N 摄氏度则报告.") }}
                </div>
            </div>
        </div>
    </fieldset>
    <div class="cbi-section cbi-tblsection" id="cbi-nfs-mount">
        <table class="table cbi-section-table">
            <thead>
                <tr class="tr cbi-section-table-titles anonymous">
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("设备") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("型号") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("序号") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("容量") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("温度") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("状态") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("健康") }}</th>
                    <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("操作") }}</th>
                </tr>
            </thead>
            <tbody>
                <tr class="tr cbi-section-table-row" v-for="(item, i) in disks">
                    <td class="td cbi-value-field">
                        <b>{{ item.path }}</b>
                    </td>
                    <td class="td cbi-value-field">
                        <b>{{ item.model }}</b>
                    </td>

                    <td class="td cbi-value-field">
                        <b>{{ item.serial }}</b>
                    </td>
                    <td class="td cbi-value-field">
                        <b>{{ item.sizeStr }}</b>
                    </td>
                    <td class="td cbi-value-field">
                        <b>{{ item.temp }}</b>
                    </td>
                    <td class="td cbi-value-field">
                        <b>{{ item.status }}</b>
                    </td>
                    <td class="td cbi-value-field">
                        <b>{{ item.health }}</b>
                    </td>
                    <td class="td cbi-value-field">
                        <button class="btn cbi-button cbi-button-apply" :title="$gettext('编辑')"
                            @click="onEditDisk(item, i)">{{ $gettext("编辑") }}</button>
                        <button class="btn cbi-button cbi-button-apply" :title="$gettext('详情')" @click="onDiskInfo(item)">{{ $gettext("详情") }}</button>
                    </td>
                </tr>

            </tbody>
        </table>
    </div>
    <span class="cbi-page-actions control-group">
        <input class="btn cbi-button cbi-button-apply" type="button" :value="$gettext('保存并应用')" @click="onSubimtSave">
    </span>
</template>
<script setup lang="ts">
import { onBeforeUnmount, PropType, reactive, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import { ActionSmartEditDisk, ActionSmartDiskInfo } from '/@/components/smart';
import request from '/@/request';
import utils from "/@/utils";

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
const setConfig = () => {
    config.global.tmpDiff = props.config.global.tmpDiff || 0
    config.global.tmpMax = props.config.global.tmpMax || 0
    config.global.enable = props.config.global.enable || false
    config.global.powermode = props.config.global.powermode || "never"
    config.devices = props.config.devices || []
    config.tasks = props.config.tasks || []
}
const disks = ref<SmartDisks>([])
const getData = async () => {
    try {
        const res = await request.Smart.List.GET()
        if (res.data) {
            const { result, error } = res.data
            if (result && result.disks) {
                disks.value = result.disks || []
            }
        }
    } catch (error) {
    }
}
const cancelGetData = utils.easyInterval(getData, 5000)
onBeforeUnmount(() => {
    cancelGetData()
})
// 保存全局配置
const onSubimtSave = async () => {
    await props.saveData({
        global: config.global,
        devices: props.config.devices,
        tasks: props.config.tasks,
    })
    setConfig()
}
// 查询设备详情
const onDiskInfo = (item: SmartDiskInfo) => {
    ActionSmartDiskInfo({
        disk: item
    })
}
// 编辑设备
const onEditDisk = async (info: SmartDiskInfo, index: number) => {
    let tmpDevice: SmartConfigDevice | null = null
    let tmpIndex: number = -1
    if (config.devices) {
        for (let i = 0; i < config.devices.length; i++) {
            if (config.devices[i].devicePath == info.path) {
                tmpDevice = config.devices[i]
                tmpIndex = i
                break
            }
        }
    }
    ActionSmartEditDisk({
        disk: info,
        device: tmpDevice,
        next: async (device: SmartConfigDevice) => {
            // 使用全局的配置
            if (device.tmpDiff == -1) {
                device.tmpDiff = config.global.tmpDiff
            }
            if (device.tmpMax == -1) {
                device.tmpMax = config.global.tmpMax
            }
            // 使用设备的路径
            if (device.devicePath == "") {
                device.devicePath = info.path
            }
            let tmpDevices = [...config.devices]
            // 如果找到设备
            if (tmpIndex >= 0) {
                tmpDevices[tmpIndex] = device
            }
            const tmpDeviceMap = new Map<string, null>()
            tmpDevices.forEach((item) => {
                if (item.devicePath != null) {
                    tmpDeviceMap.set(item.devicePath, null)
                }
            })
            for (let i = 0; i < disks.value.length; i++) {
                const tmpDisk = disks.value[i]
                // 如果没有该磁盘数据，则添加
                if (tmpDisk.path != null && !tmpDeviceMap.has(tmpDisk.path)) {
                    tmpDevices.push({
                        devicePath: tmpDisk.path,
                        tmpDiff: config.global.tmpDiff,
                        tmpMax: config.global.tmpMax,
                    })
                }
            }
            await props.saveData({
                tasks: props.config.tasks,
                global: props.config.global,
                devices: tmpDevices,
            })
            setConfig()
        }
    })
}

</script>
<style lang="scss" scoped>
</style>