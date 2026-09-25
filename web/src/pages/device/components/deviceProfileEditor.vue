<template>
    <div class="profile-editor">
        <PageState v-if="loading" kind="loading" :title="$gettext('正在读取设备资料')" />
        <PageState v-else-if="loadError" kind="error" :title="$gettext('设备资料读取失败')" :description="loadError" :action-label="$gettext('重新加载')" @action="load" />
        <template v-else-if="profile">
            <div class="profile-preview">
                <DeviceSceneIcon :scene="category" :icon-key="previewIcon" :label="iconLabel(previewIcon)" />
                <div><strong>{{ alias || device.hostname || $gettext('未命名设备') }}</strong><span><b v-if="effectiveBrand">{{ effectiveBrand }}</b><template v-if="effectiveBrand"> · </template>{{ sceneLabel(category) }}</span><small>{{ iconMode === 'manual' ? $gettext('已选择自定义图标') : $gettext('图标由品牌和类型自动推荐') }}</small></div>
            </div>

            <label><span>{{ $gettext('设备备注名') }}</span><input v-model="alias" maxlength="64" :placeholder="$gettext('例如：客厅电视')" /></label>
            <small>{{ $gettext('支持中文，只用于设备管理显示，不会写入网络主机名。') }}</small>
            <small v-if="scope === 'boot'" class="scope-note">{{ $gettext('该设备身份仅本次开机有效，重启后资料可能需要重新确认。') }}</small>

            <div class="two-columns">
                <label><span>{{ $gettext('品牌（可选）') }}</span><input v-model.trim="brand" maxlength="64" :placeholder="profile.classification?.brand || $gettext('例如：ASUS')" /></label>
                <label><span>{{ $gettext('设备类型') }}</span><select v-model="category"><option v-for="item in categories" :key="item" :value="item">{{ sceneLabel(item) }}</option></select></label>
            </div>

            <fieldset>
                <legend>{{ $gettext('设备图标') }}</legend>
                <label class="auto-icon"><input v-model="iconMode" type="radio" value="auto" />{{ $gettext('自动推荐') }} <small>{{ $gettext('优先依据品牌，其次依据设备类型') }}</small></label>
                <label class="auto-icon"><input v-model="iconMode" type="radio" value="manual" />{{ $gettext('自行选择') }}</label>
                <div v-if="iconMode === 'manual'" class="icon-grid" role="radiogroup" :aria-label="$gettext('选择设备图标')">
                    <label v-for="key in deviceIconKeys" :key="key" :class="{ selected: iconKey === key }" :title="iconLabel(key)">
                        <input v-model="iconKey" type="radio" name="device-icon" :value="key" />
                        <DeviceSceneIcon :scene="iconScene(key)" :icon-key="key" :label="iconLabel(key)" />
                        <span>{{ iconLabel(key) }}</span>
                    </label>
                </div>
            </fieldset>

            <p v-if="feedback" class="feedback" :class="feedbackKind" role="status">{{ feedback }}</p>
            <div class="actions">
                <button type="button" class="secondary" :disabled="saving" @click="restoreAutomatic">{{ $gettext('恢复全部自动推荐') }}</button>
                <button type="button" :disabled="saving || invalid" @click="save">{{ saving ? $gettext('正在保存…') : $gettext('保存设备资料') }}</button>
            </div>
        </template>
    </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request'
import PageState from './pageState.vue'
import DeviceSceneIcon from './deviceSceneIcon.vue'
import type { DeviceClassification, DeviceListItem } from '../deviceInventory'
import { editableDeviceCategories } from '../deviceClassification'
import { deviceIconKeys, deviceIconLabels, type DeviceIconKey, type DeviceScene } from '../deviceScene'
import { validDeviceAlias, type DeviceProfileResult } from '../deviceProfile'

