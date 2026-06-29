<template>
    <div class="reusable-card" 
        role="group">
        <div class="card-header">
            <div class="left">
                <slot name="icon">
                    <!-- no default icon -->
                </slot>
                <div class="title">{{ title }}</div>
            </div>

            <div class="settings-wrapper">
                <div v-if="showSettings" class="settings-btn">
                    <!-- <settingIcon class="settings-icon" /> -->
                    <slot name="settings"></slot>
                </div>
                <!-- 下拉菜单 -->
                <transition name="fade">
                    <div v-show="isOpen" class="dropdown-menu">
                        <slot name="settings-menu"></slot>
                    </div>
                </transition>
            </div>
        </div>

        <div class="card-body">
            <slot />
        </div>
    </div>
</template>

<script setup>
import { computed, ref, onMounted, onBeforeUnmount } from "vue"

import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

onMounted(() => document.addEventListener('click', handleClickOutside))
onBeforeUnmount(() => document.removeEventListener('click', handleClickOutside))

const props = defineProps({
    title: { type: String, required: true },
    showSettings: { type: Boolean, default: true },
    isSettingsMenuOpen: { type: Boolean, default: false }
})
// 用计算属性做 v-model 双向绑定
const isOpen = computed({
  get: () => props.isSettingsMenuOpen,
  set: (val) => emit('update:isSettingsMenuOpen', val)
})

const handleClickOutside = (e) => {
  if (!e.target.closest('.settings-wrapper')) {
    isOpen.value = false
  }
}

const emit = defineEmits(['settings', 'update:isSettingsMenuOpen'])
</script>

<style scoped>
.reusable-card {
    border: 1px solid;
    border-radius: 10px;
    padding: 20px 14px;
    box-sizing: border-box;
    background-clip: padding-box;
    display: flex;
    flex-direction: column;
    gap: 12px;
    border: 1px solid var(--border-color);
    background: var(--card-bg-color);
}

.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.card-header .left {
    display: flex;
    align-items: center;
    gap: 8px;
}

.card-header ::v-deep>svg {
    width: 20px;
    height: 20px;
}

.title {
    font-size: 16px;
    font-weight: 600;
    line-height: 1;
    color: var(--app-container_title-color);
}

.settings-btn {
    cursor: pointer;
}

.card-body {
    flex: 1 1 auto;
}

.card-footer {
    display: flex;
    justify-content: center;
}

.footer-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    border-radius: 6px;
    border: 1px solid var(--btn-border-color);
    justify-content: center;
    cursor: pointer;
    width: 100%;
    max-width: 600px;
    margin-top: 16px;
}

.footer-icon {
    display: inline-flex;
    align-items: center;
}

.footer-text {
    font-size: 14px;
    font-weight: normal;
    color: var(--app-container_title-color);
}

.settings-icon {
    width: 20px;
    height: 20px;
}

.settings-icon :deep(svg),
.settings-icon :deep(g),
.settings-icon :deep(path),
.settings-icon :deep(circle),
.settings-icon :deep(rect),
.settings-icon :deep(line),
.settings-icon :deep(polyline),
.settings-icon :deep(polygon) {
  fill:   var(--app-container_title-color) !important;
  stroke: var(--app-container_title-color) !important;
}

.settings-wrapper {
    position: relative;
}

.dropdown-menu {
    position: absolute;
    top: 38px;
    right: 0;
    background: #fff;
    border-radius: 6px;
    padding: 16px 0;
    min-width: 220px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
    z-index: 10;
}

.fade-enter-active,
.fade-leave-active {
    transition: opacity 0.2s;
}

.fade-enter-from,
.fade-leave-to {
    opacity: 0;
}

.dropdown-menu ::v-deep div {
    display: block;
    width: 100%;
    text-align: center;
    padding: 8px 0;
    border: none;
    background: none;
    cursor: pointer;
    font-size: 14px;
    color: #333;
    transition: background 0.2s, color 0.2s;
}

/* 鼠标悬浮 */
.dropdown-menu ::v-deep div:hover {
    background-color: #eeeeee;
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    .reusable-card {
        padding: 10px;
        border-radius: 6px;
    }

    .title {
        font-size: 14px;
    }

    .footer-btn {
        margin-top: 6px;
    }

    .dropdown-menu {
        padding: 8px 0;
        min-width: 150px;
    }
}
</style>