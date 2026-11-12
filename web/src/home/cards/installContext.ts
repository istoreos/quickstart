import type {InjectionKey} from 'vue'
export type CardInstallState='uninstalled'|'update_required'|'ready'|'installing'|'failed'|'unknown'
export type CardInstaller={inspect(id:string,signal?:AbortSignal):Promise<CardInstallState>;install(id:string,signal?:AbortSignal):Promise<void>}
export const cardInstallerKey:InjectionKey<CardInstaller>=Symbol('quickstart-card-installer')
