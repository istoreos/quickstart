<template>
    <action-component :type="1">
        <transition name="rotate" mode="out-in">
            <form class="action" @submit.prevent="onSumbit">
                <div class="action-header">
                    <div class="action-header_title">{{ $gettext("Samba共享配置") }}</div>
                </div>
                <div class="action-body">
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("服务目录路径") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" :value="samba.rootPath" disabled required :style="{
                                backgroundColor: '#eee'
                            }" />
                        </div>
                    </div>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("共享名（建议使用英文字母）") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" v-model.trim="samba.shareName" required :placeholder="$gettext('共享名称')" />
                        </div>
                    </div>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("用户名") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" required :placeholder="$gettext('账号用户名')" v-model.trim="samba.username" />
                        </div>
                    </div>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("密码") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="password" v-model.trim="samba.password" />
                        </div>
                    </div>

                    <div class="samba-item">
                        <div class="samba-item_allow">
                            <input type="checkbox" id="allow" v-model="samba.allowLegacy" />
                            <label for="allow" class="samba-allow">{{ $gettext("允许旧协议与身份验证(不安全)") }}</label>
                        </div>
                        <div class="samba-item_tips">
                            <span class="tooltip-trigger">
                                <span class="samba_tip">
                                    <HintSvg></HintSvg>
                                </span>

                                <span class="samba_dir_tip">{{ $gettext("兼容一些电视或者电视盒子") }}</span>
                            </span>
                        </div>
                    </div>
                </div>
                <div class="action-footer">
                    <div class="auto"></div>
                    <button class="cbi-button cbi-button-remove app-btn app-back" type="button" @click="onClose"
                        :disabled="disabled">{{ $gettext("关闭") }}</button>
                    <button class="cbi-button cbi-button-apply app-btn app-next" :disabled="disabled">{{ $gettext("创建") }}</button>
                </div>
            </form>
        </transition>
    </action-component>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import request from '/@/request';
import ActionComponent from "/@/components/action/modal.vue"
import Toast from '../../toast';
import HintSvg from "/@/components/svg/hint.vue"
import utils from '/@/utils';
const props = defineProps({
    rootPath: {
        type: String,
        required: true
    },
    Close: Function
})
const onClose = (e: Event) => {
    e.preventDefault()
    if (props.Close) {
        props.Close()
    }
}
const disabled = ref(false)
const samba = ref<NasCreateSamba>({
    shareName: "",
    username: "",
    password: "",
    rootPath: props.rootPath,
    allowLegacy: false
})
const onSumbit = () => {
    const _samba = samba.value
    if (_samba.rootPath == "") {
        Toast.Warning($gettext("共享路径不能为空"))
        return
    }
    if (_samba.shareName == "") {
        Toast.Warning($gettext("共享名称不能为空"))
        return
    }
    if (_samba.username == "") {
        Toast.Warning($gettext("用户名不能为空"))
        return
    }
    if (_samba.password == "") {
        Toast.Warning($gettext("密码不能为空"))
        return
    }
    const checkname = utils.checkSmabaUserName(_samba.username)
    if (checkname !== true) {
        Toast.Warning(`${checkname}`)
        return
    }
    onCreateSamba(_samba)
}
const onCreateSamba = async (_samba: NasCreateSamba) => {
    disabled.value = true
    const load = Toast.Loading($gettext("创建中..."))
    try {
        const res = await request.Nas.Samba.Create.POST(_samba)
        if (res?.data) {
            const { error, result } = res.data
            if (error) {
                Toast.Warning(error)
            }
            if (result) {
                Toast.Success($gettext("创建成功"))
                window.setTimeout(() => {
                    location.reload();
                }, 1000)
            }
        }
    } catch (error) {
        Toast.Error(error as string)
    }
    load.Close()
    disabled.value = false
}
</script>
<style lang="scss" scoped>
.action {
    width: 700px;
    height: 560px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 1000;
    margin: auto;
    padding: 0 25px;
    border: 1px solid #dfdfdf;
    border-radius: 4px;
    background: #fff;
    box-shadow: 0 1px 4px rgb(0 0 0 / 30%);

    .action-header {
        width: 100%;
        height: 70px;
        line-height: 70px;

        .action-header_title {
            margin: 0;
            color: #333;
            font: inherit;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            -moz-user-select: none;
            -webkit-user-select: none;
            user-select: none;
            font-size: 20px;
        }
    }

    .action-body {
        width: 100%;
        height: calc(100% - 140px);

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

        .samba-item {
            margin-top: -18px;
            font-size: 12px;

            .samba-item_allow {
                display: flex;
                align-items: flex-end;

                .samba-allow {
                    padding-left: 10px;
                    cursor: pointer;
                }
            }

            .samba-item_tips {
                margin-top: 10px;

                .tooltip-trigger {
                    display: flex;
                }

                .samba_dir_tip {
                    margin-left: 10px;
                }
            }
        }
    }

    .action-footer {
        width: 100%;
        height: 70px;
        line-height: 70px;
        color: #333;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        padding-bottom: 30px;

        .auto {
            flex: auto;
        }

        button {
            display: inline-block;
            width: 100px !important;
            margin: 0;
            margin-left: 1rem;
        }
    }
}
</style>
