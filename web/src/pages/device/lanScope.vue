<template><slot /></template>
<script setup lang="ts">
import { provide } from 'vue'
import { createDeviceManagement } from '/@/request/request'
import { lanClientKey, lanContextKey, type LANScope } from './lanContext'
const props = defineProps<{ networkId?: string; networks: LANScope[]; legacy?: boolean }>()
// Parent keys this boundary by network. All descendants retain one immutable client.
provide(lanClientKey, createDeviceManagement(props.legacy ? undefined : props.networkId))
provide(lanContextKey, { networkId: props.networkId, networks: props.networks, legacy: props.legacy })
</script>
