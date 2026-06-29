<template>
    <action-component :type="2">
        <div class="action-main">
            <setup-create :Close="Close" v-if="setup == 'create'" />
            <setup-info :Close="Close" v-else-if="setup == 'info' && raid != null" :raid="raid" :success="success" />
            <setup-edit :Close="Close" v-else-if="setup == 'edit' && raid != null" :raid="raid" :success="success" />
            <setup-remove :Close="Close" v-else-if="setup == 'remove' && raid != null" :raid="raid"
                :success="success" />
            <setup-recover :Close="Close" v-else-if="setup == 'recover' && raid != null" :raid="raid"
                :success="success" />
        </div>
    </action-component>
</template>
<script setup lang="ts">
import { PropType, reactive, ref } from "vue";
import ActionComponent from "/@/components/action/modal.vue"
import setupCreate from "./setup-create.vue"
import setupInfo from "./setup-info.vue"
import setupEdit from "./setup-edit.vue"
import setupRemove from "./setup-remove.vue"
import setupRecover from "./setup-recover.vue"
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
    setup: {
        type: String as PropType<RaidSetupType>,
        default: "create"
    },
    raid: {
        type: Object as PropType<Disksinfo>,
    },
    success: {
        type: Function as PropType<ActionRaidSuccess>
    }
})
</script>
<style lang="scss" scoped>
.action-main {
    width: 680px;
    background-color: #fff;
    position: relative;
    z-index: 99999;
    margin: auto;
    overflow: auto;

    :deep(.actioner-container) {
        width: 100%;

        .actioner-container_header {
            width: 100%;
            height: 50px;
            line-height: 50px;
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            font-size: 20px;
            border-bottom: 1px solid #eee;
            justify-content: center;
            padding: 0 10px;
        }

        .actioner-container_footer {
            width: 100%;
            height: 50px;
            border-top: 1px solid rgba(0, 0, 0, 0.06);
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            justify-content: flex-end;
            padding: 0 30px;

            button {
                display: inline-block;
                width: 100px !important;
                margin: 0;
                margin-left: 1rem;
            }

            .close {
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
                width: 74px;
                height: 32px;
                background: #553AFE;
                border-radius: 2px;

                &.save {
                    height: 32px;
                    background: #553AFE;
                    border-radius: 2px;
                    line-height: 16px;
                }
            }

            .next:hover,
            .close:hover {
                opacity: 0.9;
            }
        }

        .actioner-container_body {
            padding: 20px;
            width: 100%;
            height: 400px;
        }
    }
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 800px) {
    .action-main {
        width: 90%;
    }
}
</style>