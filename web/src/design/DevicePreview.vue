<template>
  <p class="design-context">{{ english?'Design mode: real device-management components, synthetic policy and traffic.':'设计模式：使用真实设备管理组件，策略与流量均为示例。' }}</p>
  <div><DeviceManagement /></div>
</template>
<script setup lang="ts">
import DeviceManagement from '../pages/device/index.vue'
const english = window.vue_lang === 'en'
</script>
<style scoped>
.design-context{position:absolute;top:13px;right:16px;max-width:55%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:11px;color:#fff;margin:0;pointer-events:none}
@media(max-width:700px){.design-context{display:none}}
</style>
