<template>
  <transition name="fade">
    <div v-if="visible" class="modal-overlay" @click.self="handleOverlayClick">
      <transition name="slide">
        <div class="modal-container" :style="{ maxWidth: computedWidth }">
          <!-- 头部 -->
          <div class="modal-header">
            <h3 class="modal-title">{{ $gettext(title) }}</h3>
            <button v-if="showClose" class="modal-close" @click="handleCancel" aria-label="Close">
              &times;
            </button>
          </div>

          <!-- 内容区域 -->
          <div class="modal-content">
            <slot></slot>
          </div>

          <!-- 底部按钮 -->
          <div class="modal-footer" v-if="footerShow">
            <slot name="footer">
              <button class="modal-button cancel" @click="handleCancel">{{$gettext('取消')}}</button>
              <button class="modal-button confirm" @click="handleConfirm">{{$gettext('保存')}}</button>
            </slot>
          </div>
        </div>
      </transition>
    </div>
  </transition>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

interface Props {
  modelValue: boolean
  title?: string
  showClose?: boolean
  closeOnClickOverlay?: boolean
  footerShow?: boolean
  width?: string | number
}

const props = withDefaults(defineProps<Props>(), {
  title: '提示',
  showClose: true,
  closeOnClickOverlay: false,
  width: '500px',
  footerShow: true
})

// 计算宽度值
const computedWidth = computed(() => {
  if (typeof props.width === 'number') {
    return `${props.width}px`
  }
  return props.width
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel', 'close'])

const visible = ref(props.modelValue)

watch(() => props.modelValue, (val) => {
  visible.value = val
})

watch(visible, (val) => {
  emit('update:modelValue', val)
  if (!val) {
    emit('close')
  }
})

const customWidth = computed(() => {
  if (typeof props.width === 'number') {
    return `${props.width}px`
  }
  return props.width
})

const handleClose = () => {
  visible.value = false
}

const handleOverlayClick = () => {
  if (props.closeOnClickOverlay) {
    handleCancel()
  }
}

const handleConfirm = () => {
  emit('confirm')
}

const handleCancel = () => {
  emit('cancel')
  handleClose()
}

// 暴露方法给父组件
defineExpose({
  show: () => visible.value = true,
  hide: () => visible.value = false
})
</script>

<style lang="scss">
$primary-color: #553AFE;
$border-radius: 8px;
$box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 999;
  backdrop-filter: blur(2px);
}

.modal-container {
  background-color: white;
  border-radius: $border-radius;
  box-shadow: $box-shadow;
  width: 90%;
  max-width: v-bind('computedWidth');
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  .modal-header {
    padding: 8px 12px;
    border-bottom: 1px solid #f0f0f0;
    display: flex;
    justify-content: space-between;
    align-items: center;

    .modal-title {
      margin: 0;
      font-size: 18px;
      color: #333;
      padding: 0;
      text-align: center;
      background: transparent !important;
    }

    .modal-close {
      background: none;
      border: none;
      font-size: 24px;
      cursor: pointer;
      color: #999;
      transition: color 0.2s;

      &:hover {
        color: #666;
      }
    }
  }

  .modal-content {
    padding: 18px;
    overflow-y: auto;
    flex: 1;
  }

  .modal-footer {
    padding: 8px 12px;
    border-top: 1px solid #f0f0f0;
    display: flex;
    justify-content: flex-end;
    gap: 12px;

    .modal-button {
      padding: 4px 16px;
      border-radius: 4px;
      font-size: 14px;
      cursor: pointer;
      transition: all 0.2s;
      border: 1px solid transparent;

      &.cancel {
        background-color: white;
        border-color: #ddd;
        color: #666;

        &:hover {
          background-color: #f5f5f5;
        }
      }

      &.confirm {
        background-color: $primary-color;
        color: white;

        &:hover {
          background-color: darken($primary-color, 5%);
        }
      }
    }
  }
}

/* 移动端适配 */
@media (max-width: 768px) {
  .modal-container {
    width: 95%;
    max-width: none;
    max-height: 90vh;
    margin: 0 10px;

    .modal-header {
      padding: 12px 16px;

      .modal-title {
        font-size: 16px;
        background: transparent !important;
      }

      .modal-close {
        font-size: 20px;
      }
    }

    .modal-content {
      padding: 16px;
    }

    .modal-footer {
      padding: 12px 16px;
      flex-direction: column-reverse;
      gap: 8px;

      .modal-button {
        width: 100%;
        padding: 10px;
        font-size: 15px;
      }
    }
  }
}

/* 过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateY(-20px);
  opacity: 0;
}
</style>