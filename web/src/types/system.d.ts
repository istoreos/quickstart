// 系统版本信息
declare interface SystemVersion {
    model?: string //设备型号
    kernelVersion?: string //内核版本
    firmwareVersion?: string //固件版本
}
declare interface SystemCheckUpdate {
    needUpdate: boolean, //版本
    msg: string, //描述
}
declare interface SystemStatus {
    cpuTemperature?: number,
    cpuUsage?: number,
    memAvailable?: string,
    memAvailablePercentage?: number,
    memTotal?: string,
    uptime?: number,     //运行时间
    uptimeHuman?: string,
    localtime?: string,  //本地时间
}

// 系统时间
declare interface SystemTime {
    uptime?: number     //运行时间
    localtime?: string  //本地时间
}