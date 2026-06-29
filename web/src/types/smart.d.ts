// 日志
declare interface ResponseSmartLog {
    result?: string;
}

declare interface SmartConfigGlobal {
    enable?: boolean //是否启用
    powermode?: "never" | "sleep" | "standby" | "idle" //电源模式
    tmpDiff?: number //温度差异
    tmpMax?: number //最大温度差异
}
declare interface SmartConfigDevice {
    tmpDiff?: number //温度差异
    tmpMax?: number //最大温度差异
    devicePath?: string //磁盘路径
}



declare type SmartConfigTaskType = "offline" | "short" | "long" | "conveyance"
declare interface SmartConfigTask {
    type?: SmartConfigTaskType //检查类型
    devicePath?: string //磁盘路径
    month?: string //月份,01 (January) to 12 (December)
    dayPerMonth?: string//每月的第几天,01 to 31
    dayPerWeek?: string//每个星期几,1 (Monday) to 7 (Sunday)
    hour?: string//每天的第几个小时,00 (midnight to just before 1 am) to 23 (11pm to just before midnight)
}

declare interface ResponseSmartConfig {
    global?: SmartConfigGlobal
    devices?: SmartConfigDevice[]
    tasks?: SmartConfigTask[]
}
declare interface RequestSmartConfig {
    global?: SmartConfigGlobal
    devices?: SmartConfigDevice[]
    tasks?: SmartConfigTask[]
}
declare interface PropsSmartConfig {
    global: SmartConfigGlobal
    devices: SmartConfigDevice[]
    tasks: SmartConfigTask[]
}


// 设备
declare interface SmartDiskInfo {
    name?: string
    path?: string
    model?: string
    sizeStr?: string
    serial?: string
    temp?: string
    health?: string
    status?: string
    nvmeVer?: string
    sataVer?: string
    rotaRate?: string
}
declare type SmartDisks = SmartDiskInfo[]
declare interface ResponseSmartList {
    disks?: SmartDisks
}

// 调试
declare interface RequestSmartTest {
    type: SmartConfigTaskType
    devicePath: string
}
declare interface ResponseSmartTest {
    result?: string
}
// 调试结果
declare interface RequestSmartTestResult {
    type: "error" | "selftest"
    devicePath: string
}
declare interface ResponseSmartTestResult {
    result?: string
}


// 设备属性
declare interface RequestSmartAttributeResult {
    devicePath: string
}
declare interface ResponseSmartAttributeResult {
    result?: string
}
declare interface RequestSmartExtendResult {
    devicePath: string
}
declare interface ResponseSmartExtendResult {
    result?: string
}