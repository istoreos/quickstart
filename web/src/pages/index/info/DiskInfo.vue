<template>
    <Card :title="$gettext('磁盘信息')" style="width: 100%; display: block;" :is-settings-menu-open="isSettingsMenuOpen" @update:isSettingsMenuOpen="(val: boolean) => isSettingsMenuOpen = val">
        <template #icon>
            <diskIcon color="#45556c" class="icon" />
        </template>
        <template #settings>
            <div class="btn_settings" @click="onHandleOpenFile">
                <folderIcon color="#0a0a0a" class="icon1 interfaceIcon" style="margin-right: 6px;" />
                <span>{{ $gettext('文件管理') }}</span>
                <div class="rotation" @click.stop="isSettingsMenuOpen = !isSettingsMenuOpen">
                    <moreItem class="moreIcon" />
                </div>
            </div>
        </template>
        <template #settings-menu>
            <div><a href="/cgi-bin/luci/admin/nas/raid">{{ $gettext("RAID管理") }}</a></div>
            <div><a href="/cgi-bin/luci/admin/nas/smart">S.M.A.R.T.</a></div>
            <div><a href="/cgi-bin/luci/admin/system/diskman">{{ $gettext("磁盘管理") }}</a></div>
            <div><a href="/cgi-bin/luci/admin/system/mounts">{{ $gettext("挂载点") }}</a></div>
        </template>
        <div class="content">
            <div class="disk_loading_icon" v-if="!disk.disks && !disk.raidList">
                <icon-loading :size="38" color="#888888" />
                <span class="disk_loading_info">{{ $gettext("正在获取磁盘信息...") }}</span>
            </div>
            <template v-if="disk.disks">
                <div class="item" style="margin-top: 4px;padding-bottom: 0;">
                    <div class="icon_box">
                        <diskIcon color="#2b6cfc" class="icon" />
                    </div>
                    <div class="info">
                        <div class="name">
                            <div>{{ $gettext("系统根目录") }}</div>
                            <!-- <span>
                                <successIcon class="icon" />正常
                            </span> -->
                        </div>
                        <div class="schedule">
                            <disk-item v-for="(item, i) in disk.disks?.filter(e => e.isSystemRoot)" :key="i"
                                :disk="item" />
                        </div>
                    </div>
                </div>
                <div class="line" v-if="disk.disks?.filter(e => !e.isSystemRoot).length>0"></div>
                <div class="item" v-if="disk.disks?.filter(e => !e.isSystemRoot).length>0">
                    <div class="icon_box" style="background: #f3e8ff;">
                        <financeIcon class="icon" />
                    </div>
                    <div class="info">
                        <div class="name">
                            <div>{{ $gettext("已挂载磁盘") }}</div>
                            <!-- <span>
                                <successIcon class="icon" />正常
                            </span> -->
                        </div>
                        <div class="schedule">
                            <disk-item v-for="(item, i) in disk.disks?.filter(e => !e.isSystemRoot)" :key="i"
                                :disk="item" :smartWarning="true" />
                        </div>
                    </div>
                </div>
            </template>
            <div class="item" v-if="disk.raidList && disk.raidList.length > 0">
                <div class="icon_box" style="background: #dbfce7;">
                    <memoryIcon color="#0bab47" class="icon" />
                </div>
                <div class="info">
                    <div class="name">
                        <div>{{ $gettext("RAID设备") }}</div>
                        <!-- <span>
                            <successIcon class="icon" />正常
                        </span> -->
                    </div>
                    <div class="schedule">
                        <disk-item v-for="(item, i) in disk.raidList" :key="i" :disk="item" />
                    </div>
                </div>
            </div>
        </div>
    </Card>
</template>

<script lang="ts" setup>
import Card from "../components/Card.vue"
import diskIcon from "/@/components/svg/disk1.vue"
import folderIcon from "/@/components/svg/folder.vue"
import successIcon from "/@/components/svg/success.vue"
import financeIcon from "/@/components/svg/finance.vue"
import memoryIcon from "/@/components/svg/memory.vue"
import diskItem from "../components/DiskItem.vue"
import moreItem from "/@/components/svg/more.vue"

