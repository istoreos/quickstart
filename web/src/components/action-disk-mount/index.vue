<template>
    <action-component :type="1">
        <transition name="rotate" mode="out-in">
            <form class="action format" @submit.prevent="onSumbit" v-if="setup == 0">
                <div class="action-header">
                    <div class="action-header_title"></div>
                </div>
                <div class="action-body">
                    <div class="disk-info">
                        <div class="disk-info_icon">
                            <svg t="1642589762094" class="icon" viewBox="0 0 1024 1024" version="1.1"
                                xmlns="http://www.w3.org/2000/svg" p-id="11301" width="128" height="128">
                                <path
                                    d="M899.892468 123.889088c0-44.342099-36.286708-80.620486-80.624646-80.620486H204.728017C160.385918 43.268602 124.107532 79.546988 124.107532 123.889088v802.847056c0 44.342099 36.278386 80.620486 80.620485 80.620486h614.539805c44.337938 0 80.624646-36.278386 80.624646-80.620486V123.889088z"
                                    fill="#D0D0DB" p-id="11302" />
                                <path
                                    d="M169.8768 977.7772V174.930143c0-44.342099 36.278386-80.620486 80.620486-80.620485h614.539804c9.936092 0 19.426974 1.905666 28.239639 5.23434-11.525534-30.507298-40.996782-52.389169-75.398629-52.389169H203.342457c-44.342099 0-80.620486 36.278386-80.620486 80.620486v802.851217c0 34.410168 21.881871 63.873094 52.385008 75.381985A79.730065 79.730065 0 0 1 169.8768 977.7772z"
                                    fill="#FFFFFF" p-id="11303" />
                                <path
                                    d="M820.657543 40.497481H206.117739c-44.342099 0-80.620486 36.278386-80.620486 80.620485v802.847057c0 44.342099 36.278386 80.620486 80.620486 80.620486h614.539804c44.337938 0 80.624646-36.278386 80.624647-80.620486V121.117966c0-44.342099-36.286708-80.620486-80.624647-80.620485z m19.60173 828.785749c0 40.846992-33.43237 74.279362-74.287684 74.279361H199.780776c-40.855313 0-74.279362-33.424048-74.279362-74.279361V129.593603c0-40.855313 33.424048-74.279362 74.279362-74.279362h566.203296c40.842831 0 74.283522 33.424048 74.283522 74.279362l-0.008321 739.689627z"
                                    fill="#6E6E96" p-id="11304" />
                                <path
                                    d="M815.106979 1024H200.567175C146.933914 1024 103.303319 980.369405 103.303319 926.736144V123.889088C103.303319 70.255827 146.933914 26.625232 200.567175 26.625232h614.539804c53.633261 0 97.268017 43.630595 97.268017 97.263856v802.847056c0 53.633261-43.634756 97.263856-97.268017 97.263856zM200.567175 59.911972C165.287391 59.911972 136.590059 88.609303 136.590059 123.889088v802.847056c0 35.279784 28.697331 63.977115 63.977116 63.977115h614.539804c35.279784 0 63.981276-28.697331 63.981276-63.977115V123.889088c0-35.279784-28.701492-63.977115-63.981276-63.977116H200.567175z"
                                    fill="#6E6E96" p-id="11305" />
                                <path d="M301.946104 941.515457h429.985632v65.841173H301.946104z" fill="#8A8AA1"
                                    p-id="11306" />
                                <path
                                    d="M731.931736 1024H301.946104a16.64337 16.64337 0 0 1-16.64337-16.64337V941.515457a16.64337 16.64337 0 0 1 16.64337-16.64337h429.985632a16.64337 16.64337 0 0 1 16.64337 16.64337v65.841173a16.64337 16.64337 0 0 1-16.64337 16.64337z m-413.342262-33.286741h396.698892v-32.554432H318.589474v32.554432z"
                                    fill="#6E6E96" p-id="11307" />
                                <path
                                    d="M337.230049 960.318304h20.804213v47.038326h-20.804213zM386.565159 960.318304h20.804213v47.038326h-20.804213zM435.891948 960.318304h20.804213v47.038326h-20.804213zM485.231219 960.318304h20.804213v47.038326h-20.804213zM534.558008 960.318304h20.804213v47.038326h-20.804213zM583.897279 960.318304h20.804213v47.038326h-20.804213zM633.224068 960.318304h20.804213v47.038326h-20.804213zM682.563339 960.318304h20.804213v47.038326h-20.804213z"
                                    fill="#FFE599" p-id="11308" />
                                <path
                                    d="M219.153659 140.794591m-26.874883 0a26.874882 26.874882 0 1 0 53.749765 0 26.874882 26.874882 0 1 0-53.749765 0Z"
                                    fill="#ADADD1" p-id="11309" />
                                <path
                                    d="M219.153659 184.312843c-23.995579 0-43.518252-19.522673-43.518253-43.518252s19.522673-43.518252 43.518253-43.518253 43.518252 19.522673 43.518252 43.518253-19.522673 43.518252-43.518252 43.518252z m0-53.749764c-5.642103 0-10.231512 4.589409-10.231512 10.231512s4.589409 10.231512 10.231512 10.231512 10.231512-4.589409 10.231511-10.231512-4.589409-10.231512-10.231511-10.231512z"
                                    fill="#6E6E96" p-id="11310" />
                                <path
                                    d="M801.28466 140.794591m-26.870721 0a26.870721 26.870721 0 1 0 53.741442 0 26.870721 26.870721 0 1 0-53.741442 0Z"
                                    fill="#ADADD1" p-id="11311" />
                                <path
                                    d="M801.28466 184.308683c-23.995579 0-43.514092-19.518512-43.514091-43.514092s19.518512-43.514092 43.514091-43.514092 43.514092 19.518512 43.514092 43.514092-19.518512 43.514092-43.514092 43.514092z m0-53.741443c-5.637942 0-10.227351 4.589409-10.227351 10.227351s4.589409 10.227351 10.227351 10.227351 10.227351-4.589409 10.227351-10.227351-4.589409-10.227351-10.227351-10.227351z"
                                    fill="#6E6E96" p-id="11312" />
                                <path
                                    d="M801.280499 905.23291m-26.870721 0a26.870721 26.870721 0 1 0 53.741443 0 26.870721 26.870721 0 1 0-53.741443 0Z"
                                    fill="#ADADD1" p-id="11313" />
                                <path
                                    d="M801.280499 948.747001c-23.995579 0-43.514092-19.518512-43.514091-43.514091s19.518512-43.514092 43.514091-43.514092 43.514092 19.518512 43.514092 43.514092-19.518512 43.514092-43.514092 43.514091z m0-53.741442c-5.637942 0-10.227351 4.589409-10.227351 10.227351s4.589409 10.227351 10.227351 10.227351 10.227351-4.589409 10.227351-10.227351-4.589409-10.227351-10.227351-10.227351z"
                                    fill="#6E6E96" p-id="11314" />
                                <path
                                    d="M219.153659 905.23291m-26.870722 0a26.870721 26.870721 0 1 0 53.741443 0 26.870721 26.870721 0 1 0-53.741443 0Z"
                                    fill="#ADADD1" p-id="11315" />
                                <path
                                    d="M219.153659 948.747001c-23.995579 0-43.514092-19.518512-43.514092-43.514091s19.518512-43.514092 43.514092-43.514092 43.514092 19.518512 43.514091 43.514092-19.522673 43.514092-43.514091 43.514091z m0-53.741442c-5.637942 0-10.227351 4.589409-10.227351 10.227351s4.589409 10.227351 10.227351 10.227351 10.227351-4.589409 10.227351-10.227351-4.589409-10.227351-10.227351-10.227351z"
                                    fill="#6E6E96" p-id="11316" />
                                <path
                                    d="M520.972857 777.43263c-142.542145 0-258.508988-115.971004-258.508988-258.52147a16.64337 16.64337 0 0 1 33.28674 0c0 124.19699 101.033579 225.23473 225.222248 225.23473s225.222248-101.03774 225.222248-225.23473c0-124.188668-101.033579-225.218087-225.222248-225.218087a16.64337 16.64337 0 0 1 0-33.286741c142.542145 0 258.508988 115.966843 258.508988 258.504828 0 142.550466-115.966843 258.521471-258.508988 258.52147z"
                                    fill="#6E6E96" p-id="11317" />
                                <path
                                    d="M520.968696 518.919481m-83.312551 0a83.312551 83.312551 0 1 0 166.625102 0 83.312551 83.312551 0 1 0-166.625102 0Z"
                                    fill="#A9A9BA" p-id="11318" />
                                <path
                                    d="M520.968696 618.875402c-55.114521 0-99.955921-44.83724-99.955921-99.95176 0-55.118682 44.8414-99.955921 99.955921-99.955921s99.95176 44.8414 99.95176 99.955921c0 55.11036-44.83724 99.95176-99.95176 99.95176z m0-166.625101c-36.761044 0-66.669181 29.908136-66.66918 66.66918s29.908136 66.66502 66.66918 66.66502 66.66502-29.908136 66.66502-66.66502c0-36.761044-29.903976-66.669181-66.66502-66.66918z"
                                    fill="#6E6E96" p-id="11319" />
                                <path d="M301.946104 941.515457h429.985632v36.977408H301.946104z" fill="#6E6E96"
                                    p-id="11320" />
                            </svg>
                        </div>
                        <div class="disk-info_mount-name" v-if="mount">
                            <span>【{{ mount.total }}】</span>
                            <span>{{ mount.mountPoint }}</span>
                        </div>
                        <div class="disk-info_mount-name" v-else-if="disk">
                            <span>【{{ disk.size }}】</span>
                            <span>{{ disk.venderModel }}</span>
                        </div>
                    </div>

                    <div class="label-item" v-if="mount">
                        <div class="label-item_key">
                            <span>{{ $gettext("目标分区") }}</span>
                        </div>
                        <div class="label-item_path">{{ mount.path }}（{{ mount.total }}，{{
                                mount?.filesystem?.toUpperCase()
                        }}）</div>
                    </div>

                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("挂载点") }}</span>
                        </div>
                        <div class="label-item_value">
                            <input type="text" v-model.trim="mountPoint" />
                        </div>
                    </div>
                </div>
                <div class="action-footer">
                    <div class="auto"></div>

                    <button class="cbi-button cbi-button-apply app-btn app-next" :disabled="disabled"
                        @click="onMount">{{ $gettext("确定") }}</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onCancel" type="button"
                        :disabled="disabled">{{ $gettext("返回") }}</button>
                </div>
            </form>
            <div class="action result" v-else-if="setup == 1">
                <div class="action-body">
                    <div class="action-body_icon">
                        <svg t="1642063181211" class="icon" viewBox="0 0 1024 1024" version="1.1"
                            xmlns="http://www.w3.org/2000/svg" p-id="5062" width="128" height="128" data-v-cda444e0>
                            <path
                                d="M512 85.333333c235.648 0 426.666667 191.018667 426.666667 426.666667s-191.018667 426.666667-426.666667 426.666667S85.333333 747.648 85.333333 512 276.352 85.333333 512 85.333333z m-74.965333 550.4L346.453333 545.152a42.666667 42.666667 0 1 0-60.330666 60.330667l120.704 120.704a42.666667 42.666667 0 0 0 60.330666 0l301.653334-301.696a42.666667 42.666667 0 1 0-60.288-60.330667l-271.530667 271.488z"
                                fill="#52C41A" p-id="5063" data-v-cda444e0 />
                        </svg>
                    </div>
                    <div class="action-body_msg">{{ $gettext("挂载成功") }}</div>
                    <div class="action-body_info" v-if="resultMount" v-html="$gettext('已成功将分区 %{dev} 挂载到 <a>%{mount}</a>', {dev:resultMount.path||'', mount:resultMount.mountPoint||''}, true)">
                    </div>
                    <div class="btns">
                        <button class="cbi-button cbi-button-apply app-btn app-next" type="button" @click="onSuccess">{{
                                action == 'nas' ? $gettext('完成') : $gettext('下一步')
                        }}</button>
                    </div>
                </div>
            </div>
        </transition>
    </action-component>
