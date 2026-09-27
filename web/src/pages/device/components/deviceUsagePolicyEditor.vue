<template>
    <section class="usage-policy" aria-labelledby="usage-policy-title">
        <div class="policy-heading">
            <div><h5 id="usage-policy-title">{{ $gettext('计划与额度') }}</h5><p>{{ $gettext('只为这台设备设置休息时段和累计流量额度；单设备设置优先于分组和默认规则。') }}</p></div>
            <button type="button" :disabled="loading" @click="load">{{ $gettext('刷新') }}</button>
        </div>
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取使用规则')" />
        <p v-else-if="error" class="feedback error" role="alert">{{ error }} <button type="button" @click="load">{{ $gettext('重试') }}</button></p>
        <template v-else>
            <div class="effective-card">
                <span><small>{{ $gettext('当前生效') }}</small><strong>{{ effectiveLabel }}</strong></span>
                <span><small>{{ $gettext('规则来源') }}</small><strong>{{ sourceLabel }}</strong></span>
                <span><small>{{ $gettext('下次变化') }}</small><strong>{{ nextChangeLabel }}</strong></span>
            </div>
            <p class="timezone">{{ $gettext('按路由器时间执行') }} · {{ timezone || '—' }}</p>

            <fieldset>
                <legend>{{ $gettext('休息时段') }}</legend>
                <label class="check">
                    <input v-model="draft.scheduleEnabled" type="checkbox" />
                    {{ $gettext('在指定时段执行规则') }}
                </label>
                <div v-if="draft.scheduleEnabled" class="schedule-editor">
                    <label><span>{{ $gettext('时段内执行') }}</span><select v-model="draft.scheduleAction"><option value="block">{{ $gettext('暂停联网') }}</option><option value="limit">{{ $gettext('限制速度') }}</option></select></label>
                    <div v-if="draft.scheduleAction==='limit'" class="columns"><label><span>{{ $gettext('上传 Mbit/s') }}</span><input v-model.number="draft.scheduleUpload" type="number" min="1" /></label><label><span>{{ $gettext('下载 Mbit/s') }}</span><input v-model.number="draft.scheduleDownload" type="number" min="1" /></label></div>
                    <div class="days" role="group" :aria-label="$gettext('执行日期')"><label v-for="day in dayOptions" :key="day.value" :class="{ selected: draft.days.includes(day.value) }"><input v-model="draft.days" type="checkbox" :value="day.value" />{{ day.label }}</label></div>
                    <div class="columns"><label><span>{{ $gettext('开始') }}</span><input v-model="draft.start" type="time" /></label><label><span>{{ $gettext('结束') }}</span><input v-model="draft.end" type="time" /></label></div>
                    <small>{{ $gettext('结束时间早于开始时间时，会自动跨到第二天。') }}</small>
                </div>
            </fieldset>

            <fieldset>
                <legend>{{ $gettext('流量额度') }}</legend>
                <label class="check">
                    <input v-model="draft.quotaEnabled" type="checkbox" />
                    {{ quotaToggleLabel }}
                </label>
                <div v-if="draft.quotaEnabled" class="quota-editor"><label><span>{{ $gettext('周期') }}</span><select v-model="draft.quotaPeriod"><option value="daily">{{ $gettext('每天') }}</option><option value="weekly">{{ $gettext('每周') }}</option><option value="monthly">{{ $gettext('每月') }}</option></select></label><label><span>{{ $gettext('额度 GB') }}</span><input v-model.number="draft.quotaGB" type="number" min="0.1" max="100000" step="0.1" /></label><label><span>{{ $gettext('达到后') }}</span><select v-model="draft.quotaAction"><option value="notify">{{ $gettext('仅提醒') }}</option><option value="block">{{ $gettext('暂停联网') }}</option></select></label></div>
            </fieldset>

            <div v-if="previewing" class="preview" role="dialog" :aria-label="$gettext('保存前预览')">
                <strong>{{ $gettext('确认这次修改') }}</strong>
                <span>{{ previewLabel }}</span>
                <small>{{ $gettext('应用后会立即重新计算这台设备的最终规则；失败时保留原设置。') }}</small>
                <div><button type="button" @click="previewing=false">{{ $gettext('返回修改') }}</button><button class="primary" type="button" :disabled="saving" @click="save">{{ saving ? $gettext('正在应用…') : $gettext('确认并应用') }}</button></div>
            </div>
            <div v-else class="actions"><button class="primary" type="button" @click="requestPreview">{{ $gettext('预览影响') }}</button></div>
            <p v-if="feedback" class="feedback" :class="feedbackKind" role="status">{{ feedback }}</p>
        </template>
    </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import PageState from './pageState.vue'

