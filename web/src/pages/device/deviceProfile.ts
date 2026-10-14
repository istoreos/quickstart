export const normalizeDeviceAlias = (value: string): string => value.trim()

export const validDeviceAlias = (value: string): boolean => {
    const normalized = normalizeDeviceAlias(value)
    return Array.from(normalized).length <= 64 && new TextEncoder().encode(normalized).length <= 256 && !/[\u0000-\u001f\u007f]/.test(normalized)
}

export type TaskTransaction = {
    task: 'profile' | 'network' | 'restrictions'
    idempotencyKey?: string
    stage: 'plan' | 'validate' | 'snapshot' | 'apply' | 'verify' | 'rollback' | 'recover'
    status: 'in_progress' | 'committed' | 'unchanged' | 'rejected' | 'failed' | 'rolled_back' | 'recovery_required'
    replayed?: boolean
    recoveryAction?: string
}

export type DeviceProfileResult = {
    profile?: {
        deviceId: string
        alias?: string
        originalHostname?: string
        scope: 'persistent' | 'boot'
        classification: { brand?: string; category: string; source: string; confidence: string }
        manualBrand?: string
        manualCategory?: string
        icon: {
            mode: 'auto' | 'manual'
            preferenceKey?: string
            resolvedKey: string
            assetKey: string
            label: string
            brandLabel?: string
        }
    }
    changed: boolean
    error?: { code: string; message: string }
    transaction?: TaskTransaction
}
