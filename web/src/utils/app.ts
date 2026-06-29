import request from '/@/request';
import Toast from "/@/components/toast";
import { useGettextLazy,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettextLazy()

const exports =  {
    installApp : (name: string, timeoutInSeconds?: number) => {
        return new Promise<boolean>((resolve, reject) => {
            let stage = 0
            request.App.Install.POST({
                name: name,
            }).then(() => {
                const timeoutChecker = setTimeout(()=>{
                    if (stage != 0)
                        return
                    stage = 1
                    resolve(false)
                }, (timeoutInSeconds || 60) * 1000)

                const check = ()=>{
                    if (stage != 0)
                        return
                    request.App.Check.POST({
                        name: name
                    }).then(res => {
                        if (stage != 0)
                            return
                        if (res?.data) {
                            const { result } = res.data
                            if (result?.status == "installed") {
                                clearTimeout(timeoutChecker)
                                stage = 1
                                resolve(true)
                                return
                            }
                        }
                    }).catch(err => {

                    }).finally(()=>{
                        if (stage != 0)
                            return
                        setTimeout(check, 3000)
                    })
                }
                setTimeout(check, 3000)
            }).catch(err => {
                if (stage != 0)
                    return
                stage = 1
                reject($gettext("安装失败，") + err)
            })
        })
    },
    checkAndInstallApp : async (pkg: string, app: string, installPkg?: string) => {
        let load = Toast.Loading($gettext("检查中..."))
        try {
            const res = await request.App.Check.POST({
                name: pkg
            })
            load.Close()
            if (res?.data) {
                const { result, error } = res.data
                if (error) {
                    Toast.Warning(error)
                } else if (result) {
                    if (result.status == "installed") {
                        return true
                    } else {
                        if (confirm($gettext("检测到你尚未安装 %{name} 插件,是否安装？", {name:app}))) {
                            load = Toast.Loading($gettext("正在安装中..."))
                            const is = await exports.installApp(installPkg || pkg)
                            load.Close()
                            if (is) {
                                return true
                            } else {
                                Toast.Error($gettext("安装失败或超时，请检查软件源或稍候重试"))
                            }
                        }
                    }
                } else {
                    Toast.Warning($gettext("检查插件状态失败"))
                }
            }
            return false
        } catch (error) {
            load.Close()
            Toast.Warning(error as string)
            return false
        }
    },
    installAndGo : async (pkg: string, app: string, href: string, installPkg?: string) => {
        if (await exports.checkAndInstallApp(pkg ,app, installPkg)) {
            location.href = href
        }
    }
}

export default exports
