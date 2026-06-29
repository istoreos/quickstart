<template>
    <action-component :Close="Close" :type="1">
            <div class="action" v-if="setup == 0">
                <h2 class="title">{{ $gettext("下载服务配置向导") }}</h2>
                <ul>
                    <li>
                        <div class="load_service input_row">
                            <div class="left">
                                <span>{{ $gettext("下载服务：") }}</span>
                            </div>
                            <div class="radios">
                                <input type="radio" value="Aria2" v-model="status" name="download" id="Aria2" />
                                <label for="Aria2">Aria2</label>
                            </div>

                            <div class="radios">
                                <input type="radio" value="qBittorrent" v-model="status" name="download" id="qB" />
                                <label for="qB">qBittorrent</label>
                            </div>

                            <div class="radios">
                                <input type="radio" value="Transmission" v-model="status" name="download" id="Tr" />
                                <label for="Tr">Transmission</label>
                            </div>

                        </div>
                    </li>

                </ul>
                <!-- Aria2配置 -->
                <form @submit.prevent="onNextAria2" v-if="status == 'Aria2'">
                    <ul>
                        <li>
                            <div class="input_row">
                                <div class="left">
                                    <span class="tooltip-trigger">
                                        <TipsSvg></TipsSvg>
                                        <div>
                                            <div class="tooltip-text tooltip-top">
                                                <span class="dowload_dir_tip">{{ $gettext("用于放置配置文件的目录。例如：/mnt/sda1/Configs/aria2；请勿使用 /tmp 或 /var ，以免重启以后任务丢失") }}</span>
                                            </div>
                                        </div>
                                    </span>
                                    <span>{{ $gettext("配置目录：") }}</span>
                                </div>

                                <div class="myinput_wrap">
                                    <editable-select v-model.trim="configPath" :title="$gettext('配置目录')" :options="disks.concat('/root').map(k=>{return {key:k+'/Configs/aria2'}})"/>
                                </div>
                            </div>
                        </li>
                        <li>
                            <div class="input_row">
                                <div class="left">
                                    <span class="tooltip-trigger">
                                        <TipsSvg></TipsSvg>
                                        <div>
                                            <div class="tooltip-text tooltip-top">
                                                <span class="dowload_dir_tip">{{ $gettext("用于放置下载文件的目录。例如：/mnt/sda1/download") }}</span>
                                            </div>
                                        </div>
                                    </span>
                                    <span>{{ $gettext("下载目录：") }}</span>
                                </div>

                                <div class="myinput_wrap">
                                    <editable-select v-model.trim="downloadPath" :title="$gettext('下载目录')" :options="dlDirList"/>
                                </div>
                            </div>
                        </li>

                        <li>
                            <div class="input_row">
                                <div class="left">
                                    <span class="tooltip-trigger">
                                        <TipsSvg></TipsSvg>
                                        <div>
                                            <div class="tooltip-text tooltip-bottom">
                                                <span class="dowload_rpc_tip">{{ $gettext("用于远程访问的令牌。") }}</span>
                                            </div>
                                        </div>
                                    </span>
                                    <span>{{ $gettext("RPC 令牌：") }}</span>
                                </div>

                                <input type="text" class="RPC_input" :placeholder="$gettext('请输入RPC令牌')" v-model.trim="rpcToken" />
                            </div>
                        </li>

                        <li>

                            <div class="input_row">
                                <div class="">
                                    <span>{{ $gettext("附加的 BT Tracker：") }}</span>
                                </div>
                                <div class="radios">
                                    <input type="radio" value="default" name="BT" id="default" v-model="trackerSource" />
                                    <label for="default">{{ $gettext("默认") }}</label>
                                </div>
                                <div class="radios">
                                    <input type="radio" value="add" name="BT" id="add" v-model="trackerSource" />
                                    <label for="add">{{ $gettext("自己添加") }}</label>
                                </div>
                            </div>
                        </li>

                        <li>
                            <div class="input_row">
                                <div class="left">
                                </div>
                                <div class="myinput_wrap Tracker_input">
                                    <textarea v-model.trim="btTracker" rows="4"
                                        v-if="trackerSource == 'add'" :placeholder="$gettext('请输入BT Tracker服务器地址，多个地址使用换行或者英文逗号分隔')"></textarea>
                                </div>
                            </div>
                        </li>
                    </ul>
                </form>

                <!-- qBittorrent配置 -->
                <form @submit.prevent="onNextqBittorrent" v-if="status == 'qBittorrent'">
                    <ul>
                        <li>
                            <div class="input_row">
                                <div class="left">
                                    <span class="tooltip-trigger">
                                        <TipsSvg></TipsSvg>
                                        <div>
                                            <div class="tooltip-text tooltip-top">
                                                <span class="dowload_dir_tip">{{ $gettext("用于放置配置文件的目录。例如：/mnt/sda1/Configs/qb；请勿使用 /tmp 或 /var ，以免重启以后任务丢失") }}</span>
                                            </div>
                                        </div>
                                    </span>
                                    <span>{{ $gettext("配置目录：") }}</span>
                                </div>

                                <div class="myinput_wrap">
                                    <editable-select v-model.trim="qbConfigPath" :title="$gettext('配置目录')" :options="disks.concat('/root').map(k=>{return {key:k+'/Configs/qb'}})"/>
                                </div>
                            </div>
                        </li>
                        <li>
                            <div class="input_row">
                                <div class="left">
                                    <span class="tooltip-trigger">
                                        <TipsSvg></TipsSvg>
                                        <div>
                                            <div class="tooltip-text tooltip-top">
                                                <span class="dowload_dir_tip">{{ $gettext("用于放置下载文件的目录。例如：/mnt/sda1/download") }}</span>
                                            </div>
                                        </div>
                                    </span>
                                    <span>{{ $gettext("下载目录：") }}</span>
                                </div>

                                <div class="myinput_wrap">
                                    <editable-select v-model.trim="qBittorrentValue" :title="$gettext('下载目录')" :options="dlDirList"/>
                                </div>
                            </div>
                        </li>
                    </ul>
                </form>

                <!-- transmission配置 -->
                <form @submit.prevent="onNextTransmission" v-if="status == 'Transmission'">
                    <ul>
                        <li>
                            <div class="input_row">
                                <div class="left">
                                    <span class="tooltip-trigger">
                                        <TipsSvg></TipsSvg>
                                        <div>
                                            <div class="tooltip-text tooltip-top">
                                                <span class="dowload_dir_tip">{{ $gettext("用于放置配置文件的目录。例如：/mnt/sda1/Configs/tr；请勿使用 /tmp 或 /var ，以免重启以后任务丢失") }}</span>
                                            </div>
                                        </div>
                                    </span>
                                    <span>{{ $gettext("配置目录：") }}</span>
                                </div>

                                <div class="myinput_wrap">
                                    <editable-select v-model.trim="trConfigPath" :title="$gettext('配置目录')" :options="disks.concat('/root').map(k=>{return {key:k+'/Configs/transmission'}})"/>
                                </div>
                            </div>
                        </li>
                        <li>
                            <div class="input_row">
                                <div class="left">
                                    <span class="tooltip-trigger">
                                        <TipsSvg></TipsSvg>
                                        <div>
                                            <div class="tooltip-text tooltip-top">
                                                <span class="dowload_dir_tip">{{ $gettext("用于放置下载文件的目录。例如：/mnt/sda1/download") }}</span>
                                            </div>
                                        </div>
                                    </span>
                                    <span>{{ $gettext("下载目录：") }}</span>
                                </div>

                                <div class="myinput_wrap">
                                    <editable-select v-model.trim="transmissionValue" :title="$gettext('下载目录')" :options="dlDirList"/>
                                </div>
                            </div>
                        </li>
                    </ul>
                </form>


                <div class="btns">
                    <!-- Aria2启动 -->
                    <button class="cbi-button cbi-button-apply" @click="onNextAria2"
                        v-if="status == 'Aria2'">{{ $gettext("启用") }}</button>
                    <!-- qBittorrent启动 -->
                    <button class="cbi-button cbi-button-apply" @click="onNextqBittorrent"
                        v-if="status == 'qBittorrent'">{{ $gettext("启用") }}</button>
                    <!-- transmission启动 -->
                    <button class="cbi-button cbi-button-apply" @click="onNextTransmission"
                        v-if="status == 'Transmission'">{{ $gettext("启用") }}</button>

                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onClose">{{ $gettext("取消") }}</button>
                </div>
            </div>

            <!-- 下载服务配置向导 -->
            <div class="action" v-else-if="setup == 1">
                <h2 class="title">{{ $gettext("%{status}下载服务配置向导", {status}) }}</h2>
                <div class="finished">
                    <FinishedSvg></FinishedSvg>
                </div>
                <p class="successed">{{ $gettext("配置成功！") }}</p>
                <div class="btns">
                    <button class="cbi-button cbi-button-apply" @click="onFinish">{{ $gettext("确定") }}</button>
                </div>
            </div>
    </action-component>
