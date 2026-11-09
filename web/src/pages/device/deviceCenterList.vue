<template>
    <section class="device-center" :aria-label="$gettext('设备')">
        <div class="device-toolbar">
            <nav class="device-filters" :aria-label="$gettext('设备筛选')">
                <button v-for="item in filters" :key="item.value" type="button"
                    :class="{ active: filter === item.value }" @click="filter = item.value">
                    {{ item.label }} <span>{{ counts[item.value] }}</span>
                </button>
            </nav>
            <div class="device-center__tools">
                <label class="device-search">
                    <span class="sr-only">{{ $gettext('搜索设备') }}</span>
                    <input v-model.trim="query" type="search" :placeholder="$gettext('搜索名称、IP、MAC 或厂商')" />
                </label>
                <select v-model="sort" :aria-label="$gettext('排序方式')">
                    <option value="recent">{{ $gettext('最近活跃') }}</option>
                    <option value="name">{{ $gettext('按名称') }}</option>
                </select>
                <button class="icon-button" type="button" :disabled="loading" @click="load" :aria-label="$gettext('刷新')">↻</button>
                <button v-if="!aggregate" class="add-device" type="button" @click="adding = !adding">{{ adding ? $gettext('取消添加') : $gettext('添加设备') }}</button>
            </div>
        <details class="filter-panel">
            <summary>{{ $gettext('更多筛选') }}<small v-if="activeAdvancedFilterCount">{{ activeAdvancedFilterCount }}</small></summary>
            <div class="filter-panel__controls">
                <label><span>{{ $gettext('连接方式') }}</span><select v-model="criteria.connection"><option value="all">{{ $gettext('全部连接') }}</option><option value="lan">{{ $gettext('有线') }}</option><option value="wifi">Wi-Fi</option><option value="unknown">{{ $gettext('未知') }}</option></select></label>
                <label><span>{{ $gettext('品牌') }}</span><select v-model="criteria.brand"><option value="">{{ $gettext('全部品牌') }}</option><option v-for="brand in availableBrands" :key="brand" :value="brand">{{ brand }}</option></select></label>
                <label><span>{{ $gettext('策略状态') }}</span><select v-model="criteria.policy"><option value="all">{{ $gettext('全部策略') }}</option><option value="none">{{ $gettext('无策略') }}</option><option value="static">{{ $gettext('静态地址') }}</option><option value="route">{{ $gettext('已指定路线') }}</option><option value="limited">{{ $gettext('已限速') }}</option><option value="blocked">{{ $gettext('已断网') }}</option></select></label>
                <button v-if="activeAdvancedFilterCount" type="button" class="clear-filters" @click="clearAdvancedFilters">{{ $gettext('清除筛选') }}</button>
            </div>
        </details>
        </div>

        <form v-if="adding" class="add-device-form" @submit.prevent="addManualDevice"><div><strong>{{ $gettext('添加尚未上线的设备') }}</strong><small>{{ $gettext('保存 MAC 和备注，设备首次出现后会自动合并，不会创建孤立规则。') }}</small></div><label><span>{{ $gettext('设备备注') }}</span><input v-model.trim="manual.alias" maxlength="64" :placeholder="$gettext('例如：孩子的电脑')" /></label><label><span>MAC</span><input v-model.trim="manual.mac" required autocomplete="off" placeholder="AA:BB:CC:DD:EE:FF" /></label><button type="submit" :disabled="addingDevice">{{ addingDevice ? $gettext('正在添加…') : $gettext('确认添加') }}</button><p v-if="addError" role="alert">{{ addError }}</p></form>

        <div v-if="telemetryHealth === 'partial' || telemetryHealth === 'stale'" class="telemetry-notice" role="status">
            {{ telemetryHealth === 'stale' ? $gettext('流量数据更新延迟，正在恢复采样') : $gettext('流量采样暂不可用，设备清单仍可正常使用') }}
        </div>

        <PageState v-if="loading" kind="loading" :title="$gettext('正在加载设备')" :description="$gettext('正在整理设备身份和在线状态…')" />
        <LoadError v-else-if="loadError" :message="loadError" @retry="load" />
        <PageState v-else-if="health.state === 'partial'" kind="info" :title="$gettext('部分设备信息暂不可用')"
            :description="$gettext('设备清单仍可使用，缺失的信息会在数据源恢复后自动补齐。')" />
        <PageState v-if="!loading && !loadError && visibleDevices.length === 0" kind="empty"
            :title="query ? $gettext('没有匹配的设备') : $gettext('当前筛选下没有设备')"
            :description="query ? $gettext('请尝试名称、IP、MAC 或厂商的其他关键词。') : $gettext('可以切换到“全部”查看本次开机发现过的设备。')"
            :action-label="filter === 'online' && counts.all > 0 && !query && !activeAdvancedFilterCount ? $gettext('查看全部设备') : ''" @action="filter = 'all'" />

        <div v-if="!loading && !loadError && visibleDevices.length" class="device-table-wrap">
            <table class="device-table">
                <colgroup><col class="identity-column" /><col class="address-column" /><col class="status-column" /><col class="connection-column" /><col class="traffic-column" /><col /></colgroup>
                <thead>
                    <tr>
                        <th>{{ $gettext('设备') }}</th>
                        <th>{{ $gettext('地址') }}</th>
                        <th>{{ $gettext('状态') }}</th>
                        <th>{{ $gettext('连接') }}</th>
                        <th>{{ $gettext('实时流量') }}</th>
                        <th>{{ $gettext('策略与操作') }}</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="device in visibleDevices" :key="rowKey(device)" :data-device-id="rowKey(device)" tabindex="0" @click="openDetails(device)" @keydown.enter.prevent="openDetails(device)" @keydown.space.prevent="openDetails(device)">
                        <td>
                            <div class="device-identity">
                                <DeviceSceneIcon :scene="device.scene" :icon-key="device.iconKey" :label="sceneLabel(device.scene)" />
                                <div>
                                    <strong :title="displayName(device)">
                                        <template v-for="(part, index) in highlight(displayName(device))" :key="index">
                                            <mark v-if="part.matched">{{ part.text }}</mark><template v-else>{{ part.text }}</template>
                                        </template>
                                    </strong>
                                    <small v-if="aggregate" class="network-label">{{ networkLabel(device) }}</small><span class="identity-meta" :title="device.brand || categoryMeta(device)"><b v-if="identityPresentation(device).showBrandInMeta">{{ device.brand }}</b><template v-if="identityPresentation(device).showBrandInMeta"> · </template>{{ categoryMeta(device) }}</span>
                                </div>
                            </div>
                        </td>
                        <td class="device-address"><span class="mono">{{ device.primaryAddress || '—' }}</span><small v-if="device.extraAddressCount">+{{ device.extraAddressCount }}</small></td>
                        <td><span class="status-pill" :class="presenceClass(device)">{{ presenceLabel(device) }}</span></td>
                        <td>{{ connectionLabel(device.connection.kind) }}</td>
                        <td><div class="traffic" :class="{ 'traffic-ready': device.telemetry?.state === 'ready' }"><span class="traffic-upload">↑ {{ trafficLabel(device, 'up') }}</span><span class="traffic-download">↓ {{ trafficLabel(device, 'down') }}</span></div></td>
                        <td>
                            <div class="policy-action">
                                <div class="policy-list">
                                    <span v-for="policy in device.policyLabels" :key="policy">{{ policyLabel(policy) }}</span>
                                    <span v-if="device.policyLabels.length === 0" class="muted">{{ $gettext('无策略') }}</span>
                                </div>
                                <button type="button" @click.stop="openDetails(device)">{{ $gettext('详情') }}</button>
                            </div>
                        </td>
                    </tr>
                </tbody>
            </table>

            <article v-for="device in visibleDevices" :key="`card-${rowKey(device)}`" :data-device-id="rowKey(device)" class="device-card" tabindex="0" @click="openDetails(device)" @keydown.enter.prevent="openDetails(device)" @keydown.space.prevent="openDetails(device)">
                <div class="device-card__top">
                    <div class="device-identity">
                        <DeviceSceneIcon :scene="device.scene" :icon-key="device.iconKey" :label="sceneLabel(device.scene)" />
                        <div><strong :title="displayName(device)">{{ displayName(device) }}</strong><small v-if="aggregate" class="network-label">{{ networkLabel(device) }}</small><span class="identity-meta"><b v-if="identityPresentation(device).showBrandInMeta">{{ device.brand }}</b><template v-if="identityPresentation(device).showBrandInMeta"> · </template>{{ categoryMeta(device) }}</span></div>
                    </div>
                    <span class="status-pill" :class="presenceClass(device)">{{ presenceLabel(device) }}</span>
                </div>
                <div class="device-card__facts">
                    <span><small>{{ $gettext('地址') }}</small><span class="mono">{{ device.primaryAddress || '—' }}</span></span>
                    <span><small>{{ $gettext('连接') }}</small>{{ connectionLabel(device.connection.kind) }}</span>
                    <span :class="{ 'traffic-ready': device.telemetry?.state === 'ready' }"><small>{{ $gettext('实时流量') }}</small><span><span class="traffic-upload">↑ {{ trafficLabel(device, 'up') }}</span> · <span class="traffic-download">↓ {{ trafficLabel(device, 'down') }}</span></span></span>
                    <span><small>{{ $gettext('策略') }}</small>{{ device.policyLabels.length ? device.policyLabels.map(policyLabel).join(' · ') : $gettext('无策略') }}</span>
                </div>
            </article>
        </div>

        <div v-if="selected" class="drawer-backdrop" @click.self="closeDetails">
            <LANScopeProvider v-if="!aggregate || lan.networks.some(network => network.networkId === selected?.networkId)" :key="rowKey(selected)" :network-id="selected.networkId || lan.networkId" :networks="lan.networks" :legacy="lan.legacy">
            <aside ref="drawer" class="device-drawer" role="dialog" aria-modal="true" tabindex="-1" :aria-label="$gettext('设备详情')" @keydown.esc="closeDetails">
                <div class="drawer-header">
                    <div class="device-identity">
                        <DeviceSceneIcon :scene="selected.scene" :icon-key="selected.iconKey" :label="sceneLabel(selected.scene)" />
                        <div><h3>{{ displayName(selected) }}</h3><span class="identity-meta"><b v-if="identityPresentation(selected).showBrandInMeta">{{ selected.brand }}</b><template v-if="identityPresentation(selected).showBrandInMeta"> · </template>{{ categoryMeta(selected) }}</span></div>
                    </div>
                    <button type="button" class="drawer-close" @click="closeDetails" :aria-label="$gettext('关闭')">×</button>
                </div>
                <p v-if="aggregate" class="network-label">{{ $gettext('所在局域网') }} · {{ networkLabel(selected) }}</p>
                <div class="drawer-summary" role="status">
                    <span><small>{{ $gettext('状态') }}</small>{{ presenceLabel(selected) }}</span>
                    <span><small>{{ $gettext('主要地址') }}</small><b class="mono">{{ selected.primaryAddress || '—' }}</b></span>
                    <span><small>{{ $gettext('连接') }}</small>{{ connectionLabel(selected.connection.kind) }}</span>
                </div>
                <div v-if="!selectedVisible" class="context-notice" role="status"><span>{{ $gettext('此设备不在当前筛选结果中') }}</span><button type="button" @click="clearAllFilters">{{ $gettext('清除筛选') }}</button></div>
                <nav class="drawer-tabs" :aria-label="$gettext('设备详情区域')">
                    <button v-for="tab in detailTabs" :key="tab.id" type="button" :class="{ active: detailTab === tab.id }" @click="detailTab = tab.id">{{ tab.label }}</button>
                </nav>
                <div v-if="detailTab === 'overview'" class="drawer-section">
                    <h4>{{ $gettext('概览') }}</h4>
                    <dl>
                        <div><dt>{{ $gettext('当前速率') }}</dt><dd>↑ {{ trafficLabel(selected, 'up') }} · ↓ {{ trafficLabel(selected, 'down') }}</dd></div>
                        <div><dt>{{ $gettext('本次开机') }}</dt><dd>↑ {{ trafficTotal(selected, 'up') }} · ↓ {{ trafficTotal(selected, 'down') }}</dd></div>
                        <div><dt>{{ $gettext('连接数') }}</dt><dd>{{ selected.telemetry?.state === 'warming_up' || !selected.telemetry ? $gettext('采集中') : selected.telemetry.connectionCount }}</dd></div>
                    </dl>
                    <div v-if="trafficHistory[rowKey(selected)]?.length" class="traffic-chart" :aria-label="$gettext('近期流量趋势')">
                        <span v-for="(point, index) in trafficHistory[rowKey(selected)]" :key="index" class="traffic-chart__bar">
                            <i class="upload" :style="{ height: `${chartHeight(rowKey(selected), point.up)}%` }"></i>
                            <i class="download" :style="{ height: `${chartHeight(rowKey(selected), point.down)}%` }"></i>
                        </span>
                    </div>
                    <small class="chart-legend"><span>↑ {{ $gettext('上传') }}</span><span>↓ {{ $gettext('下载') }}</span></small>
                    <TrafficInsightsPanel :key="selected.deviceId" :device-id="selected.deviceId" :quota-editable="false" />
                    <AdvancedNetworkTools :key="`advanced-${selected.deviceId}`" :device="selected" />
                </div>
                <div v-if="detailTab === 'profile'" class="drawer-section">
                    <h4>{{ $gettext('设备资料') }}</h4>
                    <DeviceProfileEditor :device="selected" @saved="handleProfileSaved" />
                    <details class="technical-details">
                        <summary>{{ $gettext('识别与技术信息') }}</summary>
                        <dl>
                            <div><dt>{{ $gettext('原始厂商') }}</dt><dd>{{ selected.classification.manufacturer || '—' }}</dd></div>
                            <div><dt>{{ $gettext('原始主机名') }}</dt><dd>{{ selected.hostname || '—' }}</dd></div>
                            <div><dt>{{ $gettext('识别依据') }}</dt><dd>{{ classificationSourceLabel(selected.classification.source) }}</dd></div>
                            <div><dt>{{ $gettext('识别可信度') }}</dt><dd>{{ confidenceLabel(selected.classification.confidence) }}</dd></div>
                            <div><dt>MAC</dt><dd class="mono">{{ selected.mac || '—' }}</dd></div>
                            <div v-if="selected.identity?.scope === 'boot'"><dt>{{ $gettext('身份稳定性') }}</dt><dd>{{ $gettext('本次开机') }}</dd></div>
                            <div><dt>{{ $gettext('最近出现') }}</dt><dd>{{ formatLastSeen(selected.lastSeenAt) }}</dd></div>
                        </dl>
                    </details>
                </div>
                <div v-if="detailTab === 'network'" class="drawer-section">
                    <h4>{{ $gettext('网络与上网') }}</h4>
                    <DevicePolicyPanel :key="`network-${selected.deviceId}`" :device="selected" mode="network" @saved="handlePolicySaved" />
                    <details class="technical-details">
                        <summary>{{ $gettext('地址详情') }}</summary>
                        <h4>{{ $gettext('当前地址') }}</h4>
                    <div v-if="selected.addresses.current.length" class="address-list">
                        <button v-for="address in selected.addresses.current" :key="address.address" type="button"
                            class="address-copy" :title="$gettext('复制地址')" @click="copyAddress(address.address)">
                            <code>{{ address.address }}</code><span>{{ copiedAddress === address.address ? $gettext('已复制') : $gettext('复制') }}</span>
                        </button>
                    </div>
                    <span v-else class="muted">{{ $gettext('当前没有地址') }}</span>
                    <h4 v-if="selected.addresses.historical.length">{{ $gettext('历史地址') }}</h4>
                    <div class="address-list">
                        <button v-for="address in selected.addresses.historical" :key="address.address" type="button"
                            class="address-copy" :title="$gettext('复制地址')" @click="copyAddress(address.address)">
                            <code>{{ address.address }}</code><span>{{ copiedAddress === address.address ? $gettext('已复制') : $gettext('复制') }}</span>
                        </button>
                    </div>
                    </details>
                </div>
                <div v-if="detailTab === 'management'" class="drawer-section">
                    <h4>{{ $gettext('使用管理') }}</h4>
                    <DevicePolicyPanel :key="`restrictions-${selected.deviceId}`" :device="selected" mode="restrictions" @saved="handlePolicySaved" />
                    <DeviceUsagePolicyEditor :key="`usage-policy-${selected.deviceId}`" :device-id="selected.deviceId" />
                </div>
            </aside>
            </LANScopeProvider>
            <aside v-else ref="drawer" class="device-drawer" role="dialog" aria-modal="true" tabindex="-1" :aria-label="$gettext('设备详情')" @keydown.esc="closeDetails"><h3>{{ displayName(selected) }}</h3><p>{{ $gettext('归属尚未确认，暂时无法修改此设备') }}</p><p class="mono">{{ selected.mac }}</p><p v-for="address in [...selected.addresses.current, ...selected.addresses.historical]" :key="address.address" class="mono">{{ address.address }}</p><button type="button" @click="closeDetails">{{ $gettext('关闭') }}</button></aside>
        </div>
    </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import baseRequest from '/@/request'
