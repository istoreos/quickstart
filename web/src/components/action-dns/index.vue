<template>
    <action-component :Close="Close" :type="1">
        <template v-if="setup == 0">
            <form class="actioner-dns" @submit.prevent="onSumbit">
                <div class="actioner-dns_header">
                    <span>{{ $gettext("DNS配置") }}</span>
                </div>
                <div class="actioner-dns_body">
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("DNS选项") }}</span>
                        </div>
                        <div class="label-item_value">
                            <select v-model="config.dnsProto">
                                <option value="auto" :disabled="!supportsAutoDns">{{ $gettext("自动获取DNS") }}</option>
                                <option value="manual">{{ $gettext("自定义DNS") }}</option>
                            </select>
                        </div>
                    </div>
                    <template v-if="config.dnsProto == 'manual'">
                        <div class="label-item" v-for="(item, i) in config.manualDnsIp">
                            <template v-if="i == 0">
                                <div class="label-item_key"><span>{{ $gettext("DNS服务器地址") }}</span></div>
                                <div class="label-item_value">
                                    <input type="text" :placeholder="$gettext('请输入DNS地址')" required v-model.trim="config.manualDnsIp[i]" />
                                </div>
                            </template>
                            <template v-else>
                                <div class="label-item_key">{{ $gettext("备用DNS服务器地址") }}</div>
                                <div class="label-item_value">
                                    <input type="text" :placeholder="$gettext('备用DNS地址')" v-model.trim="config.manualDnsIp[i]" />
                                </div>
                            </template>
                        </div>
                    </template>
                    <div class="label-message" v-if="msg">{{ msg }}</div>
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
                    <span>{{ $gettext("DNS配置") }}</span>
                </div>
                <div class="actioner-dns_body">
                    <div class="config-message">{{ $gettext("DNS配置已保存") }}</div>
                </div>
                <div class="actioner-dns_footer">
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onFinish">{{ $gettext("完成") }}</button>
                </div>
            </div>
        </template>
    </action-component>
</template>
<script setup lang="ts">
import { ref, computed } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from "/@/components/toast";
import ActionComponent from "/@/components/action/modal.vue"
import { useNetworkStore } from '/@/plugins/store'
import request from "/@/request";
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
})
const setup = ref(0)
const networkStore = useNetworkStore()
const networkStatus = networkStore.status
const supportsAutoDns = computed(() => {
    return networkStore.status.proto != "static"
})
const dnsListUi = () => {
    let dnsList = networkStatus.dnsList || []
    dnsList = dnsList.filter(a=>a)
    while (dnsList.length < 2) {
        dnsList.push("")
    }
    return dnsList
}
const config = ref<GuideDnsConfig>({
    interfaceName: networkStatus.defaultInterface || "",
    dnsProto: networkStatus.dnsProto || "manual", //dns配置方式 manual, auto
    manualDnsIp: dnsListUi()
})
const msg = ref<any>("")
const disabled = ref(false)
const onSumbit = async () => {
    msg.value = ""
    let _data = <GuideDnsConfig>{}
    switch (config.value.dnsProto) {
        case "auto":
            break
        case "manual":
            _data.manualDnsIp = []
            if (!config.value.manualDnsIp[0]) {
                Toast.Error($gettext("至少需要填写一个DNS"))
                return
            }
            _data.manualDnsIp = config.value.manualDnsIp.filter(a=>a)

            break
    }
    _data.dnsProto = config.value.dnsProto
    _data.interfaceName = config.value.interfaceName
    const load = Toast.Loading($gettext("配置中..."))
    try {
        const res = await request.Guide.DnsConfig.POST(_data)
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
    load.Close()
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