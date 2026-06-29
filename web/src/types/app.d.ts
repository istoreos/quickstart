declare interface AppCheckResult {
    name?: string //插件名称
    status?: "installed" | "running" | "stopped" | "uninstalled" | "not found"  //插件状态 //installed, uninstalled, not found
}