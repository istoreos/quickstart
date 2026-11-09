<template>
    <section class="lan-settings" :aria-label="$gettext('局域网设置')">
        <button v-if="loading || error" type="button" class="components-fallback" @click="openComponents()" data-open-components><HomeGlyph name="layers"/>{{ $gettext('功能组件') }}</button>
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取局域网设置')" />
        <PageState v-else-if="error" kind="error" :title="$gettext('局域网设置读取失败')" :description="error" :action-label="$gettext('重新加载')" @action="load" />
        <template v-else>
            <article class="context-card" :class="routerContext?.topologyPosition">
                <div><strong>{{ contextTitle }}</strong><span>{{ permissionScope }}</span></div>
                <span class="context-state">{{ $gettext('权限说明') }}</span>
            </article>

            <details v-if="migrationPlan && !migrationPlan.alreadyApplied && migrationPlan.items?.length" class="migration-card" aria-labelledby="migration-title">
                <summary class="migration-heading"><div><h3 id="migration-title">{{ $gettext('发现现有设备设置') }}</h3><p>{{ $gettext('先预览并接管现有设置，之后只使用当前设备管理界面维护。') }}</p></div><strong>{{ migrationPlan.items.length }} {{ $gettext('项') }}</strong></summary>
                <div class="migration-body">
                    <ul><li v-for="item in migrationPlan.items.slice(0, 5)" :key="item.id"><span>{{ migrationKind(item.kind) }}</span><b>{{ migrationSummary(item.summary) }}</b><em :class="item.disposition">{{ migrationDisposition(item.disposition) }}</em></li></ul>
                    <p v-if="migrationPlan.items.length > 5" class="migration-more">{{ $gettext('其余设置会在同一次安全接管中处理。') }}</p>
                    <p v-if="!migrationPlan.canApply" class="migration-warning">{{ $gettext('发现冲突或无法识别的设置，当前不会写入任何配置。请先在规则台账中处理。') }}</p>
                    <div class="migration-actions"><button type="button" @click="section = 'rules'">{{ $gettext('查看规则台账') }}</button><button class="primary" type="button" :disabled="!migrationPlan.canApply || migrating" @click="applyMigration">{{ migrating ? $gettext('正在接管…') : $gettext('确认接管现有设置') }}</button></div>
                </div>
            </details>
            <p v-else-if="migrationError" class="migration-read-error" role="status">{{ $gettext('暂时无法检查现有设置，不影响其他局域网设置。') }}</p>

            <div class="settings-navigation"><nav class="settings-nav" :aria-label="$gettext('设置区域')">
                <button v-for="item in sections" :key="item.id" type="button" :class="{ active: section === item.id }" @click="section = item.id">{{ item.label }}</button>
            </nav><button type="button" class="components-entry" :disabled="Boolean(saving) || speedRefreshing" @click="openComponents()" data-open-components><HomeGlyph name="layers"/>{{ $gettext('功能组件') }}</button></div>

            <div v-if="section === 'services'" class="settings-stack">
                <article class="setting-card dhcp-card">
                    <div class="card-heading"><div><h3><HomeGlyph name="network" />{{ $gettext('地址分配服务') }}</h3><p v-if="dhcpEditable">{{ $gettext('由本机为局域网设备分配地址。关闭后，设备应由其他路由器分配地址。') }}</p></div><label v-if="dhcpEditable" class="switch-line"><input v-model="dhcp.enabled" type="checkbox" :disabled="!dhcpEditable" />{{ dhcp.enabled ? $gettext('已开启') : $gettext('已关闭') }}</label><span v-else class="context-state">{{ $gettext('配置只读') }}</span></div>
                    <template v-if="dhcpState">
                    <dl v-if="!dhcpEditable" class="local-config-summary">
                        <div><dt>{{ $gettext('本机配置') }}</dt><dd>{{ dhcpState.settings.enabled ? $gettext('已开启') : $gettext('已关闭') }}</dd></div>
                        <div><dt>{{ $gettext('地址池') }}</dt><dd class="config-address">{{ dhcpState.settings.poolStart || '—' }} – {{ dhcpState.settings.poolEnd || '—' }}</dd></div>
                        <div><dt>{{ $gettext('地址租期') }}</dt><dd>{{ dhcpState.settings.leaseTime || '—' }}</dd></div>
                    </dl>
                    <p v-if="!dhcpEditable" class="setting-hint config-origin">{{ $gettext('以下为本机保存的配置，不代表其他路由器当前生效的配置。') }}</p>
                    <button v-if="!dhcpEditable" class="config-disclosure" type="button" :aria-expanded="showLocalConfig" aria-controls="lan-local-config" @click="showLocalConfig = !showLocalConfig"><HomeGlyph name="arrow" />{{ $gettext('查看本机配置') }}</button>
                    <fieldset v-if="dhcpEditable || showLocalConfig" id="lan-local-config" :disabled="!dhcpEditable">
                        <div class="two-columns"><label><span>{{ $gettext('地址池起始地址') }}</span><input v-model.trim="dhcp.poolStart" inputmode="decimal" :placeholder="dhcpEditable ? '192.168.1.100' : '—'" /></label><label><span>{{ $gettext('地址池结束地址') }}</span><input v-model.trim="dhcp.poolEnd" inputmode="decimal" :placeholder="dhcpEditable ? '192.168.1.249' : '—'" /></label></div>
                        <label><span>{{ $gettext('地址租期') }}</span><input v-model.trim="dhcp.leaseTime" :placeholder="dhcpEditable ? '12h' : '—'" /></label>
                        <label><span>{{ $gettext('默认上网路线') }}</span><select v-model="dhcp.defaultTargetId"><option v-for="target in gatewayTargets.filter(item => item.supported && item.id !== 'default' && item.gateway)" :key="target.id" :value="target.id">{{ gatewayTargetLabel(target) }}{{ target.gateway ? ` · ${target.gateway}` : '' }}</option></select></label>
                    </fieldset>
                    <button v-if="dhcpEditable" class="primary" type="button" :disabled="!dhcpEditable || saving === 'dhcp-plan'" @click="planDhcp">{{ saving === 'dhcp-plan' ? $gettext('正在检查…') : $gettext('预览地址分配设置') }}</button>
                    </template>
                    <PageState v-else kind="unavailable" :title="$gettext('本机配置暂不可用')" :description="$gettext('未能读取可靠的地址分配配置，其他服务仍可使用。')" />
                    <section v-if="pendingDhcpPlan" class="impact-preview" aria-live="polite">
                        <h4>{{ $gettext('地址分配影响') }}</h4>
                        <p v-if="!dhcp.enabled">{{ $gettext('关闭后预计影响设备') }}：{{ pendingDhcpPlan.affectedDevices }}</p>
                        <p v-if="pendingDhcpPlan.recoveryGuidance">{{ $gettext('恢复路径') }}：{{ $gettext('通过路由器局域网地址重新进入，并重新开启地址分配；或先确认外部 DHCP 已就绪。') }}</p>
                        <ul v-if="pendingDhcpPlan.conflicts.length"><li v-for="conflict in pendingDhcpPlan.conflicts" :key="`${conflict.kind}:${conflict.address}`">{{ conflict.address }} · {{ dhcpConflictLabel(conflict.kind) }}</li></ul>
                        <p v-if="pendingDhcpPlan.error && pendingDhcpPlan.error.code !== 'confirmation_required'" class="field-error">{{ pendingDhcpPlan.error.message }}</p>
                        <div class="preview-actions"><button type="button" @click="cancelDhcpPlan">{{ $gettext('取消') }}</button><button v-if="pendingDhcpPlan.error?.code === 'confirmation_required'" class="danger-link" type="button" @click="confirmDhcpDisable">{{ $gettext('我已了解影响，继续') }}</button><button v-else class="primary" type="button" :disabled="!pendingDhcpPlan.canApply || saving === 'dhcp-apply'" @click="applyDhcp">{{ saving === 'dhcp-apply' ? $gettext('正在应用…') : $gettext('确认应用') }}</button></div>
                    </section>
                </article>

                <article class="setting-card speed-card" :aria-busy="speedRefreshing">
                    <div class="card-heading"><div><h3><HomeGlyph name="activity" />{{ $gettext('设备限速服务') }}</h3><p>{{ $gettext('为设备设置最高上传和下载速度；联网权限不依赖此服务。') }}</p></div><span class="capability-pill" :class="speedStatusFresh ? speedCapability.state : 'error'">{{ speedStatusFresh ? capabilityLabel(speedCapability.state) : $gettext('状态未更新') }}</span></div>
                    <p class="current-speed-provider">{{ $gettext('当前方式') }}：{{ providerLabel(currentSpeedProvider) }}</p>
                    <div class="speed-component-entry"><span>{{ $gettext('智能设备限速（推荐）') }} · {{ nativeSpeedProvider?.available ? $gettext('已就绪') : nativeSpeedProvider ? providerReason(nativeSpeedProvider.reason) : $gettext('状态待确认') }}</span><button type="button" :disabled="Boolean(saving) || speedRefreshing" @click="openComponents('quickstart-native')" data-manage-speed-component>{{ nativeSpeedProvider?.reason === 'dependency_not_installed' ? $gettext('查看并安装') : $gettext('管理组件') }}</button></div>
                    <div v-if="!speedProviderReady || !speedStatusFresh" class="capability-action" role="status"><p>{{ speedRefreshError || speedProviderGuidance }}</p></div>
                    <template v-if="speedProviderReady">
                        <template v-if="speed.provider === 'eqos'">
                            <label class="switch-line"><input v-model="speed.enabled" type="checkbox" />{{ enableSpeedLabel }}</label>
                            <div v-if="speed.enabled" class="two-columns"><label><span>{{ $gettext('总上传带宽（Mbit/s）') }}</span><input v-model.number="speed.upload" type="number" min="1" /></label><label><span>{{ $gettext('总下载带宽（Mbit/s）') }}</span><input v-model.number="speed.download" type="number" min="1" /></label></div>
                            <p class="setting-hint">{{ $gettext('总带宽用于计算设备限速队列，不会自动平均分配给每台设备。') }}</p>
                        </template>
                        <p v-else-if="speed.provider === 'quickstart-native' && speedStatusFresh" class="service-ready">{{ $gettext('服务已就绪，请在设备详情中为需要的设备设置速度上限。') }}</p>
                        <p v-else-if="speed.provider === 'bandix'" class="setting-hint">{{ $gettext('现有 Bandix 仅用于迁移，请先完成下方步骤。') }}</p>
                        <button v-if="speed.provider === 'eqos' || speed.provider !== currentSpeedProvider" class="primary" type="button" :disabled="saving === 'speed-plan' || !speedProviderReady || speedRefreshing || !speedStatusFresh" @click="planSpeed">{{ saving === 'speed-plan' ? $gettext('正在检查…') : $gettext('预览限速服务设置') }}</button>
                    </template>
                        <section v-if="pendingSpeedPlan" class="impact-preview" aria-live="polite">
                            <h4>{{ $gettext('限速服务影响') }}</h4>
                            <p>{{ pendingSpeedPlan.changes?.length ? $gettext('将更新限速执行方式，现有规则保持不变。') : $gettext('设置没有变化。') }}</p>
                            <p v-if="pendingSpeedPlan.error" class="field-error">{{ pendingSpeedPlan.error.message }}</p>
                            <div class="preview-actions"><button type="button" @click="pendingSpeedPlan=undefined">{{ $gettext('取消') }}</button><button class="primary" type="button" :disabled="!pendingSpeedPlan.canApply || saving === 'speed-apply' || speedRefreshing || !speedStatusFresh || !speedProviderReady" @click="applySpeed">{{ saving === 'speed-apply' ? $gettext('正在应用…') : $gettext('确认应用') }}</button></div>
                        </section>
                    <div class="speed-check-actions"><button type="button" class="speed-recheck" :disabled="speedRefreshing || Boolean(saving) || installing || rateMigrating" @click="refreshSpeed">{{ speedRefreshing ? $gettext('正在检查…') : $gettext('重新检查') }}</button><button v-if="!speedProviderReady || !speedStatusFresh" type="button" class="speed-inspect" @click="showSpeedAdvanced">{{ $gettext('查看执行方式') }}</button></div>
                    <details class="advanced-group" :open="speedAdvancedOpen" @toggle="onSpeedAdvancedToggle"><summary>{{ $gettext('高级：执行方式与迁移') }}</summary>
                        <label><span>{{ $gettext('执行方式') }}</span><select ref="speedProviderControl" v-model="speed.provider" @change="pendingSpeedPlan=undefined"><option v-for="provider in selectableSpeedProviders" :key="provider.id" :value="provider.id" :disabled="!provider.available">{{ providerLabel(provider.id) }}{{ provider.available ? '' : ` · ${providerReason(provider.reason)}` }}</option></select><small>{{ providerDescription }}</small></label>
                        <section v-if="bandixProvider?.installed && speed.provider === 'bandix'" class="legacy-migration" aria-labelledby="rate-migration-title">
                            <h4 id="rate-migration-title">{{ $gettext('迁移现有 Bandix 规则') }}</h4>
                            <p>{{ $gettext('先扫描并逐条检查；不会自动停止、卸载或改写 Bandix。') }}</p>
                            <button type="button" :disabled="rateMigrating" @click="scanRateMigration">{{ rateMigrating ? $gettext('正在扫描…') : $gettext('扫描现有规则') }}</button>
                            <div v-if="rateMigrationPlan" class="migration-review" aria-live="polite">
                                <ul><li v-for="item in rateMigrationPlan.items" :key="`${item.id}:${item.mac}`"><span>{{ item.mac }}</span><b>{{ migrationRateSummary(item) }}</b><em :class="item.disposition">{{ rateMigrationDisposition(item) }}</em></li></ul>
                                <p v-if="!rateMigrationPlan.canApply" class="field-error">{{ $gettext('存在不支持或冲突的规则，全部处理完成前不会迁移。') }}</p>
                                <label v-else class="checkbox-line"><input v-model="rateMigrationReady" type="checkbox" />{{ $gettext('我已手动停止 Bandix，并启动 Quickstart 限速服务') }}</label>
                                <div class="preview-actions"><button class="primary" type="button" :disabled="!rateMigrationPlan.canApply || !rateMigrationReady || rateMigrating" @click="applyRateMigration">{{ $gettext('确认迁移并验证') }}</button></div>
                            </div>
                            <button v-if="rateMigrationID" class="danger-link" type="button" :disabled="rateMigrating" @click="rollbackRateMigration">{{ $gettext('恢复迁移前设置') }}</button>
                        </section>
                    </details>
                </article>
            </div>

            <div v-else-if="section === 'routes'" class="settings-stack">
                <article class="setting-card">
                    <div class="card-heading"><div><h3>{{ $gettext('自定义上网路线') }}</h3><p>{{ $gettext('把复杂网络工作交给旁路由或指定网关，设备可在详情中选择路线。DNS 跟随路线。') }}</p></div><button type="button" :disabled="!routeEditable" @click="startCreateRoute">{{ routeEditor ? $gettext('取消') : $gettext('添加路线') }}</button></div>
                    <p v-if="!routeEditable" class="state-message">{{ contextDescription }}</p>
                    <form v-if="routeEditor" class="route-form" @submit.prevent="planRouteSave">
                        <label><span>{{ $gettext('路线名称') }}</span><input v-model.trim="routeDraft.name" maxlength="64" required :placeholder="$gettext('例如：旁路由')" /></label>
                        <label><span>{{ $gettext('路线类型') }}</span><select v-model="routeDraft.kind"><option value="bypass">{{ $gettext('旁路由') }}</option><option value="custom">{{ $gettext('指定网关') }}</option></select></label>
                        <label><span>{{ $gettext('网关地址') }}</span><input v-model.trim="routeDraft.gateway" inputmode="decimal" required placeholder="192.168.1.2" /></label>
                        <button class="primary" type="submit" :disabled="saving === 'route-plan'">{{ saving === 'route-plan' ? $gettext('正在检查…') : routeDraft.targetId ? $gettext('预览修改') : $gettext('预览添加') }}</button>
                    </form>
                    <div v-if="routes.length" class="route-list">
                        <div v-for="route in routes" :key="route.id"><span><strong>{{ gatewayTargetLabel(route) }}</strong><small>{{ route.gateway || '—' }} · {{ routeKindLabel(route.kind) }}</small></span><div class="route-actions"><button type="button" :disabled="!routeEditable" @click="startEditRoute(route)">{{ $gettext('编辑') }}</button><button type="button" class="danger-link" :disabled="!routeEditable" @click="planRouteDelete(route)">{{ $gettext('删除') }}</button></div></div>
                    </div>
                    <PageState v-else kind="empty" :title="$gettext('还没有自定义上网路线')" :description="$gettext('设备默认通过本机上网；需要时再添加旁路由或指定网关。')" />
                    <section v-if="pendingRoutePlan" class="impact-preview" aria-live="polite">
                        <h4>{{ $gettext('影响范围') }}</h4>
                        <p v-if="pendingRoutePlan.action === 'delete'">{{ $gettext('受影响设备') }}：{{ pendingRoutePlan.referenceSummary.devices }} · {{ $gettext('受影响分组') }}：{{ pendingRoutePlan.referenceSummary.groups }}</p>
                        <p v-else>{{ pendingRoutePlan.action === 'create' ? $gettext('将添加一条可供设备和分组选择的路线。') : $gettext('将更新路线名称和网关，已有引用会继续保留。') }}</p>
                        <p v-if="pendingRoutePlan.referenceSummary.globalPolicy || pendingRoutePlan.referenceSummary.lanDefault">{{ $gettext('同时影响全局或局域网默认路线。') }}</p>
                        <label v-if="replacementRequired"><span>{{ $gettext('替代路线') }}</span><select v-model="replacementTargetId" @change="replanRouteDelete"><option value="">{{ $gettext('请选择') }}</option><option v-for="target in replacementTargets" :key="target.id" :value="target.id">{{ gatewayTargetLabel(target) }}{{ target.gateway ? ` · ${target.gateway}` : '' }}</option></select></label>
                        <p v-if="pendingRoutePlan.error && !replacementRequired" class="field-error">{{ pendingRoutePlan.error.message }}</p>
                        <div class="preview-actions"><button type="button" @click="cancelRoutePlan">{{ $gettext('取消') }}</button><button class="primary" type="button" :disabled="!pendingRoutePlan.canApply || saving === 'route-apply'" @click="applyRoutePlan">{{ saving === 'route-apply' ? $gettext('正在应用…') : $gettext('确认应用') }}</button></div>
                    </section>
                </article>
                <article class="setting-card">
                    <div class="card-heading"><div><h3>{{ $gettext('浮动网关') }}</h3><p>{{ $gettext('主路由与旁路由共享一个稳定入口，旁路由异常时自动切回主路由。') }}</p></div><span class="capability-pill" :class="floatCapability.state">{{ capabilityLabel(floatCapability.state) }}</span></div>
                    <button type="button" class="floating-component-entry" :disabled="Boolean(saving)" @click="openComponents('floatip')" data-manage-float-component>{{ floatCapability.state === 'not_installed' ? $gettext('查看并安装') : $gettext('管理组件') }}</button>
                    <div v-if="floatCapability.state === 'not_installed'" class="capability-action"><p>{{ $gettext('只有浮动网关功能会被锁定，设备和其他路线仍可管理。') }}</p></div>
                    <FloatingGatewayWizard v-else />
                </article>
            </div>

            <NetworkRulesHub v-else />
            <p v-if="feedback" class="feedback" :class="feedbackKind" role="status">{{ feedback }}</p>
        </template>
        <FeatureComponentsDrawer :open="componentsOpen" :focus-id="componentFocus" @close="componentsOpen=false" @busy="installing=$event" @updated="refreshSpeed" @configure="configureComponent"/>
    </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import baseRequest from '/@/request'
