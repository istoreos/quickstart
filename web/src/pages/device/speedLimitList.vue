<template>
    <div class="container">
        <div style="display: flex;justify-content: end;">
            <SearchVue @handleAdd="showModal = true" @refresh="handleRefresh" @batch-delete="handleBatchDelete"
                @search="handleSearch" />
        </div>
        <div>
            <CustomTable :data="tableData1" :columns="columns" rowKey="mac" :showSelection="showSelection"
                :showPagination="false" @selection-change="handleSelectionChange">
                <template #action="{ row }">
                    <button class="del-button del-button--danger" @click="handelDelete(row)">
                        <span>{{ $gettext('删除') }}</span>
                    </button>
                </template>

                <template #enabled="{ row }">
                    <span>
                        {{ row?.enabled ? $gettext('是') : $gettext('否') }}
                    </span>
                </template>

                <template #uploadSpeed="{ row }">
                    <span>
                        {{ row?.uploadSpeed || '-'}} {{ row?.uploadSpeed ? 'Mbit/s' : '' }}
                    </span>
                </template>

                <template #downloadSpeed="{ row }">
                    <span>
                        {{ row?.downloadSpeed || '-' }} {{ row?.downloadSpeed ? 'Mbit/s' : '' }}
                    </span>
                </template>
            </CustomTable>
        </div>

        <!-- 限速配置 -->
        <DialogVue v-model="showModal" title="限速配置" :show-close="true" @confirm="speedLimitConfirm"
            @cancel="handleCancel">
            <!-- 默认插槽内容 -->
            <div class="custom-content">
                <div class="item_box">
                    <div class="item_left">{{ $gettext('对设备开启限速') }}：</div>
                    <SwitchVue v-model="speedLimitData.enabled" :disabled="!globalData?.speedLimit?.enabled"
                        @beforeChange="beforeChange" />
                </div>
                <div class="item_box">
                    <div class="item_left">{{ $gettext('IP地址') }}：</div>
                    <input id="tagName" type="text" v-model.trim="speedLimitData.ip"
                        :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                </div>
                <div class="item_box">
                    <div class="item_left">{{ $gettext('MAC地址') }}：</div>
                    <input id="tagName" type="text" v-model.trim="speedLimitData.mac"
                        :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                </div>
                <div v-if="!globalData?.speedLimit?.enabled" class="tip">
                    <a href="" @click.prevent="getGloba">{{ $gettext('点我跳转全局配置') }}</a>
                </div>
                <div v-if="speedLimitData.enabled">
                    <div class="item_box">
                        <div class="item_left">{{ $gettext('禁止该设备访问网络') }}：</div>
                        <SwitchVue v-model="speedLimitData.networkAccess" @change="" />
                    </div>
                    <template v-if="!speedLimitData.networkAccess">
                        <div class="item_box">
                            <div class="item_left">{{ $gettext('下载限速（Mbit/s）') }}：</div>
                            <input id="tagName" type="text" v-model.trim="speedLimitData.downloadSpeed"
                                :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                            &nbsp; {{ $gettext('总带宽') }}
                        </div>
                        <div class="item_box">
                            <div class="item_left">{{ $gettext('上传限速（Mbit/s）') }}：</div>
                            <input id="tagName" type="text" v-model.trim="speedLimitData.uploadSpeed"
                                :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                            &nbsp; {{ $gettext('总带宽') }}
                        </div>
                        <div class="item_box">
                            <div class="item_left">{{ $gettext('注解') }}：</div>
                            <input id="tagName" type="text" v-model.trim="speedLimitData.comment"
                                :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                        </div>
                    </template>
                </div>
            </div>
        </DialogVue>
    </div>
</template>
<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import request from '/@/request';
import CustomTable from "./components/CustomTable.vue";
import SearchVue from "./components/search.vue";
import SwitchVue from "./components/switch.vue";
import DialogVue from "/@/components/dialog/index.vue";
import Toast from "/@/components/toast";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

const emit = defineEmits(['openGloba'])
const globalData = ref<any>({})
const getGlobalData = async () => {
    try {
        const { data } = await request.DeviceMangement.globalConfigs.GET()
        if (data.result) {
            globalData.value = data.result || {}
        }
    } catch (error) {

    }
}
getGlobalData()

const tableData = ref([])
const tableData1 = ref([])
const showModal = ref(false)

const columns = ref([
    // { label: '设备图片', prop: 'hostImg', width: '120px' },
    { label: '主机名称', prop: 'hostname' },
    { label: 'IP地址', prop: 'ip' },
    { label: 'MAC地址', prop: 'mac' },
    { label: '禁止网络访问', prop: 'enabled', slot: 'enabled' },
    { label: '上传限速（Mbit/s）', prop: 'uploadSpeed' , slot:'uploadSpeed' },
    { label: '下载限速（Mbit/s）', prop: 'downloadSpeed', slot:'downloadSpeed' },
    { label: '注解', prop: 'comment' },
    { label: '操作', prop: 'action', slot: 'action' }
])

