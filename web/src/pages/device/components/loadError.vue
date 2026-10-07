<template>
    <PageState kind="error" :title="$gettext('加载失败')" :description="message"
        :action-label="retry ? $gettext('重新加载') : ''" @action="$emit('retry')" />
</template>

<script setup lang="ts">
import { useGettext } from '/@/plugins/i18n'
import PageState from './pageState.vue'

const { $gettext } = useGettext()

withDefaults(defineProps<{
    message: string
    retry?: boolean
}>(), {
    retry: true,
})

defineEmits<{
    (event: 'retry'): void
}>()
</script>
