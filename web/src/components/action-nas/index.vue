<template>
    <action-component :Close="Close" :type="1" v-if="show">
        <transition name="rotate" mode="out-in">
            <div class="action" v-if="setup == 0">
                <h2 class="title">{{ $gettext("欢迎使用 NAS 配置向导") }}</h2>
                <h3 class="desc">{{ $gettext("请选择需要添加的NAS服务") }}</h3>
                <form>
                    <label>
                        <select v-model="service">
                            <option value="linkease">{{ $gettext("跨设备共享（易有云）") }}</option>
                            <option value="unishare">{{ $gettext("局域网文件共享（Samba + WebDAV）") }}</option>
                        </select>
                    </label>
                </form>
                <div class="tips" v-if="feature('unishare')" v-html="$gettext('如需对 Samba 或 WebDAV 进行更细致的权限控制，请使用“%{unishare}”', {unishare:'<a href=&quot;/cgi-bin/luci/admin/nas/unishare&quot;>' + $gettext('统一文件共享') + '</a>'}, true)"></div>
                <div class="btns">
                    <button class="cbi-button cbi-button-apply app-btn app-next" @click="onNext" type="button"
                        :disabled="disabled">{{ $gettext("下一步") }}</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onClose"
                        type="button">{{ $gettext("取消") }}</button>
                </div>
            </div>
        </transition>
    </action-component>
</template>
<script setup lang="ts">
import { PropType, ref } from "vue";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

import ActionComponent from "/@/components/action/modal.vue"
import ActionDisk from "/@/components/action-disk";
import ActionLinkease from "./linkease"
import ActionUniShare from "./unishare"
import appUtils from "/@/utils/app";
import { feature } from "/@/utils/features"

const props = defineProps({
    setup: Number as PropType<number>,
    Close: {
        type: Function,
        required: true
    },
})
const show = ref(true)
const service = ref("unishare")
const disabled = ref(false)
const setup = ref(props.setup || 0)

const onClose = () => {
    if (props.Close) {
        props.Close()
    }
}
const onNext = async () => {
    switch (service.value) {
        case "unishare":
            await checkIsInstallUniShare()
            break
        case "linkease":
            await checkIsInstallLinkease()
            break
    }
}

//检测是否安装了Linkease
const checkIsInstallLinkease = async () => {
    disabled.value = true
    if (await appUtils.checkAndInstallApp("linkease", $gettext("易有云"), "app-meta-linkease")) {
        onLinkease()
    }
    disabled.value = false
}

//检测是否安装了UniShare
const checkIsInstallUniShare = async () => {
    disabled.value = true
    const apps = [
        { pkg: "luci-app-unishare", app: $gettext("统一文件共享") },
        { pkg: "unishare", app: "UniShare" },
        { pkg: "samba4-server", app: "Samba" },
        { pkg: "webdav2", app: "WebDAV" },
    ]
    for (const item of apps) {
        if (!await appUtils.checkAndInstallApp(item.pkg, item.app)) {
            disabled.value = false
            return
        }
    }
    onDisk()
    disabled.value = false
}

const onDisk = () => {
    disabled.value = false
    show.value = false
    ActionDisk({
        Cancel: () => {
            show.value = true
        },
        Next: (rootPath: string) => {
            switch (service.value) {
                case "unishare":
                    onUniShare(rootPath)
                    break
            }
        }
    })
}
const onLinkease = () => {
    ActionLinkease({})
    onClose()
}
const onUniShare = (rootPath: string) => {
    ActionUniShare({
        rootPath: rootPath
    })
    onClose()
}
</script>
<style lang="scss" scoped>
.action {
    width: 860px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 1000;
    margin: auto;
    overflow: auto;
    padding: 1rem 87px;
    border-radius: 6px;

    h2.title {
        width: 100%;
        display: block;
        color: #1e1e1e;
        font-size: 3em;
        padding: 0;
        margin: 0;
        text-align: center;
    }

    h3.desc {
        width: 100%;
        display: block;
        color: #666;
        font-size: 1.2em;
        padding: 0;
        margin: 1rem 0;
        text-align: center;
    }

    form {
        width: 100%;
        display: block;
        padding: 2rem 0;

        label {
            width: 100%;
            display: block;
            margin: 1rem 0;

            input,
            select {
                width: 100%;
                display: block;
                height: 42px;
            }
        }
    }

    .btns {
        width: 100%;
        margin-top: 3rem;

        button {
            display: block;
            width: 100% !important;
            margin: 0.5rem 0;
        }
    }

    li.disk-item {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: space-between;
        width: 100%;
        padding: 5px 1rem;
        border-bottom: 1px solid #eee;
        cursor: pointer;

        &:hover {
            background-color: #eee;
        }

        .disk-item_f {
            display: flex;
            flex-wrap: wrap;

            .disk-item_venderModel {
                width: 100%;
            }

            .disk-item_used {
                width: 100%;
            }
        }
    }
    .tips {
        float: right;
        font-size: 0.8em;
    }
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 500px) {
    .action {
        h2.title {
            font-size: 2em;
        }
    }
}
</style>