</template>
<script setup lang="ts">
import { ref, PropType, onMounted, onBeforeMount } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from "/@/components/toast";
import ActionComponent from "/@/components/action/modal.vue"
import EditableSelect from "/@/components/editable-select/index.vue"
import TipsSvg from "/@/components/svg/tips.vue"
import FinishedSvg from "/@/components/svg/finished.vue"

import request from '/@/request';
import appUtils from '/@/utils/app';
const rpcToken = ref("")
const btTracker = ref("")
const configPath = ref("")
const downloadPath = ref("")

const trackerSource = ref('default')
const status = ref('Aria2')
const qbConfigPath = ref("")
const qBittorrentValue = ref("")
const trConfigPath = ref("")
const transmissionValue = ref("")
const dlDirList = ref<{key: string, value?: string}[]>([])
const disks = ref<string[]>([])

const props = defineProps({
    services: {
        type: Object as PropType<GuideDownloadServiceStatus>,
        required: true
    },
    partitionList: {
        type: Array as PropType<string[]>,
        required: true
    },
    defaultTab: {
        type: String as PropType<string>,
        required: false,
    },
    Close: Function
})
const setup = ref(0)

onMounted(() => {
    switch (props.defaultTab) {
        case "aria2":
            status.value = "Aria2";
            break;
        case "qbittorrent":
            status.value = "qBittorrent";
            break;
        case "transmission":
            status.value = "Transmission";
            break;
    }
    dlDirList.value = props.partitionList.map(k=>{return {key:k}})
    disks.value = props.partitionList.filter(k=>k.startsWith("/mnt/")).map(k=>k.replace(/(\/mnt\/[^/]+).*/, '$1'))

    configPath.value = props.services.aria2?.configPath || ""
    const aria2Path = props.services.aria2?.downloadPath || props.services.qbittorrent?.downloadPath || props.services.transmission?.downloadPath
    if (aria2Path) {
        downloadPath.value = aria2Path
    }
    const aria2RPC = props.services.aria2?.rpcToken
    if (aria2RPC) {
        rpcToken.value = aria2RPC
    }

    qbConfigPath.value = props.services.qbittorrent?.configPath || ""
    const qBittorrentPath = props.services.qbittorrent?.downloadPath || aria2Path || props.services.transmission?.downloadPath
    if (qBittorrentPath) {
        qBittorrentValue.value = qBittorrentPath
    }

    trConfigPath.value = props.services.transmission?.configPath || ""
    const transmissionPath = props.services.transmission?.downloadPath || aria2Path || qBittorrentPath
    if (transmissionPath) {
        transmissionValue.value = transmissionPath
    }
})

