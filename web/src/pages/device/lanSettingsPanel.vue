<template>
    <section class="lan-settings" aria-labelledby="lan-settings-title">
        <header><div><h2 id="lan-settings-title">{{ $gettext('局域网设置') }}</h2><p>{{ $gettext('管理本机提供的网络服务，并集中处理需要维护的设备规则。') }}</p></div></header>
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取局域网设置')" />
        <PageState v-else-if="error" kind="error" :title="$gettext('局域网设置读取失败')" :description="error" :action-label="$gettext('重新加载')" @action="load" />
        <template v-else>
            <article class="context-card" :class="routerContext?.topologyPosition">
                <div><strong>{{ contextTitle }}</strong><span>{{ contextDescription }}</span></div>
                <span class="context-state">{{ routeEditability }}</span>
            </article>

            <article v-if="migrationPlan && !migrationPlan.alreadyApplied && migrationPlan.items?.length" class="migration-card" aria-labelledby="migration-title">
                <div class="migration-heading"><div><h3 id="migration-title">{{ $gettext('发现现有设备设置') }}</h3><p>{{ $gettext('先预览并接管现有设置，之后只使用当前设备管理界面维护。') }}</p></div><strong>{{ migrationPlan.items.length }} {{ $gettext('项') }}</strong></div>
                <ul><li v-for="item in migrationPlan.items.slice(0, 5)" :key="item.id"><span>{{ migrationKind(item.kind) }}</span><b>{{ item.summary || '—' }}</b><em :class="item.disposition">{{ migrationDisposition(item.disposition) }}</em></li></ul>
                <p v-if="migrationPlan.items.length > 5" class="migration-more">{{ $gettext('其余设置会在同一次安全接管中处理。') }}</p>
                <p v-if="!migrationPlan.canApply" class="migration-warning">{{ $gettext('发现冲突或无法识别的设置，当前不会写入任何配置。请先在规则台账中处理。') }}</p>
                <div class="migration-actions"><button type="button" @click="section = 'rules'">{{ $gettext('查看规则台账') }}</button><button class="primary" type="button" :disabled="!migrationPlan.canApply || migrating" @click="applyMigration">{{ migrating ? $gettext('正在接管…') : $gettext('确认接管现有设置') }}</button></div>
            </article>
            <p v-else-if="migrationError" class="migration-read-error" role="status">{{ $gettext('暂时无法检查现有设置，不影响其他局域网设置。') }}</p>

            <nav class="settings-nav" :aria-label="$gettext('设置区域')">
                <button v-for="item in sections" :key="item.id" type="button" :class="{ active: section === item.id }" @click="section = item.id">{{ item.label }}</button>
            </nav>

            <div v-if="section === 'services'" class="settings-stack">
                <article class="setting-card">
                    <div class="card-heading"><div><h3>{{ $gettext('地址分配服务') }}</h3><p>{{ $gettext('由本机为局域网设备分配地址。关闭后，设备应由其他路由器分配地址。') }}</p></div><label class="switch-line"><input v-model="dhcp.enabled" type="checkbox" />{{ dhcp.enabled ? $gettext('已开启') : $gettext('已关闭') }}</label></div>
                    <label><span>{{ $gettext('默认上网出口') }}</span><select v-model="dhcp.gateway"><option v-for="gateway in gateways" :key="gateway.gateway" :value="gateway.gateway">{{ gatewayLabel(gateway) }}</option></select></label>
                    <button class="primary" type="button" :disabled="saving === 'dhcp'" @click="saveDhcp">{{ saving === 'dhcp' ? $gettext('正在保存…') : $gettext('保存地址分配设置') }}</button>
                </article>

                <article class="setting-card">
                    <div class="card-heading"><div><h3>{{ $gettext('设备限速服务') }}</h3><p>{{ $gettext('为设备设置最高上传和下载速度；联网权限不依赖此服务。') }}</p></div><span class="capability-pill" :class="speedCapability.state">{{ capabilityLabel(speedCapability.state) }}</span></div>
                    <div v-if="speedCapability.state === 'not_installed'" class="capability-action"><p>{{ $gettext('安装后会回到当前区域，尚未保存的输入会保留。') }}</p><button class="primary" type="button" :disabled="installing" @click="installCapability('device_speed_limit')">{{ installing ? $gettext('正在安装…') : $gettext('安装限速服务') }}</button></div>
                    <template v-else-if="speedCapability.state === 'available' || speedCapability.state === 'disabled'">
                        <label class="switch-line"><input v-model="speed.enabled" type="checkbox" />{{ $gettext('启用设备限速') }}</label>
                        <div v-if="speed.enabled" class="two-columns"><label><span>{{ $gettext('总上传带宽（Mbit/s）') }}</span><input v-model.number="speed.upload" type="number" min="1" /></label><label><span>{{ $gettext('总下载带宽（Mbit/s）') }}</span><input v-model.number="speed.download" type="number" min="1" /></label></div>
                        <button class="primary" type="button" :disabled="saving === 'speed'" @click="saveSpeed">{{ saving === 'speed' ? $gettext('正在保存…') : $gettext('保存限速服务设置') }}</button>
                    </template>
                    <p v-else class="state-message">{{ speedCapability.reason || $gettext('当前无法读取限速服务状态，其他设置仍可使用。') }}</p>
                </article>
            </div>

            <div v-else-if="section === 'routes'" class="settings-stack">
                <article class="setting-card">
                    <div class="card-heading"><div><h3>{{ $gettext('自定义上网路线') }}</h3><p>{{ $gettext('把复杂网络工作交给旁路由或指定网关，设备可在详情中选择路线。DNS 跟随路线。') }}</p></div><button type="button" @click="routeEditor = !routeEditor">{{ routeEditor ? $gettext('取消') : $gettext('添加路线') }}</button></div>
                    <form v-if="routeEditor" class="route-form" @submit.prevent="saveRoute"><label><span>{{ $gettext('路线名称') }}</span><input v-model.trim="routeDraft.title" maxlength="64" required :placeholder="$gettext('例如：旁路由')" /></label><label><span>{{ $gettext('网关地址') }}</span><input v-model.trim="routeDraft.gateway" inputmode="decimal" required placeholder="192.168.1.2" /></label><button class="primary" type="submit" :disabled="saving === 'route'">{{ saving === 'route' ? $gettext('正在保存…') : $gettext('保存路线') }}</button></form>
                    <div v-if="routes.length" class="route-list"><div v-for="route in routes" :key="route.tagName"><span><strong>{{ routeTitle(route) }}</strong><small>{{ route.gateway || '—' }}</small></span><button v-if="!route.autoCreated" type="button" class="danger-link" @click="deleteRoute(route)">{{ $gettext('删除') }}</button><em v-else>{{ $gettext('由系统管理') }}</em></div></div>
                    <PageState v-else kind="empty" :title="$gettext('还没有自定义上网路线')" :description="$gettext('设备默认通过本机上网；需要时再添加旁路由或指定网关。')" />
                </article>
                <article class="setting-card">
                    <div class="card-heading"><div><h3>{{ $gettext('浮动网关') }}</h3><p>{{ $gettext('主路由与旁路由共享一个稳定入口，旁路由异常时自动切回主路由。') }}</p></div><span class="capability-pill" :class="floatCapability.state">{{ capabilityLabel(floatCapability.state) }}</span></div>
                    <div v-if="floatCapability.state === 'not_installed'" class="capability-action"><p>{{ $gettext('只有浮动网关功能会被锁定，设备和其他路线仍可管理。') }}</p><button class="primary" type="button" :disabled="installing" @click="installCapability('floating_gateway')">{{ installing ? $gettext('正在安装…') : $gettext('安装浮动网关') }}</button></div>
                    <FloatingGatewayWizard v-else />
                </article>
            </div>

            <NetworkRulesHub v-else />
            <p v-if="feedback" class="feedback" :class="feedbackKind" role="status">{{ feedback }}</p>
        </template>
    </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import Toast from '/@/components/toast'
