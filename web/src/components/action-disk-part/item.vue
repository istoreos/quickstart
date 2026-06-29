<template>
    <div class="disk-content">
        <li class="disk-item">
            <div class="disk-item_name">
                <span v-if="isFreeSpace">{{ $gettext("未分区") }}</span>
                <span v-else>{{part.name}}{{ part.mountPoint ? "" : (noFileSystem ? $gettext("（未格式化）") : $gettext("（未挂载）"))}}</span>
                <span v-if="part.isSystemRoot">{{ $gettext("（系统分区）") }}</span>
            </div>
            <div class="disk_value">
                <div class="disk-item_value">
                    <div class="value-data">
                        <progress-item :value="isFreeSpace || !part.usage ? 0 : part.usage"
                            :text="isFreeSpace ? $gettext('未分区（%{total}）', {total:part.total||''}) : ( (part.mountPoint && 'swap' != part.filesystem ? part.used : $gettext('未知')) + '/' + (part.total || ''))" :style="{
                                backgroundColor: '#767676',
                            }" />
                    </div>
                </div>
                <button class="cbi-button cbi-button-apply" @click="onInitRest" v-if="isFreeSpace">{{ $gettext("分区并格式化") }}</button>
                <button class="cbi-button cbi-button-apply" v-else-if="shouldFormat" @click="onFormat">{{ $gettext("格式化分区") }}</button>
            </div>
        </li>
        <li class="disk-item" v-if="!isFreeSpace && !noFileSystem">
            <span class="disk-item_name">
                <template v-if="part.mountPoint">
                    <span v-if="'swap' == part.filesystem">{{ $gettext("已挂载为交换区") }}</span>
                    <a :href="mountUrl" target="_blank" v-else >{{part.mountPoint}}</a>
                </template>
                <template v-else>
                    <span v-if="'swap' == part.filesystem">{{ $gettext("不支持挂载") }}</span>
                    <span class="value-data buttondiv" v-else @click="onMount">{{ $gettext("手动挂载") }}</span>
                </template>
            </span>
            <div class="disk_status">
                <div class="disk_status_item" v-if="part.mountPoint && 'swap' != part.filesystem">
                    <div>{{ $gettext("可读写状态：") }}{{ part.isReadOnly ? $gettext("只读") : $gettext("读写") }}</div>
                    <div class="tooltip-trigger disk_tip" v-if="readOnlyFileSystem">
                        <HintSvg></HintSvg>
                        <div class="tooltip-text tooltip-top">
                            <div class="disk_dir_tip">{{ $gettext("此分区为只读状态，可能无法写入数据") }}</div>
                        </div>
                    </div>
                </div>
                <div class="disk_status_item">
                    <div>{{ $gettext("文件系统：") }}{{ part.filesystem?.toUpperCase() }}</div>
                    <div class="tooltip-trigger disk_tip" v-if="!part.isSystemRoot && limitedFileSystem">
                        <HintSvg></HintSvg>
                        <div class="tooltip-text tooltip-top">
                            <span class="disk_dir_tip">{{ $gettext("此文件系统不支持Docker等应用数据，建议格式化成EXT4文件系统") }}</span>
                        </div>
                    </div>
                </div>
            </div>
        </li>
    </div>
</template>
<script setup lang="ts">
import { computed, PropType } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import HintSvg from "/@/components/svg/hint.vue"
import DiskinfoSvg from "/@/components/svg/diskinfo.vue"
import DiskformattingSvg from "/@/components/svg/diskformatting.vue"
import ActionDiskFormat from "/@/components/action-disk-format";
import ActionDiskMount from '/@/components/action-disk-mount';
import Toast from "/@/components/toast";
import request from "/@/request";
const props = defineProps({
    part: {
        type: Object as PropType<MountPoint>,
        required: true
    },
    disk: {
        type: Object as PropType<NasDiskModel>,
        required: true
    },
})

