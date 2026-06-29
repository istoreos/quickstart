<template>
    <div class="container">
        <div style="display: flex;justify-content: end;">
            <SearchVue @refresh="handleRefresh" :showAdd="false" :showBatchDelete="false" ref="searchRef"
                @batch-delete="handleBatchDelete" @search="handleSearch" />
        </div>
        <div>
            <CustomTable :data="tableData1" :columns="columns" :showPagination="false">
                <template #action="{ row }">
                    <span style="color: #553AFE;cursor: pointer;" @click="openModal(row, 2)">{{
                        $gettext('静态分配') }}</span>
                    <span style="color: #553AFE;cursor: pointer;margin: 0 8px;" @click="openModal(row, 1)">{{
                        $gettext('限速配置')
                        }}</span>
                    <span style="color: #553AFE;margin: 0 8px;cursor: pointer;" @click="openModal(row, 3)">{{
                        $gettext('详情') }}</span>
                </template>

                <template #staticAssigned="{ row }">
                    <span>
                        {{ matchZh(row?.staticAssigned?.tagTitle) || matchZh(row?.staticAssigned?.tagName) || '-' }}
                    </span>
                </template>
            </CustomTable>
        </div>

        <!-- 限速配置 -->
        <DialogVue v-model="showModal1" title="限速配置" :show-close="true" @confirm="speedLimitConfirm"
            @cancel="handleCancel">
            <!-- 默认插槽内容 -->
            <div class="custom-content">
                <div class="IP_address">IP: {{ openData.ip }}</div>
                <div class="item_box">
                    <div class="item_left">{{ $gettext('对设备开启限速') }}：</div>
                    <SwitchVue v-model="speedLimitData.enabled" :disabled="!globalData?.speedLimit?.enabled"
                        @beforeChange="beforeChange" />
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
                            <div class="item_left">{{ $gettext('下载速度') }}（Mbit/s）：</div>
                            <input id="tagName" type="text" v-model.trim="speedLimitData.downloadSpeed"
                                :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                            &nbsp; {{ $gettext('总带宽') }}
                        </div>
                        <div class="item_box">
                            <div class="item_left">{{ $gettext('上传速度') }}（Mbit/s）：</div>
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

        <!-- 静态分配 -->
        <DialogVue v-model="showModal2" title="静态分配" width="550px" :show-close="true" @confirm="staticStateConfirm"
            @cancel="handleCancel">
            <!-- 默认插槽内容 -->
            <div class="custom-content">
                <div class="img_box">
                    <img src="https://fwindex.koolcenter.com/cover/x86_64/cover.png" alt="">
                </div>
                <div class="item_box">
                    <div class="item_left">{{ $gettext('名称') }}：</div>
                    <input id="tagName" type="text" @input="filterChinese" v-model.trim="staticStateData.hostname"
                        :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                </div>
                <div class="item_box">
                    <div class="item_left">MAC：</div>
                    <input id="tagName" type="text" v-model.trim="staticStateData.assignedMac"
                        :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                </div>
                <div class="item_box">
                    <div class="item_left">{{ $gettext('网关') }}：</div>
                    <select v-model="selectedDhcpOption" @change="dhcpGatewayChange">
                        <option v-if="showPlaceholder" :value="null" disabled> {{ $gettext('请选择') }} </option>
                        <option v-for="option in dhcpTags" :value="option">
                            {{ option.gateway }}({{ option.tagTitle ? matchZh(option.tagTitle) : option.tagName ?
                                option.tagName
                                : '-' }})
                        </option>
                    </select>
                </div>
                <div class="item_box">
                    <div class="item_left">{{ $gettext('MAC地址与IP绑定') }}：</div>
                    <SwitchVue v-model="staticStateData.bindIP" />
                </div>
                <div class="item_box" v-if="staticStateData.bindIP">
                    <div class="item_left">IP：</div>
                    <input id="tagName" type="text" v-model.trim="staticStateData.assignedIP"
                        :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                </div>
            </div>
        </DialogVue>

        <!-- 详情 -->
        <DialogVue v-model="showModal3" title="详情" width="550px" :footerShow="false" :show-close="true"
            @cancel="handleCancel">
            <!-- 默认插槽内容 -->
            <div class="custom-content">
                <div class="info-content">
                    <div class="img_box">
                        <img src="https://fwindex.koolcenter.com/cover/x86_64/cover.png" alt="">
                    </div>
                    <div style="margin-bottom: 16px;flex: 1;">
                        <div class="item_box">
                            <div class="item_left">{{ $gettext('名称') }}：</div>
                            {{ openData.hostname || '-' }}
                        </div>
                        <div class="item_box">
                            <div class="item_left"> {{ $gettext('IP地址') }}：</div>
                            {{ openData.ip }}
                        </div>
                        <div class="item_box">
                            <div class="item_left"> MAC：</div>
                            {{ openData.mac }}
                        </div>
                        <div class="item_box">
                            <div class="item_left"> {{ $gettext('网关') }}：</div>
                            {{ openData?.staticAssigned.dhcpGateway || '-' }}
                        </div>
                        <div class="item_box">
                            <div class="item_left"> {{ $gettext('接口') }}：</div>
                            {{ upperCaseEnglish(openData.intr) || '-' }}
                        </div>
                        <div class="item_box">
                            <div class="item_left"> {{ $gettext('标签') }}：</div>
                            {{ openData?.staticAssigned?.tagTitle || openData?.staticAssigned?.tagName || '-' }}
                        </div>
                    </div>
                </div>
                <FlowVue v-if="ipParam" :ipParam="ipParam" />
            </div>
        </DialogVue>
    </div>
