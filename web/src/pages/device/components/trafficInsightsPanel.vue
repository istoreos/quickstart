<template>
    <div class="insights-panel">
        <div class="insights-heading"><div><strong>{{ $gettext('历史用量') }}</strong><small>{{ $gettext('这是累计流量额度，不是速度上限或定时限速。') }}</small></div><button type="button" @click="quotaOpen = !quotaOpen">{{ quotaOpen ? $gettext('收起额度') : $gettext('设置额度') }}</button></div>
        <div class="range-tabs" role="tablist"><button v-for="option in ranges" :key="option.value" type="button" :class="{ active: range === option.value }" @click="selectRange(option.value)">{{ option.label }}</button></div>
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取历史用量')" />
        <p v-else-if="error" class="message error">{{ error }} <button type="button" @click="load">{{ $gettext('重试') }}</button></p>
        <template v-else>
            <div class="totals"><span><small>{{ $gettext('上传') }}</small>{{ bytes(result.uploadBytes) }}</span><span><small>{{ $gettext('下载') }}</small>{{ bytes(result.downloadBytes) }}</span><span><small>{{ $gettext('合计') }}</small>{{ bytes((result.uploadBytes || 0) + (result.downloadBytes || 0)) }}</span></div>
            <div v-if="result.buckets?.length" class="history-chart" :aria-label="$gettext('历史流量趋势')"><i v-for="bucket in result.buckets" :key="bucket.start" :style="{ height: barHeight(bucket) + '%' }" :title="`${new Date(bucket.start).toLocaleString()} · ${bytes(bucket.uploadBytes + bucket.downloadBytes)}`"></i></div>
            <p v-else class="muted">{{ $gettext('正在积累历史数据，首次使用后会按小时形成趋势。') }}</p>
            <div v-if="result.quota?.quota" class="quota-status" :class="{ exceeded: result.quota.exceeded }"><span>{{ quotaLabel(result.quota.quota.period) }} · {{ bytes(result.quota.usedBytes) }} / {{ bytes(result.quota.quota.limitBytes) }}</span><strong>{{ result.quota.exceeded ? $gettext('已达到额度') : quotaPercent + '%' }}</strong></div>
            <p class="capability">{{ $gettext('本地轻量统计') }} · {{ $gettext('每 5 分钟最多写入一次') }}<template v-if="result.bandix?.state !== 'not_installed'"> · Bandix: {{ capabilityLabel(result.bandix) }}</template></p>
        </template>
        <form v-if="quotaOpen" class="quota-form" @submit.prevent="saveQuota">
            <label class="enable"><input v-model="quota.enabled" type="checkbox" />{{ $gettext('启用流量额度') }}</label>
            <template v-if="quota.enabled"><label><span>{{ $gettext('周期') }}</span><select v-model="quota.period"><option value="daily">{{ $gettext('每天') }}</option><option value="weekly">{{ $gettext('每周') }}</option><option value="monthly">{{ $gettext('每月') }}</option></select></label><label><span>{{ $gettext('额度（GB）') }}</span><input v-model.number="quota.gigabytes" type="number" min="0.1" max="100000" step="0.1" required /></label><label><span>{{ $gettext('达到后') }}</span><select v-model="quota.action"><option value="notify">{{ $gettext('仅提醒') }}</option><option value="block">{{ $gettext('暂停联网') }}</option></select></label></template>
            <button class="save" type="submit" :disabled="saving">{{ saving ? $gettext('正在保存…') : $gettext('保存额度') }}</button>
            <p v-if="quotaFeedback" class="message" :class="quotaFeedbackKind">{{ quotaFeedback }}</p>
        </form>
    </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import PageState from './pageState.vue'
import { formatTrafficBytes } from '../deviceTelemetry'

