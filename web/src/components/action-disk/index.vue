<template>
    <action-component :type="1" v-if="show">
        <transition name="rotate" mode="out-in">
            <!-- 选择硬盘 -->
            <div class="action list">
                <div class="action-header">
                    <div class="action-header_title">{{ $gettext("请选择一个硬盘或分区") }}</div>
                </div>
                <div class="action-body">
                    <div class="disk-list">
                        <item-vue v-for="item in disk.disks" :disk="item" :onDisk="onDisk" :currDisk="currDisk"
                            :currMountPoint="currMountPoint" />

                        <item-vue v-for="item in disk.raids" :disk="item" :onDisk="onDisk" :currDisk="currDisk"
                            :currMountPoint="currMountPoint" />
                    </div>
                </div>
                <div class="action-msg">
                    <span>
                        {{ $gettext("想要更精确的配置？请前往") }}
                        <a href="/cgi-bin/luci/admin/system/diskman">{{ $gettext("高级设置") }}</a>
                    </span>
                </div>
                <div class="action-footer">
                    <div class="auto"></div>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onBack"
                        type="button">{{ $gettext("返回") }}</button>
                    <button class="cbi-button cbi-button-apply app-btn app-next" @click="onSelect"
                        type="button">{{ $gettext("下一步") }}</button>
                </div>
            </div>
        </transition>
    </action-component>
