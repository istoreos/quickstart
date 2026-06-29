<template>
    <action-component :Close="Close" :type="1">
        <template v-if="setup == 0">

            <form class="actioner-dns">

                <div class="actioner-dns_header">
                    <span>USB装机</span>
                </div>
                <div class="actioner-dns_body">
                    <p>您想将系统安装在哪里？</p>
                    <div class="label-item">
                        <div class="label-item_key">
                            <span>目标磁盘</span>
                        </div>

                        <div class="label-item_value">
                            <select>
                                <option>磁盘A（120G）</option>
                                <option>磁盘B（120G）</option>
                            </select>
                            <svg width="14px" height="14px" viewBox="0 0 14 14" version="1.1"
                                xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
                                <g id="icon_alert" stroke="none" stroke-width="1" fill="none" fill-rule="evenodd">
                                    <g id="Icon/Warning">
                                        <rect id="矩形" fill="#000000" fill-rule="nonzero" opacity="0" x="0" y="0"
                                            width="14" height="14" />
                                        <path
                                            d="M7,0.875 C3.61757813,0.875 0.875,3.61757813 0.875,7 C0.875,10.3824219 3.61757813,13.125 7,13.125 C10.3824219,13.125 13.125,10.3824219 13.125,7 C13.125,3.61757813 10.3824219,0.875 7,0.875 Z M6.5625,4.046875 C6.5625,3.98671875 6.61171875,3.9375 6.671875,3.9375 L7.328125,3.9375 C7.38828125,3.9375 7.4375,3.98671875 7.4375,4.046875 L7.4375,7.765625 C7.4375,7.82578125 7.38828125,7.875 7.328125,7.875 L6.671875,7.875 C6.61171875,7.875 6.5625,7.82578125 6.5625,7.765625 L6.5625,4.046875 Z M7,10.0625 C6.63769531,10.0625 6.34375,9.76855469 6.34375,9.40625 C6.34375,9.04394531 6.63769531,8.75 7,8.75 C7.36230469,8.75 7.65625,9.04394531 7.65625,9.40625 C7.65625,9.76855469 7.36230469,10.0625 7,10.0625 Z"
                                            id="形状" fill="#FAAD14" />
                                    </g>
                                </g>
                            </svg>

                            <span class="info">选择的磁盘会被全盘格式化</span>
                        </div>
                    </div>


                    <div class="tips_bj" v-if="isTips">
                        <div class="usb_tips">
                            <span class="tips_title">温馨提示</span>
                            <p class="tips_info">系统将安装到目标磁盘，并且磁盘会被全盘格式化，是否继续？</p>
                            <div class="actioner-dns_footer">
                                <button class="cbi-button cbi-button-apply app-btn app-next" @click="onNext"
                                    type="button">确定</button>
                                <button class="cbi-button cbi-button-remove app-btn app-back"
                                    @click="onCloseTips">取消</button>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="actioner-dns_footer">
                    <button class="cbi-button cbi-button-apply app-btn app-next" @click="onInstall"
                        type="button">立即安装</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onClose">取消</button>
                </div>



            </form>
        </template>
        <!-- 安装完成 -->

        <div class="action" v-else-if="setup == 2">
            <div class="actioner-dns">
                <h2 class="title">Docker迁移向导</h2>
                <div class="finished">
                    <FinishedSvg></FinishedSvg>
                </div>
                <p class="successed">迁移成功！{{ time }}s后自动重启</p>

            </div>
        </div>
    </action-component>
</template>
<script setup lang="ts">
import { ref } from "vue";
import FinishedSvg from "/@/components/svg/finished.vue"
import Toast from "/@/components/toast";
import ActionComponent from "/@/components/action/modal.vue"
import HintSvg from "/@/components/svg/hint.vue"
import request from "/@/request";
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
})
const setup = ref(0)
const time = ref(3)
const isTips = ref(false)
setTimeout(() => {
    time.value--
    setTimeout(() => {
        time.value--
    }, 2000)
}, 3000)

const onClose = (e: Event) => {
    e.preventDefault();
    if (props.Close) {
        props.Close()
    }
}
const onInstall = () => {
    isTips.value = true
}
const onCloseTips = () => {
    //   e.preventDefault();
    if (props.Close) {
        props.Close()
    }
    setup.value = 0
}
const onNext = () => {
    // const load = Toast.Loading("正在安装中...")
    setup.value = 2
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

        .tips_bj {
            .usb_tips {
                width: 400px;
                height: 260px;
                box-shadow: 0px 0px 10px #9d9d9d;
                text-align: center;
                padding: 20px;
                position: absolute;
                left: 0;
                right: 0;
                bottom: 0;
                top: 0;
                margin: auto;
                z-index: 9999;

                .tips_title {
                    font-size: 16px;
                    font-weight: 600;
                    padding-top: 10px;
                }

                .tips_info {
                    margin-top: 40px;
                    margin-bottom: 95px;
                    font-size: 12px;
                }
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

    .info {
        margin-left: 5px;
    }

    svg {
        vertical-align: middle;
    }

    .finished {
        display: flex;
        justify-content: center;
        margin: 80px;
        margin-bottom: 28px;
    }

    .successed {
        text-align: center;
        margin-bottom: 82px;
    }


}
</style>