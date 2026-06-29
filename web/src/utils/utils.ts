import { useGettextLazy,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettextLazy()

export const formatDate = (value: string) => {
    // 格式化日期

}
// 时间戳
export const UnixDate = () => {
    return new Date().getTime()
}
// 字节转换
export const byteToSize = (b: number) => {
    const unit = 1000;
    if (b < unit) {
        return `${b} B`
    }
    let div = unit;
    let exp = 0;
    for (let n = b / unit; n >= unit; n /= unit) {
        div *= unit;
        exp++;
    }
    let exps = [" KB", " MB", " GB", " TB", " PB", " EB"];
    let result = ((b / 100.0) / (div / 100.0)).toFixed(1) + exps[exp];
    return result
}

// 数字次数转换
export const numberToSum = (b: number) => {
    if (b == null) {
        return 0
    }
    const unit = 10000;
    if (b < unit) {
        return b;
    }
    let sum = parseInt(`${b / 10000}`)
    let count = b % 10000
    return `${sum}万${count}`
}
// 格式化日期
export const dateForm = (time: string | null) => {
    if (time) {
        try {
            var dt = new Date(time);
            var hh: number | string = dt.getHours();
            var mm: number | string = dt.getMinutes();
            var ss: number | string = dt.getSeconds()
            if (hh < 10) {
                hh = `0${hh}`
            }
            if (mm < 10) {
                mm = `0${mm}`
            }
            if (ss < 10) {
                ss = `0${ss}`
            }
            return `${hh}:${mm}:${ss}`;
        } catch (error) {
        }
    }
    return ""
}
export const stampForm = (second?: number) => {
    if (second) {
        let days = Math.floor(second / 86400);
        let hours = Math.floor(second / 3600) % 24;
        let minutes = Math.floor(second / 60) % 60;
        let seconds = second % 60;
        let duration = (days > 0 ? $ngettext("%{ days }天", "%{ days }天", days, {days: formatNumber(days)}) : "")
            + $ngettext("%{ hours }小时", "%{ hours }小时", hours, {hours: formatNumber(hours)})
            + $ngettext("%{ minutes }分", "%{ minutes }分", minutes, {minutes: formatNumber(minutes)})
            + $ngettext("%{ seconds }秒", "%{ seconds }秒", seconds, {seconds: formatNumber(seconds)})
        return duration;
    }
}
export const checkIsIP = (ip: string) => {
    let reg = /^\d+\.\d+\.\d+\.\d+$/
    return reg.test(ip);
}


export const checkSmabaUserName = (name: string): string | boolean => {
    if (name.length < 3) {
        return $gettext("用户名太短")
    }
    const toLower = name.toLowerCase()
    if (toLower != name) {
        return $gettext("用户名只能为小写")
    }
    if (new RegExp("^\\d").exec(name)) {
        return $gettext("用户名不能以数字开头")
    }
    if (new RegExp("^_").exec(name)) {
        return $gettext("用户名不能以_开头")
    }
    if (new RegExp("^[a-z0-9_]+$").exec(name)) {
        return true
    }
    return $gettext("非法的用户名")
}

export const easyInterval = (cb: Function, delay: number) => {
    let running = true
    let tk:any = null
    const tick = () => {
        tk = null
        if (running)
            cb().finally(()=>{
                if (running)
                    tk = setTimeout(tick, delay)
            })
    }
    tk = setTimeout(tick, 0)
    return () => {
        running = false
        if (tk != null)
            clearTimeout(tk)
    }
}
