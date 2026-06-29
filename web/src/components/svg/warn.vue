<template>
   <svg width="200" height="200" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" fill="none">
      <path
         d="M512 179.2l390.4 627.2H128l384-627.2m0-64c-19.2 0-44.8 12.8-51.2 32l-390.4 627.2c-25.6 44.8 6.4 96 51.2 96H896c51.2 0 83.2-57.6 51.2-96l-384-627.2c-6.4-19.2-32-32-51.2-32z"
         :fill="color" p-id="4490"></path>
      <path d="M512 640c-19.2 0-32-12.8-32-32v-192c0-19.2 12.8-32 32-32s32 12.8 32 32v192c0 19.2-12.8 32-32 32z" :fill="color"
         p-id="4491"></path>
      <path d="M512 723.2m-32 0a32 32 0 1 0 64 0 32 32 0 1 0-64 0Z" :fill="color" p-id="4492"></path>
   </svg>
</template>

<script setup lang="ts">
defineProps({
   color: {
      type: String,
      default: '#f54900' // 默认颜色
   }
})
</script>
