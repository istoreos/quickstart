<template>
  <div class="pie-chart-wrapper" :style="{ width: width || '120px', height: height || '120px' }">
    <!-- 专门给 echarts 挂载的容器 -->
    <div ref="chartDom" class="chart-dom"></div>

    <!-- 覆盖在图表之上的中心内容（兄弟节点） -->
    <div class="center-content" v-if="icon || label">
      <ChipIcon v-if="icon === 'chip'" :color="color" class="center-icon" />
      <TemperatureIcon v-else-if="icon === 'temperature'" :color="color" class="center-icon" />
      <LightningIcon v-else-if="icon === 'lightning'" :color="color" class="center-icon" />

      <div v-if="label" class="center-label" :style="{ color: color }">{{ label }}</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import * as echarts from 'echarts/core'
import { PieChart } from 'echarts/charts'
import { CanvasRenderer } from 'echarts/renderers'
import { TooltipComponent } from 'echarts/components'

import ChipIcon from '/@/components/svg/chip.vue'
import TemperatureIcon from '/@/components/svg/temperature.vue'
import LightningIcon from '/@/components/svg/lightning.vue'

echarts.use([PieChart, CanvasRenderer, TooltipComponent])

interface Props {
  value: number // 0~100
  color?: string
  icon?: string
  label?: string
  width?: string
  height?: string
}

const props = defineProps<Props>()
const chartDom = ref<HTMLElement | null>(null)
let chartInstance: echarts.ECharts | null = null

function getOption(value: number, color?: string): echarts.EChartsCoreOption {
  return {
    tooltip: { show: false },
    series: [
      {
        type: 'pie',
        radius: ['75%', '90%'],
        avoidLabelOverlap: false,
        label: { show: false },
        labelLine: { show: false },
        z: 1, 
        zlevel: 0,
        data: [
          { value: value, itemStyle: { color: color || '#409EFF' } },
          { value: Math.max(0, 100 - value), itemStyle: { color: '#f0f0f0' } },
        ],
      },
    ],
  }
}

const initChart = () => {
  if (!chartDom.value) return
  // 复用实例：如果已存在就不再 new 一个
  chartInstance = chartInstance ?? echarts.init(chartDom.value)
  chartInstance.setOption(getOption(props.value, props.color))
}

onMounted(async () => {
  await nextTick()
  initChart()
  window.addEventListener('resize', handleResize)
})

function handleResize() {
  chartInstance?.resize()
}

watch(
  () => [props.value, props.color],
  () => {
    // 优雅更新而不是重新 init
    if (chartInstance) {
      chartInstance.setOption({
        series: [
          {
            z: 1,
            zlevel: 0,
            data: [
              { value: props.value, itemStyle: { color: props.color || '#409EFF' } },
              { value: Math.max(0, 100 - props.value), itemStyle: { color: '#f0f0f0' } },
            ],
          },
        ],
      })
    } else {
      initChart()
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  chartInstance?.dispose()
  chartInstance = null
})
</script>

<style scoped>
.pie-chart-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* echarts 挂载点要填满父容器 */
.chart-dom {
  width: 100%;
  height: 100%;
}

/* overlay */
.center-content {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
  z-index: 2;            /* 高于 chart-dom（chart 的 canvas） */
  pointer-events: none;  /* 不阻塞下面的交互；如果需要让图标可点则移除或改为在图标上开启 pointer-events:auto */
}

.center-icon {
  width: 20px;
  height: 20px;
  margin-bottom: 4px;
  display: block;
  margin-left: auto;
  margin-right: auto;
}

.center-label {
  margin-top: 4px;
  font-size: 14px;
  font-weight: 600;
}
</style>
