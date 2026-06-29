<template>
    <Card :title='$gettext("系统信息")' :showFooter="false" style="width: 100%;height: 100%; display: block;">
        <template #icon>
            <computerIcon class="icon computerIcon" />
        </template>
        <template #settings>
            <div class="btn_settings" @click="openInfo">
                <computerIcon color="#0a0a0a" class="icon2 computerIcon" style="margin-right: 6px;" />
                <span>{{ $gettext('信息概览') }}</span>
            </div>
        </template>
        <template #settings-menu>
            <div><a @click="onClickSystem">{{ $gettext("系统监控") }}</a></div>
            <div><a href="/cgi-bin/luci/admin/system/flash">{{ $gettext("备份升级") }}</a></div>
            <div><a href="/cgi-bin/luci/admin/store/pages/maintance">{{ $gettext("插件备份") }}</a></div>
        </template>
        <div class="content">
            <div class="chart_box">
                <div class="chart">
                    <PieChart :value="getCpuUsage" :color="getCpuUsageColor" icon="chip" :label="`${getCpuUsage}%`" width="150px"
                        height="150px" />
                    <div>{{ $gettext("CPU使用率") }}</div>
                </div>
                <div class="chart">
                    <PieChart :value="getTemperature / 1.5" :color="getTemperatureColor" icon="temperature"
                        :label="`${getTemperature}℃`" width="150px" height="150px" />
                    <div>{{ $gettext("CPU温度") }}</div>
                </div>
                <div class="chart">
                    <PieChart :value="getMemUsage" :color="getMemUsageColor" icon="lightning" :label="`${getMemUsage}%`"
                        width="150px" height="150px" />
                    <div>{{ $gettext("内存使用率") }}</div>
                </div>
            </div>
            <div class="info">
                <div class="item1 bgcolor1">
                    <div>
                        <chipIcon color="#155dfc" class="icon1" style="margin-bottom: 0;" />
                        <span>{{ $gettext("设备型号") }}</span>
                    </div>
                    <span style="font-weight: bold;margin-top: 2px;">{{ version?.model }}</span>
                </div>
                <div class="item1 bgcolor2">
                    <div>
                        <systemIcon color="#00a63e" class="icon1" style="margin-bottom: 0;" />
                        <span>{{ $gettext("固件版本") }}</span>
                    </div>
                    <span style="font-weight: bold;margin-top: 2px;">{{ version?.firmwareVersion }}（{{ $gettext("内核")}}：{{ version?.kernelVersion }}）</span>
                </div>

                <!-- <div class="item">
                    <div>{{ $gettext("内核版本") }}：</div>
                    <span>{{ version?.kernelVersion }}</span>
                </div> -->
                <div class="item">
                    <div>{{ $gettext("系统时间") }}：</div>
                    <span>{{ systemStatus?.localtime }}</span>
                </div>
                <div class="item">
                    <div>{{ $gettext("已启动") }}：</div>
                    <span>{{ stamp(systemStatus?.uptime) }}</span>
                </div>

                <!-- 占位标签 -->
                <!-- <div></div> -->

            </div>
        </div>
    </Card>
</template>

<script lang="ts" setup>
import Card from "../components/Card.vue"
import PieChart from "../components/PieChart.vue"
import computerIcon from "/@/components/svg/computer.vue"
import chipIcon from "/@/components/svg/chip.vue"
import systemIcon from "/@/components/svg/system1.vue"

import { computed, onMounted, ref } from "vue"
import { useGettext } from '/@/plugins/i18n'
import { useSystemStore } from "/@/plugins/store"
import utils from "/@/utils"
import appUtils from "/@/utils/app";

const { $gettext } = useGettext()
const systemStore = useSystemStore()
const version = computed(() => systemStore.version)
const systemStatus = computed(() => systemStore.systemStatus)
const showBlockMenu = ref(false)
const getCpuUsage = computed(() => {
    return systemStatus.value?.cpuUsage || 0
})
const getTemperature = computed(() => {
    return systemStatus.value?.cpuTemperature || 0
})
const getMemUsage = computed(() => {
    const result = systemStatus.value?.memAvailablePercentage || 100
    return 100 - result
})

