<template><ProductionDialog :title="title" @close="$emit('close')"><slot/></ProductionDialog></template>
<script setup lang="ts">
import ProductionDialog from '../home/NetworkWidgetDialog.vue'
defineProps<{title:string}>()
defineEmits<{(event:'close'):void}>()
</script>
