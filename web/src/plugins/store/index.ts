import { createPinia, defineStore } from 'pinia'
import request from '/@/request'
import { FakerNetworkStatusResponse } from "/@/faker"
export const useAppStore = defineStore("app", {
    state: () => {
        return {
            portitemStyle: {
                show: false,
                left: 0,
                top: 0,
                portitem: {
                    name: "",
                    macAddress: "",
                    linkSpeed: "",
                    linkState: "",
                    rx_packets: "",
                    tx_packets: "",
                    interfaceNames: [],
                    master: "",
                    duplex: ""
                } as NetworkPort
            }
        }
    }
})
export const useGuideStore = defineStore("guide", {
})
export const useNasStore = defineStore("nas", {
    state: () => {
        return {
            webdav: <NasServiceWebdavInfo>{},
        }
    }
})
export const useNetworkStore = defineStore("network", {
    state: () => {
        return {
            status: <NetworkStatus>{},
            deviceList: <NetworkDeviceList>{},
        }
    },
    getters: {
    },
    actions: {
        updateNetworkStatus(networkStatus: NetworkStatus) {
            this.status = networkStatus
        },
        requestDeviceList() {
            request.Network.Device.List.GET().then(res => {
                if (res?.data) {
                    const { result } = res?.data
                    if (result) {
                        this.deviceList = result
                    }
                }
            })
        },
        incrTime() {
            if (this.status.uptimeStamp) {
                this.status.uptimeStamp++
            }
        },
    }
})
export const useSystemStore = defineStore("system", {
    state: () => {
        return {
            version: <SystemVersion>{},
            checkUpdate: <SystemCheckUpdate|null>null,
            updateChecked: <Boolean>false,
            systemStatus: <SystemStatus>{},
        }
    },
    getters: {
    },
    actions: {
        incrTime() {
            if (this.systemStatus?.uptime) {
                this.systemStatus.uptime++
            }
        },
        requestVersion() {
            request.System.Version.GET().then(res => {
                if (res?.data?.result) {
                    this.version = res.data.result
                }
            })
        },
        requestCheckUpdate() {
            if (this.updateChecked)
                return
            this.updateChecked = true
            request.System.CheckUpdate.GET().then(res => {
                if (res?.data?.result) {
                    this.checkUpdate = res.data.result
                }
            }).finally(()=>{
                if (this.checkUpdate == null)
                    this.checkUpdate = {needUpdate:false, msg:"skip"}
            })
        },
        updateSystemStatus(systemStatus: SystemStatus) {
            this.systemStatus = systemStatus
        },
    }
})
let getDataCalled = false
export const getData = () => {
    if (getDataCalled)
        return
    getDataCalled = true
    let systemStatusInit = true
    let networkStatusInit = true
    const network = useNetworkStore()
    const system = useSystemStore()
    const requestSystemStatus = function() {
        return (!systemStatusInit && document.hidden ? Promise.resolve() : request.System.Status.GET().then(res => {
            if (res?.data.result) {
                system.updateSystemStatus(res.data.result)
            }
        })).finally(()=>{
            setTimeout(requestSystemStatus, 5000)
            if (systemStatusInit) {
                setInterval(() => {
                    system.incrTime()
                }, 1000)
                systemStatusInit = false
            }
        })
    }
    const requestNetworkStatus = function() {
        return (!networkStatusInit && document.hidden ? Promise.resolve() : request.Network.Status.GET().then(res => {
            if (res?.data) {
                const { result } = res?.data
                if (result) {
                    network.updateNetworkStatus(result)
                }
            }
        })).finally(()=>{
            setTimeout(requestNetworkStatus, 5000)
            if (networkStatusInit) {
                setInterval(() => {
                    network.incrTime()
                }, 1000)
                networkStatusInit = false
            }
        })
    }
    // network
    requestNetworkStatus()
    network.requestDeviceList()
    // system
    setTimeout(()=>{
        system.requestVersion()
        requestSystemStatus()
    }, 1100)
}
