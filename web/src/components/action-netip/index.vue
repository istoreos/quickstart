<template>
    <action-component :Close="Close" :type="1">
        <template v-if="setup == 0">
            <form class="actioner-dns" @submit.prevent="onSumbit">
                <div class="actioner-dns_header">
                    <span>{{ $gettext("内网配置") }}</span>
                </div>
                <div class="actioner-dns_body">
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("IPv4地址") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" placeholder="192.168.100.1" required v-model.trim="lan.lanIp"
                                @change="updateIpPool">
                        </div>
                    </div>

                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("IPv4子网掩码") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" placeholder="255.255.255.0" required v-model.trim="lan.netMask"
                                @change="updateIpPool">
                        </div>
                    </div>

                    <div class="chose_dhcp" v-if="showDhcp">
                        <switch-box v-model="lan.enableDhcp">
                            <span class="dhcp_info" v-if="lan.enableDhcp">{{ $gettext("修改DHCP服务") }}</span>
                            <span class="dhcp_info" v-else>{{ $gettext("保持DHCP服务设置") }}</span>
                        </switch-box>
                    </div>

                    <template v-if="lan.enableDhcp">
                        <div class="label-item">
                            <div class="label-item_key">
                                <span>{{ $gettext("IP池起始地址") }}</span>
                            </div>
                            <div class="label-item_value">
                                <input type="text" placeholder="192.168.100.100" required v-model.trim="lan.dhcpStart">
                            </div>
                        </div>

                        <div class="label-item">
                            <div class="label-item_key">
                                <span>{{ $gettext("IP池结束地址") }}</span>
                            </div>
                            <div class="label-item_value">
                                <input type="text" placeholder="192.168.100.100" required v-model.trim="lan.dhcpEnd">
                            </div>
                        </div>
                    </template>

                </div>

                <div class="actioner-dns_footer">
                    <button class="cbi-button cbi-button-apply app-btn" :disabled="disabled">{{ $gettext("确认") }}</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onClose">{{ $gettext("取消") }}</button>
                </div>
            </form>
        </template>
        <template v-else-if="setup == 1">
            <div class="actioner-dns">
                <div class="actioner-dns_header">
                    <span>{{ $gettext("更换配置") }}</span>
                </div>
                <!-- 配置结果 -->
                <div class="actioner-dns_body">

                    <!-- 配置成功 -->
                    <div class="setting_status" v-if="finalResult == 'success'">
                        <div class="success_icon">
                            <svg t="1642063181211" class="icon" viewBox="0 0 1024 1024" version="1.1"
                                xmlns="http://www.w3.org/2000/svg" p-id="5062" width="128" height="128">
                                <path
                                    d="M512 85.333333c235.648 0 426.666667 191.018667 426.666667 426.666667s-191.018667 426.666667-426.666667 426.666667S85.333333 747.648 85.333333 512 276.352 85.333333 512 85.333333z m-74.965333 550.4L346.453333 545.152a42.666667 42.666667 0 1 0-60.330666 60.330667l120.704 120.704a42.666667 42.666667 0 0 0 60.330666 0l301.653334-301.696a42.666667 42.666667 0 1 0-60.288-60.330667l-271.530667 271.488z"
                                    fill="#52C41A" p-id="5063" />
                            </svg>
                        </div>
                        <div class="config-message">{{ $gettext("配置成功") }}</div>
                        <a :href="locationUrl" class="NewAdress">{{ $gettext("%{ countdown }s后 跳转新地址", {countdown: ''+countdown}) }}</a>
                    </div>

                    <!-- 配置失败 -->
                    <div class="setting_status" v-else-if="finalResult == 'fail'">
                        <div class="success_icon">
                            <svg t="1642063200324" class="icon" viewBox="0 0 1024 1024" version="1.1"
                                xmlns="http://www.w3.org/2000/svg" p-id="5898" width="128" height="128">
                                <path
                                    d="M549.044706 512l166.189176-166.249412a26.383059 26.383059 0 0 0 0-36.98447 26.383059 26.383059 0 0 0-37.044706 0L512 475.015529l-166.249412-166.249411a26.383059 26.383059 0 0 0-36.98447 0 26.383059 26.383059 0 0 0 0 37.044706L475.015529 512l-166.249411 166.249412a26.383059 26.383059 0 0 0 0 36.98447 26.383059 26.383059 0 0 0 37.044706 0L512 548.984471l166.249412 166.249411a26.383059 26.383059 0 0 0 36.98447 0 26.383059 26.383059 0 0 0 0-37.044706L548.984471 512zM512 1024a512 512 0 1 1 0-1024 512 512 0 0 1 0 1024z"
                                    fill="#E84335" p-id="5899" />
                            </svg>
                        </div>
                        <div class="config-message">{{ $gettext("配置失败") }}</div>
                        <p>{{ $gettext("请尝试重新配置") }}</p>
                        <button class="cbi-button cbi-button-apply app-btn" @click="onFinish">{{ $gettext("我知道了") }}</button>
                    </div>

                    <!-- 配置超时 -->
                    <div class="setting_status" v-else-if="finalResult == 'timeout'">
                        <div class="success_icon">
                            <svg width="128px" height="128px" viewBox="0 0 128 128" version="1.1"
                                xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
                                <g id="icon_yellow" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
                                    <g id="Icon/Warning">
                                        <rect id="矩形" fill="#000000" fill-rule="nonzero" opacity="0" x="0" y="0"
                                            width="128" height="128">
                                        </rect>
                                        <path
                                            d="M64,8 C33.075,8 8,33.075 8,64 C8,94.925 33.075,120 64,120 C94.925,120 120,94.925 120,64 C120,33.075 94.925,8 64,8 Z M60,37 C60,36.45 60.45,36 61,36 L67,36 C67.55,36 68,36.45 68,37 L68,71 C68,71.55 67.55,72 67,72 L61,72 C60.45,72 60,71.55 60,71 L60,37 Z M64,92 C60.6875,92 58,89.3125 58,86 C58,82.6875 60.6875,80 64,80 C67.3125,80 70,82.6875 70,86 C70,89.3125 67.3125,92 64,92 Z"
                                            id="形状" fill="#FAAD14"></path>
                                    </g>
                                </g>
                            </svg>
                        </div>
                        <div class="config-message">{{ $gettext("配置超时") }}</div>
                        <p>{{ $gettext("路由器 IP 可能已经修改成功。若刷新页面失败，请重新连接路由器，否则请尝试重新配置。") }}</p>
                        <button class="cbi-button cbi-button-apply app-btn" @click="onFinish">{{ $gettext("刷新页面") }}</button>
                    </div>

                </div>
            </div>
        </template>
    </action-component>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from "/@/components/toast";
