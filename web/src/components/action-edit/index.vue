<template>
    <action-component :Close="Close" :type="1">
        <form class="actioner-dns" @submit.prevent="onSumbit">
            <div class="actioner-dns_header">
                <template v-if="name == 'wan'">
                    <span>{{ e == 'edit' ? $gettext("编辑WAN") : $gettext("添加WAN") }}</span>
                </template>
                <template v-else>
                    <span>{{ e == 'edit' ? $gettext("编辑LAN") : $gettext("添加LAN") }}</span>
                </template>
            </div>
            <div class="actioner-dns_body">
                <div class="label-item">
                    <div class="label-item_key">
                        <span>{{ $gettext("名称") }}</span>
                    </div>
                    <div class="label-item_value">
                        <span>{{ infacer.name.toLocaleUpperCase() }}</span>
                    </div>
                </div>
                <div class="label-item">
                    <div class="label-item_key">
                        <span>{{ $gettext("协议（网络获取方式）") }}</span>
                    </div>
                    <div class="label-item_value">
                        <select v-model="infacer.proto">
                            <option value="dhcp">{{ $gettext("DHCP客户端") }}</option>
                            <option value="pppoe" v-if="name == 'wan'">PPPoE</option>
                            <option value="static">{{ $gettext("静态地址") }}</option>
                        </select>
                    </div>
                </div>
            </div>
            <div class="actioner-dns_footer">
                <button class="cbi-button cbi-button-apply app-btn" :disabled="disabled">{{ $gettext("保存") }}</button>
                <button class="cbi-button cbi-button-remove app-btn app-back" @click="onClose">{{ $gettext("取消") }}</button>
            </div>
        </form>
    </action-component>
</template>
<script setup lang="ts">
import { ref, reactive, PropType } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from "/@/components/toast";
import ActionComponent from "/@/components/action/modal.vue"
import request from "/@/request";
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
    e: {
        type: String as PropType<"edit" | "add">,
        required: true
    },
    name: {
        type: String as PropType<FirewallType>,
        required: true
    },
    inface: {
        type: Object as PropType<NetworkInterfaceInfo>,
        required: true
    },
    next: {
        type: Function as PropType<(inface: NetworkInterfaceInfo) => void>,
        required: true
    }
})
const disabled = ref(false)
const infacer = ref(props.inface)
const onSumbit = async () => {
    const load = Toast.Loading($gettext("配置中..."))
    load.Close()
    props.next(infacer.value)
    onClose()
}
const onClose = () => {
    if (props.Close) {
        props.Close()
    }
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
                font-size: 16px;
                color: #666;
                margin-bottom: 10px;

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
                    min-height: 36px;
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