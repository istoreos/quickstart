<template>
    <action-component :type="1">
        <transition name="rotate" mode="out-in">
            <form class="action" @submit.prevent="onSubmit">
                <div class="action-header">
                    <div class="action-header_title">{{ $gettext("局域网文件共享配置") }}</div>
                </div>
                <div class="action-body">
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("服务目录路径") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" :value="unishare.rootPath" disabled required :style="{
                                backgroundColor: '#eee'
                            }" />
                        </div>
                    </div>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("共享名（建议使用英文字母）") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" v-model.trim="unishare.shareName" required :placeholder="$gettext('共享名称')" />
                        </div>
                    </div>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("用户名") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" required :placeholder="$gettext('账号用户名')" v-model.trim="unishare.username" />
                        </div>
                    </div>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("密码") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="password" v-model.trim="unishare.password" required />
                        </div>
                    </div>

                    <div class="protocol-item">
                        <div class="protocol-item_key">{{ $gettext("共享协议") }}</div>
                        <div class="protocol-item_value">
                            <label>
                                <input type="checkbox" v-model="unishare.samba" />
                                <span>{{ $gettext("Samba") }}</span>
                            </label>
                            <label>
                                <input type="checkbox" v-model="unishare.webdav" />
                                <span>{{ $gettext("WebDAV") }}</span>
                            </label>
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
import { ref } from 'vue';
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

import request from '/@/request';
import ActionComponent from "/@/components/action/modal.vue"
import Toast from '/@/components/toast';
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
const unishare = ref<NasCreateUniShare>({
    shareName: "",
    username: "",
    password: "",
    rootPath: props.rootPath,
    samba: true,
    webdav: true
})

const validateShareName = (shareName: string) => {
    if (shareName == "") {
        Toast.Warning($gettext("共享名称不能为空"))
        return false
    }
    if (shareName.length > 15) {
        Toast.Warning($gettext("共享名称不能超过15个字符"))
        return false
    }
    if (!/^[a-z][a-z0-9_-]*$/.test(shareName)) {
        Toast.Warning($gettext("共享名称需以小写字母开头，仅支持小写字母、数字、下划线和中划线"))
        return false
    }
    return true
}

const onSubmit = async () => {
    if (disabled.value) {
        return
    }
    const _unishare = unishare.value
    if (_unishare.rootPath == "") {
        Toast.Warning($gettext("共享路径不能为空"))
        return
    }
    if (!validateShareName(_unishare.shareName)) {
        return
    }
    if (_unishare.username == "") {
        Toast.Warning($gettext("用户名不能为空"))
        return
    }
    if (_unishare.password == "") {
        Toast.Warning($gettext("密码不能为空"))
        return
    }
    if (!_unishare.samba && !_unishare.webdav) {
        Toast.Warning($gettext("请至少选择一种共享协议"))
        return
    }
    const checkname = utils.checkSmabaUserName(_unishare.username)
    if (checkname !== true) {
        Toast.Warning(`${checkname}`)
        return
    }
    await onCreateUniShare(_unishare)
}

const checkResponse = (res: { data?: { error?: string } } | undefined) => {
    const error = res?.data?.error
    if (error) {
        throw error
    }
}

const onCreateUniShare = async (_unishare: NasCreateUniShare) => {
    disabled.value = true
    const load = Toast.Loading($gettext("创建中..."))
    let success = false
    try {
        const servicesRes = await request.Share.Service.List.GET()
        checkResponse(servicesRes)
        const services = servicesRes?.data?.result?.services || []
        const serviceExists = services.some(item => item.name == _unishare.shareName)
        if (serviceExists) {
            Toast.Warning($gettext("共享名称已存在，请更换共享名"))
            return
        }

        const userName = _unishare.username
        const userPayload: ShareUserCreateRequest = {
            userName,
            password: _unishare.password
        }
        const usersRes = await request.Share.User.List.GET()
        checkResponse(usersRes)
        const users = usersRes?.data?.result?.users || []
        const userExists = users.some(item => item.userName == userName)
        const userRes = userExists
            ? await request.Share.User.Update.POST(userPayload)
            : await request.Share.User.Create.POST(userPayload)
        checkResponse(userRes)

        const servicePayload: ShareServiceCreateRequest = {
            name: _unishare.shareName,
            path: _unishare.rootPath,
            samba: _unishare.samba,
            webdav: _unishare.webdav,
            users: [{
                userName,
                rw: true
            }]
        }
        const serviceRes = await request.Share.Service.Create.POST(servicePayload)
        checkResponse(serviceRes)

        Toast.Success($gettext("创建成功"))
        success = true
        window.setTimeout(() => {
            location.reload();
        }, 1000)
    } catch (error) {
        Toast.Error(error as string)
    } finally {
        load.Close()
        if (!success) {
            disabled.value = false
        }
    }
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
    overflow: auto;
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
        overflow: auto;

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

        .protocol-item {
            width: 100%;
            margin: 1rem 0;

            .protocol-item_key {
                width: 100%;
                font-size: 12px;
                color: #666;
                margin-bottom: 8px;
            }

            .protocol-item_value {
                display: flex;
                flex-wrap: wrap;
                gap: 16px;

                label {
                    display: flex;
                    align-items: center;
                    cursor: pointer;
                    font-size: 13px;
                    color: #333;

                    input {
                        margin-right: 6px;
                    }
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
