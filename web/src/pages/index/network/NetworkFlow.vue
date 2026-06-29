<template>
    <div class="network_container" ref="containerRef">
        <!-- 标题区域 -->
        <div class="title_box">
            <div class="display_flex">
                <flowIcon color="#20c7f7" class="icon" />
                <span>{{ $gettext("网络流量") }}</span>
            </div>
            <div>
                <div class="network_tag">
                    <div class="tag_item">
                        <div class="tag_dn"></div>
                        <span>{{ $gettext("下载") }}</span>
                    </div>
                    <div class="tag_item">
                        <div class="tag_up"></div>
                        <span>{{ $gettext("上传") }}</span>
                    </div>
                </div>
                <!-- <div class="speed_box">
                    <span>{{ $gettext("下载速度") }}：</span>
                    <div style="color: #1596fd">{{ currDownload }}</div>
                </div>
                <div class="speed_box">
                    <span>{{ $gettext("上传速度") }}：</span>
                    <div style="color: #00a63e;">{{ currUpload }}</div>
                </div> -->
            </div>
        </div>

        <!-- 流量数据区域 -->
        <div class="speed">
            <div class="speed_item">
                <span>{{ $gettext("下载速度") }}</span>
                <div style="color: #1596fd">{{ currDownload }}</div>
            </div>
            <div class="speed_item">
                <span>{{ $gettext("上传速度") }}</span>
                <div style="color: #00a63e;">{{ currUpload }}</div>
            </div>
        </div>

        <!-- 图表区域 -->
        <div ref="el" class="echart"></div>

        <!-- 数据区域 -->
        <!-- <div class="speed">
            <div class="speed_item">
                <span>{{ $gettext("今日下载") }}</span>
                <div style="color: #155dfc">2.34 GB</div>
            </div>
            <div class="speed_item">
                <span>{{ $gettext("今日上传") }}</span>
                <div style="color: #00aea8">473 MB</div>
            </div>
            <div class="speed_item">
                <span>{{ $gettext("峰值下载") }}</span>
                <div style="color: #9839fb">1.2 MB/s</div>
            </div>
            <div class="speed_item">
                <span>{{ $gettext("峰值上传") }}</span>
                <div style="color: #f54a00">156.8 KB/s</div>
            </div>
        </div> -->
    </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import flowIcon from "/@/components/svg/flow.vue"
import * as echarts from 'echarts/core';
import {
    TitleComponent,
    GridComponent,
    TooltipComponent,
    LegendComponent,
} from 'echarts/components';
import {
    LineChart,
} from 'echarts/charts';
import {
    CanvasRenderer
} from 'echarts/renderers';
import request from '/@/request';
import utils from '/@/utils';
import { isDark } from '/@/utils/theme';
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()
echarts.use([
    TitleComponent,
    GridComponent,
    LineChart,
    CanvasRenderer,
    TooltipComponent,
    LegendComponent,
]);

const statistics = ref<NetworkStatisticsModel[]>()
const renderTime = (index: number) => {
    const item = statistics.value?.[index]
    return (!item || item.startTime == 0) ? "" : (dateForm(item.startTime * 1000 as unknown as string) + "-" + dateForm(item.endTime * 1000 as unknown as string))
}
const upload = computed(() => {
    let items = <{
        value: number
    }[]>[]
    statistics.value?.forEach(item => {
        items.push({
            value: item.uploadSpeed
        })
    })
    return items
})
const download = computed(() => {
    let items = <{
        value: number
    }[]>[]
    statistics.value?.forEach(item => {
        items.push({
            value: item.downloadSpeed
        })
    })
    return items

})
const currUpload = computed(() => {
    let result = ""
    if (statistics.value) {
        let count = statistics.value?.length || 0
        if (count > 0) {
            let item = statistics.value[count - 1]
            result = byteToSize(item.uploadSpeed) + "/s"
        }
    }
    return result
})
const currDownload = computed(() => {
    let result = ""
    if (statistics.value) {
        let count = statistics.value?.length || 0
        if (count > 0) {
            let item = statistics.value[count - 1]
            result = byteToSize(item.downloadSpeed) + "/s"
        }
    }
    return result
})

const stack = computed(() => {
    let items = <{
        value: number
    }[]>[]
    statistics.value?.forEach(item => {
        items.push({
            value: item.downloadSpeed + item.uploadSpeed
        })
    })
    return items
})

