export type CapabilityState = 'available' | 'disabled' | 'not_installed' | 'error'

export type DeviceCapability = {
    state: CapabilityState
    reason?: string
}

type CapabilityKey = 'speedLimit' | 'floatGateway'

type LegacyModuleState = {
    installed?: boolean
    enabled?: boolean
}

type GlobalConfigLike = {
    capabilities?: Partial<Record<CapabilityKey, DeviceCapability>>
    speedLimit?: LegacyModuleState
    floatGateway?: LegacyModuleState
}

export const resolveCapability = (config: GlobalConfigLike | null | undefined, key: CapabilityKey): DeviceCapability => {
    const explicit = config?.capabilities?.[key]
    if (explicit?.state) return explicit

    const legacy = config?.[key]
    if (!legacy?.installed) {
        return { state: 'not_installed', reason: 'dependency_not_installed' }
    }
    return { state: legacy.enabled ? 'available' : 'disabled' }
}

export const capabilityAllowsConfiguration = (capability: DeviceCapability): boolean => {
    return capability.state === 'available' || capability.state === 'disabled'
}