const props = defineProps<{ device: DeviceListItem }>()
const emit = defineEmits<{ (event: 'saved', value: { alias: string; classification: DeviceClassification; iconKey: DeviceIconKey }): void }>()
const { $gettext } = useGettext()
const categories = editableDeviceCategories
const profile = ref<DeviceProfileResult['profile']>()
const loading = ref(true), saving = ref(false), loadError = ref(''), feedback = ref(''), feedbackKind = ref<'success'|'error'>('success')
const alias = ref(''), brand = ref(''), category = ref<DeviceScene>('computer'), iconMode = ref<'auto'|'manual'>('auto'), iconKey = ref<DeviceIconKey>('computer'), scope = ref<'persistent'|'boot'>('persistent')
const invalid = computed(() => !validDeviceAlias(alias.value) || Array.from(brand.value).length > 64)
const effectiveBrand = computed(() => brand.value || profile.value?.classification.brand || '')
const previewIcon = computed(() => iconMode.value === 'manual' ? iconKey.value : (profile.value?.icon?.resolvedKey || category.value) as DeviceIconKey)
const iconLabel = (key: DeviceIconKey) => $gettext(deviceIconLabels[key])
const sceneLabel = (scene: DeviceScene) => ({ phone: $gettext('手机'), computer: $gettext('电脑'), tablet: $gettext('平板'), tv: $gettext('电视与影音'), network: $gettext('网络设备'), 'smart-home': $gettext('智能家居'), camera: $gettext('摄像头'), gaming: $gettext('游戏设备'), storage: $gettext('存储与服务器'), printer: $gettext('打印机'), wearable: $gettext('穿戴设备'), unknown: $gettext('未知设备') }[scene])
const iconScene = (key: DeviceIconKey): DeviceScene => ({ desktop: 'computer', laptop: 'computer', 'smart-speaker': 'smart-home', 'smart-bulb': 'smart-home', thermostat: 'smart-home', sensor: 'smart-home', 'door-lock': 'smart-home', 'robot-vacuum': 'smart-home', 'air-conditioner': 'smart-home', projector: 'tv', 'set-top-box': 'tv', 'game-console': 'gaming', 'handheld-game': 'gaming', 'home-server': 'storage', 'network-switch': 'network', 'access-point': 'network', 'network-bridge': 'network', 'e-reader': 'tablet' } as Partial<Record<DeviceIconKey, DeviceScene>>)[key] || key as DeviceScene

const applyResult = (result: DeviceProfileResult) => {
    if (result.error) throw Object.assign(new Error(result.error.message), { code: result.error.code })
    if (!result.profile) throw new Error($gettext('设备资料不可用'))
    profile.value = result.profile
    alias.value = result.profile.alias || ''
    brand.value = result.profile.manualBrand || ''
    category.value = (result.profile.manualCategory || result.profile.classification.category) as DeviceScene
    iconMode.value = result.profile.icon.mode
    iconKey.value = (result.profile.icon.preferenceKey || result.profile.icon.resolvedKey || 'computer') as DeviceIconKey
    scope.value = result.profile.scope
    emit('saved', { alias: alias.value, classification: result.profile.classification as DeviceClassification, iconKey: result.profile.icon.assetKey as DeviceIconKey })
}
const load = async () => {
    loading.value = true; loadError.value = ''; feedback.value = ''
    try { applyResult((await request.DeviceMangement.deviceProfileV2.GET(props.device.deviceId)).data?.result as DeviceProfileResult) }
    catch (error: any) { loadError.value = error?.message || $gettext('读取设备资料失败') }
    finally { loading.value = false }
}
const post = async (body: Record<string, unknown>, success: string) => {
    saving.value = true; feedback.value = ''
    try {
        const response = await request.DeviceMangement.deviceProfileV2.POST({ deviceId: props.device.deviceId, idempotencyKey: `${props.device.deviceId}:profile:${Date.now()}`, ...body } as any)
        const result = response.data?.result as DeviceProfileResult
        applyResult(result)
        feedbackKind.value = 'success'; feedback.value = result.changed ? success : $gettext('设置没有变化')
    } catch (error: any) { feedbackKind.value = 'error'; feedback.value = error?.message || $gettext('保存失败，请稍后重试') }
    finally { saving.value = false }
}
const save = () => { if (!invalid.value) post({ action: 'patch', alias: alias.value.trim(), brand: brand.value.trim(), ...(profile.value?.manualCategory || category.value !== profile.value?.classification.category ? { category: category.value } : {}), iconMode: iconMode.value, ...(iconMode.value === 'manual' ? { iconKey: iconKey.value } : {}) }, $gettext('设备资料已更新')) }
const restoreAutomatic = () => post({ action: 'reset' }, $gettext('已恢复系统自动推荐'))
watch(() => props.device.deviceId, load, { immediate: true })
</script>

