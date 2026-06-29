<template>
    <action-component :Close="Close" :type="1">
        <template v-if="setup == 0">
            <form class="actioner-dns" @submit.prevent="onSumbit">
                <div class="actioner-dns_header">
                    <span>{{ $gettext("软件源配置") }}</span>
                </div>
                <div class="actioner-dns_body">

                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("当前软件源") }}</span>
                        </div>
                        <div class="label-item_value">
                            <p class="item_info">{{ softsourceIfo?.name }}</p>
                        </div>
                    </div>

                    <div class="label-item">
                        <div class="label-item_key">
                            <span>{{ $gettext("切换软件源") }}</span>
                        </div>
                        <div class="label-item_value">

                            <select name="" id="" v-model.trim="softwareSourceValue">
                                <option selected="true" value="">{{ $gettext("请选择软件源") }}</option>
                                <option :value="item.identity" v-for="(item, i) in softsourceListIfo?.softSourceList"
                                    :key="i">{{ item.name }}
                                </option>
                            </select>
                        </div>
                    </div>
                </div>
                <div class="actioner-dns_footer">
                    <button class="cbi-button cbi-button-apply app-btn"
                        :disabled="softwareSourceValue == ''">{{ $gettext("确认") }}</button>
                    <button class="cbi-button cbi-button-remove app-btn app-back" @click="onClose">{{ $gettext("取消") }}</button>

                </div>
            </form>
        </template>
        <template v-if="setup == 1">
            <form class="actioner-dns">
                <div class="actioner-dns_header">
                    <span class="softsource_tit">{{ $gettext("软件源配置") }}</span>
                </div>
                <div class="actioner-dns_body">

                    <div class="finished">
                        <FinishedSvg></FinishedSvg>
                    </div>
                    <p class="successed">{{ $gettext("配置成功！") }}</p>
                    <div class="btns ">
                        <button class="cbi-button cbi-button-apply softsource_successed" @click="onFinish">{{ $gettext("确定") }}</button>
                    </div>
                </div>
            </form>
        </template>
    </action-component>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from "/@/components/toast";
import ActionComponent from "/@/components/action/modal.vue"
import request from "/@/request";
import FinishedSvg from "/@/components/svg/finished.vue"
import itemVue from "../action-disk/item.vue";
const props = defineProps({
    Close: {
        type: Function,
        required: true
    },
})
const setup = ref(0)
const softwareSourceValue = ref("")
const softsourceIfo = ref<GuideSoftSourceInfo>()
const softsourceListIfo = ref<GuideSoftSourceList>()

const getData = () => {

    //请求获取软件源配置列表
    request.Guide.SoftSourceList.GET().then(res => {
        if (res?.data?.result) {
            const result = res?.data?.result
            softsourceListIfo.value = result
        }
    })
        //请求获取当前软件源配置
        .then(() => request.Guide.GetSoftSource.GET())
        .then(res => {
            if (res?.data?.result) {
                const result = res.data.result
                softsourceIfo.value = result.softSource
                if (softsourceListIfo.value?.softSourceList.find(e => e.identity == result.softSource.identity)) {
                    softwareSourceValue.value = result.softSource.identity
                }
            }
        })
}
getData()

const onClose = (e: Event) => {
    e.preventDefault();
    if (props.Close) {
        props.Close()
    }
}

const onSumbit = (e: Event) => {
    const load = Toast.Loading($gettext("正在切换中..."))
    request.Guide.SoftSource.POST({ softSourceIdentity: softwareSourceValue.value }).then(res => {
        if (res?.data) {
            if ((res.data.success || 0) == 0) {
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
        }).finally(() => load.Close())

}
const onFinish = (e: Event) => {
    e.preventDefault()
    location.reload()
}
</script>
<style lang="scss" scoped>
.actioner-dns {
    width: 800px;
    background-color: #fff;
    position: relative;
    z-index: 99999;
    margin: auto;
    overflow: auto;

    .actioner-dns_header {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        padding: 1rem;
        font-size: 2em;
        border-bottom: 1px solid #eee;
    }

    .actioner-dns_body {
        padding: 1rem;
        min-height: 50vh;

        .label-item {
            width: 100%;
            margin: 1rem 0;

            .label-item_key {
                width: 100%;
                font-size: 12px;
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
                    height: 36px;
                }

                // textarea {
                //     width: 100%;
                // }
            }
        }

        .label-message {
            width: 100%;
            text-align: left;
            font-size: 14px;
            color: #f00;
            text-align: center;
        }
    }

    .config-message {
        width: 100%;
        min-height: inherit;
        height: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: center;
        font-size: 2em;
    }

    .actioner-dns_footer {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        justify-content: flex-end;
        padding: 1rem;
        font-size: 2em;
        border-top: 1px solid #eee;

        button {
            display: inline-block;
            width: 100px !important;
            margin: 0;
            margin-left: 1rem;
        }
    }

    .select-editable {
        position: relative;
        // background-color: white;
        border: solid grey 1px;
        width: 100%;
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

}

.successed {
    text-align: center;
    font-size: 14px;
    margin-bottom: 104px;
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

.btns {
    text-align: center;
}

.item_info {
    margin-left: 10px;
}

.softsource_tit {
    margin: 0 auto;
}

.softsource_successed {
    width: 20% !important;
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 1400px) {
    .actioner-dns {
        .actioner-dns_body {
            min-height: 34vh;
        }
    }
}

@media screen and (max-width: 860px) {
    .actioner-dns {
        width: 100%;
    }
}

</style>