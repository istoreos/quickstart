<template>
    <form class="form-container" @submit.prevent="hanldeSubmit">
        <div class="form-item">
            <div class="label-name">
                <span>{{ $gettext("启用Wi-Fi") }}</span>
            </div>
            <div class="label-value switch_label">
                <!-- PC端显示的单选按钮 -->
                <div class="label-flex pc-radio">
                    <label>
                        <input type="radio" :value="false" v-model="iface.disabled" :disabled="disabled" />
                        {{ $gettext("开启") }}
                    </label>
                    <label>
                        <input type="radio" :value="true" v-model="iface.disabled" :disabled="disabled" />
                        {{ $gettext("关闭") }}
                    </label>
                </div>

                <!-- 移动端显示的开关按钮 -->
                <div class="mobile-switch" @click="toggleSwitch">
                    <div :class="['switch-core', { 'is-checked': !iface.disabled, 'is-disabled': disabled }]">
                        <div class="switch-button"></div>
                    </div>
                </div>
            </div>
        </div>
        <div class="form-item" v-if="!iface.isGuest">
            <div class="label-name">
                <span>{{ $gettext("发射功率") }}</span>
            </div>
            <div class="label-value">
                <select v-model="iface.txpower" :disabled="disabled" @change="handleSubmitPower">
                    <option :value="100">{{ $gettext("最大") }}</option>
                    <option :value="70">{{ $gettext("高") }}</option>
                    <option :value="50">{{ $gettext("中") }}</option>
                    <option :value="30">{{ $gettext("低") }}</option>
                </select>
                <div class="select-arrow" v-if="!disabled"></div>
            </div>
        </div>
        <div class="form-item">
            <div class="label-name">
                <span>{{ $gettext("Wi-Fi名称（SSID）") }}</span>
            </div>
            <div class="label-value">
                <input v-model="iface.ssid" :disabled="disabled" :placeholder="$gettext('请输入WIFI名称')" />
            </div>
        </div>
        <div class="form-item">
            <div class="label-name">
                <span>{{ $gettext("Wi-Fi 安全性") }}</span>
            </div>
            <div class="label-value">
                <select v-model="iface.encryption" :disabled="disabled">
                    <option :value="label" v-for="label in iface.encryptSelects">{{ label }}</option>
                </select>
                <div class="select-arrow" v-if="!disabled"></div>
            </div>
        </div>
        <div class="form-item">
            <div class="label-name">
                <span>{{ $gettext("Wi-Fi 密码") }}</span>
            </div>
            <div class="label-value">
                <input v-model="iface.key" class="password_input" :type="seeStatus ? 'password' : ''"
                    :disabled="disabled" :placeholder="$gettext('请输入WIFI密码')" />
                <div v-if="!disabled" @click="seeStatus = !seeStatus">
                    <!-- 闭眼图标（隐藏密码） -->
                    <svg class="seeIcon" viewBox="0 0 22 22" xmlns="http://www.w3.org/2000/svg" v-if="seeStatus">
                        <path
                            d="M12 6c3.79 0 7.17 2.13 8.82 5.5-.59 1.22-1.42 2.27-2.41 3.12l1.41 1.41c1.39-1.23 2.49-2.77 3.18-4.53C21.27 7.11 17 4 12 4c-1.27 0-2.49.2-3.64.57l1.65 1.65C10.66 6.09 11.32 6 12 6zm-1.07 1.14L13 9.21c.57.25 1.03.71 1.28 1.28l2.07 2.07c.08-.34.14-.7.14-1.07C16.5 9.01 14.48 7 12 7c-.37 0-.72.05-1.07.14zM2.01 3.87l2.68 2.68C3.06 7.83 1.77 9.53 1 11.5 2.73 15.89 7 19 12 19c1.52 0 2.98-.29 4.32-.82l3.42 3.42 1.41-1.41L3.42 2.45 2.01 3.87zm7.5 7.5l2.61 2.61c-.04.01-.08.02-.12.02-1.38 0-2.5-1.12-2.5-2.5 0-.05.01-.08.01-.13zm-3.4-3.4l1.75 1.75c-.23.55-.36 1.15-.36 1.78 0 2.48 2.02 4.5 4.5 4.5.63 0 1.23-.13 1.77-.36l.98.98c-.88.24-1.8.38-2.75.38-3.79 0-7.17-2.13-8.82-5.5.7-1.43 1.72-2.61 2.93-3.53z"
                            fill="currentColor" />
                    </svg>
                    <!-- 睁眼图标（显示密码） -->
                    <svg class="seeIcon" viewBox="0 0 22 22" xmlns="http://www.w3.org/2000/svg" v-else>
                        <path
                            d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"
                            fill="currentColor" />
                    </svg>
                </div>
            </div>
        </div>
        <div class="form-item">
            <div class="label-name">
                <span>{{ $gettext("SSID 可见性") }}</span>
            </div>
            <div class="label-value">
                <select v-model="iface.hidden" :disabled="disabled">
                    <option :value="false">{{ $gettext("显示") }}</option>
                    <option :value="true">{{ $gettext("隐藏") }}</option>
                </select>
                <div class="select-arrow" v-if="!disabled"></div>
            </div>
        </div>
        <div class="form-item" v-if="!iface.isGuest">
            <div class="label-name">
                <span>{{ $gettext("无线模式") }}</span>
            </div>
            <div class="label-value">
                <select v-model="iface.hwmode" :disabled="disabled">
                    <option :value="label" v-for="label in iface.hwmodeSelects">{{ label }}</option>
                    <!-- <option v-if="iface.band === '2g'" value="11n/ax">
                        11n/ax
                    </option>
                    <option v-if="iface.band === '2g'" value="11g/n/ax">
                        11g/n/ax
                    </option>
                    <option v-if="iface.band === '2g'" value="11b/g/n/ax">
                        11b/g/n/ax
                    </option>
                    <option v-if="iface.band === '2g'" value="11b/g/n">
                        11b/g/n
                    </option>

                    <option v-if="iface.band === '5g'" value="11ac/ax">
                        11ac/ax
                    </option>
                    <option v-if="iface.band === '5g'" value="11n/ac/ax">
                        11n/ac/ax
                    </option>
                    <option v-if="iface.band === '5g'" value="11a/n/ac/ax">
                        11a/n/ac/ax
                    </option>
                    <option v-if="iface.band === '5g'" value="11a/n/ac">
                        11b/g/n
                    </option> -->
                </select>
                <div class="select-arrow" v-if="!disabled"></div>
            </div>
        </div>
        <div class="form-item" v-if="!iface.isGuest">
            <div class="label-name">
                <span>{{ $gettext("频宽") }}</span>
            </div>
            <div class="label-value">
                <select v-model="iface.htmode" :disabled="disabled">
                    <option value="20">20 MHz</option>
                    <option value="40">40 MHz</option>
                    <option v-if="iface.band === '2g'" value="auto">
                        20/40 MHz
                    </option>
                    <option v-if="iface.band === '5g'" value="80">
                        80 MHz
                    </option>
                    <option v-if="iface.band === '5g'" value="160">
                        160 MHz
                    </option>
                </select>
                <div class="select-arrow" v-if="!disabled"></div>
            </div>
        </div>
        <div class="form-item" v-if="!iface.isGuest">
            <div class="label-name">
                <span>{{ $gettext("信道") }}</span>
            </div>
            <div class="label-value">
                <select v-model.number="iface.channel" v-if="iface.band === '2g'" :disabled="disabled">
                    <option :value="0">{{ $gettext("自动") }}</option>
                    <option :value="i" v-for="i in 13">{{ i }}</option>
                </select>
                <select v-model.number="iface.channel" v-if="iface.band === '5g'" :disabled="disabled">
                    <option :value="0">{{ $gettext("自动") }}</option>
                    <option :value="36">36</option>
                    <option :value="40">40</option>
                    <option :value="44">44</option>
                    <option :value="48">48</option>
                    <option :value="52">52(DFS)</option>
                    <option :value="56">56(DFS)</option>
                    <option :value="60">60(DFS)</option>
                    <option :value="64">64(DFS)</option>
                    <option :value="149">149</option>
                    <option :value="153">153</option>
                    <option :value="157">157</option>
                    <option :value="161">161</option>
                </select>
                <div class="select-arrow" v-if="!disabled"></div>
            </div>
        </div>
        <div class="form-item" v-if="!iface.isGuest">
            <div class="label-name">
                <span>{{ $gettext("网络") }}</span>
            </div>
            <div class="label-value">
                <input type="text" :placeholder="$gettext('请配置网络名称')" required v-model="iface.network" :disabled="disabled">
            </div>
        </div>
        <div class="label-btns">
            <button class="btn primary-btn" :disabled="disabled" @click="hanldeSubmit">{{ $gettext("保存配置") }}</button>
        </div>
    </form>
