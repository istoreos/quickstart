<template>
    <div id="page" v-if="setup == 0">
        <h2 class="title">{{ $gettext("配置宽带账号") }}</h2>
        <h3 class="desc">{{ $gettext("请确保您已将路由 WAN 口连接到光猫") }}</h3>
        <div class="network-message">
            <li v-if="status == -1011" v-html="noWanTips">
            </li>
        </div>
        <form @submit.prevent="sumbitData">
            <label>
                <div class="label-key">
                    <span>{{ $gettext("宽带账号") }}</span>
                </div>
                <input type="text" v-model.trim="ppoe.account" :placeholder="$gettext('宽带账号')" required :disabled="disabled" />
            </label>
            <label>
                <div class="label-key">
                    <span>{{ $gettext("密码") }}</span>
                </div>
                <input type="password" v-model.trim="ppoe.password" :placeholder="$gettext('宽带密码')" required :disabled="disabled" />
            </label>
            <div class="chose_dhcp" v-if="lanDHCPSwitch">
                <switch-box v-model="ppoe.enableLanDhcp">
                    <span class="dhcp_info">{{ $gettext("启用LAN口DHCP服务（用于从旁路由模式恢复成默认状态）") }}</span>
                </switch-box>
            </div>
            <div class="msg" v-if="msg">{{ msg }}</div>
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
import { useRoute } from 'vue-router'
const route = useRoute()
const goType = route.query.type

import Toast from '/@/components/toast';
import request from '/@/request';
const noWanTips = $gettext("由于您的设备<span>没有 WAN 口</span>，无法使用本设置向导，具体请看%{link}", 
                    {link:'<a href="https://doc.linkease.com/zh/guide/istoreos/question.html#%E7%BD%91%E7%BB%9C" target="_blank" rel="noopener noreferrer">'+$gettext("链接")+'</a>'}, true)

const setup = ref(0)
const ppoe = ref<GuidePppoe>({} as GuidePppoe)
const msg = ref<any>("")
const disabled = ref(false)
const status = ref(0)
const lanDHCPSwitch = ref(false)

const getData = async () => {
    disabled.value = true
    try {
        const resp = await Promise.all([
            request.Guide.Pppoe.GET(),
            request.Guide.GetLan.GET()
        ])
        if (resp[0].data) {
            const { success, error, result } = resp[0].data
            if (result) {
                result.enableLanDhcp = false
                ppoe.value = result
            }
            if (success == -1011) {
                disabled.value = true
                status.value = success
            }
        }
        if (resp[1].data?.result) {
            const result = resp[1].data?.result
            if (!result.enableDhcp) {
                lanDHCPSwitch.value = true
                ppoe.value.enableLanDhcp = true
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
const sumbitData = async () => {
    const _account = ppoe.value.account || ""
    const _password = ppoe.value.password || ""
    if (_account == "") {
        msg.value = $gettext("账号不能为空")
        return
    }
    if (_password == "") {
        msg.value = $gettext("密码不能为空")
        return
    }
    disabled.value = true
    const load = Toast.Loading($gettext("配置中..."))
    try {
        const res = await request.Guide.Pppoe.POST({
            account: _account,
            password: _password,
        })
        if (res?.data) {
            const { error, success } = res.data
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

            input {
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
    .msg {
        width: 100%;
        display: block;
        height: 36px;
        line-height: 36px;
        color: #f00;
        font-size: 1.3em;
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