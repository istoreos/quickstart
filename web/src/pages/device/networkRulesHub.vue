<template>
    <section class="rules-hub" aria-labelledby="network-rules-title">
        <div class="rules-header"><div><h2 id="network-rules-title">{{ $gettext('规则台账') }}</h2><p>{{ $gettext('集中检查离线、孤立或需要修复的规则；新设置请从对应设备详情进入。') }}</p></div></div>
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取网络规则')" />
        <PageState v-else-if="loadError" kind="error" :title="$gettext('网络规则读取失败')" :description="loadError" :action-label="$gettext('重新加载')" @action="load" />
        <template v-else>
            <div class="rule-toolbar">
                <input v-model.trim="query" type="search" :placeholder="$gettext('搜索设备、MAC、IP 或规则')" />
                <select v-model="kind"><option value="">{{ $gettext('全部类型') }}</option><option value="static">{{ $gettext('地址预留') }}</option><option value="route">{{ $gettext('上网路线') }}</option><option value="speed">{{ $gettext('设备限速') }}</option><option value="access">{{ $gettext('联网权限') }}</option></select>
                <select v-model="status"><option value="">{{ $gettext('全部状态') }}</option><option value="active">{{ $gettext('正常') }}</option><option value="orphaned">{{ $gettext('离线或孤立') }}</option><option value="unreachable">{{ $gettext('网关不可达') }}</option><option value="unsupported">{{ $gettext('需要修复') }}</option></select>
            </div>
            <div class="selection-bar" v-if="selected.size"><span>{{ $gettext('已选择') }} {{ selected.size }} / 128</span><button type="button" class="danger" :disabled="applying" @click="removeSelected">{{ applying ? $gettext('正在应用…') : $gettext('删除所选规则') }}</button></div>
            <PageState v-if="!filtered.length" kind="empty" :title="$gettext('没有匹配的网络规则')" :description="$gettext('可从设备详情添加地址、路线和使用限制。')" />
            <div v-else class="rules-table-wrap"><table><thead><tr><th><input type="checkbox" :checked="allVisibleSelected" @change="toggleVisible" :aria-label="$gettext('选择当前结果')" /></th><th>{{ $gettext('设备') }}</th><th>{{ $gettext('规则类型') }}</th><th>{{ $gettext('最终设置') }}</th><th>{{ $gettext('状态') }}</th></tr></thead><tbody><tr v-for="rule in filtered" :key="rule.id" :class="{ orphaned: rule.orphaned }"><td><input type="checkbox" :checked="selected.has(rule.id)" @change="toggle(rule.id)" :aria-label="`${$gettext('选择')} ${rule.displayName}`" /></td><td><strong>{{ rule.displayName || $gettext('未命名设备') }}</strong><small>{{ rule.mac || rule.ip || '—' }}</small></td><td><span class="kind-pill">{{ kindLabel(rule.kind) }}</span></td><td>{{ ruleSummary(rule.summary) }}<small v-if="rule.nextAction" class="next-action">{{ nextActionLabel(rule.nextAction) }}</small></td><td><span class="state-pill" :class="rule.status">{{ statusLabel(rule.status) }}</span><button v-if="repairable(rule)" type="button" class="repair" @click="repairRule(rule)">{{ $gettext('恢复默认路线') }}</button></td></tr></tbody></table></div>
            <div v-if="filtered.length" class="rule-cards">
                <article v-for="rule in filtered" :key="`card-${rule.id}`" :class="{ orphaned: rule.orphaned }">
                    <label class="rule-select"><input type="checkbox" :checked="selected.has(rule.id)" @change="toggle(rule.id)" :aria-label="`${$gettext('选择')} ${rule.displayName}`" /></label>
                    <div class="rule-identity"><strong>{{ rule.displayName || $gettext('未命名设备') }}</strong><small>{{ rule.mac || rule.ip || '—' }}</small></div>
                    <span class="state-pill" :class="rule.status">{{ statusLabel(rule.status) }}</span>
                    <div class="rule-result"><span class="kind-pill">{{ kindLabel(rule.kind) }}</span><span>{{ ruleSummary(rule.summary) }}</span></div><p v-if="rule.nextAction" class="next-action">{{ nextActionLabel(rule.nextAction) }}</p><button v-if="repairable(rule)" type="button" class="repair" @click="repairRule(rule)">{{ $gettext('恢复默认路线') }}</button>
                </article>
            </div>
            <p v-if="feedback" class="feedback" :class="feedbackKind">{{ feedback }}</p>
        </template>
    </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import PageState from './components/pageState.vue'
