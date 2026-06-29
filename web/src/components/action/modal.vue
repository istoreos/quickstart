<template>
    <action-component-base :Close="Close" :type="type" :title="title">
        <slot></slot>
    </action-component-base>
</template>
<script setup lang="ts">
import ActionComponentBase from "./index.vue"
import { onMounted, onUnmounted, PropType, ref } from 'vue'
const props = defineProps({
    Close: {
        type: Function,
    },
    type: {
        type: Number as PropType<number>
    },
    title: String as PropType<string>
})

</script>
<style lang="scss">
</style>