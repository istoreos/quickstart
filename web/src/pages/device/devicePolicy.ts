export type PolicyCapability = { state: 'available' | 'disabled' | 'not_installed' | 'error'; reason?: string }
export type StaticPolicy = { enabled: boolean; assignedIP: string; bindIP: boolean; hostname: string; tagName: string; tagTitle?: string }
export type SpeedPolicy = { enabled: boolean; uploadSpeed: number; downloadSpeed: number }
export type AccessPolicy = { networkAccess: boolean }

export type DevicePolicy = {
    deviceId: string
    displayName?: string
    mac: string
    currentIPv4?: string
    static: StaticPolicy
    speed: SpeedPolicy
    access: AccessPolicy
    capabilities: Record<'static' | 'speed' | 'access', PolicyCapability>
}

export const policyAvailable = (policy: DevicePolicy | null, kind: 'static' | 'speed' | 'access'): boolean =>
    policy?.capabilities?.[kind]?.state === 'available'

export const policyUnavailableReason = (policy: DevicePolicy | null, kind: 'static' | 'speed' | 'access'): string => {
    const capability = policy?.capabilities?.[kind]
    if (!capability || capability.state === 'available') return ''
    if (capability.reason === 'dependency_not_installed' || capability.state === 'not_installed') return '所需组件尚未安装'
    if (capability.state === 'disabled') return '请先在全局设置中启用该能力'
    return capability.reason || '当前暂不可用'
}

export const policyLabelsFromPolicy = (policy: DevicePolicy): string[] => {
    const labels: string[] = []
    if (policy.static.enabled) labels.push('static')
    if (!policy.access.networkAccess) labels.push('blocked')
    else if (policy.speed.enabled) labels.push('limited')
    return labels
}

export const policyMacSuffix = (mac: string): string => mac.replace(/[^0-9a-f]/gi, '').slice(-4).toUpperCase()

export const policyErrorLabel = (code: string, fallback = ''): string => ({
    dependency_not_installed: '所需组件尚未安装',
    conflict: '该设置与现有规则冲突',
    validation_failed: '请检查填写内容',
    apply_failed: '应用失败，原设置已恢复',
}[code] || fallback || '保存失败')