const { $gettext } = useGettext()
const loading = ref(true), applying = ref(false), loadError = ref(''), query = ref(''), kind = ref(''), status = ref(''), version = ref(''), feedback = ref(''), feedbackKind = ref<'success'|'error'>('success')
const rules = ref<any[]>([]), selected = ref(new Set<string>())
const filtered = computed(() => { const needle = query.value.toLowerCase(); return rules.value.filter(rule => (!kind.value || rule.kind === kind.value) && (!status.value || rule.status === status.value) && (!needle || [rule.displayName, rule.mac, rule.ip, rule.summary].some(value => String(value || '').toLowerCase().includes(needle)))) })
const allVisibleSelected = computed(() => filtered.value.length > 0 && filtered.value.every(rule => selected.value.has(rule.id)))
const kindLabel = (value: string) => ({ static: $gettext('地址预留'), route: $gettext('上网路线'), speed: $gettext('设备限速'), access: $gettext('联网权限') } as Record<string,string>)[value] || value
const statusLabel = (value: string) => ({ active: $gettext('正常'), orphaned: $gettext('离线或孤立'), unreachable: $gettext('网关不可达'), unsupported: $gettext('需要修复') } as Record<string,string>)[value] || value
const nextActionLabel = (value:string) => ({
    restore_default_or_choose: $gettext('恢复默认路线，或在设备详情中选择其他可用路线'),
    check_gateway_or_restore_default: $gettext('确认网关设备已开机并接入当前局域网；仍不可达时恢复默认路线'),
    restore_default_or_edit: $gettext('恢复默认路线，或编辑路线后重新应用'),
} as Record<string,string>)[value] || $gettext('请在设备详情中检查这条路线')
const ruleSummary = (value: string) => {
    let result = String(value || '—')
    for (const [source, label] of [['跟随网络默认', $gettext('默认路线')], ['本机路由', $gettext('本机路由')], ['旁路由', $gettext('旁路由')], ['浮动网关', $gettext('浮动网关')], ['自定义网关', $gettext('自定义网关')]]) result = result.replace(source, label)
    return result
}
const toggle = (id: string) => { const next = new Set(selected.value); next.has(id) ? next.delete(id) : next.size < 128 && next.add(id); selected.value = next }
const toggleVisible = () => { const next = new Set(selected.value), remove = allVisibleSelected.value; for (const rule of filtered.value) { if (remove) next.delete(rule.id); else if (next.size < 128) next.add(rule.id) }; selected.value = next }
const load = async () => { loading.value = true; loadError.value = ''; try { const { data } = await request.DeviceMangement.networkRulesV2.GET(); if (data.result?.error) throw new Error(data.result.error.message); rules.value = data.result?.rules || []; version.value = data.result?.version || ''; selected.value = new Set([...selected.value].filter(id => rules.value.some(rule => rule.id === id))) } catch (error: any) { loadError.value = error?.message || $gettext('读取结果失败') } finally { loading.value = false } }
const applyRemoval = async (ruleIds:string[], question:string) => { feedback.value = ''; const requestBody = { action: 'delete', ruleIds, expectedVersion: version.value }; try { const preview = (await request.DeviceMangement.networkRulesV2.PLAN(requestBody)).data.result?.plan; if (preview?.error) throw new Error(preview.error.message); if (!window.confirm(`${question}\n${$gettext('影响规则')}：${preview.affectedCount}\n${$gettext('失败时将恢复')}：${preview.rollbackScope.join('、')}`)) return; applying.value = true; const result = (await request.DeviceMangement.networkRulesV2.APPLY(requestBody)).data.result; if (result?.error) throw new Error(result.error.message); feedbackKind.value = 'success'; feedback.value = $gettext('规则已删除'); selected.value = new Set(); await load() } catch (error: any) { feedbackKind.value = 'error'; feedback.value = error?.message || $gettext('删除失败，原配置已恢复') } finally { applying.value = false } }
const removeSelected = () => applyRemoval([...selected.value], $gettext('确认删除所选规则？'))
const repairable = (rule:any) => rule.kind === 'route' && ['unreachable','unsupported'].includes(rule.status)
const repairRule = (rule:any) => applyRemoval([rule.id], $gettext('恢复默认路线？设备重新获取地址后生效。'))
load()
</script>

