// 网速统计
declare interface NetworkStatisticsModel {
    // time?: string               //时间
    // speed?: number              //速率单位KB

    downloadSpeed: number
    endTime: number
    startTime: number
    uploadSpeed: number
}
declare interface NetworkStatistics {
    slots: number,
    items: NetworkStatisticsModel[],
    // upload?: NetworkStatisticsModel[]   //上传
    // download?: NetworkStatisticsModel[] //下载
}
// 网络状态
declare interface NetworkStatus {
    defaultInterface?: string
    internetConnected?: boolean //是否已连接互联网
    networkInfo?: string //网络状态 netSuccess，dnsFailed, netFailed
    proto?: string  //工作模式 pppoe, static, dhcp
    ipv4addr?: string //ipv4地址
    ipv4mask?: number //ipv4前缀长度
    ipv6addr?: string //ipv6地址
    gateway?: string //网关地址
    dnsList?: string[] //dns地址
    dnsProto?: string  //dns配置方式 manual, auto 
    uptime?: string  //在线时间，eg 5h24m33s
    uptimeStamp?: number //在线时间，毫秒
    dnsReady?: boolean
}

// 设备信息
declare interface NetworkDevice {
    name?: string //设备名称
    type?: string //设备类型 mobile, pc, other
    ipv4addr?: string //设备的ipv4
    ipv6addr?: string //设备的ipv6
    mac?: string //设备的mac
}
// 设备列表
declare interface NetworkDeviceList {
    devices?: NetworkDevice[]
}

declare interface NetworkHomeboxEnable {
    enabel?: boolean
    port?: string
}
declare interface NetworkCheckPublickNet {
    address: string
}

declare interface NetworkPort {
    name: string,
    macAddress: string,
    linkSpeed: string,
    linkState: string,
    rx_packets: string,
    tx_packets: string,
    interfaceNames: string[],
    master: string,
    duplex: string
}
declare interface NetworkPortList {
    ports: NetworkPort[]
}

declare type interfaceNamesType = "LAN" | "LAN6" | string
declare type NetworkPortInfoName = "eth0" | "eth1" | string

declare interface NetworkPortInfo {
    name: NetworkPortInfoName,//名称
    macAddress: string,//mac地址
    linkSpeed: string,//链路状态
    duplex: string,//接口链接状态
    linkState: string,//是否插入网线
    rx_packets: string,//接收
    tx_packets: string,//发送
    interfaceNames: interfaceNamesType[]//接口数组
    master: string
}


declare type FirewallType = "wan" | "lan"
declare interface NetworkInterfaceInfo {
    name: string,
    proto: string,//协议
    ipv4Addr: string,//ipv4地址
    ipv6Addr: string,//ipv6地址
    deviceNames: string[],
    portName: string,//使用的接口
    ports: NetworkPortInfo[]
    firewallType: FirewallType
}



declare interface NetworkInterfaceGetConfigResponseResult {
    devices: NetworkPortInfo[]
    interfaces: NetworkInterfaceInfo[]
}



declare interface NetworkInterfaceConfig {
    name: string,
    proto: string // 协议
    devices: string[] // eth0
    firewallType: FirewallType
}
declare interface NetworkInterfaceSetConfigRequest {
    configs: NetworkInterfaceConfig[]
}