<template><CardShell :manifest="manifest" :form="summary?'summary':'standard'"/></template>
<script setup lang="ts">
import CardShell from '../home/cards/CardShell.vue'
import {cardManifest} from '../home/cards/contract'
defineProps<{summary?:boolean}>()
const manifest=cardManifest((window as any).__quickstartPreviewCardManifest)
</script>
