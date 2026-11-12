export const deviceScopeKey = (device: { networkId?: string; deviceId: string }) => `${device.networkId || ''}\u0000${device.deviceId}`