import ActionComponent from "/@/components/action/modal.vue"
import request from "/@/request";
import SwitchBox from "/@/components/switch/index.vue"
import { config } from "process";
import iputils from '/@/utils/iputils'

const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
})

const setup = ref(0)

const lan = ref<GuideLanSetting>({
    lanIp: "",
    netMask: "255.255.255.0",
    enableDhcp: false,
    dhcpStart: "",
    dhcpEnd: "",
})

const msg = ref<any>("")
const disabled = ref(false)
const isShowOpen = ref(true)
const isShowClose = ref(false)
const locationUrl = ref("")
const countdown = ref(2)
const showDhcp = ref(false)

const finalResult = ref("timeout")

const ShowClose = () => {
    lan.value.enableDhcp = false
}

const ShowOpen = () => {
    lan.value.enableDhcp = true
}

let accessFromLan: boolean = true

//获取lan口网络信息接口
const getDateLan = () => {
    request.Guide.GetLan.GET().then(res => {
        if (res.data.result) {
            showDhcp.value = res.data.result.enableDhcp || false
            res.data.result.enableDhcp = false
            lan.value = res.data.result
            if (res.data.result.lanIp !== location.hostname) {
                accessFromLan = false
            }
        }
    })
}
getDateLan()

const updateIpPool = () => {
    const lan0 = lan.value
    if (!iputils.isValidIPv4(lan0.lanIp)) {
        Toast.Warning($gettext("IPv4地址格式错误"))
        return
    }
    if (!iputils.isValidMask(lan0.netMask)) {
        Toast.Warning($gettext("IPv4子网掩码格式错误"))
        return
    }
    const range = iputils.calcMaskRange(lan0.lanIp, lan0.netMask)
    lan0.dhcpStart = range[0]
    lan0.dhcpEnd = range[1]
    lan.value = lan0
}

