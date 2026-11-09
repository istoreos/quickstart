<template>
    <main id="page" class="device-management" :aria-label="$gettext('局域网设备管理')">
        <nav class="breadcrumbs" :aria-label="$gettext('面包屑')"><router-link to="/">{{ $gettext('首页') }}</router-link><span aria-hidden="true">›</span><span>{{ $gettext('局域网设备管理') }}</span></nav>
        <p v-if="loading" role="status">{{ $gettext('正在读取局域网…') }}</p>
        <p v-else-if="error" role="alert">{{ error }} <button type="button" @click="loadNetworks">{{ $gettext('重试') }}</button></p>
        <p v-if="!loading && !error && !validNetwork" role="alert">{{ $gettext('此局域网已不存在或不可访问') }} <button type="button" @click="selectNetwork(networks.length === 1 ? networks[0].networkId : 'all')">{{ $gettext('返回设备列表') }}</button></p>
        <div v-if="!loading && !error && validNetwork" class="device-workspace" :class="{ 'has-network-nav': multipleNetworks }">
            <aside v-if="multipleNetworks" class="network-sidebar">
                <h2>{{ $gettext('网络范围') }}</h2>
                <nav class="network-navigation" :aria-label="$gettext('选择局域网')">
                    <button type="button" data-network="all" :aria-current="networkId === 'all' ? 'true' : undefined" @click="selectNetwork('all')">
                        <HomeGlyph name="layers" />
                        <span><strong>{{ $gettext('全部局域网') }}</strong><small>{{ $gettext('设备总览') }}</small></span>
                    </button>
                    <div class="network-divider"></div>
                    <button v-for="network in networks" :key="network.networkId" type="button" :data-network="network.networkId" :aria-current="networkId === network.networkId ? 'true' : undefined" @click="selectNetwork(network.networkId)">
                        <HomeGlyph name="network" />
                        <span><strong>{{ network.displayName }}</strong><small>{{ networkSummary(network) }}</small></span>
                    </button>
                </nav>
                <p>{{ $gettext('网络配置请前往系统网络设置') }}</p>
            </aside>
            <div class="workspace-panel">
                <label v-if="multipleNetworks" class="lan-selector">{{ $gettext('局域网') }}
                    <select :value="networkId" @change="selectNetwork(($event.target as HTMLSelectElement).value)"><option value="all">{{ $gettext('全部局域网') }}</option><option v-for="network in networks" :key="network.networkId" :value="network.networkId">{{ network.displayName }}</option></select>
                </label>
                <div v-if="multipleNetworks" class="scope-heading">
                    <h1>{{ currentNetwork?.displayName || $gettext('全部局域网') }}</h1>
                    <p>{{ currentNetwork ? $gettext('此网络的设备、分组与设置') : $gettext('查看各个局域网中的设备') }}</p>
                </div>
                <nav class="primary-tabs" role="tablist" :aria-label="$gettext('局域网设备管理')">
                    <button v-for="tab in tabs" :id="`lan-tab-${tab.id}`" :key="tab.id" type="button" role="tab" aria-controls="lan-panel" :aria-selected="activeTab === tab.id" :class="{ active: activeTab === tab.id }" @click="selectTab(tab.id)"><HomeGlyph :name="tab.icon" />{{ tab.label }}</button>
                </nav>
                <section id="lan-panel" class="primary-content" role="tabpanel" :aria-labelledby="`lan-tab-${activeTab}`">
                    <p v-if="currentNetwork?.readOnlyReason" class="scope-notice" role="status">{{ readOnlyMessage }}</p>
                    <p v-if="networkId === 'all' && activeTab !== 'devices'" class="scope-placeholder" role="status">{{ $gettext('请选择一个局域网，管理分组、计划和网络设置') }}</p>
                    <LANScopeProvider v-else :key="`${networkId}:${activeTab}:${legacy}`" :network-id="networkId" :networks="networks" :legacy="legacy">
                        <DeviceCenterList v-if="activeTab === 'devices'" />
                        <DeviceGroupsPanel v-else-if="activeTab === 'groups'" />
                        <LanSettingsPanel v-else />
                    </LANScopeProvider>
                </section>
            </div>
        </div>
    </main>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import LANScopeProvider from './lanScope.vue'
import { discoverLANs, type DiscoveredLAN } from './lanDiscovery'
import { useGettext } from '/@/plugins/i18n'
import DeviceCenterList from './deviceCenterList.vue'
import DeviceGroupsPanel from './deviceGroupsPanel.vue'
import LanSettingsPanel from './lanSettingsPanel.vue'
import HomeGlyph from '/@/home/HomeGlyph.vue'
import './deviceVisualTokens.css'

