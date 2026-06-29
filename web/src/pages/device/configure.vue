<template>
    <div class="tab-container">
        <div class="tab-header">
            <button v-for="tab in tabs" :key="tab.id" class="tab-button" :class="{ active: activeTab === tab.id }"
                @click="handelTab(tab.id)">
                {{ tab.label }}
            </button>
        </div>
        <div class="tab-content_g">

            <!-- IP限速 -->
            <div v-show="activeTab === 'ip'">
                <div class="not_installed" v-if="!eqosShow">
                    <svg t="1752659436579" class="icon" viewBox="0 0 1024 1024" version="1.1"
                        xmlns="http://www.w3.org/2000/svg" p-id="4943" width="150" height="150">
                        <path d="M216.896 97.232l-55.792 106.24 704.784 8.928-24.352-75.888-38.496-39.28z"
                            fill="#FFFFFF" p-id="4944"></path>
                        <path d="M192.016 255.968h655.968v592H192.016z" fill="#FFFFFF" p-id="4945"></path>
                        <path
                            d="M921.904 187.008l-66.72-80.656a69.744 69.744 0 0 0-55.168-26.32h-576a71.296 71.296 0 0 0-55.664 26.416l-66.256 80.56a93.984 93.984 0 0 0-22.08 61.024v600a96.288 96.288 0 0 0 96 96h672a96.288 96.288 0 0 0 96-96v-600a93.984 93.984 0 0 0-22.112-61.024zM512.016 777.856L246.128 512.032h166.144v-132.976h199.392v132.976h166.128zM179.664 179.664l33.152-66.464h598.128l33.2 66.464z"
                            fill="#909399" p-id="4946"></path>
                    </svg>
                    <span>{{ $gettext('软件暂未安装') }}</span>
                    <div class="not_installed_btn" @click="openMode('app-meta-eqos')">{{ $gettext("立即安装") }}</div>
                </div>
                <div v-else>
                    <div class="item_box">
                        <div class="item_left">{{ $gettext('IP限速') }}：</div>
                        <SwitchVue v-model="speedLimitData.enabled" @change="speedLimitChange" />
                    </div>
                    <template v-if="speedLimitData.enabled">
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
                    </template>
                    <div class="item_box">
                        <div class="item_left">
                            <button class="add-button add-button--danger" @click="ipSave">{{ $gettext('保存') }}</button>
                        </div>
                    </div>
                    <!-- <div style="display: flex;justify-content: center;margin-top: 16px;">
                        <button class="add-button" @click="">{{ $gettext('取消') }}</button>
                        <button class="add-button add-button--danger" @click="ipSave">{{ $gettext('保存') }}</button>
                    </div> -->
                </div>
            </div>

            <!-- 浮动网关 -->
            <div v-show="activeTab === 'gateway'">
                <div class="not_installed" v-if="!floatipShow">
                    <svg t="1752659436579" class="icon" viewBox="0 0 1024 1024" version="1.1"
                        xmlns="http://www.w3.org/2000/svg" p-id="4943" width="150" height="150">
                        <path d="M216.896 97.232l-55.792 106.24 704.784 8.928-24.352-75.888-38.496-39.28z"
                            fill="#FFFFFF" p-id="4944"></path>
                        <path d="M192.016 255.968h655.968v592H192.016z" fill="#FFFFFF" p-id="4945"></path>
                        <path
                            d="M921.904 187.008l-66.72-80.656a69.744 69.744 0 0 0-55.168-26.32h-576a71.296 71.296 0 0 0-55.664 26.416l-66.256 80.56a93.984 93.984 0 0 0-22.08 61.024v600a96.288 96.288 0 0 0 96 96h672a96.288 96.288 0 0 0 96-96v-600a93.984 93.984 0 0 0-22.112-61.024zM512.016 777.856L246.128 512.032h166.144v-132.976h199.392v132.976h166.128zM179.664 179.664l33.152-66.464h598.128l33.2 66.464z"
                            fill="#909399" p-id="4946"></path>
                    </svg>
                    <span>{{ $gettext('软件暂未安装') }}</span>
                    <div class="not_installed_btn" @click="openMode('app-meta-floatip')">{{ $gettext("立即安装") }}</div>
                </div>
                <div v-else>
                    <div class="item_box">
                        <div class="item_left">{{ $gettext('浮动网关') }}：</div>
                        <SwitchVue v-model="floatGatewayData.enabled" @change="floatGatewayChange" />
                    </div>
                    <div class="item_box">
                        <div class="item_left">{{ $gettext('节点角色') }}：</div>
                        <select v-model="floatGatewayData.role" @change="">
                            <option v-if="showPlaceholder" value="" disabled> {{ $gettext('请选择') }} </option>
                            <option v-for="option in nodeRole" :value="option.value">
                                {{ option.name }}
                            </option>
                        </select>
                    </div>
                    <div class="item_box">
                        <div class="item_left">{{ $gettext('浮动网关') }}IP：</div>
                        <input id="tagName" type="text" v-model.trim="floatGatewayData.setIP"
                            :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                    </div>
                    <div class="item_box">
                        <div class="item_left">{{ $gettext('旁路由IP') }}：</div>
                        <input id="tagName" type="text" v-model.trim="floatGatewayData.checkIP"
                            :placeholder="$gettext('请输入') + '...'" class="tag-input" />
                    </div>
                    <div class="item_box">
                        <div class="item_left">
                            <button class="add-button add-button--danger" @click="floatGatewaySave">{{ $gettext('保存')
                                }}</button>
                        </div>
                    </div>
                    <!-- <div style="display: flex;justify-content: center;margin-top: 16px;">
                        <button class="add-button" @click="">{{ $gettext('取消') }}</button>
                        <button class="add-button add-button--danger" style="width: 100px;" @click="">{{ $gettext('保存') }}</button>
                    </div> -->
                </div>
            </div>

            <!-- 局域网DHCP -->
            <div v-show="activeTab === 'tag'">
                <div style="margin-bottom: 16px;">
                    <div class="item_box">
                        <div class="item_left">{{ $gettext('启用') }}DHCP：</div>
                        <SwitchVue v-model="DHCPData.dhcpEnabled" @change="dhcpChange" />
                    </div>
                    <div class="item_box">
                        <div class="item_left">DHCP{{ $gettext('网关') }}：</div>
                        <select v-model="DHCPData.dhcpGateway" @change="">
                            <!-- <option v-if="showPlaceholder1" value="" disabled> {{ $gettext('请选择') }} </option> -->
                            <option v-for="option in globalData?.dhcpGlobal?.gatewaySels" :value="option.gateway">
                                {{ option.gateway }} ({{ option.title ? matchZh(option.title) : '' }})
                            </option>
                        </select>
                    </div>
                    <div class="item_box">
                        <div class="item_left">
                            <button class="add-button add-button--danger" @click="DHCPSave">{{
                                $gettext('保存')
                                }}</button>
                        </div>
                    </div>
                </div>
                <div style="display: flex;justify-content: end;margin-bottom: 8px;">
                    <button class="add-button add-button--danger" @click="addLabel">
                        <span>{{ $gettext('添加') }}</span>
                    </button>
                </div>
                <CustomTable :data="tableData" :columns="columns" :showSelection="false" :showPagination="false"
                    theadBgColor="#e8e6f9">
                    <template #action="{ row }">
                        <span style="color: #553AFE;cursor: pointer;" @click="openEdit(row)" v-if="!row.autoCreated">{{
                            $gettext('编辑')
                        }}</span>
                        <span style="color: #F04134;cursor: pointer;margin-left: 18px;" @click="handleDelete(row)"
                            v-if="!row.autoCreated">{{
                                $gettext('删除')
                            }}</span>
                    </template>
                    <template #tagTitle="{ row }">
                        <span>{{ matchZh(row.tagTitle) }} </span>
                    </template>
                    <template #tagName="{ row }">
                        <span>{{ row.tagName || '-' }} </span>
                    </template>
                    <template #gateway="{ row }">
                        <span>{{ row.gateway || '-' }}</span>
                    </template>
                </CustomTable>
            </div>
        </div>
        <DialogVue ref="tagDialogRef" :title="dialogTitle" @confirm="handleTagConfirm" />
    </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, nextTick } from 'vue';
