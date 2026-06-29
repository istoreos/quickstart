<template>
    <div class="ddnsto-bind">
        <div class="ddnsto-container_bg"></div>
        <div class="ddnsto-container">
            <!-- wifi配置页 -->
            <Wifi v-if="setup === 1"></Wifi>

            <!-- 网络配置向导 -->
            <div class="popular_container" v-else-if="setup === 2">
                <p>{{ $gettext("欢迎使用网络配置向导") }}</p>
                <div class="tip">{{ $gettext("选择一种连接方式以开始") }}</div>
                <div class="popular_item">
                    <div class="network-containers">
                        <div class="network-container_item">
                            <router-link to="/network/pppoe?type=index">
                                <div class="cover" @click="goTo()">
                                    <div class="thumbnail">
                                        <DialSvg></DialSvg>
                                        <span>{{ $gettext("宽带拨号连接") }}</span>
                                    </div>
                                </div>
                            </router-link>
                        </div>
                        <div class="network-container_item">
                            <router-link to="/network/dhcp?type=index">
                                <div class="cover" @click="goTo()">
                                    <div class="thumbnail">
                                        <RouterSvg></RouterSvg>
                                        <span>{{ $gettext("连接现有路由器") }}</span>
                                    </div>
                                </div>
                            </router-link>
                        </div>
                        <div class="network-container_item">
                            <router-link to="/network/gateway?type=index">
                                <div class="cover" @click="goTo()">
                                    <div class="thumbnail">
                                        <SiderouterSvg></SiderouterSvg>
                                        <span>{{ $gettext("配置为旁路由") }}</span>
                                    </div>
                                </div>
                            </router-link>
                        </div>
                    </div>
                    <div class="popular_txt">{{ $gettext("没找到想要的配置？请使用") }}<a
                            href="/cgi-bin/luci/admin/network/network">{{ $gettext("高级模式") }}</a></div>
                </div>
                <div class="btn1" @click="next(3)">{{ $gettext("跳过") }}</div>
            </div>

            <!-- 欢迎-协议页  -->
            <div class="hello" v-else-if="setup === 3">
                <img src="https://assets.koolcenter.com/istoreos/firmware-guide/icon_huanying@2x.png" alt="">
                <p>{{ $gettext("欢迎使用iStoreOS") }}</p>
                <div class="radio_container">
                    <div class="radio" :class="isShow ? 'active' : ''" @click="isShow = !isShow">
                        <div class="yuan" :class="isShow ? 'yuan1' : ''"></div>
                    </div>
                    <div class="radio_text">{{ $gettext("我已完整阅读并同意") }}<a
                            href=" https://www.linkease.com/rd/istoreos-user-agreement/" target="_blank"
                            rel="noopener noreferrer">{{ $gettext("《iStoreOS固件用户协议》") }}</a></div>
                </div>
                <div class="confirm" @click="next(4)">{{ $gettext("确定") }}</div>
            </div>

            <!-- 设置系统密码  -->
            <div class="set_up" v-else-if="setup === 4">
                <div class="title">{{ $gettext("设置系统密码") }}</div>
                <p>{{ $gettext("此设备还未设置密码，请先设置密码。如遗忘密码，可以通过重置设备恢复初始密码。") }}</p>
                <div class="password">
                    <div class="input_tip">{{ $gettext("请填写密码") }}：</div>
                    <input class="password_input" type="password" v-model.trim="password"
                        :placeholder="$gettext('请填写密码')" />
                </div>
                <div class="password">
                    <div class="input_tip">{{ $gettext("请再次填写密码") }}：</div>
                    <div class="input_box">
                        <input class="password_input" type="password" v-model.trim="password1"
                            :placeholder="$gettext('请再次填写密码')" />
                        <span class="password_tip">{{ $gettext("长度为6～20个英文、数字结合") }}</span>
                    </div>
                </div>
                <div class="occupy"></div>
                <div class="btn1" @click="save()">{{ $gettext("保存") }}</div>
                <div class="skip" @click="next(5)">{{ $gettext("跳过") }}</div>
            </div>

            <!-- 开启远程域名访问 -->
            <div class="domain_container" v-else-if="setup === 5">
                <div class="title">{{ $gettext("开启远程域名访问") }}</div>
                <p>{{ $gettext("通过安全加密通道，随时随地远程管理你的iStoreOS") }}</p>
                <div class="domain_img">
                    <img src="https://assets.koolcenter.com/istoreos/firmware-guide/iStore-domain.png" alt="">
                </div>
                <div class="btn1" @click="openAlert()">{{ $gettext("立即启用") }}</div>
                <div class="skip" @click="next(6)">{{ $gettext("跳过") }}</div>
                <div class="tip">{{ $gettext("域名服务由") }} <a href="https://ddnsto.com" target="_blank"
                        rel="noopener noreferrer">{{ $gettext("ddnsto.com") }}</a> {{ $gettext("提供") }}</div>
            </div>

            <!-- 公众号二维码 -->
            <div class="wx_qr" v-else>
                <p>{{ $gettext("关注iStoreOS公众号") }}</p>
                <div class="txt">{{ $gettext("掌握最新动态") }}</div>
                <div>
                    <img src="https://assets.koolcenter.com/istoreos/firmware-guide/istoreos-qr.jpg" alt="">
                </div>
                <div class="btn1" @click="close()">{{ $gettext("开始探索iStoreOS") }}</div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, onUnmounted, provide } from 'vue'
