<template>
    <div class="actioner-container">
        <div class="actioner-container_header">
            <span>{{ $gettext("域名配置向导") }}</span>
        </div>
        <div class="actioner-container_body">
            <div class="label-item">
                <label>
                    <input type="radio" v-model="active" value="ddnsto">
                    <span>{{ $gettext("DDNSTO") }}</span>
                </label>
                <p class="label_info">{{ $gettext("DDNSTO 是一个不需要公网IP也可以在外网访问的穿透域名服务，一个浏览器搞定内网穿透，远程访问Openwrt、远程终端、远程桌面...") }}</p>
            </div>
            <div class="label-item">
                <label>
                    <input type="radio" v-model="active" value="ali">
                    <span>{{ $gettext("阿里云") }}</span>
                </label>
                <p class="label_info">
                    {{ $gettext("为拥有动态IP的主机配置一个固定的可访问域名") }}
                </p>
            </div>
            <div class="label-item">
                <label>
                    <input type="radio" v-model="active" value="dnspod">
                    <span>{{ $gettext("Dnspod") }}</span>
                </label>
                <p class="label_info">
                    {{ $gettext("为拥有动态IP的主机配置一个固定的可访问域名") }}
                </p>
            </div>
            <div class="label-item">
                <label>
                    <input type="radio" v-model="active" value="oray">
                    <span>{{ $gettext("花生壳") }}</span>
                </label>
                <p class="label_info">
                    {{ $gettext("为拥有动态IP的主机配置一个固定的可访问域名") }}
                </p>
            </div>
        </div>
        <div class="actioner-container_footer">
            <div class="close" @click="onClose">{{ $gettext("取消") }}</div>
            <div class="next" @click="onNext">{{ $gettext("下一步") }}</div>
        </div>
    </div>
</template>
<script setup lang="ts">
import { PropType, ref } from "vue";
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

const props = defineProps({
    onSetup: {
        type: Function as PropType<(v?: string) => void>,
        required: true
    },
    active: {
        type: String,
        default: "ddnsto"
    }
})
const emit = defineEmits([
    "update:active"
])
const onClose = () => {
    props.onSetup()
}
const active = ref(props.active)
const onNext = () => {
    emit("update:active", active.value)
    switch (active.value) {
        case "ddnsto":
            props.onSetup('ddnsto')
            break
        case "ali":
            props.onSetup('ddns-ali')
            break
        case "dnspod":
            props.onSetup('ddns-dnspod')
            break
        case "oray":
            props.onSetup('ddns-oray')
            break

    }
}


</script>
<style lang="scss" scoped>
h3 {
    text-align: center;
    margin-bottom: 20px;
}

.label-item {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    margin: 1rem 0;
    padding: 0 30px;

    label {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        width: 100%;
        height: 26px;
        line-height: 26px;
        cursor: pointer;

        input[type="radio"] {
            top: 0;
            right: 0;
            vertical-align: middle;
        }

        span {
            font-size: 14px;
            font-family: PingFangSC-Regular, PingFang SC;
            font-weight: 400;
            color: rgba(0, 0, 0, 0.83);
            display: inline-block;
            margin-left: 10px;
        }
    }

    p.label_info {
        color: #999;
        font-size: 12px;
        padding-left: 24px;
        line-height: 20px;
    }



    .label-item_key {
        display: flex;
        flex-wrap: wrap;
        align-items: center;


        .ddnsto_serve {
            flex: 0 0 100%;
            display: flex;
            justify-content: space-between;
            margin-bottom: 14px;
        }

        .ddnsto_serve_item {
            flex: 0 0 100%;
            display: flex;
            justify-content: space-between;
        }
    }
}
</style>


