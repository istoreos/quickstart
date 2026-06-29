<template>
    <article>
        <slot />
    </article>
</template>
<script setup lang="ts">
</script>
<style lang="scss" scoped>
article {
    flex: 0 0 100%;
    max-width: 20%;
    position: relative;
    border-radius: 4px;
    padding: 10px;
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 1080px) {
    article {
        max-width: 33.333%;
    }
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    article {
        max-width: 50%;
    }
}
</style>