</template>
<script setup lang="ts">
import Toast from "/@/components/toast";
import request from '/@/request';
import { inject, Ref, ref, watch } from 'vue'
import { useGettext, formatNumber } from '/@/plugins/i18n'
const { $gettext, $ngettext } = useGettext()
const props = defineProps<{
    data: Iface
}>()
const iface = ref(Object.assign({}, props.data))
const emit = defineEmits<{
    (e: 'getData'): Promise<void>;
}>();
const disabled = inject<Ref<boolean>>("disabled")!
const seeStatus = ref<boolean>(true)
const toggleSwitch = () => {
    if (!disabled.value) {
        iface.value.disabled = !iface.value.disabled;
    }
};
watch(
    () => iface.value.disabled, // 监听 iface.disabled
    (newValue) => {
        hanldeSubmitEnable()
    }
)
// 修改发射功率
const handleSubmitPower = async () => {
    if (iface.value.txpower === props.data.txpower) {
        return
    }
    const load = Toast.Loading('配置中...')
    try {
        const { data } = await request.Quickwifi.Power.POST({
            device: iface.value.device || "",
            txpower: iface.value.txpower || 0,
        })
        const { error, success } = data
        if (error) {
            throw error
        }
        if (success == null || success == 0) {
            Toast.Success("配置完成")
        }
    } catch (error) {
        Toast.Error("配置失败，请重试");
        throw error
    } finally {
        load.Close()
    }
}
// 是否启用
const hanldeSubmitEnable = async () => {
    // if (iface.value.disabled === props.data.disabled) {
    //     return;
    // }
    const load = Toast.Loading('配置中...'); // 开始加载

    try {
        const { data } = await request.Quickwifi.Switch.POST({
            ifaceName: iface.value.ifaceName || "",
            enable: !iface.value.disabled,
        });

        const { error, success } = data;
        if (error) {
            throw error; // 如果有错误，抛出
        }
        if (success == null || success == 0) {
            Toast.Success("配置完成")
        }
    } catch (error) {
        console.error("请求出错:", error);
        Toast.Error("配置失败，请重试");
        throw error; // 继续抛出错误（可选）
    } finally {
        load.Close(); 
    }
};
// 提交配置
const handleSubmitIface = async () => {
    // 先创建副本以避免直接修改原对象
    const ifaceCopy = { ...iface.value };
    // 删除不需要的属性
    delete ifaceCopy.encryptSelects;
    delete ifaceCopy.hwmodeSelects;
    delete ifaceCopy.disabled;
    delete ifaceCopy.txpower;
    const { data } = await request.Quickwifi.Edit.POST(ifaceCopy as Ifaces)
    // const { data } = await request.Quickwifi.Edit.POST(iface.value as Ifaces)
    const { error, success } = data
    if (error) {
        throw error
    }
    if (success == null || success == 0) {
        Toast.Success("配置完成")
    }
}
// 应用配置
const hanldeSubmit = async () => {
    if (disabled.value) {
        return
    }
    disabled.value = true
    const load = Toast.Loading('配置中...')
    try {
        // await handleSubmitPower()
        await handleSubmitIface()
        // await hanldeSubmitEnable()
        await emit("getData")
    } catch (error) {
        const errMsg = Toast.Error(`${error}`)
        setTimeout(() => {
            errMsg.Close()
        }, 2000)
    } finally {
        load.Close()
        disabled.value = false
    }
}
</script>
<style lang="scss" scoped>
button {
    outline: none;
    cursor: pointer;
    border: none;
}