</template>
<script setup lang="ts">
import { onMounted, PropType, ref, provide, reactive } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import ActionComponent from "/@/components/action/modal.vue"
import request from "/@/request";
import itemVue from "./item.vue";
import Toast from "/@/components/toast";
import ActionDiskFormat from "/@/components/action-disk-format";
const props = defineProps({
    Cancel: {
        type: Function as PropType<() => void>
    },
    Next: {
        type: Function as PropType<(rootPath: string) => void>
    },
    Close: {
        type: Function as PropType<() => void>
    }
})
const show = ref(true)
const disk = reactive({
    disks: [] as NasDiskModel[],
    raids: [] as NasDiskModel[],
})
const getData = async () => {
    const resp = await Promise.all([
        request.Nas.Disk.Status.GET(),
        request.Raid.List.GET()
    ])
    try {
        if (resp[0]) {
            const res = resp[0]
            if (res?.data.result) {
                disk.disks = res?.data.result.disks || []
            }
        }
        if (resp[1]) {
            const res = resp[1]
            if (res.data.result) {
                disk.raids = res.data.result.disks || []
            }
        }
    } catch (error) {
        Toast.Warning(error as string)
    }
}
getData()
// 选中的设备
const currDisk = ref<NasDiskModel | null>()
const currMountPoint = ref<MountPoint | null>()
const onDisk = (_disk: NasDiskModel, _mount: MountPoint | null) => {
    currDisk.value = _disk
    currMountPoint.value = _mount
}
const onClose = () => {
    if (props.Close) {
        props.Close()
    }
}
const onBack = () => {
    if (props.Cancel) {
        props.Cancel()
    }
    onClose()
}
const onNext = (rootPath: string) => {
    if (props.Next) {
        props.Next(rootPath)
    }
    onClose()
}
const onSelect = () => {
    if (currDisk.value == null) {
        Toast.Warning($gettext("请选择目标硬盘"))
        return
    }
    if (currDisk.value.childrens != null && currDisk.value.childrens.length > 0) {
        if (currMountPoint.value == null) {
            Toast.Warning($gettext("请选择硬盘分区"))
            return
        }
    }
    if (currMountPoint.value != null) {
        // 选中了尚未挂载的分区
        if (currMountPoint.value.mountPoint == null || currMountPoint.value.mountPoint == "") {
            Toast.Warning($gettext("该分区尚未挂载，请先去挂载"))
            return
        }
    }
    show.value = false
    ActionDiskFormat({
        action: "nas",
        disk: currDisk.value,
        mount: currMountPoint.value,
        Cancel: () => {
            show.value = true
        },
        Next: (rootPath) => {
            onNext(rootPath)
        }
    })
}
</script>
<style lang="scss" scoped>
.action {
    .action-footer {
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
.action.list {
    width: 700px;
    height: 560px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 1000;
    margin: auto;
    overflow: auto;
    padding: 0 25px;

    border: 1px solid #dfdfdf;
    border-radius: 4px;
    background: #fff;
    box-shadow: 0 1px 4px rgb(0 0 0 / 30%);

    .action-header {
        width: 100%;
        height: 70px;
        line-height: 70px;

        .action-header_title {
            margin: 0;
            color: #333;
            font: inherit;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            -moz-user-select: none;
            -webkit-user-select: none;
            user-select: none;
            font-size: 20px;
        }
    }

    .action-body {
        width: 100%;
        height: calc(100% - 176px);
    }

    .action-msg {
        width: 100%;
        height: 36px;
        line-height: 36px;
        text-align: center;
    }

    .action-footer {
        width: 100%;
        height: 70px;
        line-height: 70px;
        color: #333;
        display: flex;
        flex-wrap: wrap;
        align-items: center;

        .auto {
            flex: auto;
        }
    }

    .disk-list {
        width: 100%;
        height: 100%;
        border: 1px solid #dfe1e5;
        overflow: auto;
    }
}
</style>
<style lang="scss" scoped>
.action.format {
    width: 700px;
    height: 560px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 1000;
    margin: auto;
    overflow: auto;
    padding: 0 25px;
    border: 1px solid #dfdfdf;
    border-radius: 4px;
    background: #fff;
    box-shadow: 0 1px 4px rgb(0 0 0 / 30%);

    .action-header {
        width: 100%;
        height: 70px;
        line-height: 70px;

        .action-header_title {
            margin: 0;
            color: #333;
            font: inherit;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            -moz-user-select: none;
            -webkit-user-select: none;
            user-select: none;
            font-size: 20px;
        }
    }

    .action-body {
        width: 100%;
        height: calc(100% - 140px);
        overflow: auto;
    }

    .action-footer {
        width: 100%;
        height: 70px;
        line-height: 70px;
        color: #333;
        display: flex;
        flex-wrap: wrap;
        align-items: center;

        .auto {
            flex: auto;
        }
    }

    .disk-list {
        width: 100%;
        height: 100%;
        border: 1px solid #dfe1e5;
        overflow: auto;
    }

    .label-item {
        width: 100%;
        margin: 1rem 0;

        .label-item_key {
            width: 100%;
            font-size: 16px;
            color: #666;

            span {
                white-space: nowrap;
                overflow: hidden;
                text-overflow: ellipsis;
            }

            span:before {
                content: "*";
                color: #f56c6c;
                margin-right: 4px;
            }
        }

        .label-item_value {
            width: 100%;
            margin-top: 5px;

            select,
            input {
                width: 100%;
                height: 36px;
            }
        }
    }

    .auto {
        flex: auto;
    }

    p.msg {
        margin: 0.5rem 0;
        color: red;
    }

    .disk-info {
        width: 100%;
        text-align: center;

        .disk-info_icon {
            width: 100px;
            height: 100px;
            margin: 0 auto;

            svg {
                width: 100%;
                height: 100%;
            }
        }

        .disk-info_mount-name {
            margin: 1rem 0;
            font-size: 1.5em;
            color: #333;
        }
    }
}
</style>
<style lang="scss" scoped>
.action.result {
    width: 700px;
    height: 560px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 1000;
    margin: auto;
    overflow: auto;
    padding: 0 25px;
    border: 1px solid #dfdfdf;
    border-radius: 4px;
    background: #fff;
    box-shadow: 0 1px 4px rgb(0 0 0 / 30%);

    .action-header {
        width: 100%;
        height: 70px;
        line-height: 70px;

        .action-header_title {
            margin: 0;
            color: #333;
            font: inherit;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            -moz-user-select: none;
            -webkit-user-select: none;
            user-select: none;
            font-size: 20px;
        }
    }

    .action-body {
        width: 100%;
        height: calc(100% - 140px);
        overflow: auto;

        .format-result {
            width: 100%;
            text-align: center;
            font-size: 2em;
            color: #333;
            margin: 1rem 0;
        }

        .format-info {
            width: 100%;
            text-align: center;
            font-size: 1.3em;

            a {
                color: #f70324;
            }
        }
    }

    .action-footer {
        width: 100%;
        height: 70px;
        line-height: 70px;
        color: #333;
    }

    .auto {
        flex: auto;
    }
}
</style>
<style lang="scss" scoped>
@media screen and (max-width: 1000px) {
    .action.list {
        width: 136%;
    }
}

@media screen and (max-width: 900px) {
    .action.list {
        width: 126%;
    }
}

@media screen and (max-width: 800px) {
    .action.list {
        width: 112%;
    }
}

@media screen and (max-width: 700px) {
    .action.list {
        width: 100%;
    }
}

// @media screen and (max-width: 600px){
//     .action{
//         width: 116%;
//     }
// }
@media screen and (max-width: 500px) {
    .action.list {
        width: 80%;
    }
}
</style>