import PageState from './components/pageState.vue'
import FloatingGatewayWizard from './components/floatingGatewayWizard.vue'
import NetworkRulesHub from './networkRulesHub.vue'
import { resolveCapability, type DeviceCapability } from './deviceCapabilities'

const { $gettext } = useGettext()
const draftKey = 'quickstart.lan-settings-draft.v1'
const restored = (() => { try { return JSON.parse(sessionStorage.getItem(draftKey) || '{}') } catch { return {} } })()
const loading = ref(true), error = ref(''), saving = ref(''), installing = ref(false), migrating = ref(false), feedback = ref(''), feedbackKind = ref<'success'|'error'>('success')
const section = ref<'services'|'routes'|'rules'>(['services','routes','rules'].includes(restored.section) ? restored.section : 'services')
const routerContext = ref<any>(), globalData = ref<any>({}), speedCapability = ref<DeviceCapability>({ state: 'error' }), floatCapability = ref<DeviceCapability>({ state: 'error' })
const migrationPlan = ref<any>(), migrationError = ref(false)
const dhcp = reactive({ enabled: true, gateway: '' }), speed = reactive({ enabled: false, upload: 100, download: 1000 })
const routeEditor = ref(Boolean(restored.routeEditor)), routeDraft = reactive({ title: restored.routeTitle || '', gateway: restored.routeGateway || '' })
const sections = computed(() => [{ id: 'services' as const, label: $gettext('网络服务') }, { id: 'routes' as const, label: $gettext('上网路线') }, { id: 'rules' as const, label: $gettext('规则台账') }])
const gateways = computed<any[]>(() => globalData.value?.dhcpGlobal?.gatewaySels || [])
const routes = computed<any[]>(() => globalData.value?.dhcpTags || [])
const contextTitle = computed(() => ({ local: $gettext('本机负责局域网地址分配'), external_observed: $gettext('检测到其他设备负责地址分配'), none_detected: $gettext('尚未检测到地址分配服务'), ambiguous: $gettext('局域网角色需要确认'), error: $gettext('暂时无法判断局域网角色') } as Record<string,string>)[routerContext.value?.dhcpAuthority] || $gettext('局域网角色待确认'))
const contextDescription = computed(() => routerContext.value?.routeEditability?.guidance || (routerContext.value?.routeEditability?.editable ? $gettext('可在设备详情中设置上网路线。') : $gettext('当前仅提供查看与引导，不会自动修改其他设备。')))
const routeEditability = computed(() => routerContext.value?.routeEditability?.editable ? $gettext('路线可编辑') : $gettext('只读引导'))
const gatewayLabel = (value: any) => `${value.gateway} · ${({ default: $gettext('默认出口'), parent: $gettext('上级路由'), myself: $gettext('本设备'), bypass: $gettext('旁路由'), floatip: $gettext('浮动网关') } as Record<string,string>)[value.title] || value.title || $gettext('指定网关')}`
const routeTitle = (value: any) => ({ default: $gettext('默认路线'), parent: $gettext('上级路由'), myself: $gettext('本设备'), bypass: $gettext('旁路由'), floatip: $gettext('浮动网关') } as Record<string,string>)[value.tagTitle] || value.tagTitle || $gettext('自定义路线')
const capabilityLabel = (state: string) => ({ available: $gettext('可用'), disabled: $gettext('未启用'), not_installed: $gettext('未安装'), unsupported: $gettext('不支持'), error: $gettext('需要检查') } as Record<string,string>)[state] || $gettext('未知')
const migrationKind = (kind:string) => ({ alias:$gettext('设备资料'), hostname:$gettext('局域网主机名'), static:$gettext('地址预留'), route:$gettext('上网路线'), speed:$gettext('设备限速'), access:$gettext('联网权限'), floating_gateway:$gettext('浮动网关') } as Record<string,string>)[kind] || $gettext('设备规则')
const migrationDisposition = (value:string) => ({ adopt:$gettext('可以接管'), normalize:$gettext('将安全整理'), conflict:$gettext('存在冲突'), unresolved:$gettext('需要确认') } as Record<string,string>)[value] || $gettext('需要确认')
const rememberDraft = () => sessionStorage.setItem(draftKey, JSON.stringify({ section: section.value, routeEditor: routeEditor.value, routeTitle: routeDraft.title, routeGateway: routeDraft.gateway }))
watch([section, routeEditor, () => routeDraft.title, () => routeDraft.gateway], rememberDraft)

