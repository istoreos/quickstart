<template>
    <div class="app-container_status-label_bg" :style="`transform: translateX(${transform}px);`" ref="el"
        @mouseenter="onMouseenter" @mouseleave="onMouseleave">
        <template v-if="item.linkState == 'DOWN'">
            <div class="app-container_status-label_iconer">
                <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
                    xmlns:v="https://vecta.io/nano" width="48" height="38" viewBox="0 0 12.7 10.05">
                    <defs>
                        <filter id="A" color-interpolation-filters="sRGB">
                            <feColorMatrix result="A" values="2 -0.5 -0.5 0 0 -0.5 2 -0.5 0 0 -0.5 -0.5 2 0 0 0 0 0 1 0 " />
                            <feColorMatrix values="0 0 0 -1 0 0 0 0 -1 0 0 0 0 -1 0 0 0 0 1 0" />
                            <feColorMatrix in="A" values="2 -0.5 -0.5 0 0 -0.5 2 -0.5 0 0 -0.5 -0.5 2 0 0 0 0 0 1 0 " />
                        </filter>
                        <path id="B" d="M80.56 75.75h3.91v22.79h-3.91z" />
                    </defs>
                    <g transform="translate(0 -286.95)">
                        <rect x=".21" y="287.25" width="12.33" height="9.5" ry=".57" fill="#e6e6e6" stroke="#e6e6e6"
                            stroke-linejoin="round" stroke-width=".37" paint-order="normal" />
                        <path transform="matrix(.105 0 0 .0989 -6.0834 280.6)"
                            d="M73.96 75.66h89.41c2.31 0 4.17 1.86 4.17 4.17v52.65h-21.74v9.41h-8.69v12.59h-36.87v-12.59h-8.69v-9.41H69.79V79.83c0-2.31 1.86-4.17 4.17-4.17z"
                            fill="#999" filter="url(#A)" stroke="#999" stroke-width="2.5" />
                        <g transform="matrix(.1048 0 0 .1048 -6.0999 280.7)" fill="#fff" filter="url(#A)" stroke="#fff">
                            <use xlink:href="#B" />
                            <use xlink:href="#B" x="73.04" />
                            <use xlink:href="#B" x="52.17" />
                            <use xlink:href="#B" x="41.74" />
                            <use xlink:href="#B" x="31.3" />
                            <use xlink:href="#B" x="20.87" />
                            <use xlink:href="#B" x="10.43" />
                            <use xlink:href="#B" x="62.61" />
                        </g>
                        <rect x="1.24" y="294.55" width="1.6" height="1.38" ry=".11" fill="#ccc" stroke="#ccc"
                            stroke-width=".22" paint-order="normal" />
                    </g>
                </svg>
            </div>
            <div class="app-container_status-label_text">
                <div class="text_status">{{ $gettext("已断开") }}</div>
                <div class="text_info">
                    {{ item.name }}
                    {{ item.interfaceNames ? `(${item.interfaceNames.join(",").toLocaleUpperCase()})` : "" }}
                </div>
            </div>
        </template>
        <template v-else>
            <div class="app-container_status-label_iconer">
                <svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="48" height="38"
                    viewBox="0 0 12.7 10.05" xmlns:v="https://vecta.io/nano">
                    <defs>
                        <filter id="A" color-interpolation-filters="sRGB">
                            <feColorMatrix result="A" values="2 -0.5 -0.5 0 0 -0.5 2 -0.5 0 0 -0.5 -0.5 2 0 0 0 0 0 1 0 " />
                            <feColorMatrix values="0 0 0 -1 0 0 0 0 -1 0 0 0 0 -1 0 0 0 0 1 0" />
                            <feColorMatrix in="A" values="2 -0.5 -0.5 0 0 -0.5 2 -0.5 0 0 -0.5 -0.5 2 0 0 0 0 0 1 0 " />
                        </filter>
                        <path id="B" d="M80.56 75.75h3.91v22.79h-3.91z" />
                    </defs>
                    <g transform="translate(-.03 -287.07)">
                        <rect x=".24" y="287.36" width="12.33" height="9.5" ry=".57" fill="#e6e6e6" stroke="#e6e6e6"
                            stroke-linejoin="round" stroke-width=".37" paint-order="normal" />
                        <path transform="matrix(.105 0 0 .0989 -6.0532 280.72)"
                            d="M73.96 75.66h89.41c2.31 0 4.17 1.86 4.17 4.17v52.65h-21.74v9.41h-8.69v12.59h-36.87v-12.59h-8.69v-9.41H69.79V79.83c0-2.31 1.86-4.17 4.17-4.17z"
                            fill="#4d4d4d" filter="url(#A)" stroke="#4d4d4d" stroke-width="2.5" />
                        <g transform="matrix(.1048 0 0 .1048 -6.0697 280.81)" fill="#fff" filter="url(#A)" stroke="#fff">
                            <use xlink:href="#B" />
                            <use xlink:href="#B" x="73.04" />
                            <use xlink:href="#B" x="52.17" />
                            <use xlink:href="#B" x="41.74" />
                            <use xlink:href="#B" x="31.3" />
                            <use xlink:href="#B" x="20.87" />
                            <use xlink:href="#B" x="10.43" />
                            <use xlink:href="#B" x="62.61" />
                        </g>
                        <rect x="1.27" y="294.67" width="1.6" height="1.38" ry=".11" fill="#55d400" stroke="#55d400"
                            stroke-width=".22" paint-order="normal" />
                    </g>
                </svg>
            </div>
            <div class="app-container_status-label_text">
                <div> {{ item.linkSpeed }}</div>
                <div class="text_info">
                    {{ item.name }}
                    {{ item.interfaceNames ? `(${item.interfaceNames.join(",").toLocaleUpperCase()})` : "" }}
                </div>
            </div>
        </template>
    </div>
</template>
<script setup lang="ts">
import { computed, PropType, reactive, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import { useAppStore } from '/@/plugins/store';
const appStore = useAppStore()
const props = defineProps({
    item: {
        type: Object as PropType<NetworkPort>,
        required: true
    },
    transform: {
        type: Number,
        default: 0
    }
})
const el = ref<HTMLElement | null>(null)
// 鼠标移入事件
const onMouseenter = (payload: MouseEvent) => {
    const target = payload.target as HTMLInputElement
    const { left, top } = target.getBoundingClientRect()
    appStore.portitemStyle.show = true
    appStore.portitemStyle.left = left
    appStore.portitemStyle.top = top
    appStore.portitemStyle.portitem = props.item

}
// 鼠标移出事件
const onMouseleave = (payload: MouseEvent) => {
    appStore.portitemStyle.show = false
    // setTimeout(() => {
    //     if (appStore.portitemStyle.show) {
    //         appStore.portitemStyle.show = false
    //     }
    // }, 1000)
}
</script>
<style lang="scss" scoped>
.app-container_status-label_bg {
    flex: 0 0 230px;
    width: 230px;
    height: 80px;
    display: flex;
    background: var(--app-container_status-label_bg);
    align-items: center;
    padding: 10px;
    margin-right: 10px;
    transition: 0.3s;

    .app-container_status-label_text {
        margin-left: 6px;
        font-size: 14px;
        line-height: 22px;
        text-align: left;

        .text_status {
            color: #999;
        }

        .text_style {
            margin: 6px 0;

            &.close {
                color: #999;
            }
        }

        .text_info {
            font-weight: 700;
            font-size: 14px;
        }
    }

}
</style>