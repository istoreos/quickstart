<template>
    <div id="page">
        <h2 name="content"> {{ $gettext("RAID管理") }}</h2>
        <div class="cbi-map-descr">
            <p>
                {{ $gettext("RAID （ Redundant Array of Independent Disks ）即独立磁盘冗余阵列，简称为「磁盘阵列」，"
                + "就是用多个独立的磁盘组成在一起形成一个大的磁盘系统，从而实现比单块磁盘更好的存储性能和更高的可靠性。") }}
            </p>
            <p style="color: #f5365b;margin-top: 10px;">
                * {{ $gettext("RAID功能正在公测中，使用过程中如造成数据丢失等问题，概不负责，请谨慎使用。") }}
            </p>
            <p style="color: #f5365b;margin-top: 10px;">
                * {{ $gettext("如果由于系统重置或重启导致的RAID设备丢失可以尝试通过下方'扫描恢复RAID'按钮恢复") }}
            </p>
        </div>
        <div class="btns">
            <button class="btn cbi-button cbi-button-apply" @click="onCreateRaid()">{{ $gettext("创建RAID") }}</button>
            <button class="btn cbi-button cbi-button-apply" @click="onRecoverRaid()"
                :disabled="disabled">{{ $gettext("扫描恢复RAID") }}</button>
        </div>
        <div>
            <div class="cbi-section cbi-tblsection" id="cbi-nfs-mount">
                <table class="table cbi-section-table">
                    <tbody style="">
                        <tr class="tr cbi-section-table-titles anonymous">
                            <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("名称") }}</th>
                            <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("设备") }}</th>
                            <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("状态") }}</th>
                            <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("级别") }}</th>
                            <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("容量") }}</th>
                            <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("成员") }}</th>
                            <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("挂载信息") }}</th>
                            <th class="th cbi-section-table-cell" data-widget="value">{{ $gettext("操作") }}</th>
                        </tr>
                        <tr class="tr cbi-section-table-row" v-for="item in raid.disksList">
                            <td class="td cbi-value-field">
                                <b>{{ item.name }}</b>
                                <!-- <b v-for="(item, i) in raidList" :key="i">{{ item }}</b> -->
                            </td>
                            <td class="td cbi-value-field">
                                <b>{{ item.path }}</b>
                            </td>

                            <td class="td cbi-value-field" :title="item.status + (item.rebuildStatus || '')">
                                <b class="item-status">
                                    {{ item.status }}
                                </b>
                                <b class="item-status item-status-detail" v-if="item.rebuildStatus">
                                    &mldr;
                                </b>
                            </td>
                            <td class="td cbi-value-field">
                                <b>{{ item.level }}</b>
                            </td>
                            <td class="td cbi-value-field">
                                <b>{{ item.size }}</b>
                            </td>
                            <td class="td cbi-value-field">
                                <b v-for="ole in item.members">{{ ole }}
                                    <br />
                                </b>
                            </td>
                            <td class="td cbi-value-field">
                                <template v-if="getRaidMountPoint(item).length > 0">
                                    <b v-for="ole in getRaidMountPoint(item)">
                                        {{ ole }}
                                        <br />
                                    </b>
                                </template>
                                <template v-else>
                                    <a href="/cgi-bin/luci/admin/quickstart/">{{ $gettext("去挂载") }}</a>
                                </template>
                            </td>
                            <td class="td cbi-section-table-cell nowrap cbi-section-actions">
                                <button class="btn cbi-button cbi-button-apply" :title="$gettext('扩充')" :disabled="getDisabled(item)"
                                    @click="onEidtRaidItem(item)">{{ $gettext("扩充") }}</button>
                                <button class="btn cbi-button cbi-button-apply" :title="$gettext('移除')" :disabled="getDisabled(item)"
                                    @click="onRemoveRaidItem(item)">{{ $gettext("移除") }}</button>
                                <button class="btn cbi-button cbi-button-apply" :title="$gettext('恢复')"
                                    @click="onRecoverRaidItem(item)">{{ $gettext("恢复") }}</button>
                                <button class="btn cbi-button cbi-button-apply" :title="$gettext('详情')"
                                    @click="onRaidItemInfo(item)">{{ $gettext("详情") }}</button>
                                <button class="cbi-button cbi-button-remove"
                                    :title="$gettext('如果您在RAID磁盘阵列中创建了文件系统，先卸载文件系统，后删除文件系统删除操作可能会导致数据丢失，请谨慎操作。')"
                                    @click="onDeleteRaidItem(item)">{{ $gettext("删除") }}</button>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>