const load = async () => {
    loading.value = true; error.value = ''
    try {
        const [contextResponse, globalResponse, migrationResponse] = await Promise.all([request.DeviceMangement.routerContextV2.GET(), request.DeviceMangement.globalConfigs.GET(), request.DeviceMangement.lanDeviceMigrationV2.PLAN().catch(() => undefined)])
        routerContext.value = contextResponse.data?.result
        globalData.value = globalResponse.data?.result || {}
        migrationPlan.value = migrationResponse?.data?.result?.plan
        migrationError.value = !migrationResponse
        speedCapability.value = resolveCapability(globalData.value, 'speedLimit')
        floatCapability.value = resolveCapability(globalData.value, 'floatGateway')
        dhcp.enabled = Boolean(globalData.value?.dhcpGlobal?.dhcpEnabled)
        dhcp.gateway = globalData.value?.dhcpGlobal?.dhcpGateway || gateways.value[0]?.gateway || ''
        speed.enabled = Boolean(globalData.value?.speedLimit?.enabled)
        speed.upload = globalData.value?.speedLimit?.uploadSpeed || 100
        speed.download = globalData.value?.speedLimit?.downloadSpeed || 1000
    } catch (reason: any) { error.value = reason?.message || $gettext('读取结果失败') }
    finally { loading.value = false }
}
const saveDhcp = async () => { if (!dhcp.enabled && !window.confirm($gettext('关闭地址分配服务后，新设备可能无法联网。确定继续吗？'))) return; saving.value = 'dhcp'; try { await request.DeviceMangement.dhcpGatewayConfig.POST({ dhcpEnabled: dhcp.enabled, dhcpGateway: dhcp.gateway }); feedbackKind.value = 'success'; feedback.value = $gettext('地址分配设置已保存'); await load() } catch (reason:any) { feedbackKind.value='error'; feedback.value=reason?.message||$gettext('保存失败') } finally { saving.value='' } }
const saveSpeed = async () => { if (speed.enabled && (!speed.upload || !speed.download)) return; saving.value='speed'; try { await request.DeviceMangement.enableSpeedLimit.POST({ enabled:speed.enabled, uploadSpeed:speed.enabled?speed.upload:0, downloadSpeed:speed.enabled?speed.download:0 }); feedbackKind.value='success'; feedback.value=$gettext('限速服务设置已保存'); await load() } catch(reason:any){ feedbackKind.value='error'; feedback.value=reason?.message||$gettext('保存失败') } finally{ saving.value='' } }
const saveRoute = async () => { saving.value='route'; try { await request.DeviceMangement.dhcpTagsConfig.POST({ action:'add', tagTitle:routeDraft.title, tagName:`route_${Date.now().toString(36)}`, dhcpOption:[`3,${routeDraft.gateway}`,`6,${routeDraft.gateway}`] }); routeDraft.title='';routeDraft.gateway='';routeEditor.value=false;feedbackKind.value='success';feedback.value=$gettext('上网路线已添加');await load() } catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('保存失败')} finally{saving.value=''} }
const deleteRoute = async (route:any) => { if(!window.confirm(`${$gettext('删除上网路线')}“${routeTitle(route)}”？`))return; try{await request.DeviceMangement.dhcpTagsConfig.POST({action:'delete',tagTitle:route.tagTitle||'',tagName:route.tagName||'',dhcpOption:route.dhcpOption||[]});feedbackKind.value='success';feedback.value=$gettext('上网路线已删除');await load()}catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('删除失败')} }
const applyMigration = async () => {
    if (!migrationPlan.value?.canApply || !window.confirm($gettext('确认接管现有设置？接管前会创建快照，失败时自动恢复。'))) return
    migrating.value = true; feedback.value = ''
    try {
        const result = (await request.DeviceMangement.lanDeviceMigrationV2.APPLY({ expectedVersion: migrationPlan.value.version })).data?.result
        if (result?.error) throw new Error(result.error.message)
        feedbackKind.value = 'success'; feedback.value = result?.changed ? $gettext('现有设置已安全接管') : $gettext('现有设置无需重复接管')
        await load()
    } catch (reason:any) { feedbackKind.value='error'; feedback.value=reason?.message||$gettext('接管失败，原设置已保留') }
    finally { migrating.value=false }
}
const installCapability = async (capabilityKey:string) => {
    rememberDraft(); installing.value=true; feedback.value=''
    try {
        const draftToken = `lan-settings:${section.value}`
        const plan = (await request.DeviceMangement.capabilityActionV2.PLAN({ capabilityKey, action:'install', draftToken })).data?.result?.plan
        if (plan?.error) throw new Error(plan.error.message)
        if (!plan?.canApply) throw new Error($gettext('当前无法安装，请检查存储空间和软件源。'))
        if (!window.confirm(`${$gettext('确认安装所需组件？')}\n${$gettext('至少需要可用空间')} ${Math.ceil(plan.requiredFreeBytes/1048576)} MB`)) return
        const result = (await request.DeviceMangement.capabilityActionV2.APPLY({ capabilityKey, action:'install', draftToken, confirm:true, expectedState:plan.current?.state })).data?.result
        if (result?.error?.code === 'install_pending') throw new Error($gettext('组件仍在安装，稍后刷新即可继续，草稿已经保留。'))
        if (result?.error) throw new Error(result.error.message)
        feedbackKind.value='success';feedback.value=$gettext('组件安装完成，草稿已恢复');await load()
    } catch(reason:any){feedbackKind.value='error';feedback.value=reason?.message||$gettext('安装失败，可稍后重试');Toast.Warning(feedback.value)} finally{installing.value=false}
}
load()
</script>

