<template>
    <div class="item_container">
        <div class="item">
            <div class="item_title">
                <wifiIcon color="#155dfc" class="icon" />
                <span>{{ $gettext('内网测速') }}</span>
            </div>
            <p>{{ $gettext('检测本地网络环境，获取内网访问地址') }}</p>
            <div class="wifi_btn" @click="intranetShow = true">
                <div>
                    <wifiIcon color="#ffffff" class="icon1" />
                    <span>{{ $gettext('开始内网测速') }}</span>
                </div>
            </div>
            <p class="tip">{{ $gettext('点击按钮获取内网测速地址，通过访问这些地址来测试内网连接速度') }}</p>
        </div>
        <div class="item">
            <div class="item_title">
                <DNSIcon color="#2bb55e" class="icon" />
                <span>{{ $gettext('内网测速') }}</span>
            </div>
            <p>{{ $gettext('选择测速节点，进行网络速度和连接质量测试') }}</p>
            <div class="select_box">
                <div>{{ $gettext('选择测速节点') }}</div>
                <select v-model="selectedNode" id="node" required :class="{ 'is-placeholder': !selectedNode }">
                    <option :value="null" disabled selected>{{ $gettext('请选择测速节点') }}</option>
                    <option v-for="option in nodeList" :value="option">
                        {{ option.title }} - {{ option.describe }}
                    </option>
                </select>
            </div>
            <div class="wifi_btn m-20" :class="{ 'is-bg': !selectedNode || startSpeed }" @click="startTest">
                <div class="start_btn">
                    <span class="icon3-wrap" :class="{ 'is-rotating': startSpeed }" v-if="startSpeed">
                        <timeIcon color="#ffffff" class="icon3" />
                    </span>
                    <DNSIcon color="#ffffff" class="icon2" v-else />
                    <span>{{ $gettext('开始测速') }}</span>
                </div>
            </div>
            <div class="progress" v-if="startSpeed">
                <p>
                    {{ $gettext('测速进度') }}
                    <span>25%</span>
                </p>
                <ProgressBar :percentage="25" :showPercentage="false" height="10px" borderRadius="10px" color="#030213"
                    backgroundColor="#cdccd0" />
            </div>
        </div>
        <div class="item">
            <div class="item_title">
                <logIcon color="#ff6900" class="icon2" />
                <span>{{ $gettext('测速日志') }}</span>
            </div>
            <p>{{ $gettext('实时测速过程记录') }}</p>
            <div class="log_info">
                <p v-for="item in 20">[17:00:20] 正在连接到测试服务器...</p>
            </div>
        </div>
        <div class="item">
            <div class="item_title">
                <LightningIcon color="#9865ff" class="icon2" />
                <span>{{ $gettext('测速结果') }}</span>
            </div>
            <p>{{ $gettext('测速节点') }}：CDN节点 (北京)</p>
            <div class="result_box">
                <div class="result">
                    <div class="result_item">
                        <downloadIcon color="#155dfc" class="icon_speed" />
                        <div class="speed_value">105.5</div>
                        <span class="unit">Mbps</span>
                        <span class="status status_bg1">优秀</span>
                        <div class="speed_title">{{ $gettext('下载速度') }}</div>
                    </div>
                    <div class="result_item">
                        <uploadIcon color="#00a63e" class="icon_speed1" />
                        <div class="speed_value">105.5</div>
                        <span class="unit">Mbps</span>
                        <span class="status status_bg2">良好</span>
                        <div class="speed_title">{{ $gettext('上传速度') }}</div>
                    </div>
                </div>
                <div class="line"></div>
                <div class="result">
                    <div class="result_state">
                        <div>18 ms</div>
                        <span class="status status_bg2">良好</span>
                        <span class="result_txt">延迟</span>
                    </div>
                    <div class="result_state">
                        <div>18 ms</div>
                        <span class="result_txt">延迟</span>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- 详情 -->
    <DialogVue v-model="intranetShow" title="内网访问地址" width="550px" :footerShow="false" :show-close="true">
        <!-- 默认插槽内容 -->
        <div class="custom-content">
            <p>以下是检测到的内网地址，请点击访问进行测速</p>
            <div class="address_box">
                <span>http://192.168.1.1</span>
                <div>访问</div>
            </div>
            <div class="address_box">
                <span>http://192.168.1.1</span>
                <div>访问</div>
            </div>
            <div class="address_box">
                <span>http://192.168.1.1</span>
                <div>访问</div>
            </div>
            <div class="address_box">
                <span>http://192.168.1.1</span>
                <div>访问</div>
            </div>
            <div class="address_box">
                <span>http://192.168.1.1</span>
                <div>访问</div>
            </div>
        </div>
    </DialogVue>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import wifiIcon from '/@/components/svg/wifi.vue'
