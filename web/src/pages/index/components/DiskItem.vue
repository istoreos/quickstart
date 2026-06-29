<template>
    <!-- 错误内容 -->
    <li class="disk-item error" v-if="status == 'error'">
        <div class="disk-item_name" :title="diskName">
            <span>{{ diskName }} </span>
        </div>
        <div class="disk-item_value">
            <div class="value-data">
                <span class="error">{{ disk.errorInfo }}</span>
            </div>
        </div>
    </li>
    <!-- 成功内容 -->
    <li class="disk-item" v-else-if="status == 'success'">
        <div class="disk-item_name" :title="diskName">
            <span>{{ diskName }} </span>
        </div>
        <template v-if="disk.smartWarning && smartWarning">
            <div class="disk_value">
                <div class="value-data">
                    <a href="/cgi-bin/luci/admin/nas/smart">
                        <span class="error"> {{ $gettext("S.M.A.R.T异常") }}</span>
                    </a>
                </div>
            </div>
        </template>
        <template v-else>
            <div class="disk_value">
                <div class="disk-item_value">
                    <div class="value-data">
                        <ProgressBar :percentage="disk.usage || 0" :showPercentage="false" height="10px"
                            borderRadius="10px" :color="getColor(disk.usage || 0)" backgroundColor="#f4f5f7" />
                        <div>
                            <span>{{ $gettext("使用率") }}：{{ disk.usage || 0 }}%</span>
                            <span>{{ $gettext("已使用") }}：{{ disk.used }}</span>
                        </div>
                    </div>

                    <div class="disk-item-tooltip">
                        <span>{{ $gettext("仅统计已挂载分区") }}</span>
                    </div>
                </div>
                <div class="disk_icon">
                    <span class="tooltip-trigger"
                        v-if="disk.isDockerRoot && disk.isSystemRoot && disk.usage && disk.usage >= 90">
                        <span class="disk_tip">
                            <HintSvg></HintSvg>
                        </span>
                        <div>
                            <div class="tooltip-text tooltip-top">
                                <span class="disk_dir_tip">{{
                                    $gettext("您的系统空间已不足，检测到您的Docker根目录位于系统根目录上，可能会影响系统的正常运行，建议使用Docker迁移向导将Docker根目录迁移到外置硬盘上。")
                                    }}</span>
                            </div>
                        </div>
                    </span>
                    <span class="tooltip-trigger" v-if="diskFormatTips">
                        <span class="disk_tip">
                            <HintSvg></HintSvg>
                        </span>
                        <div>
                            <div class="tooltip-text tooltip-top">
                                <span class="disk_dir_tip">{{ $gettext("分区存在异常，点击分区列表查看错误") }}</span>
                            </div>
                        </div>
                    </span>
                    <span class="disk_infoicon" v-if="diskFormatTips && (disk.childrens?.length || 0) == 1"
                        @click="onFormat()">
                        <DiskformattingSvg></DiskformattingSvg>
                    </span>
                    <span class="disk_infoicon" v-if="(disk.childrens?.length || 0) > 0" @click="onInfo()">
                        <DiskinfoSvg style="color: var(--app-container_title-color)" ></DiskinfoSvg>
                    </span>
                </div>
            </div>
        </template>

    </li>

    <!-- 格式化并挂载内容 -->
    <li class="disk-item load" v-else-if="status == 'load'">
        <div class="disk-item_name" :title="diskName">
            <span>{{ diskName }} </span>
        </div>
        <div class="disk_value">
            <div class="disk-item_value">
                <div class="value-data">
                    <button @click="onFormat()">{{ $gettext("格式化并挂载") }}</button>
                </div>
            </div>
        </div>
    </li>

    <!-- 未挂载内容 -->
    <li class="disk-item load" v-else-if="status == 'unmounted'">
        <div class="disk-item_name" :title="diskName">
            <span>{{ diskName }}</span>
        </div>
        <div class="disk_value">
            <div class="disk-item_value" v-if="(disk.childrens?.length || 0) == 1">
                <div class="value-data">
                    <button @click="onInfo()" v-if="'swap' == (disk.childrens?.[0]?.filesystem)">{{ $gettext("查看详情")
                        }}</button>
                    <button @click="onMount()" v-else>{{ $gettext("手动挂载") }}</button>
                </div>
            </div>

            <div class="disk_icon">
                <span class="disk_infoicon" @click="onInfo()" v-if="(disk.childrens?.length || 0) > 1">
                    <DiskinfoSvg style="color: var(--app-container_title-color)" ></DiskinfoSvg>
                </span>
            </div>
        </div>
    </li>
</template>
<script setup lang="ts">
import { computed, PropType } from 'vue';
import { useGettext, formatNumber } from '/@/plugins/i18n'
const { $gettext } = useGettext()

