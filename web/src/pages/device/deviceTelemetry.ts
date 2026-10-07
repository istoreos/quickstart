export type DeviceTelemetryState = 'warming_up' | 'ready' | 'stale' | 'partial'

export type DeviceTelemetryItem = {
    deviceId: string
    uploadSpeed: number
    downloadSpeed: number
    uploadBytes: number
    downloadBytes: number
    connectionCount: number
    state: DeviceTelemetryState
    sampledAt?: string
}

export const formatTrafficBytes = (value: number, suffix = ''): string => {
    if (!Number.isFinite(value) || value < 0) return '—'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let amount = value
    let index = 0
    while (amount >= 1000 && index < units.length - 1) {
        amount /= 1000
        index++
    }
    const digits = index === 0 || amount >= 100 ? 0 : amount >= 10 ? 1 : 2
    return `${amount.toFixed(digits)} ${units[index]}${suffix}`
}

export const telemetrySpeedLabel = (item: DeviceTelemetryItem | undefined, direction: 'up' | 'down', warmingLabel = '采集中'): string => {
    if (!item || item.state === 'warming_up') return warmingLabel
    const value = direction === 'up' ? item.uploadSpeed : item.downloadSpeed
    return formatTrafficBytes(value, '/s')
}

export const telemetryBackoff = (failures: number): number => Math.min(30000, 3000 * (2 ** Math.max(0, failures - 1)))

export const telemetryByDevice = (items: DeviceTelemetryItem[]): Map<string, DeviceTelemetryItem> =>
    new Map(items.map(item => [item.deviceId, item]))
