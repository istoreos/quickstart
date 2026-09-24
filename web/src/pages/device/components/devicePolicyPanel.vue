<template>
    <div class="policy-panel">
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取设备策略')" />
        <PageState v-else-if="loadError" kind="error" :title="$gettext('策略读取失败')" :description="loadError"
            :action-label="$gettext('重新加载')" @action="load" />
        <template v-else-if="policy">
            <div v-if="feedback" class="policy-feedback" :class="feedbackKind" role="status">{{ feedback }}</div>

            <section class="policy-card">
                <div class="policy-card__heading"><div><strong>{{ $gettext('静态地址') }}</strong><small>{{ $gettext('让设备始终使用同一个 IPv4 地址') }}</small></div>
                    <label class="policy-toggle"><input v-model="staticForm.enabled" type="checkbox" :disabled="!available('static')" /><span>{{ staticForm.enabled ? $gettext('已启用') : $gettext('未启用') }}</span></label>
                </div>
                <p v-if="reason('static')" class="policy-reason">{{ reason('static') }}</p>
                <template v-else>
                    <label v-if="staticForm.enabled"><span>{{ $gettext('IPv4 地址') }}</span><input v-model.trim="staticForm.assignedIP" inputmode="decimal" placeholder="192.168.100.50" /></label>
                    <details>
                        <summary>{{ $gettext('高级选项') }}</summary>
                        <label><span>{{ $gettext('设备名称') }}</span><input v-model.trim="staticForm.hostname" /></label>
                        <label><span>{{ $gettext('DHCP 标签') }}</span><input v-model.trim="staticForm.tagName" /></label>
                        <label class="checkbox-line"><input v-model="staticForm.bindIP" type="checkbox" />{{ $gettext('绑定 MAC 与 IPv4') }}</label>
                    </details>
                    <button type="button" :disabled="saving !== ''" @click="saveStatic">{{ saving === 'static' ? $gettext('正在保存…') : $gettext('保存静态地址') }}</button>
                </template>
            </section>

            <section class="policy-card">
                <div class="policy-card__heading"><div><strong>{{ $gettext('设备限速') }}</strong><small>{{ $gettext('限制这台设备的最高速率') }}</small></div>
                    <label class="policy-toggle"><input v-model="speedForm.enabled" type="checkbox" :disabled="!available('speed')" /><span>{{ speedForm.enabled ? $gettext('已启用') : $gettext('未启用') }}</span></label>
                </div>
                <p v-if="reason('speed')" class="policy-reason">{{ reason('speed') }}</p>
                <template v-else>
                    <div v-if="speedForm.enabled" class="field-grid">
                        <label><span>{{ $gettext('上传上限（Mbit/s）') }}</span><input v-model.number="speedForm.uploadSpeed" type="number" min="1" /></label>
                        <label><span>{{ $gettext('下载上限（Mbit/s）') }}</span><input v-model.number="speedForm.downloadSpeed" type="number" min="1" /></label>
                    </div>
                    <button type="button" :disabled="saving !== ''" @click="saveSpeed">{{ saving === 'speed' ? $gettext('正在保存…') : $gettext('保存限速') }}</button>
                </template>
            </section>

            <section class="policy-card">
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
import { reactive, ref, watch } from 'vue'
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
    type DevicePolicy,
} from '../devicePolicy'

const props = defineProps<{ device: DeviceListItem }>()
const emit = defineEmits<{ (event: 'saved', labels: string[]): void }>()
const { $gettext } = useGettext()

const loading = ref(true)
const loadError = ref('')
const saving = ref('')
const feedback = ref('')
const feedbackKind = ref<'success' | 'error'>('success')
const policy = ref<DevicePolicy | null>(null)
const staticForm = reactive({ enabled: false, assignedIP: '', bindIP: false, hostname: '', tagName: '' })
const speedForm = reactive({ enabled: false, uploadSpeed: 100, downloadSpeed: 1000 })

const syncForms = () => {
    if (!policy.value) return
    Object.assign(staticForm, {
        enabled: policy.value.static.enabled,
        assignedIP: policy.value.static.assignedIP || policy.value.currentIPv4 || '',
        bindIP: policy.value.static.bindIP,
        hostname: policy.value.static.hostname || props.device.displayName || '',
        tagName: policy.value.static.tagName || '',
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
        const response = await request.DeviceMangement.devicePolicyV2.GET(props.device.deviceId)
        if (response.data?.result?.error) throw new Error(response.data.result.error.message)
        if (!response.data?.result?.policy) throw new Error($gettext('设备策略不可用'))
        policy.value = response.data.result.policy
        syncForms()
    } catch (error: any) {
        loadError.value = error?.response?.data?.error || error?.message || $gettext('读取结果失败')
    } finally {
        loading.value = false
    }
}
const available = (kind: 'static' | 'speed' | 'access') => policyAvailable(policy.value, kind)
const reason = (kind: 'static' | 'speed' | 'access') => policyUnavailableReason(policy.value, kind)
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
const saveStatic = () => apply('static', { ...staticForm })
const saveSpeed = () => apply('speed', { ...speedForm })
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
.policy-card input:not([type='checkbox']) { width: 100%; min-height: 34px; padding: 6px 9px; color: inherit; background: transparent; border: 1px solid rgba(127, 127, 127, .3); border-radius: 6px; }
.policy-card button { width: 100%; min-height: 36px; margin-top: 12px; color: #fff; background: #553afe; border: 1px solid #553afe; border-radius: 6px; cursor: pointer; }
.policy-card button:disabled { cursor: not-allowed; opacity: .55; }
.policy-card button.danger { background: #b94343; border-color: #b94343; }
.policy-reason { margin: 10px 0 0; padding: 8px 10px; opacity: .72; background: rgba(127, 127, 127, .07); border-radius: 6px; font-size: 12px; }
details { margin-top: 11px; }
summary { color: #553afe; cursor: pointer; font-size: 12px; }
.checkbox-line { flex-direction: row !important; align-items: center; }
.field-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 9px; }
.access-state { flex: none; padding: 3px 7px; border-radius: 999px; font-size: 12px; }
.access-state.allowed { color: #176b45; background: rgba(38, 162, 105, .1); }
.access-state.blocked { color: #9b3b16; background: #fff1e8; }
@media (max-width: 420px) { .field-grid { grid-template-columns: 1fr; } }
</style>
