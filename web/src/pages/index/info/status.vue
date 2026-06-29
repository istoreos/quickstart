<template>
    <ul class="app-container_docker">
        <li class="docker-item">
            <div class="docker-item_name">
                <span>{{ $gettext("当前状态:") }}</span>
            </div>
            <template v-if="props.docker?.status">
                <div class=" docker-item_value" v-if="!isInstall">
                    <span class="configure">{{ $gettext("未安装") }}</span>
                </div>
                <div class="docker-item_value" v-else>
                    <label class="input-switch">
                        <input type="checkbox" hidden :value="!data.enable" v-model="data.enable"
                            :disabled="data.disabled" @change="onChangeSwitchDocker">
                        <span :class="data.enable ? 'enable' : 'close'">
                            <em></em>
                        </span>
                    </label>
                    <span class="status-icon" v-if="data.enable">{{ $gettext("运行中") }}</span>
                    <span class="status-icon" style="background: #e9ebef;color: #4a5565;" v-if="!data.enable">{{ $gettext("未启用") }}</span>
                </div>
            </template>
        </li>
        <li class="content" v-if="docker?.status == 'running'">
            <div class="docker-item_name">
                <span :style="{color:'var(--app-container_title-color)'}">{{ $gettext("Docker根目录：") }}</span>
            </div>
            <div class="docker_box">
                <div class="path">{{ docker?.path }}</div>
                <span v-if="docker?.errorInfo">
                    <span class="tooltip-trigger">
                        <span class="docker_tip">
                            <HintSvg></HintSvg>
                        </span>
                        <div>
                            <div class="tooltip-text tooltip-top">
                                <span class="docker_dir_tip">{{ docker.errorInfo }}</span>
                            </div>
                        </div>
                    </span>
                </span>
            </div>
            <!-- <div class="docker_num">
                <div class="num_item">
                    <div>{{ $gettext("容器数量") }}</div>
                    <span style="color: #155dfc;">3</span>
                </div>
                <div class="num_item">
                    <div>{{ $gettext("镜像数量") }}</div>
                    <span style="color: #00a63e;">12</span>
                </div>
            </div> -->
        </li>
    </ul>
</template>
<script setup lang="ts">
import { log } from 'console';
import { computed, PropType, reactive, ref } from 'vue';
import { useGettext, formatNumber } from '/@/plugins/i18n'
const { $gettext, $ngettext } = useGettext()

import HintSvg from "/@/components/svg/hint.vue"
import request from '/@/request';
import Toast from '/@/components/toast';
const props = defineProps({
    docker: {
        type: Object as PropType<GuideDockerStatus>,
    }
})

const isInstall = computed(() => {
    return props.docker?.status != 'not installed'
})
const data = reactive({
    enable: props.docker?.status == "running",
    disabled: false
})
const onChangeSwitchDocker = async () => {
    data.disabled = true
    try {
        const res = await request.Guide.DockerSwitch.POST({
            enable: data.enable
        })
        if (res?.data) {
            const { success, error } = res.data
            if (error) {
                data.enable = !data.enable
                throw error
            }
            if ((success || 0) == 0) {
            }
        }
    } catch (error) {
        Toast.Warning(`${error}`)
    } finally {
        data.disabled = false
    }
}
</script>
<style lang="scss" scoped>
li.docker-item {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    margin: 1.5rem 0;

    .docker-item_name {
        flex: 0 0 100%;
        max-width: 50%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding-right: 10px;
        color: var(--app-container_title-color);
    }

    .docker-item_value {
        flex: 0 0 100%;
        max-width: 50%;
        padding-left: 10px;
        display: flex;
        justify-content: flex-end;
        align-items: center;

        .configure {
            color: #297ff3;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            padding: 3px;
            overflow-x: hidden;
            text-overflow: ellipsis;
        }

        .configure.enabel {
            color: #888;
            overflow-x: hidden;
            text-overflow: ellipsis;
        }
    }

    .docker-item_root {
        display: flex;
        justify-content: space-between;
        flex-wrap: wrap;
        margin-top: 16px;
        max-width: 323px;
        flex: 0 0 100%;
    }
}

