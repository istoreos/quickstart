<template>
    <div class="container">
        <div style="display: flex;justify-content: end;">
            <SearchVue @handleAdd="showModal = true" @refresh="handleRefresh" @batch-delete="handleBatchDelete"
                @search="handleSearch" />
        </div>
        <div>
                <CustomTable :data="tableData1" :columns="columns" rowKey="assignedMac" :showSelection="showSelection"
                :showPagination="false" @selection-change="handleSelectionChange">
                <!-- @vue-ignore -->
                <template #action="{ row }">
                    <button class="del-button edit-button" @click="handleEdit(row)">
                        <span>{{ $gettext('编辑') }}</span>
                    </button>
                    <button class="del-button del-button--danger" @click="handelDelete(row)">
                        <span>{{ $gettext('删除') }}</span>
                    </button>
                </template>

                <!-- @vue-ignore -->
                <template #tagTitle="{ row }">
                    <span>
                        {{ matchZh(row?.tagTitle) || matchZh(row?.tagName) || '-' }}
                    </span>
                </template>

                <!-- @vue-ignore -->
                <template #bindIP="{ row }">
                    <span>
                        {{ row?.bindIP ? $gettext('是') : $gettext('否') }}
                    </span>
                </template>
            </CustomTable>
        </div>

        <!-- 静态分配 -->
        <DialogVue v-model="showModal" title="静态分配" width="550px" :show-close="true" @confirm="staticStateConfirm"
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
    </div>
</template>
<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import request from '/@/request';
import CustomTable from "./components/CustomTable.vue";
import SearchVue from "./components/search.vue";
import SwitchVue from "./components/switch.vue";
import Toast from "/@/components/toast";
import DialogVue from "/@/components/dialog/index.vue";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()
const dhcpTags = ref<any>([])
const getGlobalData = async () => {
    try {
        const { data } = await request.DeviceMangement.globalConfigs.GET()
        if (data.result) {
            dhcpTags.value = data.result?.dhcpTags || []
        }
    } catch (error) {

    }
}
getGlobalData()

const tableData = ref<any[]>([])
const tableData1 = ref<any[]>([])

const showModal = ref(false)

const columns = ref([
    // { label: '设备图片', prop: 'hostImg', width: '120px' },
    { label: '主机名称', prop: 'hostname' },
    { label: 'IP地址', prop: 'assignedIP' },
    { label: 'MAC地址', prop: 'assignedMac' },
    { label: '静态IP绑定', prop: 'bindIP', slot: 'bindIP' },
    { label: '标签', prop: 'tagTitle', slot: 'tagTitle' },
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

const filterChinese = (e: any) => {
    // 移除中文字符与空白字符，避免名称中出现空格
    staticStateData.hostname = e.target.value.replace(/[\u4e00-\u9fa5\s]/g, '');
};

const getData = async () => {
    let load = Toast.Loading($gettext("加载中..."))
    try {
        const { data } = await request.DeviceMangement.listStaticDevices.GET()
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
        const ipMatch = device.assignedIP.includes(keyword);
        // 检查MAC地址是否包含关键词（忽略大小写）
        const macMatch = device.assignedMac.toLowerCase().includes(keyword.toLowerCase());
        return ipMatch || macMatch;
    });
}
// 搜索
const handleSearch = (keyword: string) => {
    if (keyword === '') tableData1.value = tableData.value
    tableData1.value = searchDevices(keyword)
}

// 编辑静态分配
const handleEdit = (row: any) => {
    staticStateData.hostname = row?.hostname || ''
    staticStateData.assignedIP = row?.assignedIP || ''
    staticStateData.assignedMac = row?.assignedMac || ''
    staticStateData.bindIP = row?.bindIP || false
    staticStateData.tagTitle = row?.tagTitle || ''
    staticStateData.tagName = row?.tagName || ''

    if (!row?.dhcpGateway) {
        selectedDhcpOption.value = dhcpTags.value[0] || null
        staticStateData.dhcpGateway = selectedDhcpOption.value?.gateway || ''
    } else {
        dhcpTags.value.forEach((item: any) => {
            if (item.gateway === row?.dhcpGateway) {
                selectedDhcpOption.value = item
            }
        })
        staticStateData.dhcpGateway = row?.dhcpGateway || dhcpTags.value[0] || ''
    }

    // 和设备列表里的静态分配保持一致：已有记录没有 action 字段时，按 add 处理（后端做覆盖/更新）
    staticStateData.action = row?.action || 'add'
    showModal.value = true
}

// 单项删除
const handelDelete = async (row: any) => {
    if (confirm($gettext("温馨提示：删除设备的静态分配可能影响此设备的联网，请谨慎操作！"))) {
        let obj = {
            hostname: row.hostname || '',
            assignedIP: row.assignedIP || '',
            assignedMac: row.assignedMac || '',
            tagTitle: row.tagTitle || '',
            bindIP: row.bindIP || false,
            tagName: row.tagName || '',
            dhcpGateway: row.dhcpGateway || '',
            action: 'delete'
        }
        deleteDevice(obj, 1)
    }
}
const deleteDevice = async (obj: any, type?: number) => {
    let load = Toast.Loading($gettext("删除中..."))
    try {
        const { data } = await request.DeviceMangement.staticDeviceConfig.POST(obj)
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
    if (confirm($gettext("温馨提示：删除设备的静态分配可能影响此设备的联网，请谨慎操作！"))) {
        try {
            const deletePromises = selectedItems.value.map((item: any) => {
                const obj = {
                    hostname: item.hostname || '',
                    assignedIP: item.assignedIP || '',
                    assignedMac: item.assignedMac || '',
                    tagTitle: item.tagTitle || '',
                    bindIP: item.bindIP || false,
                    tagName: item.tagName || '',
                    dhcpGateway: item.dhcpGateway || '',
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

const showPlaceholder = computed(() => !staticStateData.dhcpGateway)
const selectedDhcpOption = ref<any>(null);
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

// 校验ip和mac
const validateNetworkAddress = (type: 'ip' | 'mac', value: string) => {
    if (!value) return false;

    const patterns = {
        ip: /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
        mac: /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$|^([0-9A-Fa-f]{4}\.){2}([0-9A-Fa-f]{4})$/
    };
    return patterns[type].test(value.trim());
}

//静态分配确认按钮
const staticStateConfirm = async () => {
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
        if (JSON.stringify(data) === '{}') {
            showModal.value = false
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

// 取消
const handleCancel = () => {
    selectedDhcpOption.value = null
    staticStateData.hostname = ''
    staticStateData.assignedIP = ''
    staticStateData.assignedMac = ''
    staticStateData.bindIP = false
    staticStateData.dhcpGateway = ''
    staticStateData.tagName = ''
    staticStateData.tagTitle = ''
    staticStateData.action = 'add'
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

.edit-button {
    color: #553AFE;
    border-color: #553AFE;
}

.edit-button:hover {
    background: rgba(85, 58, 254, 0.1);
    border-color: #553AFE;
}

.edit-button:active {
    background: rgba(85, 58, 254, 0.2);
    border-color: #553AFE;
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
