<template>
    <div class="bg" ref="bg"></div>
</template>
<script setup lang="ts">
import { getCurrentInstance, onMounted, ref } from 'vue';
// const curr = getCurrentInstance()
// const { ctx } = curr
const bg = ref<HTMLElement | null>()
onMounted(() => {
    if (bg.value) {
        const body = document.querySelector("body");
        if (body) {
            if (body.append) {
                body.append(bg.value);
            } else {
                body.appendChild(bg.value);
            }
        }
    }
})
</script>
<style lang="scss" scoped>
.bg {
    position: fixed;
    top: 0;
    bottom: 0;
    left: 0;
    right: 0;
    width: 100%;
    height: 100%;
    z-index: -1;
    background: linear-gradient(180deg, #4587ff, #3f90fe, #6dd5fa, #fff);
}
</style>