const getData = async () => {
    let load = Toast.Loading($gettext("加载中..."))
    try {
        const { data } = await request.DeviceMangement.listSpeedLimitedDevices.GET()
        // console.log(data, '=======');
        if (data.result) {
            tableData.value = data.result || []
            tableData1.value = data.result || []
        }
    } catch (error) {

    } finally {
        load.Close()
    }
}
getData()

// 英文转大写
const upperCaseEnglish = (str: string) => {
    if (/^[a-zA-Z\s]+$/.test(str)) {
        return str.toUpperCase()
    }
    return str
}

const showSelection = ref(true)
const selectedItems = ref([])

const handleSelectionChange = (items: any) => {
    selectedItems.value = items
}

// 刷新
const handleRefresh = async () => {
    tableData.value = []
    let load = Toast.Loading($gettext("加载中..."))
    await getData()
    load.Close()
}

const searchDevices = (keyword: string) => {
    return tableData.value.filter((device: any) => {
        // 检查IP地址是否包含关键词
        const ipMatch = device.ip.includes(keyword);
        // 检查MAC地址是否包含关键词（忽略大小写）
        const macMatch = device.mac.toLowerCase().includes(keyword.toLowerCase());
        return ipMatch || macMatch;
    });
}
// 搜索
const handleSearch = (keyword: string) => {
    if (keyword === '') tableData1.value = tableData.value
    tableData1.value = searchDevices(keyword)
}

// 单项删除
const handelDelete = async (row: any) => {
    if (confirm($gettext("温馨提示：删除设备的限速配置可能影响此设备的带宽，请谨慎操作！"))) {
        let obj = {
            ip: row.ip || '',
            mac: row.mac || '',
            uploadSpeed: row.uploadSpeed || 0,
            downloadSpeed: row.downloadSpeed || 0,
            networkAccess: row.networkAccess || false,
            comment: '',
            action: 'delete'
        }
        deleteDevice(obj, 1)
    }
}
const deleteDevice = async (obj: any, type?: number) => {
    let load = Toast.Loading($gettext("删除中..."))
    try {
        const { data } = await request.DeviceMangement.speedLimitConfig.POST(obj)
        if (type == 1) {
            if (JSON.stringify(data) === '{}') {
                Toast.Success('删除成功' + ' !')
            } else {
                Toast.Success(data?.error || '删除失败！')
            }
            getData()
        }
        return data;
    } catch (error) {

    } finally {
        load.Close()
    }

}
// 批量删除
const handleBatchDelete = async () => {
    if (selectedItems.value.length === 0) {
        return Toast.Warning($gettext("请勾选要删除的数据") + ' !')
    }
    if (confirm($gettext("温馨提示：删除设备的限速配置可能影响此设备的带宽，请谨慎操作！"))) {
        try {
            const deletePromises = selectedItems.value.map((item: any) => {
                const obj = {
                    ip: item.ip || '',
                    mac: item.mac || '',
                    uploadSpeed: item.uploadSpeed || 0,
                    downloadSpeed: item.downloadSpeed || 0,
                    networkAccess: item.networkAccess || false,
                    comment: '',
                    action: 'delete'
                };
                return deleteDevice(obj);
            });
            await Promise.all(deletePromises);
            Toast.Success($gettext('所有删除操作已完成'));
            getData();
        } catch (error) {

        }
    }
}

