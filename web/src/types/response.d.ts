// cpu信息
declare interface ResponseSystemCpuStatus {
    usage?: number; //使用率
}

// 系统信息
declare interface ResponseSystemStatus {
    cpuTemperature: number;
    cpuUsage: number;
    localtime?: string;
    memAvailable?: string;
    memAvailablePercentage: number;
    memTotal?: string;
    uptime?: number;
    uptimeHuman?: string;
}

// cpu温度
declare interface ResponseSystemCpuTemperature {
    temperature?: number; //温度
}
declare interface ResponseSystemMemeryStatus {
    total?: string;
    available?: string;
    availablePercentage?: number; //可用百分比
}

// 网速统计
declare interface NetworkStatisticsItem {
    downloadSpeed: number;
    endTime: number;
    startTime: number;
    uploadSpeed: number;
}
declare interface ResponseNetworkStatistics {
    slots: number;
    items: NetworkStatisticsItem[];
    // upload?: NetworkStatisticsModel[]   //上传
    // download?: NetworkStatisticsModel[] //下载
}

// 系统版本信息
declare interface ResponseSystemVersion {
    kernelVersion?: string; //内核版本
    firmwareVersion?: string; //固件版本
    model?: string;
    quickstart: string;
}
// 系统时间
declare interface ResponseSystemTime extends SystemTime {
    uptime?: number; //运行时间
    uptimeHuman?: string;
    localtime?: string; //本地时间
}

// 网络状态
declare interface ResponseNetworkStatus {
    defaultInterface?: string;
    internetConnected?: boolean; //是否已连接互联网
    networkInfo?: string; //网络状态 netSuccess，dnsFailed, netFailed
    proto?: string; //工作模式 pppoe, static, dhcp
    ipv4addr?: string; //ipv4地址
    ipv6addr?: string; //ipv6地址
    gateway?: string; //网关地址
    dnsList?: string[]; //dns地址
    dnsProto?: string; //dns配置方式 manual, auto
    uptime?: string; //在线时间，eg 5h24m33s
    uptimeStamp?: number; //在线时间，毫秒
    dnsReady?: boolean;
}

// 设备信息
declare interface NetworkDevice {
    name?: string; //设备名称
    type?: string; //设备类型 mobile, pc, other
    ipv4addr?: string; //设备的ipv4
    ipv6addr?: string; //设备的ipv6
    mac?: string; //设备的mac
}
// 设备列表
declare interface ResponseNetworkDeviceList {
    devices?: NetworkDevice[];
}

// PPPoE配置
declare interface ResponseGuidePppoe {
    account?: string | null; //拨号账号
    password?: string; //拨号密码
}
declare interface RequestGuidePppoe {
    account: string;
    password: string;
}
// 接口信息
declare interface NetworkInterface {
    name: string;
    macAddress: string;
    linkSpeed: string;
    linkState: string;
    rx_packets: string;
    tx_packets: string;
    interfaceNames: string[];
    master: string;
    duplex: string;
}
declare interface ResponseNetworkInterface {
    ports: NetworkInterface[];
}

// 客户端配置
declare interface GuideClientModel {
    wanProto?: string; //WAN 接口配置方式 ,static, dhcp
    dnsProto?: string; //DNS 配置方式 static, dhcp
    staticIp?: string; // 静态IP地址
    subnetMask?: string; //子网掩码
    manualDnsIp?: string[]; //dns地址
    gateway?: string; //网关地址
}
// 网关模式
declare interface GuideGateway {
    staticLanIp: string; //静态IP地址
    subnetMask: string; //子网掩码
    staticDnsIp: string; //DNS服务器IP
    gateway: string; //网关地址
    enableDhcp: boolean; //是否开启dhcp
}
// 内网配置
declare interface GuideLanSetting {
    lanIp: string;
    netMask: string;
    enableDhcp: boolean;
    dhcpStart: string;
    dhcpEnd: string;
}
// dns配置
declare interface GuideDnsConfig {
    interfaceName: string;
    dnsProto: string; //dns配置方式 manual, auto
    manualDnsIp: string[];
}

// 软件源配置
declare interface GuideSoftSourceInfo {
    name: string;
    url: string;
    identity: string;
}
// 可用软件源列表
declare interface ResponseGuideSoftSource {
    softSourceList: GuideSoftSourceInfo[];
}
// 当前软件源
declare interface ResponseGuideSoftSource {
    softSource: GuideSoftSourceInfo;
}

