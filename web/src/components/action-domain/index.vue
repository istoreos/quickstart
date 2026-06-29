<template>
    <action-component :Close="Close" :type="1">
        <div class="action-main">
            <template v-if="setup == 'index'">
                <setup-index :onSetup="onSetup" v-model:active="active" />
            </template>
            <template v-else-if="setup == 'ddns-ali'">
                <setup-ddns :onSetup="onSetup" v-model:target="target" name="ali" />
            </template>
            <template v-else-if="setup == 'ddns-dnspod'">
                <setup-ddns :onSetup="onSetup" v-model:target="target" name="dnspod" />
            </template>
            <template v-else-if="setup == 'ddns-oray'">
                <setup-ddns :onSetup="onSetup" v-model:target="target" name="oray" />
            </template>
            <template v-else-if="setup == 'ddns-success'">
                <setup-ddns-success :onSetup="onSetup" :target="target" />
            </template>
        </div>
    </action-component>
</template>
<script setup lang="ts">
import { reactive, ref } from "vue";
import ActionComponent from "/@/components/action/modal.vue"
import setupIndex from "./setup-index.vue"
import SetupDdnsSuccess from "./setup-ddns-success.vue"
import ActionDdnsto from "/@/components/action-ddnsto"
import SetupDdns from "./setup-ddns.vue";
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
const target = ref("")
const setup = ref("index")
const onSetup = (v?: string) => {
    if (v != null) {
        if (v == "ddnsto") {
            onDdnsto()
            return
        }
        setup.value = v
    } else {
        onClose()
    }
}
const onDdnsto = () => {
    onClose()
    ActionDdnsto({
        url: props.url
    })
}
const onClose = () => {
    if (props.Close) {
        props.Close()
    }
}
const active = ref("ddnsto")
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

                &.save {
                    height: 32px;
                    background: #553AFE;
                    border-radius: 2px;
                    line-height: 16px;
                }
            }

            .next:hover,
            .close:hover {
                opacity: 0.9;
            }
        }

        .actioner-container_body {
            padding: 1rem;
            width: 100%;
            height: 400px;

            a {
                text-decoration: none;
            }
        }

        .actioner-container_body.ali,
        .actioner-container_body.dnspod,
        .actioner-container_body.oray {
            height: 451px;
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