// 跳转全局配置
const getGloba = () => {
    showModal.value = false
    emit('openGloba')
}
// 设备限速开关按钮改变前触发
const beforeChange = (e: boolean) => {
    if (e == true) {
        if (!globalData.value?.speedLimit?.enabled) {
            return Toast.Warning($gettext('请前往全局配置开启限速'))
        }
    }
}
//限速配置参数
const speedLimitData = reactive({
    ip: '',
    mac: '',
    uploadSpeed: 100 as string | number,
    downloadSpeed: 1000 as string | number,
    networkAccess: false,
    enabled: false,
    comment: '',
    action: 'add'
})
const validatePositiveNumberRegex = (value: string | number) => {
    return /^([1-9]\d*(\.\d+)?|0\.\d*[1-9]\d*)$/.test(value.toString());
}
// 校验ip和mac
const validateNetworkAddress = (type: 'ip' | 'mac', value: string) => {
    if (!value) return false;

    const patterns = {
        ip: /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
        mac: /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$|^([0-9A-Fa-f]{4}\.){2}([0-9A-Fa-f]{4})$/
    };
    return patterns[type].test(value.trim());
}
//限速配置确认按钮
const speedLimitConfirm = async () => {
    if (!globalData.value?.speedLimit?.enabled) {
        return Toast.Warning($gettext('请前往全局配置开启限速'));
    }
    if (!speedLimitData.ip) {
        return Toast.Warning(`${$gettext('请输入')}${$gettext('IP')}`);
    }
    if (!validateNetworkAddress('ip', speedLimitData.ip)) {
        return Toast.Warning(`${$gettext('请输入正确的IP地址')}`);
    }
    if (!speedLimitData.mac) {
        return Toast.Warning(`${$gettext('请输入')}${$gettext('MAC')}`);
    }
    if (!validateNetworkAddress('mac', speedLimitData.mac)) {
        return Toast.Warning(`${$gettext('请输入正确的MAC地址')}`);
    }
    if (!speedLimitData.networkAccess) {
        if (!speedLimitData.downloadSpeed) {
            return Toast.Warning(`${$gettext('请输入')}${$gettext('下载速度')}`);
        }
        if (!validatePositiveNumberRegex(speedLimitData.downloadSpeed)) {
            return Toast.Warning(`${$gettext('请输入正确的下载速度')}`);
        }
        if (!speedLimitData.uploadSpeed) {
            return Toast.Warning(`${$gettext('请输入')}${$gettext('上传速度')}`);
        }
        if (!validatePositiveNumberRegex(speedLimitData.uploadSpeed)) {
            return Toast.Warning(`${$gettext('请输入正确的上传速度')}`);
        }
        speedLimitData.downloadSpeed = Number(speedLimitData.downloadSpeed)
        speedLimitData.uploadSpeed = Number(speedLimitData.uploadSpeed)
    } else {
        speedLimitData.downloadSpeed = 0
        speedLimitData.uploadSpeed = 0
    }
    let load = Toast.Loading($gettext("保存中..."))
    try {
        speedLimitData.networkAccess = !speedLimitData.networkAccess
        const { data } = await request.DeviceMangement.speedLimitConfig.POST(speedLimitData)
        if (JSON.stringify(data) === '{}') {
            handleRefresh()
            Toast.Success("保存成功" + ' !')
        } else {
            Toast.Success(data?.error || '保存失败！')
        }
        showModal.value = false
        handleCancel()
    } catch (error: any) {
        Toast.Warning(`${error?.error} || ${error?.message}`)
    } finally {
        speedLimitData.downloadSpeed = 1000
        speedLimitData.uploadSpeed = 100
        load.Close()
    }
}

// 取消
const handleCancel = () => {
    speedLimitData.ip = ''
    speedLimitData.mac = ''
    speedLimitData.uploadSpeed = 100
    speedLimitData.downloadSpeed = 1000
    speedLimitData.networkAccess = false
    speedLimitData.comment = ''
    speedLimitData.action = 'add'
}


</script>



<style lang="scss" scoped>
.custom-content {
    position: relative;

    .img_box {
        position: absolute;
        right: 0;
        top: 0;
        width: 100px;
        height: 100px;

        >img {
            width: 100%;
            height: 100%;
        }
    }

    .IP_address {
        text-align: center;
        padding: 14px 16px;
        background: rgba(85, 58, 254, 0.1);
        border-radius: 8px;
        margin-bottom: 16px;
    }

    .tip {
        text-align: center;
        margin-top: 16px;
        font-size: 12px;
    }

    .item_box {
        margin-top: 12px;
        display: flex;
        align-items: center;

        >input {
            width: 45%;
            color: var(--card-box-shadow);
            background: transparent !important;
            &::placeholder {
                color: #8898aa;
            }
        }

        >select {
            width: 45%;
            color: var(--card-box-shadow);
            background: transparent !important;
            >option {
                padding: 4px 12px !important;
            }
        }

        .item_left {
            width: 140px;
            text-align: right;
        }
    }
}

.del-button {
    display: inline-flex;
    justify-content: center;
    align-items: center;
    line-height: 1;
    white-space: nowrap;
    cursor: pointer;
    background: #fff;
    border: 1px solid #dcdfe6;
    color: #606266;
    text-align: center;
    box-sizing: border-box;
    outline: none;
    margin: 0;
    transition: all 0.1s;
    font-weight: 500;
    user-select: none;
    padding: 6px 10px;
    font-size: 14px;
    border-radius: 4px;
    margin-right: 8px;
}

/* 删除按钮特定样式 */
.del-button--danger {
    color: #fff;
    background-color: #f56c6c;
    border-color: #f56c6c;
}

/* 悬停效果 */
.del-button--danger:hover {
    background: #f78989;
    border-color: #f78989;
    color: #fff;
}

/* 激活效果 */
.del-button--danger:active {
    background: #dd6161;
    border-color: #dd6161;
    color: #fff;
}

/* 禁用状态 */
.del-button.is-disabled {
    opacity: 0.5;
    cursor: not-allowed;
}
</style>
<style lang="scss" scoped>
/* 移动端样式 - 基于827px设计图 */
@media (max-width: 827px) {}
</style>