import { useLANClient, useLANContext, deviceScopeKey } from './lanContext'
import LANScopeProvider from './lanScope.vue'
import { createDeviceManagement } from '/@/request/request'
const lan = useLANContext()
const aggregate = lan.networkId === 'all'
const rowKey = deviceScopeKey
const networkLabel = (device: { networkId?: string }) => lan.networks.find(item => item.networkId === device.networkId)?.displayName || $gettext('待确认归属')
const request = { ...baseRequest, DeviceMangement: useLANClient() }
import {useRoute, useRouter} from 'vue-router'
const route=useRoute(), router=useRouter()
import LoadError from './components/loadError.vue'
import PageState from './components/pageState.vue'
import DeviceProfileEditor from './components/deviceProfileEditor.vue'
import DevicePolicyPanel from './components/devicePolicyPanel.vue'
import DeviceUsagePolicyEditor from './components/deviceUsagePolicyEditor.vue'
import DeviceSceneIcon from './components/deviceSceneIcon.vue'
import TrafficInsightsPanel from './components/trafficInsightsPanel.vue'
import AdvancedNetworkTools from './components/advancedNetworkTools.vue'
import { requestErrorMessage } from './requestError'
import { deviceDisplayName, devicePresenceLabel, deviceCategoryLabel } from './devicePresentation'
import {
    buildDeviceListItems,
    deviceIdentityPresentation,
    deviceCounts,
    selectDeviceListItems,
    splitHighlight,
    type DeviceFilter,
    type DeviceClassification,
    type DeviceListItem,
    type DeviceListCriteria,
    type DeviceSort,
    type InventoryDevice,
    type DeviceNetworkRule,
} from './deviceInventory'
import type { DeviceScene } from './deviceScene'
import type { DeviceIconKey } from './deviceScene'
import {
    formatTrafficBytes,
    telemetryBackoff,
    telemetryByDevice,
    telemetrySpeedLabel,
    type DeviceTelemetryItem,
    type DeviceTelemetryState,
} from './deviceTelemetry'