</template>
<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request';
import CustomTable from "./components/CustomTable.vue";
import SearchVue from "./components/search.vue";
import SwitchVue from "./components/switch.vue";
import FlowVue from "./components/flow.vue";
import Toast from "/@/components/toast";
import DialogVue from "/@/components/dialog/index.vue";
const { $gettext } = useGettext()
const emit = defineEmits(['openGloba'])

const filterChinese = (e: any) => {
    // 移除中文字符与空白字符，避免名称中出现空格
    staticStateData.hostname = e.target.value.replace(/[\u4e00-\u9fa5\s]/g, '');
};

const timer = ref<NodeJS.Timeout | null>(null)
// 启动定时器
const startPolling = () => {
    stopPolling() // 先停止已有定时器
    speedsForDevices() // 立即执行一次
    timer.value = setInterval(speedsForDevices, 3000)
}

// 停止定时器
const stopPolling = () => {
    if (timer.value) {
        clearInterval(timer.value)
        timer.value = null
    }
}

// 组件挂载时启动定时器
onMounted(async () => {
    await getData()
    if(tableData.value.length === 0) return
    startPolling()
})

// 组件卸载时清除定时器
onUnmounted(() => {
    stopPolling()
})

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
const showModal1 = ref(false)
const showModal2 = ref(false)
const showModal3 = ref(false)

const showPlaceholder = computed(() => !staticStateData.dhcpGateway)

const columns = ref([
    // { label: '设备图片', prop: 'hostImg', width: '120px' },
    { label: '主机名称', prop: 'hostname' },
    { label: 'IP地址', prop: 'ip' },
    { label: 'MAC地址', prop: 'mac' },
    { label: '上传速度', prop: 'uploadSpeedStr' },
    { label: '下载速度', prop: 'downloadSpeedStr' },
    { label: '标签', prop: 'staticAssigned', slot: 'staticAssigned' },
    { label: '操作', prop: 'action', slot: 'action' }
])

type GatewayType = 'default' | 'parent' | 'myself' | 'bypass' | 'floatip'
const matchZh = (str: string): string => {
    const obj: Record<GatewayType, string> = {
        'default': $gettext('默认网关'),
        'parent': $gettext('上级路由'),
        'myself': $gettext('本设备'),
        'bypass': $gettext('旁路由'),
        'floatip': $gettext('浮动网关')
    };

    // 使用类型断言确保访问安全
    return obj[str as GatewayType] || str;
}

