<template>
  <div class="progress-bar-wrapper">
    <div class="progress-bar" :style="progressBarStyle">
      <div 
        class="progress-fill" 
        :style="progressFillStyle"
      >
        <span v-if="showPercentage" class="percentage-text">
          {{ Math.round(percentage) }}%
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

// Props 定义
const props = defineProps({
  // 进度百分比 (0-100)
  percentage: {
    type: Number,
    default: 0,
    validator: (value) => value >= 0 && value <= 100
  },
  // 进度条颜色
  color: {
    type: String,
    default: '#4CAF50'
  },
  // 背景颜色
  backgroundColor: {
    type: String,
    default: '#e0e0e0'
  },
  // 高度
  height: {
    type: [String, Number],
    default: '20px'
  },
  // 圆角半径
  borderRadius: {
    type: [String, Number],
    default: '4px'
  },
  // 是否显示百分比文字
  showPercentage: {
    type: Boolean,
    default: true
  },
  // 是否使用渐变色
  gradient: {
    type: Boolean,
    default: false
  },
  // 渐变色配置
  gradientColors: {
    type: String,
    default: 'linear-gradient(90deg, #4CAF50, #45a049)'
  },
  // 动画持续时间（毫秒）
  duration: {
    type: Number,
    default: 1000
  }
})

// 计算样式
const progressBarStyle = computed(() => ({
  height: typeof props.height === 'number' ? `${props.height}px` : props.height,
  borderRadius: typeof props.borderRadius === 'number' ? `${props.borderRadius}px` : props.borderRadius,
  backgroundColor: props.backgroundColor,
  overflow: 'hidden'
}))

const progressFillStyle = computed(() => {
  const baseStyle = {
    height: '100%',
    width: `${props.percentage}%`,
    borderRadius: typeof props.borderRadius === 'number' ? `${props.borderRadius}px` : props.borderRadius,
    transition: `width ${props.duration}ms cubic-bezier(0.4, 0, 0.2, 1)`,
    position: 'relative',
    overflow: 'hidden'
  }

  if (props.gradient) {
    return {
      ...baseStyle,
      background: props.gradientColors
    }
  } else {
    return {
      ...baseStyle,
      background: props.color
    }
  }
})
</script>

<style scoped>
.progress-bar-wrapper {
  width: 100%;
  margin-bottom: 0;
}

.progress-bar {
  width: 100%;
  position: relative;
  box-shadow: inset 0 1px 3px rgba(0, 0, 0, 0.1);
}

.progress-fill {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: 8px;
  box-sizing: border-box;
}

.percentage-text {
  color: white;
  font-size: 12px;
  font-weight: bold;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  white-space: nowrap;
  line-height: 1;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .percentage-text {
    font-size: 10px;
    padding-right: 4px;
  }
}

/* 深色主题支持 */
@media (prefers-color-scheme: dark) {
  .progress-bar {
    box-shadow: inset 0 1px 3px rgba(255, 255, 255, 0.1);
  }
}
</style>