import { reactive, ref } from 'vue';
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request';
import appUtils from "/@/utils/app";

const { $gettext } = useGettext()
const isSettingsMenuOpen = ref(false)

const disk = reactive({
    disks: null as NasDiskModel[] | null,
    raidList: null as Disksinfo[] | null,
})
const getData = () => {
    request.Nas.Disk.Status.GET().then(res => {
        if (res?.data?.result) {
            const result = res.data.result
            disk.disks = result.disks || []
        }
    })

}
const getRaidList = async () => {
    try {
        const res = await request.Raid.List.GET()
        if (res?.data) {
            const { success, error, result } = res.data
            if (result) {
                disk.raidList = result.disks || []
            }
            if (error) {
                throw error
            }
        }
    } catch (error) {
        console.log(error);
    }

}
getRaidList()
getData()

const onHandleOpenFile = () => {
    appUtils.installAndGo("luci-app-linkease", $gettext("易有云"), "/cgi-bin/luci/admin/services/linkease/file/", "app-meta-linkease")
}
</script>

<style lang="scss" scoped>
.icon {
    width: 1.3rem;
    height: 1.3rem;
}

.icon1 {
    width: 1rem;
    height: 1rem;
}

:deep(.folderIcon) {
    path {
        fill: var(--app-container_title-color) !important;
    }
}

a {
    color: #1e1e1e;
    text-decoration: none;
    cursor: pointer;
    font-size: 14px;
    display: block;

}

.content {
    .disk_loading_icon {
        height: 100px;
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 10px;

        .disk_loading_info {
            font-size: 16px;
            color: #333;
            margin-top: 12px;
        }
    }

    .line {
        height: 1px;
        background: var(--btn-border-color);
        margin: 0;
    }

    .item {
        display: flex;
        margin-top: 8px;
        padding: 10px;

        // &:hover {
        //     background: #f9fafb;
        //     border-radius: 8px;
        // }

        .icon_box {
            width: 1.5rem;
            height: 1.5rem;
            background: #dbeafe;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: 4px;

            .icon {
                width: 0.8rem;
                height: 0.8rem;
            }
        }

        .info {
            flex: 1;

            .name {
                display: flex;
                justify-content: space-between;
                align-items: center;
                margin-left: 12px;
                margin-top: 6px;

                >div {
                    font-size: 14px;
                    color: var(--app-container_title-color);
                }

                >span {
                    display: inline-flex;
                    align-items: center;
                    padding: 4px 6px;
                    line-height: 1;
                    border: 1px solid #d8e3db;
                    background: #f0fdf4;
                    border-radius: 4px;
                    color: #008236;
                    font-size: 12px;
                    font-weight: normal;

                    .icon {
                        width: 0.7rem;
                        height: 0.7rem;
                        margin-right: 4px;
                    }
                }
            }

            .schedule {
                margin-top: 12px;

                span {
                    font-size: 12px;
                    color: #6a7280;
                    font-weight: normal;
                }

                >div {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    margin-top: 8px;
                }
            }
        }
    }
}

.btn_settings {
    position: relative;
    padding: 6px 34px 6px 18px;
    border-radius: 4px;
    border: 1px solid var(--btn-border-color);
    line-height: 1;
    display: flex;
    align-items: center;
}
.rotation {
    position: absolute;
    right: 2px;
    top: 50%;
    height: 100%;
    transform: translateY(-50%);
    border-left: 1px solid var(--btn-border-color);
    display: flex;
    align-items: center;

    .moreIcon {
        transform: rotate(90deg);
    }
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    .content {
        .item {
            margin-top: 0px;
        }

        .line {
            height: 1px;
            background: #e5e7eb;
            margin: 0 0 10px;
        }
    }
}
</style>