import RouterSvg from "../svg/router.vue"
import DialSvg from "../svg/dial.vue"
import Wifi from "./wifi.vue"
import SiderouterSvg from "../svg/siderouter.vue"
import ActionDomain from "../action-domain"
import Toast from "../toast";
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request';
const { $gettext } = useGettext()

const props = defineProps({
    close: {
        type: Function,
        required: true
    },
    init: {
        type: Number,
        default: 0
    }
});

const setup = ref(props.init)
const password = ref("")
const password1 = ref("")

provide('setup', setup)
const isShow = ref(false)

let setupNum = sessionStorage.getItem('setupNum');
if (setupNum) {
    setupNum = JSON.parse(setupNum);
    setup.value = setupNum
    sessionStorage.removeItem('setupNum')
}

let mounted = false
let timerId = undefined
const DdnsStatus = ref({})
const getDdns = function () {
    if (!mounted)
        return
    (document.hidden ? Promise.resolve() : request.Guide.GetDdns.GET().then(res => {
        if (res?.data) {
            if ((res?.data?.success || 0) == 0) {
                if (res.data.result) {
                    DdnsStatus.value = res.data.result
                }
            }

        }
    })).then(() => {
        if (!mounted)
            return
        timerId = window.setTimeout(getDdns, 3000)
    })
}

const goTo = () => {
    sessionStorage.setItem('setupNum', JSON.stringify(setup.value))
}

const close = async () => {
    try {
        // localStorage.setItem('firstOpen', JSON.stringify(true));
        const res = await request.GuidePage.completeGuide.POST()
        props.close()
        location.reload();
    } catch (error) {
        console.error('请求失败:', error);
    }
}

const openAlert = () => {
    ActionDomain({
        url: DdnsStatus.value.ddnstoDomain
    })
}

const validateString = (value) => {
    const englishOnly = /^[a-zA-Z]{6,20}$/;
    const alphanumeric = /^(?=.*[a-zA-Z])(?=.*\d)[a-zA-Z\d]{6,20}$/;
    return englishOnly.test(value) || alphanumeric.test(value)
};
const save = async () => {
    if (!password.value || !password1.value) {
        return Toast.Warning($gettext('请输入密码'))
    }
    if (!validateString(password.value)) {
        return Toast.Warning($gettext('密码格式不正确'))
    }
    if (password.value !== password1.value) {
        return Toast.Warning($gettext('两次密码不一致'))
    }
    const res = await request.GuidePage.setPassword.POST({ password: password.value })
    console.log(res, 'setPassword');
    if (res?.data?.success == 0) {
        Toast.Success($gettext("系统密码设置成功"))
        setTimeout(() => {
            next(5)
        }, 400)
    }
}

const next = (num) => {
    if (num === 4 && !isShow.value) {
        return Toast.Warning('请勾选用户协议')
    }
    setup.value = num
}