const getData = async () => {
    try {
        const res = await request.Network.Statistics.GET()
        if (res.data) {
            if (res.data.result?.items) {
                const slots = res.data.result.slots || 10
                if (res.data.result.items.length < slots) {
                    let items = res.data.result.items
                    while (items.length < slots) {
                        items = [{ downloadSpeed: 0, endTime: 0, startTime: 0, uploadSpeed: 0 }].concat(items)
                    }
                    statistics.value = items
                } else if (res.data.result.items.length > slots) {
                    statistics.value = res.data.result.items.slice(slots - res.data.result.items.length)
                } else {
                    statistics.value = res.data.result.items
                }
            }
        }
    } catch (error) {
        console.log(error);
    }
}
const dateForm = utils.dateForm
const byteToSize = utils.byteToSize
// https://echarts.apache.org/examples/zh/editor.html?c=area-basic&lang=ts

const el = ref<HTMLElement | null>()
const containerRef = ref<HTMLElement | null>() 
let MyChart: echarts.ECharts | null = null
let resizeObserver: ResizeObserver | null = null
let onWindowResize: (() => void) | null = null   // 为了卸载时能移除

// 基础面积图
const NewBasic = (event: HTMLElement) => {
    const dark = isDark()
    MyChart = echarts.init(event, dark ? 'dark' : "light");
    MyChart.setOption({
        animation: false,
        backgroundColor: dark ? "#2c2c2c" : "#fff",
        //  backgroundColor: "#fff",
        color: ["transparent", "transparent"],
        //   color: ["rgba(32, 199, 247, 0)", "rgba(85, 58, 254, 0)"],
        tooltip: {
            trigger: 'axis',
            formatter: (params: any) => {
                if (Array.isArray(params)) {
                    let result = ``
                    if (params.length > 0) {
                        result = renderTime(params[0].axisValue)
                    }
                    for (let i = 0; i < params.length; i++) {
                        result = `${result}<br>${params[i].seriesName}: ${byteToSize(params[i].value)}/s`
                    }
                    return result.toString()
                } else {
                    const param = params
                    return `${renderTime(param.axisValue)}<br>${param.seriesName}: ${byteToSize(param.value as number)}/s`
                }
            }
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            //data: selpData,
            splitLine: {
                lineStyle: {
                    // 使用深浅的间隔色
                    color: ['#999']
                },
                show: false
            },
            name: "",
            show: false,
            nameGap: 0,
            nameTextStyle: {
                height: 0,
                lineHeight: 0,
                padding: 0
            },
        },
        title: {
            text: '',
            textStyle: {
                fontSize: 12,
                color: dark ? "#cccccc" : "rgba(0, 0, 0, 0.6)"
            },
            top: "10px",
            left: "10px",

        },
        yAxis: {
            type: 'value',
            name: "",
            // minInterval: 100000,
            // interval: 10000,
            minInterval: 10000,
            interval: 1000,
            // maxInterval: 10000000,
            // width: 100,
            axisLabel: {
                formatter: function (value: number, index: number) {
                    return `${byteToSize(value)}/s`
                },
                color: "#fff",
                show: false,
            },
            nameTextStyle: {
                color: "#fff"
            },
            splitLine: {
                lineStyle: {
                    // 使用深浅的间隔色
                    color: ['#999']
                },
                show: false
            }
        },
        series: [
            {
                name: $gettext("下载"),
                data: download.value,
                type: 'line',
                // stack: 'Total', //数据堆积
                symbol: 'none',
                showSymbol: false,
                symbolSize: 0,
                smooth: true,
                areaStyle: {
                    color:
                    {
                        type: 'linear',
                        x: 0,
                        y: 0,
                        x2: 0,
                        y2: 1,
                        colorStops: [{
                            offset: 0, color: 'rgba(32, 199, 247, 1)' // 0% 处的颜色
                        }, {
                            offset: 1, color: 'rgba(32, 199, 247, 0.1)' // 100% 处的颜色
                        }],
                        global: false
                    },
                },
            },
            {
                name: $gettext("上传"),
                data: upload.value,
                type: 'line',
                // stack: 'Total',
                symbol: 'none',
                showSymbol: false,
                symbolSize: 0,
                smooth: true,
                areaStyle: {
                    color:
                    {
                        type: 'linear',
                        x: 0,
                        y: 0,
                        x2: 0,
                        y2: 1,
                        colorStops: [{
                            offset: 0, color: 'rgba(85, 58, 254, 1)' // 0% 处的颜色
                        }, {
                            offset: 1, color: 'rgba(85, 58, 254, 0.1)' // 100% 处的颜色
                        }],
                        global: false // 缺省为 false
                    }
                },

            },
            // {
            //     name: "总量",
            //     data: stack.value,
            //     type: 'line',
            //     // stack: 'Total',
            //     smooth: true,
            //     areaStyle: {},

            // }
        ],
        grid: {
            left: '2%',
            right: '2%',
            bottom: '0%',
            top: "10%",
            containLabel: true
        },
    });
    return MyChart
}