// nas
declare interface MountPoint {
    mountPoint?: string; //挂载点
    path?: string; //设备路径
    name?: string; //设备名称
    filesystem?: string; //文件系统类型
    total?: string; //设备容量
    used?: string; //设备可用量
    usage?: number; //使用百分比
    uuid?: string;
    isReadOnly?: boolean;
    isSystemRoot?: boolean; //是否为系统所在分区
    sizeInt: string;
    openChildren?: boolean; //是否打开分区，仅UI使用
    childrens?: Folder[];
}
declare interface NasDiskModel {
    name?: string;
    venderModel?: string; //设备型号
    path?: string; //路径
    total?: string; //容量
    size?: string; //总容量
    used?: string; // 设备可用量
    usage?: number; //使用百分比
    errorInfo?: string; //错误信息
    childrens?: MountPoint[]; //挂载的路径，空这位未挂载
    isSystemRoot?: boolean; // 系统所在分区是否在该磁盘
    isDockerRoot?: boolean; // docker所在分区是否在该磁盘
    isExternalDisk?: boolean; // 是否为外置磁盘
    partyLabelType?: string; // 分区标签类型
    tranName?: string; // 分区名称
    openChildren?: boolean; //是否打开磁盘，仅UI使用
    smartWarning?: boolean;
}
declare interface NasDiskStatus {
    disks?: NasDiskModel[];
}
declare interface NasDiskInitrest {
    name?: string;
    venderModel?: string;
    path?: string;
    total?: string;
    used?: string;
    usage?: number;
    size?: string;
    sizeInt?: string;
    childrens?: MountPoint[];
}
declare interface NasDiskPartitionMount {
    name: string;
    mountPoint: string;
    path: string;
    filesystem: string;
    uuid: string;
    sizeInt: string;
    total: string;
    used: string;
    usage: number;
    isReadOnly: boolean;
    isSystemRoot: boolean;
    isDockerRoot: boolean;
}

declare interface ServiceModel {
    serviceType?: string; //NAS 服务类型 linkease, samba, webdav
    name?: string; //类型为sambe的共享名或webdav的url
    directory?: string; //服务指向的目录
}

declare interface UserModel {
    userName?: string;
    password?: string;
}

declare interface NasServiceSambaInfo {
    shareName?: string;
    path?: string;
}
declare interface NasServiceWebdavInfo {
    path?: string;
    port?: string;
    username?: string;
    password?: string;
}
declare interface NasServiceLinkeaseInfo {
    enabel?: boolean;
    port?: string;
}
declare interface NasServiceStatus {
    // services?: ServiceModel[]
    sambas?: NasServiceSambaInfo[];
    webdav?: NasServiceWebdavInfo;
    linkease?: NasServiceLinkeaseInfo;
}
declare interface NasSambaResult {
    sambaUrl?: string; //samba服务地址
    username?: string; //用户名
}
declare interface NasWebdavResult {
    webdavUrl?: string; //webdav服务地址
    username?: string; //用户名
}
declare interface ShareUserInfo {
    userName?: string;
    password?: string;
}

declare interface ShareUserCreateRequest {
    userName: string;
    password: string;
}

declare interface ShareUserListResponse {
    users: ShareUserInfo[];
}

declare interface ShareServiceUserPermission {
    userName?: string;
    ro?: boolean;
    rw?: boolean;
}

declare interface ShareServiceInfo {
    name?: string;
    path?: string;
    samba?: boolean;
    webdav?: boolean;
    users: ShareServiceUserPermission[];
}

declare interface ShareServiceCreateRequest {
    name: string;
    path: string;
    samba: boolean;
    webdav: boolean;
    users: ShareServiceUserPermission[];
}

declare interface ShareServiceListResponse {
    services: ShareServiceInfo[];
}

declare interface NasCreateUniShare {
    shareName: string;
    username: string;
    password: string;
    rootPath: string;
    samba: boolean;
    webdav: boolean;
}

declare interface NasGetSandbox {
    // status: "running" | "stopped";
    status: string;
}

declare interface NasSandboxDisks {
    disks: NasSandboxDisksInfo[];
}
declare interface NasSandboxDisksInfo {
    name: string;
    venderModel: string;
    path: string;
    total: string;
    used: string;
    usage: number;
    size: string;
    partyLabelType: string;
    sizeInt: string;
    childrens: childrensInfo[];
    isSystemRoot: boolean;
    isDockerRoot: boolean;
    isExternalDisk: boolean;
}
declare interface childrensInfo {
    name: string;
    mountPoint: string;
    path: string;
    filesystem: string;
    uuid: string;
    sizeInt: number;
    total: string;
    used: string;
    usage: number;
    isReadOnly: boolean;
    isSystemRoot: boolean;
    isDockerRoot: boolean;
}