// 屏蔽滚动
onMounted(() => {
    document.body.setAttribute('lock-scroll', true)
    mounted = true
    timerId = window.setTimeout(getDdns, 1100)
})
onUnmounted(() => {
    if (timerId !== undefined)
        window.clearTimeout(timerId)
    mounted = false
})
onBeforeUnmount(() => {
    document.body.removeAttribute('lock-scroll')
})
</script>
<style lang="scss" scoped>
.ddnsto-bind {
    position: fixed;
    top: 0;
    bottom: 0;
    left: 0;
    right: 0;
    text-align: center;
    z-index: 100;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    overflow: hidden;

    .ddnsto-container_bg {
        left: 0;
        right: 0;
        top: 0;
        bottom: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.4);
        position: absolute;
        cursor: pointer;
        overflow: hidden;
    }

    .btn1 {
        padding: 10px 16px;
        background: #0060FF;
        border-radius: 4px;
        font-size: 16px;
        color: #FFFFFF;
        line-height: 22px;
        display: inline;
        cursor: pointer;
    }

    .ddnsto-container {
        position: relative;
        display: block;
        width: 582px;
        background: #fff;
        box-shadow: 0px 12px 48px 16px rgba(0, 0, 0, 0.03),
            0px 9px 28px 0px rgba(0, 0, 0, 0.05),
            0px 6px 16px -8px rgba(0, 0, 0, 0.08);
        border-radius: 2px;
        transition: 0.3s;

        .hello {
            padding: 40px 0;

            >img {
                width: 100px;
                height: 100px;
            }

            >p {
                font-family: PingFangSC, PingFang SC;
                font-weight: 500;
                font-size: 24px;
                line-height: 33px;
                margin: 24px 0 32px;
            }

            .radio_container {
                display: flex;
                align-items: center;
                justify-content: center;
                margin-bottom: 32px;

                .radio {
                    width: 16px;
                    height: 16px;
                    border: 1px solid #ccc;
                    border-radius: 50%;
                    margin-right: 10px;
                    cursor: pointer;
                }

                .active {
                    background: none;
                    border: 1px solid #0060FF !important;
                }

                .yuan {
                    margin-top: 50%;
                    margin-left: 50%;
                    transform: translate(-50%, -50%);
                    width: 12px;
                    height: 12px;
                    background: #fff;
                    border-radius: 50%;
                }

                .yuan1 {
                    background: #0060FF !important;
                }

                .radio_text {
                    font-size: 16px;
                    color: rgba(0, 0, 0, 0.83);
                    line-height: 24px;

                    >a {
                        color: #0060FF;
                    }
                }
            }

            .confirm {
                background: #0060FF;
                border-radius: 6px;
                padding: 10px 20px;
                color: #fff;
                display: inline;
                font-size: 16px;
                line-height: 22px;
                cursor: pointer;
            }
        }

        .popular_container {
            padding: 26px 14px 30px;

            >p {
                font-weight: 500;
                font-size: 20px;
                line-height: 33px;
                font-weight: 600;
            }

            .tip {
                color: #616262;
                margin: 6px 0 20px;
            }

            .popular_item {
                .popular_txt {
                    padding-left: 47px;
                    font-size: 12px;
                    text-align: left;
                    margin-bottom: 30px;

                    >a {
                        color: #0060FF;
                        margin-left: 6px;
                    }
                }

                .network-containers {
                    width: 100%;
                    display: flex;
                    flex-wrap: wrap;
                    align-items: center;
                    align-content: center;
                    justify-content: center;
                    margin: 1rem 0 0;

                    .network-container_item {
                        flex: 0 0 100%;
                        // max-width: 33.333%;
                        position: relative;
                        border-radius: 4px;
                        padding: 10px;
                        cursor: pointer;
                        max-width: 160px;
                        width: 160px;
                        height: 205px;

                        .cover {
                            .thumbnail {
                                svg {
                                    width: 80px;
                                    height: 80px;
                                }

                                >span {
                                    font-size: 12px !important;
                                    line-height: 12px !important;
                                    margin-bottom: 6px;
                                }
                            }
                        }

                        a {
                            position: relative;
                            display: block;
                            width: 100%;

                            // &:hover {
                            //     transform: scale(1.05);
                            //     transition: 0.4s;
                            //     position: relative;

                            //     .cover {
                            //         .thumbnail {
                            //             box-shadow: 0px 6px 40px 0px #1c67f2;
                            //         }
                            //     }
                            // }

                            .cover {
                                position: relative;
                                padding-top: 130%;
                                z-index: 1;

                                .thumbnail {
                                    position: absolute;
                                    top: 0;
                                    left: 0;
                                    width: 100%;
                                    height: 100%;
                                    object-fit: contain;
                                    border-radius: 8px;
                                    overflow: hidden;
                                    z-index: 1;
                                    display: flex;
                                    flex-wrap: wrap;
                                    align-items: center;
                                    align-content: center;
                                    justify-content: center;
                                    background-color: #2dc8fd;

                                    i {
                                        display: block;
                                        font-size: 100px;
                                        color: #eee;
                                    }

                                    span {
                                        display: block;
                                        text-align: center;
                                        width: 100%;
                                        color: #eeee;
                                        font-size: 2em;
                                        line-height: 1.5;
                                        font-size: 22px;
                                        font-family: PingFangSC-Semibold, PingFang SC;
                                        color: #FFFFFF;
                                        line-height: 40px;
                                    }
                                }
                            }
                        }
                    }

                    .network-container_item {
                        &:nth-child(9n + 1) {
                            a {
                                .cover {
                                    .thumbnail {
                                        background: linear-gradient(138deg, #FF6E6B 0%, #FF6966 100%);
                                    }
                                }
                            }
                        }

                        &:nth-child(9n + 2) {
                            a {
                                .cover {
                                    .thumbnail {
                                        background: linear-gradient(145deg, #37D5A9 0%, #42D8B0 100%);
                                    }
                                }
                            }
                        }

                        &:nth-child(9n + 3) {
                            a {
                                .cover {
                                    .thumbnail {
                                        background: linear-gradient(145deg, #549AFF 0%, #2C82FF 100%);
                                    }
                                }
                            }
                        }

                        &:nth-child(9n + 4) {
                            a {
                                .cover {
                                    .thumbnail {
                                        background-color: #9b58de;
                                    }
                                }
                            }
                        }

                        &:nth-child(9n + 5) {
                            a {
                                .cover {
                                    .thumbnail {
                                        background-color: #297ff3;
                                    }
                                }
                            }
                        }

                        &:nth-child(9n + 6) {
                            a {
                                .cover {
                                    .thumbnail {
                                        background-color: #27aa8f;
                                    }
                                }
                            }
                        }

                        &:nth-child(9n + 7) {
                            a {
                                .cover {
                                    .thumbnail {
                                        background-color: #f15a4a;
                                    }
                                }
                            }
                        }

                        &:nth-child(9n + 8) {
                            a {
                                .cover {
                                    .thumbnail {
                                        background-color: #439c07;
                                    }
                                }
                            }
                        }

                        &:nth-child(9n + 9) {}

                        &:nth-child(9n + 10) {}
                    }
                }
            }
        }

        .wx_qr {
            padding: 65px 0 32px;

            >p {
                font-size: 24px;
                line-height: 33px;
            }

            .txt {
                color: rgba(255, 255, 255, 0.83);
                font-size: 16px;
                line-height: 24px;
            }

            img {
                width: 250px;
                height: 250px;
                margin: 16px 0 40px;
            }
        }

        .set_up {
            padding: 16px 24px;

            .title {
                font-size: 16px;
                font-weight: 600;
            }

            >p {
                font-size: 16px;
                color: rgba(0, 0, 0, 0.8);
                line-height: 22px;
                margin: 16px 0;
                text-align: left;
            }

            .password {
                display: flex;
                align-items: center;
                justify-content: center;
                margin-bottom: 16px;

                .input_tip {
                    width: 25%;
                    text-align: right;
                }

                .input_box {
                    width: 50%;
                    position: relative;

                    .password_input {
                        width: 100%;
                    }

                    .password_tip {
                        font-size: 14px;
                        color: rgba(0, 0, 0, 0.6);
                        line-height: 20px;
                        position: absolute;
                        bottom: -20px;
                        left: 0;
                    }
                }

                .password_input {
                    padding: 2px 6px;
                    width: 50%;
                }
            }

            .occupy {
                height: 30px;
            }

            .skip {
                font-size: 16px;
                color: #0060FF;
                line-height: 22px;
                margin: 20px 0 0;
                cursor: pointer;
            }
        }

        .domain_container {
            padding: 40px 0 16px;

            .title {
                font-size: 24px;
                line-height: 33px;
            }

            >p {
                font-size: 16px;
                color: rgba(0, 0, 0, 0.83);
                line-height: 24px;
                margin-top: 16px;
            }

            .domain_img {
                display: flex;
                justify-content: center;
                margin: 30px 0;

                >img {
                    width: 308px;
                    height: 204px;
                }
            }

            .skip {
                font-size: 16px;
                color: #0060FF;
                line-height: 22px;
                margin: 30px 0 20px;
                cursor: pointer;
            }

            .tip {
                font-size: 14px;
                color: rgba(0, 0, 0, 0.83);
                line-height: 24px;

                >a {
                    color: #0060FF;
                }
            }

            .btn1 {
                margin-top: 30px;
            }
        }
    }
}
</style>