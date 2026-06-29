declare interface GuidePppoe {
    account?: string | null //拨号账号
    password?: string //拨号密码
    enableLanDhcp: boolean //开启lan口的dhcp服务，用于从旁路由模式恢复
}

declare interface GuideDnsConfig {
    interfaceName: string,
    dnsProto: string, //dns配置方式 manual, auto
    manualDnsIp: string[]
}

declare interface GuideClientModel {
    wanProto?: string,//WAN 接口配置方式 ,static, dhcp
    dnsProto?: string,//DNS 配置方式 static, dhcp
    staticIp?: string,// 静态IP地址
    subnetMask?: string, //子网掩码
    manualDnsIp?: string[] //dns地址
    gateway?: string //网关地址
    enableLanDhcp: boolean //开启lan口的dhcp服务，用于从旁路由模式恢复
}


declare interface GuideGateway {
    staticLanIp: string, //静态IP地址
    subnetMask: string,  //子网掩码
    staticDnsIp: string,//DNS服务器IP
    gateway: string //网关地址
    enableDhcp: boolean //是否开启dhcp
    dhcp6c: boolean //是否开启dhcp6客户端
    enableNat: boolean //是否开启NAT
}

declare interface GuideDockerStatus {
    status: string,
    path: string,
    errorInfo: string
}
declare interface GuideDockerPartitionList {
    partitionList: string[]
}
declare interface GuideDockerTransfer {
    path: string,
    emptyPathWarning: boolean
}
declare interface GuideAria2Init {
    scope: string,
    detail: string
}

declare interface GuideqBitorrentInit {
    scope: string,
    detail: string
}

declare interface GuideTransmissionInit {
    scope: string,
    detail: string
}

declare interface GuideLanSetting {
    lanIp: string,
    netMask: string,
    enableDhcp: boolean,
    dhcpStart: string,
    dhcpEnd: string
}



declare interface GuideDownloadServiceStatus {
    // services?: ServiceModel[]
    aria2?: GuideDownloadServiceAria2Info
    qbittorrent?: GuideDownloadServiceqBittorrentInfo
    transmission?: GuideDownloadServiceTransmissionInfo
}

declare interface GuideDownloadServiceAria2Info {
    status?: string,
    downloadPath?: string,
    configPath?: string,
    rpcToken: string,
    webPath: string,
    rpcPort: number,
}
declare interface GuideDownloadServiceqBittorrentInfo {
    status: string,
    downloadPath: string,
    configPath?: string,
    webPath: string
}
declare interface GuideDownloadServiceTransmissionInfo {
    status: string,
    downloadPath: string,
    configPath?: string,
    webPath: string
}

declare interface GuideDownloadPartitionList {
    partitionList?: string[]
}

declare interface GuideSoftSource {
    softSource: GuideSoftSourceInfo,

}
declare interface GuideSoftSourceInfo {
    name: string,
    url: string,
    identity: string
}

declare interface GuideSoftSourceList {
    softSourceList: GuideSoftSourceInfo[]
}

declare interface GuideDdns {
    ipv4Domain: string,
    ipv6Domain: string,
    ddnstoDomain: string
}

declare interface GuideDdntoConfig {
    deviceId: string,
    netAddr: string
}

declare interface GuideDockerSwitchRequest {
    enable: boolean
}