import DialogVue from "./components/dialog.vue";
import appUtils from "/@/utils/app";
import Toast from "/@/components/toast";
import SwitchVue from "./components/switch.vue";
import request from '/@/request';
import CustomTable from "./components/CustomTable.vue";
import { useGettext } from '/@/plugins/i18n'
import { on } from 'events';
const { $gettext } = useGettext()

const showPlaceholder = computed(() => !floatGatewayData.role)
// const showPlaceholder1 = computed(() => !DHCPData.dhcpGateway)
const tableData = ref([])
const columns = ref([
    { label: '标题', prop: 'tagTitle', slot: 'tagTitle' },
    { label: 'ID', prop: 'tagName', slot: 'tagName' },
    { label: '网关', prop: 'gateway', slot: 'gateway' },
    { label: '操作', prop: 'action', slot: 'action' }
])

const nodeRole = ref([
    { name: $gettext('主路由'), value: 'fallback' },
    { name: $gettext('旁路由'), value: 'main' }
])
// DHCP数据
const DHCPData = reactive({
    dhcpEnabled: false,
    dhcpGateway: ''
})
// IP限速数据
const speedLimitData = reactive({
    enabled: false,
    uploadSpeed: '' as string | number,
    downloadSpeed: '' as string | number,
    installed: true,
    // action: 'post'
})
// 浮动网格数据
const floatGatewayData = reactive({
    enabled: false,
    role: '',
    setIP: '',
    checkIP: '',
    // action: 'post'
})

