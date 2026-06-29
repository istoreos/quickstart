declare interface RaidDetail {
    detail: string
}
declare interface RaidList {
    disks: Disksinfo[]
}
declare interface Disksinfo {
    name: string,
    path: string,
    venderModel: string,
    active: string,
    status: string,
    level: string,
    members: string[],
    total: string,
    used: string,
    usage: number,
    size: string,
    tranName: string,
    partLabelType: string,
    sizeInt: string,
    childrens: Childrensinfo[]
    rebuildStatus?: string
}
declare interface Childrensinfo {
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

declare interface RaidCreateList {
    members: Membersinfo[]
}
declare interface Membersinfo {
    name: string,
    path: string
    model: string
    sizeStr: string
}