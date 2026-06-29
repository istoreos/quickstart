<template>
    <div class="switch" :class="{
        'is-checked': modelValue,
        'is-disabled': disabled
    }" @click="toggleSwitch">

        <span class="switch__core" :style="{
            backgroundColor: modelValue ? activeColor : inactiveColor,
            borderColor: modelValue ? activeColor : inactiveColor
        }">
            <span class="switch__button"></span>
        </span>
        <input type="checkbox" class="switch__input" :checked="modelValue" :disabled="disabled">

    </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
    modelValue: boolean
    disabled?: boolean
    activeColor?: string
    inactiveColor?: string
}

const props = withDefaults(defineProps<Props>(), {
    disabled: false,
    activeColor: '#409EFF',
    inactiveColor: '#DCDFE6',
    modelValue: false
})

const emit = defineEmits(['update:modelValue', 'change', 'beforeChange'])

const toggleSwitch = () => {
    emit('beforeChange', !props.modelValue)
    if (props.disabled) return
    const newValue = !props.modelValue
    emit('update:modelValue', newValue)
    emit('change', newValue)
}

const checked = computed(() => props.modelValue)
</script>

<style scoped lang="scss">
.switch {
    display: inline-flex;
    align-items: center;
    position: relative;
    font-size: 14px;
    height: 20px;

    &.is-disabled {
        opacity: 0.6;
        cursor: not-allowed;

        .switch__core {
            cursor: not-allowed;
        }
    }
}

.switch__input {
    position: absolute;
    width: 0;
    height: 0;
    opacity: 0;
    margin: 0;
    z-index: -1;
}

.switch__core {
    margin: 0;
    display: inline-block;
    position: relative;
    width: 40px;
    height: 20px;
    border: 1px solid;
    outline: none;
    border-radius: 10px;
    box-sizing: border-box;
    cursor: pointer;
    transition: border-color 0.3s, background-color 0.3s;
}

.switch__button {
    position: absolute;
    top: 1px;
    left: 1px;
    border-radius: 100%;
    transition: all 0.3s;
    width: 16px;
    height: 16px;
    background-color: #fff;
    box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.2);
}

.switch.is-checked .switch__button {
    transform: translateX(20px);
}
</style>