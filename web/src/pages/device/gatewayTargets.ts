export type GatewayTargetKind = 'default' | 'self' | 'upstream' | 'bypass' | 'floating' | 'custom'

export interface GatewayTarget {
    id: string
    name: string
    kind: GatewayTargetKind
    gateway?: string
    dns?: string[]
    supported: boolean
    reasons?: string[]
    referenceCount: number
    version: string
}

export interface GatewayReferenceSummary {
    devices: number
    groups: number
    globalPolicy: number
    lanDefault: number
}

export interface GatewayTargetMutationRequest {
    action: 'create' | 'update' | 'delete'
    targetId?: string
    name?: string
    kind?: 'bypass' | 'custom'
    gateway?: string
    replacementTargetId?: string
    expectedVersion?: string
    idempotencyKey?: string
}

export interface GatewayTargetMutationPlan {
    action: GatewayTargetMutationRequest['action']
    target?: GatewayTarget
    replacementTargetId?: string
    referenceSummary: GatewayReferenceSummary
    affectedDevices: string[]
    version: string
    rollbackPoint: string
    canApply: boolean
    error?: { code: string; message: string }
}