//  Aria2启动 
const onNextAria2 = () => {
    let cfgPath = configPath.value
    let dlPath = downloadPath.value
    if (cfgPath == null || cfgPath == "" || dlPath == null || dlPath == "") {
        return
    }

    (async ()=>{
        //判断Aria2是否安装
        const is = await appUtils.checkAndInstallApp("app-meta-aria2", "Aria2")
        if (is) {
            const load = Toast.Loading($gettext("配置中..."))
            //配置Aria2
            request.Guide.Aria2Init.POST({ configPath:cfgPath, downloadPath: dlPath, rpcToken: rpcToken.value, btTracker: trackerSource.value == 'add' ? btTracker.value : "" })
                .then(res => {
                    if (res?.data) {
                        if ((res.data.success || 0) == 0) {
                            setup.value = 1
                            return
                        } else if (res.data?.error) {
                            throw res.data.error
                        }
                    }
                    throw $gettext("未知错误")
                })
                .catch(error => Toast.Error(error))
                .finally(() => load.Close())
        }
    })()
}

//  qBittorrent启动 
const onNextqBittorrent = () => {
    let cfgPath = qbConfigPath.value
    let dlPath = qBittorrentValue.value
    if (cfgPath == null || cfgPath == "" || dlPath == null || dlPath == "") {
        return
    }

    (async ()=>{
        //判断qBittorrent是否安装
        const is = await appUtils.checkAndInstallApp("app-meta-qbittorrent", "qBittorrent")
        if (is) {
            const load = Toast.Loading($gettext("配置中..."))
            //配置qbitorrent
            request.Guide.qbitorrentInit.POST({ configPath:cfgPath, downloadPath: dlPath })
                .then(res => {
                    if (res?.data) {
                        if ((res.data.success || 0) == 0) {
                            setup.value = 1
                            return
                        } else if (res.data?.error) {
                            throw res.data.error
                        }
                    }
                    throw $gettext("未知错误")
                })
                .catch(error => Toast.Error(error))
                .finally(() => load.Close())
        }
    })()
}