const { $gettext } = useGettext()

const devices = ref<DeviceListItem[]>([])
const health = ref<{ state: string; reasons: string[] }>({ state: 'ready', reasons: [] })
const loading = ref(true)
const adding = ref(false), addingDevice = ref(false), addError = ref('')
const manual = ref({ alias: '', mac: '' })
const loadError = ref('')
const filter = ref<DeviceFilter>(route.query.filter === 'all' ? 'all' : 'online')
const sort = ref<DeviceSort>('recent')
const query = ref('')
const criteria = ref<Required<DeviceListCriteria>>({ connection: 'all', brand: '', policy: 'all' })
const selected = ref<DeviceListItem | null>(null)
const drawer = ref<HTMLElement | null>(null)
const detailTab = ref<'overview'|'profile'|'network'|'management'>('overview')
const copiedAddress = ref('')
const telemetryItems = ref(new Map<string, DeviceTelemetryItem>())
const telemetryHealth = ref<DeviceTelemetryState>('warming_up')
const trafficHistory = ref<Record<string, Array<{ up: number; down: number }>>>({})
let telemetryTimer: number | undefined
let telemetryFailures = 0
let disposed = false
let previousFocus: HTMLElement | null = null
let previousDeviceID = ''

const counts = computed(() => deviceCounts(devices.value))
const visibleDevices = computed(() => selectDeviceListItems(devices.value, filter.value, query.value, sort.value, criteria.value))
const availableBrands = computed(() => [...new Set(devices.value.map(device => device.brand).filter(Boolean))].sort((left, right) => left.localeCompare(right)))
const activeAdvancedFilterCount = computed(() => Number(criteria.value.connection !== 'all') + Number(Boolean(criteria.value.brand)) + Number(criteria.value.policy !== 'all'))
const selectedVisible = computed(() => !selected.value || visibleDevices.value.some(device => rowKey(device) === rowKey(selected.value!)))
const filters = computed(() => [
    { value: 'online' as DeviceFilter, label: $gettext('在线') },
    { value: 'all' as DeviceFilter, label: $gettext('全部') },
    { value: 'controlled' as DeviceFilter, label: $gettext('已设置') },
])
const detailTabs = computed(() => [
    { id: 'overview' as const, label: $gettext('概览') },
    { id: 'profile' as const, label: $gettext('设备资料') },
    { id: 'network' as const, label: $gettext('网络与上网') },
    { id: 'management' as const, label: $gettext('使用管理') },
])

