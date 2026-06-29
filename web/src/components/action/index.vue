<template>
    <div id="actioner">
        <div class="bg" @click="close"></div>
        <template v-if="type != null">
            <!-- <transition name="dialog"> -->
            <slot></slot>
            <!-- </transition> -->
        </template>
        <template v-else>
            <div class="action-container" v-if="status">
                <div class="action-container_header">
                    <div></div>
                    <div class="title">{{ title }}</div>
                    <button class="close" :title="$gettext('关闭')" @click="close">
                        <CloseSvg></CloseSvg>
                    </button>
                </div>
                <div class="action-container_body">
                    <slot></slot>
                </div>
            </div>
        </template>
    </div>
</template>
<script setup lang="ts">
import CloseSvg from "/@/components/svg/close.vue"
import { onMounted, onUnmounted, PropType, ref } from 'vue'
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()
const props = defineProps({
    Close: {
        type: Function,
    },
    type: {
        type: Number as PropType<number>
    },
    title: String as PropType<string>
})
const status = ref(false)
onMounted(() => {
    status.value = true
    document.body.setAttribute('lock-scroll', "true")
})
onUnmounted(() => {
    document.body.removeAttribute('lock-scroll')
})
const close = () => {
    if (props.Close) {
        status.value = false
        setTimeout(() => {
            if (props.Close) {
                props.Close()
            }
        }, 300)
    }
}
</script>
<style lang="scss">
// 锁定滚动
[lock-scroll="true"] {
    overflow: hidden !important;
    height: 100vh;
}
</style>
<style lang="scss" scoped>
.bg {
    position: fixed;
    top: 0px;
    left: 0px;
    bottom: 0;
    right: 0;
    width: 100%;
    height: 100%;
    background: rgba(0, 0, 0, 0.5);
    z-index: 999;
}

#actioner {
    position: fixed;
    z-index: 1000;
    width: 100%;
    height: 100%;
    top: 0px;
    bottom: 0;
    left: 0;
    right: 0;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    overflow: auto;
}

#actioner {
    -webkit-box-sizing: border-box;
    -webkit-tap-highlight-color: transparent;
    box-sizing: border-box;
    word-wrap: break-word;
    outline: none;

    :deep(*) {
        -webkit-box-sizing: border-box;
        -webkit-tap-highlight-color: transparent;
        box-sizing: border-box;
        word-wrap: break-word;
        outline: none;
    }
}

.action-container {
    width: 100%;
    height: 100%;
    background-color: #fff;
    position: fixed;
    z-index: 9999;
    top: 0;
    bottom: 0;
    left: 0;
    right: 0;
    margin: auto;
    overflow: auto;

    .action-container_header {
        width: 100%;
        height: 36px;
        line-height: 36px;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: space-between;
        padding: 0 0.625rem;
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        // @extend %BgAnimation;
        border-bottom: 1px solid #1e1e1e;
        background-color: rgb(37, 37, 38);

        .title {
            color: #eee;
            font-size: 16px;
        }

        button.close {
            width: 36px;
            height: 36px;
            margin: 0;
            padding: 10px;
            background: none;
            border: none;
            cursor: pointer;
            opacity: 1;

            :deep(svg.icon) {
                width: 100%;
                height: 100%;

                path {
                    fill: #eee;
                }
            }

            &:hover {
                opacity: 0.9;
            }
        }
    }

    .action-container_body {
        width: 100%;
        height: 100%;
        padding-top: 36px;
    }
}
</style>
<style lang="scss">
/*缩放动画*/
@keyframes dialogEnter {
    from {
        transform: scale(0);
    }

    to {
        transform: scale(1);
    }
}

@keyframes dialogLeave {
    from {
        transform: scale(1);
    }

    to {
        transform: scale(0);
    }
}

.dialog-enter-active {
    animation: dialogEnter 0.3s linear forwards;
}

.dialog-leave-active {
    animation: dialogLeave 0.3s linear forwards;
}
</style>