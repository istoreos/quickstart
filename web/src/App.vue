<template>
    <div id="main">
        <router-view></router-view>
        <tooltip />
    </div>
</template>
<script setup lang="ts">
import tooltip from "/@/components/tooltip/index.vue"
</script>
<style lang="scss">
#main,
#actioner {
    .app-btn {
        min-height: 36px;
    }
}
</style>
<style lang="scss" scoped>
#main {
    width: 100%;
}

#main {
    -webkit-box-sizing: border-box;
    -webkit-tap-highlight-color: transparent;
    box-sizing: border-box;
    word-wrap: break-word;
    outline: none;

    :deep(*) {
        -webkit-box-sizing: border-box;
        -webkit-tap-highlight-color: transparent;
        box-sizing: border-box;
        word-wrap: break-word;
        outline: none;
    }
}
</style>