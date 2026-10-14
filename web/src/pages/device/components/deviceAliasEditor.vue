<template>
    <div class="alias-editor">
        <label>
            <span>{{ $gettext('设备备注名') }}</span>
            <input v-model="alias" maxlength="64" :placeholder="$gettext('例如：客厅电视')"
                :disabled="loading || saving" @keydown.enter.prevent="save" />
        </label>
        <small>{{ $gettext('支持中文，仅用于本页面显示，不会修改 DHCP 主机名。') }}</small>
        <small v-if="scope === 'boot'" class="scope-note">{{ $gettext('该设备身份仅本次开机有效，备注会在路由器重启后清除。') }}</small>
        <small v-if="invalid" class="error" role="alert">{{ $gettext('备注名最多 64 个字符，不能包含控制字符。') }}</small>
        <p v-if="feedback" :class="feedbackKind" role="status">{{ feedback }}</p>
        <div class="actions">
            <button v-if="currentAlias" type="button" class="secondary" :disabled="loading || saving" @click="clearAlias">{{ $gettext('清除备注') }}</button>
            <button type="button" :disabled="loading || saving || invalid || alias.trim() === currentAlias" @click="save">
                {{ saving ? $gettext('正在保存…') : $gettext('保存备注') }}
            </button>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import type { DeviceListItem } from '../deviceInventory'
import { normalizeDeviceAlias, validDeviceAlias, type DeviceProfileResult } from '../deviceProfile'

const props = defineProps<{ device: DeviceListItem }>()
const emit = defineEmits<{ (event: 'saved', alias: string): void }>()
const { $gettext } = useGettext()
const alias = ref('')
const currentAlias = ref('')
const scope = ref<'persistent' | 'boot'>('persistent')
const loading = ref(false)
const saving = ref(false)
const feedback = ref('')
const feedbackKind = ref<'success' | 'error'>('success')
const invalid = computed(() => !validDeviceAlias(alias.value))

const applyResult = (result: DeviceProfileResult) => {
    if (result.error) throw Object.assign(new Error(result.error.message), { code: result.error.code })
    if (!result.profile) throw new Error($gettext('设备资料不可用'))
    currentAlias.value = result.profile.alias || ''
    alias.value = currentAlias.value
    scope.value = result.profile.scope
    emit('saved', currentAlias.value)
}
const load = async () => {
    loading.value = true
    feedback.value = ''
    try {
        const response = await request.DeviceMangement.deviceProfileV2.GET(props.device.deviceId)
        applyResult(response.data?.result as DeviceProfileResult)
    } catch (error: any) {
        feedbackKind.value = 'error'
        feedback.value = error?.message || $gettext('读取设备资料失败')
    } finally {
        loading.value = false
    }
}
const save = async () => {
    if (invalid.value) return
    saving.value = true
    feedback.value = ''
    try {
        const response = await request.DeviceMangement.deviceProfileV2.POST({
            deviceId: props.device.deviceId,
            idempotencyKey: `${props.device.deviceId}:profile:${Date.now()}`,
            action: 'patch',
            alias: normalizeDeviceAlias(alias.value),
        })
        const result = response.data?.result as DeviceProfileResult
        applyResult(result)
        feedbackKind.value = 'success'
        feedback.value = result.changed ? $gettext('设备备注已更新') : $gettext('设置没有变化')
    } catch (error: any) {
        feedbackKind.value = 'error'
        feedback.value = error?.message || $gettext('保存失败，请稍后重试')
    } finally {
        saving.value = false
    }
}
const clearAlias = () => { alias.value = ''; save() }

watch(() => props.device.deviceId, load, { immediate: true })
</script>

<style lang="scss" scoped>
.alias-editor { display: flex; flex-direction: column; gap: 7px; min-width: 0; margin-bottom: 16px; padding-bottom: 16px; overflow: hidden; border-bottom: 1px solid rgba(127, 127, 127, .14); }
label { display: flex; flex-direction: column; gap: 6px; font-size: 12px; }
input { box-sizing: border-box; width: 100%; min-width: 0; min-height: 36px; padding: 7px 9px; color: inherit; background: transparent; border: 1px solid rgba(127, 127, 127, .3); border-radius: 6px; }
small { opacity: .65; }
.scope-note { padding: 8px; background: rgba(127, 127, 127, .07); border-radius: 6px; }
.error, p.error { color: #9b3b16; opacity: 1; }
p { margin: 0; padding: 7px 9px; border-radius: 6px; font-size: 12px; }
p.success { color: #176b45; background: rgba(38, 162, 105, .1); }
.actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; }
button { min-height: 34px; padding: 6px 11px; color: #fff; background: #553afe; border: 1px solid #553afe; border-radius: 6px; cursor: pointer; }
button.secondary { color: #553afe; background: transparent; border-color: rgba(85, 58, 254, .42); }
button:disabled { cursor: not-allowed; opacity: .55; }
@media (max-width: 420px) { .actions { display: grid; grid-template-columns: minmax(0, 1fr); } button { width: 100%; white-space: normal; } }
</style>