const globalData = ref<any>({})
const getGlobalData = async () => {
    try {
        const { data } = await request.DeviceMangement.globalConfigs.GET()
        if (data.result) {
            globalData.value = data.result || {}
            DHCPData.dhcpEnabled = data.result?.dhcpGlobal?.dhcpEnabled || false
            if (data.result?.dhcpGlobal?.dhcpGateway) {
                DHCPData.dhcpGateway = data.result?.dhcpGlobal?.dhcpGateway
            } else {
                const myselfData = data.result?.dhcpGlobal?.gatewaySels.find((item: any) => item.title === "myself");
                if (myselfData) {
                    DHCPData.dhcpGateway = myselfData.gateway
                } else {
                    DHCPData.dhcpGateway = ''
                }
            }
            tableData.value = data.result?.dhcpTags || []
            speedLimitData.enabled = data.result?.speedLimit?.enabled || false
            speedLimitData.uploadSpeed = data.result?.speedLimit?.uploadSpeed || ''
            speedLimitData.downloadSpeed = data.result?.speedLimit?.downloadSpeed || ''
            floatGatewayData.enabled = data.result?.floatGateway?.enabled || false
            floatGatewayData.role = data.result?.floatGateway?.role || ''
            floatGatewayData.setIP = data.result?.floatGateway?.setIP || ''
            floatGatewayData.checkIP = data.result?.floatGateway?.checkIP || ''
        }
    } catch (error) {

    }
}
getGlobalData()

// DHCP配置保存
const DHCPSave = async () => {
    // if (!DHCPData.dhcpGateway) {
    //     return Toast.Warning(`${$gettext('请选择')}DHCP${$gettext('网关')}`)
    // }
    let load = Toast.Loading($gettext("保存中..."))
    try {
        const { data } = await request.DeviceMangement.dhcpGatewayConfig.POST(DHCPData)
        if (JSON.stringify(data) === '{}') {
            Toast.Success($gettext("保存成功"));
            getGlobalData()
        } else {
            Toast.Success(data?.error || '保存失败！')
        }

    } catch (error: any) {
        Toast.Warning(`${error?.error} || ${error?.message}`)
    } finally {
        load.Close()
    }
}