// 跳转全局配置
const getGloba = () => {
    showModal1.value = false
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

const dhcpTags = ref<any>([])
const getData = async () => {
    let load = Toast.Loading($gettext("加载中..."))
    try {
        const { data } = await request.DeviceMangement.listDevices.GET()
        // console.log(data, '=======');
        if (data.result) {
            tableData.value = data.result?.devices || []
            tableData1.value = data.result?.devices || []
            dhcpTags.value = data.result?.dhcpTags || []
        }
    } catch (error) {

    } finally {
        load.Close()
    }
}

const mergeSpeedData = (result: any, devices: any) => {
    // 首先创建一个IP到result项的映射，方便快速查找
    const resultMap = <any>{};
    result.forEach((item: any) => {
        if (item.ip) {
            resultMap[item.ip] = {
                downloadSpeedStr: item.downloadSpeedStr || '0 B',
                uploadSpeedStr: item.uploadSpeedStr || '0 B'
            };
        }
    });
    // 遍历devices数组，更新或添加速度信息
    return devices.map((device: any) => {
        if (device.ip && resultMap[device.ip]) {
            return {
                ...device,
                downloadSpeedStr: resultMap[device.ip].downloadSpeedStr,
                uploadSpeedStr: resultMap[device.ip].uploadSpeedStr
            };
        }
        // 如果没有匹配的result数据，确保至少有默认值
        return {
            ...device,
            downloadSpeedStr: device.downloadSpeedStr || '0 B',
            uploadSpeedStr: device.uploadSpeedStr || '0 B'
        };
    });
}
const speedsForDevices = async () => {
    try {
        const { data } = await request.DeviceMangement.speedsForDevices.GET()

        if (data.result) {
            tableData1.value = mergeSpeedData(data.result, tableData1.value)
        }
    } catch (error) {

    }
}

// 英文转大写
const upperCaseEnglish = (str: string) => {
    if (/^[a-zA-Z\s]+$/.test(str)) {
        return str.toUpperCase()
    }
    return str
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
//静态分配参数
const staticStateData = reactive({
    hostname: '',
    assignedIP: '',
    assignedMac: '',
    bindIP: false,
    tagTitle: '',
    tagName: '',
    dhcpGateway: '',
    action: 'add'
})

const ipParam = ref('')
const openData = ref<any>({}) //打开弹窗的那台设备数据
// 打开弹窗
const openModal = (row: any, type: number) => {
    openData.value = row
    if (type === 1) {
        speedLimitData.ip = row.ip || ''
        speedLimitData.mac = row.mac || ''
        speedLimitData.uploadSpeed = row?.speedLimit?.uploadSpeed || 100
        speedLimitData.downloadSpeed = row?.speedLimit?.downloadSpeed || 1000
        speedLimitData.networkAccess = !row?.speedLimit?.networkAccess || false
        speedLimitData.enabled = row?.speedLimit?.enabled || false
        speedLimitData.comment = row?.speedLimit?.comment || ''
        speedLimitData.action = row?.speedLimit?.action || 'add'
        showModal1.value = true
    } else if (type === 2) {
        staticStateData.hostname = row?.staticAssigned?.hostname || ''
        staticStateData.assignedIP = row?.staticAssigned?.assignedIP || ''
        staticStateData.assignedMac = row?.staticAssigned?.assignedMac || ''
        staticStateData.bindIP = row?.staticAssigned?.bindIP || false
        staticStateData.tagTitle = row?.staticAssigned?.tagTitle || ''
        staticStateData.tagName = row?.staticAssigned?.tagName || ''
        if (!row?.staticAssigned?.dhcpGateway) {
            selectedDhcpOption.value = dhcpTags.value[0] || null
            staticStateData.dhcpGateway = selectedDhcpOption.value?.gateway || ''
        } else {
            dhcpTags.value.forEach((item: any) => {
                if (item.gateway === row?.staticAssigned?.dhcpGateway) {
                    selectedDhcpOption.value = item
                }
            })
            staticStateData.dhcpGateway = row?.staticAssigned?.dhcpGateway || dhcpTags.value[0] || ''
        }

        staticStateData.action = row?.staticAssigned?.action || 'add'
        showModal2.value = true
    } else if (type === 3) {
        ipParam.value = ''
        ipParam.value = row.ip
        showModal3.value = true
    }
}

const selectedDhcpOption = ref<any>(dhcpTags.value[0] || null);
// 选择网关时触发
const dhcpGatewayChange = () => {
    if (selectedDhcpOption.value) {
        staticStateData.dhcpGateway = selectedDhcpOption.value?.gateway || ''
        staticStateData.tagName = selectedDhcpOption.value?.tagName || ''
        staticStateData.tagTitle = selectedDhcpOption.value?.tagTitle || ''
    } else {
        // 处理重置情况
        staticStateData.dhcpGateway = ''
        staticStateData.tagName = ''
        staticStateData.tagTitle = ''
    }
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
const validatePositiveNumberRegex = (value: string | number) => {
    return /^([1-9]\d*(\.\d+)?|0\.\d*[1-9]\d*)$/.test(value.toString());
}
//静态分配确认按钮
const staticStateConfirm = async () => {
    if (!staticStateData.hostname) {
        return Toast.Warning(`${$gettext('请输入')}${$gettext('名称')}`);
    }
    if (!staticStateData.assignedMac) {
        return Toast.Warning(`${$gettext('请输入')}${$gettext('MAC')}`);
    }
    if (!validateNetworkAddress('mac', staticStateData.assignedMac)) {
        return Toast.Warning(`${$gettext('请输入正确的MAC地址')}`);
    }
    if (!staticStateData.dhcpGateway) {
        return Toast.Warning(`${$gettext('请选择')}${$gettext('网关')}`);
    }
    if (staticStateData.bindIP) {
        if (!staticStateData.assignedIP) {
            return Toast.Warning(`${$gettext('请输入')}${$gettext('IP')}`);
        }
        if (!validateNetworkAddress('ip', staticStateData.assignedIP)) {
            return Toast.Warning(`${$gettext('请输入正确的IP地址')}`);
        }
    } else {
        staticStateData.assignedIP = ''
    }
    let load = Toast.Loading($gettext("保存中..."))
    try {
        const { data } = await request.DeviceMangement.staticDeviceConfig.POST(staticStateData)
        // console.log(data, '=======');
        if (JSON.stringify(data) === '{}') {
            showModal2.value = false
            handleCancel()
            handleRefresh()
            Toast.Success("保存成功" + ' !')
        } else {
            Toast.Success(data?.error || '保存失败！')
        }
    } catch (error: any) {
        Toast.Warning(`${error?.error} || ${error?.message}`)
    } finally {
        load.Close()
    }
}

//限速配置确认按钮
const speedLimitConfirm = async () => {
    if (!globalData.value?.speedLimit?.enabled) {
        return Toast.Warning($gettext('请前往全局配置开启限速'));
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
            showModal1.value = false
            handleCancel()
            handleRefresh()
            Toast.Success("保存成功" + ' !')
        } else {
            Toast.Success(data?.error || '保存失败！')
        }
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
    openData.value = {}
    selectedDhcpOption.value = dhcpTags.value[0] || null
    staticStateData.hostname = ''
    staticStateData.assignedIP = ''
    staticStateData.assignedMac = ''
    staticStateData.bindIP = false
    staticStateData.dhcpGateway = ''
    staticStateData.tagName = ''
    staticStateData.tagTitle = ''
    staticStateData.action = 'add'

    speedLimitData.ip = ''
    speedLimitData.mac = ''
    speedLimitData.uploadSpeed = 100
    speedLimitData.downloadSpeed = 1000
    speedLimitData.networkAccess = false
    speedLimitData.comment = ''
    speedLimitData.action = 'add'
}

const selectedItems = ref([])

const searchRef = ref<any>(null)
// 刷新
const handleRefresh = async () => {
    tableData.value = []
    await getData()
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

// 批量删除
const handleBatchDelete = () => {
    if (selectedItems.value.length === 0) {
        return Toast.Warning($gettext("请勾选要删除的数据") + ' !')
    }
}

</script>



<style lang="scss" scoped>
:deep(.tag-input) {
    padding: 4px 12px;
}

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
            background: transparent !important;
            color: var(--card-box-shadow);
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

.info-content {
    display: flex;

    .img_box {
        position: relative;
    }

    .item_box {
        .item_left {
            width: 100px;
        }
    }
}
</style>

<style lang="scss" scoped>
/* 移动端样式 - 基于827px设计图 */
@media (max-width: 827px) {}
</style>