</template>
<script setup lang="ts">
import { onMounted, PropType, ref, provide } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import ActionComponent from "/@/components/action/modal.vue"
import request from "/@/request";
import Toast from "/@/components/toast";
const props = defineProps({
    action: String,
    disk: {
        type: Object as PropType<NasDiskModel>,
        required: true
    },
    mount: {
        type: Object as PropType<MountPoint>
    },
    Close: {
        type: Function as PropType<() => void>
    },
    Cancel: {
        type: Function as PropType<() => void>
    },
    Next: {
        type: Function as PropType<(rootPath: string) => void>
    }
})
const onClose = () => {
    if (props.Close) {
        props.Close()
    }
}
const onCancel = (e: Event) => {
    e.preventDefault()
    if (props.Cancel) {
        props.Cancel()
    }
    onClose()
}
const onNext = (_rootPath: string) => {
    if (props.Next) {
        props.Next(_rootPath)
    }
    onClose()
    return
}
const disabled = ref(false)
const setup = ref(0)
const mountPoint = ref("/mnt/data_" + props?.mount?.name);
const onSetup = (_setup: number) => {
    setup.value = _setup
}
const value = ref(props.mount ? "" : "format")
const resultDisk = ref<NasDiskModel | null>()
const resultMount = ref<MountPoint | null>()

