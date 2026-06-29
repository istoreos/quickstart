<template>
    <label class="checkbox_switch">
        <input type="checkbox" v-model="value">
        <span class="checkbox_switch_on">
            <svg-switch-on />
        </span>
        <span class="checkbox_switch_off">
            <svg-switch-off />
        </span>
        <slot></slot>
    </label>
</template>
<script setup lang="ts">
import SvgSwitchOn from '/@/components/svg/switch-on.vue';
import SvgSwitchOff from '/@/components/svg/switch-off.vue';
import { computed } from 'vue';
const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    }
})
const emits = defineEmits(['update:modelValue'])
const value = computed<boolean>({
    get: ()=>props.modelValue.valueOf(),
    set: (v: boolean)=>emits('update:modelValue', v)
})
</script>
<style lang="scss" scoped>
label.checkbox_switch {
    cursor: pointer;
    display: flex !important;
    align-items: center;
    width: initial !important;
    input[type="checkbox"] {
        height: 0 !important;
        width: 0 !important;
        opacity: 0 !important;
        margin: 0 !important;
        padding: 0 !important;
        border: none !important;
    }
    .checkbox_switch_on, .checkbox_switch_off {
        flex: none;
    }
    .checkbox_switch_on {
        display: none !important;
    }
    .checkbox_switch_off {
        display: inline-flex !important;
    }
    input[type="checkbox"]:checked ~ .checkbox_switch_on {
        display: inline-flex !important;
    }
    input[type="checkbox"]:checked ~ .checkbox_switch_off {
        display: none !important;
    }
    svg {
        height: 1em;
        width: 2em;
    }
}
</style>