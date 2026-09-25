import type { DeviceScene } from './deviceScene'

export type InventoryAddress = {
    family: 4 | 6
    address: string
    primary?: boolean
}

export type InventoryDevice = {
    deviceId: string
    displayName?: string
    hostname?: string
    online: boolean
    presenceState?: 'online' | 'offline' | 'never_seen'
    lastSeenAt: string
    mac: string
    vendor?: string
    classification?: DeviceClassification
    icon?: { mode: 'auto' | 'manual'; preferenceKey?: string; resolvedKey: string; assetKey: string; label: string; brandLabel?: string }
    identity?: { kind: 'mac' | 'duid_iaid'; scope: 'persistent' | 'boot' }
    addresses: {
        current: InventoryAddress[]
        historical: InventoryAddress[]
    }
    connection: { kind: 'lan' | 'wifi' | 'unknown' }
}

export type DeviceClassification = {
    brand: string
    manufacturer: string
    category: DeviceScene
    source: 'model' | 'hostname' | 'manufacturer_default' | 'fallback' | 'manual'
    confidence: 'high' | 'medium' | 'low'
}

export type LegacyDevice = {
    hostname?: string
    ip?: string
    mac?: string
    uploadSpeedStr?: string
    downloadSpeedStr?: string
    staticAssigned?: Record<string, unknown>
    speedLimit?: Record<string, unknown>
}

export type DevicePolicyRules = {
    static?: Array<{ assignedMac?: string }>
    speed?: Array<{ mac?: string; ip?: string; enabled?: boolean; networkAccess?: boolean }>
}

export type DeviceListItem = InventoryDevice & {
    scene: DeviceScene
    brand: string
    classification: DeviceClassification
    primaryAddress: string
    extraAddressCount: number
    controlled: boolean
    policyLabels: string[]
    uploadSpeedStr: string
    downloadSpeedStr: string
    telemetry?: import('./deviceTelemetry').DeviceTelemetryItem
    iconKey?: import('./deviceScene').DeviceIconKey
    legacy?: LegacyDevice
}

export type DeviceFilter = 'online' | 'all' | 'controlled'
export type DeviceSort = 'recent' | 'name'

export type DeviceIdentityMetaMode = 'category' | 'classification-source' | 'pending'

export const deviceIdentityPresentation = (device: Pick<DeviceListItem, 'displayName' | 'hostname' | 'brand' | 'classification'>) => {
    const hasExplicitName = Boolean(String(device.displayName || device.hostname || '').trim())
    return {
        showBrandInMeta: hasExplicitName && Boolean(device.brand),
        metaMode: device.classification.source === 'fallback'
            ? 'pending' as DeviceIdentityMetaMode
            : !hasExplicitName && device.brand
                ? 'classification-source' as DeviceIdentityMetaMode
                : 'category' as DeviceIdentityMetaMode,
    }
}

const normalized = (value: unknown): string => String(value || '').trim().toLowerCase()

const brandRules: Array<{ brand: string; patterns: RegExp[] }> = [
    { brand: 'ASUS', patterns: [/\basustek\b/i, /\basus\b/i] },
    { brand: 'Apple', patterns: [/\bapple(?:,? inc\.?)?\b/i] },
    { brand: 'Samsung', patterns: [/\bsamsung\b/i] },
    { brand: 'Xiaomi', patterns: [/\bxiaomi\b/i, /北京小米/i] },
    { brand: 'HUAWEI', patterns: [/\bhuawei\b/i, /华为/i] },
]

export const normalizeDeviceBrand = (manufacturer?: string): string => {
    const value = String(manufacturer || '').trim().normalize('NFKC')
    return brandRules.find(rule => rule.patterns.some(pattern => pattern.test(value)))?.brand || ''
}

