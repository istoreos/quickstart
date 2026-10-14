<template>
    <section class="device-center" aria-labelledby="device-center-title">
        <div class="device-center__header">
            <div>
                <h2 id="device-center-title">{{ $gettext('设备') }}</h2>
                <p>{{ $gettext('查看当前在线和本次开机发现过的局域网设备') }}</p>
            </div>
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
                <button class="add-device" type="button" @click="adding = !adding">{{ adding ? $gettext('取消添加') : $gettext('添加设备') }}</button>
            </div>
        </div>

        <form v-if="adding" class="add-device-form" @submit.prevent="addManualDevice"><div><strong>{{ $gettext('添加尚未上线的设备') }}</strong><small>{{ $gettext('保存 MAC 和备注，设备首次出现后会自动合并，不会创建孤立规则。') }}</small></div><label><span>{{ $gettext('设备备注') }}</span><input v-model.trim="manual.alias" maxlength="64" :placeholder="$gettext('例如：孩子的电脑')" /></label><label><span>MAC</span><input v-model.trim="manual.mac" required autocomplete="off" placeholder="AA:BB:CC:DD:EE:FF" /></label><button type="submit" :disabled="addingDevice">{{ addingDevice ? $gettext('正在添加…') : $gettext('确认添加') }}</button><p v-if="addError" role="alert">{{ addError }}</p></form>

        <nav class="device-filters" :aria-label="$gettext('设备筛选')">
            <button v-for="item in filters" :key="item.value" type="button"
                :class="{ active: filter === item.value }" @click="filter = item.value">
                {{ item.label }} <span>{{ counts[item.value] }}</span>
            </button>
        </nav>

        <details class="filter-panel">
            <summary>{{ $gettext('更多筛选') }}<small v-if="activeAdvancedFilterCount">{{ activeAdvancedFilterCount }}</small></summary>
            <div class="filter-panel__controls">
                <label><span>{{ $gettext('连接方式') }}</span><select v-model="criteria.connection"><option value="all">{{ $gettext('全部连接') }}</option><option value="lan">{{ $gettext('有线') }}</option><option value="wifi">Wi-Fi</option><option value="unknown">{{ $gettext('未知') }}</option></select></label>
                <label><span>{{ $gettext('品牌') }}</span><select v-model="criteria.brand"><option value="">{{ $gettext('全部品牌') }}</option><option v-for="brand in availableBrands" :key="brand" :value="brand">{{ brand }}</option></select></label>
                <label><span>{{ $gettext('策略状态') }}</span><select v-model="criteria.policy"><option value="all">{{ $gettext('全部策略') }}</option><option value="none">{{ $gettext('无策略') }}</option><option value="static">{{ $gettext('静态地址') }}</option><option value="route">{{ $gettext('已指定路线') }}</option><option value="limited">{{ $gettext('已限速') }}</option><option value="blocked">{{ $gettext('已断网') }}</option></select></label>
                <button v-if="activeAdvancedFilterCount" type="button" class="clear-filters" @click="clearAdvancedFilters">{{ $gettext('清除筛选') }}</button>
            </div>
        </details>

        <div v-if="telemetryHealth === 'partial' || telemetryHealth === 'stale'" class="telemetry-notice" role="status">
            {{ telemetryHealth === 'stale' ? $gettext('流量数据更新延迟，正在恢复采样') : $gettext('流量采样暂不可用，设备清单仍可正常使用') }}
        </div>

        <PageState v-if="loading" kind="loading" :title="$gettext('正在加载设备')" :description="$gettext('正在整理设备身份和在线状态…')" />
        <LoadError v-else-if="loadError" :message="loadError" @retry="load" />
        <PageState v-else-if="health.state === 'partial'" kind="info" :title="$gettext('部分设备信息暂不可用')"
            :description="$gettext('设备清单仍可使用，缺失的信息会在数据源恢复后自动补齐。')" />
        <PageState v-if="!loading && !loadError && visibleDevices.length === 0" kind="empty"
            :title="query ? $gettext('没有匹配的设备') : $gettext('当前筛选下没有设备')"
            :description="query ? $gettext('请尝试名称、IP、MAC 或厂商的其他关键词。') : $gettext('可以切换到“全部”查看本次开机发现过的设备。')" />

        <div v-if="!loading && !loadError && visibleDevices.length" class="device-table-wrap">
            <table class="device-table">
                <thead>
                    <tr>
                        <th>{{ $gettext('设备') }}</th>
                        <th>{{ $gettext('状态') }}</th>
                        <th>{{ $gettext('地址') }}</th>
                        <th>{{ $gettext('连接') }}</th>
                        <th>{{ $gettext('实时流量') }}</th>
                        <th>{{ $gettext('策略与操作') }}</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="device in visibleDevices" :key="device.deviceId" :data-device-id="device.deviceId" tabindex="0" @click="openDetails(device)" @keydown.enter.prevent="openDetails(device)" @keydown.space.prevent="openDetails(device)">
                        <td>
                            <div class="device-identity">
                                <DeviceSceneIcon :scene="device.scene" :icon-key="device.iconKey" :label="sceneLabel(device.scene)" />
                                <div>
                                    <strong>
                                        <template v-for="(part, index) in highlight(displayName(device))" :key="index">
                                            <mark v-if="part.matched">{{ part.text }}</mark><template v-else>{{ part.text }}</template>
                                        </template>
                                    </strong>
                                    <span class="identity-meta" :title="device.brand || categoryMeta(device)"><b v-if="identityPresentation(device).showBrandInMeta">{{ device.brand }}</b><template v-if="identityPresentation(device).showBrandInMeta"> · </template>{{ categoryMeta(device) }}</span>
                                </div>
                            </div>
                        </td>
                        <td><span class="status-pill" :class="presenceClass(device)">{{ presenceLabel(device) }}</span></td>
                        <td><span class="mono">{{ device.primaryAddress || '—' }}</span><small v-if="device.extraAddressCount">+{{ device.extraAddressCount }}</small></td>
                        <td>{{ connectionLabel(device.connection.kind) }}</td>
                        <td><div class="traffic"><span>↑ {{ trafficLabel(device, 'up') }}</span><span>↓ {{ trafficLabel(device, 'down') }}</span></div></td>
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

            <article v-for="device in visibleDevices" :key="`card-${device.deviceId}`" :data-device-id="device.deviceId" class="device-card" tabindex="0" @click="openDetails(device)" @keydown.enter.prevent="openDetails(device)" @keydown.space.prevent="openDetails(device)">
                <div class="device-card__top">
                    <div class="device-identity">
                        <DeviceSceneIcon :scene="device.scene" :icon-key="device.iconKey" :label="sceneLabel(device.scene)" />
                        <div><strong>{{ displayName(device) }}</strong><span class="identity-meta"><b v-if="identityPresentation(device).showBrandInMeta">{{ device.brand }}</b><template v-if="identityPresentation(device).showBrandInMeta"> · </template>{{ categoryMeta(device) }}</span></div>
                    </div>
                    <span class="status-pill" :class="presenceClass(device)">{{ presenceLabel(device) }}</span>
                </div>
                <div class="device-card__facts">
                    <span><small>{{ $gettext('地址') }}</small><span class="mono">{{ device.primaryAddress || '—' }}</span></span>
                    <span><small>{{ $gettext('连接') }}</small>{{ connectionLabel(device.connection.kind) }}</span>
                    <span><small>{{ $gettext('实时流量') }}</small>↑ {{ trafficLabel(device, 'up') }} · ↓ {{ trafficLabel(device, 'down') }}</span>
                    <span><small>{{ $gettext('策略') }}</small>{{ device.policyLabels.length ? device.policyLabels.map(policyLabel).join(' · ') : $gettext('无策略') }}</span>
                </div>
            </article>
        </div>

        <div v-if="selected" class="drawer-backdrop" @click.self="closeDetails">
            <aside ref="drawer" class="device-drawer" role="dialog" aria-modal="true" tabindex="-1" :aria-label="$gettext('设备详情')" @keydown.esc="closeDetails">
                <div class="drawer-header">
                    <div class="device-identity">
                        <DeviceSceneIcon :scene="selected.scene" :icon-key="selected.iconKey" :label="sceneLabel(selected.scene)" />
                        <div><h3>{{ displayName(selected) }}</h3><span class="identity-meta"><b v-if="identityPresentation(selected).showBrandInMeta">{{ selected.brand }}</b><template v-if="identityPresentation(selected).showBrandInMeta"> · </template>{{ categoryMeta(selected) }}</span></div>
                    </div>
                    <button type="button" class="drawer-close" @click="closeDetails" :aria-label="$gettext('关闭')">×</button>
                </div>
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
                    <div v-if="trafficHistory[selected.deviceId]?.length" class="traffic-chart" :aria-label="$gettext('近期流量趋势')">
                        <span v-for="(point, index) in trafficHistory[selected.deviceId]" :key="index" class="traffic-chart__bar">
                            <i class="upload" :style="{ height: `${chartHeight(selected.deviceId, point.up)}%` }"></i>
                            <i class="download" :style="{ height: `${chartHeight(selected.deviceId, point.down)}%` }"></i>
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
        </div>
    </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import LoadError from './components/loadError.vue'