import DNSIcon from "/@/components/svg/DNS.vue"
import timeIcon from "/@/components/svg/time.vue"
import logIcon from "/@/components/svg/log.vue"
import LightningIcon from '/@/components/svg/lightning.vue'
import downloadIcon from "/@/components/svg/download1.vue"
import uploadIcon from "/@/components/svg/upload.vue"

import ProgressBar from "/@/components/ProgressBar/index.vue"
import DialogVue from "/@/components/dialog/index.vue";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

const intranetShow = ref(false)
const nodeList = ref<any>([
    { title: 'SpeedTest.Net', describe: '全球标准网速测试服务', value: 'SpeedTest' },
    { title: 'CDN节点测试', describe: '测试到主要CDN节点的连接速度', value: 'CDN' },
    { title: 'UST高校节点', describe: '中科大教育网节点测试', value: 'UST' },
    { title: 'NAT类型测速', describe: '检测网络NAT类型和连通性', value: 'NAT' },
])
const selectedNode = ref<any>(null);

const startSpeed = ref(false)

const startTest = () => {
    if (!selectedNode.value) {
        return;
    }
    startSpeed.value = true;
}

</script>
<style lang="scss" scoped>
.icon {
    width: 24px;
    height: 24px;
    margin-right: 6px;
}

.icon1 {
    width: 16px;
    height: 16px;
    margin-right: 8px;
}

.icon2 {
    width: 20px;
    height: 20px;
    margin-right: 8px;
}

.icon3 {
    width: 17px;
    height: 17px;
    margin-right: 8px;
}

.m-20 {
    margin: 20px 0 !important;
}

:deep(.modal-container .modal-header) {
    border-bottom: none;
    padding-bottom: 0;
    padding-left: 20px;

    .modal-title {
        text-align: left;
    }
}

:deep(.modal-container .modal-content) {
    padding-top: 0;
    padding-left: 20px;
    padding: 0 20px;
    padding: 0 20px 20px;
}