const props = defineProps<{ deviceId: string }>()
const { $gettext } = useGettext()
const range = ref('today'), loading = ref(true), saving = ref(false), error = ref(''), result = ref<any>({}), quotaOpen = ref(false), quotaFeedback = ref(''), quotaFeedbackKind = ref<'success'|'error'>('success')
const quota = reactive({ enabled: false, period: 'monthly', gigabytes: 10, action: 'notify' })
const ranges = computed(() => [{ value: 'today', label: $gettext('今天') }, { value: 'week', label: $gettext('本周') }, { value: 'month', label: $gettext('本月') }])
const bytes = (value: number) => formatTrafficBytes(value || 0)
const quotaPercent = computed(() => Math.min(100, Math.round((result.value.quota?.usedBytes || 0) * 100 / Math.max(1, result.value.quota?.quota?.limitBytes || 1))))
const quotaLabel = (value: string) => ({ daily: $gettext('每日额度'), weekly: $gettext('每周额度'), monthly: $gettext('每月额度') } as Record<string,string>)[value] || value
const capabilityLabel = (value: any) => value.state === 'limited' ? $gettext('受硬件卸载限制') : value.state === 'available' ? $gettext('可用') : $gettext('未安装')
const barHeight = (bucket: any) => { const max = Math.max(1, ...(result.value.buckets || []).map((item: any) => item.uploadBytes + item.downloadBytes)); return Math.max(4, Math.round((bucket.uploadBytes + bucket.downloadBytes) * 100 / max)) }
const applyQuota = () => { const current = result.value.quota?.quota; quota.enabled = Boolean(current?.enabled); if (current) { quota.period = current.period; quota.gigabytes = Math.round(current.limitBytes / 1073741824 * 10) / 10; quota.action = current.action } }
const load = async () => { loading.value = true; error.value = ''; try { const response = await request.DeviceMangement.trafficInsightsV2.GET(props.deviceId, range.value); if (response.data.result?.error) throw new Error(response.data.result.error.message); result.value = response.data.result || {}; applyQuota() } catch (reason: any) { error.value = reason?.message || $gettext('历史用量读取失败') } finally { loading.value = false } }
const selectRange = (value: string) => { if (range.value === value) return; range.value = value; load() }
const saveQuota = async () => { saving.value = true; quotaFeedback.value = ''; try { const response = await request.DeviceMangement.trafficInsightsV2.QUOTA({ deviceId: props.deviceId, enabled: quota.enabled, period: quota.period, limitBytes: Math.round(quota.gigabytes * 1073741824), action: quota.action }); if (response.data.result?.error) throw new Error(response.data.result.error.message); quotaFeedbackKind.value = 'success'; quotaFeedback.value = $gettext('流量额度已保存'); await load() } catch (reason: any) { quotaFeedbackKind.value = 'error'; quotaFeedback.value = reason?.message || $gettext('保存失败') } finally { saving.value = false } }
load()
</script>

<style lang="scss" scoped>
.insights-panel { margin-top: 14px; padding-top: 13px; border-top: 1px solid rgba(127,127,127,.15); }.insights-heading { display: flex; justify-content: space-between; gap: 10px; align-items: flex-start; }.insights-heading > div { display: grid; gap: 3px; }.insights-heading small,.capability { opacity: .62; }.insights-panel button { padding: 5px 9px; color: #553afe; background: transparent; border: 1px solid rgba(85,58,254,.28); border-radius: 6px; cursor: pointer; }.range-tabs { display: flex; gap: 5px; margin: 12px 0; }.range-tabs button.active { color: #fff; background: #553afe; }.totals { display: grid; grid-template-columns: repeat(3,1fr); gap: 7px; }.totals span { display: grid; gap: 2px; padding: 8px; background: rgba(85,58,254,.05); border-radius: 7px; }.totals small { opacity: .6; }.history-chart { display: flex; align-items: end; gap: 3px; height: 80px; margin-top: 10px; padding: 6px; border-bottom: 1px solid rgba(127,127,127,.2); }.history-chart i { flex: 1; min-width: 3px; max-width: 18px; background: linear-gradient(#806cff,#553afe); border-radius: 3px 3px 0 0; }.quota-status { display: flex; justify-content: space-between; gap: 8px; margin-top: 10px; padding: 8px; color: #176b45; background: rgba(38,162,105,.1); border-radius: 7px; }.quota-status.exceeded { color: #9b3b16; background: #fff1e8; }.capability { margin: 9px 0 0; font-size: 11px; }.quota-form { display: grid; grid-template-columns: repeat(3,1fr); gap: 8px; margin-top: 10px; padding: 10px; background: rgba(85,58,254,.04); border-radius: 8px; }.quota-form label:not(.enable) { display: grid; gap: 4px; }.quota-form input,.quota-form select { min-width: 0; padding: 7px; color: inherit; background: transparent; border: 1px solid rgba(127,127,127,.25); border-radius: 6px; }.quota-form .enable,.quota-form .save,.quota-form .message { grid-column: 1 / -1; }.message { margin: 8px 0; }.message.error { color: #9b3b16; }.message.success { color: #176b45; }.muted { opacity: .6; }
@media (max-width: 520px) { .quota-form { grid-template-columns: 1fr; }.totals { grid-template-columns: 1fr; } }
</style>