import PageState from './components/pageState.vue'
import DeviceProfileEditor from './components/deviceProfileEditor.vue'
import DevicePolicyPanel from './components/devicePolicyPanel.vue'
import DeviceUsagePolicyEditor from './components/deviceUsagePolicyEditor.vue'
import DeviceSceneIcon from './components/deviceSceneIcon.vue'
import TrafficInsightsPanel from './components/trafficInsightsPanel.vue'
import AdvancedNetworkTools from './components/advancedNetworkTools.vue'
import { requestErrorMessage } from './requestError'
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
const filter = ref<DeviceFilter>('online')
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
let previousFocus: HTMLElement | null = null
let previousDeviceID = ''

const counts = computed(() => deviceCounts(devices.value))
const visibleDevices = computed(() => selectDeviceListItems(devices.value, filter.value, query.value, sort.value, criteria.value))
const availableBrands = computed(() => [...new Set(devices.value.map(device => device.brand).filter(Boolean))].sort((left, right) => left.localeCompare(right)))
const activeAdvancedFilterCount = computed(() => Number(criteria.value.connection !== 'all') + Number(Boolean(criteria.value.brand)) + Number(criteria.value.policy !== 'all'))
const selectedVisible = computed(() => !selected.value || visibleDevices.value.some(device => device.deviceId === selected.value?.deviceId))
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
            request.DeviceMangement.networkRulesV2.GET().catch(() => null),
        ])
        const inventoryData = inventoryResponse.data
        if (!inventoryData?.result) throw inventoryData?.error || $gettext('设备清单不可用')
        const inventoryDevices: InventoryDevice[] = inventoryData.result.devices || []
        const networkRules: DeviceNetworkRule[] | undefined = networkRulesResponse?.data?.result?.rules
        devices.value = buildDeviceListItems(inventoryDevices, [], undefined, networkRules)
        applyTelemetry(telemetryItems.value)
        health.value = inventoryData.result.health || { state: 'ready', reasons: [] }
        if (selected.value) {
            selected.value = devices.value.find(device => device.deviceId === selected.value?.deviceId) || null
        }
    } catch (error) {
        loadError.value = requestErrorMessage(error, `${$gettext('读取结果失败')}，${$gettext('请刷新界面')}`)
    } finally {
        loading.value = false
    }
}