/* PC端样式 - 保持不变 */
.pc-radio {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
}

.label-flex.pc-radio label {
    width: 100px;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    cursor: pointer;
    color: #666;
}

.label-flex.pc-radio input[type="radio"] {
    margin: 0;
    margin-right: 4px;
    top: 0;
}

/* 移动端开关按钮样式 - 默认隐藏 */
.mobile-switch {
    display: none;
    align-items: center;
}

.switch-core {
    position: relative;
    width: 50px;
    // height: 20px;
    border: 1px solid #dcdfe6;
    outline: none;
    border-radius: 20px;
    box-sizing: border-box;
    background: #dcdfe6;
    cursor: pointer;
    transition: border-color .3s, background-color .3s;
    display: inline-block;
}

.switch-core.is-checked {
    border-color: #409eff;
    background-color: #409eff;
}

.switch-core.is-disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.switch-button {
    position: absolute;
    top: 1px;
    left: 1px;
    border-radius: 100%;
    transition: all .3s;
    width: 16px;
    height: 16px;
    background-color: #fff;
}

.switch-core.is-checked .switch-button {
    transform: translateX(20px);
}

.switch-label {
    font-size: 14px;
    color: #999;
}

.switch-label.active {
    color: #409eff;
}

form.form-container {
    display: block;
    width: 100%;
    padding: 0 1rem;
    margin-top: 50px;

    .label-name {
        display: block;
        width: 100%;
        margin-bottom: 0.5rem;
        color: var(--item-label_key-span-color);

        span {
            &::before {
                content: "*";
                color: #f56c6c;
                margin-right: 4px;
                width: 10px;
                display: inline-block;
                vertical-align: middle;
            }
        }
    }

    .label-value {
        display: block;
        width: 100%;
        margin-bottom: 1rem;

        // padding-left: 10px;
        input,
        select {
            display: block;
            width: 100%;
            height: 42px;
            background: none;
            border: 1px solid #c2c2c2;
            color: var(--item-label_key-span-color);
            font-size: 14px;

            >option {
                color: #8898aa;
            }

            &:focus {
                transition: 0.2s;
                border: 1px solid #418dfe;
            }
        }

        select {
            border-radius: 3px;
            padding: 0 10px;
        }

        input {
            border-left: none !important;
            border-right: none !important;
            border-top: none !important;
            box-shadow: none !important;
            padding: 0 10px;
        }

        input[type="checkbox"],
        input[type="radio"] {
            width: auto;
        }

        input[type="radio"] {
            margin: 0;
            margin-right: 4px;
            top: 0;
        }

        input:disabled {
            background-color: #eee;
            border: 1px solid #c2c2c2;
            border-radius: 3px;
        }

        input {
            &::placeholder {
                color: var(--item-label_value-span-color);
                opacity: 0.54;
                font-size: 14px;
            }

            &:-ms-input-placeholder {
                color: var(--item-label_value-span-color);
                opacity: 0.54;
                font-size: 14px;
            }

            &::-ms-input-placeholder {
                color: var(--item-label_value-span-color);
                opacity: 0.54;
                font-size: 14px;
            }
        }
    }

    .label-btns {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        justify-content: flex-end;
    }

    .label-msg {
        display: block;
        width: 100%;
        color: #ff3b3b;
        font-size: 14px;
    }

    .label-msg.warning {
        color: #f9ad1e;
    }

    .label-flex {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;

        label {
            width: 100px;
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            cursor: pointer;
            color: #666;
        }
    }
}

