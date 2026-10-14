<template>
    <div class="floating-wizard">
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取浮动网关状态')" />
        <PageState v-else-if="loadError" kind="error" :title="$gettext('浮动网关状态不可用')" :description="loadError"
            :action-label="$gettext('重新加载')" @action="load" />
        <div v-else-if="status.capability === 'not_installed'" class="capability-card">
            <strong>{{ $gettext('尚未安装浮动网关组件') }}</strong>
            <span>{{ $gettext('安装后可让两台路由器共同提供一个稳定的虚拟网关。') }}</span>
            <button type="button" @click="$emit('install')">{{ $gettext('立即安装') }}</button>
        </div>
        <template v-else>
            <div class="status-card">
                <div><small>{{ $gettext('运行状态') }}</small><strong>{{ statusLabel }}</strong></div>
                <div><small>{{ $gettext('虚拟 IP 持有者') }}</small><strong>{{ holderLabel }}</strong></div>
                <div><small>{{ $gettext('对端状态') }}</small><strong>{{ probeLabel(status.peerState) }}</strong></div>
                <div><small>{{ $gettext('外网检测') }}</small><strong>{{ probeLabel(status.externalState) }}</strong></div>
            </div>

            <section v-if="configured && !editing" class="configured-card">
                <div><small>{{ $gettext('虚拟网关 IPv4') }}</small><strong>{{ form.virtualIP || '—' }}</strong></div>
                <div><small>{{ $gettext('节点职责') }}</small><strong>{{ roleLabel }}</strong></div>
                <div><small>{{ $gettext('对端节点 IPv4') }}</small><strong>{{ peerText || '—' }}</strong></div>
                <button type="button" class="secondary" @click="startEditing">{{ $gettext('编辑') }}</button>
            </section>

            <template v-else>
                <ol class="wizard-steps" :aria-label="$gettext('配置步骤')">
                    <li v-for="item in 3" :key="item" :class="{ active: step === item, done: step > item }"><b>{{ item }}</b><span>{{ stepName(item) }}</span></li>
                </ol>

                <section v-if="step === 1" class="wizard-card">
                    <label class="toggle-line"><input v-model="form.enabled" type="checkbox" /><span>{{ $gettext('启用浮动网关') }}</span></label>
                    <p>{{ $gettext('两台节点必须配置相同的虚拟 IP，并选择互补的职责。') }}</p>
                    <div class="role-grid">
                        <label :class="{ selected: form.role === 'preferred' }"><input v-model="form.role" type="radio" value="preferred" /><strong>{{ $gettext('优先服务节点') }}</strong><small>{{ $gettext('正常情况下提供虚拟网关') }}</small></label>
                        <label :class="{ selected: form.role === 'takeover' }"><input v-model="form.role" type="radio" value="takeover" /><strong>{{ $gettext('故障接管节点') }}</strong><small>{{ $gettext('优先节点不可达时接管') }}</small></label>
                    </div>
                </section>

                <section v-else-if="step === 2" class="wizard-card form-grid">
                    <label><span>{{ $gettext('虚拟网关 IPv4') }}</span><input v-model.trim="form.virtualIP" inputmode="decimal" placeholder="192.168.100.3" /></label>
                    <label><span>{{ $gettext('对端节点 IPv4') }}</span><input v-model.trim="peerText" inputmode="decimal" placeholder="192.168.100.2" /><small>{{ $gettext('多个地址用逗号分隔') }}</small></label>
                    <label><span>{{ $gettext('外网检测地址（可选）') }}</span><input v-model.trim="form.healthURL" type="url" placeholder="https://example.com/health" /></label>
                    <label><span>{{ $gettext('检测超时（秒）') }}</span><input v-model.number="form.healthTimeoutSec" type="number" min="1" max="30" /></label>
                </section>

                <section v-else class="wizard-card review-card">
                    <PageState v-if="planning" kind="loading" :title="$gettext('正在预检配置')" />
                    <template v-else-if="plan">
                        <div class="review-state" :class="plan.error ? 'error' : 'ready'">
                            <strong>{{ plan.error ? $gettext('预检未通过') : plan.changed ? $gettext('可以安全应用') : $gettext('配置没有变化') }}</strong>
                            <span v-if="plan.error">{{ plan.error.message }}</span>
                        </div>
                        <ul v-if="plan.changes?.length"><li v-for="change in plan.changes" :key="change">{{ changeLabel(change) }}</li></ul>
                        <p v-if="plan.affectedDevices?.length" class="warning">{{ $gettext('已有设备使用该虚拟网关，请先迁移这些设备。') }}（{{ plan.affectedDevices.length }}）</p>
                        <p v-if="plan.requiresDrill" class="hint">{{ $gettext('应用后建议在维护窗口执行受控切换演练。') }}</p>
                    </template>
                </section>

                <div class="wizard-actions">
                    <button v-if="configured && step === 1" type="button" class="secondary" :disabled="saving" @click="editing = false">{{ $gettext('取消') }}</button>
                    <button v-if="step > 1" type="button" class="secondary" :disabled="saving" @click="step--">{{ $gettext('上一步') }}</button>
                    <button v-if="step < 3" type="button" :disabled="saving" @click="next">{{ $gettext('下一步') }}</button>
                    <button v-else type="button" :disabled="saving || planning || !!plan?.error || !plan?.changed" @click="apply">{{ saving ? $gettext('正在应用…') : $gettext('应用配置') }}</button>
                </div>
            </template>
            <div v-if="feedback" class="feedback" :class="feedbackKind">{{ feedback }}</div>
            <details v-if="form.enabled" class="drill-box"><summary>{{ $gettext('故障切换演练') }}</summary><p>{{ $gettext('演练会短暂改变网关持有者，必须先完成双端配对校验。系统不会自动开始。') }}</p><button type="button" class="secondary" @click="loadDrillPlan">{{ $gettext('查看演练计划') }}</button><p v-if="drillMessage" class="hint">{{ drillMessage }}</p></details>
        </template>
    </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import PageState from './pageState.vue'

