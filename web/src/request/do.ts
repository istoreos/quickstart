import { useGettextLazy,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettextLazy()

export declare interface Responser<T> extends globalThis.Response {
    data: T
}
export const Do = async<T>(url: string, data: RequestInit) => {
    return new Promise<Responser<T>>(async (resolve, reject) => {
        try {
            const response = await fetch(url, data)
            if (Math.floor(response.status / 100) != 2) {
                throw response.status + " " + response.statusText
            }
            const result = <Responser<T>>{
                ...response
            }
            result.data = <T>await response.json()
            resolve(result)
        } catch (error) {
            const err = error as Error
            reject($gettext("网络异常：") + (err?.message || error))
        }
    })
}
declare interface Configer {
    baseURL?: string
    headers?: HeadersInit
}
class Requester {
    private config: Configer = {
        baseURL: "",
        headers: {}
    }
    // 
    constructor(data: Configer) {
        if (data.baseURL) {
            this.config.baseURL = data.baseURL
        }
        if (data.headers) {
            this.config.headers = data.headers
        }
    }
    static create(data: Configer) {
        return new Requester(data)
    }
    public async Do<T>(url: string, data: RequestInit) {
        return new Promise<Responser<T>>(async (resolve, reject) => {
            try {
                const useRequest = this.useRequest({
                    baseURL: this.config.baseURL,
                    headers: this.config.headers
                })
                const baseURL = useRequest.baseURL || ""
                url = `${baseURL}${url}`
                if (data.headers == null) {
                    data.headers = {}
                }
                if (useRequest.headers) {
                    data.headers = {
                        ...useRequest.headers,
                    }
                }
                const response = await fetch(url, data)
                const result = <Responser<T>>{
                    ...response
                }
                result.data = <T>await response.json()
                resolve(this.useResponse<T>(result))
            } catch (error) {
                this.useError(error)
                reject(error)
            }
        })
    }
    public async TEXT(url: string, data: RequestInit) {
        return new Promise<Responser<string>>(async (resolve, reject) => {
            try {
                const useRequest = this.useRequest({
                    baseURL: this.config.baseURL,
                    headers: this.config.headers
                })
                const baseURL = useRequest.baseURL || ""
                url = `${baseURL}${url}`
                if (data.headers == null) {
                    data.headers = {}
                }
                if (useRequest.headers) {
                    data.headers = {
                        ...useRequest.headers,
                    }
                }
                const response = await fetch(url, data)
                const result = <Responser<string>>{
                    ...response
                }
                result.data = <string>await response.text()
                resolve(result)
            } catch (error) {
                this.useError(error)
                reject(error)
            }
        })
    }
    private useRequest = (config: Configer) => {
        return config
    }
    private useResponse = <T>(res: Responser<T>) => {
        return res
    }
    private useError = (err: any) => {
        return err
    }
    // 拦截器
    public interceptors() {
        const self = this
        return {
            requset: {
                use(f: (config: Configer) => Configer) {
                    self.useRequest = f
                }
            },
            response: {
                use(
                    f: (res: Responser<any>) => Responser<any>,
                    e?: (err: any) => any
                ) {
                    self.useResponse = f
                    if (e) {
                        self.useError = e
                    }
                }
            }
        }
    }
}
const Goxios = Requester.create({})
Goxios.interceptors().requset.use(
    (config => {
        return config
    }),
)
Goxios.interceptors().response.use(
    ((res) => {
        if (res.data) {
            if (res.data.success == null) {
                res.data.success == 0
            }
        }
        return res
    }),
)