const checkAndInstallApp = async (pkg: string, app: string) => {
    let load = Toast.Loading($gettext("检查中..."))
    try {
        const res = await request.App.Check.POST({
            name: pkg
        })
        load.Close()
        if (res?.data) {
            const { result, error } = res.data
            if (error) {
                Toast.Warning(error)
            } else if (result) {
                if (result.status == "installed") {
                    return true
                } else {
                    return false
                }
            } else {
                Toast.Warning($gettext("检查插件状态失败"))
            }
        }
        return false
    } catch (error) {
        load.Close()
        Toast.Warning(error as string)
        return false
    }
}
const eqosShow = ref(false)
const floatipShow = ref(false)
//检测是否安装了浮动网关
const checkIsInstallFloatip = async () => {
    if (await checkAndInstallApp("app-meta-floatip", "Floatip")) {
        floatipShow.value = true
    }
}
//检测是否安装了IP限速
const checkIsInstallEqos = async () => {
    if (await checkAndInstallApp("app-meta-eqos", "Eqos")) {
        eqosShow.value = true
    }
}
checkIsInstallFloatip()
checkIsInstallEqos()

type Tab = {
    id: string;
    label: string;
};

const tabs: Tab[] = [
    { id: 'tag', label: $gettext('DHCP') },
    { id: 'gateway', label: $gettext('浮动网关') },
    { id: 'ip', label: $gettext('IP限速') },
];

const activeTab = ref<string>('tag');
defineExpose({
    activeTab
})
const handelTab = (id: string) => {
    activeTab.value = id
}

const tagDialogRef = ref()

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

const pkgStr = ref('app-meta-floatip')
const dialogTitle = computed(() => {
  if (pkgStr.value === 'app-meta-floatip') {
    return $gettext("浮动网关")
  } else {
    return $gettext("IP限速")
  }
})
// 立即安装IP限速
const openMode = async (pkg: string) => {
    pkgStr.value = pkg
    // if (pkg == 'app-meta-floatip') {
    //     tagDialogRef.value.title = $gettext("浮动网关")
    // } else {
    //     tagDialogRef.value.title = $gettext("IP限速")
    // }
    tagDialogRef.value.openInstallDialog()
    const is = await appUtils.installApp(pkg)
    if (is) {
        if (pkg == 'app-meta-floatip') {
            checkIsInstallFloatip()
        } else {
            checkIsInstallEqos()
        }
        tagDialogRef.value.showInstallResult()
        return true
    } else {
        tagDialogRef.value.cancelInstall()
        Toast.Error($gettext("安装失败或超时，请检查软件源或稍候重试"))
    }
}

const validatePositiveNumberRegex = (value: string | number) => {
    return /^([1-9]\d*(\.\d+)?|0\.\d*[1-9]\d*)$/.test(value.toString());
}
// ip限速配置保存
const ipSave = async () => {
    if (speedLimitData.enabled) {
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
        const { data } = await request.DeviceMangement.enableSpeedLimit.POST(speedLimitData)
        if (JSON.stringify(data) === '{}') {
            Toast.Success($gettext("保存成功"));
            getGlobalData()
        } else {
            Toast.Success(data?.error || '保存失败！')
        }

    } catch (error: any) {
        Toast.Warning(`${error?.error} || ${error?.message}`)
    } finally {
        load.Close()
    }
}