const sceneRules: Array<{ scene: DeviceScene; patterns: RegExp[] }> = [
    { scene: 'storage', patterns: [/\bnas\b/i, /\bserver\b/i, /storage/i, /truenas/i, /openmediavault/i, /synology/i, /qnap/i, /群晖/i, /威联通/i] },
    { scene: 'camera', patterns: [/camera/i, /webcam/i, /doorbell/i, /\bipc[-_ ]/i, /cctv/i, /摄像头/i, /监控/i, /门铃/i] },
    { scene: 'gaming', patterns: [/gaming/i, /game[-_ ]?console/i, /playstation/i, /\bps[345]\b/i, /xbox/i, /nintendo/i, /steam[-_ ]?deck/i, /游戏/i, /主机/i] },
    { scene: 'printer', patterns: [/printer/i, /laserjet/i, /deskjet/i, /officejet/i, /打印机/i, /一体机/i] },
    { scene: 'wearable', patterns: [/smart[-_ ]?watch/i, /fitness[-_ ]?(band|tracker)/i, /wearable/i, /watch/i, /手表/i, /手环/i] },
    { scene: 'tv', patterns: [/(^|[-_ ])tv($|[-_ ])/i, /television/i, /smart[-_ ]?tv/i, /chromecast/i, /streaming[-_ ]?(box|stick)/i, /电视/i, /投影/i] },
    { scene: 'tablet', patterns: [/tablet/i, /\bipad\b/i, /e[-_ ]?reader/i, /kindle/i, /平板/i, /阅读器/i] },
    { scene: 'phone', patterns: [/phone/i, /iphone/i, /android/i, /mobile/i, /pixel[-_ ]?\d/i, /手机/i] },
    { scene: 'network', patterns: [/router/i, /gateway/i, /openwrt/i, /access[-_ ]?point/i, /\bmesh\b/i, /repeater/i, /\bswitch\b/i, /路由/i, /网关/i, /交换机/i, /无线接入点/i] },
    { scene: 'smart-home', patterns: [/smart[-_ ]?home/i, /home[-_ ]?assistant/i, /speaker/i, /thermostat/i, /sensor/i, /vacuum/i, /air[-_ ]?purifier/i, /light[-_ ]?(bulb|strip)/i, /smart[-_ ]?plug/i, /智能家居/i, /音箱/i, /传感器/i, /扫地/i, /灯泡/i, /插座/i] },
    { scene: 'computer', patterns: [/desktop/i, /laptop/i, /notebook/i, /workstation/i, /macbook/i, /imac/i, /windows/i, /ubuntu/i, /\bpc\b/i, /电脑/i, /笔记本/i, /工作站/i] },
]

export const detectDeviceScene = (device: Pick<InventoryDevice, 'displayName' | 'hostname' | 'vendor'>): DeviceScene => {
    const descriptiveSource = [device.displayName, device.hostname]
        .filter(Boolean)
        .join(' ')
        .normalize('NFKC')
    const explicit = sceneRules.find(rule => rule.patterns.some(pattern => pattern.test(descriptiveSource)))?.scene
    if (explicit) return explicit

    const combinedSource = `${descriptiveSource} ${device.vendor || ''}`.normalize('NFKC')
    if (/\b(?:rt|gt)-?ax\d+[a-z0-9-]*\b|\bzenwifi\b|\brouter\b|\bgateway\b|\baccess[-_ ]?point\b|\bmesh\b|\brepeater\b|路由|网关|交换机/i.test(combinedSource)) {
        return 'network'
    }
    if (normalizeDeviceBrand(device.vendor) === 'ASUS') return 'network'
    return 'unknown'
}

const validScenes = new Set<DeviceScene>(['phone', 'computer', 'tablet', 'tv', 'network', 'smart-home', 'camera', 'gaming', 'storage', 'printer', 'wearable'])
const validSources = new Set<DeviceClassification['source']>(['model', 'hostname', 'manufacturer_default', 'fallback', 'manual'])
const validConfidence = new Set<DeviceClassification['confidence']>(['high', 'medium', 'low'])

export const resolveDeviceClassification = (device: InventoryDevice): DeviceClassification => {
    const classification = device.classification
    if (classification && validScenes.has(classification.category) && validSources.has(classification.source) && validConfidence.has(classification.confidence)) {
        return {
            brand: String(classification.brand || '').trim(),
            manufacturer: String(classification.manufacturer || device.vendor || '').trim(),
            category: classification.category,
            source: classification.source,
            confidence: classification.confidence,
        }
    }

    const legacyScene = detectDeviceScene(device)
    const category = legacyScene === 'unknown' ? 'computer' : legacyScene
    return {
        brand: normalizeDeviceBrand(device.vendor),
        manufacturer: String(device.vendor || '').trim(),
        category,
        source: legacyScene === 'unknown' ? 'fallback' : normalizeDeviceBrand(device.vendor) === 'ASUS' && category === 'network' && !String(device.displayName || device.hostname || '').trim() ? 'manufacturer_default' : 'hostname',
        confidence: legacyScene === 'unknown' ? 'low' : 'medium',
    }
}

