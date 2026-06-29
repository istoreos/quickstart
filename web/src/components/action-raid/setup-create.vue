<template>
    <div class="actioner-container">
        <div class="actioner-container_header">
            <span>{{ $gettext("RAID创建向导") }}</span>
        </div>
        <template v-if="status == 'init'">
            <div class="actioner-container_body">
                <p>{{ $gettext("RAID磁盘阵列是用多个独立的磁盘组成在一起形成一个大的磁盘系统，从而实现比单块磁盘更好的存储性能和更高的可靠性。") }}</p>

                <div class="label-item">
                    <div class="label-item_key">
                        <span>{{ $gettext("RAID级别：") }}</span>
                    </div>
                    <div class="label-item_value">
                        <select v-model="selectReidType" @change="onSelectChange">
                            <option :value="item.name" v-for="item in raidType">{{ item.title }}</option>
                        </select>
                    </div>
                    <div class="label-item_tips">
                        <HintSvg />
                        {{ getRaidInfo(selectReidType) }}
                    </div>
                </div>
                <div class="label-item">
                    <div class="label-item_key">
                        <span>{{ $gettext("磁盘阵列成员：") }}</span>
                    </div>
                    <div class="label-item_value" v-if="raidConfig.loading">
                        <span class="msg-warning">
                            {{ $gettext("检测中...") }}
                        </span>
                    </div>
                    <div class="label-item_value" v-else>
                        <template v-if="raidConfig.members.length > 0">
                            <label v-for="item in raidConfig.members">
                                <input type="checkbox" v-model="selectDisk" :value="item.path">
                                【{{ item.model }}】{{ item.name }} {{ item.path }} [{{ item.sizeStr }}]
                            </label>
                        </template>
                        <template v-else>
                            <span class="msg-warning">
                                {{ $gettext("检测不到可用磁盘阵列成员") }}
                            </span>
                        </template>
                    </div>
                    <div class="label-item_tips">
                        <HintSvg />
                        {{ $gettext("选择将要用于创建 RAID 的硬盘，通过 USB 接入的设备不会在列表中显示（USB接入不稳定）。") }}
                    </div>
                </div>
            </div>
            <div class="actioner-container_footer">
                <div class="close" @click="onClose" :disabled="disabled">{{ $gettext("取消") }}</div>
                <div class="next" @click="onNext" :disabled="disabled">{{ $gettext("创建") }}</div>
            </div>
        </template>
        <!-- 挂载中 -->
        <template v-else-if="status == 'loading'">
            <div class="actioner-container_body setup-loading">
                <icon-loading :size="60" color="#666" />
                <span>{{ $gettext("正在创建中...") }}</span>
            </div>
        </template>
        <!-- 错误 -->
        <template v-else-if="status == 'error'">
            <div class="actioner-container_body setup-error">
                <icon-error />
                <span>{{ msg }}</span>
            </div>
            <div class="actioner-container_footer">
                <div class="close" @click="onClose">{{ $gettext("关闭") }}</div>
                <div class="next" @click="onNext" :disabled="disabled">{{ $gettext("重新创建") }}</div>
            </div>
        </template>
        <!-- 成功 -->
        <template v-else-if="status == 'success'">
            <div class="actioner-container_body setup-success">
                <icon-success />
                <div class="body-title">{{ $gettext("创建成功") }}</div>
            </div>
            <div class="actioner-container_footer">
                <div class="close" @click="onClose">{{ $gettext("关闭") }}</div>
            </div>
        </template>
    </div>
