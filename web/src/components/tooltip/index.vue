<template>
    <div class="disk-item-tooltip" :style="portitemStyle" v-if="show" @mouseenter="onMouseenter"
        @mouseleave="onMouseleave">
        <div>{{ getDuplex(portitem.duplex) }}</div>
        <div>{{ $gettext("名称：") }}{{ portitem.name || "--" }}</div>
        <div>{{ $gettext("MAC：") }}{{ portitem.macAddress || "--" }}</div>
        <!-- <div>关联网桥：{{ portitem.master || "--" }}</div> -->
        <div>{{ $gettext("接收：") }}{{ portitem.rx_packets || "--" }}</div>
        <div>{{ $gettext("发送：") }}{{ portitem.tx_packets || "--" }}</div>
    </div>
</template>
<script lang="ts" setup>
import { computed, reactive } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import { useAppStore } from '/@/plugins/store';
const appStore = useAppStore()
const portitem = computed(() => {
    return appStore.portitemStyle.portitem
})
const show = computed(() => {
    return appStore.portitemStyle.show
})

const portitemStyle = computed(() => {
    return {
        bottom: `calc(100% - ${appStore.portitemStyle.top}px)`,
        left: `${appStore.portitemStyle.left}px`,
    }
})

const getDuplex = (v: string): string | void => {
    switch (v) {
        case "full":
            return $gettext("全双工")
        case "half":
            return $gettext("半双工")
    }
}
// 鼠标移入事件
const onMouseenter = (payload: MouseEvent) => {
    appStore.portitemStyle.show = true
}
// 鼠标移出事件
const onMouseleave = (payload: MouseEvent) => {
    appStore.portitemStyle.show = false
}

</script>
<style lang="scss" scoped>
.disk-item-tooltip {
    position: fixed;
    background: rgba(0, 0, 0, 0.7);
    z-index: 10111;
    color: #fff;
    padding: 0.5rem 1rem;
    font-size: 1em;
    min-width: 200px;
    line-height: 24px;

    &::after {
        content: "";
        position: absolute;
        bottom: -6px;
        border-color: #4c4c4c rgba(0, 0, 0, 0) rgba(0, 0, 0, 0);
        left: 0;
        right: 0;
        text-align: center;
        width: 0;
        margin: 0 auto;
        border-width: 6px 8px 0;
        border-style: solid;
    }
}
</style>