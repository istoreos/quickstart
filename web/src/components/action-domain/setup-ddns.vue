<template>
    <div class="actioner-container">
        <form class="actioner-dns" @submit.prevent="onSave">
            <div class="actioner-container_header">
                <span>{{ $gettext("域名配置向导") }}</span>
            </div>
            <div class="actioner-container_body" :class="name">
                <div class="title_info" v-if="name == 'ali'">
                    <p>
                        {{ $gettext("阿里云") }}
                    </p>
                    <span>
                        {{ $gettext("为拥有动态IP的主机配置一个固定的可访问域名") }}
                    </span>
                    <a href="https://doc.linkease.com/zh/guide/istoreos/basic/domain.html#%E9%98%BF%E9%87%8C%E4%BA%91"
                        target="_blank">{{ $gettext("查看教程") }}&gt;&gt;</a>
                </div>
                <div class="title_info" v-else-if="name == 'dnspod'">
                    <p>
                        {{ $gettext("dnspod") }}
                    </p>
                    <span>
                        {{ $gettext("为拥有动态IP的主机配置一个固定的可访问域名") }}
                    </span>
                    <a href="https://doc.linkease.com/zh/guide/istoreos/basic/domain.html#dnspod"
                        target="_blank">{{ $gettext("查看教程") }}&gt;&gt;</a>
                </div>
                <div class="title_info" v-else-if="name == 'oray'">
                    <p>
                        {{ $gettext("花生壳") }}
                    </p>
                    <span>
                        {{ $gettext("为拥有动态IP的主机配置一个固定的可访问域名") }}
                    </span>
                    <a href="https://doc.linkease.com/zh/guide/istoreos/basic/domain.html#%E8%8A%B1%E7%94%9F%E5%A3%B3"
                        target="_blank">{{ $gettext("查看教程") }}&gt;&gt;</a>
                </div>

                <div class="label-item">
                    <div class="label-item_key">
                        <span>{{ $gettext("IP地址版本：") }}</span>
                    </div>
                    <div class="label-item_value">
                        <select name="" id="" v-model="ipVersion">
                            <option value="ipv4">{{ $gettext("IPv4地址") }}</option>
                            <option value="ipv6">{{ $gettext("IPv6地址") }}</option>
                        </select>
                    </div>
                    <div class="label_tips">
                        <svg width="14px" height="14px" viewBox="0 0 14 14" version="1.1"
                            xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
                            <g id="icon_alert" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
                                <g id="Icon/Warning">
                                    <rect id="矩形" fill="#000000" fill-rule="nonzero" opacity="0" x="0" y="0" width="14"
                                        height="14" />
                                    <path
                                        d="M7,0.875 C3.61757813,0.875 0.875,3.61757813 0.875,7 C0.875,10.3824219 3.61757813,13.125 7,13.125 C10.3824219,13.125 13.125,10.3824219 13.125,7 C13.125,3.61757813 10.3824219,0.875 7,0.875 Z M6.5625,4.046875 C6.5625,3.98671875 6.61171875,3.9375 6.671875,3.9375 L7.328125,3.9375 C7.38828125,3.9375 7.4375,3.98671875 7.4375,4.046875 L7.4375,7.765625 C7.4375,7.82578125 7.38828125,7.875 7.328125,7.875 L6.671875,7.875 C6.61171875,7.875 6.5625,7.82578125 6.5625,7.765625 L6.5625,4.046875 Z M7,10.0625 C6.63769531,10.0625 6.34375,9.76855469 6.34375,9.40625 C6.34375,9.04394531 6.63769531,8.75 7,8.75 C7.36230469,8.75 7.65625,9.04394531 7.65625,9.40625 C7.65625,9.76855469 7.36230469,10.0625 7,10.0625 Z"
                                        id="形状" fill="#FAAD14" />
                                </g>
                            </g>
                        </svg>

                        <span class="info">{{ $gettext("设定哪一个 IP 地址（IPv4 或 IPv6）会被发送给 DDNS 提供商") }}</span>
                    </div>
                </div>

                <div class="label-item">
                    <div class="label-item_key">
                        <span>{{ $gettext("域名：") }}</span>
                    </div>
                    <div class="label-item_value">
                        <input type="text" placeholder="myhost.example.com" v-model.trim="domain" required>
                    </div>
                </div>

                <div class="label-item">
                    <div class="label-item_key">
                        <span>{{ $gettext("用户名：") }}</span>
                    </div>
                    <div class="label-item_value">
                        <input type="text" v-model.trim="userName" :placeholder="$gettext('请输入用户名')" required>
                    </div>
                </div>

                <div class="label-item">
                    <div class="label-item_key">
                        <span>{{ $gettext("密码：") }}</span>
                    </div>
                    <div class="label-item_value">
                        <input type="password" v-model.trim="password" :placeholder="$gettext('请输入密码')" required>
                    </div>
                </div>
            </div>
            <div class="actioner-container_footer">
                <div class="close" @click="onBack" type="button">{{ $gettext("返回") }}</div>
                <button class="next save" type="submit" :disabled="disabled">{{ $gettext("保存") }}</button>
            </div>
        </form>
    </div>