import HintSvg from "/@/components/svg/hint.vue"
import DiskinfoSvg from "/@/components/svg/diskinfo.vue"
import DiskformattingSvg from "/@/components/svg/diskformatting.vue"
import ActionDiskFormat from '/@/components/action-disk-format';
import ActionDiskPart from '/@/components/action-disk-part';
import ActionDiskMount from '/@/components/action-disk-mount';
import ProgressBar from "/@/components/ProgressBar/index.vue"

const props = defineProps({
    disk: {
        type: Object as PropType<NasDiskModel>,
        required: true
    },
    smartWarning: { // 是现实有S.M.A.R.T异常
        type: Boolean as PropType<boolean>
    }
})

const status = computed(() => {
    // return "load"
    if (props.disk.errorInfo) {
        return "error"
    }
    // 避免格式化系统盘
    if (props.disk.isSystemRoot) {
        return "success"
    }

    // 新硬盘，未分区
    if (props.disk.childrens == null || props.disk.childrens.length == 0 || (props.disk.childrens.length == 1 && 'No FileSystem' == props.disk.childrens[0].filesystem)) {
        return "load"
    }
    // 未挂载
    if (0 == props.disk.childrens.filter(e => e.mountPoint).length) {
        return "unmounted"
    }
    // let count = props.disk.childrens.length
    // if (count > 0) {
    //     for (let i = 0; i < count; i++) {
    //         let item = props.disk.childrens[i]
    //         if (item.mountPoint == null || item.mountPoint == "") {
    //             return "load"
    //         }
    //     }
    // }
    return "success"
})
const diskName = computed(() => {
    const disk = props.disk
    let name = disk.name
    if (disk.size) {
        name += `【${disk.size}】`
    }
    if (disk.venderModel) {
        name += `(${disk.venderModel})`
    }
    return name
})
const diskFormatTips = computed(() => {
    const disk = props.disk
    return !disk.isSystemRoot && (disk.childrens?.filter(e => (e.isReadOnly && 'swap' != e.filesystem)).length || 0) > 0
})

const onFormat = () => {
    ActionDiskFormat({
        action: "disk",
        disk: props.disk,
        Cancel: () => {
        },
        Next: () => {
            location.reload();
        }
    })
}

const onInfo = () => {
    ActionDiskPart({
        action: "disk",
        disk: props.disk,
        Cancel: () => {
        },
        Next: () => {
            location.reload();
        }
    })
}
const onMount = () => {
    const _dksk = props.disk
    const _mount = _dksk.childrens || []
    ActionDiskMount({
        action: "nas",
        disk: _dksk,
        mount: _mount[0],
        Cancel: () => {
        },
        Next: () => {
            location.reload();
        }
    })
}

const getColor = (value: number) => {
    if (value < 50) return "#2fc867"; // Green
    if (value > 50 && value <= 75) return "#f97316"; // Orange
    return "#dc2626"; // Red
}
</script>
<style lang="scss" scoped>
li.disk-item.error {
    color: red;
}

li.disk-item {
    width: 100%;
    // display: flex;
    margin: 0 0 1rem;
    // align-items: center;

    .disk-item_name {
        flex: 0 0 100%;
        max-width: 50%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding-right: 10px;

        >span {
            color: #6a7280;
        }
    }

    .disk_icon {
        padding-left: 1rem;
        align-self: center;
        align-items: center;
        flex: none;
        display: flex;
    }

    .disk_value {
        display: flex;
        justify-content: flex-end;

        .disk-item_value {
            flex: auto;
            position: relative;
            cursor: help;
            display: flex;
            align-items: center;

            .value-data {
                width: 100%;
                text-overflow: ellipsis;
                white-space: nowrap;

                >div {
                    margin-top: 10px;
                    display: flex;
                    justify-content: space-between;
                }

                button {
                    background: none;
                    border: none;
                    width: 100%;
                    text-align: right;
                    color: #297ff3;
                    cursor: pointer;
                    padding: 0;
                    margin: 0;
                    line-height: normal;

                    &:hover {
                        opacity: 0.7;
                    }
                }
            }

            .disk-item-tooltip {
                position: absolute;
                background: rgba(0, 0, 0, 0.7);
                z-index: 10111;
                color: #fff;
                padding: 0.5rem 1rem;
                left: 30%;
                right: 30%;
                bottom: 100%;
                margin-bottom: 6px;
                text-align: center;
                font-size: 1em;
                visibility: hidden;
                opacity: 0;

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

            &:hover {
                .disk-item-tooltip {
                    visibility: visible;
                    transition: 0.7s;
                    opacity: 1;
                }
            }
        }
    }
}

.disk_infoicon {
    margin-left: 10px;
    cursor: pointer;
    margin-bottom: 10px;
}

.tooltip-trigger {
    flex: none;
    cursor: help;
}

.tooltip-trigger {
    position: relative;
    display: inline-block;
    cursor: help;
    margin-right: 6px;
    margin-left: 10px;
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

.tooltip-trigger .tooltip-text .disk_dir_tip {
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
</style>