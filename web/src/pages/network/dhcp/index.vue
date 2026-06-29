<template>
    <!-- 配置的 -->
    <div id="page" v-if="setup == 0">
        <h2 class="title">{{ $gettext("配置互联网") }}</h2>
        <h3 class="desc">{{ $gettext("请确保您已将本设备 WAN 口连接到上级路由器局域网（ LAN ）接口") }}</h3>
        <div class="network-message">
            <li v-if="status == -1011" v-html="noWanTips">
            </li>
        </div>
        <form @submit.prevent="sumbitData">
            <!-- IP -->
            <label>
                <div class="label-key">
                    <span>{{ $gettext("WAN 接口配置方式") }}</span>
                </div>
                <select v-model="config.wanProto" @input="onWanProto">
                    <option value="dhcp">{{ $gettext("自动获取IP地址（DHCP）") }}</option>
                    <option value="static">{{ $gettext("静态IP地址") }}</option>
                </select>
            </label>
            <template v-if="config.wanProto == 'static'">
                <label>
                    <div class="label-key">
                        <span>{{ $gettext("IP地址") }}</span>
                    </div>
                    <input type="text" v-model.trim="config.staticIp" :placeholder="$gettext('静态IP地址')" required :disabled="disabled"
                        @input="onIP" />
                </label>
                <p class="msg" v-if="staticIpMsg">{{ staticIpMsg }}</p>
                <label>
                    <div class="label-key">
                        <span>{{ $gettext("子网掩码") }}</span>
                    </div>
                    <input type="text" v-model.trim="config.subnetMask" :placeholder="$gettext('子网掩码')" required :disabled="disabled"
                        @input="onMask" />
                </label>
                <p class="msg" v-if="subnetMaskMsg">{{ subnetMaskMsg }}</p>
                <label>
                    <div class="label-key">
                        <span>{{ $gettext("网关地址") }}</span>
                    </div>
                    <input type="text" v-model.trim="config.gateway" :placeholder="$gettext('网关地址')" required :disabled="disabled" />
                </label>
            </template>
            <!-- DNS -->
            <label>
                <div class="label-key">
                    <span>{{ $gettext("DNS 配置方式") }}</span>
                </div>
                <select v-model="config.dnsProto">
                    <option value="auto" :disabled="config.wanProto == 'static'">{{ $gettext("自动获取（DHCP）") }}</option>
                    <option value="manual">{{ $gettext("手工配置") }}</option>
                </select>
            </label>
            <template v-if="config.dnsProto == 'manual'">
                <template v-if="config.manualDnsIp != null && config.manualDnsIp.length > 0">
                    <label v-for="(item, i) in config.manualDnsIp">
                        <div class="label-key">
                            <span>{{ $gettext("DNS服务器") }}</span>
                        </div>
                        <input type="text" v-model.trim="config.manualDnsIp[i]" :placeholder="$gettext('DNS服务器')" required
                            :disabled="disabled" />
                    </label>
                </template>
                <template v-else>
                    <label>
                        <div class="label-key">
                            <span>{{ $gettext("DNS服务器") }}</span>
                        </div>
                        <input type="text" v-model.trim="dnsAddr" :placeholder="$gettext('DNS服务器')" required :disabled="disabled" />
                    </label>
                    <label>
                        <div class="label-key">{{ $gettext("备用DNS服务器") }}</div>
                        <input type="text" v-model.trim="dnsAddr2" :placeholder="$gettext('备用DNS服务器')" :disabled="disabled" />
                    </label>
                </template>
            </template>
            <div class="chose_dhcp" v-if="lanDHCPSwitch">
                <switch-box v-model="config.enableLanDhcp">
                    <span class="dhcp_info">{{ $gettext("启用LAN口DHCP服务（用于从旁路由模式恢复成默认状态）") }}</span>
                </switch-box>
            </div>
            <div class="msgs" v-if="msg">{{ msg }}</div>
            <div class="btns">
                <button class="cbi-button cbi-button-apply app-btn app-next" :disabled="disabled">{{ $gettext("保存配置") }}</button>
                <router-link :to="goType=='index'?'/':'/network'" custom v-slot="{ navigate }">
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="navigate">{{ $gettext("返回") }}</button>
                </router-link>
            </div>
        </form>
    </div>
    <!-- 配置完成 -->
    <div id="page" v-else-if="setup == 1">
        <h2 class="title">{{ $gettext("配置成功") }}</h2>
        <div class="btns">
            <router-link to="/" custom v-slot="{ navigate }">
                <button class="cbi-button cbi-button-apply app-btn app-next" @click="navigate">{{ $gettext("进入控制台") }}</button>
            </router-link>
            <router-link :to="goType=='index'?'/':'/network'" custom v-slot="{ navigate }">
                <button class="cbi-button cbi-button-remove app-btn app-back" @click="navigate">{{ $gettext("返回") }}</button>
            </router-link>
        </div>
    </div>
