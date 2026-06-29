<template>
    <action-component :Close="Close" :type="1">
        <template v-if="setup == 0">
            <div class="actioner-dns">
                <div class="actioner-dns_header">
                    <span>{{ $gettext("沙箱模式配置向导") }}</span>
                </div>
                <div class="actioner-dns_body">
                    <p class="sandbox_info">{{ $gettext("一个简易沙箱，方便用来实验系统配置和程序，方便开发未完成的软件，但不保护 Docker 和硬盘的数据") }}</p>
                    <div class="sandbox_environment">
                        <p>{{ $gettext("当前处于沙箱环境：") }}</p>
                        <p>{{ $gettext("1、点击“提交”可将变更合并到非沙箱环境") }}</p>
                        <p>{{ $gettext("2、点击“重置”可将沙箱恢复到初始状态") }}</p>
                        <p>{{ $gettext("3、点击“退出”可退出沙箱环境，并放弃沙箱中的数据") }}</p>
                    </div>
                    <div class="sandbox_environment_info">{{ $gettext("以上操作都将重启设备，设备重启完成后会自动刷新页面。如果 IP 变化可能需要") }}<span
                            class="sandbox_environment_reboot">{{ $gettext("手动在地址栏输入地址") }}</span>
                        <p class="sandbox_environment_tex" v-html="$gettext('如需<b>临时</b>退出沙箱环境，请将设备关机后拔出相关磁盘，启动前插入相关磁盘可再次进入沙箱。<br/> 注意临时退出沙箱环境以后升级固件会导致之前的沙箱数据无效', {}, true)"></p>
                    </div>
                </div>
                <div class="actioner-dns_footer">
                    <button class="cbi-button cbi-button-apply app-btn" @click="onSubmit"
                        :disabled="running">{{ $gettext("提交") }}</button>
                    <button class="cbi-button cbi-button-apply app-btn" @click="onReset" :disabled="running">{{ $gettext("重置") }}</button>
                    <button class="cbi-button cbi-button-apply app-btn" @click="onExit"
                        :disabled="running">{{ $gettext("退出") }}</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onClose">{{ $gettext("取消") }}</button>
                </div>
            </div>
        </template>
    </action-component>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import ActionComponent from "/@/components/action/modal.vue"
import request from "../../request";
import Toast from "../toast";
const props = defineProps({
    Close: {
        type: Function,
        required: true
    }
})

const setup = ref(0)
const running = ref(false)

const waitReboot = () => {
    new Promise((resolve, reject) => {
        const target = "/luci-static/resources/icons/loading.gif"
        const rejectFn = () => {
            window.setTimeout(check, 2000);
        }
        const check = () => {
            const img = new Image()

            img.onload = resolve
            img.onerror = rejectFn

            img.src = target
        }
        window.setTimeout(check, 10000)
    }).then(() => {
        window.setTimeout(() => {
            location.reload()
        }, 2000);
    })
}

const onSubmit = () => {
    running.value = true
    const load = Toast.Loading($gettext("提交中..."))
    request.Nas.SandboxCommit.POST().then(res => {
        if (res?.data) {
            if ((res?.data?.success || 0) == 0) {
                Toast.Loading($gettext("设备重启中..."))
                return
            } else if (res?.data?.error) {
                alert(res.data.error)
            }
        }
        throw $gettext("未知错误")
    }).then(waitReboot)
        .catch(error => { Toast.Error(error); running.value = false })
        .finally(() => load.Close())
}
const onReset = () => {
    running.value = true
    const load = Toast.Loading($gettext("重置中..."))
    request.Nas.SandboxReset.POST().then(res => {
        if (res?.data) {
            if ((res?.data?.success || 0) == 0) {
                Toast.Loading($gettext("设备重启中... 若页面长时间未刷新可能需要手动填写地址"))
                return
            } else if (res?.data?.error) {
                alert(res.data.error)
            }
        }
        throw $gettext("未知错误")
    }).then(waitReboot)
        .catch(error => { Toast.Error(error); running.value = false })
        .finally(() => load.Close())

}
const onExit = () => {
    if (!confirm($gettext("确定放弃沙箱中的数据？再次进入沙箱需要重新格式化相应磁盘分区")))
        return;
    running.value = true
    const load = Toast.Loading($gettext("执行中..."))
    request.Nas.SandboxExit.POST().then(res => {
        if (res?.data) {
            if ((res?.data?.success || 0) == 0) {
                Toast.Loading($gettext("设备重启中... 若页面长时间未刷新可能需要手动填写地址"))
                return
            } else if (res?.data?.error) {
                alert(res.data.error)
            }
        }
        throw $gettext("未知错误")
    }).then(waitReboot)
        .catch(error => { Toast.Error(error); running.value = false })
        .finally(() => load.Close())

}