<style lang="scss" scoped>
.profile-editor { display: grid; gap: 11px; }.profile-preview { display: flex; align-items: center; gap: 11px; padding: 11px; background: rgba(85,58,254,.05); border-radius: 9px; }.profile-preview > div { display: grid; gap: 2px; }.profile-preview span,.profile-preview small,.profile-editor > small { opacity: .65; }.profile-preview b { color: #553afe; }.profile-editor > label,.two-columns label { display: grid; gap: 5px; font-size: 12px; }.profile-editor input:not([type=radio]),.profile-editor select { box-sizing: border-box; width: 100%; min-width: 0; min-height: 36px; padding: 7px 9px; color: inherit; background: transparent; border: 1px solid rgba(127,127,127,.3); border-radius: 7px; }.two-columns { display: grid; grid-template-columns: 1fr 1fr; gap: 9px; }.scope-note { padding: 8px; background: rgba(127,127,127,.07); border-radius: 6px; }fieldset { margin: 0; padding: 10px; border: 1px solid rgba(127,127,127,.18); border-radius: 8px; }legend { padding: 0 5px; font-weight: 600; }.auto-icon { display: flex; align-items: center; gap: 7px; margin: 5px 0; }.auto-icon small { opacity: .6; }.icon-grid { display: grid; grid-template-columns: repeat(5,minmax(0,1fr)); gap: 6px; max-height: 250px; margin-top: 9px; overflow-y: auto; }.icon-grid label { position: relative; display: grid; justify-items: center; gap: 4px; min-width: 0; padding: 7px 3px; border: 1px solid rgba(127,127,127,.15); border-radius: 8px; cursor: pointer; }.icon-grid label.selected { color: #553afe; background: rgba(85,58,254,.07); border-color: #806cff; }.icon-grid input { position: absolute; opacity: 0; }.icon-grid span { width: 100%; overflow: hidden; font-size: 10px; text-align: center; text-overflow: ellipsis; white-space: nowrap; }.icon-grid :deep(.device-scene-icon) { width: 34px; height: 34px; }.icon-grid :deep(img) { width: 31px; height: 31px; }.feedback { margin: 0; padding: 8px; border-radius: 7px; }.feedback.success { color: #176b45; background: rgba(38,162,105,.1); }.feedback.error { color: #9b3b16; background: #fff1e8; }.actions { display: flex; justify-content: flex-end; gap: 8px; }.actions button { min-height: 36px; padding: 7px 11px; color: #fff; background: #553afe; border: 1px solid #553afe; border-radius: 7px; cursor: pointer; }.actions .secondary { color: #553afe; background: transparent; border-color: rgba(85,58,254,.35); }button:focus-visible,input:focus-visible,select:focus-visible,label:focus-within { outline: 2px solid #553afe; outline-offset: 2px; }button:disabled { opacity: .55; }
@media(max-width:520px){.two-columns{grid-template-columns:1fr}.icon-grid{grid-template-columns:repeat(3,minmax(0,1fr))}.actions{display:grid}.actions button{width:100%;white-space:normal}}
</style>
