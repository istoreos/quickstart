<template>
    <iframe
        class="ttyd-iframe"
        :src="host"
        allowfullscreen
        mozallowfullscreen="mozallowfullscreen"
        msallowfullscreen="msallowfullscreen"
        oallowfullscreen="oallowfullscreen"
        webkitallowfullscreen="webkitallowfullscreen"
    ></iframe>
</template>
<script setup lang="ts">
const host = `${window.location.protocol}//${window.location.hostname}:7681`

</script>
<style lang="scss" scoped>
iframe.ttyd-iframe {
    width: 100%;
    height: 100%;
    display: block;
    border: none;
}
</style>