<template>
    <action-component :Close="Close" :type="1">
        <div class="actioner-ttyd">
            <div class="actioner-ttyd_header">
                <li
                    class="tab-item"
                    :class="{
                        on: i == onTab
                    }"
                    v-for="(item,i) in tabs"
                    :key="i"
                    @click="currTab(i)"
                >
                    <span>{{ $gettext("窗口 %{ item }", {item}) }}</span>
                    <button class="clear" @click.prevent.stop="clearTab(i)">
                        <close-svg />
                    </button>
                </li>
                <li class="tab-item" @click="currAdd()" v-if="tabs.length < 3">+</li>

                <div class="auto"></div>
                <button class="close" @click="Close()">
                    <close-svg />
                </button>
            </div>
            <div class="actioner-ttyd_body">
                <template v-for="(item,i) in tabs" :key="i">
                    <keep-alive>
                        <ttyd v-show="i == onTab" />
                    </keep-alive>
                </template>
            </div>
        </div>
    </action-component>
</template>
<script setup lang="ts">
import { ref } from "vue"
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import CloseSvg from "/@/components/svg/close.vue"
import ActionComponent from "/@/components/action/modal.vue"
import Ttyd from "./ttyd.vue"
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
})
const tabs = ref<string[]>(["0"])
const onTab = ref(0)
const currTab = (i: number) => {
    onTab.value = i
}
const clearTab = (i: number) => {
    tabs.value.splice(i, 1)
    const index = tabs.value.length - 1
    if (index >= 0) {
        onTab.value = index
    }
    if (tabs.value.length == 0) {
        props.Close()
    }
}
const currAdd = () => {
    const index = tabs.value.length
    tabs.value.push(`${index}`)
    onTab.value = index
}
</script>
<style lang="scss" scoped>
.actioner-ttyd {
    width: 100%;
    height: 100%;
    background-color: #2b2b2b;
    position: relative;
    z-index: 99999;
    margin: auto;
    overflow: auto;
    .actioner-ttyd_header {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        height: 36px;
        border-bottom: 1px solid #1e1e1e;
        background-color: rgb(37, 37, 38);
        display: flex;
        flex-wrap: nowrap;
        align-items: center;
        overflow-y: hidden;
        overflow-x: auto;
        li.tab-item:hover {
            opacity: 0.9;
        }
        li.tab-item {
            flex: 0 0 100%;
            max-width: 161px;
            height: 100%;
            border-right: 1px solid #252526;
            background-color: #2d2d2d;
            color: #eee;
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            justify-content: center;
            font-size: 13px;
            font-weight: 400;
            cursor: pointer;
            position: relative;
            span {
                display: inline-block;
                max-width: 130px;
                flex: 0 0 100%;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
                padding-left: 1rem;
            }
            button.clear {
                display: inline-block;
                width: 30px;
                height: 100%;
                background: none;
                border: none;
                margin: 0;
                padding: 0;
                display: flex;
                flex-wrap: wrap;
                align-items: center;
                justify-content: center;
                cursor: pointer;
                &:hover {
                    background-color: rgba(90, 93, 94, 0.31);
                }
                :deep(svg) {
                    width: 10px;
                    height: 10px;
                    path {
                        fill: rgba(255, 255, 255, 0.5);
                    }
                }
            }
        }
        li.tab-item.on {
            border-right: 1px solid rgb(37, 37, 38);
            background-color: #363636;
        }
        .auto {
            flex: auto;
        }
        button.close {
            display: inline-block;
            width: 50px;
            height: 100%;
            background: none;
            border: none;
            margin: 0;
            padding: 0;
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            &:hover {
                background-color: rgba(90, 93, 94, 0.31);
            }
            :deep(svg) {
                width: 10px;
                height: 10px;
                path {
                    fill: #eee;
                }
            }
        }
    }
    .actioner-ttyd_body {
        width: 100%;
        height: calc(100% - 36px);
        padding: 0.5rem 0;
    }
}
</style>