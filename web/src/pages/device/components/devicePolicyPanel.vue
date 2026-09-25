<template>
    <div class="policy-panel">
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取设备策略')" />
        <PageState v-else-if="loadError" kind="error" :title="$gettext('策略读取失败')" :description="loadError"
            :action-label="$gettext('重新加载')" @action="load" />
        <template v-else-if="policy">
            <div v-if="feedback" class="policy-feedback" :class="feedbackKind" role="status">{{ feedback }}</div>

            <section v-if="mode === 'all' || mode === 'network'" class="policy-card">
                <div class="policy-card__heading"><div><strong>{{ $gettext('地址与上网路线') }}</strong><small>{{ $gettext('固定设备地址，并选择它通过哪台路由器上网') }}</small></div>
                    <label class="policy-toggle"><input v-model="staticForm.enabled" type="checkbox" :disabled="!available('static') || routeLocked" /><span>{{ $gettext('地址预留') }} · {{ staticForm.enabled ? $gettext('已启用') : $gettext('未启用') }}</span></label>
                </div>
                <p v-if="routeLocked" class="policy-reason">{{ routerContext?.routeEditability?.guidance || $gettext('本机当前不负责地址分配，请到实际分配地址的路由器上修改。') }}</p>
                <p v-if="reason('static')" class="policy-reason">{{ reason('static') }}</p>
                <template v-else>
                    <label><span>{{ $gettext('上网路线') }}</span>
                        <select v-model="staticForm.targetId" :disabled="routeLocked || networkPolicy.targets.length === 0">
                            <option v-for="target in networkPolicy.targets" :key="target.id" :value="target.id" :disabled="!target.supported">
                                {{ target.name }}{{ target.gateway ? ` · ${target.gateway}` : '' }}{{ target.supported ? '' : $gettext('（不可用）') }}
                            </option>
                        </select>
                    </label>
                    <div class="effect-flow" :aria-label="$gettext('上网路线状态')"><span><small>{{ $gettext('想要的路线') }}</small>{{ desiredPathText }}</span><span><small>{{ $gettext('路由器配置') }}</small>{{ appliedPathText }}</span><span><small>{{ $gettext('设备当前状态') }}</small>{{ pathStateText }}</span></div>
                    <small v-if="networkPolicy.path.effect.needsAttention" class="attention" role="status">{{ $gettext('设置已保存，但设备可能需要断开并重新连接后才会使用新路线。') }}</small>
                    <label v-if="staticForm.enabled"><span>{{ $gettext('IPv4 地址') }}</span><input v-model.trim="staticForm.assignedIP" inputmode="decimal" placeholder="192.168.100.50" :disabled="routeLocked" /></label>
                    <details>
                        <summary>{{ $gettext('高级选项') }}</summary>
                        <label><span>{{ $gettext('DHCP 主机名（可选）') }}</span><input v-model.trim="staticForm.hostname"
                            maxlength="63" autocomplete="off" autocapitalize="none" spellcheck="false" /></label>
                        <small class="field-hint">{{ $gettext('用于局域网名称解析，仅支持英文、数字和中间连字符') }}</small>
                        <small v-if="hostnameInvalid" class="field-error" role="alert">{{ $gettext('请输入 1～63 位英文、数字或中间连字符，不能以连字符开头或结尾') }}</small>
                        <label class="checkbox-line"><input v-model="staticForm.bindIP" type="checkbox" />{{ $gettext('绑定 MAC 与 IPv4') }}</label>
                    </details>
                    <button type="button" :disabled="routeLocked || saving !== '' || hostnameInvalid || !staticForm.targetId" @click="saveNetworkPolicy">{{ saving === 'static' ? $gettext('正在保存…') : $gettext('保存地址与上网路线') }}</button>
                </template>
            </section>

            <section v-if="mode === 'all' || mode === 'restrictions'" class="policy-card">
                <div class="policy-card__heading"><div><strong>{{ $gettext('设备限速') }}</strong><small>{{ $gettext('限制这台设备的最高速率') }}</small></div>
                    <label class="policy-toggle"><input v-model="speedForm.enabled" type="checkbox" :disabled="!available('speed')" /><span>{{ speedForm.enabled ? $gettext('已启用') : $gettext('未启用') }}</span></label>
                </div>
                <div v-if="reason('speed')" class="policy-reason"><span>{{ reason('speed') }}</span><button v-if="policy.capabilities.speed?.state === 'not_installed'" type="button" :disabled="saving !== ''" @click="installSpeed">{{ saving === 'install' ? $gettext('正在安装…') : $gettext('安装限速服务') }}</button></div>
                <template v-else>
                    <div v-if="speedForm.enabled" class="field-grid">
                        <label><span>{{ $gettext('上传上限（Mbit/s）') }}</span><input v-model.number="speedForm.uploadSpeed" type="number" min="1" /></label>
                        <label><span>{{ $gettext('下载上限（Mbit/s）') }}</span><input v-model.number="speedForm.downloadSpeed" type="number" min="1" /></label>
                    </div>
                    <button type="button" :disabled="saving !== ''" @click="saveSpeed">{{ saving === 'speed' ? $gettext('正在保存…') : $gettext('保存限速') }}</button>
                </template>
            </section>

            <section v-if="mode === 'all' || mode === 'restrictions'" class="policy-card">
                <div class="policy-card__heading"><div><strong>{{ $gettext('联网权限') }}</strong><small>{{ policy.access.networkAccess ? $gettext('当前允许访问互联网') : $gettext('当前已断开互联网') }}</small></div>
                    <span class="access-state" :class="policy.access.networkAccess ? 'allowed' : 'blocked'">{{ policy.access.networkAccess ? $gettext('允许') : $gettext('已断网') }}</span>
                </div>
                <p v-if="reason('access')" class="policy-reason">{{ reason('access') }}</p>
                <button v-else type="button" :class="{ danger: policy.access.networkAccess }" :disabled="saving !== ''" @click="toggleAccess">
                    {{ saving === 'access' ? $gettext('正在应用…') : policy.access.networkAccess ? $gettext('断开网络') : $gettext('恢复联网') }}
                </button>
            </section>
        </template>
    </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import PageState from './pageState.vue'
