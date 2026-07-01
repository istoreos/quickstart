
declare interface MountPoint {
    mountPoint?: string //挂载点
    path?: string //设备路径
    name?: string //设备名称
    filesystem?: string //文件系统类型
    total?: string //设备容量
    used?: string  //设备可用量
    usage?: number //使用百分比
    uuid?: string,
    isReadOnly?: boolean,
    isSystemRoot?: boolean //是否为系统所在分区

}
declare interface NasDiskModel {
    name?: string
    venderModel?: string //设备型号
    path?: string //路径
    total?: string //容量
    size?: string //总容量
    used?: string // 设备可用量
    usage?: number //使用百分比
    errorInfo?: string //错误信息
    childrens?: MountPoint[] //挂载的路径，空这位未挂载
    isSystemRoot?: boolean // 系统所在分区是否在该磁盘
    isDockerRoot?: boolean // docker所在分区是否在该磁盘
    smartWarning?: boolean //smart错误警告
}
declare interface NasDiskStatus {
    disks?: NasDiskModel[]
}
declare interface NasDiskInitrest {
    name?: string,
    venderModel?: string,
    path?: string,
    total?: string,
    used?: string,
    usage?: number,
    size?: string,
    sizeInt?: string,
    childrens?: MountPoint[]
}
declare interface NasDiskPartitionMount {
    name: string,
    mountPoint: string,
    path: string,
    filesystem: string,
    uuid: string,
    sizeInt: string,
    total: string,
    used: string,
    usage: number,
    isReadOnly: boolean,
    isSystemRoot: boolean,
    isDockerRoot: boolean
}

declare interface ServiceModel {
    serviceType?: string //NAS 服务类型 linkease, samba, webdav 
    name?: string //类型为sambe的共享名或webdav的url
    directory?: string //服务指向的目录
}

declare interface NasServiceSambaInfo {
    shareName?: string
    path?: string
}
declare interface NasServiceWebdavInfo {
    path?: string
    port?: string
    username?: string
    password?: string
}
declare interface NasServiceLinkeaseInfo {
    enabel?: boolean
    port?: string
}
declare interface NasServiceStatus {
    // services?: ServiceModel[]
    sambas?: NasServiceSambaInfo[]
    webdav?: NasServiceSambaInfo
    linkease?: NasServiceLinkeaseInfo
}
declare interface NasSambaResult {
    sambaUrl?: string //samba服务地址
    username?: string //用户名
}
declare interface NasWebdavResult {
    webdavUrl?: string //webdav服务地址
    username?: string //用户名
}
declare interface ShareUserInfo {
    userName?: string
    password?: string
}

declare interface ShareUserCreateRequest {
    userName: string
    password: string
}

declare interface ShareUserListResponse {
    users: ShareUserInfo[]
}

declare interface ShareServiceUserPermission {
    userName?: string
    ro?: boolean
    rw?: boolean
}

declare interface ShareServiceInfo {
    name?: string
    path?: string
    samba?: boolean
    webdav?: boolean
    users: ShareServiceUserPermission[]
}

declare interface ShareServiceCreateRequest {
    name: string
    path: string
    samba: boolean
    webdav: boolean
    users: ShareServiceUserPermission[]
}

declare interface ShareServiceListResponse {
    services: ShareServiceInfo[]
}

declare interface NasCreateUniShare {
    shareName: string
    username: string
    password: string
    rootPath: string
    samba: boolean
    webdav: boolean
}

declare interface NasGetSandbox {
    status: string
}

declare interface NasSandboxDisks {
    disks: NasSandboxDisksInfo[]
}
declare interface NasSandboxDisksInfo {
    name: string,
    venderModel: string,
    path: string,
    total: string,
    used: string,
    usage: number,
    size: string,
    partLabelType: string,
    sizeInt: string,
    childrens: childrensInfo[],
    isSystemRoot: boolean,
    isDockerRoot: boolean,
    isExternalDisk: boolean
}
declare interface childrensInfo {
    name: string,
    mountPoint: string,
    path: string,
    filesystem: string,
    uuid: string,
    sizeInt: number,
    total: string,
    used: string,
    usage: number,
    isReadOnly: boolean,
    isSystemRoot: boolean,
    isDockerRoot: boolean
}