const props = defineProps<{ deviceId: string }>()
const { $gettext } = useGettext()
const quotaToggleLabel = $gettext('限制累计用量')
const loading = ref(true), saving = ref(false), error = ref(''), feedback = ref(''), feedbackKind = ref<'success'|'error'>('success'), previewing = ref(false)
const version = ref(''), timezone = ref(''), policyBase = ref<Record<string, any>>({}), effective = ref<any>(null), groups = ref<any[]>([]), managedScheduleId = ref('device-rest')
const draft = reactive({ scheduleEnabled: false, scheduleAction: 'block', scheduleUpload: 20, scheduleDownload: 100, days: [1,2,3,4,5] as number[], start: '22:00', end: '07:00', quotaEnabled: false, quotaPeriod: 'monthly', quotaGB: 10, quotaAction: 'notify' })
const dayOptions = computed(() => [{value:1,label:$gettext('一')},{value:2,label:$gettext('二')},{value:3,label:$gettext('三')},{value:4,label:$gettext('四')},{value:5,label:$gettext('五')},{value:6,label:$gettext('六')},{value:0,label:$gettext('日')}])
const minutes = (value:string) => { const [hour, minute] = value.split(':').map(Number); return hour * 60 + minute }
const clock = (value:number) => `${String(Math.floor(value/60)).padStart(2,'0')}:${String(value%60).padStart(2,'0')}`
const formatDate = (value:string) => value ? new Date(value).toLocaleString([], { weekday:'short', hour:'2-digit', minute:'2-digit' }) : $gettext('暂无计划')
const effectiveLabel = computed(() => effective.value ? (effective.value.networkAccess ? $gettext('允许联网') : $gettext('已暂停联网')) : $gettext('系统默认'))
const sourceLabel = computed(() => {
    const sources:string[] = effective.value?.sources || []
    const labels = sources.filter(source => source !== 'system_default').map(source => source === 'global' ? $gettext('默认规则') : source === 'device' ? $gettext('本设备') : source.startsWith('group:') ? (groups.value.find(group => group.id === source.slice(6))?.name || $gettext('设备分组')) : source)
    return labels.join(' → ') || $gettext('系统默认')
})
const nextChangeLabel = computed(() => formatDate(effective.value?.nextScheduleAt || ''))
const previewLabel = computed(() => {
    const parts = [draft.scheduleEnabled ? `${draft.scheduleAction==='limit'?$gettext('时段限速'):$gettext('休息时段')} ${draft.start}–${draft.end}` : $gettext('不设置时段规则')]
    parts.push(draft.quotaEnabled ? `${draft.quotaGB} GB · ${quotaPeriodLabel(draft.quotaPeriod)}` : $gettext('不设置流量额度'))
    return parts.join('；')
})
const quotaPeriodLabel = (value:string) => ({daily:$gettext('每天'),weekly:$gettext('每周'),monthly:$gettext('每月')} as Record<string,string>)[value] || value

const applyResponse = (result:any) => {
    version.value = result?.version || ''
    timezone.value = result?.timezone || ''
    groups.value = result?.groups || []
    effective.value = (result?.effective || []).find((item:any) => item.deviceId === props.deviceId) || null
    const policy = result?.devicePolicies?.[props.deviceId] || {}
    policyBase.value = { ...policy, schedules: [...(policy.schedules || [])] }
    const schedule = policyBase.value.schedules.find((item:any) => item?.action === 'block' || item?.action === 'limit')
    managedScheduleId.value = schedule?.id || 'device-schedule'
    draft.scheduleEnabled = Boolean(schedule?.enabled)
    draft.days = schedule?.days?.length ? [...schedule.days] : [1,2,3,4,5]
    draft.start = schedule ? clock(schedule.startMinute) : '22:00'
    draft.end = schedule ? clock(schedule.endMinute) : '07:00'
    draft.scheduleAction = schedule?.action || 'block'
    draft.scheduleUpload = schedule?.speed?.uploadSpeed || 20
    draft.scheduleDownload = schedule?.speed?.downloadSpeed || 100
    draft.quotaEnabled = Boolean(policy.quota?.enabled)
    draft.quotaPeriod = policy.quota?.period || 'monthly'
    draft.quotaGB = policy.quota?.limitBytes ? Math.round(policy.quota.limitBytes / 1073741824 * 10) / 10 : 10
    draft.quotaAction = policy.quota?.action || 'notify'
}
const load = async () => { loading.value = true; error.value = ''; previewing.value = false; try { const result = (await request.DeviceMangement.deviceGroupsV2.GET()).data.result; if (result?.error) throw new Error(result.error.message); applyResponse(result) } catch (reason:any) { error.value = reason?.message || $gettext('读取使用规则失败') } finally { loading.value = false } }
const requestPreview = () => { feedback.value = ''; if (draft.scheduleEnabled && !draft.days.length) { feedbackKind.value='error'; feedback.value=$gettext('请至少选择一天'); return } if (draft.scheduleEnabled && (!draft.start || !draft.end)) { feedbackKind.value='error'; feedback.value=$gettext('请填写完整时间'); return } if (draft.scheduleEnabled && draft.scheduleAction==='limit' && (!draft.scheduleUpload || !draft.scheduleDownload)) { feedbackKind.value='error';feedback.value=$gettext('请填写完整的时段速度上限');return } if (draft.quotaEnabled && (!Number.isFinite(draft.quotaGB) || draft.quotaGB < .1)) { feedbackKind.value='error'; feedback.value=$gettext('流量额度至少为 0.1 GB'); return } previewing.value = true }
const buildPolicy = () => {
    const schedules = (policyBase.value.schedules || []).filter((item:any) => item?.id !== managedScheduleId.value)
    if (draft.scheduleEnabled) schedules.push({ id:managedScheduleId.value, enabled:true, days:[...draft.days], startMinute:minutes(draft.start), endMinute:minutes(draft.end), action:draft.scheduleAction, speed:draft.scheduleAction==='limit'?{enabled:true,uploadSpeed:draft.scheduleUpload,downloadSpeed:draft.scheduleDownload}:undefined })
    const next:any = { ...policyBase.value, schedules, quota:draft.quotaEnabled ? { enabled:true, period:draft.quotaPeriod, limitBytes:Math.round(draft.quotaGB*1073741824), action:draft.quotaAction } : undefined }
    if (next.access === undefined && next.speed === undefined && !next.targetId && !next.quota && !next.schedules.length) return null
    return next
}
const save = async () => { saving.value=true; feedback.value=''; try { const result=(await request.DeviceMangement.deviceGroupsV2.POST({action:'set_device_policy',deviceId:props.deviceId,devicePolicy:buildPolicy(),expectedVersion:version.value})).data.result; if(result?.error)throw new Error(result.error.message); applyResponse(result); previewing.value=false; feedbackKind.value='success'; feedback.value=$gettext('使用规则已应用') } catch(reason:any) { feedbackKind.value='error'; feedback.value=reason?.message||$gettext('保存失败，原设置未改变') } finally { saving.value=false } }
watch(() => props.deviceId, load, { immediate:true })
</script>