//提交更换内网地址接口
const onSumbit = () => {
    const lan0 = lan.value
    if (!iputils.isValidIPv4(lan0.lanIp)) {
        Toast.Warning($gettext("IPv4地址格式错误"))
        return
    }
    if (!iputils.isValidMask(lan0.netMask)) {
        Toast.Warning($gettext("IPv4子网掩码格式错误"))
        return
    }
    if (lan0.enableDhcp && !(iputils.isValidIPv4(lan0.dhcpStart) && iputils.isValidIPv4(lan0.dhcpEnd) && iputils.isValidMaskRange(lan0.lanIp, lan0.netMask, lan0.dhcpStart, lan0.dhcpEnd))) {
        Toast.Warning($gettext("DHCP的IP池格式错误或超出子网范围"))
        return
    }

    const load = Toast.Loading($gettext("正在配置…请稍等"), 30)
    let stage = 0
    const setStage = (result: string)=>{
        finalResult.value = result
        setup.value = 1
        stage = 1
        load.Close()
    }

    const checkOnline = () => {
        const end = new Date().getTime() + 30000
        const newLocaltion = accessFromLan ? (location.protocol + "//" + lan0.lanIp + (location.port ? (":" + location.port) : "")) : location.origin
        const target = newLocaltion + "/luci-static/resources/icons/loading.gif"
        const rejectFn = () => {
            if (stage != 0)
                return
            if (new Date().getTime() > end) {
                setStage("timeout")
            } else {
                window.setTimeout(check, 2000);
            }
        }
        const resolveFn = () => {
            if (stage != 0)
                return
            locationUrl.value = newLocaltion + location.pathname
            setStage("success")
            window.setTimeout(() => {
                countdown.value = 1
            }, 1000)
            window.setTimeout(() => {
                location.href = locationUrl.value
            }, 2000);
        }
        const check = () => {
            if (stage != 0)
                return
            console.log("check online ", target)
            const img = new Image()

            img.onload = resolveFn
            img.onerror = rejectFn

            img.src = target
        }
        window.setTimeout(check, 5000)
    }

    request.Guide.LanIp.POST(lan0)
        .then(res => {
            if (res?.data) {
                if ((res.data.success || 0) == 0) {
                    //setup.value = 1
                    return
                } else if (res.data?.error) {
                    throw res.data.error
                }
            }
            throw $gettext("未知错误")
        })
        .catch(error => { 
            if (stage != 0)
                return
            setStage("fail")
            Toast.Error(error)
        })

    checkOnline()
    window.setTimeout(()=>{
        if (stage != 0)
            return
        setStage("timeout")
    }, 30000);
}

const onClose = (e: Event) => {
    e.preventDefault();
    if (props.Close) {
        props.Close()
    }
}

const onFinish = (e: Event) => {
    location.reload()
}

</script>
<style lang="scss" scoped>
.actioner-dns {
    width: 860px;
    background-color: #fff;
    position: relative;
    z-index: 99999;
    margin: auto;
    overflow: auto;

    .actioner-dns_header {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        padding: 1rem;
        font-size: 2em;
        border-bottom: 1px solid #eee;
    }

    .actioner-dns_body {
        padding: 1rem;
        min-height: 50vh;

        .label-item {
            width: 100%;
            margin: 1rem 0;

            .label-item_key {
                width: 100%;
                font-size: 12px;
                color: #666;

                span {
                    white-space: nowrap;
                    overflow: hidden;
                    text-overflow: ellipsis;
                }

                span:before {
                    content: "*";
                    color: #f56c6c;
                    margin-right: 4px;
                }
            }

            .label-item_value {
                width: 100%;
                margin-top: 5px;

                select,
                input {
                    width: 100%;
                    height: 36px;
                }
            }
        }
        .chose_dhcp {
            height: 1em;
            font-size: 1.3em;
            .dhcp_info {
                margin-left: 10px;
                user-select: none;
            }
        }
        .label-message {
            width: 100%;
            text-align: left;
            font-size: 14px;
            color: #f00;
            text-align: center;
        }
    }

    .config-message {
        width: 100%;
        min-height: inherit;
        height: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: center;
        font-size: 2em;
    }

    .actioner-dns_footer {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: flex-end;
        padding: 1rem;
        font-size: 2em;
        border-top: 1px solid #eee;

        button {
            display: inline-block;
            width: 100px !important;
            margin: 0;
            margin-left: 1rem;
        }
    }
}

.setting_status {
    text-align: center;

    p {
        margin: 10px 0;
    }

    a {
        text-align: center;
        display: block;
        text-decoration: none;
    }
}

.NewAdress {
    margin-top: 10px;
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 1400px) {
    .actioner-dns {
        .actioner-dns_body {
            min-height: 34vh;
        }
    }
}

@media screen and (max-width: 800px) {
    .actioner-dns {
        width: 100%;
    }
}

</style>