<template>
    <fieldset class="cbi-section">
        <textarea :value="logText" disabled></textarea>
    </fieldset>
</template>
<script setup lang="ts">
import { ref } from 'vue';
import request from '/@/request';
const logText = ref<any>("")
const getData = async () => {
    try {
        const res = await request.Smart.Log.GET()
        if (res.data) {
            const { result, error } = res.data
            if (result && result.result) {
                logText.value = result.result
            }
            if (error) {
                logText.value = error
            }
        }
    } catch (error) {
        logText.value = error
    }
}
await getData()
</script>
<style lang="scss" scoped>
textarea {
    display: block;
    width: 100%;
    height: 500px;
    padding: 1rem;
    font-size: 14px;
    resize: none;
    border: 1px solid #999;
    border-radius: 3px;
}
</style>