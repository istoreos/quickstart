<template>
    <action-component :Close="Close" :type="1">
            <div class="action" v-if="setup == 0">
                <h2 class="title">{{ $gettext("Docker迁移向导") }}</h2>
                <p class="desc">{{ $gettext("当系统根目录空间不足时，可将Docker根目录迁移到外置硬盘，以保证系统的正常运行（目标分区不支持NTFS，FAT等文件系统）") }}</p>
                <div class="roots">
                    <span class="roots_tit">{{ $gettext("Docker根目录：") }}</span>
                    <span class="root">{{ docker?.path }}</span>
                </div>

                <div class="move">
                    <span class="roots_tit">{{ $gettext("迁移到：") }}</span>

                    <div v-if="dockerList?.partitionList?.length">
                        <form @submit.prevent="onNext">
                            <label>
                                <div class="select-editable">
                                    <select v-model.trim="backupValue">
                                        <option selected :value="null">{{ $gettext("请选择迁移路径") }}</option>
                                        <option :value="item" v-for="(item, i) in dockerList?.partitionList" :key="i">{{
                                                item
                                        }}</option>
                                        <option value="useInput">{{ $gettext("- -自定义- -") }}</option>
                                    </select>
                                    <input type="text" v-model.trim="useBackupValue" required
                                        v-if="backupValue == 'useInput'" :placeholder="$gettext('请输入迁移路径')" />
                                </div>
                            </label>
                        </form>
                    </div>

                    <div class="tips" v-else-if="dockerList">
                        <div class="tips_content">
                            <HintSvg></HintSvg>
                            <span class="tip">{{ $gettext("检测到您还没有挂载外置硬盘或分区小于8GB，需要您接上硬盘并格式化或手动挂载硬盘后，再执行Docker迁移向导，将Docker迁移到目标硬盘。") }}</span>
                        </div>
                    </div>
                </div>

                <div class="btns" v-if="dockerList?.partitionList?.length">
                    <button class="cbi-button cbi-button-apply" @click="onNext">{{ $gettext("确定") }}</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" type="button"
                        @click="onClose">{{ $gettext("取消") }}</button>
                </div>
                <div class="btns" v-else>
                    <button class="cbi-button cbi-button-apply" @click="onClose">{{ $gettext("确定") }}</button>
                </div>
            </div>
            <div class="action docker_success" v-else-if="setup == 1">
                <h2 class="title">{{ $gettext("Docker迁移向导") }}</h2>
                <div class="finished">
                    <FinishedSvg></FinishedSvg>
                </div>
                <p class="successed">{{ $gettext("迁移成功！") }}</p>
                <div class="btns">
                    <button class="cbi-button cbi-button-apply" @click="onFinish">{{ $gettext("确定") }}</button>
                </div>
            </div>
            <div class="action docker_download" v-else-if="setup == 2">
                <h2 class="title">{{ $gettext("Docker迁移向导") }}</h2>
                <div class="finished">
                    <tipsyellow></tipsyellow>
                </div>
                <p class="successed">{{ $gettext("该目标路径不为空") }}</p>
                <div class="docker_moves">
                    <div class="moves change">
                        <input type="radio" id="move" name="moves" v-model="overwriteDir" value="">
                        <label for="move">{{ $gettext("更换目录（不覆盖目标路径，仅将Docker目录修改为目标路径）") }}</label>
                    </div>
                    <div class="moves">
                        <input type="radio" id="cover" name="moves" v-model="overwriteDir" value="true">
                        <label for="cover">{{ $gettext("覆盖迁移（覆盖目标路径，继续迁移会清空该目标路径下的文件）") }}</label>
                    </div>
                </div>

                <div class="btns">
                    <button v-if="allowContinue" class="cbi-button cbi-button-apply" @click="onForce">{{ $gettext("确定") }}</button>
                    <button class="cbi-button cbi-button-apply" @click="onBack">{{ $gettext("返回") }}</button>
                    <button v-if="!allowContinue" class="cbi-button cbi-button-remove app-btn app-back" type="button"
                        @click="onFinish">{{ $gettext("取消") }}</button>
                </div>
            </div>

    </action-component>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from "/@/components/toast";