import { useLANClient } from './lanContext'
const request = { ...baseRequest, DeviceMangement: useLANClient() }
import PageState from './components/pageState.vue'
import FloatingGatewayWizard from './components/floatingGatewayWizard.vue'
import NetworkRulesHub from './networkRulesHub.vue'
import { resolveCapability, type DeviceCapability } from './deviceCapabilities'
import type { GatewayTarget, GatewayTargetMutationPlan, GatewayTargetMutationRequest } from './gatewayTargets'
import type { LanDhcpSettingsResult } from './lanDhcpSettings'
import { readDhcpSnapshot, readSpeedSnapshot } from './lanSettingsStatus'
import HomeGlyph from '/@/home/HomeGlyph.vue'
import FeatureComponentsDrawer from './components/featureComponentsDrawer.vue'
import type { ComponentId } from './featureComponents'

const { $gettext } = useGettext()
const enableSpeedLabel = $gettext('启用设备限速')
const draftKey = 'quickstart.lan-settings-draft.v1'
const restored = (() => { try { return JSON.parse(sessionStorage.getItem(draftKey) || '{}') } catch { return {} } })()
const loading = ref(true), error = ref(''), saving = ref(''), installing = ref(false), migrating = ref(false), feedback = ref(''), feedbackKind = ref<'success'|'error'>('success')
const componentsOpen=ref(false),componentFocus=ref<ComponentId>()
const openComponents=(id?:ComponentId)=>{componentFocus.value=id;componentsOpen.value=true}
const section = ref<'services'|'routes'|'rules'>(['services','routes','rules'].includes(restored.section) ? restored.section : 'services')
const routerContext = ref<any>(), globalData = ref<any>({}), speedCapability = ref<DeviceCapability>({ state: 'error' }), floatCapability = ref<DeviceCapability>({ state: 'error' })
const migrationPlan = ref<any>(), migrationError = ref(false)
const rateMigrationPlan = ref<any>(), rateMigrationReady = ref(false), rateMigrationID = ref(''), rateMigrating = ref(false)
const dhcp = reactive({ enabled: true, poolStart: '', poolEnd: '', leaseTime: '12h', defaultTargetId: 'self' }), speed = reactive({ enabled: false, upload: 100, download: 1000, provider: '' })
const dhcpState = ref<LanDhcpSettingsResult>()
const pendingDhcpPlan = ref<LanDhcpSettingsResult>()
const pendingSpeedPlan = ref<any>()
const showLocalConfig = ref(false), speedAdvancedOpen = ref(false)
const speedProviderControl = ref<HTMLSelectElement>()
const speedRefreshing = ref(false), speedRefreshError = ref(''), speedStatusFresh = ref(true)
let speedCheck: AbortController | undefined
let speedCheckGeneration = 0
let speedCheckTimer: number | undefined
const cancelSpeedCheck = () => {
    speedCheckGeneration++; speedCheck?.abort(); window.clearTimeout(speedCheckTimer)
    speedCheck = undefined; speedCheckTimer = undefined; speedRefreshing.value = false
}
onBeforeUnmount(cancelSpeedCheck)
const routeEditor = ref(Boolean(restored.routeEditor))
const routeDraft = reactive<{ targetId: string; name: string; kind: 'bypass'|'custom'; gateway: string }>({ targetId: '', name: restored.routeName || '', kind: restored.routeKind === 'custom' ? 'custom' : 'bypass', gateway: restored.routeGateway || '' })
const gatewayTargets = ref<GatewayTarget[]>([])
const pendingRoutePlan = ref<GatewayTargetMutationPlan>()
const pendingRouteRequest = ref<GatewayTargetMutationRequest>()
const replacementTargetId = ref('')
const sections = computed(() => [{ id: 'services' as const, label: $gettext('网络服务') }, { id: 'routes' as const, label: $gettext('上网路线') }, { id: 'rules' as const, label: $gettext('规则台账') }])
const dhcpEditable = computed(() => Boolean(dhcpState.value?.editable && dhcpState.value.settings))
const routes = computed(() => gatewayTargets.value.filter(target => target.kind === 'bypass' || target.kind === 'custom'))
const speedProviders = computed(() => Array.isArray(globalData.value?.speedLimit?.providers) ? globalData.value.speedLimit.providers : [])
const selectableSpeedProviders = computed(() => speedProviders.value.filter((provider:any) => provider.id !== 'bandix' || speed.provider === 'bandix'))
const selectedSpeedProvider = computed(() => speedProviders.value.find((provider:any) => provider.id === speed.provider))
const bandixProvider = computed(() => speedProviders.value.find((provider:any) => provider.id === 'bandix'))
const currentSpeedProvider = computed(() => globalData.value?.speedLimit?.provider || '')
const speedProviderReady = computed(() => Boolean(selectedSpeedProvider.value?.available))
const nativeSpeedProvider = computed(() => speedProviders.value.find((provider:any) => provider.id === 'quickstart-native'))
const providerLabel = (id:string) => ({'quickstart-native':$gettext('智能设备限速（推荐）'),bandix:$gettext('现有 Bandix（仅用于迁移）'),eqos:$gettext('兼容模式（按 IPv4）')} as Record<string,string>)[id] || id
const providerReason = (reason:string) => ({dependency_not_installed:$gettext('未安装'),kernel_not_supported:$gettext('系统内核不支持'),engine_unavailable:$gettext('服务未运行'),interface_not_supported:$gettext('版本不兼容')} as Record<string,string>)[reason] || $gettext('当前不可用')
const speedProviderGuidance = computed(() => speedCapability.value.reason === 'selected_provider_unavailable' ? $gettext('已安装限速服务，但当前执行方式不可用。') : speed.provider === 'quickstart-native' ? $gettext('Quickstart 限速服务未就绪。请安装或启动服务后重新检查；系统不会自动改用其他方式。') : $gettext('当前执行方式不可用，请先安装或启用对应服务。'))
const providerDescription = computed(() => speed.provider === 'quickstart-native' ? $gettext('按设备身份同时覆盖 IPv4 与 IPv6，并验证规则是否真正生效。') : speed.provider === 'bandix' ? $gettext('仅用于查看和迁移现有规则，不再作为新规则的默认执行方式。') : $gettext('资源占用较低，按固定 IPv4 执行。'))
const routeEditable = computed(() => Boolean(routerContext.value?.routeEditability?.editable))
const replacementRequired = computed(() => pendingRoutePlan.value?.error?.code === 'replacement_required')
const replacementTargets = computed(() => gatewayTargets.value.filter(target => target.supported && target.id !== pendingRoutePlan.value?.target?.id))
const contextTitle = computed(() => ({ local: $gettext('本机负责局域网地址分配'), external_observed: $gettext('检测到其他设备负责地址分配'), none_detected: $gettext('尚未检测到地址分配服务'), ambiguous: $gettext('局域网角色需要确认'), error: $gettext('暂时无法判断局域网角色') } as Record<string,string>)[routerContext.value?.dhcpAuthority] || $gettext('局域网角色待确认'))
const contextDescription = computed(() => {
    if (routerContext.value?.routeEditability?.editable) return $gettext('可在设备详情中设置上网路线。')
    return ({
        local_dhcp_unhealthy: $gettext('当前无法读取地址分配状态，其他设置仍可使用。'),
        multiple_dhcp_evidence: $gettext('局域网角色需要确认'),
        external_dhcp_authority: $gettext('检测到其他设备负责地址分配，请在负责地址分配的路由器上管理相关设置。'),
        multiple_external_dhcp_servers: $gettext('局域网角色需要确认'),
        dhcp_authority_not_detected: $gettext('当前仅提供查看与引导，不会自动修改其他设备。'),
        router_context_unavailable: $gettext('暂时无法判断局域网角色'),
    } as Record<string,string>)[routerContext.value?.routeEditability?.reason] || $gettext('当前仅提供查看与引导，不会自动修改其他设备。')
})
const permissionScope = computed(() => [
    dhcpEditable.value ? $gettext('地址配置可编辑') : $gettext('地址配置只读'),
    routeEditable.value ? $gettext('路线可编辑') : $gettext('路线只读'),
    $gettext('其他服务按各自权限管理'),
].join(' · '))
const routeKindLabel = (value: string) => ({ bypass: $gettext('旁路由'), custom: $gettext('指定网关') } as Record<string,string>)[value] || $gettext('自定义路线')
const dhcpConflictLabel = (value:string) => ({router:$gettext('路由器地址'),floating_gateway:$gettext('浮动网关'),gateway_node:$gettext('网关节点'),address_reservation:$gettext('固定地址')} as Record<string,string>)[value] || $gettext('受保护地址')
const capabilityLabel = (state: string) => ({ available: $gettext('可用'), disabled: $gettext('未启用'), not_installed: $gettext('未安装'), unsupported: $gettext('不支持'), error: $gettext('需要检查') } as Record<string,string>)[state] || $gettext('未知')
const migrationKind = (kind:string) => ({ alias:$gettext('设备资料'), hostname:$gettext('局域网主机名'), static:$gettext('地址预留'), route:$gettext('上网路线'), speed:$gettext('设备限速'), access:$gettext('联网权限'), floating_gateway:$gettext('浮动网关') } as Record<string,string>)[kind] || $gettext('设备规则')
const migrationDisposition = (value:string) => ({ adopt:$gettext('可以接管'), normalize:$gettext('将安全整理'), conflict:$gettext('存在冲突'), unresolved:$gettext('需要确认') } as Record<string,string>)[value] || $gettext('需要确认')
const gatewayTargetLabel = (target: GatewayTarget) => {
    if (target.kind === 'default' || target.name === '跟随网络默认') return $gettext('默认路线')
    if (target.kind === 'self') return $gettext('本机路由')
    if (target.kind === 'bypass' && target.name === '旁路由') return $gettext('旁路由')
    if (target.kind === 'floating' && target.name === '浮动网关') return $gettext('浮动网关')
    if (target.name === '自定义网关') return $gettext('自定义网关')
    return target.name
}
const migrationSummary = (value:string) => {
    if (!value) return '—'
    for (const [source, label] of [['本机路由', $gettext('本机路由')], ['旁路由', $gettext('旁路由')], ['浮动网关', $gettext('浮动网关')], ['自定义网关', $gettext('自定义网关')]]) {
        if (value === source || value.startsWith(`${source} ·`)) return value.replace(source, label)
    }
    return value
}
const rememberDraft = () => sessionStorage.setItem(draftKey, JSON.stringify({ section: section.value, routeEditor: routeEditor.value, routeName: routeDraft.name, routeKind: routeDraft.kind, routeGateway: routeDraft.gateway }))
watch([section, routeEditor, () => routeDraft.name, () => routeDraft.kind, () => routeDraft.gateway], rememberDraft)