<script setup lang="ts">
import { onBeforeUnmount, reactive, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import request from '/@/request';
import ActionRaid from "/@/components/action-raid"
import ActionConfirm from "/@/components/action-confirm"
import HintSvg from "/@/components/svg/hint.vue"
import Toast from '/@/components/toast';
import Action from '/@/components/action-ddnsto';
import utils from "/@/utils";

const raid = reactive({
    disksList: [] as Disksinfo[]
})
const getRaidList = async () => {
    try {
        const res = await request.Raid.List.GET()
        if (res?.data) {
            const { success, error, result } = res.data
            if (result) {
                raid.disksList = result.disks || []
            }
            if (error) {
                throw error
            }
        }
    } catch (error) {
        console.log(error);
    }

}
const cancelGetRaidList = utils.easyInterval(getRaidList, 5000)
onBeforeUnmount(() => {
    cancelGetRaidList()
})
const getDisabled = (item: Disksinfo) => {
    switch (item.level) {
        case "raid0":
        case "jbod":
            return true
    }
    if (item.status.indexOf("degraded") != -1) {
        return true
    }
    if (item.status.indexOf("resyncing(") != -1) {
        return true
    }
    return false
}
const getRaidMountPoint = (item: Disksinfo): string[] => {
    let mountPoint: string[] = []
    if (item.childrens) {
        item.childrens.forEach(item => {
            if (item.mountPoint) {
                mountPoint.push(`${item.name}(${item.mountPoint})`)
            }
        })
    }
    return mountPoint
}

// 创建raid
const onCreateRaid = async () => {
    ActionRaid({
        setup: "create",
        success: () => {
            getRaidList()
        }
    })
}
const onRaidItemInfo = (item: Disksinfo) => {
    ActionRaid({
        setup: "info",
        raid: item
    })
}
const onDeleteRaidItem = async (item: Disksinfo) => {
    if (item.childrens && item.childrens.length > 0) {
        const mountPoints = item.childrens.filter(item => {
            return item.mountPoint
        })
        if (mountPoints.length > 0) {
            ActionConfirm({
                content: $gettext("删除 RAID 设备之前请先卸载文件系统"),
                nextTitle: $gettext("去卸载"),
                next: () => {
                    location.href = `/cgi-bin/luci/admin/system/mounts`
                },
                clearTitle: $gettext("取消"),
                clear: () => { }
            })
            return
        }
    }

    if (!confirm($gettext("确定要删除 %{name} 该磁盘阵列吗？删除操作可能会导致数据丢失，请谨慎操作。删除成功后，如需另外组RAID，建议稍等几分钟后再试。", {name:item.name}))) {
        return
    }
    if (!confirm($gettext("确定要删除 %{name} 吗?", {name:item.name}))) {
        return
    }
    const load = Toast.Loading($gettext("删除中..."))
    try {
        const res = await request.Raid.Delete.POST({
            path: item.path,
            members: item.members,
        })
        if (res.data) {
            const { success, error } = res.data
            if (error) {
                throw error
            }
            if ((success || 0) == 0) {
                Toast.Success($gettext("删除成功"))
                getRaidList()
            }
        }
    } catch (error) {
        Toast.Error(`${error}`)
    } finally {
        load.Close()
    }


}
const onEidtRaidItem = (item: Disksinfo) => {
    ActionRaid({
        setup: "edit",
        raid: item,
        success: () => {
            getRaidList()
        }
    })
}
const onRemoveRaidItem = (item: Disksinfo) => {
    ActionRaid({
        setup: "remove",
        raid: item,
        success: () => {
            getRaidList()
        }
    })
}
const onRecoverRaidItem = (item: Disksinfo) => {
    ActionRaid({
        setup: "recover",
        raid: item,
        success: () => {
            getRaidList()
        }
    })
}
const disabled = ref(false)
const onRecoverRaid = () => {
    ActionConfirm({
        content: $gettext("将扫描磁盘里 RAID 的磁盘阵列配置并恢复，确定要恢复 RAID 磁盘阵列吗？"),
        nextTitle: $gettext("确定"),
        next: async () => {
            disabled.value = true
            const load = Toast.Loading($gettext("扫描中..."))
            try {
                const res = await request.Raid.Autofix.POST()
                if (res.data) {
                    const { error, success } = res.data
                    if (error) {
                        throw error
                    }
                    if ((success || 0) == 0) {
                        Toast.Success($gettext("恢复完成"))
                        getRaidList()
                    }
                }
            } catch (error) {
                Toast.Error(`${error}`)
            } finally {
                load.Close()
                disabled.value = false
            }
        },
        clearTitle: $gettext("取消"),
        clear: () => { }
    })
}

</script>
<style lang="scss" scoped>
.cbi-map-descr {
    margin-bottom: 32px;
}

.item-status {
    word-break: break-all;
    text-overflow: ellipsis;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
    overflow: hidden;
}
.item-status-detail {
    text-decoration: underline;
    cursor: help;
}
</style>