.tooltip-trigger {
    position: relative;
    display: inline-block;
    cursor: help;
}

.tooltip-trigger .tooltip-text {
    visibility: hidden;
    position: absolute;
    padding: 0.5rem 1rem;
    /* tooltip 内间距 */
    background-color: #555;
    color: #fff;
    text-align: center;
    border-radius: 6px;
    z-index: 1;
    opacity: 0;
    transition: opacity 0.6s;
}

.tooltip-trigger .tooltip-text span {
    color: #fff;
}

.tooltip-trigger .tooltip-text .docker_dir_tip {
    min-width: 15rem;
    display: inline-block;
}

.tooltip-trigger:hover .tooltip-text {
    visibility: visible;
    opacity: 1;
}

.tooltip-top {
    bottom: 100%;
    left: 50%;
    margin-bottom: 5px;
    /* tooltip 与触发元素的距离 - 5px */
    transform: translate(-50%, 0);
    margin-left: 12px;
}

.tooltip-right {
    top: 50%;
    left: 100%;
    margin-left: 5px;
    /* tooltip 与触发元素的距离 - 5px */
    transform: translate(0, -50%);
}

.tooltip-left {
    top: 50%;
    right: 100%;
    margin-right: 5px;
    /* tooltip 与触发元素的距离 - 5px */
    transform: translate(0, -50%);
}

/* 角标 */
.tooltip-top::after {
    content: "";
    position: absolute;
    top: 100%;
    left: 50%;
    margin-left: -5px;
    border-width: 5px;
    border-style: solid;
    border-color: #555 transparent transparent transparent;
}

.tooltip-bottom::after {
    content: "";
    position: absolute;
    bottom: 100%;
    left: 50%;
    margin-left: -5px;
    border-width: 5px;
    border-style: solid;
    border-color: transparent transparent #555 transparent;
}



.input-switch {
    display: inline-block;
    cursor: pointer;
    position: relative;

    span {
        display: block;
        position: relative;
        width: 50px;
        height: 20px;
        border-radius: 10px;
        padding: 2px;

        em {
            display: block;
            width: 16px;
            height: 16px;
            background-color: #fff;
            border-radius: 10px;
        }
    }

    span.enable {
        background-color: #52c41a;
        transition: 0.3s;

        em {
            transform: translateX(30px);
            transition: 0.3s;
        }
    }

    span.close {
        background-color: #cecece;
        transition: 0.3s;

        em {
            transform: translateX(0px);
            transition: 0.3s;
        }
    }
}

.content {
    color: #333;
    margin-top: 20px;
    margin-bottom: 20px;
    font-weight: normal;

    .status {
        display: flex;
        justify-content: space-between;
        padding-bottom: 20px;
        border-bottom: 1px solid #e8e8e8;
        margin: 0 6px;
    }

    .docker_box {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin: 20px 6px;
        // margin-right: 0;

        .title {
            margin-bottom: 20px;
        }

        .path {
            flex: 1;
            border: 1px solid #e0e1e1;
            background: #f9fafb;
            border-radius: 4px;
            padding: 8px 10px;
        }
    }

    .docker_num {
        display: flex;

        .num_item {
            flex: 1;
            display: flex;
            justify-content: center;
            align-items: center;
            flex-direction: column;
            color: var(--app-container_title-color);

            >span {
                font-size: 20px;
                margin-top: 6px;
            }
        }
    }
}

.docker_tip svg {
    vertical-align: bottom;
    margin-left: 14px;
    width: 1.5em;
    height: 1.5em;
}

.status-icon {
    display: inline-block;
    margin-left: 10px;
    font-size: 12px;
    color: #008236;
    padding: 4px 6px;
    background: #dbfce7;
    border-radius: 6px;
}
</style>