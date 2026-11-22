// Memory-only model for the storage design preview; not a deployment contract.
export type Purpose = 'files' | 'hosts' | 'both' | 'apps'
export type StorageTarget = { id:string; label:string; capacity:number; kind:'disk'|'partition'|'free'; token:string; effect:string; preserved:string }
export type StoragePlan = { purpose:Purpose; hosts:number; apps:number; management:'existing'|'application'|'separate'; managementSize:number; files?:number; preserveExisting?:boolean; fileMount?:string; managementInFiles?:boolean }
export type ApplicationVolume = { id:number; kind:'docker'|'management'; name:string; capacity:number; used:number; connected?:boolean }
export const purposes:{id:Purpose; title:string; description:string; detail:string}[] = [
  {id:'hosts',title:'用于 LightHost 主机',description:'创建 Linux 主机和虚拟机，保存它们的磁盘与恢复点。',detail:'适合主要用来运行 LightHost 的设备'},
  {id:'both',title:'同时用于主机和应用',description:'一部分给 LightHost 主机，另一部分按需创建应用卷。',detail:'适合同时运行主机和宿主 Docker 的设备'},
  {id:'apps',title:'用于应用存储',description:'按需创建 Docker 数据卷、LightHost 管理卷等。',detail:'需要使用时，再选择用途和容量'},
]
export const storageTargets:StorageTarget[] = [
  {id:'disk',label:'整块数据盘 · sdb',capacity:128,kind:'disk',token:'sdb',effect:'清空 sdb 上的全部分区和文件',preserved:'系统盘 sda、家庭文件盘 sdc 保留'},
  {id:'partition',label:'独立分区 · sdc2',capacity:96,kind:'partition',token:'sdc2',effect:'清空 sdc2 覆盖范围内的数据；按方案重新划分这段空间',preserved:'同盘 sdc1 的家庭文件与系统盘 sda 保留'},
  {id:'free',label:'系统盘剩余空间 · sda',capacity:116,kind:'free',token:'sda-free',effect:'在未分配空间中新建存储，不格式化已有系统分区',preserved:'sda 的启动、系统镜像和 overlay 分区保留'},
]