// 统一的重算图表尺寸
const resizeChart = () => {
    if (!MyChart || !el.value) return
    const w = Math.max(el.value.clientWidth, 50)
    const h = Math.max(el.value.clientHeight, 50)
    MyChart.resize({
        width: w,
        height: h
    })
}

onMounted(() => {
    setTimeout(async () => {
        if (el.value) {
            await getData()
            const myChart = NewBasic(el.value)
            const element = el.value

            // 初次渲染按父盒子来一遍
            resizeChart()

            onWindowResize = () => {
                resizeChart()
            }
            window.addEventListener("resize", onWindowResize)

            // 新增：监听外层容器和图表本身，flex/2:1/开关F12都会触发
            if ('ResizeObserver' in window) {
                resizeObserver = new ResizeObserver(() => {
                    resizeChart()
                })
                if (containerRef.value) {
                    resizeObserver.observe(containerRef.value)
                }
                resizeObserver.observe(element)
            }

            const tick = async () => {
                if (MyChart == null) {
                    return
                }
                if (!document.hidden) {
                    await getData()
                    if (MyChart == null) {
                        return
                    }
                    myChart.setOption({
                        series: [
                            {

                                name: $gettext("下载"),
                                data: download.value,
                                type: 'line',
                                areaStyle: {},
                                smooth: true,

                                // label: {
                                //     show: true,
                                //     formatter: '{c}Mbs'

                                // }
                            },
                            {
                                name: $gettext("上传"),
                                data: upload.value,
                                type: 'line',
                                areaStyle: {},
                                smooth: true,

                                // label: {
                                //     show: true,
                                //     formatter: '{c}Mbs'
                                // },
                            },
                            // {
                            //     name: "总量",
                            //     data: stack.value,
                            //     type: 'line',
                            //     // stack: 'Total',
                            //     smooth: true,
                            //     areaStyle: {},
                            // }
                        ],
                    });
                    // 数据刷完也同步一下尺寸，避免轴变了撑出去
                    resizeChart()
                }
                setTimeout(tick, 5000)
            }
            setTimeout(tick, 5000)
        }
    }, 900)
})

onUnmounted(() => {
    if (MyChart != null) {
        MyChart.dispose()
        MyChart = null
    }
    if (onWindowResize) {
        window.removeEventListener("resize", onWindowResize)
        onWindowResize = null
    }
    if (resizeObserver) {
        resizeObserver.disconnect()
        resizeObserver = null
    }
})
</script>

<style lang="scss" scoped>
.icon {
    width: 1.5rem;
    height: 1.5rem;
    margin-right: 6px;
}

.display_flex {
    display: flex;
    align-items: center;
}

.network_container {
    border: 1px solid var(--border-color);
    border-radius: 10px;
    padding: 20px 14px;
    box-sizing: border-box;
    background-clip: padding-box;
    background: var(--card-bg-color);
    height: 100%;
    display: flex;
    flex-direction: column;
    /* 为了让下面 .echart 能在 flex 场景下被压缩而不是把外面撑爆 */
    min-height: 0;

    .title_box {
        display: flex;
        justify-content: space-between;
        // align-items: flex-start;
        align-items: center;
        margin-bottom: 26px;
        flex-shrink: 0;

        >span {
            font-size: 16px;
            font-weight: 600;
        }

        .network_tag {
            display: flex;
            align-items: center;

            .tag_item {
                display: flex;
                align-items: center;
                font-size: 12px;
                margin-left: 16px;

                >span {
                    line-height: 1;
                }

                >div {
                    width: 12px;
                    height: 12px;
                    border-radius: 50%;
                    margin-right: 6px;
                }

                .tag_dn {
                    background: #20c7f7;
                }

                .tag_up {
                    background: #553afe;
                }
            }
        }
    }

    .echart {
        flex: 1;
        min-height: 200px;
        /* 防止 flex 子项在横向被压时出现溢出 */
        min-width: 0;
    }

    .speed {
        display: flex;
        flex-shrink: 0;

        .speed_item {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;

            >span {
                font-size: 12px;
                color: #999;
                margin-bottom: 10px;
            }

            >div {
                font-size: 16px;
                color: #333;
            }
        }
    }
}

.speed_box {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    margin-top: 16px;
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    .network_container {
        border-radius: 6px;
        padding: 10px;

        .title_box {
            margin-bottom: 16px;

            >span {
                font-size: 14px;
                font-weight: 600;
            }
        }
    }
}
</style>
