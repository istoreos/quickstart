<template>
    <action-component :Close="Close" :type="1">
        <template v-if="setup == 0 || setup == 2">
            <div class="actioner-dns">
                <div class="actioner-dns_header">
                    <span>{{ $gettext("沙箱模式配置向导") }}</span>
                </div>
                <div class="actioner-dns_body">
                    <p class="sandbox_info">{{ $gettext("一个简易沙箱，方便用来实验系统配置和程序，方便开发未完成的软件，但不保护 Docker 和硬盘的数据") }}</p>
                    <template v-if="setup == 0">
                    <template v-if="!sandboxDisks">
                        <div class="disk_loading_icon">
                            <icon-loading :size="38" color="currentColor"/>
                            <span class="disk_loading_info">{{ $gettext("正在加载中...") }}</span>
                        </div>
                    </template>

                    <template v-if="sandboxDisks && sandboxDisks.disks.length == 0">
                        <div class="disk_tips">
                            <HintSvg></HintSvg>
                            <span>{{ $gettext("检测不到挂载的磁盘信息，请先插上磁盘，建议使用U盘或者移动硬盘，方便装卸") }}</span>
                        </div>
                    </template>
                    <template v-if="sandboxDisks && sandboxDisks.disks.length > 0">
                        <div class="label-item">
                            <div class="label-item_key">
                                <span>{{ $gettext("目标磁盘（建议选择U盘或者移动硬盘，方便装卸）") }}</span>
                            </div>
                            <div class="label-item_value">
                                <select name="" id="" @change="onDiskChange" v-model="diskpath">
                                    <option value="">{{ $gettext("请选择目标磁盘") }}</option>
                                    <option :value="item.path" v-for="(item, i) in sandboxDisks.disks" :key="i">{{
                                            item.venderModel
                                    }}（{{ item.size }}）
                                    </option>
                                </select>
                            </div>
                        </div>
                        <div class="label-item">
                            <div class="label-item_key">
                                <span>{{ $gettext("目标分区（分区大小须大于2G，将此分区作为外部 overlay 使用）") }}</span>
                            </div>
                            <div class="label-item_value">

                                <select name="" id="" v-model="partition">
                                    <option selected="true" value="">{{ $gettext("请选择目标分区") }}</option>
                                    <option :value="item.path" v-for="(item, i) in candidateParts" :key="i"
                                        :disabled="item.sizeInt < ((1 << 30) * 1) || item.isSystemRoot">
                                        {{ item.name }}（{{ item.filesystem || $gettext("未格式化") }}）{{ item.total }}</option>
                                </select>
                            </div>
                        </div>
                        <div class="sandbox_tips">
                            <HintSvg></HintSvg>
                            <span>{{ $gettext("此操作会将会删除该分区全部数据") }}</span>
                        </div>
                    </template>
                    </template>
                    <template v-if="setup == 2">
                        <p class="sandbox_info timeout">{{ $gettext("即将重启设备") }} <span>（{{ time }}s）</span> </p>
                        <p class="sandbox_roboot_tips">{{ $gettext("等待设备重启，重启完成后") }}<span class="sandbox_roboot_refresh">{{ $gettext("请刷新界面") }}</span> </p>
                    </template>
                </div>
                <template v-if="setup == 0">
                <div class="actioner-dns_footer">
                    <button class="cbi-button cbi-button-apply app-btn" :disabled="!partition" @click="setup=1">{{ $gettext("开启沙箱") }}</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onClose">{{ $gettext("取消") }}</button>
                </div>
                </template>
            </div>
        </template>

        <template v-if="setup == 1">
            <div class="actioner-tips">
                <div class="actioner-tips_header">
                    <span>{{ $gettext("温馨提示") }}</span>
                </div>
                <div class="actioner-tips_body">
                    <p class="sandbox_info">{{ $gettext("此操作会将会删除该分区全部数据，并格式化成EXT4，随后自动重启进入沙箱模式，是否继续？") }}</p>
                </div>
                <div class="actioner-tips_footer">
                    <button class="cbi-button cbi-button-apply app-btn" @click="onSumbit">{{ $gettext("继续") }}</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onCloseSandbox">{{ $gettext("取消") }}</button>
                </div>
            </div>
        </template>

        <template v-if="setup == 3">
            <div class="actioner-tips">
                <div class="actioner-tips_header">
                    <span>{{ $gettext("错误") }}</span>
                </div>
                <div class="actioner-tips_body">
                    <p class="sandbox_info">{{ error }}</p>
                </div>
                <div class="actioner-tips_footer">
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onCloseSandbox">{{ $gettext("取消") }}</button>
                </div>
            </div>
        </template>
    </action-component>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from "/@/components/toast";