// 版本更新
declare interface ResponseStstemCheckUpdate {
    needUpdate: boolean; //版本
    msg: string; //描述
}
// docker状态
declare interface GuideDockerStatus {
    status: string;
    path: string;
    errorInfo: string;
}
// docker推荐安装目录
declare interface GuideDockerPartitionList {
    partitionList: string[];
}
declare interface GuideDockerTransfer {
    path: string;
    emptyPathWarning: boolean;
}

// raid
declare interface RaidDetail {
    detail: string;
}
declare interface RaidList {
    disks: Disksinfo[];
}
declare interface Disksinfo {
    name: string;
    path: string;
    venderModel: string;
    active: string;
    status: string;
    level: string;
    members: string[];
    total: string;
    used: string;
    usage: number;
    size: string;
    tranName: string;
    partyLabelType: string;
    sizeInt: string;
    childrens: Childrensinfo[];
    rebuildStatus?: string;
}
declare interface Childrensinfo {
    name: string;
    mountPoint: string;
    path: string;
    filesystem: string;
    uuid: string;
    sizeInt: string;
    total: string;
    used: string;
    usage: number;
    isReadOnly: boolean;
    isSystemRoot: boolean;
    isDockerRoot: boolean;
}

declare interface RaidCreateList {
    members: Membersinfo[];
}
declare interface Membersinfo {
    name: string;
    path: string;
    model: string;
    sizeStr: string;
}

// ddns
declare interface ResponseGuideDdns {
    ipv4Domain: string;
    ipv6Domain: string;
    ddnstoDomain: string;
}
declare interface RequestGuideDdns {
    ipVersion: string;
    serviceName: string;
    domain: string;
    userName: string;
    password: string;
}
declare interface GuideDdntoConfig {
    deviceId: string;
    netAddr: string;
}

// 插件
declare interface ResponseAppCheck {
    name?: string; //插件名称
    status?: "installed" | "running" | "stopped" | "uninstalled" | "not found"; //插件状态 //installed, uninstalled, not found
}
declare interface RequestAppCheck {
    name: string; //插件名称
    checkRunning?: boolean; //插件是否在运行
}

declare interface RequestAppInstall {
    name: string; //插件名称
}

// 检测设备是否有公网ip
declare interface NetworkCheckPublickNet {
    address: string;
}

declare type ipVersionType = "ipv4" | "ipv6";
declare interface ResultGuideDdns {
    ipVersion: ipVersionType;
    serviceName: string;
    domain: string;
    userName: string;
    password: string;
}

// declare interface ShareServiceInfo {
//     name: string;
//     path: string;
//     samba: boolean;
//     webdav: boolean;
//     users: ShareServiceUserPermission[];
// }

// 文件共享
// declare interface ShareServiceListResponse {
//     services: ShareServiceInfo[];
// }

declare interface Folder {
    fileType: string;
    iconType: string;
    mode: number;
    modifiedTick: string;
    modifiedTime: string;
    name: string;
    rights: string;
    rootPath: string;
    size: string;
    childrens?: Folder[];
    openChildren?: boolean;
}

declare interface ResponseFolders {
    entries: Folder[];
}

declare interface ShareServicDeleteRequest {
    name: string;
}
declare interface Ifaces {
    ifaceName: string;
    device: string;
    ssid: string;
    key: string;
    encryption: string;
    hidden: boolean;
    channel: number;
    htmode: string;
    hwmode: string;
    txpower: number;
    // txpower: boolean;
    network: string;
}
declare interface Ifaces {
    ifaceName: string;
    device: string;
    ssid: string;
    key: string;
    encryption: string;
    hidden: boolean;
    channel: number;
    htmode: string;
    hwmode: string;
    txpower: number;
    // txpower: boolean;
    network: string;
}
declare interface EnableIface {
    ifaceName: string;
    enable: boolean;
}
declare interface SetDevice {
    device: string;
    txpower: number;
}
declare interface ResponseIfaces {
    entries: Ifaces[];
    ifaces: Iface[];
}
declare interface Iface {
    // 类型
    band?: string,
    // 设备
    device?: string,
    // 🔐方式
    encryption?: string,
    encryptSelects?: string[],
    hwmodeSelects?:string[],
    // hwmodeSelects?:
    htmode?: string,
    hwmode?: string,
    ifaceName?: string,
    // 是否访客wifi
    isGuest?: boolean,
    key?: string,
    ssid?: string
    txpower?: number
    // 是否禁用
    disabled?: boolean
    // 是否隐藏
    hidden?: boolean
    // 信道
    channel: number;
    ifaceIndex:number
    // 网络
    network?: string
}


declare interface GuideNeedSetupInfo {
    need?: boolean
    wifi?: boolean
}
