<template>
  <DialogVue v-model="visibleProxy" :title="$gettext('模块显示设置')" width="640px" :show-close="true" @cancel="handleCancel"
    @confirm="handleSave">
    <span>{{ $gettext('选择要在首页显示的功能模块，隐藏不常用的模块可以让界面更简洁') }}</span>
    <div class="module-settings">
      <div class="module-settings__header">
        <div class="module-settings__summary">
          <div class="module-settings__badge">{{ enabledCount }}</div>
          <div class="module-settings__texts">
            <span class="module-settings__title">{{ $gettext('已显示模块') }}</span>
            <span class="module-settings__sub">{{ $gettext('共') }}{{ modules.length }}{{ $gettext('个模块') }}</span>
          </div>
        </div>
        <button class="module-settings__toggle-all" type="button" @click="handleEnableAll">
          {{ $gettext('全部显示') }}
        </button>
      </div>

      <div class="module-settings__list">
        <div v-for="item in modules" :key="item.key" class="module-settings__item"
          :class="{ 'module-settings__item--disabled': item.disabled }">
          <div class="module-settings__info">
            <div class="module-settings__name">
              <span class="module-settings__dot" :class="{ 'is-active': localStates[item.key] }"></span>
              <span>{{ item.title }}</span>
            </div>
            <p class="module-settings__desc">{{ item.description }}</p>
          </div>
          <SwitchVue :model-value="localStates[item.key]" :disabled="item.disabled" active-color="#553AFE"
            inactive-color="#E5E6EB" @change="(value: boolean) => handleToggle(item.key, value)" />
        </div>
      </div>
    </div>

    <template #footer>
      <div class="module-settings__footer">
        <button class="module-settings__btn module-settings__btn--secondary" type="button" @click="handleCancel">
          {{ $gettext('取消') }}
        </button>
        <button class="module-settings__btn module-settings__btn--primary" type="button" @click="handleSave">
          {{ $gettext('保存设置') }}
        </button>
      </div>
    </template>
  </DialogVue>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import DialogVue from '/@/components/dialog/index.vue'
import SwitchVue from '../../device/components/switch.vue'
import { useGettext } from '/@/plugins/i18n'
import Toast from "/@/components/toast";

const { $gettext } = useGettext()

export interface ModuleSettingItem {
  key: string
  title: string
  description: string
  disabled?: boolean
}

interface Props {
  visible: boolean
  modules: ModuleSettingItem[]
  states: Record<string, boolean>
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update:states', value: Record<string, boolean>): void
  (e: 'cancel'): void
  (e: 'save', value: Record<string, boolean>): void
}>()

const visibleProxy = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

const createLocalStates = () => {
  const result: Record<string, boolean> = {}
  props.modules.forEach((item) => {
    result[item.key] = props.states?.[item.key] ?? true
  })
  return result
}

const localStates = reactive<Record<string, boolean>>(createLocalStates())

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      Object.assign(localStates, createLocalStates())
    }
  }
)

watch(
  () => props.states,
  () => {
    Object.assign(localStates, createLocalStates())
  },
  { deep: true }
)

watch(
  () => props.modules,
  () => {
    Object.assign(localStates, createLocalStates())
  },
  { deep: true }
)

const enabledCount = computed(() =>
  props.modules.reduce((count, item) => (localStates[item.key] ? count + 1 : count), 0)
)

// const handleToggle = (key: string, value: boolean) => {
//   if (props.modules.find((item) => item.key === key)?.disabled) {
//     return
//   }
//   localStates[key] = value
// }
const handleToggle = (key: string, value: boolean) => {
  const item = props.modules.find((it) => it.key === key)
  if (!item) return
  if (item.disabled) return

  // 如果是要关闭（value === false），并且当前只能剩下最后一个开启项，则阻止并提示
  if (value === false) {
    const enabledBefore = props.modules.reduce(
      (count, it) => (localStates[it.key] ? count + 1 : count),
      0
    )
    if (enabledBefore <= 1) {
      return Toast.Warning($gettext('请至少保留一项！'))
    }
  }

  // 允许切换
  localStates[key] = value
}

const handleCancel = () => {
  emit('cancel')
  visibleProxy.value = false
}

const handleSave = () => {
  const merged = { ...(props.states || {}) }
  props.modules.forEach((item) => {
    merged[item.key] = !!localStates[item.key]
  })
  // 只触发save事件，不直接更新状态和关闭弹窗
  // 由父组件决定是否更新状态和关闭弹窗
  emit('save', merged)
}

// 仅提供“全部显示”能力
const handleEnableAll = () => {
  props.modules.forEach((item) => {
    if (!item.disabled) {
      localStates[item.key] = true
    }
  })
}
</script>

