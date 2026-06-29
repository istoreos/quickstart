<template>
    <div class="actioner-container">
        <div class="actioner-container_body">
            <textarea :value="info"></textarea>
        </div>
        <div class="actioner-container_footer">
            <div class="close" @click="onClose">{{ $gettext("关闭") }}</div>
        </div>
    </div>
</template>
<script setup lang="ts">
import { PropType, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import request from '/@/request';
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
    raid: {
        type: Object as PropType<Disksinfo>,
        required: true
    }
})
const onClose = () => {
    // props.onSetup()
    props.Close()
}
const info = ref("")
const getData = () => {
    request.Raid.Detail.POST({
        path: props.raid.path
    }).then(res => {
        if (res.data) {
            const { result, error } = res.data
            if (error) {
                info.value = error
            } else {
                info.value = result?.detail || ""
            }
        }
    }).catch(err => {
        info.value = err.message
    })
}
getData()
</script>
<style lang="scss" scoped>
textarea {
    display: block;
    width: 100%;
    height: 100%;
    border: none;
    resize: none;
}
</style>