const load = async () => {
    loading.value = true
    loadError.value = ''
    try {
        const [inventoryResponse, networkRulesResponse] = await Promise.all([
            request.DeviceMangement.deviceInventoryV2.GET(),
            aggregate ? Promise.resolve(null) : request.DeviceMangement.networkRulesV2.GET().catch(() => null),
        ])
        const inventoryData = inventoryResponse.data
        if (!inventoryData?.result) throw inventoryData?.error || $gettext('设备清单不可用')
        const inventoryDevices: InventoryDevice[] = inventoryData.result.devices || []
        const networkRules: DeviceNetworkRule[] | undefined = networkRulesResponse?.data?.result?.rules
        if (aggregate) {
            const byNetwork = await Promise.all(lan.networks.map(async network => {
                const response = await createDeviceManagement(network.networkId).networkRulesV2.GET().catch(() => null)
                return buildDeviceListItems(inventoryDevices.filter(device => device.networkId === network.networkId), [], undefined, response?.data?.result?.rules)
            }))
            devices.value = [...byNetwork.flat(), ...buildDeviceListItems(inventoryDevices.filter(device => !lan.networks.some(network => network.networkId === device.networkId)))]
        } else devices.value = buildDeviceListItems(inventoryDevices, [], undefined, networkRules)
        applyTelemetry(telemetryItems.value)
        health.value = inventoryData.result.health || { state: 'ready', reasons: [] }
        if (selected.value) {
            selected.value = devices.value.find(device => rowKey(device) === rowKey(selected.value!)) || null
        }
    } catch (error) {
        loadError.value = requestErrorMessage(error, `${$gettext('读取结果失败')}，${$gettext('请刷新界面')}`)
    } finally {
        loading.value = false
    }
}

