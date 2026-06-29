<template>
    <svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" fill="none" width="128" height="128">
        <path
            d="M511.232 438.8352L112.9984 40.6016A51.2 51.2 0 0 0 40.6016 112.9984L438.784 511.232 40.6016 909.4656a51.2 51.2 0 1 0 72.3968 72.448l398.2336-398.2848 398.2336 398.2848a51.2 51.2 0 1 0 72.448-72.448l-398.2848-398.2336 398.2848-398.2336A51.2 51.2 0 0 0 909.4656 40.6016L511.232 438.784z"
            p-id="1217" :fill="color" />
    </svg>
</template>
<script setup lang="ts">
defineProps({
   color: {
      type: String,
      default: '#000000' // 默认颜色
   }
})
</script>
<style lang="scss" scoped></style>