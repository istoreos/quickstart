<template>
    <span class="device-scene-icon" role="img" :aria-label="label" :title="label">
        <img :src="source" alt="" aria-hidden="true" decoding="async" loading="lazy" @error="useFallback" />
    </span>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { deviceSceneIconPath, type DeviceScene } from '../deviceScene'

const props = defineProps<{ scene: DeviceScene; label: string }>()
const fallback = deviceSceneIconPath('computer')
const source = ref(deviceSceneIconPath(props.scene))

watch(() => props.scene, scene => { source.value = deviceSceneIconPath(scene) })
const useFallback = () => { source.value = fallback }
</script>

<style lang="scss" scoped>
.device-scene-icon {
    display: inline-flex;
    flex: none;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    overflow: hidden;
    background: linear-gradient(145deg, rgba(111, 99, 200, .11), rgba(95, 218, 180, .07));
    border: 1px solid rgba(111, 99, 200, .12);
    border-radius: 12px;
}

img {
    display: block;
    width: 38px;
    height: 38px;
    object-fit: contain;
}
</style>
