export type DeviceScene =
    | 'phone'
    | 'computer'
    | 'tablet'
    | 'tv'
    | 'network'
    | 'smart-home'
    | 'camera'
    | 'gaming'
    | 'storage'
    | 'printer'
    | 'wearable'
    | 'unknown'

export const deviceSceneIconPath = (scene: DeviceScene): string =>
    `/luci-static/quickstart/device-icons/${scene === 'unknown' ? 'computer' : scene}.webp`
