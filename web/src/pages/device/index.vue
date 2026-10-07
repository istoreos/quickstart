<template>
    <main id="page" class="device-management">
        <nav class="breadcrumbs" :aria-label="$gettext('面包屑')"><router-link to="/">{{ $gettext('首页') }}</router-link><span aria-hidden="true">›</span><span>{{ $gettext('局域网设备管理') }}</span></nav>
        <header class="page-heading"><div><h1>{{ $gettext('局域网设备管理') }}</h1><p>{{ $gettext('认识每台设备，并用简单的方式安排它如何连接和上网。') }}</p></div></header>
        <nav class="primary-tabs" role="tablist" :aria-label="$gettext('局域网设备管理')">
            <button v-for="tab in tabs" :key="tab.id" type="button" role="tab" :aria-selected="activeTab === tab.id" :class="{ active: activeTab === tab.id }" @click="activeTab = tab.id"><span aria-hidden="true">{{ tab.icon }}</span>{{ tab.label }}</button>
        </nav>
        <section class="primary-content" role="tabpanel">
            <DeviceCenterList v-if="activeTab === 'devices'" />
            <DeviceGroupsPanel v-else-if="activeTab === 'groups'" />
            <LanSettingsPanel v-else />
        </section>
    </main>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import DeviceCenterList from './deviceCenterList.vue'
import DeviceGroupsPanel from './deviceGroupsPanel.vue'
import LanSettingsPanel from './lanSettingsPanel.vue'

const { $gettext } = useGettext()
const activeTab = ref<'devices'|'groups'|'settings'>('devices')
const tabs = computed(() => [
    { id: 'devices' as const, icon: '◉', label: $gettext('设备') },
    { id: 'groups' as const, icon: '◫', label: $gettext('分组与计划') },
    { id: 'settings' as const, icon: '⚙', label: $gettext('局域网设置') },
])
</script>

<style lang="scss" scoped>
.device-management { --device-accent: #553afe; max-width: 1320px; margin: 0 auto; color: var(--flow-span-color); }.breadcrumbs { display: flex; align-items: center; gap: 7px; margin-bottom: 18px; padding-top: 4px; font-size: 13px; }.breadcrumbs a { color: var(--breadcrumbs-tit-color); text-decoration: none; }.breadcrumbs span:last-child { color: var(--breadcrumbs-tit-color1); }.page-heading { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 18px; }.page-heading h1 { margin: 0; padding: 0; color: inherit; background: none!important; font-size: 25px; text-align: left; }.page-heading p { margin: 6px 0 0; opacity: .65; }.primary-tabs { display: flex; gap: 6px; padding: 5px; background: rgba(127,127,127,.07); border-radius: 11px 11px 0 0; }.primary-tabs button { display: inline-flex; align-items: center; gap: 7px; min-height: 42px; padding: 8px 16px; color: inherit; background: transparent; border: 0; border-radius: 8px; cursor: pointer; }.primary-tabs button.active { color: var(--device-accent); background: var(--card-bg-color); box-shadow: 0 2px 8px rgba(30,20,90,.07); font-weight: 600; }.primary-tabs button:focus-visible { outline: 2px solid var(--device-accent); outline-offset: 2px; }.primary-content { min-height: 62vh; padding: 18px; background: var(--card-bg-color); border-radius: 0 0 11px 11px; }
@media(prefers-color-scheme:dark){.device-management{--device-accent:#a697ff}}
@media(max-width:700px){.page-heading h1{font-size:21px}.page-heading p{max-width:34ch}.primary-tabs{overflow-x:auto}.primary-tabs button{flex:1;justify-content:center;min-width:max-content;padding:8px 11px}.primary-content{padding:13px}}
@media(max-width:420px){.page-heading p{font-size:12px}.primary-tabs button span{display:none}.primary-content{padding:11px 8px}}
</style>

<style lang="scss">
.device-management header::before,
.device-management header::after { content: none!important; display: none!important; pointer-events: none!important; }
.device-management > .page-heading,
.device-management .group-header,
.device-management .lan-settings > header { color: var(--tit-color)!important; background: transparent!important; }
@media(max-width:420px){a.btn[href="/cgi-bin/luci/admin/system/admin"]{max-width:140px;white-space:normal;overflow-wrap:anywhere;text-align:center}}
@media(prefers-color-scheme:dark){body:not([theme="light"]){background:#151518}}
body[theme="dark"]{background:#151518}
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
body[theme="dark"] .device-management .batch-result.failed { color:#ffbd8a!important; background:rgba(255,145,77,.12)!important }
</style>