import type { DeviceListItem } from '../deviceInventory'
import {
    policyAvailable,
    policyErrorLabel,
    policyLabelsFromPolicy,
    policyMacSuffix,
    policyUnavailableReason,
    normalizeDhcpHostname,
    validDhcpHostname,
    internetPathStateLabel,
    type DevicePolicy,
    type DeviceNetworkPolicy,
} from '../devicePolicy'

const props = withDefaults(defineProps<{ device: DeviceListItem; mode?: 'all' | 'network' | 'restrictions' }>(), { mode: 'all' })
const mode = computed(() => props.mode)
const emit = defineEmits<{ (event: 'saved', labels: string[]): void }>()
const { $gettext } = useGettext()

const loading = ref(true)
const loadError = ref('')
const saving = ref('')
const feedback = ref('')
const feedbackKind = ref<'success' | 'error'>('success')
const policy = ref<DevicePolicy | null>(null)
const emptyAddress = { enabled: false, assignedIP: '', bindIP: false, hostname: '' }
const networkPolicy = ref<DeviceNetworkPolicy>({
    deviceId: '', static: { ...emptyAddress },
    path: { targetId: '', effect: {
        desired: { targetId: '', static: { ...emptyAddress } },
        applied: { targetId: '', configVersion: '', state: 'server_configuration_observed' },
        observed: { state: 'unverifiable', reason: 'not_loaded' }, needsAttention: false,
    } },
    targets: [], version: '',
})
const routerContext = ref<any>(null)
const staticForm = reactive({ enabled: false, assignedIP: '', bindIP: false, hostname: '', targetId: '' })
const speedForm = reactive({ enabled: false, uploadSpeed: 100, downloadSpeed: 1000 })

