<template>
    <action-component :type="1">
        <transition name="rotate" mode="out-in">
            <div class="action">
                <div class="action-body">
                    <div class="icon">
                        <svg t="1642063181211" class="icon" viewBox="0 0 1024 1024" version="1.1"
                            xmlns="http://www.w3.org/2000/svg" p-id="5062" width="128" height="128" data-v-cda444e0>
                            <path
                                d="M512 85.333333c235.648 0 426.666667 191.018667 426.666667 426.666667s-191.018667 426.666667-426.666667 426.666667S85.333333 747.648 85.333333 512 276.352 85.333333 512 85.333333z m-74.965333 550.4L346.453333 545.152a42.666667 42.666667 0 1 0-60.330666 60.330667l120.704 120.704a42.666667 42.666667 0 0 0 60.330666 0l301.653334-301.696a42.666667 42.666667 0 1 0-60.288-60.330667l-271.530667 271.488z"
                                fill="#52C41A" p-id="5063" data-v-cda444e0 />
                        </svg>
                    </div>
                    <h2 class="title">{{ $gettext("服务已启动") }}</h2>
                    <div class="info">
                        <span>{{ $gettext("前往") }}</span>

                        <a :href="target" target="_blank" rel="noopener noreferrer">{{ target }}</a>
                        <span>{{ $gettext("继续配置") }}</span>
                    </div>
                    <div class="btns">
                        <button class="cbi-button cbi-button-remove app-btn app-back" type="button"
                            @click="onClose">{{ $gettext("关闭") }}</button>
                    </div>
                </div>
            </div>
        </transition>
    </action-component>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import request from '/@/request';
import ActionComponent from "/@/components/action/modal.vue"

const props = defineProps({
    Close: Function
})
const port = ref("")
const target = computed(() => {
    return `http://${location.hostname}:${port.value}`
})
const getData = () => {
    request.Nas.Linkease.Enable.POST().then(res => {
        if (res?.data?.result) {
            port.value = res.data.result?.port || ""
        }
    })
}
getData()
const onClose = () => {
    if (props.Close) {
        props.Close()
    }
    location.reload()
}
</script>
<style lang="scss" scoped>
.action {
    width: 700px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 1000;
    margin: auto;
    overflow: auto;
    padding: 1rem 87px;
    border-radius: 6px;

    .action-body {
        width: 100%;
        text-align: center;
        padding: 3rem 0;

        h2.title {
            width: 100%;
            display: block;
            color: #1e1e1e;
            font-size: 3em;
            padding: 0;
            margin: 0;
            text-align: center;
        }

        .info {
            color: #666;
            font-size: 1.3em;
            margin: 1rem 0;
        }

        .btns {
            width: 100%;
            margin-top: 3rem;

            button {
                display: block;
                width: 100% !important;
                margin: 0.5rem 0;
            }
        }
    }
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 1000px) {
    .action.format {

        .action-body {
            h2.title {
                font-size: 20px;
            }
        }
    }
}

@media screen and (max-width: 900px) {
    .action {

        .action-body {
            h2.title {
                font-size: 20px;
            }
        }

    }
}

@media screen and (max-width: 800px) {
    .action {
        .action-body {
            h2.title {
                font-size: 20px;
            }
        }
    }
}

@media screen and (max-width: 700px) {
    .action {

        .action-body {
            h2.title {
                font-size: 20px;
            }
        }
    }
}

@media screen and (max-width: 500px) {
    .action {

        .action-body {
            h2.title {
                font-size: 20px;
            }
        }
    }
}
</style>