</template>
<script setup lang="ts">
import { ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from '/@/components/toast';
import request from '/@/request';
import utils from '/@/utils';
const noWanTips = $gettext("由于您的设备<span>没有 WAN 口</span>，无法使用本设置向导，具体请看%{link}", 
                    {link:'<a href="https://doc.linkease.com/zh/guide/istoreos/question.html#%E7%BD%91%E7%BB%9C" target="_blank" rel="noopener noreferrer">'+$gettext("链接")+'</a>'}, true)
const setup = ref(0)
const config = ref<GuideClientModel>({} as GuideClientModel)
const msg = ref<any>("")
const dnsAddr = ref("")
const dnsAddr2 = ref("")
const disabled = ref(false)
const staticIpMsg = ref("")
const subnetMaskMsg = ref("")
const status = ref(0)
const isIP = utils.checkIsIP
const lanDHCPSwitch = ref(false)
import { useRoute } from 'vue-router'
const route = useRoute()
const goType = route.query.type
const getData = async () => {
    disabled.value = true
    try {
        const resp = await Promise.all([
            request.Guide.ClientModel.GET(),
            request.Guide.GetLan.GET()
        ])
        if (resp[0]) {
            const res = resp[0]
            if (res.data) {
                const { success, error, result } = res.data
                if (result) {
                    if (result.wanProto != "dhcp" && result.wanProto != "static") {
                        result.wanProto = "dhcp"
                        result.dnsProto = "auto"
                    }
                    result.enableLanDhcp = false
                    config.value = result
                }
                if (success == -1011) {
                    status.value = success
                    disabled.value = true
                }
            }
        }
        if (resp[1].data?.result) {
            const result = resp[1].data?.result
            if (!result.enableDhcp) {
                lanDHCPSwitch.value = true
                config.value.enableLanDhcp = true
            }
        }
    } catch (error) {
        msg.value = error
    }
    if (status.value == 0) {
        disabled.value = false
    }
}
getData()


const onWanProto = (el: Event) => {
    const target = <HTMLInputElement>el.target
    if (target.value == "static") {
        if (config.value.staticIp == null || config.value.staticIp == "") {
            config.value.staticIp = "192.168.1.100"
        }
        if (config.value.subnetMask == null || config.value.subnetMask == "") {
            config.value.subnetMask = "255.255.255.0"
        }
        if (config.value.dnsProto == "auto") {
            setTimeout(()=>config.value.dnsProto = "manual", 0)
        }
    } else {
        if (config.value.dnsProto == "manual") {
            setTimeout(()=>config.value.dnsProto = "auto", 0)
        }
    }
}
const onIP = (el: Event) => {
    const target = <HTMLInputElement>el.target
    if (target.value == "") {
        staticIpMsg.value = ""
        return
    }
    if (isIP(target.value)) {
        staticIpMsg.value = ""
    } else {
        staticIpMsg.value = $gettext("请输入合法的IP地址")
    }
}
const onMask = (el: Event) => {
    const target = <HTMLInputElement>el.target
    if (target.value == "") {
        subnetMaskMsg.value = ""
        return
    }
    if (isIP(target.value)) {
        subnetMaskMsg.value = ""
    } else {
        subnetMaskMsg.value = $gettext("请输入合法的地址")
    }
}
const sumbitData = async () => {
    const _data = <GuideClientModel>{}
    switch (config.value.wanProto) {
        case "dhcp":
            break
        case "static":
            _data.staticIp = config.value.staticIp
            _data.subnetMask = config.value.subnetMask
            _data.gateway = config.value.gateway || ''
            break
    }
    switch (config.value.dnsProto) {
        case "auto":
            break
        case "manual":
            _data.manualDnsIp = []
            if (config.value.manualDnsIp != null && config.value.manualDnsIp.length > 0) {
                _data.manualDnsIp = config.value.manualDnsIp
            } else {
                _data.manualDnsIp.push(dnsAddr.value)
                if (dnsAddr2.value) {
                    _data.manualDnsIp.push(dnsAddr2.value)
                }
            }
            break
    }
    _data.dnsProto = config.value.dnsProto
    _data.wanProto = config.value.wanProto
    _data.enableLanDhcp = config.value.enableLanDhcp
    const load = Toast.Loading($gettext("配置中...."))
    disabled.value = true
    try {
        const res = await request.Guide.ClientModel.POST(_data)
        if (res?.data) {
            const { success, error } = res?.data
            if (error) {
                msg.value = error
            }
            if (success == null || success == 0) {
                Toast.Success($gettext("配置成功"))
                setup.value = 1
            }
        }
    } catch (error) {
        msg.value = error
    }
    disabled.value = false
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
        margin: 0;
        text-align: left;
        background-color: #f4f5f7;
        box-shadow: none;
    }

    .network-message {
        margin: 0.5rem 0;

        li {
            margin: 0.5rem 0;
            font-size: 20px;
            color: #000;
            font-weight: 550;

            span {
                color: #f00;
            }

            a {
                color: blue;
            }
        }
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
    .chose_dhcp {
        height: 1em;
        font-size: 1.3em;
        .dhcp_info {
            margin-left: 10px;
            user-select: none;
        }
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
        //     // height: 36px;
        //     // line-height: 36px;
        //     margin: 1rem 0;
        //     text-align: center;
        // }
    }
}
</style>