// 计算 CPU 使用率颜色
const getCpuUsageColor = computed(() => {
    const cpuUsage = getCpuUsage.value;
    if (cpuUsage < 76) return "#3b82f6";
    if (cpuUsage >= 76 && cpuUsage < 96) return "#f59e0b";
    return "#ef4444";
});
// 计算 CPU 温度颜色
const getTemperatureColor = computed(() => {
    const temperature = getTemperature.value;
    // if (temperature <= 55) return "#22c55e"; // Green
    // if (temperature > 55 && temperature <= 70) return "#f97316"; // Orange
    // return "#dc2626"; // Red
    if (temperature <= 115) return "#22c55e";
    return "#f97316"
});
// 计算内存使用率颜色
const getMemUsageColor = computed(() => {
    const memUsage = getMemUsage.value;
    if (memUsage < 76) return "#8b5cf6"; // Green
    if (memUsage >= 76 && memUsage < 96) return "#fb923c"; // Orange
    return "#b91c1c"; // Red
});

const openInfo = () => {
    location.href = "/cgi-bin/luci/admin/status/overview"
}

const stamp = utils.stampForm
onMounted(() => {
})

const BlockMenu = () => {
    showBlockMenu.value = !showBlockMenu.value
}

const onClickSystem = () => {
    BlockMenu()
    appUtils.installAndGo("app-meta-netdata", "NetData", "/cgi-bin/luci/admin/status/netdata")
}
</script>

<style lang="scss" scoped>
.icon {
    width: 1.3rem;
    height: 1.3rem;
}

.icon1 {
    width: 2rem;
    height: 2rem;
    margin-bottom: 8px;
}
.icon2 {
    width: 1rem;
    height: 1rem;
}

:deep(.computerIcon) {
    path {
        fill: var(--app-container_title-color) !important;
    }
}

a {
    color: #1e1e1e;
    text-decoration: none;
    cursor: pointer;
    font-size: 14px;
    display: block;

}

.content {
    color: #333;
    margin-top: 20px;
    margin-bottom: 20px;
    padding: 0 10px;
    font-weight: normal;

    .chart_box {
        padding-bottom: 20px;
        border-bottom: 1px solid var(--btn-border-color);
        margin-bottom: 20px;
        display: flex;

        .chart {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            color: var(--app-container_title-color);

            >div {
                margin-top: 4px;
            }
        }
    }

    .info {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 16px;

        .item {
            display: flex;
            justify-content: center;

            >div {
                color: var(--app-container_title-color);
            }

            >span {
                color: var(--app-container_status-label_block);
                font-size: 16px;
                line-height: 1;
            }
        }

        .item1 {
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            margin-top: 20px;
            padding: 30px;

            >div {
                display: flex;
                align-items: center;
                margin-bottom: 8px;

                >span {
                    margin-left: 8px;
                }
            }
        }

        .bgcolor1 {
            background: #e9f2ff;
            border-radius: 10px;
            border: 1px solid #bedbff;
            color: #155dfc;
        }

        .bgcolor2 {
            background: #ebfdf1;
            border-radius: 10px;
            border: 1px solid #b9f8cf;
            color: #008236;
        }
    }
}
.btn_settings {
    position: relative;
    padding: 6px 18px;
    border-radius: 4px;
    border: 1px solid var(--btn-border-color);
    line-height: 1;
    display: flex;
    align-items: center;
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    .content {
        margin-top: 10px;
        margin-bottom: 10px;
        padding: 0 4px;

        .chart_box {
            padding-bottom: 10px;
            margin-bottom: 10px;
            flex-wrap: nowrap;
            overflow-x: auto;
            -webkit-overflow-scrolling: touch;

            &::-webkit-scrollbar {
                display: none; // ✅ 推荐：直接隐藏滚动条（webkit）
            }

            scrollbar-width: none;
        }

        .info {
            grid-template-columns: repeat(1, 1fr);
            gap: 6px;

            .item1 {
                margin-top: 6px;
                padding: 10px;
            }

            .bgcolor1 {
                background: #e9f2ff;
                border-radius: 10px;
                border: 1px solid #bedbff;
                color: #155dfc;
            }

            .bgcolor2 {
                background: #ebfdf1;
                border-radius: 10px;
                border: 1px solid #b9f8cf;
                color: #008236;
            }
        }
    }
}
</style>