const { $gettext } = useGettext()
const route = useRoute(), router = useRouter()
const networks = ref<DiscoveredLAN[]>([]), loading = ref(true), error = ref(''), legacy = ref(false)
const initialNetwork = ref('')
const networkId = computed(() => typeof route.query.network === 'string' ? route.query.network : initialNetwork.value)
const currentNetwork = computed(() => networks.value.find(item => item.networkId === networkId.value))
const multipleNetworks = computed(() => networks.value.length > 1)
function networkSummary(network: DiscoveredLAN) {
    if (network.state === 'down') return $gettext('未连接')
    return network.ipv4Prefixes?.[0] || network.ipv6Prefixes?.[0] || (network.readOnlyReason ? $gettext('暂时只读') : $gettext('局域网'))
}
const readOnlyMessage = computed(() => currentNetwork.value?.readOnlyReason === 'network_down' ? $gettext('此局域网尚未连接，网络设置暂时只读') : currentNetwork.value?.readOnlyReason === 'shared_network_device' ? $gettext('此局域网与其他网络共用连接，请先确认独立的网络配置') : $gettext('此局域网的地址或连接尚未就绪，网络设置暂时只读'))
const validNetwork = computed(() => networkId.value === 'all' && !legacy.value || Boolean(currentNetwork.value))
const activeTab = computed(() => ['groups', 'settings'].includes(String(route.query.tab)) ? String(route.query.tab) : 'devices')
function selectNetwork(network: string) {
    try { localStorage.setItem('quickstart.lan', network) } catch { /* Storage can be disabled. */ }
    void router.replace({ query: { ...route.query, network, device: undefined } })
}
function selectTab(tab: string) { void router.replace({ query: { ...route.query, tab } }) }
async function loadNetworks() {
    loading.value = true; error.value = ''
    try {
        const result = await discoverLANs(); networks.value = result.networks; legacy.value = result.legacy
        let saved = ''; try { saved = localStorage.getItem('quickstart.lan') || '' } catch { /* Optional preference. */ }
        initialNetwork.value = result.networks.length === 1 ? result.networks[0].networkId : saved === 'all' || result.networks.some(item => item.networkId === saved) ? saved : 'all'
    } catch (reason) { error.value = reason instanceof Error ? $gettext(reason.message) : $gettext('局域网列表不可用') }
    finally { loading.value = false }
}
onMounted(loadNetworks)
const tabs = computed(() => [
    { id: 'devices' as const, icon: 'devices', label: $gettext('设备') },
    { id: 'groups' as const, icon: 'layers', label: $gettext('分组与计划') },
    { id: 'settings' as const, icon: 'tune', label: $gettext('局域网设置') },
])
</script>