defineEmits<{ (event: 'install'): void }>()
const { $gettext } = useGettext()
const loading = ref(true), planning = ref(false), saving = ref(false), editing = ref(true)
const loadError = ref(''), feedback = ref(''), feedbackKind = ref<'success' | 'error'>('success'), drillMessage = ref('')
const step = ref(1), version = ref(''), plan = ref<any>(null), status = reactive<any>({ capability: '', state: 'disabled', holder: 'none', peerState: 'unverifiable', externalState: 'unverifiable' })
const form = reactive({ enabled: false, role: 'preferred', virtualIP: '', peerIPs: [] as string[], healthURL: '', healthTimeoutSec: 5 })
const peerText = computed({ get: () => form.peerIPs.join(', '), set: value => { form.peerIPs = value.split(/[,，\s]+/).map(item => item.trim()).filter(Boolean) } })
const configured = computed(() => Boolean(form.virtualIP || form.peerIPs.length))
const roleLabel = computed(() => form.role === 'takeover' ? $gettext('故障接管节点') : $gettext('优先服务节点'))
const statusLabel = computed(() => ({ disabled: $gettext('未启用'), starting: $gettext('等待状态确认'), healthy: $gettext('运行正常'), degraded: $gettext('部分异常'), failover: $gettext('正在接管'), error: $gettext('运行异常') } as Record<string, string>)[status.state] || $gettext('未知'))
const holderLabel = computed(() => ({ local: $gettext('当前节点'), peer: $gettext('对端节点'), none: $gettext('无人持有'), unknown: $gettext('等待确认') } as Record<string, string>)[status.holder] || $gettext('等待确认'))
const probeLabel = (value: string) => ({ reachable: $gettext('可达'), unreachable: $gettext('不可达'), healthy: $gettext('正常'), failed: $gettext('失败'), unverifiable: $gettext('尚未验证') }[value] || $gettext('尚未验证'))
const stepName = (value: number) => [$gettext('节点职责'), $gettext('地址与检测'), $gettext('预检与应用')][value - 1]
const changeLabel = (value: string) => ({ change_enabled_state: $gettext('更改启用状态'), change_node_responsibility: $gettext('更改节点职责'), change_virtual_ip: $gettext('更改虚拟 IP'), change_peer_nodes: $gettext('更改对端节点'), change_health_check: $gettext('更改健康检测') }[value] || value)
const payload = () => ({ config: { ...form }, expectedVersion: version.value })
const startEditing = () => { editing.value = true; step.value = 1; plan.value = null; feedback.value = '' }
const load = async () => {
    loading.value = true; loadError.value = ''
    try {
        const { data } = await request.DeviceMangement.floatingGatewayV2.GET()
        if (data.result?.error) throw new Error(data.result.error.message)
        Object.assign(status, data.result?.status || {})
        Object.assign(form, data.result?.config || {})
        version.value = data.result?.plan?.version || ''
        editing.value = !configured.value
    } catch (error: any) { loadError.value = error?.message || $gettext('读取结果失败') }
    finally { loading.value = false }
}
const next = async () => { if (step.value < 2) { step.value++; return }; step.value = 3; planning.value = true; plan.value = null; try { const { data } = await request.DeviceMangement.floatingGatewayV2.PLAN(payload()); plan.value = data.result?.plan; version.value = plan.value?.version || version.value } catch (error: any) { feedbackKind.value = 'error'; feedback.value = error?.message || $gettext('预检失败') } finally { planning.value = false } }
const apply = async () => { saving.value = true; feedback.value = ''; try { const { data } = await request.DeviceMangement.floatingGatewayV2.APPLY(payload()); if (data.result?.error || data.result?.plan?.error) throw new Error(data.result?.error?.message || data.result?.plan?.error?.message); feedbackKind.value = 'success'; feedback.value = data.result?.changed ? $gettext('配置已保存，请在双端完成后验证实际持有者') : $gettext('配置没有变化'); await load() } catch (error: any) { feedbackKind.value = 'error'; feedback.value = error?.message || $gettext('应用失败，原配置已恢复') } finally { saving.value = false } }
const loadDrillPlan = async () => { try { const { data } = await request.DeviceMangement.floatingGatewayV2.DRILL(); drillMessage.value = data.result?.error?.message || $gettext('演练计划已就绪') } catch (error: any) { drillMessage.value = error?.message || $gettext('无法生成演练计划') } }
load()
</script>

