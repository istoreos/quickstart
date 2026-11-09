import { inject, type InjectionKey } from 'vue'
import type { DiscoveredLAN } from './lanDiscovery'
import { createDeviceManagement, DeviceMangement } from '/@/request/request'

export type LANScope = DiscoveredLAN
export type LANContext = { networkId?: string; networks: LANScope[]; legacy?: boolean }
export const lanContextKey: InjectionKey<LANContext> = Symbol('lan-context')
export const lanClientKey: InjectionKey<ReturnType<typeof createDeviceManagement>> = Symbol('lan-client')
export const useLANContext = () => inject(lanContextKey, { networks: [] })
export const useLANClient = () => inject(lanClientKey, DeviceMangement)
export { deviceScopeKey } from './lanIdentity'