const syncForms = () => {
    if (!policy.value) return
    Object.assign(staticForm, {
        enabled: networkPolicy.value.static.enabled,
        assignedIP: networkPolicy.value.static.assignedIP || policy.value.currentIPv4 || '',
        bindIP: networkPolicy.value.static.bindIP,
        hostname: networkPolicy.value.static.hostname || '',
        targetId: networkPolicy.value.path.targetId,
    })
    Object.assign(speedForm, {
        enabled: policy.value.speed.enabled,
        uploadSpeed: policy.value.speed.uploadSpeed || 100,
        downloadSpeed: policy.value.speed.downloadSpeed || 1000,
    })
}
const load = async () => {
    loading.value = true
    loadError.value = ''
    feedback.value = ''
    try {
        const [policyResponse, networkResponse, contextResponse] = await Promise.all([
            request.DeviceMangement.devicePolicyV2.GET(props.device.deviceId),
            request.DeviceMangement.deviceNetworkPolicyV2.GET(props.device.deviceId),
            request.DeviceMangement.routerContextV2.GET().catch(() => null),
        ])
        if (policyResponse.data?.result?.error) throw new Error(policyResponse.data.result.error.message)
        if (networkResponse.data?.result?.error) throw new Error(networkResponse.data.result.error.message)
        if (!policyResponse.data?.result?.policy || !networkResponse.data?.result?.policy) throw new Error($gettext('设备策略不可用'))
        policy.value = policyResponse.data.result.policy
        networkPolicy.value = networkResponse.data.result.policy
        routerContext.value = contextResponse?.data?.result || null
        syncForms()
        restoreDraft()
    } catch (error: any) {
        loadError.value = error?.response?.data?.error || error?.message || $gettext('读取结果失败')
    } finally {
        loading.value = false
    }
}
const available = (kind: 'static' | 'speed' | 'access') => policyAvailable(policy.value, kind)
const reason = (kind: 'static' | 'speed' | 'access') => policyUnavailableReason(policy.value, kind)
const hostnameInvalid = computed(() => !validDhcpHostname(staticForm.hostname))
const routeLocked = computed(() => routerContext.value?.routeEditability?.editable === false)
const pathStateText = computed(() => internetPathStateLabel(networkPolicy.value.path.effect.observed.state))
const desiredPathText = computed(() => networkPolicy.value.targets.find(item => item.id === networkPolicy.value.path.effect.desired?.targetId)?.name || $gettext('默认路线'))
const appliedPathText = computed(() => networkPolicy.value.path.effect.applied?.state === 'server_configuration_observed' ? $gettext('已写入') : $gettext('等待确认'))
const draftKey = computed(() => `quickstart.device-policy-draft.${props.device.deviceId}`)
const rememberDraft = () => sessionStorage.setItem(draftKey.value, JSON.stringify({ speed: { ...speedForm } }))
const restoreDraft = () => { try { const draft = JSON.parse(sessionStorage.getItem(draftKey.value) || '{}'); if (draft.speed) Object.assign(speedForm, draft.speed) } catch (_) {} }
const apply = async (kind: 'static' | 'speed' | 'access', value: Record<string, unknown>) => {
    saving.value = kind
    feedback.value = ''
    try {
        const response = await request.DeviceMangement.devicePolicyV2.POST({
            deviceId: props.device.deviceId,
            kind,
            idempotencyKey: `${props.device.deviceId}:${kind}:${Date.now()}`,
            [kind]: value,
        })
        const result = response.data?.result
        if (result?.error) throw Object.assign(new Error(result.error.message), { code: result.error.code })
        policy.value = result.policy
        syncForms()
        feedbackKind.value = 'success'
        feedback.value = result.changed ? $gettext('已保存并生效') : $gettext('设置没有变化')
        emit('saved', policyLabelsFromPolicy(result.policy))
    } catch (error: any) {
        feedbackKind.value = 'error'
        feedback.value = policyErrorLabel(error?.code, error?.message)
    } finally {
        saving.value = ''
    }
}
const saveNetworkPolicy = async () => {
    if (hostnameInvalid.value) return
    saving.value = 'static'
    feedback.value = ''
    try {
        const response = await request.DeviceMangement.deviceNetworkPolicyV2.POST({
            deviceId: props.device.deviceId,
            idempotencyKey: `${props.device.deviceId}:network:${Date.now()}`,
            static: {
                enabled: staticForm.enabled,
                assignedIP: staticForm.enabled ? staticForm.assignedIP : '',
                bindIP: staticForm.enabled && staticForm.bindIP,
                hostname: staticForm.enabled ? normalizeDhcpHostname(staticForm.hostname) : '',
            },
            targetId: staticForm.targetId,
            expectedVersion: networkPolicy.value.version,
        })
        const result = response.data?.result
        if (result?.error) throw Object.assign(new Error(result.error.message), { code: result.error.code })
        if (!result?.policy) throw new Error($gettext('设备策略不可用'))
        networkPolicy.value = result.policy
        if (policy.value) Object.assign(policy.value.static, result.policy.static)
        syncForms()
        feedbackKind.value = 'success'
        feedback.value = result.changed
            ? $gettext('配置已保存，等待设备重新获取地址')
            : $gettext('设置没有变化')
        if (policy.value) emit('saved', policyLabelsFromPolicy(policy.value))
    } catch (error: any) {
        feedbackKind.value = 'error'
        feedback.value = policyErrorLabel(error?.code, error?.message)
    } finally {
        saving.value = ''
    }
}
const saveSpeed = () => apply('speed', { ...speedForm })
const installSpeed = async () => {
    rememberDraft(); saving.value = 'install'; feedback.value = ''
    try {
        const draftToken = `${props.device.deviceId}:restrictions`
        const plan = (await request.DeviceMangement.capabilityActionV2.PLAN({ capabilityKey: 'device_speed_limit', action: 'install', draftToken })).data?.result?.plan
        if (plan?.error) throw new Error(plan.error.message)
        if (!plan?.canApply || !window.confirm(`${$gettext('安装限速服务？')}\n${$gettext('尚未保存的限速输入会保留。')}`)) return
        const result = (await request.DeviceMangement.capabilityActionV2.APPLY({ capabilityKey: 'device_speed_limit', action: 'install', draftToken, confirm: true, expectedState: plan.current?.state })).data?.result
        if (result?.error?.code === 'install_pending') throw new Error($gettext('组件仍在安装，稍后刷新即可继续，草稿已经保留。'))
        if (result?.error) throw new Error(result.error.message)
        feedbackKind.value = 'success'; feedback.value = $gettext('限速服务已安装，输入内容已恢复'); await load()
    } catch (error: any) { feedbackKind.value = 'error'; feedback.value = error?.message || $gettext('安装失败，可稍后重试') }
    finally { saving.value = '' }
}
const toggleAccess = () => {
    if (!policy.value) return
    const next = !policy.value.access.networkAccess
    if (!next) {
        const name = policy.value.displayName || props.device.displayName || props.device.hostname || $gettext('未命名设备')
        const suffix = policyMacSuffix(props.device.mac)
        if (!window.confirm(`${$gettext('确认断开设备网络？')}\n${name} · MAC …${suffix}`)) return
    }
    apply('access', { networkAccess: next })
}

