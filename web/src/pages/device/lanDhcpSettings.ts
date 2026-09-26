export interface LanDhcpSettings {
    enabled: boolean
    poolStart: string
    poolEnd: string
    leaseTime: string
    defaultTargetId: string
}

export interface LanDhcpConflict {
    address: string
    kind: 'router' | 'floating_gateway' | 'gateway_node' | 'address_reservation' | string
}

export interface LanDhcpSettingsResult {
    settings: LanDhcpSettings
    version: string
    editable: boolean
    readOnlyReason?: string
    affectedDevices: number
    recoveryGuidance?: string
    conflicts: LanDhcpConflict[]
    reloadServices: string[]
    canApply: boolean
    changed: boolean
    error?: { code: string; message: string }
}
