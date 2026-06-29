<template>
    <action-component :Close="Close" :type="1">
        <div class="action">
            <div class="action-header">
                {{ title || $gettext("提示") }}
            </div>
            <div class="action-body" v-html="content">

            </div>
            <div class="action-footer">
                <div class="clear" @click="onClear" v-if="clear">{{ clearTitle || $gettext("返回") }}</div>
                <div class="next" @click="onNext">{{ nextTitle || $gettext("确定") }}</div>
                <div class="next" @click="onContinue" v-if="continuer">{{ continuerTitle || $gettext("继续保存") }}</div>

            </div>
        </div>
    </action-component>
</template>
<script setup lang="ts">
import { PropType } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()
import ActionComponent from "/@/components/action/modal.vue"

const props = defineProps({
    Close: Function,
    next: {
        type: Function as PropType<() => void>
    },
    clear: {
        type: Function as PropType<() => void>
    },
    continuer: {
        type: Function as PropType<() => void>
    },
    nextTitle: {
        type: String
    },
    clearTitle: {
        type: String
    },
    continuerTitle: {
        type: String
    },
    title: {
        type: String
    },
    content: {
        type: String
    },
})
const onNext = () => {
    if (props.next) {
        props.next()

    }
    if (props.Close) {
        props.Close()
    }
}
const onClear = () => {
    if (props.clear) {
        props.clear()
    }
    if (props.Close) {
        props.Close()
    }
}

const onContinue = () => {
    if (props.continuer) {
        props.continuer()
    }
    if (props.Close) {
        props.Close()
    }
}
</script>
<style lang="scss" scoped>
.action {
    width: 500px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 99999;
    margin: auto;
    border-radius: 4px;
    padding: 10px 0;

    .action-header {
        width: 100%;
        font-family: PingFangSC-Medium, PingFang SC;
        font-weight: 500;
        padding-left: 1rem;
        padding-right: 1rem;
        text-align: left;
        font-size: 18px;
        line-height: 1;
        color: #303133;
    }

    .action-body {
        display: block;
        margin: 2rem 0;
        line-height: 24px;
        padding: 0 15px;
        color: #606266;
        font-size: 14px;
    }

    .action-footer {
        width: 100%;
        height: 50px;
        border-top: 1px solid rgba(0, 0, 0, 0.06);
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: flex-end;
        padding: 0 30px;
    }


    .next {
        min-width: 65px;
        font-family: PingFangSC-Regular, PingFang SC;
        margin-left: 20px;
        line-height: 32px;
        text-align: center;
        cursor: pointer;
        font-size: 14px;
        font-family: PingFangSC-Regular, PingFang SC;
        font-weight: 400;
        color: #fff;
        margin-left: 20px;
        width: 120px;
        height: 32px;
        background: #553AFE;
        border-radius: 2px;

        &:hover {
            opacity: .8;
        }
    }


    .clear {
        min-width: 65px;
        font-family: PingFangSC-Regular, PingFang SC;
        font-weight: 400;
        line-height: 30px;
        text-align: center;
        cursor: pointer;
        height: 32px;
        border-radius: 2px;
        border: 1px solid rgba(0, 0, 0, 0.15);
        font-size: 14px;
        font-family: PingFangSC-Regular, PingFang SC;
        color: rgba(0, 0, 0, 0.83);
        line-height: 32px;

        &:hover {
            opacity: .8;
        }

    }
}
</style>