<style lang="scss" scoped>
.device-management { width: 100%; margin: 0; color: var(--device-text); font-size: 14px; }
.breadcrumbs { display: flex; align-items: center; gap: 7px; margin-bottom: 24px; padding-top: 4px; font-size: 13px; }
.breadcrumbs a { color: var(--breadcrumbs-tit-color); text-decoration: none; }
.breadcrumbs span:last-child { color: var(--breadcrumbs-tit-color1); }
.device-workspace { max-width: 1640px; margin: 0 auto; }
.device-workspace.has-network-nav { display: grid; grid-template-columns: 208px minmax(0, 1fr); gap: 20px; align-items: start; }
.network-sidebar { position: sticky; top: 24px; padding: 22px 12px 16px; color: var(--device-text); background: var(--device-panel); border: 1px solid var(--device-line); border-radius: 12px; }
.network-sidebar h2 { margin: 0 12px 18px; padding: 0; color: var(--device-muted); background: none; border: 0; box-shadow: none; text-align: left; font-size: 12px; font-weight: 500; }
.network-navigation { display: grid; gap: 6px; }
.network-navigation button { display: flex; align-items: center; gap: 12px; width: 100%; min-height: 66px; margin: 0; padding: 12px; text-align: left; color: var(--device-text); background: transparent; border: 0; border-radius: 9px; cursor: pointer; }
.network-navigation button:hover { background: var(--device-soft); }
.network-navigation button[aria-current="true"] { color: var(--device-accent); background: var(--device-accent-soft); }
.network-navigation svg { flex: none; width: 21px; height: 21px; }
.network-navigation button > span { display: grid; gap: 5px; min-width: 0; }
.network-navigation strong { font-size: 14px; font-weight: 600; overflow-wrap: anywhere; }
.network-navigation small { color: var(--device-muted); font-size: 11px; overflow-wrap: anywhere; }
.network-divider { height: 1px; margin: 8px 12px; background: var(--device-line); }
.network-sidebar > p { margin: 28px 12px 0; padding-top: 16px; color: var(--device-muted); border-top: 1px solid var(--device-line); font-size: 11px; line-height: 1.8; }
.workspace-panel { min-width: 0; background: var(--device-panel); border: 1px solid var(--device-line); border-radius: 12px; }
.scope-heading { padding: 26px 26px 14px; }
.scope-heading h1 { margin: 0; padding: 0; color: var(--device-text); background: none; border: 0; box-shadow: none; text-align: left; font-size: 25px; font-weight: 650; line-height: 1.4; letter-spacing: -.5px; }
.scope-heading p { margin: 7px 0 0; color: var(--device-muted); font-size: 13px; }
.primary-tabs { display: flex; gap: 24px; margin: 0 26px; border-bottom: 1px solid var(--device-line); }
.primary-tabs button { display: inline-flex; align-items: center; justify-content: center; gap: 8px; min-height: 52px; margin: 0; padding: 10px 2px; color: var(--device-muted); background: transparent; border: 0; border-bottom: 2px solid transparent; cursor: pointer; }
.primary-tabs button svg { width: 17px; height: 17px; flex: none; }
.primary-tabs button.active { color: var(--device-accent); border-bottom-color: var(--device-accent); font-weight: 600; }
.primary-tabs button:focus-visible, .network-navigation button:focus-visible { outline: 2px solid var(--device-accent); outline-offset: 2px; }
.primary-content { min-height: 58vh; padding: 24px 26px; }
.scope-notice { margin: 0 0 18px; padding: 12px 16px; color: var(--device-muted); background: var(--device-soft); border-radius: 8px; font-size: 13px; line-height: 1.6; }
.scope-placeholder { margin: 0; padding: 70px 18px; color: var(--device-muted); text-align: center; line-height: 1.8; }
.lan-selector { display: none; }
@media(max-width:900px) {
    .device-workspace.has-network-nav { display: block; }
    .network-sidebar { display: none; }
    .lan-selector { display: flex; align-items: center; gap: 10px; margin: 18px 22px 0; color: var(--device-muted); font-size: 12px; }
    .lan-selector select { flex: 1; min-width: 0; min-height: 44px; padding: 8px 12px; color: var(--device-text); background: var(--device-soft); border: 1px solid var(--device-line); border-radius: 7px; font: inherit; }
    .scope-heading { padding: 20px 22px 12px; }
    .primary-tabs { margin: 0 22px; }
    .primary-content { padding: 20px 22px; }
}
@media(max-width:420px) {
    .breadcrumbs { margin-bottom: 18px; }
    .scope-heading { padding: 18px 16px 8px; }
    .scope-heading h1 { font-size: 22px; }
    .lan-selector { margin: 16px 16px 0; }
    .primary-tabs { gap: 8px; margin: 0 16px; }
    .primary-tabs button { flex: 1; min-width: 0; gap: 5px; padding: 8px 0; font-size: 12px; white-space: normal; }
    .primary-tabs button svg { display: none; }
    .primary-content { padding: 18px 14px; }
}
</style>

<style lang="scss">
.device-management header::before,
.device-management header::after { content: none!important; display: none!important; pointer-events: none!important; }
.device-management .group-header,
.device-management .lan-settings > header { color: var(--tit-color)!important; background: transparent!important; }
@media(max-width:420px){a.btn[href="/cgi-bin/luci/admin/system/admin"]{max-width:140px;white-space:normal;overflow-wrap:anywhere;text-align:center}}
@media(prefers-color-scheme:dark){body:not([theme="light"]){background:#151518}}
body[theme="dark"],body[data-quickstart-theme="dark"]{background:#151518}
@media(prefers-color-scheme:dark){
    body:not([theme="light"]) .device-management .feedback.error,
    body:not([theme="light"]) .device-management .batch-result.partial,
    body:not([theme="light"]) .device-management .batch-result.failed,
    body:not([theme="light"]) .device-management .state-pill.orphaned,
    body:not([theme="light"]) .device-management .state-pill.unsupported,
    body:not([theme="light"]) .device-management .capability-pill.not_installed,
    body:not([theme="light"]) .device-management .capability-pill.error { color:#ffbd8a!important; background:rgba(255,145,77,.12)!important }
    body:not([theme="light"]) .device-management .feedback.success,
    body:not([theme="light"]) .device-management .batch-result.all_success,
    body:not([theme="light"]) .device-management .capability-pill.available { color:#7cdda9!important; background:rgba(38,162,105,.14)!important }
    body:not([theme="light"]) .device-management .telemetry-notice,
    body:not([theme="light"]) .device-management .attention { color:#ffd37a!important; background:rgba(255,196,77,.11)!important; border-color:rgba(255,196,77,.3)!important }
}
body[theme="dark"] .device-management .feedback.error,
body[theme="dark"] .device-management .batch-result.partial,
body[theme="dark"] .device-management .batch-result.failed,
body[data-quickstart-theme="dark"] .device-management .feedback.error,
body[data-quickstart-theme="dark"] .device-management .batch-result.partial,
body[data-quickstart-theme="dark"] .device-management .batch-result.failed { color:#ffbd8a!important; background:rgba(255,145,77,.12)!important }
</style>