import ActionComponent from "/@/components/action/modal.vue"
import request from "/@/request";
import HintSvg from "/@/components/svg/hint.vue"
import path, { resolve } from "path";
import { rejects } from "assert";

const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
})

const setup = ref(0)
const status = ref("disk")
const error = ref("")
const time = ref(3)
const diskpath = ref("")
const candidateParts = ref<childrensInfo[]>([])
const partition = ref("")

const sandboxDisks = ref<NasSandboxDisks | null>(null)



const getSandbox = () => {

    request.Nas.SandboxDisks.GET().then(res => {
        if (res?.data) {
            if (res.data?.result) {
                sandboxDisks.value = res.data.result
                return
            }
        }
        throw $gettext("加载磁盘信息失败")
    }).catch((e) => {
        error.value = e
        setup.value = 3
    })
}
getSandbox()

const RebootSystem = () => {
    return request.System.Reboot.POST()
        .then(res => {
            if (res?.data) {
                if ((res?.data?.success || 0) == 0) {
                    return
                }
            }
            throw $gettext("未知错误")
        })
}

const onDiskChange = (e: Event) => {
    partition.value = ""
    candidateParts.value = (diskpath.value && sandboxDisks.value?.disks.find(d => d.path == diskpath.value)?.childrens) || []
}

//倒计时
const updateTime = () => {
    if (time.value > 0) {
        time.value -= 1
        window.setTimeout(updateTime, 1000)
    }
}

const onClose = (e: Event) => {
    e.preventDefault();
    if (props.Close) {
        props.Close()
    }
}

//页面自动刷新
const waitReboot = () => {
    new Promise((resolve, reject) => {
        //加载的目标网址
        const target = "/luci-static/resources/icons/loading.gif"
        //定时调用图像
        const rejectFn = () => {
            window.setTimeout(check, 2000)
        }
        const check = () => {
            //创建一个图像
            const img = new Image()
            //当图像装载完毕时
            img.onload = resolve
            //当图像装载发生错误时
            img.onerror = rejectFn
            //图像装载目标路径
            img.src = target
        }
        //定时调用图像
        window.setTimeout(check, 10000)

    }).then(() => {
        //刷新页面
        window.setTimeout(() => {
            location.reload()
        }, 2000)
    })
}

const onSumbit = (e: Event) => {
    const load = Toast.Loading($gettext("配置沙箱重启中..."))
    request.Nas.Sandbox.POST({ path: partition.value })
        .then(res => {
            if (res?.data) {
                if ((res.data.success || 0) == 0) {
                    setup.value = 2
                    window.setTimeout(updateTime, 1000)
                    return RebootSystem()
                } else if (res.data?.error) {
                    throw res.data.error
                }
            }
            throw $gettext("未知错误")
        }).then(waitReboot)
        .catch(error => Toast.Warning(error))
        .finally(() => load.Close())

}

const onCloseSandbox = () => {
    setup.value = 0
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

        .sandbox_roboot_tips {
            margin-top: 24px;
            text-align: center;
        }

        .disk_loading_icon {
            position: absolute;
            left: 50%;
            transform: translate(-50%, 0px);
            display: flex;
            flex-direction: column;
            align-items: center;
            padding: 10px;

            .disk_loading_info {
                margin-top: 5px;
            }
        }

        .disk_tips {
            text-align: center;
            font-size: 16px;
            margin-top: 159px;
            color: #f9ad1e;

            svg {
                vertical-align: middle;
            }

            span {
                margin-left: 6px;
            }
        }

        .sandbox_info {
            text-align: center;
            line-height: 22px;
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

        .sandbox_tips {
            svg {
                vertical-align: middle;
            }

            span {
                font-size: 12px;
                margin-left: 4px;
            }
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

.sandbox_roboot_refresh {
    color: #5e72e4;
}

option:disabled {
    background-color: rgb(224, 224, 224);

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
        }

        .actioner-dns_footer {
            button {
                width: 100% !important;
                margin-bottom: 10px;
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