.form-item {
    display: flex;
    align-items: center;
    height: 55px;

    .label-name {
        width: 200px !important;
    }

    .label-value {
        width: 300px !important;
        padding-top: 10px;
        position: relative;
        display: flex !important;
        align-items: center;
    }
}

p {
    font-size: 1em;
    color: #999;
    line-height: 26px;
    text-align: left;
    margin-bottom: 1rem;
}

.label-btns {
    width: 500px !important;
    margin-top: 1rem;

    .btn {
        width: 300px !important;
        text-align: center;
        border-radius: 32px;

        &:hover {
            background: #5279f7;
            transition: 0.3;
        }
    }

    .primary-btn {
        border: none;
        background: #5279f7;
        color: #fff;
        margin-bottom: 10px;


        &:hover {
            opacity: 0.9;
            transition: 0.3;
        }
    }
}

select:disabled {
    background-color: #eee !important;
    border: 1px solid #c2c2c2 !important;
}

.seeIcon {
    width: 22px;
    height: 22px;
    position: absolute;
    cursor: pointer;
    z-index: 1;
    right: 6px;
    top: 50%;
    transform: translateY(-50%) scale(1);
    transition: all 0.3s ease;
    transform-origin: center;
}

.seeIcon:hover {
    transform: translateY(-50%) scale(1.1);
}