const noFileSystem = computed(() => 'No FileSystem' == props.part.filesystem)
const limitedFileSystem = computed(() => props.part.filesystem && ['ntfs','vfat','exfat'].indexOf(props.part.filesystem) >= 0)
const readOnlyFileSystem = computed(() => props.part.mountPoint && props.part.isReadOnly && 'swap' != props.part.filesystem)
const shouldFormat = computed(() => noFileSystem.value ||
    (!props.part.isSystemRoot &&
        (readOnlyFileSystem.value || limitedFileSystem.value || (!props.part.mountPoint && 'swap' == props.part.filesystem))
    ))

const onFormat = function () {
    ActionDiskFormat({
        action: "disk",
        disk: props.disk,
        mount: props.part,
        Cancel: () => {
        },
        Next: (rootPath) => {
            location.reload()
        }
    })
}
const onMount = () => {
    ActionDiskMount({
        action: "nas",
        disk: props.disk,
        mount: props.part,
        Cancel: () => {
        },
        Next: () => {
            location.reload();
        }
    })
}
const onInitRest = async () => {
    const load = Toast.Loading($gettext("处理中..."))
    try {
        const res = await request.Nas.Disk.InitRest.POST({
            name: props.disk.name,
            path: props.disk.path
        })
        if (res?.data) {
            const { result, error } = res?.data
            if (error) {
                Toast.Warning(error)
            }
            if (result) {
                Toast.Success($gettext("挂载成功"))
                location.reload()
            }
        }
    } catch (error) {
        Toast.Error(error as string)
    }
    load.Close()
}
const isFreeSpace = computed(() => props.part.filesystem == 'Free Space')
const mountUrl = computed(() => {
    const mountPoint = (props.part.mountPoint?props.part.mountPoint:"")
    if (mountPoint.indexOf("/mnt/") == 0) {
        return "/cgi-bin/luci/admin/services/linkease/file/?path=/" + mountPoint.substring(5)
    } else {
        return "/cgi-bin/luci/admin/services/linkease/file/?path=/root" + mountPoint
    }
})
</script>
<style lang="scss" scoped>
li.disk-item.error {
    color: red;
}

.disk-content {
    padding: 1rem;
    border: 1px solid #cfcfcf;
    margin: 16px 0;

    li.disk-item {
        width: 100%;
        display: flex;
        // margin: 1rem 0;
        align-items: center;

        .disk-item_name {
            flex: 0 0 50%;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            padding-right: 10px;
        }

        .value-data {
            width: 100%;
            text-overflow: ellipsis;
            white-space: nowrap;
            height: 100%;
            color: #297ff3;
            cursor: default;

            button {
                background: none;
                border: none;
                width: 100%;
                text-align: right;
                color: #297ff3;
                cursor: pointer;

                &:hover {
                    opacity: 0.7;
                }
            }
            &.buttondiv {
                cursor: pointer;
            }
        }

        .disk_value {
            flex: 0 0 50%;
            display: flex;
            justify-content: space-between;
            align-items: center;

            .cbi-button {
                margin-left: 10px;
            }

            .disk-item_value {
                flex: auto;
                padding-left: 10px;
                position: relative;

                .disk-item-tooltip {
                    position: absolute;
                    background: rgba(0, 0, 0, 0.7);
                    z-index: 10111;
                    color: #fff;
                    padding: 0.5rem 1rem;
                    left: 10px;
                    right: 0;
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

    .disk_status {
        display: flex;
        text-align: left;
        padding-left: 10px;
        font-size: 12px;
        padding-top: 6px;

        .disk_status_item {
            display: flex;
            margin-right: 20px;

            .disk_tip {
                display: flex;
                align-items: center;
            }
        }
    }
}

.disk_infoicon {
    margin-left: 10px;
    cursor: pointer;
}

.tooltip-trigger {
    flex: none;
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
<style lang="scss" scoped>
@media screen and (max-width: 1000px) {
    .disk-content {
        li.disk-item {
            .disk_value {
                display: block;
            }
        }
        .disk_status {
            flex-wrap: wrap;
        }
    }

}

</style>