const onClose = (e: Event) => {
    e.preventDefault()
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

        span {
            margin: 0 auto;
        }
    }

    .actioner-dns_body {
        padding: 1rem;
        min-height: 50vh;

        .sandbox_info {
            text-align: center;
            line-height: 22px;

        }

        .sandbox_environment {
            font-size: 16px;
            line-height: 28px;
            margin: 20px 0;
        }

        .sandbox_environment_info {
            font-size: 16px;
            line-height: 28px;

            .sandbox_environment_reboot {
                color: #5e72e4;
            }

            .sandbox_environment_tex {
                color: red;
                font-size: 0.9em;
            }
        }
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



.actioner-tips {
    width: 400px;
    background-color: #fff;
    position: relative;
    z-index: 99999;
    margin: auto;
    overflow: auto;

    .actioner-tips_header {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        padding: 1rem;
        font-size: 2em;
        border-bottom: 1px solid #eee;
    }

    .sandbox_info {
        padding: 62px 54px;
        line-height: 20px;
    }

    .actioner-tips_footer {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: flex-end;
        padding: 1rem;
        font-size: 2em;
        border-top: 1px solid #eee;
    }

}

.timeout {
    margin-top: 114px;

    span {
        color: #5e72e4;
    }
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 1400px) {
    .actioner-tips_footer {
        button {
            width: 100% !important;
        }
    }
}

@media screen and (max-width: 900px) {
    .actioner-dns {
        width: 100%;
    }
}

@media screen and (max-width: 700px) {
    .actioner-dns {
        .actioner-dns_body {
            min-height: 42vh;
        }

        .actioner-dns_footer {
            button {
                width: 100% !important;
                margin-bottom: 10px;
            }
        }
    }

    .actioner-tips {
        width: 80%;
        line-height: 22px;

        .sandbox_info {
            padding: 34px 10px;
            font-size: 10px;
        }

        .actioner-tips_header {
            font-size: 20px;
        }

        .actioner-tips_footer {
            button {
                width: 100% !important;
            }
        }

    }
}

@media screen and (max-width: 600px) {
    .actioner-dns {
        .actioner-dns_footer {
            button {
                width: 100% !important;
                margin-bottom: 10px;
                margin-left: 0;
            }
        }
    }
}

@media screen and (max-width: 500px) {
    .actioner-dns {
        .actioner-dns_body {
            .label-item {
                .label-item_key {
                    width: 228px;
                    overflow: hidden;
                    text-overflow: ellipsis;


                }
            }
        }

    }
}

@media screen and (max-width: 400px) {
    .actioner-dns {

        .actioner-dns_body {
            .label-item {
                .label-item_key {
                    width: 163px;
                    overflow: hidden;
                    text-overflow: ellipsis;

                }
            }

            .sandbox_info {
                font-size: 10px;
            }

            .sandbox_environment {
                font-size: 12px;
            }

            .sandbox_environment_info {
                font-size: 12px;
            }
        }
    }

    .actioner-tips {
        .sandbox_info {
            padding: 3px 10px;

        }

    }
}
</style>