// 校验ip和mac
const validateNetworkAddress = (type: 'ip' | 'mac', value: string) => {
    if (!value) return false;
    const patterns = {
        ip: /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(?:\/([0-9]|[1-2][0-9]|3[0-2]))?$/,
        mac: /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$|^([0-9A-Fa-f]{4}\.){2}([0-9A-Fa-f]{4})$/
    };

    return patterns[type].test(value.trim());
}
// 保存浮动网关配置
const floatGatewaySave = async () => {
    if (!floatGatewayData.role) {
        return Toast.Warning($gettext('请选择节点角色'))
    }
    if (!floatGatewayData.setIP) {
        return Toast.Warning(`${$gettext('请输入')}${$gettext('浮动网关')}IP`)
    }
    if (!validateNetworkAddress('ip', floatGatewayData.setIP)) {
        return Toast.Warning(`${$gettext('请输入正确的浮动网关IP地址')}`);
    }
    if (!floatGatewayData.checkIP) {
        return Toast.Warning(`${$gettext('请输入')}${$gettext('旁路由')}IP`)
    }
    if (!validateNetworkAddress('ip', floatGatewayData.checkIP)) {
        return Toast.Warning(`${$gettext('请输入正确的旁路由IP地址')}`);
    }
    let load = Toast.Loading($gettext("保存中..."))
    try {
        const { data } = await request.DeviceMangement.enableFloatGateway.POST(floatGatewayData)
        if (JSON.stringify(data) === '{}') {
            Toast.Success($gettext("保存成功"));
            getGlobalData()
        } else {
            Toast.Success(data?.error || '保存失败！')
        }
    } catch (error: any) {
        Toast.Warning(`${error?.error} || ${error?.message}`)
    } finally {
        load.Close()
    }
}

const dhcpChange = (val: boolean) => {
    if (!val) {
        if (!confirm($gettext("温馨提示：关闭DHCP可能影响局域网内设备的IP分配和联网，请谨慎操作！"))) {
            DHCPData.dhcpEnabled = true
        }
    }
}
// 浮动网关关闭提示
const floatGatewayChange = (val: boolean) => {
    if(!val){
        if(!confirm($gettext("温馨提示：关闭浮动网关可能影响正在使用浮动网关的设备，请谨慎操作！"))){
            floatGatewayData.enabled = true
        }
    }
}
// ip限速关闭提示
const speedLimitChange = (val: boolean) => {
    if(!val){
        if(!confirm($gettext("温馨提示：关闭限速会让已配置限速的设备的带宽限制全部失效，请谨慎操作！"))){
            speedLimitData.enabled = true
        }
    }
}

// 删除
const handleDelete = async (row: any) => {
    if (confirm($gettext("温馨提示：删除网关标签可能影响正在使用此标签的设备，请谨慎操作！"))) {
        let load = Toast.Loading($gettext("删除中..."))
        let obj = {
            action: 'delete',
            tagTitle: row.tagTitle || '',
            tagName: row.tagName || '',
            dhcpOption: row?.dhcpOption || []
        }
        try {
            const { data } = await request.DeviceMangement.dhcpTagsConfig.POST(obj)
            if (JSON.stringify(data) === '{}') {
                Toast.Success($gettext("删除成功"));
                getGlobalData()
            } else {
                Toast.Success(data?.error || '删除失败！')
            }
        } catch (error: any) {
            Toast.Warning(`${error?.error} || ${error?.message}`)
        } finally {
            load.Close()
        }

    }
}

const addLabel = () => {
    tagDialogRef.value.openTagDialog()
}
const dhcpOption = ref([])
const openEdit = async (row: any) => {
    dhcpOption.value = row.dhcpOption ? row.dhcpOption : []
    tagDialogRef.value.tagTitle = row.tagTitle || ''
    tagDialogRef.value.tagName = row.tagName || ''
    tagDialogRef.value.gateway = row.gateway || ''
    await nextTick()
    tagDialogRef.value.openEditTagDialog()
}

