import type { DeviceClassification } from './deviceInventory'
import type { DeviceScene } from './deviceScene'

export const editableDeviceCategories: DeviceScene[] = [
    'computer', 'phone', 'tablet', 'tv', 'network', 'smart-home',
    'camera', 'gaming', 'storage', 'printer', 'wearable',
]

export type DeviceClassificationResult = {
    deviceId: string
    classification: DeviceClassification
    override: { active: boolean; category: DeviceScene; scope: 'persistent' | 'boot' }
    changed: boolean
    error?: { code: string; message: string }
}

export const classificationErrorLabel = (code?: string, fallback?: string): string => ({
    not_found: '设备已离线或身份发生变化，请刷新后重试',
    validation_failed: '请选择有效的设备类型',
    write_failed: '保存失败，请检查设备存储空间后重试',
}[code || ''] || fallback || '操作失败，请稍后重试')
