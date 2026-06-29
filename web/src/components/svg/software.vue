<template>
   <svg width="200" height="200" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" fill="none">
      <path
         d="M511.3 116.7l339.1 193.8v387.6L511.3 891.9 172.2 698.1V310.5l339.1-193.8zM802 345.6L535.5 516.7v305.5L802 670V345.6z m-581.4 0.3V670l266.5 152.3V516.7L220.6 345.9z m434.1-87.3L401.1 405l110.3 71.4 248.9-161.5L658 256.4c-1.1 0.7-2.2 1.5-3.3 2.2z m-143.4-86L262.2 314.9l93.4 60.5c0.5-0.4 1.1-0.7 1.6-1l252.3-145.7-98.2-56.1z m0 0"
         :fill="color" p-id="14790"></path>
   </svg>
</template>

<script setup lang="ts">
defineProps({
   color: {
      type: String,
      default: '#9810f9' // 默认颜色
   }
})
</script>