// 处理标签确认
const handleTagConfirm = async (tagData: any) => {
    let load = Toast.Loading($gettext("保存中..."))
    const arr = [`3,${tagData.gateway}`, `6,${tagData.gateway}`]
    let obj = {
        action: tagData.type == 1 ? 'add' : 'modify',
        tagTitle: tagData.tagTitle,
        tagName: tagData.tagName,
        dhcpOption: arr
    }
    try {
        const { data } = await request.DeviceMangement.dhcpTagsConfig.POST(obj)
        if (JSON.stringify(data) === '{}') {
            Toast.Success($gettext("保存成功"));
            getGlobalData()
        } else {
            Toast.Success(data?.error || '保存失败！')
        }
    } catch (error: any) {
        Toast.Warning(`${error?.error} || ${error?.message}`)
    } finally {
        load.Close()
    }
}
</script>

<style lang="scss" scoped>
.add-button {
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
    padding: 8px 12px;
    font-size: 14px;
    border-radius: 4px;
    margin-right: 8px;
}

.add-button--danger {
    color: #fff;
    background-color: #553afe;
    border-color: #553afe;
}

/* 悬停效果 */
.add-button--danger:hover {
    background: #5c44f8;
    border-color: #5c44f8;
    color: #fff;
}

/* 激活效果 */
.add-button--danger:active {
    background: #553AFE;
    border-color: #553AFE;
    color: #fff;
}

/* 禁用状态 */
.add-button.is-disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.tab-container {
    display: flex;
    flex-direction: row;
    width: 100%;
    margin: 0 auto;
    border-radius: 8px;
    overflow: hidden;

    @media (max-width: 768px) {
        flex-direction: column;
    }
}

.tab-header {
    display: flex;
    flex-direction: column;
    width: 120px;

    @media (max-width: 768px) {
        flex-direction: row;
        width: 100%;
        overflow-x: auto;
        white-space: nowrap;
    }
}

.tab-button {
    padding: 12px 16px;
    text-align: left;
    border: none;
    background: transparent !important;
    cursor: pointer;
    font-size: 14px;
    color: var(--flow-span-color);
    transition: all 0.3s ease;
    border-radius: 8px 0 0 8px;

    &:hover {
        background: var(--tag-bg-color) !important;
    }

    &.active {
        background: var(--tag-bg-color) !important;
        font-weight: 500;
        margin: 0;
    }

    @media (max-width: 768px) {
        border-radius: 8px 8px 0 0;
        text-align: center;
        flex: 1;
        min-width: max-content;

        &.active {
            border-radius: 8px 8px 0 0;
        }
    }
}

.tab-content_g {
    flex: 1;
    padding: 20px;
    background: var(--tag-bg-color);
    border-radius: 0 8px 8px 0;
    min-height: 60vh;

    .not_installed {
        display: flex;
        flex-direction: column;
        align-items: center;

        >span {
            color: var(--tit-color);
            margin: 20px 0;
        }

        .not_installed_btn {
            padding: 6px 16px;
            background: #553AFE;
            border-radius: 4px;
            font-size: 14px;
            color: #FFFFFF;
            cursor: pointer;
        }
    }

    @media (max-width: 768px) {
        border-radius: 0 0 8px 8px;
    }
}

.item_box {
    margin-top: 12px;
    display: flex;
    align-items: center;
    color: var(--tit-color) !important;

    >input {
        width: 40%;
        color: var(--card-box-shadow);
        background: transparent !important;

        @media (max-width: 768px) {
            width: 70%;
        }

        color: var(--tit-color) !important;

        &::placeholder {
            color: var(--item-label_key-span-color);
        }
    }

    >select {
        background: transparent !important;
        width: 40%;
        color: var(--card-box-shadow);

        @media (max-width: 768px) {
            width: 70%;
        }

        >option {
            padding: 4px 12px !important;
        }
    }

    .item_left {
        width: 140px;

        @media (max-width: 768px) {
            width: 100px;
        }

        text-align: right;
    }
}
</style>