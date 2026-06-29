<template>
    <action-component :type="1">
        <div class="action-main">
            <template v-if="setup == 'ddnsto-install'">
                <setup-ddnsto-install :onSetup="onSetup" />
            </template>
            <template v-else-if="setup == 'ddnsto-login'">
                <setup-ddnsto-login :onSetup="onSetup" :onDdnstoConfig="onDdnstoConfig" />
            </template>
            <template v-else-if="setup == 'ddnsto-run'">
                <setup-ddnsto-run :onSetup="onSetup" :token="ddnstConfig.token"
                    :onDdnstoLocalConfig="onDdnstoLocalConfig" />
            </template>
            <template v-else-if="setup == 'ddnsto-bind'">
                <setup-ddnsto-bind :onSetup="onSetup" :config="{
                    token: ddnstConfig.token,
                    sign: ddnstConfig.sign,
                    domain: ddnstConfig.domain,
                    netaddr: ddnstConfig.netaddr,
                    routerId: ddnstConfig.routerId
                }" v-model:domain="ddnstConfig.domain" />
            </template>
            <template v-else-if="setup == 'ddnsto-save'">
                <setup-ddnsto-save :onSetup="onSetup" :target="ddnstConfig.domain" />
            </template>
        </div>
    </action-component>
</template>
<script setup lang="ts">
import { reactive, ref } from "vue";
import ActionComponent from "/@/components/action/modal.vue"
import setupDdnstoLogin from "./setup-ddnsto-login.vue"
import setupDdnstoBind from "./setup-ddnsto-bind.vue"
import setupDdnstoSave from "./setup-ddnsto-save.vue"
import SetupDdnstoLogin from "./setup-ddnsto-login.vue"
import SetupDdnstoInstall from "./setup-ddnsto-install.vue"
import SetupDdnstoRun from "./setup-ddnsto-run.vue"
import SetupDdnstoSave from "./setup-ddnsto-save.vue"
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
    url: {
        type: String,
        required: true
    }
})
const setup = ref("ddnsto-install")

const onSetup = (v?: string) => {
    if (v != null) {
        setup.value = v
    } else {
        onClose()
    }
}
const onClose = () => {
    if (props.Close) {
        props.Close()
    }
}


const ddnstConfig = reactive({
    sign: "",
    token: "",
    domain: props.url,
    netaddr: '',
    routerId: ''
})
const onDdnstoConfig = (sign: string, token: string) => {
    ddnstConfig.sign = sign
    ddnstConfig.token = token
}
const onDdnstoLocalConfig = (netaddr: string, routerId: string) => {
    ddnstConfig.netaddr = netaddr
    ddnstConfig.routerId = routerId
}
</script>
<style lang="scss" scoped>
.action-main {
    width: 680px;
    background-color: #fff;
    position: relative;
    z-index: 99999;
    margin: auto;
    overflow: auto;

    :deep(.actioner-container) {
        width: 100%;

        .actioner-container_header {
            width: 100%;
            height: 50px;
            line-height: 50px;
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            font-size: 20px;
            border-bottom: 1px solid #eee;
            justify-content: center;
            padding: 0 10px;
        }

        .actioner-container_footer {
            width: 100%;
            height: 50px;
            border-top: 1px solid rgba(0, 0, 0, 0.06);
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            justify-content: flex-end;
            padding: 0 30px;

            button {
                display: inline-block;
                width: 100px !important;
                margin: 0;
                margin-left: 1rem;
            }

            .close {
                min-width: 65px;
                font-family: PingFangSC-Regular, PingFang SC;
                font-weight: 400;
                color: #0060ff;
                line-height: 30px;
                text-align: center;
                cursor: pointer;
                height: 32px;
                border-radius: 2px;
                border: 1px solid rgba(0, 0, 0, 0.15);
                font-size: 14px;
                font-family: PingFangSC-Regular, PingFang SC;
                color: rgba(0, 0, 0, 0.83);
                line-height: 32px;
            }

            .next {
                min-width: 65px;
                font-family: PingFangSC-Regular, PingFang SC;
                margin-left: 20px;
                line-height: 32px;
                text-align: center;
                cursor: pointer;
                font-size: 14px;
                font-family: PingFangSC-Regular, PingFang SC;
                font-weight: 400;
                color: #fff;
                margin-left: 20px;
                width: 74px;
                height: 32px;
                background: #553AFE;
                border-radius: 2px;
            }

            .next:hover,
            .close:hover {
                opacity: 0.9;
            }
        }

        .actioner-container_body {
            padding: 1rem;
            text-align: center;
            width: 100%;
            height: 400px;

            a {
                text-decoration: none;
            }
        }


        .actioner-container_body.ddnsto-bind {
            height: 280px;
        }
    }
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 800px) {
    .action-main {
        width: 90%;
    }
}
</style>