</template>
<script setup lang="ts">
import { PropType, reactive, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import HintSvg from "/@/components/svg/hint.vue"
import Toast from "/@/components/toast";
import request from "/@/request";
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
    success: {
        type: Function as PropType<ActionRaidSuccess>
    }
})
const onClose = () => {
    // props.onSetup()
    props.Close()

}
const onSuccess = () => {
    if (props.success) {
        props.success()
    }
}
const status = ref<"init" | "loading" | "error" | "success">("init")
const msg = ref<any>("")
const raidType = [
    {
        name: "jbod",
        title: $gettext("JBOD (线性)"),
        info: $gettext("至少需要2块硬盘，将多个硬盘合并为单个存储空间，其容量等于所有硬盘容量的总和。不提供数据冗余。"),
        select: 2
    },
    {
        name: "raid0",
        title: $gettext("RAID 0 (条带)"),
        info: $gettext("至少需要2块硬盘，“区块延展”功能是将数据分成多个块，并将数据块分散到组成的多个硬盘上以提高性能的过程。不提供数据冗余。"),
        select: 2
    },
    {
        name: "raid1",
        title: $gettext("RAID 1 (镜像)"),
        info: $gettext("至少需要2块硬盘，同时向所有硬盘写入相同的数据。提供数据冗余。"),
        select: 2
    },
    {
        name: "raid5",
        title: "RAID 5 ",
        info: $gettext("至少需要3块硬盘，执行段落分块延展，并对分布到所有组成硬盘上的数据执行奇偶校验，从而提供比 RAID 1 更有效的数据冗余。"),
        select: 3
    },
    {
        name: "raid6",
        title: "RAID 6 ",
        info: $gettext("至少需要4块硬盘，执行两个层级的数据奇偶校验以存储等于 2 个硬盘容量的冗余数据，提供比 RAID 5 更大程度的数据冗余。"),
        select: 4
    },
    {
        name: "raid10",
        title: "RAID 10",
        info: $gettext("至少需要4块硬盘，提供 RAID 0 的性能和 RAID 1 的数据保护级别，将硬盘组合进镜像数据的由两个硬盘组成的组。"),
        select: 4
    }
]
const selectReidType = ref("raid5")
const selectDisk = ref<string[]>([])
const getRaidInfo = (e: string) => {
    let info = ""
    raidType.forEach(item => {
        if (item.name === e) {
            info = item.info
        }
    })
    return info
}
const disabled = ref(false)
const raidConfig = reactive({
    loading: false,
    members: [] as Membersinfo[]
})
const onSelectChange = (e: Event) => {
    // const target = e.target as HTMLInputElement
    // if (target.value != selectReidType.value) {
    //     selectDisk.value = []
    // }
}
const getData = async () => {
    raidConfig.loading = true
    try {
        const res = await request.Raid.CreateList.GET()
        if (res?.data) {
            const { success, error, result } = res.data
            if (result) {
                raidConfig.members = result.members || []
            }
            if (error) {
                throw error
            }
        }
    } catch (error) {
        console.log(error);
    } finally {
        raidConfig.loading = false
    }
}
getData()
const onNext = async () => {
    const raid = raidType.filter((item) => item.name === selectReidType.value)[0]
    const disks = selectDisk.value
    if (!raid) {
        Toast.Warning($gettext("请选择raid类型"))
        return
    }
    if (disks.length == 0) {
        Toast.Warning($gettext("请选择磁盘"))
        return
    }
    if (raid.select > disks.length) {
        Toast.Warning($gettext("请选择至少%{min}块磁盘", {min: ''+raid.select}))
        return
    }
    if (!confirm($gettext("是否立即创建 %{name}？选择的硬盘所有分区将会被清除，此操作可能会导致硬盘数据丢失，请谨慎操作。", {name:raid.name}))) {
        return
    }
    if (!confirm($gettext("确定创建 %{name}？该操作不可逆,请谨慎操作", {name:raid.name}))) {
        return
    }
    disabled.value = true
    status.value = 'loading'
    try {
        const res = await request.Raid.Create.POST({
            level: raid.name,
            devicePaths: disks
        })
        if (res.data) {
            const { success, error, result } = res.data
            if (error) {
                throw error
            }
            if ((success || 0) == 0) {
                status.value = "success"
                onSuccess()
            }
        }
    } catch (error) {
        msg.value = error
        status.value = "error"
    } finally {
        disabled.value = false
    }
}
</script>
<style lang="scss" scoped>
p {
    line-height: 22px;
    font-size: 14px;
}


.label-item {
    width: 100%;
    margin: 10px 0;

    .label-item_key {
        width: 100%;
        font-size: 14px;
        color: #999;
        margin-bottom: 6px;

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
        margin: 10px 0;

        select,
        input {
            width: 100%;
            height: 36px;
            line-height: 36px;
            color: #000;
        }

        // option {
        //     color: #999;
        // }

        input::placeholder {
            color: #999;
            font-size: 12PX;
        }

        label {
            width: 100%;
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            cursor: pointer;
            margin: 0.5rem;
            border-bottom: 1px solid #eee;
            padding-bottom: 10px;
            font-size: 14px;
            color: #666;

            input[type="checkbox"] {
                top: 0;
            }
        }
    }

    .label-item_tips {
        margin-top: 10px;
        color: #666;
        font-size: 14px;

        svg {
            vertical-align: top;
        }
    }



}

span.msg-warning {
    width: 100%;
    text-align: left;
    font-size: 14px;
    color: #f00;
    display: block;
    margin: 10px 0;
}

.label-message {
    width: 100%;
    text-align: left;
    font-size: 14px;
    color: #f00;
    text-align: center;
}



.actioner-container_body.setup-loading {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    align-content: center;
    text-align: center;

    span {
        width: 100%;
        display: block;
        font-size: 1.2em;
        margin-top: 1rem;
        color: #666;
    }
}

.actioner-container_body.setup-error {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: center;
    align-content: center;
    text-align: center;

    svg.icon {
        width: 100px;
        height: 100px;
    }

    span {
        width: 100%;
        display: block;
        font-size: 1.4em;
        color: #ff6b6b;
    }
}

.actioner-container_body.setup-success {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    align-content: center;
    justify-content: center;

    svg.icon {
        width: 100px;
        height: 100px;
    }

    .body-title {
        width: 100%;
        display: block;
        color: #1e1e1e;
        font-size: 2em;
        padding: 0;
        margin: 1rem 0;
        text-align: center;
    }

    .body-tips {
        text-align: center;
    }

    .body-info {
        color: #666;
        font-size: 1.3em;
        margin: 1rem 0;
        width: 100%;
        text-align: center;

        span {
            display: block;
        }
    }

    .body-tips {
        margin: 1rem 0;
        display: block;
        width: 100%;
    }

    .body-btns {
        width: 100%;
        margin-top: 3rem;

        button {
            display: block;
            width: 100% !important;
            margin: 0.5rem 0;
        }
    }
}
</style>