/* 移动端适配 */
@media only screen and (max-width: 1050px) {

    form.form-container {
        padding: 0;
        margin-top: -16px;

        .form-item {
            position: relative;
            height: auto;
            margin-bottom: 0;
            height: 50px;
            padding-top: 6px;
            border-bottom: 1px solid rgba(0, 0, 0, 0.16) !important;

            .label-name {
                width: 100% !important;
                margin-bottom: 0;
                font-size: 14px;

                >span {
                    color: var(--item-label_key-span-color);
                }
            }

            .label-value {
                width: 100% !important;
                margin-bottom: 0;
                padding-top: 0;

                input,
                select {
                    height: 40px;
                    font-size: 14px;
                }

                .password_input {
                    padding-right: 24px;
                }

                input {
                    border: none;
                    text-align: right;
                    padding: 0;
                }

                select:disabled {
                    border: none !important;
                }

                select {
                    padding-right: 16px !important;
                    /* 去除边框 */
                    border: none;
                    /* 去除默认外观 */
                    appearance: none;
                    -webkit-appearance: none;
                    -moz-appearance: none;
                    /* 移除内边距（可选） */
                    padding: 0;
                    /* 移除轮廓线（点击时的外边框） */
                    outline: none;
                    /* 设置背景（可选） */
                    background: transparent;
                    text-align: right;
                }

                ::selection {
                    background: transparent;
                    color: inherit;
                }

                /* 为了浏览器兼容性 */
                ::-moz-selection {
                    background: transparent;
                    color: inherit;
                }
            }
        }

        .label-flex {
            display: flex;

            label {
                width: 100%;
                margin-bottom: 0.5rem;
            }
        }

        .label-btns {
            width: 100% !important;
            margin-top: 1.5rem;

            .btn {
                width: 100% !important;
                height: 44px;
                font-size: 16px;
            }
        }
    }

    .seeIcon {
        width: 20px;
        height: 20px;
        right: 0;
    }

    .pc-radio {
        display: none !important;
    }

    .label-flex {
        display: none !important;
    }

    .mobile-switch {
        display: flex;
        align-items: center;
        height: 50px;
    }

    .switch_label {
        display: flex;
        justify-content: end;
    }

    .switch-core {
        width: 50px;
        height: 24px;
    }

    .switch-button {
        width: 20px;
        height: 20px;
    }

    .switch-core.is-checked .switch-button {
        transform: translateX(26px);
    }

    .select-arrow {
        position: absolute;
        right: 6px;
        top: 50% !important;
        transform: translateY(-50%) !important;
        width: 10px;
        height: 10px;
        border-top: 2px solid #606165;
        border-right: 2px solid #606165;
        transform: translateY(-50%) rotate(45deg) !important;
        pointer-events: none;
    }
}
</style>