.item_container {
    max-width: 1400px;
    margin: 0 auto;
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    grid-gap: 24px;
    font-size: 16px;

    .item {
        padding: 16px;
        background: var(--card-bg-color);
        border-radius: 10px;

        .item_title {
            display: flex;
            align-items: center;

            >span {
                color: var(--app-container_title-color);
                display: inline-block;
                padding-top: 2px;
            }
        }

        >p {
            color: #717182;
            margin-top: 10px;
        }

        .wifi_btn {
            margin: 40px 0;
            display: flex;
            justify-content: center;

            >div {
                background: #000;
                padding: 12px 24px;
                border-radius: 6px;
                display: flex;
                align-items: center;
                color: #fff;
                cursor: pointer;
                font-size: 14px;

                >span {
                    display: inline-block;
                    padding-top: 2px;
                }
            }

            .start_btn {
                padding: 10px;
                width: 100%;
                display: flex;
                justify-content: center;
            }
        }

        .progress {
            >p {
                display: flex;
                justify-content: space-between;
                font-size: 16px;
                color: #0a0a0a;
                margin-bottom: 0;
                margin-bottom: 12px;
            }
        }

        .is-bg {
            opacity: 0.6;
            pointer-events: none;
            cursor: not-allowed;
        }

        .select_box {
            margin-top: 30px;
            color: var(--app-container_title-color);

            >select {
                width: 100%;
                background: #f3f3f5;
                border-radius: 6px;
                color: var(--app-container_title-color);
            }

            >select.is-placeholder {
                color: #9aa0a6;
            }

            option[disabled] {
                color: #9aa0a6;
            }

            option[hidden] {
                display: none;
            }
        }

        .tip {
            text-align: center;
            font-size: 14px;
        }

        .log_info {
            padding: 24px 16px;
            margin-top: 20px;
            background: black;
            border-radius: 8px;
            max-height: 300px;
            overflow-y: auto;

            &::-webkit-scrollbar {
                height: 6px;
            }

            &::-webkit-scrollbar-thumb {
                background: #777780;
                border-radius: 6px;
            }

            >p {
                font-size: 14px;
                color: #05df72;
                margin-bottom: 12px;
            }

            >p:last-child {
                margin-bottom: 0;
            }
        }

        .result_box {
            margin-top: 20px;

            .result {
                display: grid;
                grid-template-columns: repeat(2, 1fr);
                grid-gap: 12px;

                .result_state {
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    color: var(--app-container_title-color);

                    >div {
                        margin-bottom: 6px;
                        font-weight: bold;
                        font-size: 18px;
                        letter-spacing: 0.1em;
                    }

                    .result_txt {
                        font-size: 12px;
                    }
                }

                .result_item {
                    background: #ececf0;
                    border-radius: 6px;
                    padding: 20px 16px;
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    justify-content: center;

                    .result_title {
                        font-size: 16px;
                        color: #0a0a0a;
                        margin-bottom: 12px;
                    }

                    .speed_value {
                        font-size: 24px;
                        font-weight: bold;
                        letter-spacing: 0.1em;
                    }

                    .speed_title {
                        font-size: 12px;
                    }

                    .icon_speed {
                        width: 30px;
                        height: 30px;
                        margin-bottom: 10px;
                    }

                    .icon_speed1 {
                        width: 26px;
                        height: 26px;
                        margin-bottom: 10px;
                    }

                    .unit {
                        color: #717182;
                        display: inline-block;
                        margin-bottom: 6px;
                    }
                }
            }
        }
    }
}

.custom-content {
    p {
        color: #717182;
    }

    .address_box {
        background: #ececf0;
        border-radius: 6px;
        padding: 10px 12px;
        margin-top: 16px;
        display: flex;
        justify-content: space-between;
        align-items: center;

        >span {
            font-size: 16px;
        }

        >div {
            background: #fff;
            padding: 6px 12px;
            border-radius: 4px;
            cursor: pointer;
        }
    }
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}

.icon3-wrap {
    display: inline-flex;
    width: 17px;
    height: 17px;
    margin-right: 8px;
    align-items: center;
    justify-content: center;
}

.icon3-wrap .icon3 {
    width: 100%;
    height: 100%;
}

.is-rotating {
    animation: spin 1s linear infinite;
    transform-origin: center;
    transform-box: fill-box;
    will-change: transform;
}

.line {
    height: 1px;
    background: #d9d9d9;
    margin: 20px 0;
}

.status {
    display: inline-block;
    padding: 4px 12px;
    color: #fff;
    border-radius: 6px;
    font-size: 12px;
    margin-bottom: 10px;
}

.status_bg1 {
    background: #00c950;
}

.status_bg2 {
    background: #2b7fff;
}

.status_bg3 {
    background: #ef4444;
}

.status_bg4 {
    background: #f0b100;
}
</style>

<style lang="scss" scoped>
/* 移动端样式 - 基于827px设计图 */
@media (max-width: 827px) {
    .item_container {
        grid-template-columns: repeat(1, 1fr);
        grid-gap: 12px;
        padding-bottom: 16px;
    }
}
</style>