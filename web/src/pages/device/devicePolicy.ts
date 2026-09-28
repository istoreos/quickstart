import type { CapabilityState } from './deviceCapabilities'
import type { GatewayTargetKind } from './gatewayTargets'

export type PolicyCapability = { state: CapabilityState; reason?: string }
export type StaticPolicy = { enabled: boolean; assignedIP: string; bindIP: boolean; hostname: string; tagName: string; tagTitle?: string }
export type SpeedPolicy = { enabled: boolean; uploadSpeed: number; downloadSpeed: number }
export type RateLimitEnforcement = {
    provider: string
    executionNode: 'local' | 'remote' | 'unknown'
    targetId?: string
    gateway?: string
    state: 'inactive' | 'ready' | 'verified' | 'loaded_unverified' | 'configured_not_loaded' | 'configured_unstable' | 'configured_wrong_node' | 'needs_address_reservation' | 'unavailable'
    reason?: string
    configured: boolean
    loaded: boolean
    verified: boolean
    canApply: boolean
    addressState: 'stable' | 'stable_identity' | 'current_only' | 'missing'
    ipv4?: string
    ipv6State: 'not_present' | 'unsupported' | 'unknown'
    offloadState: 'compatible' | 'risk' | 'unknown'
    warnings?: string[]
    observedAt?: string
}
export type AccessPolicy = { networkAccess: boolean }
export type InternetPathTarget = {
    id: string
    name: string
    kind: GatewayTargetKind
    gateway?: string
    supported: boolean
    reasons?: string[]
}
export type DeviceNetworkPolicy = {
    deviceId: string
    static: Omit<StaticPolicy, 'tagName' | 'tagTitle'>
    path: {
        targetId: string
        effect: {
            desired: { targetId: string; static: Omit<StaticPolicy, 'tagName' | 'tagTitle'>; updatedAt?: string }
            applied: { targetId: string; configVersion: string; state: 'server_applied' | 'server_configuration_observed' | 'apply_failed'; appliedAt?: string }
            observed: { state: 'pending_renewal' | 'lease_observed_unverifiable' | 'unverifiable' | 'observation_error' | 'failed'; reason?: string; source?: string; observedAt?: string }
            needsAttention: boolean
            attentionReason?: string
        }
    }
    targets: InternetPathTarget[]
    version: string
}

export type DevicePolicy = {
    deviceId: string
    displayName?: string
    mac: string
    currentIPv4?: string
    static: StaticPolicy
    speed: SpeedPolicy
    rateLimit?: RateLimitEnforcement
    access: AccessPolicy
    capabilities: Record<'static' | 'speed' | 'access', PolicyCapability>
    version: string
}

export type PolicyPlan = {
    kind?: 'speed' | 'access'
    changes: Array<{ kind: string; description: string }>
    reloadServices: string[]
    requiresRenewal?: boolean
    recoveryAction: string
    version: string
    canApply: boolean
    error?: { code: string; message: string }
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

export const normalizeDhcpHostname = (value: string): string => String(value || '').trim().toLowerCase()

export const validDhcpHostname = (value: string): boolean => {
    const hostname = normalizeDhcpHostname(value)
    return hostname === '' || /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(hostname)
}

export const policyErrorLabel = (code: string, fallback = ''): string => ({
    dependency_not_installed: '所需组件尚未安装',
    conflict: '该设置与现有规则冲突',
    validation_failed: '请检查填写内容',
    apply_failed: '应用失败，原设置已恢复',
    rolled_back: '应用失败，原设置已恢复，可以重试',
    recovery_required: '自动恢复未完成，请按提示处理',
    transaction_unavailable: '暂时无法安全保存，请稍后重试',
    effect_record_failed: '配置已写入，但状态记录需要恢复',
	address_conflict: '该地址已分配给其他设备，请选择一个未占用地址后重试',
	gateway_outside_lan: '该网关不在当前局域网，请检查网关 IP 或选择其他路线',
	gateway_unreachable: '暂时无法连接该网关，请确认设备已开机并接入当前局域网后重试',
}[code] || fallback || '保存失败')

export const internetPathStateLabel = (state: string): string => ({
    pending_renewal: '等待设备重新获取地址',
    lease_observed_unverifiable: '已观察到新租约，终端网关与 DNS 无法验证',
    unverifiable: '服务器配置可读，终端效果无法验证',
    observation_error: '暂时无法读取续租状态',
    failed: '配置应用失败',
}[state] || '状态未知')