<style lang="scss" scoped>
.lan-settings { color: var(--tit-color); }.lan-settings header h2,.setting-card h3,.migration-card h3 { margin:0;padding:0;color:inherit;background:none!important;text-align:left }.lan-settings header h2{font-size:20px}.lan-settings header p,.card-heading p{margin:5px 0 0;opacity:.65}.context-card{display:flex;justify-content:space-between;gap:14px;align-items:center;margin:16px 0 12px;padding:13px 14px;background:rgba(85,58,254,.06);border:1px solid rgba(85,58,254,.14);border-radius:9px}.context-card>div{display:grid;gap:4px}.context-card span{opacity:.68}.context-state{flex:none;padding:4px 8px;background:var(--card-bg-color);border-radius:999px;font-size:12px}.migration-card{margin:0 0 13px;padding:13px 14px;border:1px solid rgba(85,58,254,.18);border-radius:9px;background:rgba(85,58,254,.045)}.migration-heading,.migration-actions{display:flex;justify-content:space-between;gap:10px;align-items:flex-start}.migration-heading h3{font-size:16px}.migration-heading p{margin:4px 0 0;opacity:.67}.migration-heading>strong{flex:none;padding:4px 8px;background:var(--card-bg-color);border-radius:999px;font-size:12px}.migration-card ul{display:grid;gap:5px;margin:11px 0;padding:0;list-style:none}.migration-card li{display:grid;grid-template-columns:minmax(100px,.7fr) minmax(120px,1.4fr) auto;gap:8px;align-items:center;padding:7px 8px;background:rgba(127,127,127,.05);border-radius:6px}.migration-card li b{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.migration-card li em{font-size:12px;font-style:normal}.migration-card li em.conflict,.migration-card li em.unresolved,.migration-warning{color:#9b3b16}.migration-more,.migration-warning,.migration-read-error{margin:8px 0;font-size:12px}.migration-actions{justify-content:flex-end}.migration-actions button{min-height:34px;padding:6px 11px;color:var(--device-accent);background:transparent;border:1px solid rgba(85,58,254,.34);border-radius:7px;cursor:pointer}.migration-actions .primary{color:#fff;background:#553afe;border-color:#553afe}.settings-nav{display:flex;gap:5px;margin-bottom:13px;border-bottom:1px solid rgba(127,127,127,.14)}.settings-nav button{padding:9px 12px;color:inherit;background:transparent;border:0;border-bottom:2px solid transparent;cursor:pointer}.settings-nav button.active{color:var(--device-accent);border-color:var(--device-accent);font-weight:600}.settings-stack{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.setting-card{min-width:0;padding:14px;border:1px solid rgba(127,127,127,.15);border-radius:9px}.setting-card h3{font-size:16px}.card-heading{display:flex;justify-content:space-between;gap:12px;align-items:flex-start;margin-bottom:13px}.setting-card label:not(.switch-line){display:grid;gap:5px;margin:10px 0;font-size:12px}.setting-card input:not([type=checkbox]),.setting-card select{box-sizing:border-box;width:100%;min-width:0;min-height:36px;padding:7px 9px;color:inherit;background:transparent;border:1px solid rgba(127,127,127,.28);border-radius:7px}.switch-line{display:flex;align-items:center;gap:7px;white-space:nowrap}.two-columns,.route-form{display:grid;grid-template-columns:1fr 1fr;gap:9px}.route-form button{grid-column:1/-1}.setting-card button{min-height:34px;padding:6px 11px;color:var(--device-accent);background:transparent;border:1px solid rgba(85,58,254,.34);border-radius:7px;cursor:pointer}.setting-card .primary{color:#fff;background:#553afe;border-color:#553afe}.capability-pill{flex:none;padding:4px 8px;border-radius:999px;background:rgba(127,127,127,.08);font-size:12px}.capability-pill.available{color:#176b45;background:rgba(38,162,105,.1)}.capability-pill.not_installed,.capability-pill.error{color:#9b3b16;background:#fff1e8}.capability-action,.state-message{padding:10px;background:rgba(127,127,127,.06);border-radius:8px}.capability-action p{margin:0 0 9px}.route-list{display:grid;gap:6px;margin-top:12px}.route-list>div{display:flex;justify-content:space-between;gap:10px;align-items:center;padding:9px;background:rgba(127,127,127,.05);border-radius:7px}.route-list span{display:grid;gap:2px}.route-list small,.route-list em{opacity:.62;font-style:normal}.danger-link{color:#a13c3c!important;border-color:rgba(161,60,60,.3)!important}.feedback{margin:12px 0 0;padding:9px;border-radius:7px}.feedback.success{color:#176b45;background:rgba(38,162,105,.1)}.feedback.error{color:#9b3b16;background:#fff1e8}button:focus-visible,input:focus-visible,select:focus-visible{outline:2px solid var(--device-accent);outline-offset:2px}button:disabled{opacity:.55;cursor:not-allowed}
@media(max-width:900px){.settings-stack{grid-template-columns:1fr}}
@media(max-width:520px){.context-card,.card-heading,.migration-heading{align-items:flex-start;flex-direction:column}.migration-card li{grid-template-columns:1fr auto}.migration-card li b{grid-column:1/-1;grid-row:2}.settings-nav{overflow-x:auto}.settings-nav button{flex:none}.two-columns,.route-form{grid-template-columns:1fr}.setting-card{padding:12px}}
</style>