//  transmission启动 
const onNextTransmission = () => {
    let cfgPath = trConfigPath.value
    let dlPath = transmissionValue.value
    if (cfgPath == null || cfgPath == "" || dlPath == null || dlPath == "") {
        return
    }

    (async ()=>{
        //判断Transmission是否安装
        const is = await appUtils.checkAndInstallApp("app-meta-transmission", "Transmission")
        if (is) {
            const load = Toast.Loading($gettext("配置中..."))
            //配置Transmission
            request.Guide.transmissionInit.POST({ configPath:cfgPath, downloadPath: dlPath })
                .then(res => {
                    if (res?.data) {
                        if ((res.data.success || 0) == 0) {
                            setup.value = 1
                            return
                        } else if (res.data?.error) {
                            throw res.data.error
                        }
                    }
                    throw $gettext("未知错误")
                })
                .catch(error => Toast.Error(error))
                .finally(() => load.Close())
        }
    })()
}

const onClose = (e: Event) => {
    e.preventDefault()
    if (props.Close) {
        props.Close()
    }
}

const onFinish = (e: Event) => {
    e.preventDefault()
    location.reload()
}
</script>


<style lang="scss" scoped>
.action {
    width: 860px;
    max-height: 90%;
    background-color: #fff;
    position: relative;
    z-index: 1000;
    margin: auto;
    padding: 1rem 87px;
    border-radius: 6px;

    p {
        color: #999;
        font-size: 14px;
    }

    input {
        font-size: 14px;
        font-family: PingFangSC-Regular, PingFang SC;
    }

    h2.title {
        width: 100%;
        color: #1e1e1e;
        font-size: 22px;
        font-family: PingFangSC-Medium, PingFang SC;
        padding: 0;
        margin: 0;
        text-align: center;
    }

    span {
        font-size: 14px;
        font-family: PingFangSC-Medium, PingFang SC;
        color: rgba(0, 0, 0, 0.83);
        font-weight: 700;
    }

    form {
        label {
            width: 100%;

            input,
            select {
                height: 100%;
                font-size: 14px;
            }
        }
    }
    .myinput_wrap, .RPC_input  {
        width: 85%;
    }
    .myinput_wrap textarea {
        width: 100%;
        padding: 2px 3px;
        border: 1px solid #dee2e6;
        border-radius: 0.25rem;
    }
    .input_row {
        margin: 16px 0;
        display: flex;
        justify-content: left;
        align-items: center;
        .radios {
            margin-right: 10px;

            input {
                cursor: pointer;
            }

            label {
                cursor: pointer;
            }

        }
    }

    .Tracker {
        label {
            margin-right: 10px;
            cursor: pointer;
        }
    }

    .Tracker_input {
        padding: 6px 2px;
    }

    .btns {
        width: 100%;
        margin: 0 auto;
        margin-top: 42px;

        button {
            display: block;
            width: 100% !important;
            margin: 0.5rem 0;
        }
    }

    .tooltip-trigger {
        position: relative;
        display: inline-block;
        cursor: help;
        margin-right: 6px;
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

    .tooltip-trigger .tooltip-text .dowload_dir_tip {
        min-width: 14rem;
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

    .tooltip-bottom {
        top: 100%;
        left: 50%;
        margin-top: 5px;
        /* tooltip 与触发元素的距离 - 5px */
        transform: translate(-50%, 0);

        .dowload_rpc_tip {
            min-width: 10rem;
            display: inline-block;
        }
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

    .successed {
        text-align: center;
        font-size: 14px;
    }

    .finished {
        display: flex;
        justify-content: center;
        margin: 80px;
        margin-bottom: 28px;
    }
}

.left {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    width: 110px;
    flex: none;
}

.select-editable {
    position: relative;
    // background-color: white;
    border: 1px solid #dee2e6;
    border-radius: 0.25rem;
    margin: 0.25rem 0.1rem;
}

.select-editable select {
    top: 0px;
    left: 0px;
    font-size: 14px;
    border: none;
    width: 100%;
    margin: 0;
}

.select-editable input {
    position: absolute;
    top: -4px;
    left: 0px;
    width: 95%;
    padding: 1px;
    font-size: 14px;
    border: none;
}

.select-editable select:focus,
.select-editable input:focus {
    outline: none;
}

::placeholder {
    color: #999;
}

</style>
<style lang="scss" scoped>

@media screen and (max-width: 500px) {
    .action {
        width: 100%;
        .input_row {
            display: block;
            .myinput_wrap, .RPC_input {
                width: 100%;
            }
        }
    }
}

</style>