import ActionComponent from "/@/components/action/modal.vue"
import FinishedSvg from "/@/components/svg/finished.vue"
import HintSvg from "/@/components/svg/hint.vue"
import tipsyellow from "/@/components/svg/tipsyellow.vue"
import { computed, PropType } from 'vue';
import request from '/@/request';
const docker = ref<GuideDockerStatus>()
const dockerList = ref<GuideDockerPartitionList>()
const setup = ref(0)
const backupValue = ref("null")
const useBackupValue = ref("")
const info = ref<NasDiskStatus>()
const allowContinue = ref(false)
const overwriteDir = ref("")


const props = defineProps({
    rootPath: {
        type: String,
        required: true
    },
    Close: Function,
})
const getData = () => {
    //请求硬盘状态
    request.Nas.Disk.Status.GET().then(res => {
        if (res?.data.result) {
            info.value = res?.data.result
        }
    })

    //请求docker状态
    request.Guide.DockerStatus.GET().then(res => {
        if (res?.data?.result) {
            const result = res.data.result
            docker.value = result
        }
    })
    //请求docker路径列表接口
    request.Guide.DockerPartitionList.GET().then(res => {
        //setup.value = 0
        if (res?.data?.result) {
            const result = res.data.result
            dockerList.value = result
        }
    })
}
getData()

const doTransfer = (force: boolean) => {
    let _value = backupValue.value
    if (_value == "useInput") {
        _value = useBackupValue.value
    }
    if (_value == null || _value == "null" || _value == "") {
        return
    }
    const load = Toast.Loading($gettext("正在迁移中..."))
    //迁移docker接口
    const Success = request.Guide.DockerTransfer.POST({ path: _value, force: force, overwriteDir: !!overwriteDir.value })
        .then(res => {
            if (res?.data) {
                if ((res.data.success || 0) == 0) {
                    if (res.data.result?.emptyPathWarning) {
                        allowContinue.value = true
                        setup.value = 2
                        return
                    }
                    setup.value = 1
                    return
                } else if (res.data.error) {
                    throw res.data.error
                }
            }
            throw $gettext("未知错误")
        })
        .catch(error => {
            Toast.Error(error)
        })
        .finally(() => load.Close())
}

const onNext = () => {
    allowContinue.value = false
    doTransfer(false)
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
const onBack = (e: Event) => {
    e.preventDefault()
    setup.value = 0
}
const onForce = (e: Event) => {
    e.preventDefault()
    doTransfer(true)
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
    overflow: auto;
    padding: 1rem 87px;
    border-radius: 6px;

    h2.title {
        width: 100%;
        display: block;
        color: #1e1e1e;
        font-size: 22px;
        padding: 0;
        margin: 0;
        text-align: center;
    }

    .roots {
        display: flex;
        max-width: 342px;
        align-items: center;
        margin-top: 32px;
        margin-bottom: 16px;

        .root {
            color: rgba(0, 0, 0, 0.83);
            font-size: 14px;
            text-align: center;
        }
    }

    .move {
        display: flex;
        justify-content: left;
        align-items: center;
    }

    .change {
        width: 678px;
    }

    .desc {
        width: 100%;
        display: block;
        font-size: 1.2em;
        padding: 0;
        margin: 1rem 0;
        margin-top: 32px;
        font-size: 14px;
        font-family: PingFangSC-Medium, PingFang SC;
        color: rgba(0, 0, 0, 0.83);
    }

    form {
        width: 100%;
        display: block;
    }

    .tips {
        width: 477px;

        .tip {
            color: #faad14;
            padding-left: 6px;
        }
    }

    .btns {
        width: 100%;
        margin: 0 auto;
        margin-top: 104px;

        button {
            display: block;
            width: 100% !important;
            margin-left: 0;
            margin-right: 0;
        }
    }

    .roots_tit {
        color: rgba(0, 0, 0, 0.83);
        font-size: 14px;
        font-weight: 700;
        width: 118px;
        text-align: right;
        flex: none;
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

    .docker_moves {
        text-align: center;

        .moves {
            margin-top: 10px;

            input {
                cursor: pointer;
            }

            label {
                margin-left: 10px;
                cursor: pointer;
            }

        }
    }
}

.select-editable {
    position: relative;
    // background-color: white;
    border: solid grey 1px;
    width: 438px;
    height: 34px;
}

.select-editable select {
    position: absolute;
    top: 0px;
    left: 0px;
    font-size: 14px;
    border: none;
    width: 100%;
    height: 100%;
    margin: 0;
}

.select-editable input {
    position: absolute;
    top: 0px;
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

@media screen and (max-width: 800px) {
    .action {
        width: 100%;
    }

    .docker_download {
        width: 80%;
    }
}

</style>