</template>
<script setup lang="ts">
import { PropType, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import request from "/@/request";
import Toast from "../toast";
import ActionConfirm from "/@/components/action-confirm"
const props = defineProps({
    name: {
        type: String,
        default: "ali"
    },
    onSetup: {
        type: Function as PropType<(v?: string) => void>,
        required: true
    },
    target: {
        type: String,
        required: true
    },
})
const emit = defineEmits([
    "update:target",
])

const ipVersion = ref("ipv4")
const serviceName = ref(props.name)
const domain = ref("")
const userName = ref("")
const password = ref("")
const disabled = ref(false)

const onBack = () => {
    props.onSetup('index')
}
const onSave = () => {
    disabled.value = true
    const load = Toast.Loading($gettext("检测中..."))
    request.Network.CheckPublickNet.POST({
        ipVersion: ipVersion.value
    }).then(res => {
        if (res?.data) {
            if (res?.data?.error) {
                Toast.Warning(res?.data.error)
                return
            }
            if ((res?.data?.success || 0) == 0) {
                const result = res.data.result
                if (result && result.address) {
                    onToNext()
                } else {
                    onConfirm()
                }
                return
            }
        } throw $gettext("未知错误")
    }).catch(error => {
        Toast.Error(error)
    }).finally(() => {
        load.Close()
        disabled.value = false
    })
}

const onConfirm = () => {
    ActionConfirm({
        title: $gettext("温馨提示"),
        nextTitle: $gettext("使用DDNSTO"),
        continuerTitle: $gettext("继续保存"),
        content: $gettext("检测到您的wan口没有公网IP或者IPv6地址，可以使用DDNSTO配置远程域名访问"),
        next() {
            onToDDNSTO()
        },
        continuer() {
            onToNext()
        },
        clear() {
        }
    })
}
const onToDDNSTO = () => {
    props.onSetup("ddnsto")
}
const onToNext = () => {
    disabled.value = true
    const load = Toast.Loading($gettext("配置中..."))
    request.Guide.PostDdns.POST({
        ipVersion: ipVersion.value,
        serviceName: serviceName.value,
        domain: domain.value,
        userName: userName.value,
        password: password.value
    }).then(res => {
        if (res?.data) {
            const { error, scope, success } = res.data
            if (error == "-100" && scope == "guide.ddns") {
                ActionConfirm({
                    title: $gettext("温馨提示"),
                    content: $gettext("检测到你有未保存的配置，可前往页面右上角点击查看，保存并应用或者恢复配置后继续"),
                    next() {
                    }
                })
                return
            }
            if (error) {
                Toast.Warning(error)
                return
            }
            if ((success || 0) == 0) {
                emit("update:target", domain.value)
                props.onSetup("ddns-success")
                return
            }
        } throw $gettext("未知错误")
    }).catch(error => {
        Toast.Error(error)
    }).finally(() => {
        load.Close()
        disabled.value = false
    })

}
</script>
<style lang="scss" scoped>
.title_info {
    display: block;
    width: 100%;
    text-align: center;

    p {
        font-size: 20px;
        margin-bottom: 10px;
    }
}

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
            color: #000;
        }

        // option {
        //     color: #999;
        // }

        input::placeholder {
            color: #999;
            font-size: 12PX;
        }
    }

    .label_tips {
        display: flex;
        margin-top: 6px;

        .info {
            margin-left: 8px;
        }
    }
}

.label-message {
    width: 100%;
    text-align: left;
    font-size: 14px;
    color: #f00;
    text-align: center;
}
</style>
