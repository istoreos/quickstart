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

export const deviceIconKeys = [
    'computer', 'phone', 'tablet', 'tv', 'network', 'smart-home', 'camera', 'gaming', 'storage', 'printer', 'wearable', 'unknown',
    'desktop', 'laptop', 'smart-speaker', 'smart-bulb', 'thermostat', 'sensor', 'door-lock', 'robot-vacuum', 'air-conditioner',
    'projector', 'set-top-box', 'game-console', 'handheld-game', 'home-server', 'network-switch', 'access-point', 'network-bridge', 'e-reader'
] as const

export type DeviceIconKey = typeof deviceIconKeys[number]

export const isDeviceIconKey = (value: unknown): value is DeviceIconKey =>
    typeof value === 'string' && (deviceIconKeys as readonly string[]).includes(value)

export const deviceIconLabels: Record<DeviceIconKey, string> = {
    computer: '通用电脑', phone: '手机', tablet: '平板电脑', tv: '电视', network: '无线路由器',
    'smart-home': '智能家居', camera: '摄像头', gaming: '游戏设备', storage: '网络存储', printer: '打印机', wearable: '穿戴设备', unknown: '未知设备',
    desktop: '台式电脑', laptop: '笔记本电脑', 'smart-speaker': '智能音箱', 'smart-bulb': '智能灯', thermostat: '温控器', sensor: '传感器',
    'door-lock': '智能门锁', 'robot-vacuum': '扫地机器人', 'air-conditioner': '空调', projector: '投影仪', 'set-top-box': '电视盒子',
    'game-console': '游戏主机', 'handheld-game': '掌上游戏机', 'home-server': '家庭服务器', 'network-switch': '网络交换机',
    'access-point': '无线接入点', 'network-bridge': '网络桥接器', 'e-reader': '电子阅读器'
}

export const deviceIconPath = (key: DeviceIconKey): string =>
    `/luci-static/quickstart/device-icons/${key === 'unknown' ? 'computer' : key}.webp`

export const deviceSceneIconPath = (scene: DeviceScene): string =>
    deviceIconPath(scene)
