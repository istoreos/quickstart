<template>
    <div id="page" v-if="setup == 0">
        <h2 class="title">{{ $gettext("旁路由配置前的准备工作") }}</h2>
        <code>
            {{ $gettext("旁路由模式，也叫单臂路由模式。") }}
            <p>{{ $gettext("您可以用上一级路由（主路由）拨号，然后用本路由来实现一些高级功能。") }}</p>
            <br>
            <p>{{ $gettext("本向导支持自动或手动配置：") }}</p>
            <p>{{ $gettext("点击“自动配置”按钮开始自动配置向导；") }}</p>
            <p>{{ $gettext("手动配置则需自行获取主路由器的IP地址（例如 192.168.2.1 ）和子网掩码，记录以备后续填写，点击“手动配置”按钮，切换到参数配置页，按实际情况自行填写。") }}</p>
        </code>
        <div class="btns">
            <button class="cbi-button cbi-button-success app-btn app-next" @click="setup = 3">{{ $gettext("自动配置...") }}</button>
            <button class="cbi-button cbi-button-neutral app-btn app-next" @click="onNext(false)">{{ $gettext("手动配置...") }}</button>
            <router-link :to="goType=='index'?'/':'/network'" custom v-slot="{ navigate }">
                <button class="cbi-button cbi-button-remove app-btn app-back" @click="navigate">{{ $gettext("返回") }}</button>
            </router-link>
        </div>
    </div>
    <div id="page" v-else-if="setup == 1">
        <h2 class="title">{{ $gettext("配置旁路由网络") }}</h2>
        <h3 class="desc">{{ $gettext("现在，请你配置旁路由信息") }}</h3>
        <form @submit.prevent="sumbitData">
            <label>
                <div class="label-key">
                    <span>{{ $gettext("LAN 接口配置方式") }}</span>
                </div>
                <div class="label-value">
                    <select disabled :style="{
                        backgroundColor: 'rgba(215, 215, 215, 1)',
                        color: '#333'
                    }">
                        <option>{{ $gettext("旁路由模式仅支持静态IP地址") }}</option>
                    </select>
                </div>
            </label>
            <label>
                <div class="label-key">
                    <span>{{ $gettext("IP 地址") }}</span>
                </div>
                <input type="text" v-model.trim="config.staticLanIp" :placeholder="$gettext('IP地址')" required />
            </label>
            <label>
                <div class="label-key">
                    <span>{{ $gettext("子网掩码") }}</span>
                </div>
                <input type="text" v-model.trim="config.subnetMask" :placeholder="$gettext('子网掩码')" required />
            </label>
            <label>
                <div class="label-key">
                    <span>{{ $gettext("网关地址") }}</span>
                </div>
                <input type="text" v-model.trim="config.gateway" :placeholder="$gettext('网关地址')" required />
            </label>
            <label>
                <div class="label-key">
                    <span>{{ $gettext("DNS服务器") }}</span>
                </div>
                <input type="text" v-model.trim="config.staticDnsIp" :placeholder="$gettext('223.5.5.5')" required />
            </label>

            <div class="msgs" v-if="msg">{{ msg }}</div>
            <div class="switch_inline">
                <switch-box v-model="config.enableDhcp">
                    <span class="switch_info" v-if="config.enableDhcp">{{ $gettext("提供 DHCPv4 服务（需要关闭主路由 DHCP 服务）") }}</span>
                    <span class="switch_info" v-else>{{ $gettext("提供 DHCPv4 服务") }}</span>
                </switch-box>
            </div>
            <div class="switch_inline">
                <switch-box v-model="config.dhcp6c">
                    <span class="switch_info" >{{ $gettext("自动获取 IPV6（即开启 DHCPv6 客户端）") }}</span>
                </switch-box>
            </div>
            <div class="switch_inline">
                <switch-box v-model="config.enableNat">
                    <span class="switch_info" >{{ $gettext("开启 NAT（可修复某些无线热点不能访问外网问题）") }}</span>
                </switch-box>
            </div>
            <div class="btns">
                <button class="cbi-button cbi-button-apply app-btn app-next" >{{ $gettext("保存配置") }}</button>
                <router-link :to="goType=='index'?'/':'/network'" custom v-slot="{ navigate }">
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="navigate">{{ $gettext("返回") }}</button>
                </router-link>
            </div>
        </form>
    </div>
    <!-- 配置完成 -->
    <div id="page" v-else-if="setup == 2">
        <h2 class="title">{{ $gettext("配置成功") }}</h2>
        <h3 class="desc">{{ $gettext("现在，将本路由WAN口断开，将其中一个LAN口与主路由连接，并将当前浏览器设备连接到主路由。点击“进入控制台”浏览器将跳转到新的路由IP") }}</h3>
        <div class="btns">
            <button class="cbi-button cbi-button-apply app-btn app-next" @click="gotoNewIp">{{ $gettext("进入控制台") }}</button>
        </div>
    </div>
    <!-- 自动配置 -->
    <div id="page" v-else-if="setup == 3">
        <h2 class="title">{{ $gettext("旁路由自动配置") }}</h2>
        <code>
            {{ $gettext("首先确认主路由开启了 DHCP 服务，确认本路由 WAN 口是 DHCP 客户端模式（默认即是，如果不是可使用“连接现有路由器”向导改成 DHCP 客户端），然后将本路由 WAN 口与主路由的 LAN 连接，以自动获取配置。") }}
        </code>
        {{ $gettext("1. 满足上述条件以后，点击“当前 IPv4 上游信息”以刷新当前连接信息，成功以后，“自动填写”按钮将被激活。(失败可再次点击)") }}
        <button class="cbi-button cbi-button-neutral" :class="upstreamInvalid?'cbi-button-neutral':'cbi-button-success'" :disabled="refreshDisabled" @click="refreshUpstream">{{ $gettext("当前 IPv4 上游信息（点此刷新）") }}
            <p style="text-align: left">
                <ul>
                    <li>{{ $gettext("IP 地址: ") }}{{upstream?.ipv4addr}}</li>
                    <li>{{ $gettext("子网掩码: ") }}{{upstream?.ipv4mask && iputils.prefixToMask(upstream.ipv4mask)}}</li>
                    <li>{{ $gettext("网关地址: ") }}{{upstream?.gateway}}</li>
                    <li>{{ $gettext("DNS服务器: ") }}{{upstream?.dnsList && upstream.dnsList[0] || (upstreamInvalid?"":$gettext("（无DNS服务器，请之后自行填写公共DNS服务器，例如 223.5.5.5）"))}}</li>
                </ul>
            </p>
        </button>

        {{ $gettext("2. 点击“自动填写”，将切换到参数页并自动填写。此时依然可以自行调整参数。") }}
        <div class="btns">
            <button class="cbi-button cbi-button-apply app-btn app-next" :disabled="upstreamInvalid" @click="onNext(true)">{{ $gettext("自动填写...") }}{{upstreamInvalid ? $gettext("（请先获取IPv4上游信息）") : ""}}</button>
            <router-link :to="goType=='index'?'/':'/network'" custom v-slot="{ navigate }">
                <button class="cbi-button cbi-button-remove app-btn app-back" @click="navigate">{{ $gettext("返回") }}</button>
            </router-link>
        </div>
    </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from '/@/components/toast';
