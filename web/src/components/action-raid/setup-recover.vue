<template>
    <div class="actioner-container">
        <div class="actioner-container_header">
            <span>RAID - {{ raid.name }} {{ $gettext("恢复") }}</span>
        </div>
        <div class="actioner-container_body">
            <div class="label-item">
                <div class="label-item_key">
                    {{ $gettext("设备") }}
                </div>
                <div class="label-item_value">
                    <select disabled>
                        <option>{{ raid.name }}_{{ raid.venderModel }} ({{ raid.path }}，{{ raid.level }}，{{ raid.size
                        }})
                        </option>
                    </select>
                </div>
            </div>
            <div class="label-item">
                <div class="label-item_key">
                    {{ $gettext("选择硬盘（选择空闲的硬盘恢复RAID设备）：") }}
                </div>
                <div class="label-item_value" v-if="raidConfig.loading">
                    <span class="msg-warning">
                        {{ $gettext("检测中...") }}
                    </span>
                </div>
                <div class="label-item_value" v-else>
                    <template v-if="raidConfig.members.length > 0">
                        <label v-for="item in raidConfig.members">
                            <input type="radio" v-model="selectDisk" :value="item.path">
                            【{{ item.model }}】{{ item.name }} {{ item.path }} [{{ item.sizeStr }}]
                        </label>
                    </template>
                    <template v-else>
                        <span class="msg-warning">
                            {{ $gettext("检测不到可用磁盘阵列成员") }}
                        </span>
                    </template>
                </div>
            </div>
        </div>
        <div class="actioner-container_footer">
            <div class="close" @click="onClose" :disabled="disabled">{{ $gettext("取消") }}</div>
            <div class="next" @click="onNext" :disabled="disabled">{{ $gettext("保存") }}</div>
        </div>
    </div>
</template>
<script setup lang="ts">
import { PropType, reactive, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from '../toast';
import request from '/@/request';
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
    raid: {
        type: Object as PropType<Disksinfo>,
        required: true
    },
    success: {
        type: Function as PropType<ActionRaidSuccess>
    }
})
const onClose = () => {
    props.Close()
}
const onSuccess = () => {
    if (props.success) {
        props.success()
    }
}
const onNext = async () => {
    const memberPath = selectDisk.value
    if (memberPath == "") {
        Toast.Warning($gettext("请选择要添加的硬盘"))
        return
    }
    disabled.value = true
    const load = Toast.Loading($gettext("保存中..."))
    try {
        const res = await request.Raid.Recover.POST({
            path: props.raid.path,
            memberPath: memberPath
        })
        if (res.data) {
            const { error, success } = res.data
            if (error) {
                throw error
            }
            if ((success || 0) == 0) {
                Toast.Success($gettext("保存成功"))
                onSuccess()
                onClose()
            }
        }
    } catch (error) {
        Toast.Error(`${error}`)
    } finally {
        disabled.value = false
        load.Close()
    }

}
const raidConfig = reactive({
    loading: false,
    members: [] as Membersinfo[]
})
const disabled = ref(false)
const selectDisk = ref<string>("")
const getData = async () => {
    raidConfig.loading = true
    disabled.value = true
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
        disabled.value = false
        raidConfig.loading = false
    }
}
getData()


</script>
<style lang="scss" scoped>
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

        select {
            width: 100%;
            height: 36px;
            line-height: 36px;
            color: #000;
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

            input[type="radio"] {
                top: 0;
                margin: 0;
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
</style>