const displayName = (device: DeviceListItem) => {
    const name = device.displayName || device.hostname || ''
    if (name) return name
    if (device.brand) return `${device.brand} ${device.classification.source === 'fallback' ? $gettext('设备') : sceneLabel(device.scene)}`
    return $gettext('未命名设备')
}
const presenceLabel = (device: DeviceListItem) => device.presenceState === 'never_seen' ? $gettext('尚未上线') : device.online ? $gettext('在线') : $gettext('离线')
const presenceClass = (device: DeviceListItem) => device.presenceState === 'never_seen' ? 'never-seen' : device.online ? 'online' : 'offline'
const connectionLabel = (kind: string) => kind === 'wifi' ? 'Wi-Fi' : kind === 'lan' ? $gettext('有线') : $gettext('未知')
const sceneLabel = (scene: DeviceScene) => ({
    phone: $gettext('手机'),
    computer: $gettext('电脑'),
    tablet: $gettext('平板'),
    tv: $gettext('电视与影音'),
    network: $gettext('网络设备'),
    'smart-home': $gettext('智能家居'),
    camera: $gettext('摄像头'),
    gaming: $gettext('游戏设备'),
    storage: $gettext('存储与服务器'),
    printer: $gettext('打印机'),
    wearable: $gettext('穿戴设备'),
    unknown: $gettext('未知设备'),
}[scene])
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
const openDetails = async (device: DeviceListItem) => { previousFocus = document.activeElement as HTMLElement | null; previousDeviceID = device.deviceId; selected.value = device; detailTab.value = 'overview'; await nextTick(); drawer.value?.focus() }
const closeDetails = async () => {
    selected.value = null
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
    const deviceId = selected.value.deviceId
    devices.value = devices.value.map(device => device.deviceId === deviceId ? {
        ...device,
        displayName: value.alias || device.hostname || '',
        classification: value.classification,
        scene: value.classification.category,
        brand: value.classification.brand,
        iconKey: value.iconKey,
    } : device)
    selected.value = devices.value.find(device => device.deviceId === deviceId) || null
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
    for (const device of devices.value) device.telemetry = items.get(device.deviceId)
    if (selected.value) selected.value = devices.value.find(device => device.deviceId === selected.value?.deviceId) || null
}
const recordTrafficHistory = (items: DeviceTelemetryItem[]) => {
    const next = { ...trafficHistory.value }
    for (const item of items) {
        if (item.state !== 'ready') continue
        next[item.deviceId] = [...(next[item.deviceId] || []), { up: item.uploadSpeed, down: item.downloadSpeed }].slice(-20)
    }
    trafficHistory.value = next
}
const scheduleTelemetry = (delay: number) => {
    if (telemetryTimer !== undefined) window.clearTimeout(telemetryTimer)
    telemetryTimer = undefined
    if (!document.hidden) telemetryTimer = window.setTimeout(loadTelemetry, delay)
}
const loadTelemetry = async () => {
    if (document.hidden) return
    try {
        const response = await request.DeviceMangement.deviceTrafficV2.GET()
        if (!response.data?.result) throw response.data?.error || new Error('telemetry unavailable')
        const items: DeviceTelemetryItem[] = response.data.result.items || []
        telemetryItems.value = telemetryByDevice(items)
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

onMounted(async () => {
    await load()
    document.addEventListener('visibilitychange', handleVisibility)
    loadTelemetry()
})
onUnmounted(() => {
    document.removeEventListener('visibilitychange', handleVisibility)
    if (telemetryTimer !== undefined) window.clearTimeout(telemetryTimer)
})
</script>

<style lang="scss" scoped>
.device-center { color: var(--flow-span-color); }
.device-center__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin-bottom: 14px; }
.device-center__header h2 { margin: 0; padding: 0; color: inherit; background: none !important; font-size: 20px; text-align: left; }
.device-center__header p { margin: 5px 0 0; opacity: .65; font-size: 13px; }
.device-center__tools { display: flex; align-items: center; gap: 8px; }
.device-search input { width: 280px; max-width: 100%; min-height: 36px; padding: 7px 12px; color: inherit; background: transparent; border: 1px solid rgba(127, 127, 127, .3); border-radius: 7px; }
select, .icon-button { min-height: 36px; color: inherit; background: transparent; border: 1px solid rgba(127, 127, 127, .3); border-radius: 7px; }
.add-device { min-height:36px;padding:7px 11px;color:#fff;background:#553afe;border:1px solid #553afe;border-radius:7px;cursor:pointer;white-space:nowrap }.add-device-form{display:grid;grid-template-columns:1.5fr 1fr 1fr auto;gap:9px;align-items:end;margin:0 0 14px;padding:12px;background:rgba(85,58,254,.05);border:1px solid rgba(85,58,254,.14);border-radius:9px}.add-device-form>div,.add-device-form label{display:grid;gap:4px}.add-device-form small{opacity:.62}.add-device-form label span{font-size:12px}.add-device-form input{box-sizing:border-box;min-width:0;min-height:36px;padding:7px 9px;color:inherit;background:transparent;border:1px solid rgba(127,127,127,.28);border-radius:7px}.add-device-form button{min-height:36px;padding:7px 11px;color:#fff;background:#553afe;border:1px solid #553afe;border-radius:7px}.add-device-form p{grid-column:1/-1;margin:0;color:#9b3b16}
select { padding: 0 28px 0 10px; }
.icon-button { width: 38px; cursor: pointer; font-size: 19px; }
.device-filters { display: flex; gap: 6px; margin-bottom: 14px; }
.device-filters button { padding: 7px 12px; color: inherit; background: transparent; border: 1px solid transparent; border-radius: 999px; cursor: pointer; }
.device-filters button span { margin-left: 5px; opacity: .6; }
.device-filters button.active { color: #553afe; background: rgba(85, 58, 254, .09); border-color: rgba(85, 58, 254, .18); }
.filter-panel { margin: -5px 0 14px; border: 0; }.filter-panel > summary { display: inline-flex; align-items: center; gap: 7px; min-height: 34px; color: #553afe; cursor: pointer; font-size: 13px; }.filter-panel > summary small { display: inline-grid; place-items: center; min-width: 20px; height: 20px; color: #fff; background: #553afe; border-radius: 999px; }.filter-panel__controls { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 9px; padding: 10px; background: rgba(127,127,127,.045); border: 1px solid rgba(127,127,127,.12); border-radius: 8px; }.filter-panel__controls label { display: grid; gap: 4px; min-width: 145px; }.filter-panel__controls label span { font-size: 12px; opacity: .65; }.clear-filters,.context-notice button { min-height: 36px; padding: 6px 10px; color: #553afe; background: transparent; border: 1px solid rgba(85,58,254,.35); border-radius: 7px; cursor: pointer; }
.telemetry-notice { margin: 0 0 12px; padding: 8px 11px; color: #8a5a00; background: #fff8e8; border: 1px solid #f3d18a; border-radius: 7px; font-size: 12px; }
.device-table-wrap { width: 100%; overflow: hidden; }
.device-table { width: 100%; border-collapse: collapse; }
.device-table th { padding: 12px 10px; text-align: left; white-space: nowrap; font-size: 13px; font-weight: 500; opacity: .7; background: rgba(127, 127, 127, .055); }
.device-table td { padding: 15px 10px; border-bottom: 1px solid rgba(127, 127, 127, .11); vertical-align: middle; }
.device-table tbody tr { cursor: pointer; transition: background .15s ease; }
.device-table tbody tr:hover { background: rgba(85, 58, 254, .035); }
.device-table tbody tr:focus-visible,.device-card:focus-visible { outline: 2px solid #553afe; outline-offset: -2px; background: rgba(85,58,254,.04); }
.device-identity { display: flex; align-items: center; gap: 10px; min-width: 0; }
.device-identity div { display: flex; flex-direction: column; min-width: 0; }
.device-identity strong { max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.device-identity span { margin-top: 2px; opacity: .72; font-size: 12px; }
.identity-meta { display: block; max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.identity-meta b { color: #553afe; font-weight: 600; }
mark { color: inherit; background: #fff0a6; border-radius: 2px; }
.status-pill { display: inline-flex; align-items: center; gap: 6px; white-space: nowrap; }
.status-pill::before { width: 7px; height: 7px; content: ''; border-radius: 50%; background: #a8adb7; }
.status-pill.online::before { background: #26a269; box-shadow: 0 0 0 3px rgba(38, 162, 105, .12); }
.status-pill.offline { opacity: .65; }
.status-pill.never-seen::before { background:#806cff;box-shadow:0 0 0 3px rgba(128,108,255,.12) }.status-pill.never-seen{color:#624bd1}
.mono, code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
td > small { margin-left: 5px; color: #553afe; }
.traffic { display: flex; flex-direction: column; gap: 3px; white-space: nowrap; font-size: 13px; }
.policy-action { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.policy-list { display: flex; flex-wrap: wrap; gap: 5px; }
.policy-list > span:not(.muted) { padding: 3px 7px; color: #553afe; background: rgba(85, 58, 254, .08); border-radius: 999px; font-size: 12px; }
.policy-action button, .secondary-button { padding: 6px 10px; color: #553afe; background: transparent; border: 1px solid rgba(85, 58, 254, .45); border-radius: 6px; cursor: pointer; white-space: nowrap; }
button:focus-visible, input:focus-visible, select:focus-visible { outline: 2px solid #553afe; outline-offset: 2px; }
.muted { opacity: .55; }
.device-card { display: none; }
.drawer-backdrop { position: fixed; inset: 0; z-index: 1000; display: flex; justify-content: flex-end; background: rgba(0, 0, 0, .36); }
.device-drawer { width: min(430px, 100%); height: 100%; padding: 22px; overflow-y: auto; color: var(--flow-span-color); background: #fff; box-shadow: -8px 0 28px rgba(0, 0, 0, .18); }
.drawer-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.device-drawer h3 { margin: 0; padding: 0; color: inherit; background: none !important; font-size: 19px; text-align: left; }
.drawer-close { color: inherit; background: transparent; border: 0; cursor: pointer; font-size: 28px; }
.drawer-summary { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 7px; margin: 16px 0 10px; }.drawer-summary > span { display: grid; gap: 3px; min-width: 0; padding: 9px; background: rgba(85,58,254,.05); border-radius: 8px; overflow-wrap: anywhere; }.drawer-summary small { opacity: .58; }.context-notice { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin: 0 0 10px; padding: 8px 10px; color: #745500; background: rgba(255,196,77,.12); border-radius: 7px; font-size: 12px; }.drawer-tabs { position: sticky; top: -22px; z-index: 2; display: flex; gap: 4px; margin: 0 -22px; padding: 9px 22px; overflow-x: auto; background: var(--card-bg-color); border-bottom: 1px solid rgba(127,127,127,.12); }.drawer-tabs button { flex: none; padding: 7px 9px; color: inherit; background: transparent; border: 0; border-radius: 7px; cursor: pointer; }.drawer-tabs button.active { color: #553afe; background: rgba(85,58,254,.09); font-weight: 600; }
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
.address-copy span { flex: none; color: #553afe; font-size: 12px; }
.traffic-chart { display: flex; align-items: flex-end; gap: 3px; height: 72px; margin-top: 13px; padding: 8px; background: rgba(127, 127, 127, .055); border-radius: 7px; }
.traffic-chart__bar { display: flex; flex: 1; align-items: flex-end; justify-content: center; gap: 1px; height: 100%; }
.traffic-chart__bar i { display: block; width: min(4px, 45%); min-height: 2px; border-radius: 2px 2px 0 0; }
.traffic-chart__bar .upload { background: #8c7cf0; }
.traffic-chart__bar .download { background: #32a873; }
.chart-legend { display: flex; gap: 14px; margin-top: 6px; opacity: .65; }
.drawer-footer { padding-top: 20px; }
.technical-details { margin-top: 14px; padding-top: 10px; border-top: 1px solid rgba(127,127,127,.14); }.technical-details summary { color: #553afe; cursor: pointer; font-weight: 600; }.technical-details[open] summary { margin-bottom: 8px; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }

@media (max-width: 980px) {
    .device-center__header { flex-direction: column; }
    .device-center__tools { width: 100%; }
    .device-search { flex: 1; }
    .device-search input { width: 100%; }
    .add-device-form { grid-template-columns:1fr 1fr }.add-device-form>div,.add-device-form button{grid-column:1/-1}
    .device-table th:nth-child(4), .device-table td:nth-child(4) { display: none; }
}

@media (max-width: 700px) {
    .device-center__tools { flex-wrap: wrap; }
    .device-search { flex-basis: 100%; }
    .device-table { display: none; }
    .device-card { display: block; margin-bottom: 10px; padding: 14px; border: 1px solid rgba(127, 127, 127, .15); border-radius: 9px; cursor: pointer; }
    .device-card__top { display: flex; align-items: flex-start; justify-content: space-between; gap: 10px; }
    .device-card__facts { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-top: 15px; }
    .device-card__facts > span { display: flex; flex-direction: column; min-width: 0; overflow-wrap: anywhere; }
    .device-card__facts small { margin-bottom: 3px; opacity: .55; }
    .identity-meta { max-width: min(220px, 58vw); }
    .filter-panel__controls label { flex: 1 1 135px; min-width: 0; }
}

@media (max-width: 420px) {
    .device-center__header p { max-width: 30ch; }
    .device-filters { overflow-x: auto; }
    .device-filters button { flex: none; }
    .device-card__facts { grid-template-columns: 1fr; }
    .device-drawer { padding: 18px 16px; }
    .drawer-summary { grid-template-columns: 1fr 1fr; }.drawer-summary > span:last-child { grid-column: 1 / -1; }.drawer-tabs { top: -18px; margin: 0 -16px; padding: 8px 16px; }
    .add-device-form{grid-template-columns:1fr}.add-device-form>*{grid-column:1!important}.add-device{flex:1}
    .filter-panel__controls { display: grid; grid-template-columns: 1fr; }.clear-filters { width: 100%; }
}
@media (prefers-color-scheme: dark) { .device-drawer { background: #202024; } }
:global(body[theme="dark"]) .device-drawer { background: #202024; }
:global(body[theme="light"]) .device-drawer { background: #fff; }
</style>
