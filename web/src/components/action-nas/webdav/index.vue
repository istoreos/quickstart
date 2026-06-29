<template>
    <action-component :type="1">
        <transition name="rotate" mode="out-in">
            <form class="action" @submit.prevent="onSumbit">
                <div class="action-header">
                    <div class="action-header_title">{{ $gettext("Webdav共享配置") }}</div>
                </div>
                <div class="action-body">
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("服务目录路径") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input
                                type="text"
                                :value="webdav.rootPath"
                                disabled
                                required
                                :style="{
                                    backgroundColor: '#eee'
                                }"
                            />
                        </div>
                    </div>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("用户名") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input
                                type="text"
                                required
                                :placeholder="$gettext('账号用户名')"
                                v-model.trim="webdav.username"
                            />
                        </div>
                    </div>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("密码") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="password" v-model.trim="webdav.password" />
                        </div>
                    </div>
                </div>
                <div class="action-footer">
                    <div class="auto"></div>
                    <button
                        class="cbi-button cbi-button-remove app-btn app-back"
                        type="button"
                        @click="onClose"
                        :disabled="disabled"
                    >{{ $gettext("关闭") }}</button>
                    <button
                        class="cbi-button cbi-button-apply app-btn app-next"
                        :disabled="disabled"
                    >{{ $gettext("创建") }}</button>
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
import Toast from '/@/components/toast';
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
const webdav = ref<NasCreateWebdav>({
    username: "root",
    password: "",
    rootPath: props.rootPath,
})
const getData = async () => {
    const load = Toast.Loading($gettext("加载中..."))
    disabled.value = true
    try {
        const res = await request.Nas.Webdav.Status.GET()
        if (res?.data) {
            const { result, error } = res.data
            if (error) {
                Toast.Warning(error)
                return
            }
            if (result) {
                if (result.username) {
                    webdav.value.username = result.username
                }
                if (result.password) {
                    webdav.value.password = result.password
                }
            }
        }

    } catch (error) {
        Toast.Error(error as string)
    }
    disabled.value = false
    load.Close()
}
getData()
const onSumbit = () => {
    const _webdav = webdav.value
    if (_webdav.rootPath == "") {
        Toast.Warning($gettext("共享路径不能为空"))
        return
    }
    if (_webdav.username == "") {
        Toast.Warning($gettext("用户名不能为空"))
        return
    }
    if (_webdav.password == "") {
        Toast.Warning($gettext("密码不能为空"))
        return
    }
    onCreateWebdav(_webdav)
}
const onCreateWebdav = async (_webdav: NasCreateWebdav) => {
    disabled.value = true
    const load = Toast.Loading($gettext("创建中..."))
    try {
        const res = await request.Nas.Webdav.Create.POST(_webdav)
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