const load = async () => {
    cancelSpeedCheck()
    loading.value = true; error.value = ''
    try {
        const [contextResponse, globalResponse, targetResponse, dhcpResponse, migrationResponse] = await Promise.all([request.DeviceMangement.routerContextV2.GET(), request.DeviceMangement.globalConfigs.GET().catch(() => undefined), request.DeviceMangement.gatewayTargetsV2.GET(), request.DeviceMangement.lanDhcpSettingsV2.GET().catch(() => undefined), request.DeviceMangement.lanDeviceMigrationV2.PLAN().catch(() => undefined)])
        routerContext.value = contextResponse.data?.result
        globalData.value = globalResponse?.data?.result || {}
        gatewayTargets.value = targetResponse.data?.result?.targets || []
        dhcpState.value = readDhcpSnapshot(dhcpResponse?.data)
        migrationPlan.value = migrationResponse?.data?.result?.plan
        migrationError.value = !migrationResponse
        speedCapability.value = resolveCapability(globalData.value, 'speedLimit')
        floatCapability.value = resolveCapability(globalData.value, 'floatGateway')
        if (dhcpState.value?.settings) Object.assign(dhcp, dhcpState.value.settings)
        speed.enabled = Boolean(globalData.value?.speedLimit?.enabled)
        speed.upload = globalData.value?.speedLimit?.uploadSpeed || 100
        speed.download = globalData.value?.speedLimit?.downloadSpeed || 1000
        speed.provider = globalData.value?.speedLimit?.provider || speedProviders.value.find((item:any)=>item.available)?.id || ''
        try { readSpeedSnapshot(globalResponse?.data); speedStatusFresh.value = true; speedRefreshError.value = '' }
        catch { speedStatusFresh.value = false; speedRefreshError.value = $gettext('检查失败，状态未更新。请重试，草稿已保留。') }
    } catch (reason: any) { error.value = reason?.message || $gettext('读取结果失败') }
    finally { loading.value = false }
}
const onSpeedAdvancedToggle = (event: Event) => { speedAdvancedOpen.value = (event.target as HTMLDetailsElement).open }
const showSpeedAdvanced = async () => {
    speedAdvancedOpen.value = true
    await nextTick()
    speedProviderControl.value?.focus({ preventScroll: true })
    speedProviderControl.value?.scrollIntoView({ block: 'nearest' })
}
const refreshSpeed = async () => {
    if (speedRefreshing.value || saving.value || installing.value || rateMigrating.value) return
    const generation = ++speedCheckGeneration
    const controller = new AbortController()
    speedCheck = controller
    speedRefreshing.value = true; speedRefreshError.value = ''
    let timedOut = false
    const timer = window.setTimeout(() => { timedOut = true; controller.abort() }, 10000)
    speedCheckTimer = timer
    try {
        const response = await request.DeviceMangement.globalConfigs.GET(controller.signal)
        if (generation !== speedCheckGeneration) return
        if (controller.signal.aborted) throw new Error('aborted_service_check')
        const snapshot = readSpeedSnapshot(response.data)
        // Observations only. Draft values and pending plans retain their owners.
        globalData.value = { ...globalData.value, ...response.data.result, speedLimit: snapshot.speedLimit }
        floatCapability.value = resolveCapability(globalData.value, 'floatGateway')
        speedCapability.value = snapshot.capability as DeviceCapability
        speedStatusFresh.value = true
    } catch {
        if (generation !== speedCheckGeneration) return
        speedStatusFresh.value = false
        speedRefreshError.value = timedOut ? $gettext('检查超时，请重试。草稿已保留。') : $gettext('检查失败，状态未更新。请重试，草稿已保留。')
    } finally {
        window.clearTimeout(timer)
        if (generation === speedCheckGeneration) { speedRefreshing.value = false; speedCheck = undefined; speedCheckTimer = undefined }
    }
}
const cancelDhcpPlan=()=>{pendingDhcpPlan.value=undefined}
const requestDhcpPlan=async(confirmDisable=false)=>{saving.value='dhcp-plan';feedback.value='';try{const result=(await request.DeviceMangement.lanDhcpSettingsV2.PLAN({settings:{...dhcp},confirmDisable,idempotencyKey:`lan-dhcp-${Date.now().toString(36)}`})).data?.result as LanDhcpSettingsResult;pendingDhcpPlan.value=result;if(result?.error&&result.error.code!=='confirmation_required'){feedbackKind.value='error';feedback.value=result.error.message}}catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('无法预览地址分配设置')}finally{saving.value=''}}
const planDhcp=()=>requestDhcpPlan(false)
const confirmDhcpDisable=()=>requestDhcpPlan(true)
const applyDhcp=async()=>{if(!pendingDhcpPlan.value?.canApply)return;saving.value='dhcp-apply';feedback.value='';try{const result=(await request.DeviceMangement.lanDhcpSettingsV2.APPLY({settings:{...dhcp},confirmDisable:!dhcp.enabled,expectedVersion:pendingDhcpPlan.value.version,idempotencyKey:`lan-dhcp-${pendingDhcpPlan.value.version}`})).data?.result;if(result?.error)throw new Error(result.error.message);feedbackKind.value='success';feedback.value=result.changed?$gettext('地址分配设置已保存'):$gettext('设置没有变化');cancelDhcpPlan();await load()}catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('应用失败，原设置已恢复')}finally{saving.value=''}}
const speedSettingsPayload = () => ({ enabled:speed.provider === 'eqos' ? speed.enabled : true, uploadSpeed:speed.upload, downloadSpeed:speed.download, provider:speed.provider })
const planSpeed = async () => {
    if (speed.enabled && (!speed.upload || !speed.download)) return
    saving.value='speed-plan';feedback.value=''
    try {
        const result=(await request.DeviceMangement.rateLimitSettingsV2.PLAN({settings:speedSettingsPayload(),idempotencyKey:`rate-limit-${Date.now().toString(36)}`})).data?.result
        pendingSpeedPlan.value=result
        if(result?.error){feedbackKind.value='error';feedback.value=result.error.message}
    } catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('无法预览限速服务设置')} finally{saving.value=''}
}
const applySpeed = async () => {
    if(!pendingSpeedPlan.value?.canApply)return
    saving.value='speed-apply';feedback.value=''
    try {
        const result=(await request.DeviceMangement.rateLimitSettingsV2.APPLY({settings:speedSettingsPayload(),expectedVersion:pendingSpeedPlan.value.version,idempotencyKey:`rate-limit-${pendingSpeedPlan.value.version}`})).data?.result
        if(result?.error)throw Object.assign(new Error(result.error.message),{code:result.error.code})
        feedbackKind.value='success';feedback.value=result?.changed?$gettext('限速服务设置已应用'):$gettext('设置没有变化');pendingSpeedPlan.value=undefined;await load()
    } catch(reason:any){feedbackKind.value='error';feedback.value=reason?.code==='rolled_back'?$gettext('应用失败，原设置已恢复'):reason?.message||$gettext('应用失败，原设置已恢复')} finally{saving.value=''}
}
const migrationRateSummary = (item:any) => `${Math.round((item.uploadBitsPerSecond || 0) / 1000000)} ↑ · ${Math.round((item.downloadBitsPerSecond || 0) / 1000000)} ↓ Mbit/s`
const rateMigrationDisposition = (item:any) => ({convert:$gettext('可以迁移'),unsupported:$gettext('不支持'),conflict:$gettext('存在冲突')} as Record<string,string>)[item.disposition] || $gettext('需要确认')
const scanRateMigration = async () => { rateMigrating.value=true;feedback.value='';rateMigrationReady.value=false;try{const result=(await request.DeviceMangement.rateLimitMigrationV2.PLAN()).data?.result;if(result?.error)throw new Error(result.error.message);rateMigrationPlan.value=result?.plan}catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('无法扫描现有规则')}finally{rateMigrating.value=false} }
const applyRateMigration = async () => { if(!rateMigrationPlan.value?.canApply||!rateMigrationReady.value)return;rateMigrating.value=true;feedback.value='';try{const result=(await request.DeviceMangement.rateLimitMigrationV2.APPLY({expectedVersion:rateMigrationPlan.value.version,plan:rateMigrationPlan.value})).data?.result;if(result?.error)throw Object.assign(new Error(result.error.message),{code:result.error.code});rateMigrationID.value=result?.migrationId||'';feedbackKind.value='success';feedback.value=$gettext('规则已写入并验证，执行方式已切换');await load()}catch(reason:any){feedbackKind.value='error';feedback.value=reason?.code==='target_unavailable'?$gettext('请先手动停止 Bandix，并启动 Quickstart 限速服务后重试。'):reason?.message||$gettext('迁移失败，原设置已保留')}finally{rateMigrating.value=false} }
const rollbackRateMigration = async () => { if(!rateMigrationID.value)return;rateMigrating.value=true;feedback.value='';try{const result=(await request.DeviceMangement.rateLimitMigrationV2.ROLLBACK({migrationId:rateMigrationID.value})).data?.result;if(result?.error)throw new Error(result.error.message);feedbackKind.value='success';feedback.value=$gettext('已恢复迁移前设置');rateMigrationID.value='';rateMigrationPlan.value=undefined;await load()}catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('恢复失败，请按提示处理')}finally{rateMigrating.value=false} }
const resetRouteEditor = () => { routeEditor.value=false;routeDraft.targetId='';routeDraft.name='';routeDraft.kind='bypass';routeDraft.gateway='' }
const cancelRoutePlan = () => { pendingRoutePlan.value=undefined;pendingRouteRequest.value=undefined;replacementTargetId.value='' }
const startCreateRoute = () => { cancelRoutePlan();if(routeEditor.value){resetRouteEditor();return}routeEditor.value=true;routeDraft.targetId='';routeDraft.name='';routeDraft.kind='bypass';routeDraft.gateway='' }
const startEditRoute = (route:GatewayTarget) => { cancelRoutePlan();routeEditor.value=true;routeDraft.targetId=route.id;routeDraft.name=route.name;routeDraft.kind=route.kind==='custom'?'custom':'bypass';routeDraft.gateway=route.gateway||'' }
const planRoute = async (payload:GatewayTargetMutationRequest) => {
    saving.value='route-plan';feedback.value=''
    try {
        const plan=(await request.DeviceMangement.gatewayTargetV2.PLAN(payload)).data?.result as GatewayTargetMutationPlan
        pendingRoutePlan.value=plan;pendingRouteRequest.value=payload
        if(plan?.error?.code!=='replacement_required' && plan?.error) feedback.value=plan.error.message
    } catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('无法预览这次修改')} finally{saving.value=''}
}
const planRouteSave = async () => planRoute({action:routeDraft.targetId?'update':'create',targetId:routeDraft.targetId||undefined,name:routeDraft.name,kind:routeDraft.kind,gateway:routeDraft.gateway,idempotencyKey:`route-${Date.now().toString(36)}`})
const planRouteDelete = async (route:GatewayTarget) => { resetRouteEditor();replacementTargetId.value='';await planRoute({action:'delete',targetId:route.id,idempotencyKey:`route-delete-${Date.now().toString(36)}`}) }
const replanRouteDelete = async () => { if(!pendingRouteRequest.value||!replacementTargetId.value)return;await planRoute({...pendingRouteRequest.value,replacementTargetId:replacementTargetId.value}) }
const applyRoutePlan = async () => {
    if(!pendingRoutePlan.value?.canApply||!pendingRouteRequest.value)return
    saving.value='route-apply';feedback.value=''
    try {
        const result=(await request.DeviceMangement.gatewayTargetV2.APPLY({...pendingRouteRequest.value,expectedVersion:pendingRoutePlan.value.version})).data?.result
        if(result?.error)throw new Error(result.error.message)
        feedbackKind.value='success';feedback.value=result?.changed?$gettext('上网路线已更新'):$gettext('上网路线没有变化')
        cancelRoutePlan();resetRouteEditor();await load()
    }catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('应用失败，原设置已保留')}finally{saving.value=''}
}
const applyMigration = async () => {
    if (!migrationPlan.value?.canApply || !window.confirm($gettext('确认接管现有设置？接管前会创建快照，失败时自动恢复。'))) return
    migrating.value = true; feedback.value = ''
    try {
        const result = (await request.DeviceMangement.lanDeviceMigrationV2.APPLY({ expectedVersion: migrationPlan.value.version })).data?.result
        if (result?.error) throw Object.assign(new Error(result.error.message), { code: result.error.code })
        feedbackKind.value = 'success'; feedback.value = result?.changed ? $gettext('现有设置已安全接管') : $gettext('现有设置无需重复接管')
        await load()
    } catch (reason:any) {
        feedbackKind.value='error'
        feedback.value=reason?.code==='rolled_back'?$gettext('接管失败，原设置已恢复，可以重试'):reason?.code==='recovery_required'?$gettext('自动恢复未完成，请按提示处理'):reason?.message||$gettext('接管失败，原设置已保留')
    }
    finally { migrating.value=false }
}
const configureComponent=async(id:ComponentId)=>{
    if(id==='floatip'){section.value='routes';return}
    section.value='services';await refreshSpeed()
    const provider=speedProviders.value.find((item:any)=>item.id===id)
    if(!provider?.available||!speedStatusFresh.value)return
    if(id!==currentSpeedProvider.value){speed.provider=id;await planSpeed()}
    await showSpeedAdvanced()
}
load()
</script>

