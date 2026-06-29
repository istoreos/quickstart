<template>
    <action-component :Close="Close" :type="1">
        <transition name="rotate" mode="out-in">
            <div class="action" v-if="setup == 0">
                <h2 class="title">{{ $gettext("分区信息") }} - {{( disk.name || "?" ) + (disk.isSystemRoot?$gettext("（系统盘）"):"")}}</h2>
                <ul>
                    <li>
                        <div class="app-container_info">
                            <span>{{ $gettext("分区 / 挂载点") }}</span>
                            <span>{{ $gettext("容量") }}</span>
                        </div>

                        <div class="app-container_body">
                            <part-item v-for="(item, i) in disk.childrens" :key="i" :part="item" :disk="disk" />
                        </div>
                    </li>
                </ul>
                <div class="action-footer">
                    <div class="auto"></div>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onCancel"
                        type="button">{{ $gettext("返回") }}</button>
                </div>
            </div>
        </transition>
    </action-component>
</template>
<script setup lang="ts">
import { onMounted, PropType, ref, provide } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import ActionComponent from "/@/components/action/modal.vue"
import request from '/@/request';
import partItem from "./item.vue"
import Toast from "/@/components/toast";
const props = defineProps({
    disk: {
        type: Object as PropType<NasDiskModel>,
        required: true
    },
    Close: {
        type: Function as PropType<() => void>
    },
    Cancel: {
        type: Function as PropType<() => void>
    },
    Next: {
        type: Function as PropType<(rootPath: string) => void>
    }

})
const setup = ref(0)
const onClose = () => {
    if (props.Close) {
        props.Close()
    }
}
const onCancel = (e: Event) => {
    e.preventDefault()
    if (props.Cancel) {
        props.Cancel()
    }
    onClose()
}
</script>
<style lang="scss" scoped>
.action {
    width: 860px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 1000;
    margin: auto;
    padding: 3rem;
    border-radius: 6px;

    display: flex;
    flex-direction: column;
    flex-wrap: nowrap;

    ul {
        overflow: auto;

        .app-container_info {
            display: flex;
            justify-content: space-between;
            max-width: 56%;
            margin-top: 18px;
            font-weight: 600;
        }

        .app-container_body {
            width: 100%;
            height: 100%;
        }
    }

    .action-footer {
        text-align: center;
        margin-top: 46px;

        button {
            display: inline-block;
            width: 100px !important;
            margin: 0;
            margin-left: 1rem;
        }
    }
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 1000px) {
    .action {
        width: 160%;
    }
}

@media screen and (max-width: 800px) {
    .action {
        width: 138%;
    }
}

@media screen and (max-width: 700px) {
    .action {
        width: 132%;
    }
}

@media screen and (max-width: 600px) {
    .action {
        width: 116%;
    }
}

@media screen and (max-width: 500px) {
    .action {
        width: 100%;
    }
}

@media screen and (max-width:400px) {
    .action {
        width: 90%;
    }
}

@media screen and (max-width:300px) {
    .action {
        width: 100%;
    }
}
</style>
