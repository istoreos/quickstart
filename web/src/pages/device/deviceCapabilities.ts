export type CapabilityState = 'available' | 'disabled' | 'not_installed' | 'unsupported' | 'error'

export type CapabilityAction = {
    kind: 'install' | 'enable' | 'retry'
    target?: string
    requiresConfirmation?: boolean
}

export type DeviceCapability = {
    state: CapabilityState
    reason?: string
    actions?: CapabilityAction[]
    desiredRetained?: boolean
}

export type CapabilityKey = 'internetAccess' | 'speedLimit' | 'floatGateway' | 'trafficInsights'

const canonicalKey: Record<CapabilityKey, string> = {
    internetAccess: 'internet_access',
    speedLimit: 'device_speed_limit',
    floatGateway: 'floating_gateway',
    trafficInsights: 'traffic_insights',
}

type GlobalConfigLike = {
    capabilities?: Partial<Record<CapabilityKey, DeviceCapability>> & {
        items?: Record<string, DeviceCapability>
    }
}

export const resolveCapability = (config: GlobalConfigLike | null | undefined, key: CapabilityKey): DeviceCapability => {
    const capability = config?.capabilities?.items?.[canonicalKey[key]] ?? config?.capabilities?.[key]
    return capability?.state ? capability : { state: 'error', reason: 'capability_missing' }
}

export const capabilityAllowsConfiguration = (capability: DeviceCapability): boolean => {
    return capability.state === 'available' || capability.state === 'disabled'
}