<style lang="scss" scoped>
:deep(.modal-header) {
  border-bottom: none;
  padding-bottom: 0;

  .modal-title {
    text-align: left;
    padding-left: 6px;
  }
}

:deep(.modal-content) {
  padding-top: 0;
}

:depp(.modal-footer) {
  border-top: none !important;
}

.module-settings {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  color: #1d2129;

  .module-settings__header {
    padding: 8px 12px;
    background: linear-gradient(to right, #eff6ff, #f4f6ff, #f7f5ff, #faf6ff);
    background: -webkit-linear-gradient(left, #eff6ff, #f4f6ff, #f7f5ff, #faf6ff);
    border-radius: 10px;
  }

  .module-settings__badge {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: linear-gradient(135deg, #5677ff 0%, #9c56ff 100%);
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 18px;
  }

  .module-settings__toggle-all {
    border-radius: 8px;
  }

  &__header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }

  &__summary {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  &__badge {
    width: 36px;
    height: 36px;
    border-radius: 12px;
    background: linear-gradient(135deg, #5c3efe 0%, #7a62ff 100%);
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: 18px;
  }

  &__texts {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  &__title {
    font-size: 16px;
    font-weight: 600;
  }

  &__sub {
    font-size: 13px;
    color: #86909c;
  }

  &__toggle-all {
    min-width: 96px;
    height: 34px;
    padding: 0 16px;
    border-radius: 17px;
    border: 1px solid #e5e6eb;
    background: #fff;
    color: #1d2129;
    cursor: pointer;
    font-size: 14px;
    transition: all 0.2s;
    display: inline-block;
    line-height: 34px;
    /* 与高度一致，保证纯文本垂直居中 */
    box-sizing: border-box;
    -webkit-appearance: none;
    appearance: none;
    vertical-align: middle;

    &:hover {
      border-color: #553afe;
      color: #553afe;
    }
  }

  &__list {
    display: flex;
    flex-direction: column;
    gap: 16px;
    /* 增加每个模块间距 */
    max-height: 420px;
    overflow-y: auto;
    padding-right: 8px;
    /* 给滚动条预留空间，避免紧贴内容 */
  }

  &__item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 10px;
    border-radius: 8px;
    background: transparent;
    border: 1px solid #e5e7eb;
    gap: 24px;
    transition: background-color 0.2s ease, border-color 0.2s ease;

    &:hover {
      border-color: #8fc6ff;
      background: #fafcfe;
    }

    &--disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  &__info {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  &__name {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    font-size: 15px;
  }

  &__dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background-color: #c9ccd3;
    transition: background-color 0.2s;

    &.is-active {
      background: #553afe;
    }
  }

  &__desc {
    margin: 0;
    font-size: 13px;
    color: #86909c;
    line-height: 1.4;
    /* 与标题左侧圆点对齐：圆点10px + 间距8px = 18px */
    margin-left: 18px;
  }

  &__footer {
    width: 100%;
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  &__btn {
    min-width: 96px;
    height: 36px;
    border-radius: 8px;
    border: 1px solid transparent;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.3s ease;

    &--secondary {
      background: #fff;
      border-color: #e5e6eb;
      color: #1d2129;

      &:hover {
        border-color: #553afe;
        color: #553afe;
      }
    }

    &--primary {
      background: linear-gradient(135deg, #5c3efe 0%, #7a62ff 100%);
      color: #fff;
      border: none;

      &:hover {
        opacity: 0.8;
      }
    }
  }
}

@media (max-width: 768px) {
  .module-settings {
    /* 保持与 PC 相同的布局，仅调整字号与底部按钮排列 */
    &__header {
      flex-direction: row;
      align-items: center;
    }

    &__toggle-all {
      width: auto;
      height: 34px;
      line-height: 34px;
      font-size: 14px;
      padding: 0 14px;
    }

    &__item {
      flex-direction: row;
      align-items: center;
      padding: 14px 10px;
    }

    /* 字体整体略缩小，保证在移动端观感协调 */
    &__badge {
      font-size: 16px;
    }

    &__title {
      font-size: 15px;
    }

    &__sub {
      font-size: 12px;
    }

    &__name {
      font-size: 14px;
    }

    &__desc {
      font-size: 12px;
      margin-left: 18px;
    }

    &__list {
      max-height: 65vh;
      padding-right: 8px;
    }

    /* 仅底部按钮上下排列，满宽 */
    &__footer {
      flex-direction: column-reverse;
      align-items: stretch;
      gap: 8px;
    }

    &__btn {
      width: 100%;
      height: 36px;
      font-size: 14px;
    }
  }
}
</style>
