
<template>
    <div class="actioner-container">
        <div class="actioner-container_body">
            <svg t="1642063181211" class="icon" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg"
                p-id="5062" width="128" height="128" data-v-cda444e0>
                <path
                    d="M512 85.333333c235.648 0 426.666667 191.018667 426.666667 426.666667s-191.018667 426.666667-426.666667 426.666667S85.333333 747.648 85.333333 512 276.352 85.333333 512 85.333333z m-74.965333 550.4L346.453333 545.152a42.666667 42.666667 0 1 0-60.330666 60.330667l120.704 120.704a42.666667 42.666667 0 0 0 60.330666 0l301.653334-301.696a42.666667 42.666667 0 1 0-60.288-60.330667l-271.530667 271.488z"
                    fill="#52C41A" p-id="5063" data-v-cda444e0 />
            </svg>
            <div class="body-title">{{ $gettext("添加成功") }}</div>
            <p class="body-tips">{{ $gettext("请稍等1分钟生效后再使用。") }}</p>
            <div class="body-info">
                <span>{{ $gettext("访问地址：") }}</span>
                <a :href="target" target="_blank" rel="noopener noreferrer">{{ target }}</a>
            </div>
            <div>
                <span>{{ $gettext("可前往") }}</span>
                <a href="https://www.ddnsto.com/app/#/devices" target="_blank">{{ $gettext("DDNSTO控制台") }}</a>
                <span>{{ $gettext("查看更多详情") }}</span>
            </div>
        </div>
        <div class="actioner-container_footer">
            <div class="close" @click="onClose">{{ $gettext("关闭") }}</div>
        </div>
    </div>
</template>
<script setup lang="ts">
import { computed, ref, PropType } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

const props = defineProps({
    onSetup: {
        type: Function as PropType<(v?: string) => void>,
        required: true
    },
    target: {
        type: String,
        required: true
    }
})
const onClose = () => {
    localStorage.setItem('firstOpen', JSON.stringify(true));
    // props.onSetup()
    location.reload()
}
</script>
<style lang="scss" scoped>
.actioner-container_body {

    display: flex;
    flex-wrap: wrap;
    align-items: center;
    align-content: center;
    justify-content: center;

    svg.icon {
        width: 100px;
        height: 100px;
    }

    .body-title {
        width: 100%;
        display: block;
        color: #1e1e1e;
        font-size: 2em;
        padding: 0;
        margin: 1rem 0;
        text-align: center;
    }

    .body-info {
        color: #666;
        font-size: 1.3em;
        margin: 1rem 0;
        width: 100%;
    }

    .body-tips {
        margin: 1rem 0;
        display: block;
        width: 100%;
    }

    .body-btns {
        width: 100%;
        margin-top: 3rem;

        button {
            display: block;
            width: 100% !important;
            margin: 0.5rem 0;
        }
    }
}
</style>