const displayName = (device: DeviceListItem) => deviceDisplayName(device, $gettext)
const presenceLabel = (device: DeviceListItem) => devicePresenceLabel(device, $gettext)
const presenceClass = (device: DeviceListItem) => device.presenceState === 'never_seen' ? 'never-seen' : device.online ? 'online' : 'offline'
const connectionLabel = (kind: string) => kind === 'wifi' ? 'Wi-Fi' : kind === 'lan' ? $gettext('有线') : $gettext('未知')
const sceneLabel = (scene: DeviceScene) => deviceCategoryLabel(scene, $gettext)
const categoryMeta = (device: DeviceListItem) => {
    const presentation = identityPresentation(device)
    if (presentation.metaMode === 'pending') return $gettext('类型待确认')
    if (presentation.metaMode === 'classification-source') return classificationSourceLabel(device.classification.source)
    return sceneLabel(device.scene)
}
const identityPresentation = (device: DeviceListItem) => deviceIdentityPresentation(device)
const classificationSourceLabel = (source: DeviceListItem['classification']['source']) => ({
    model: $gettext('型号特征'),
    hostname: $gettext('设备名称'),
    manufacturer_default: $gettext('厂商默认类型'),
    fallback: $gettext('安全回退'),
    manual: $gettext('手动设置'),
}[source])
const confidenceLabel = (confidence: DeviceListItem['classification']['confidence']) => ({
    high: $gettext('高'),
    medium: $gettext('中'),
    low: $gettext('低'),
}[confidence])
const policyLabel = (policy: string) => ({ static: $gettext('静态地址'), route: $gettext('已指定路线'), limited: $gettext('已限速'), blocked: $gettext('已断网') }[policy] || policy)
const trafficLabel = (device: DeviceListItem, direction: 'up' | 'down') => telemetrySpeedLabel(device.telemetry, direction, $gettext('采集中'))
const trafficTotal = (device: DeviceListItem, direction: 'up' | 'down') => {
    if (!device.telemetry || device.telemetry.state === 'warming_up') return $gettext('采集中')
    return formatTrafficBytes(direction === 'up' ? device.telemetry.uploadBytes : device.telemetry.downloadBytes)
}
const chartHeight = (deviceId: string, value: number) => {
    const points = trafficHistory.value[deviceId] || []
    const max = Math.max(1, ...points.flatMap(point => [point.up, point.down]))
    return Math.max(3, Math.round(value * 100 / max))
}
const highlight = (value: string) => splitHighlight(value, query.value)
const clearAdvancedFilters = () => { criteria.value = { connection: 'all', brand: '', policy: 'all' } }
const clearAllFilters = () => { filter.value = 'all'; query.value = ''; clearAdvancedFilters() }
const openDetails = async (device: DeviceListItem) => { previousFocus = document.activeElement as HTMLElement | null; previousDeviceID = rowKey(device); selected.value = device; detailTab.value = 'overview'; if(route.query.device !== device.deviceId || route.query.deviceNetwork !== device.networkId) void router.replace({query:{...route.query,device:device.deviceId,deviceNetwork:device.networkId}}); await nextTick(); drawer.value?.focus() }
const closeDetails = async () => {
    selected.value = null
    if(route.query.device) void router.replace({query:{...route.query,device:undefined,deviceNetwork:undefined}})
    await nextTick()
    const fallback = Array.from(document.querySelectorAll<HTMLElement>('[data-device-id]')).find(element => element.dataset.deviceId === previousDeviceID && element.offsetParent !== null)
    ;(previousFocus?.isConnected ? previousFocus : fallback)?.focus()
    previousFocus = null
    previousDeviceID = ''
}
const handlePolicySaved = (labels: string[]) => {
    if (!selected.value) return
    selected.value.policyLabels = labels
    selected.value.controlled = labels.length > 0
}
const handleProfileSaved = (value: { alias: string; classification: DeviceClassification; iconKey: DeviceIconKey }) => {
    if (!selected.value) return
    const deviceId = rowKey(selected.value)
    devices.value = devices.value.map(device => rowKey(device) === deviceId ? {
        ...device,
        displayName: value.alias || device.hostname || '',
        classification: value.classification,
        scene: value.classification.category,
        brand: value.classification.brand,
        iconKey: value.iconKey,
    } : device)
    selected.value = devices.value.find(device => rowKey(device) === deviceId) || null
}
const copyAddress = async (value: string) => {
    try {
        await navigator.clipboard.writeText(value)
    } catch (_) {
        const input = document.createElement('textarea')
        input.value = value
        input.style.position = 'fixed'
        input.style.opacity = '0'
        document.body.appendChild(input)
        input.select()
        document.execCommand('copy')
        input.remove()
    }
    copiedAddress.value = value
    window.setTimeout(() => { if (copiedAddress.value === value) copiedAddress.value = '' }, 1500)
}
const addManualDevice = async () => {
    addingDevice.value = true; addError.value = ''
    try {
        const response = await request.DeviceMangement.deviceInventoryV2.POST({ ...manual.value })
        if (!response.data?.result) throw new Error(response.data?.error || $gettext('添加失败'))
        manual.value = { alias: '', mac: '' }; adding.value = false; await load(); filter.value = 'all'
    } catch (error: any) { addError.value = error?.response?.data?.error || error?.message || $gettext('添加失败，请检查 MAC 地址') }
    finally { addingDevice.value = false }
}
const formatLastSeen = (value: string) => {
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

const applyTelemetry = (items: Map<string, DeviceTelemetryItem>) => {
    for (const device of devices.value) device.telemetry = items.get(rowKey(device))
    if (selected.value) selected.value = devices.value.find(device => rowKey(device) === rowKey(selected.value!)) || null
}
const recordTrafficHistory = (items: DeviceTelemetryItem[]) => {
    const next = { ...trafficHistory.value }
    for (const item of items) {
        if (item.state !== 'ready') continue
        next[rowKey(item)] = [...(next[rowKey(item)] || []), { up: item.uploadSpeed, down: item.downloadSpeed }].slice(-20)
    }
    trafficHistory.value = next
}
const scheduleTelemetry = (delay: number) => {
    if (telemetryTimer !== undefined) window.clearTimeout(telemetryTimer)
    telemetryTimer = undefined
    if (!disposed && !document.hidden) telemetryTimer = window.setTimeout(loadTelemetry, delay)
}
const loadTelemetry = async () => {
    if (document.hidden) return
    try {
        const response = await request.DeviceMangement.deviceTrafficV2.GET()
        if (!response.data?.result) throw response.data?.error || new Error('telemetry unavailable')
        const items: DeviceTelemetryItem[] = response.data.result.items || []
        telemetryItems.value = telemetryByDevice(items, rowKey)
        telemetryHealth.value = response.data.result.health?.state || 'ready'
        telemetryFailures = 0
        applyTelemetry(telemetryItems.value)
        recordTrafficHistory(items)
        scheduleTelemetry(3000)
    } catch (_) {
        telemetryHealth.value = 'partial'
        telemetryFailures++
        scheduleTelemetry(telemetryBackoff(telemetryFailures))
    }
}
const handleVisibility = () => {
    if (document.hidden) {
        if (telemetryTimer !== undefined) window.clearTimeout(telemetryTimer)
        telemetryTimer = undefined
        return
    }
    loadTelemetry()
}

watch(()=>route.query.device, value=>{if(!value)selected.value=null})
onMounted(async () => {
    await load()
    if (disposed) return
    const requested=typeof route.query.device==='string'?route.query.device:''
    if(requested){const matches=devices.value.filter(device=>device.deviceId===requested&&(!route.query.deviceNetwork||device.networkId===route.query.deviceNetwork));const target=matches.length===1?matches[0]:undefined;if(target){filter.value='all';await openDetails(target)}}
    document.addEventListener('visibilitychange', handleVisibility)
    loadTelemetry()
})
onUnmounted(() => {
    disposed = true
    document.removeEventListener('visibilitychange', handleVisibility)
    if (telemetryTimer !== undefined) window.clearTimeout(telemetryTimer)
})
</script>

<style lang="scss" scoped>
.device-center { color: var(--device-text); container-type: inline-size; }
.device-toolbar { display: grid; grid-template-columns: auto auto minmax(0,1fr); gap: 8px 12px; align-items: start; margin-bottom: 14px; }
.device-center__tools { grid-column: 3; grid-row: 1; display: flex; justify-content: flex-end; align-items: center; gap: 8px; min-width: 0; }
.device-search { flex: 1; max-width: 320px; min-width: 120px; }
.device-search input { box-sizing: border-box; width: 100%; min-height: 44px; padding: 7px 12px; color: inherit; background: var(--device-panel); border: 1px solid var(--device-line); border-radius: 7px; }
.device-center__tools select { width: auto; max-width: 100%; }
select, .icon-button { min-height: 44px; color: inherit; background: var(--device-panel); border: 1px solid var(--device-line); border-radius: 7px; }
.add-device { min-height:44px;padding:7px 11px;color:var(--device-primary-text);background:var(--device-primary-bg);border:1px solid var(--device-primary-bg);border-radius:7px;cursor:pointer;white-space:nowrap }.add-device-form{display:grid;grid-template-columns:1.5fr 1fr 1fr auto;gap:9px;align-items:end;margin:0 0 14px;padding:12px;background:var(--device-soft);border:1px solid var(--device-line);border-radius:9px}.add-device-form>div,.add-device-form label{display:grid;gap:4px}.add-device-form small{opacity:.62}.add-device-form label span{font-size:12px}.add-device-form input{box-sizing:border-box;min-width:0;min-height:36px;padding:7px 9px;color:inherit;background:transparent;border:1px solid rgba(127,127,127,.28);border-radius:7px}.add-device-form button{min-height:36px;padding:7px 11px;color:var(--device-primary-text);background:var(--device-primary-bg);border:1px solid var(--device-primary-bg);border-radius:7px}.add-device-form p{grid-column:1/-1;margin:0;color:#9b3b16}
select { padding: 0 28px 0 10px; }
.icon-button { flex: none; width: 44px; cursor: pointer; font-size: 22px; }
.device-filters { display: flex; gap: 3px; margin: 0; }
.device-filters button { min-width: 44px; min-height: 44px; padding: 7px 10px; white-space: nowrap; color: var(--device-muted); background: transparent; border: 1px solid transparent; border-radius: 7px; cursor: pointer; }
.device-filters button span { margin-left: 4px; font-size: 12px; }
.device-filters button.active { color: var(--device-accent); background: var(--device-accent-soft); border-color: var(--device-line); }
.filter-panel { grid-column: 2; grid-row: 1; margin: 0; border: 0; }.filter-panel[open] { grid-column: 1 / -1; grid-row: auto; }.filter-panel > summary { display: inline-flex; align-items: center; gap: 7px; min-height: 44px; color: var(--device-muted); cursor: pointer; font-size: 12px; white-space: nowrap; }.filter-panel > summary small { display: inline-grid; place-items: center; min-width: 20px; height: 20px; color: #fff; background: var(--device-accent); border-radius: 999px; }.filter-panel__controls { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 9px; padding: 12px; background: var(--device-soft); border: 1px solid var(--device-line); border-radius: 8px; }.filter-panel__controls label { display: grid; gap: 4px; min-width: 145px; }.filter-panel__controls label span { font-size: 12px; opacity: .65; }.clear-filters,.context-notice button { min-height: 36px; padding: 6px 10px; color: var(--device-accent); background: transparent; border: 1px solid var(--device-line); border-radius: 7px; cursor: pointer; }
.telemetry-notice { margin: 0 0 12px; padding: 8px 11px; color: #8a5a00; background: #fff8e8; border: 1px solid #f3d18a; border-radius: 7px; font-size: 12px; }
.device-center :deep(.page-state) { color: var(--device-text); background: var(--device-soft); border-color: var(--device-line); }
.device-center :deep(.page-state__icon),.device-center :deep(.page-state button) { color: var(--device-accent); }
.device-center :deep(.page-state button) { min-height: 44px; border-color: var(--device-line); }
.device-center :deep(.page-state--empty) { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16px; min-height: 280px; padding: 36px 20px; text-align: center; background: transparent; border: 0; }
.device-center :deep(.page-state--empty .page-state__body) { flex: none; max-width: 38ch; }
.device-center :deep(.page-state--empty .page-state__icon) { width: 48px; height: 48px; font-size: 26px; background: var(--device-soft); border: 0; }
.device-table-wrap { width: 100%; overflow: hidden; }
.device-table { width: 100%; table-layout: fixed; border-collapse: collapse; }
.identity-column { width: clamp(180px,24cqw,280px); }.address-column { width: clamp(145px,19cqw,220px); }.status-column { width: 90px; }.connection-column { width: 76px; }.traffic-column { width: 104px; }
.device-table th { padding: 11px 10px; text-align: left; white-space: nowrap; font-size: 12px; font-weight: 500; color: var(--device-muted); background: var(--device-soft); border-bottom: 1px solid var(--device-line); }
.device-table td { padding: 13px 10px; border-bottom: 1px solid var(--device-line); vertical-align: middle; overflow-wrap: anywhere; }
.device-table tbody tr:nth-child(n) { background: var(--device-panel); cursor: pointer; transition: background .15s ease; }
.device-table tbody tr:nth-child(n):hover { background: var(--device-hover); }
.device-table tbody tr:focus-visible,.device-card:focus-visible { outline: 2px solid var(--device-accent); outline-offset: -2px; background: var(--device-hover); }
.device-identity { display: flex; align-items: center; gap: 10px; min-width: 0; }
.device-identity div { display: flex; flex-direction: column; min-width: 0; }
.device-identity strong { max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 14px; font-weight: 600; }
.device-identity .identity-meta { margin-top: 4px; color: var(--device-muted); font-size: 12px; }
.identity-meta { display: block; max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.identity-meta b { font-weight: 500; }
mark { color: inherit; background: #fff0a6; border-radius: 2px; }
.status-pill { display: inline-flex; align-items: center; gap: 6px; white-space: nowrap; }
.status-pill::before { width: 7px; height: 7px; content: ''; border-radius: 50%; background: #a8adb7; }
.status-pill.online { color: var(--device-good); }.status-pill.online::before { background: currentColor; }
.status-pill.offline,.status-pill.never-seen { color: var(--device-muted); }
.device-address { font-size: 13px; overflow-wrap: anywhere; }
.mono, code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
td > small { margin-left: 5px; color: var(--device-accent); }
.traffic { display: flex; flex-direction: column; gap: 3px; white-space: nowrap; font-size: 13px; }
.traffic-ready .traffic-upload { color: var(--device-upload); }.traffic-ready .traffic-download { color: var(--device-download); }
.policy-action { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px; }
.policy-list { display: flex; flex: 1; min-width: 80px; flex-wrap: wrap; gap: 5px; }
.policy-list > span:not(.muted) { padding: 3px 7px; color: var(--device-accent); background: var(--device-accent-soft); border-radius: 999px; font-size: 12px; }
.policy-action button, .secondary-button { min-height: 44px; padding: 6px 10px; color: var(--device-text); background: var(--device-panel); border: 1px solid var(--device-line); border-radius: 7px; cursor: pointer; white-space: nowrap; }
button:focus-visible, input:focus-visible, select:focus-visible { outline: 2px solid var(--device-accent); outline-offset: 2px; }
.clear-filters,.context-notice button,.add-device-form button,.drawer-tabs button,.address-copy { min-height: 44px; }
.muted { color: var(--device-muted); }
.device-card { display: none; }
.drawer-backdrop { position: fixed; inset: 0; z-index: 1000; display: flex; justify-content: flex-end; background: rgba(0, 0, 0, .36); }
.device-drawer { width: min(430px, 100%); height: 100%; padding: 22px; overflow-y: auto; color: var(--device-text); background: var(--device-panel); box-shadow: -8px 0 28px rgba(0, 0, 0, .18); }
.drawer-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.device-drawer h3 { margin: 0; padding: 0; color: inherit; background: none !important; font-size: 19px; text-align: left; }
.drawer-close { flex: none; width: 44px; height: 44px; color: inherit; background: transparent; border: 0; cursor: pointer; font-size: 28px; }
.drawer-summary { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 7px; margin: 16px 0 10px; }.drawer-summary > span { display: grid; gap: 3px; min-width: 0; padding: 9px; background: var(--device-soft); border-radius: 8px; overflow-wrap: anywhere; }.drawer-summary small { opacity: .58; }.context-notice { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin: 0 0 10px; padding: 8px 10px; color: #745500; background: rgba(255,196,77,.12); border-radius: 7px; font-size: 12px; }.drawer-tabs { position: sticky; top: -22px; z-index: 2; display: flex; gap: 4px; margin: 0 -22px; padding: 9px 22px; overflow-x: auto; background: var(--card-bg-color); border-bottom: 1px solid rgba(127,127,127,.12); }.drawer-tabs button { flex: none; padding: 7px 9px; color: inherit; background: transparent; border: 0; border-radius: 7px; cursor: pointer; }.drawer-tabs button.active { color: var(--device-accent); background: var(--device-accent-soft); font-weight: 600; }
.drawer-section { padding: 18px 0; border-bottom: 1px solid rgba(127, 127, 127, .14); }
.drawer-section h4 { margin: 0 0 10px; font-size: 14px; }
dl { margin: 0; }
dl div { display: grid; grid-template-columns: 100px 1fr; gap: 12px; padding: 7px 0; }
dt { opacity: .58; }
dd { margin: 0; overflow-wrap: anywhere; }
.address-list { display: flex; flex-direction: column; gap: 7px; }
code { padding: 7px 9px; overflow-wrap: anywhere; background: rgba(127, 127, 127, .07); border-radius: 5px; }
.address-copy { display: flex; align-items: center; justify-content: space-between; gap: 10px; width: 100%; padding: 0; color: inherit; text-align: left; background: transparent; border: 0; cursor: pointer; }
.address-copy code { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.address-copy span { flex: none; color: var(--device-accent); font-size: 12px; }
.traffic-chart { display: flex; align-items: flex-end; gap: 3px; height: 72px; margin-top: 13px; padding: 8px; background: rgba(127, 127, 127, .055); border-radius: 7px; }
.traffic-chart__bar { display: flex; flex: 1; align-items: flex-end; justify-content: center; gap: 1px; height: 100%; }
.traffic-chart__bar i { display: block; width: min(4px, 45%); min-height: 2px; border-radius: 2px 2px 0 0; }
.traffic-chart__bar .upload { background: var(--device-upload); }
.traffic-chart__bar .download { background: var(--device-download); }
.chart-legend { display: flex; gap: 14px; margin-top: 6px; }.chart-legend>span:first-child{color:var(--device-upload)}.chart-legend>span:last-child{color:var(--device-download)}
.drawer-footer { padding-top: 20px; }
.technical-details { margin-top: 14px; padding-top: 10px; border-top: 1px solid rgba(127,127,127,.14); }.technical-details summary { color: var(--device-accent); cursor: pointer; font-weight: 600; }.technical-details[open] summary { margin-bottom: 8px; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }

@media (max-width: 980px) {
    .device-center__tools { width: 100%; }
    .device-search { flex: 1; }
    .device-search input { width: 100%; }
    .add-device-form { grid-template-columns:1fr 1fr }.add-device-form>div,.add-device-form button{grid-column:1/-1}
    .device-table th:nth-child(4), .device-table td:nth-child(4) { display: none; }
    .connection-column { display: none; }
}

@container (max-width: 760px) {
    .device-toolbar { grid-template-columns: minmax(0,1fr) auto; }.device-center__tools { grid-column:1/-1; grid-row:1; width:100%; }.device-filters{grid-column:1;grid-row:2}.filter-panel{grid-column:2;grid-row:2}
}
@container (max-width: 760px) {
    .device-center__tools { flex-wrap: wrap; }
    .device-search { flex-basis: 100%; max-width: none; }
    .device-center__tools select { flex: 1; min-width: 0; width: 0; }
    .device-table { display: none; }
    .device-card { display: block; margin-bottom: 10px; padding: 14px; border: 1px solid var(--device-line); border-radius: 9px; cursor: pointer; }
    .device-card__top { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
    .device-card__top>.device-identity { flex: 1; }.device-card__top>.status-pill { flex: none; font-size:12px; }
    .device-card__facts { display: grid; grid-template-columns: 1fr 1fr; gap: 10px 12px; margin-top: 12px; font-size:13px; }
    .device-card__facts > span { display: flex; flex-direction: column; min-width: 0; overflow-wrap: anywhere; }
    .device-card__facts small { margin-bottom: 3px; color: var(--device-muted); font-size:12px; }
    .identity-meta { max-width: min(220px, 58vw); }
    .filter-panel__controls label { flex: 1 1 135px; min-width: 0; }
}
@container (max-width: 320px) {
    .device-toolbar { grid-template-columns: minmax(0,1fr); }
    .device-filters { grid-column: 1; grid-row: 2; flex-wrap: wrap; }
    .filter-panel { grid-column: 1; grid-row: 3; }
    .device-search input { min-width: 0; }
}
@media (max-width: 360px) {
    .device-toolbar { grid-template-columns: minmax(0,1fr); }
    .device-filters { justify-content: space-between; }
    .filter-panel { grid-column: 1; grid-row: 3; }
}

@media (max-width: 420px) {
    .device-toolbar { gap:8px 4px; }.device-filters button { padding:7px 6px; }
    .device-filters button { flex: none; }
    .device-card__facts > span:first-child,.device-card__facts > span:last-child { grid-column:1/-1; }
    .device-drawer { padding: 18px 16px; }
    .drawer-summary { grid-template-columns: 1fr 1fr; }.drawer-summary > span:last-child { grid-column: 1 / -1; }.drawer-tabs { top: -18px; margin: 0 -16px; padding: 8px 16px; }
    .add-device-form{grid-template-columns:1fr}.add-device-form>*{grid-column:1!important}.add-device{flex:1}
    .filter-panel__controls { display: grid; grid-template-columns: 1fr; }.clear-filters { width: 100%; }
}
</style>
