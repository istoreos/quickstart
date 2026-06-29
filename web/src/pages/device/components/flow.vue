<template>
    <div class="flow">
        <div ref="el" class="echart"></div>
        <div class="flow-data">
            <span v-if="currUpload">{{ $gettext("上传:") }} {{ currUpload }}</span>
            <span v-if="currDownload">{{ $gettext("下载:") }} {{ currDownload }}</span>
        </div>
    </div>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
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
const props = defineProps({
    ipParam: {
        type: String,
    }
})
const ipParam = ref(props.ipParam)
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

const getData = async () => {
    try {
        const res = await request.DeviceMangement.speedsForOneDevice.POST({ip:ipParam.value})
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
const el = ref<HTMLElement | null>()

let MyChart: echarts.ECharts | null = null
// 基础面积图
const NewBasic = (event: HTMLElement) => {
    const dark = isDark()
    MyChart = echarts.init(event, dark ? 'dark' : "light");
    MyChart.setOption({
        animation: false,
        backgroundColor: dark ? "#88888822" : "#fff",
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
            text: $gettext("流量统计"),
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
        legend: {
            padding: 0,
            align: "right",
            top: "10px",
            data: [
                {
                    name: $gettext("上传"),
                    itemStyle: {
                        color: 'rgb(85, 58, 254)'
                    }
                },
                {
                    name: $gettext("下载"),
                    itemStyle: {
                        color: 'rgb(32, 199, 247)'
                    }
                }
            ],
            textStyle: {
                color: dark ? "#cccccc" : "rgba(0, 0, 0, 0.6)"
            },
            lineStyle: {
                color: "#333"
            }
            // inactiveBorderColor: "#333"
        },
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

onMounted(() => {
    setTimeout(async () => {
        if (el.value) {
            await getData()
            const myChart = NewBasic(el.value)
            //myChart.appendData
            const element = el.value
            myChart.resize({
                width: element.clientWidth,
                height: element.clientHeight
            })
            window.addEventListener("resize", () => {
                myChart.resize({
                    width: element.clientWidth,
                    height: element.clientHeight,
                })
            })
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
})
</script>
<style lang="scss" scoped>
.flow {
    position: relative;
    height: 260px;
    // background-color: var(--flow-bg-color);


    .echart {
        width: 100%;
        height: 100%;
    }

    .flow-data {
        position: absolute;
        right: 10px;
        top: 10px;

        span {
            display: block;
            color: var(--flow-span-color);
            font-size: 12px;
            margin-bottom: 5px;
            font-weight: 600;
            font-family: PingFangSC-Semibold, PingFang SC;

        }
    }
}

@media screen and(max-width: 600px) {
    .flow {
        height: 55vw;
    }
}
</style>