<style lang="scss" scoped>
.settings-navigation{display:flex;align-items:center;gap:12px;margin-bottom:13px}.settings-navigation .settings-nav{flex:1;min-width:0;margin:0}.components-entry,.components-fallback{display:inline-flex;align-items:center;justify-content:center;gap:7px;flex:none;min-height:44px;padding:8px 12px;background:transparent;border:1px solid var(--device-line);border-radius:7px;color:var(--device-accent);font:inherit;cursor:pointer}.components-entry svg,.components-fallback svg{width:18px;height:18px}.components-fallback{margin-bottom:12px}.current-speed-provider{margin:0 0 8px;font-size:12px;color:var(--device-muted)}.speed-component-entry{display:flex;align-items:center;justify-content:space-between;gap:10px;margin:0 0 14px;padding-bottom:12px;border-bottom:1px solid var(--device-line)}.speed-component-entry>span{font-size:12px;color:var(--device-muted);line-height:1.6}.speed-component-entry>button{flex:none}.floating-component-entry{margin-bottom:10px}@media(max-width:700px){.settings-navigation{flex-wrap:wrap;gap:8px}.settings-navigation .settings-nav{order:1;flex-basis:100%}.components-entry{margin-left:auto}.speed-component-entry{align-items:flex-start;flex-wrap:wrap}}
.lan-settings fieldset{min-width:0;margin:0;padding:0;border:0}
.setting-card h3{display:flex;align-items:center;gap:8px}.setting-card h3>svg{width:18px;height:18px;flex:none;color:var(--device-accent)}
.settings-nav button{min-height:44px}.context-card strong{font-size:14px}.context-state{white-space:nowrap}
.local-config-summary{display:grid;gap:12px;margin:0}.local-config-summary>div{display:grid;grid-template-columns:90px minmax(0,1fr);gap:12px;align-items:baseline}
.local-config-summary dt{font-size:12px;color:var(--device-muted)}.local-config-summary dd{margin:0;font-size:14px;overflow-wrap:anywhere}.config-address{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:13px!important}
.config-origin{margin:14px 0 4px}.setting-card .config-disclosure{display:inline-flex;align-items:center;gap:6px;padding:6px 0;border:0;color:var(--device-accent)}.config-disclosure svg{width:14px;height:14px;transform:rotate(0deg)}.config-disclosure[aria-expanded=true] svg{transform:rotate(90deg)}
.speed-check-actions{display:flex;flex-wrap:wrap;gap:8px;margin-top:12px}.setting-card .speed-inspect{color:var(--device-accent)}
.switch-line{min-height:44px;cursor:pointer}
.advanced-group>summary{min-height:44px;display:flex;align-items:center;gap:6px;list-style:none}.advanced-group>summary::-webkit-details-marker{display:none}.advanced-group>summary::before{content:'▸';font-size:12px}.advanced-group[open]>summary::before{transform:rotate(90deg)}
.lan-settings :deep(.page-state){color:var(--device-text);background:var(--device-soft);border-color:var(--device-line)}.lan-settings :deep(.page-state__icon){color:var(--device-accent)}.lan-settings :deep(.page-state__body span){color:var(--device-muted);opacity:1}
.lan-settings { color: var(--device-text); }.setting-card h3,.migration-card h3 { margin:0;padding:0;color:inherit;background:none!important;text-align:left }.card-heading p{margin:5px 0 0;color:var(--device-muted);font-size:12px;line-height:1.5}.context-card{display:flex;justify-content:space-between;gap:14px;align-items:center;margin:0 0 14px;padding:13px 14px;background:var(--device-soft);border:1px solid var(--device-line);border-radius:9px}.context-card>div{display:grid;gap:4px}.context-card span{color:var(--device-muted);font-size:12px;line-height:1.6}.context-state{flex:none;padding:4px 8px;background:var(--device-panel);border-radius:999px;font-size:12px}.migration-card{margin:0 0 13px;padding:0;border:1px solid var(--device-line);border-radius:9px;background:var(--device-soft)}.migration-heading,.migration-actions{display:flex;justify-content:space-between;gap:10px;align-items:flex-start}.migration-heading{min-height:44px;padding:13px 14px;cursor:pointer;list-style:none}.migration-heading::-webkit-details-marker{display:none}.migration-heading h3{font-size:16px}.migration-heading p{margin:4px 0 0;opacity:.67}.migration-heading>strong{flex:none;padding:4px 8px;background:var(--device-panel);border-radius:999px;font-size:12px}.migration-body{padding:0 14px 13px}.migration-card ul{display:grid;gap:5px;margin:0 0 11px;padding:0;list-style:none}.migration-card li{display:grid;grid-template-columns:minmax(100px,.7fr) minmax(120px,1.4fr) auto;gap:8px;align-items:center;padding:7px 8px;background:rgba(127,127,127,.05);border-radius:6px}.migration-card li b{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.migration-card li em{font-size:12px;font-style:normal}.migration-card li em.conflict,.migration-card li em.unresolved,.migration-warning{color:#9b3b16}.migration-more,.migration-warning,.migration-read-error{margin:8px 0;font-size:12px}.migration-actions{justify-content:flex-end}.migration-actions button{min-height:44px;padding:6px 11px;color:var(--device-accent);background:transparent;border:1px solid var(--device-line);border-radius:7px;cursor:pointer}.migration-actions .primary{color:var(--device-primary-text);background:var(--device-primary-bg);border-color:var(--device-primary-bg)}.settings-nav{display:flex;gap:5px;margin-bottom:13px;border-bottom:1px solid rgba(127,127,127,.14)}.settings-nav button{padding:9px 12px;color:inherit;background:transparent;border:0;border-bottom:2px solid transparent;cursor:pointer}.settings-nav button.active{color:var(--device-accent);border-color:var(--device-accent);font-weight:600}.settings-stack{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;align-items:start}.setting-card{min-width:0;padding:14px;border:1px solid var(--device-line);border-radius:9px}.setting-card h3{font-size:16px}.card-heading{display:flex;justify-content:space-between;gap:12px;align-items:flex-start;margin-bottom:13px}.setting-card label:not(.switch-line){display:grid;gap:5px;margin:10px 0;font-size:12px}.setting-card input:not([type=checkbox]),.setting-card select{box-sizing:border-box;width:100%;min-width:0;min-height:44px;padding:7px 9px;color:inherit;background:transparent;border:1px solid var(--device-line);border-radius:7px}.switch-line{display:flex;align-items:center;gap:7px;white-space:nowrap}.two-columns,.route-form{display:grid;grid-template-columns:1fr 1fr;gap:9px}.route-form button{grid-column:1/-1}.setting-card button{min-height:44px;padding:6px 11px;color:var(--device-accent);background:transparent;border:1px solid var(--device-line);border-radius:7px;cursor:pointer}.setting-card .primary{color:var(--device-primary-text);background:var(--device-primary-bg);border-color:var(--device-primary-bg)}.capability-pill{flex:none;padding:4px 8px;border-radius:999px;background:rgba(127,127,127,.08);font-size:12px}.capability-pill.available{color:var(--device-good);background:var(--device-soft)}.capability-pill.not_installed,.capability-pill.error,.capability-pill.unsupported{color:var(--device-warning);background:var(--device-warning-soft)}.capability-action,.state-message{padding:0;color:var(--device-muted);font-size:13px;line-height:1.6}.capability-action p{margin:0 0 9px}.route-list{display:grid;gap:6px;margin-top:12px}.route-list>div{display:flex;justify-content:space-between;gap:10px;align-items:center;padding:9px;background:rgba(127,127,127,.05);border-radius:7px}.route-list span{display:grid;gap:2px}.route-list small{opacity:.62}.route-actions,.preview-actions{display:flex;gap:6px;align-items:center}.impact-preview{margin-top:12px;padding:12px;border:1px solid var(--device-line);border-radius:8px;background:var(--device-accent-soft)}.impact-preview h4{margin:0 0 6px}.impact-preview p{margin:5px 0}.preview-actions{justify-content:flex-end;margin-top:10px}.field-error{color:#9b3b16}.danger-link{color:#a13c3c!important;border-color:rgba(161,60,60,.3)!important}.feedback{margin:12px 0 0;padding:9px;border-radius:7px}.feedback.success{color:var(--device-good);background:var(--device-soft)}.feedback.error{color:#9b3b16;background:#fff1e8}button:focus-visible,input:focus-visible,select:focus-visible,summary:focus-visible{outline:2px solid var(--device-accent);outline-offset:2px}button:disabled{opacity:.55;cursor:not-allowed}
.capability-actions{display:flex;gap:7px;flex-wrap:wrap}.service-ready{margin:0 0 12px;padding:10px;color:var(--device-good);background:var(--device-soft);border-radius:8px}.advanced-group{margin-top:13px;padding-top:11px;border-top:1px solid rgba(127,127,127,.14)}.advanced-group summary{cursor:pointer;font-weight:600}.advanced-group>label{margin-top:12px!important}.advanced-group small{opacity:.66}.legacy-migration{margin-top:12px;padding:11px;border-radius:8px;background:rgba(127,127,127,.05)}.legacy-migration h4{margin:0}.legacy-migration p{margin:6px 0 10px}.migration-review{margin-top:10px}.migration-review ul{display:grid;gap:5px;margin:0 0 8px;padding:0;list-style:none}.migration-review li{display:grid;grid-template-columns:minmax(100px,.7fr) minmax(120px,1.4fr) auto;gap:8px;align-items:center;padding:7px 8px;background:rgba(127,127,127,.05);border-radius:6px}.migration-review li b{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.migration-review li em{font-size:12px;font-style:normal}.migration-review li em.unsupported,.migration-review li em.conflict{color:#9b3b16}.migration-review li em.convert{color:#176b45}.checkbox-line{display:flex!important;align-items:flex-start;gap:7px}.setting-hint{font-size:12px;color:var(--device-muted);line-height:1.6}
@media(max-width:900px){.settings-stack{grid-template-columns:1fr}}
@media(max-width:520px){.context-card,.card-heading,.migration-heading{align-items:flex-start;flex-direction:column}.migration-card li,.migration-review li{grid-template-columns:1fr auto}.migration-card li b,.migration-review li b{grid-column:1/-1;grid-row:2}.settings-nav{overflow-x:auto}.settings-nav button{flex:1;min-width:0;white-space:normal;padding:8px 6px;font-size:13px}.two-columns,.route-form{grid-template-columns:1fr}.route-list>div{align-items:flex-start;flex-direction:column}.route-actions{width:100%;justify-content:flex-end}.setting-card{padding:12px}}
</style>