<style lang="scss" scoped>
.usage-policy{margin-top:16px;padding-top:14px;border-top:1px solid rgba(127,127,127,.15)}.policy-heading{display:flex;justify-content:space-between;gap:12px;align-items:flex-start}.policy-heading h5{margin:0;font-size:14px}.policy-heading p{margin:4px 0 0;opacity:.62}.usage-policy button{min-height:32px;padding:6px 10px;color:#553afe;background:transparent;border:1px solid rgba(85,58,254,.32);border-radius:7px;cursor:pointer}.effective-card{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:7px;margin:12px 0 4px}.effective-card span{display:grid;gap:3px;padding:9px;background:rgba(85,58,254,.05);border-radius:7px}.effective-card small,.timezone,.schedule-editor>small{opacity:.62}.timezone{margin:5px 0 12px;font-size:12px}.usage-policy fieldset{margin:10px 0;padding:11px;border:1px solid rgba(127,127,127,.17);border-radius:8px}.usage-policy legend{padding:0 5px;font-weight:600}.check{display:flex;align-items:center;gap:7px}.schedule-editor,.quota-editor{display:grid;gap:9px;margin-top:10px}.days{display:flex;flex-wrap:wrap;gap:5px}.days label{position:relative;padding:5px 8px;border:1px solid rgba(127,127,127,.22);border-radius:999px;cursor:pointer}.days label.selected{color:#553afe;background:rgba(85,58,254,.08)}.days input{position:absolute;opacity:0}.columns,.quota-editor{grid-template-columns:repeat(2,minmax(0,1fr))}.quota-editor{grid-template-columns:repeat(3,minmax(0,1fr))}.columns{display:grid;gap:8px}.columns label,.quota-editor label{display:grid;gap:4px;font-size:12px}.usage-policy input:not([type=checkbox]),.usage-policy select{box-sizing:border-box;width:100%;min-width:0;min-height:34px;padding:6px 8px;color:inherit;background:transparent;border:1px solid rgba(127,127,127,.26);border-radius:6px}.preview{display:grid;gap:6px;padding:11px;background:rgba(85,58,254,.06);border:1px solid rgba(85,58,254,.16);border-radius:8px}.preview>div,.actions{display:flex;justify-content:flex-end;gap:7px;margin-top:5px}.usage-policy .primary{color:#fff;background:#553afe;border-color:#553afe}.feedback{padding:8px;border-radius:7px}.feedback.error{color:#9b3b16;background:#fff1e8}.feedback.success{color:#176b45;background:rgba(38,162,105,.1)}button:focus-visible,input:focus-visible,select:focus-visible{outline:2px solid #553afe;outline-offset:2px}@media(max-width:420px){.effective-card,.columns,.quota-editor{grid-template-columns:1fr}.policy-heading{align-items:stretch}.policy-heading>button{flex:none}}
</style>