const onMount = async () => {
    const _mount = props.mount
    if (_mount == null) {
        Toast.Warning($gettext("获取不到分区"))
        return
    }
    if (_mount.path == null || _mount.path == "") {
        Toast.Warning($gettext("获取不到分区路径"))
        return
    }
    if (_mount.uuid == null || _mount.uuid == "") {
        Toast.Warning($gettext("获取不到分区ID"))
        return
    }
    disabled.value = true
    const load = Toast.Loading($gettext("挂载中..."))
    try {
        const res = await request.Nas.Disk.Partition.Mount.POST({
            path: _mount.path,
            uuid: _mount.uuid,
            mountPoint: mountPoint.value
        })
        if (res?.data) {
            const { result, error } = res?.data
            if (error) {
                Toast.Warning(error)
            }
            if (result) {
                Toast.Success($gettext("挂载成功"))
                resultMount.value = result
                onSetup(1)
            }
        }
    } catch (error) {
        Toast.Error(error as string)
    }
    load.Close()
    disabled.value = false
}
const onSuccess = () => {
    if (resultMount.value) {
        if (resultMount.value.mountPoint) {
            onNext(resultMount.value.mountPoint)
            return
        }
    }
    Toast.Warning($gettext("读取结果失败"))
}
const onSumbit = () => {

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

        .label-item_path {
            padding: 0 14px;
            background-color: #e5e5e5;
            width: 100%;
            height: 28px;
            line-height: 28px;
            margin-top: 10px;
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

    .action-body {
        width: 100%;
        height: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        align-content: center;
        justify-content: center;

        .action-body_icon {
            width: 100px;
            height: 100px;

            svg.icon {
                width: 100%;
                height: 100%;
            }
        }

        .action-body_msg {
            font-size: 2em;
            color: #666;
            text-align: center;
            width: 100%;
            margin: 1rem 0;
        }

        .action-body_info {
            margin: 1rem 0;
            width: 100%;
            text-align: center;
            color: #666;
            font-size: 1.2em;

            a {
                color: #0000fb;
            }
        }
    }

    .btns {
        width: 100%;
        text-align: center;
        margin: 1rem 0;
    }
}
</style>