import request from '/@/request';
import iputils from '/@/utils/iputils';
import { useRoute } from 'vue-router'
const route = useRoute()
const goType = route.query.type
const setup = ref(0)
const msg = ref<any>("")
const refreshDisabled = ref(false)
const upstreamInvalid = computed(()=>!(upstream.value?.ipv4addr && upstream.value?.ipv4mask && upstream.value?.gateway))

const config = ref<GuideGateway>({
    subnetMask: "255.255.255.0",
    staticDnsIp: "223.5.5.5",
    staticLanIp: "",
    gateway: "",
    enableDhcp: true,
    dhcp6c: false,
    enableNat: false,
})

const upstream = ref<NetworkStatus>()

const refreshUpstream = ()=>{
    refreshDisabled.value = true
    request.Network.Status.GET().then(res => {
        if (res?.data) {
            const { result } = res?.data
            if (result) {
                upstream.value = result;
            }
        }
    }).finally(()=>{
        refreshDisabled.value=false
    })
}

const onNext = (auto: boolean) => {
    if (auto) {
        config.value.staticLanIp = upstream.value?.ipv4addr || ""
        config.value.subnetMask = upstream.value?.ipv4mask && iputils.prefixToMask(upstream.value.ipv4mask) || ""
        config.value.gateway = upstream.value?.gateway || ""
        config.value.staticDnsIp = upstream.value?.dnsList && upstream.value?.dnsList[0] || "223.5.5.5"
    }
    setup.value = 1
}
const gotoNewIp = (event: Event) => {
    window.location.href = location.protocol + "//" + config.value.staticLanIp + (location.port ? (":" + location.port) : "")
}
const sumbitData = async () => {
    const _data = config.value
    const load = Toast.Loading($gettext("配置中..."))
    try {
        const res = await request.Guide.GatewayRouter.POST(_data)
        if (res?.data) {
            const { success, error } = res?.data
            if (error) {
                msg.value = error
            }
            if (success == null || success == 0) {
                setTimeout(()=>{
                    setup.value = 2
                    load.Close()
                }, 5000)
                return
            }
        }
    } catch (error) {
        msg.value = error
    }
    load.Close()
}
</script>
<style lang="scss" scoped>
#page {
    width: 100%;
    padding: 1rem;
    margin: 0 auto;
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-start;
    max-width: 600px;
    margin-top: 100px;

    h2.title {
        width: 100%;
        display: block;
        color: #1e1e1e;
        font-size: 3em;
        padding: 0;
        margin: 0;
        text-align: left;
        background-color: #f4f5f7;
        box-shadow: none;
        margin-bottom: 10px;
    }

    h3.desc {
        width: 100%;
        display: block;
        color: #666;
        font-size: 1.2em;
        padding: 0;
        text-align: left;
        background-color: #f4f5f7;
        box-shadow: none;
    }

    code {
        background-color: #eee;
        display: block;
        width: 100%;
        font-size: 1.3em;
        padding: 1rem;
        line-height: 2;
        margin: 2rem 0;
    }

    div.info {
        width: 100%;
        display: block;
        margin: 1rem 0;
        font-size: 1.3em;
        text-align: left;
    }

    .msgs {
        width: 100%;
        display: block;
        height: 36px;
        line-height: 36px;
        color: #f00;
        font-size: 1.3em;
    }

    p.msg {
        width: 100%;
        display: block;
        color: #f00;
        font-size: 1em;
    }

    .btns {
        width: 100%;
        display: block;
        margin-top: 3rem;

        button {
            display: block;
            width: 100% !important;
            margin: 0.5rem 0;
        }

        // a {
        //     cursor: pointer;
        //     display: block;
        //     width: 100% !important;
        //     height: 36px;
        //     line-height: 36px;
        //     margin: 1rem 0;
        //     text-align: center;
        // }
    }

    form {
        display: block;
        width: 100%;
        margin: 3rem 0;

        label {
            display: block;
            width: 100%;
            margin: 1rem 0;

            .label-key {
                display: block;
                width: 100%;
                font-size: 1.3em;
                margin-bottom: 0.5rem;

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

            input,
            select {
                width: 100%;
                display: block;
                height: 42px;
            }
        }
    }
}

.switch_inline {
    height: 1em;
    font-size: 1.3em;

    .switch_info {
        margin-left: 10px;
        user-select: none;
    }
}
</style>