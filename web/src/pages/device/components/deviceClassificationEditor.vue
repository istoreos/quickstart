<template>
    <div class="classification-editor">
        <div class="classification-summary">
            <div>
                <strong>{{ sceneLabel(current.category) }}</strong>
                <small>{{ override.active ? $gettext('已手动设置') : $gettext('由系统自动判断') }}</small>
            </div>
            <button v-if="!editing" type="button" class="link-button" :disabled="loading || saving" @click="beginEdit">
                {{ $gettext('修改设备类型') }}
            </button>
        </div>

        <p v-if="scope === 'boot'" class="scope-note">{{ $gettext('该设备身份仅本次开机有效，修改会在路由器重启后自动清除。') }}</p>
        <p v-if="feedback" class="classification-feedback" :class="feedbackKind" role="status">{{ feedback }}</p>

        <div v-if="editing" class="classification-form">
            <div class="category-grid" role="radiogroup" :aria-label="$gettext('选择设备类型')">
                <label v-for="category in editableDeviceCategories" :key="category" :class="{ selected: draft === category }">
                    <input v-model="draft" type="radio" name="device-category" :value="category" />
                    <DeviceSceneIcon :scene="category" :label="sceneLabel(category)" />
                    <span>{{ sceneLabel(category) }}</span>
                </label>
            </div>
            <div class="editor-actions">
                <button type="button" class="secondary" :disabled="saving" @click="cancelEdit">{{ $gettext('取消') }}</button>
                <button type="button" :disabled="saving || !draft" @click="save">{{ saving ? $gettext('正在保存…') : $gettext('保存类型') }}</button>
            </div>
        </div>
        <button v-if="override.active && !editing" type="button" class="reset-button" :disabled="saving" @click="reset">
            {{ saving ? $gettext('正在恢复…') : $gettext('恢复自动判断') }}
        </button>
    </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import type { DeviceClassification, DeviceListItem } from '../deviceInventory'
import type { DeviceScene } from '../deviceScene'
import { editableDeviceCategories, type DeviceClassificationResult } from '../deviceClassification'
import DeviceSceneIcon from './deviceSceneIcon.vue'

const props = defineProps<{ device: DeviceListItem }>()
const emit = defineEmits<{ (event: 'saved', classification: DeviceClassification): void }>()
const { $gettext } = useGettext()

const current = ref<DeviceClassification>(props.device.classification)
const override = ref({ active: props.device.classification.source === 'manual', category: props.device.scene })
const scope = ref<'persistent' | 'boot'>(props.device.identity?.scope === 'boot' ? 'boot' : 'persistent')
const draft = ref<DeviceScene>(props.device.scene)
const loading = ref(false)
const saving = ref(false)
const editing = ref(false)
const feedback = ref('')
const feedbackKind = ref<'success' | 'error'>('success')

const sceneLabel = (scene: DeviceScene) => ({
    phone: $gettext('手机'), computer: $gettext('电脑'), tablet: $gettext('平板'), tv: $gettext('电视与影音'),
    network: $gettext('网络设备'), 'smart-home': $gettext('智能家居'), camera: $gettext('摄像头'),
    gaming: $gettext('游戏设备'), storage: $gettext('存储与服务器'), printer: $gettext('打印机'),
    wearable: $gettext('穿戴设备'), unknown: $gettext('未知设备'),
}[scene])
const errorLabel = (code?: string, fallback?: string) => ({
    not_found: $gettext('设备已离线或身份发生变化，请刷新后重试'),
    validation_failed: $gettext('请选择有效的设备类型'),
    write_failed: $gettext('保存失败，请检查设备存储空间后重试'),
}[code || ''] || fallback || $gettext('操作失败，请稍后重试'))

const applyResult = (result: DeviceClassificationResult) => {
    if (result.error) throw Object.assign(new Error(result.error.message), { code: result.error.code })
    current.value = result.classification
    override.value = result.override
    scope.value = result.override.scope
    draft.value = result.classification.category
    emit('saved', result.classification)
}
const load = async () => {
    loading.value = true
    feedback.value = ''
    try {
        const response = await request.DeviceMangement.deviceClassificationV2.GET(props.device.deviceId)
        const result = response.data?.result as DeviceClassificationResult | undefined
        if (!result) throw new Error($gettext('设备类型不可用'))
        applyResult(result)
    } catch (error: any) {
        feedbackKind.value = 'error'
        feedback.value = errorLabel(error?.code, error?.message)
    } finally {
        loading.value = false
    }
}
const beginEdit = () => {
    draft.value = current.value.category
    feedback.value = ''
    editing.value = true
}
const cancelEdit = () => {
    draft.value = current.value.category
    feedback.value = ''
    editing.value = false
}
const apply = async (action: 'set' | 'reset') => {
    saving.value = true
    feedback.value = ''
    try {
        const response = await request.DeviceMangement.deviceClassificationV2.POST({
            deviceId: props.device.deviceId,
            action,
            ...(action === 'set' ? { category: draft.value } : {}),
        })
        const result = response.data?.result as DeviceClassificationResult | undefined
        if (!result) throw new Error($gettext('设备类型不可用'))
        applyResult(result)
        editing.value = false
        feedbackKind.value = 'success'
        feedback.value = result.changed ? (action === 'set' ? $gettext('设备类型已更新') : $gettext('已恢复自动判断')) : $gettext('设置没有变化')
    } catch (error: any) {
        feedbackKind.value = 'error'
        feedback.value = errorLabel(error?.code, error?.message)
    } finally {
        saving.value = false
    }
}
const save = () => apply('set')
const reset = () => apply('reset')

watch(() => props.device.deviceId, load, { immediate: true })
</script>

<style lang="scss" scoped>
.classification-editor { display: flex; flex-direction: column; gap: 10px; }
.classification-summary { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.classification-summary > div { display: flex; flex-direction: column; gap: 3px; }
.classification-summary small { opacity: .62; }
button { min-height: 34px; padding: 6px 11px; color: #fff; background: #553afe; border: 1px solid #553afe; border-radius: 6px; cursor: pointer; }
button:disabled { cursor: not-allowed; opacity: .55; }
.link-button, .reset-button, .editor-actions .secondary { color: #553afe; background: transparent; border-color: rgba(85, 58, 254, .42); }
.reset-button { align-self: flex-start; }
.scope-note, .classification-feedback { margin: 0; padding: 8px 10px; border-radius: 6px; font-size: 12px; }
.scope-note { background: rgba(127, 127, 127, .07); opacity: .76; }
.classification-feedback.success { color: #176b45; background: rgba(38, 162, 105, .1); }
.classification-feedback.error { color: #9b3b16; background: #fff1e8; }
.classification-form { display: flex; flex-direction: column; gap: 12px; }
.category-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 7px; }
.category-grid label { position: relative; display: flex; flex-direction: column; align-items: center; gap: 5px; padding: 9px 4px; border: 1px solid rgba(127, 127, 127, .18); border-radius: 8px; cursor: pointer; font-size: 12px; text-align: center; }
.category-grid label.selected { color: #553afe; background: rgba(85, 58, 254, .07); border-color: rgba(85, 58, 254, .5); }
.category-grid label:focus-within { outline: 2px solid #553afe !important; outline-offset: 2px; box-shadow: 0 0 0 3px rgba(85, 58, 254, .14); }
.category-grid input { position: absolute; opacity: 0; pointer-events: none; }
.category-grid :deep(.device-scene-icon) { width: 38px; height: 38px; }
.editor-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
@media (max-width: 360px) { .category-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
