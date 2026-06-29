<template>
    <div id="page">
        <div class="title">{{ $gettext("欢迎使用网络配置向导") }}</div>
        <div class="desc">{{ $gettext("选择一种连接方式以开始") }}</div>
        <div class="network-containers">
            <div class="network-container_item">
                <router-link to="/network/pppoe">
                    <div class="cover">
                        <div class="thumbnail">
                            <DialSvg></DialSvg>
                            <span>{{ $gettext("宽带拨号连接") }}</span>
                        </div>
                    </div>
                </router-link>
            </div>
            <div class="network-container_item">
                <router-link to="/network/dhcp">
                    <div class="cover">
                        <div class="thumbnail">
                            <RouterSvg></RouterSvg>
                            <span>{{ $gettext("连接现有路由器") }}</span>
                        </div>
                    </div>
                </router-link>
            </div>
            <div class="network-container_item">
                <router-link to="/network/gateway">
                    <div class="cover">
                        <div class="thumbnail">
                            <SiderouterSvg></SiderouterSvg>
                            <span>{{ $gettext("配置为旁路由") }}</span>
                        </div>
                    </div>
                </router-link>
            </div>
        </div>
        <div class="info" v-html="advTips">
        </div>
    </div>
</template>
<script setup lang="ts">
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import RouterSvg from "/@/components/svg/router.vue"
import DialSvg from "/@/components/svg/dial.vue"
import SiderouterSvg from "/@/components/svg/siderouter.vue"

const advTips = $gettext("没找到想要的配置？请使用%{link}", {link:'<a href="/cgi-bin/luci/admin/network/network">'+$gettext("高级模式")+'</a>'}, true)
</script>
<style lang="scss" scoped>
#page {
    width: 100%;
    padding: 1rem;
    margin: 0 auto;
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-start;
    align-items: center;
    align-content: center;
    max-width: 800px;
    height: 100vh;
    overflow: overlay;

    .title {
        width: 100%;
        display: block;
        text-align: center;
        font-size: 32px;
        font-family: PingFangSC-Semibold, PingFang SC;
        font-weight: 600;
        color: rgba(0, 0, 0, 0.8);
        line-height: 45px;

    }

    .desc {
        width: 100%;
        display: block;
        font-size: 24px;
        font-family: PingFangSC-Semibold, PingFang SC;
        font-weight: 600;
        color: rgba(0, 0, 0, 0.6);
        line-height: 33px;
        text-align: center;
        margin-top: 10px;
    }

    div.info {
        width: 100%;
        display: block;
        font-size: 1.6em;
        font-size: 16px;
        margin-left: 34px;
    }

    .network-containers {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        align-content: center;
        justify-content: center;
        margin: 3rem 0 1rem 0;

        .network-container_item {
            flex: 0 0 100%;
            // max-width: 33.333%;
            position: relative;
            border-radius: 4px;
            padding: 10px;
            cursor: pointer;
            max-width: 240px;
            width: 240px;
            height: 308px;

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
</style>

<style lang="scss" scoped>
@media screen and(max-width: 992px) {}
</style>