const hasStaticPolicy = (legacy?: LegacyDevice): boolean => {
    const value = legacy?.staticAssigned || {}
    return Boolean(value.hostname || value.tagName || value.tagTitle || value.bindIP || value.action === 'modify')
}

const policyLabels = (legacy?: LegacyDevice): string[] => {
    const labels: string[] = []
    if (hasStaticPolicy(legacy)) labels.push('static')
    if (legacy?.speedLimit?.enabled) {
        labels.push(legacy.speedLimit.networkAccess === false ? 'blocked' : 'limited')
    }
    return labels
}

const policyLabelsFromRules = (device: InventoryDevice, rules?: DevicePolicyRules): string[] => {
    if (!rules) return []
    const mac = normalized(device.mac)
    const addresses = new Set(device.addresses.current.map(item => normalized(item.address)))
    const labels: string[] = []
    if ((rules.static || []).some(rule => normalized(rule.assignedMac) === mac)) labels.push('static')
    const speed = (rules.speed || []).find(rule => normalized(rule.mac) === mac || addresses.has(normalized(rule.ip)))
    if (speed?.enabled) labels.push(speed.networkAccess === false ? 'blocked' : 'limited')
    return labels
}

export const buildDeviceListItems = (devices: InventoryDevice[], legacyDevices: LegacyDevice[] = [], rules?: DevicePolicyRules): DeviceListItem[] => {
    const legacyByMac = new Map(legacyDevices.map(device => [normalized(device.mac), device]))
    return devices.map(device => {
        const legacy = legacyByMac.get(normalized(device.mac))
        const primary = device.addresses.current.find(address => address.primary)
            || device.addresses.current[0]
            || device.addresses.historical[0]
        const labels = rules ? policyLabelsFromRules(device, rules) : policyLabels(legacy)
        const classification = resolveDeviceClassification(device)
        return {
            ...device,
            scene: classification.category,
            brand: classification.brand,
            classification,
            iconKey: device.icon?.assetKey as import('./deviceScene').DeviceIconKey | undefined,
            displayName: device.displayName || '',
            primaryAddress: primary?.address || '',
            extraAddressCount: Math.max(0, device.addresses.current.length + device.addresses.historical.length - (primary ? 1 : 0)),
            controlled: labels.length > 0,
            policyLabels: labels,
            uploadSpeedStr: legacy?.uploadSpeedStr || '',
            downloadSpeedStr: legacy?.downloadSpeedStr || '',
            legacy,
        }
    })
}

export const selectDeviceListItems = (
    devices: DeviceListItem[],
    filter: DeviceFilter,
    query: string,
    sort: DeviceSort,
): DeviceListItem[] => {
    const keyword = normalized(query)
    const selected = devices.filter(device => {
        if (filter === 'online' && !device.online) return false
        if (filter === 'controlled' && !device.controlled) return false
        if (!keyword) return true
        const addresses = [...device.addresses.current, ...device.addresses.historical].map(item => item.address)
        return [device.displayName, device.hostname, device.mac, device.vendor, device.brand, ...addresses]
            .some(value => normalized(value).includes(keyword))
    })

    return [...selected].sort((left, right) => {
        if (sort === 'name') {
            return normalized(left.displayName || left.hostname || left.mac)
                .localeCompare(normalized(right.displayName || right.hostname || right.mac))
        }
        if (left.online !== right.online) return left.online ? -1 : 1
        const recent = Date.parse(right.lastSeenAt) - Date.parse(left.lastSeenAt)
        return recent || left.deviceId.localeCompare(right.deviceId)
    })
}

export const deviceCounts = (devices: DeviceListItem[]) => ({
    all: devices.length,
    online: devices.filter(device => device.online).length,
    controlled: devices.filter(device => device.controlled).length,
})

export const splitHighlight = (value: string, query: string): Array<{ text: string; matched: boolean }> => {
    const keyword = query.trim()
    if (!keyword) return [{ text: value, matched: false }]
    const lowerValue = value.toLowerCase()
    const lowerKeyword = keyword.toLowerCase()
    const parts: Array<{ text: string; matched: boolean }> = []
    let cursor = 0
    while (cursor < value.length) {
        const index = lowerValue.indexOf(lowerKeyword, cursor)
        if (index < 0) {
            parts.push({ text: value.slice(cursor), matched: false })
            break
        }
        if (index > cursor) parts.push({ text: value.slice(cursor, index), matched: false })
        parts.push({ text: value.slice(index, index + keyword.length), matched: true })
        cursor = index + keyword.length
    }
    return parts.length ? parts : [{ text: value, matched: false }]
}