watch(() => props.device.deviceId, load, { immediate: true })
</script>

<style lang="scss" scoped>
.policy-panel { display: flex; flex-direction: column; gap: 12px; }
.policy-feedback { padding: 9px 11px; border-radius: 7px; font-size: 13px; }
.policy-feedback.success { color: #176b45; background: rgba(38, 162, 105, .1); }
.policy-feedback.error { color: #9b3b16; background: #fff1e8; }
.policy-card { padding: 13px; border: 1px solid rgba(127, 127, 127, .16); border-radius: 8px; }
.policy-card__heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.policy-card__heading > div { display: flex; flex-direction: column; gap: 3px; }
.policy-card__heading small { opacity: .62; }
.policy-toggle { display: flex !important; flex-direction: row !important; align-items: center; gap: 6px; white-space: nowrap; }
.policy-card label { display: flex; flex-direction: column; gap: 5px; margin-top: 11px; font-size: 12px; }
.policy-card input:not([type='checkbox']), .policy-card select { width: 100%; min-height: 34px; padding: 6px 9px; color: inherit; background: transparent; border: 1px solid rgba(127, 127, 127, .3); border-radius: 6px; }
.field-hint, .field-error { display: block; margin-top: 5px; font-size: 11px; }
.field-hint { opacity: .62; }
.field-error { color: #b94343; }
.policy-card button { width: 100%; min-height: 36px; margin-top: 12px; color: #fff; background: #553afe; border: 1px solid #553afe; border-radius: 6px; cursor: pointer; }
.policy-card button:disabled { cursor: not-allowed; opacity: .55; }
.policy-card button.danger { background: #b94343; border-color: #b94343; }
.policy-reason { margin: 10px 0 0; padding: 8px 10px; opacity: .72; background: rgba(127, 127, 127, .07); border-radius: 6px; font-size: 12px; }
.policy-reason button { width: auto; margin: 8px 0 0; }.effect-flow { display: grid; grid-template-columns: repeat(3,minmax(0,1fr)); gap: 6px; margin-top: 9px; }.effect-flow span { display: grid; gap: 3px; padding: 7px; background: rgba(85,58,254,.05); border-radius: 6px; overflow-wrap: anywhere; }.effect-flow small { opacity: .6; }.attention { display: block; margin-top: 7px; padding: 7px; color: #8a5a00; background: #fff8e8; border-radius: 6px; }
details { margin-top: 11px; }
summary { color: #553afe; cursor: pointer; font-size: 12px; }
.checkbox-line { flex-direction: row !important; align-items: center; }
.field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 9px; }
.access-state { flex: none; padding: 3px 7px; border-radius: 999px; font-size: 12px; }
.access-state.allowed { color: #176b45; background: rgba(38, 162, 105, .1); }
.access-state.blocked { color: #9b3b16; background: #fff1e8; }
@media (max-width: 420px) { .field-grid,.effect-flow { grid-template-columns: 1fr; } }
</style>