<style lang="scss" scoped>
.rules-hub { color: var(--tit-color); }.rules-header { display: flex; justify-content: space-between; gap: 16px; align-items: flex-start; }.rules-hub h2 { margin: 0; padding: 0; color: inherit; background: none !important; font-size: 20px; text-align: left; }.rules-header p { margin: 5px 0 0; opacity: .65; }.rules-hub button { min-height: 34px; padding: 6px 12px; border-radius: 6px; cursor: pointer; }.secondary { color: #553afe; background: transparent; border: 1px solid rgba(85,58,254,.4); }.danger { color: #fff; background: #b94343; border: 1px solid #b94343; }
.rule-toolbar { display: grid; grid-template-columns: minmax(220px,1fr) 150px 150px; gap: 9px; margin: 18px 0 10px; }.rule-toolbar input,.rule-toolbar select { min-width: 0; min-height: 36px; padding: 7px 9px; color: inherit; background: transparent; border: 1px solid rgba(127,127,127,.25); border-radius: 7px; }.selection-bar { display: flex; justify-content: space-between; align-items: center; padding: 8px 10px; margin-bottom: 8px; background: rgba(85,58,254,.06); border-radius: 7px; }
.rules-table-wrap { overflow-x: auto; }table { width: 100%; border-collapse: collapse; }th,td { padding: 10px 8px; text-align: left; border-bottom: 1px solid rgba(127,127,127,.12); }td strong,td small { display: block; }td small { margin-top: 3px; opacity: .6; }.orphaned { background: rgba(185,67,67,.035); }.kind-pill,.state-pill { display: inline-block; padding: 3px 7px; border-radius: 999px; background: rgba(85,58,254,.08); }.state-pill.orphaned,.state-pill.unsupported,.state-pill.unreachable { color:#9b3b16;background:#fff1e8 }.next-action{display:block;margin:4px 0 0;max-width:420px;color:#8a5a00;font-size:12px}.rules-hub button.repair{display:block;margin-top:5px;padding:3px 7px;min-height:28px;color:#553afe;background:transparent;border:1px solid rgba(85,58,254,.35)}.rule-cards{display:none}.feedback { padding: 9px; border-radius: 7px; }.feedback.success { color: #176b45; background: rgba(38, 162, 105, .1); }.feedback.error { color: #9b3b16; background: #fff1e8; }
@media (max-width: 640px) { .rules-header { flex-direction: column; }.rule-toolbar { grid-template-columns: 1fr 1fr; }.rule-toolbar input { grid-column: 1 / 3; }.rules-table-wrap{display:none}.rule-cards{display:grid;gap:8px}.rule-cards article{display:grid;grid-template-columns:40px minmax(0,1fr) auto;gap:7px;align-items:center;padding:10px;border:1px solid rgba(127,127,127,.15);border-radius:8px}.rule-select{display:grid;place-items:center;width:40px;height:40px}.rule-identity{display:grid;min-width:0}.rule-identity strong,.rule-identity small{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.rule-identity small{opacity:.62}.rule-result,.rule-cards .next-action,.rule-cards .repair{grid-column:2/-1}.rule-result{display:flex;align-items:center;gap:7px;min-width:0}.rule-result>span:last-child{overflow-wrap:anywhere} }
</style>