<style lang="scss" scoped>
.floating-wizard { max-width: 760px; color: var(--tit-color); }
.status-card { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 18px; }
.status-card > div { padding: 10px; border: 1px solid rgba(127,127,127,.16); border-radius: 8px; background: rgba(127,127,127,.04); }
.status-card small, .status-card strong { display: block; }.status-card small { margin-bottom: 4px; opacity: .65; }
.wizard-steps { display: grid; grid-template-columns: repeat(3, 1fr); padding: 0; margin: 0 0 16px; list-style: none; }.wizard-steps li { display: flex; align-items: center; gap: 7px; opacity: .5; }.wizard-steps li::after { content: ''; height: 1px; flex: 1; background: currentColor; opacity: .2; }.wizard-steps li:last-child::after { display: none; }.wizard-steps li.active,.wizard-steps li.done { color: #553afe; opacity: 1; }.wizard-steps b { display: grid; place-items: center; width: 24px; height: 24px; border: 1px solid currentColor; border-radius: 50%; }
.wizard-card,.capability-card,.drill-box,.configured-card { padding: 16px; border: 1px solid rgba(127,127,127,.2); border-radius: 9px; }.capability-card { display: flex; flex-direction: column; align-items: flex-start; gap: 9px; }.configured-card{display:grid;grid-template-columns:repeat(3,minmax(0,1fr)) auto;gap:10px;align-items:center}.configured-card>div{display:grid;gap:4px;min-width:0}.configured-card small{opacity:.65}.configured-card strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.toggle-line { display: flex; gap: 8px; align-items: center; }.role-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }.role-grid label { display: grid; grid-template-columns: auto 1fr; gap: 4px 8px; padding: 12px; border: 1px solid rgba(127,127,127,.2); border-radius: 8px; cursor: pointer; }.role-grid label.selected { border-color: #553afe; background: rgba(85,58,254,.06); }.role-grid input { grid-row: 1 / 3; }.role-grid small { opacity: .65; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }.form-grid label { display: flex; flex-direction: column; gap: 5px; }.form-grid input { min-width: 0; padding: 8px; color: inherit; background: transparent; border: 1px solid rgba(127,127,127,.3); border-radius: 6px; }.form-grid small,.hint { opacity: .65; }
.review-state { display: flex; flex-direction: column; gap: 4px; padding: 10px; border-radius: 7px; }.review-state.ready { color: #176b45; background: rgba(38,162,105,.1); }.review-state.error,.warning,.feedback.error { color: #9b3b16; background: #fff1e8; }.warning,.feedback { padding: 9px; border-radius: 7px; }.feedback.success { color: #176b45; background: rgba(38,162,105,.1); }
.wizard-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 14px; }.floating-wizard button { min-height: 36px; padding: 7px 14px; color: #fff; background: #553afe; border: 1px solid #553afe; border-radius: 6px; cursor: pointer; }.floating-wizard button.secondary { color: #553afe; background: transparent; }.floating-wizard button:disabled { opacity: .5; cursor: not-allowed; }.drill-box { margin-top: 16px; }.drill-box summary { color: #553afe; cursor: pointer; }
@media (max-width: 640px) { .status-card { grid-template-columns: 1fr 1fr; }.form-grid,.role-grid,.configured-card { grid-template-columns: 1fr; }.wizard-steps span { display: none; }.configured-card button{width:100%} }
</style>
