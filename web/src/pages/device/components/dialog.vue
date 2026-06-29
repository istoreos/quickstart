<template>
    <!-- 模式1: 安装/结果弹窗 -->
    <transition name="fade">
        <div v-if="showInstallDialog && mode === 'install'" class="dialog-overlay">
            <div class="dialog-container">
                <div class="dialog-title">{{ title }}</div>
                <div class="loading-animation">
                    <div class="spinner"></div>
                </div>
                <p class="dialog-message">{{$gettext("正在安装")}}...</p>
                <button class="dialog-button" @click="cancelInstall">{{$gettext("关闭")}}</button>
            </div>
        </div>
    </transition>

    <transition name="fade">
        <div v-if="showResultDialog && mode === 'install'" class="dialog-overlay">
            <div class="dialog-container">
                <div class="dialog-title">{{$gettext("结果")}}</div>
                <div style="display: flex;justify-content: center;">
                    <svg t="1752661662572" class="icon" viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" p-id="5921" width="50" height="50"><path d="M0 512C0 229.234759 229.234759 0 512 0s512 229.234759 512 512-229.234759 512-512 512S0 794.765241 0 512z m419.310345 194.630621a35.310345 35.310345 0 0 0 49.399172 1.271172l335.518897-311.931586a35.310345 35.310345 0 0 0-48.075035-51.729655l-309.124413 289.544827-145.125518-149.645241a35.310345 35.310345 0 1 0-50.688 49.169655l168.112552 173.320828z" fill="#553afe" p-id="5922"></path></svg>
                </div>
                <p class="dialog-message">{{$gettext("安装成功")}}！</p>
                <button class="dialog-button" @click="closeResultDialog">{{$gettext("确定")}}</button>
            </div>
        </div>
    </transition>

    <!-- 模式2: 添加标签弹窗 -->
    <transition name="fade">
        <div v-if="showTagDialog && mode === 'tag'" class="dialog-overlay">
            <div class="dialog-container tag-dialog">
                <div class="dialog-title">{{ type === '1' ? $gettext("添加标签") :  $gettext("编辑标签")}}</div>

                <div class="warning-message">
                    <svg class="warning-icon" viewBox="0 0 24 24">
                        <path fill="currentColor" d="M12,2L1,21H23M12,6L19.53,19H4.47M11,10V14H13V10M11,16V18H13V16" />
                    </svg>
                    <span>{{$gettext("注意：添加ID时，请勿将“odhcpd”或网络接口（例如“lan”,“wan”,“wan6”等）作为ID，此举将产生冲突。建议在ID前面加上前缀“t_”以杜绝此类冲突。")}}</span>
                </div>

                <div class="input-group">
                    <label for="tagTitle">{{$gettext("标题")}}：</label>
                    <input id="tagTitle" v-model="tagTitle" type="text" :placeholder="$gettext('请输入')+'...'" class="tag-input" />
                </div>
                <div class="input-group">
                    <label for="tagName">{{$gettext("ID")}}：</label>
                    <input id="tagName" v-model="tagName" @input="filterChinese" :disabled="type == '2'" type="text" :placeholder="$gettext('请输入')+'...'" class="tag-input" />
                </div>
                <div class="input-group">
                    <label for="gateway">{{$gettext("网关")}}：</label>
                    <input id="gateway" v-model="gateway" type="text" :placeholder="$gettext('请输入')+'...'" class="tag-input" />
                </div>

                <div class="button-group">
                    <button class="cancel-button" @click="closeTagDialog">{{$gettext("取消")}}</button>
                    <button class="confirm-button" @click="confirmTag">{{$gettext("确定")}}</button>
                </div>
            </div>
        </div>
    </transition>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import Toast from "/@/components/toast";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

defineProps({
  title: {
    type: String,
    required: true
  }
})

// 通用状态
const mode = ref<'install' | 'tag'>('install')
// const title = $gettext("浮动网关")

// 模式1: 安装/结果状态
const showInstallDialog = ref(false)
const showResultDialog = ref(false)

// 模式2: 添加标签状态
const showTagDialog = ref(false)
const tagTitle = ref('')
const tagName = ref('')
const gateway = ref('')

// 打开安装弹窗
const openInstallDialog = () => {
    mode.value = 'install'
    showInstallDialog.value = true
}

// 取消安装
const cancelInstall = () => {
    showInstallDialog.value = false
}

// 安装完成，显示结果
const showInstallResult = () => {
    showInstallDialog.value = false
    setTimeout(() => {
        showResultDialog.value = true
    }, 300)
}

// 关闭结果弹窗
const closeResultDialog = () => {
    showResultDialog.value = false
}

const type = ref('1')
// 打开添加标签弹窗
const openTagDialog = () => {
    mode.value = 'tag'
    type.value = '1'
    tagName.value = ''
    tagTitle.value = ''
    gateway.value = ''
    showTagDialog.value = true
}
const openEditTagDialog = () => {
    mode.value = 'tag'
    type.value = '2'
    showTagDialog.value = true
}

// 关闭添加标签弹窗
const closeTagDialog = () => {
    showTagDialog.value = false
}

const filterChinese = (e: any) => {
    /// 移除所有中文字符（Unicode范围 \u4e00-\u9fa5）
    tagName.value = e.target.value.replace(/[\u4e00-\u9fa5]/g, '');
};

// 校验ip和mac
const validateNetworkAddress = (type: 'ip' | 'mac', value: string) => {
    if (!value) return false;

    const patterns = {
        ip: /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
        mac: /^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$|^([0-9A-Fa-f]{4}\.){2}([0-9A-Fa-f]{4})$/
    };

    return patterns[type].test(value.trim());
}
// 确认添加标签
const confirmTag = () => {
    if(!validateNetworkAddress('ip', gateway.value.trim())) {
        return Toast.Warning(`${$gettext('请输入正确的IP地址')}`);
    }
    if (tagTitle.value.trim() &&tagName.value.trim() && gateway.value.trim()) {
        emit('confirm', { tagTitle:tagTitle.value.trim(),tagName: tagName.value.trim(), gateway: gateway.value.trim() , type: type.value })
        closeTagDialog()
    }
}

// 监听标签名称变化
watch(tagTitle, (newVal) => {
    emit('update:modelValue', newVal)
})
watch(tagName, (newVal) => {
    emit('update:modelValue', newVal)
})
watch(gateway, (newVal) => {
    emit('update:modelValue', newVal)
})
// 定义事件
const emit = defineEmits(['confirm', 'update:modelValue'])

// 暴露方法给父组件
defineExpose({
    tagTitle,
    tagName,
    gateway,
    openInstallDialog,
    showInstallResult,
    openTagDialog,
    closeTagDialog,
    cancelInstall,
    openEditTagDialog
})
</script>

<style lang="scss" scoped>
.dialog-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    padding: 16px;
}

.dialog-container {
    background-color: white;
    border-radius: 12px;
    padding: 16px;
    width: 100%;
    max-width: 400px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);

    .dialog-title {
        margin: 0 0 20px 0;
        font-size: 1.2rem;
        font-weight: 500;
        color: #333;
        text-align: left;
    }

    .dialog-message {
        margin: 20px 0;
        font-size: 1rem;
        color: #666;
        text-align: center;
    }

    // 模式1: 安装/结果样式
    .loading-animation {
        margin: 20px 0;

        .spinner {
            width: 40px;
            height: 40px;
            margin: 0 auto;
            border: 4px solid rgba(110, 72, 170, 0.2);
            border-radius: 50%;
            border-top-color: #8d78fa;
            animation: spin 1s linear infinite;
        }
    }

    .dialog-button {
        background-color: #553afe;
        color: white;
        border: none;
        border-radius: 6px;
        padding: 4px 20px;
        font-size: 1rem;
        cursor: pointer;
        transition: all 0.2s;
        margin: 0 auto;
        display: block;

        &:hover {
            background-color: #553afe;
            opacity: 0.9;
        }

        &:active {
            transform: scale(0.98);
        }
    }

    // 模式2: 添加标签样式
    .warning-message {
        display: flex;
        gap: 8px;
        background-color: #FFF8E1;
        border-left: 4px solid #FFC107;
        padding: 12px;
        margin-bottom: 20px;
        border-radius: 4px;
        font-size: 0.9rem;
        color: #333;
        text-align: left;

        .warning-icon {
            flex-shrink: 0;
            width: 20px;
            height: 20px;
            color: #FFA000;
        }
    }

    .input-group {
        margin-bottom: 16px;
        text-align: left;

        label {
            display: block;
            margin-bottom: 8px;
            font-size: 0.95rem;
            color: #333;
        }

        .tag-input {
            width: 100%;
            padding: 10px 12px;
            border: 1px solid #ddd;
            border-radius: 6px;
            font-size: 1rem;
            transition: border-color 0.2s;

            &:focus {
                outline: none;
                border-color: #6e48aa;
            }
        }
    }

    .button-group {
        display: flex;
        justify-content: flex-end;
        gap: 12px;

        .cancel-button {
            background-color: #fff;
            color: #333;
            border: 1px solid #ddd;
            border-radius: 6px;
            padding: 4px 20px;
            font-size: 1rem;
            cursor: pointer;
            transition: all 0.2s;

            &:hover {
                background-color: #e0e0e0;
            }

            &:active {
                transform: scale(0.98);
            }
        }

        .confirm-button {
            background-color: #553AFE;
            color: white;
            border: none;
            border-radius: 6px;
            padding: 4px 20px;
            font-size: 1rem;
            cursor: pointer;
            transition: all 0.2s;

            &:hover {
                background-color: #553AFE;
                opacity: 0.9;
            }

            &:active {
                transform: scale(0.98);
            }
        }
    }
}

.tag-dialog {
    max-width: 500px;
}

// 过渡动画
.fade-enter-active,
.fade-leave-active {
    transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
    opacity: 0;
}

// 旋转动画
@keyframes spin {
    0% {
        transform: rotate(0deg);
    }

    100% {
        transform: rotate(360deg);
    }
}

@keyframes rotate-circle {
    0% {
        transform: rotate(-45deg);
    }

    5% {
        transform: rotate(-45deg);
    }

    12% {
        transform: rotate(-405deg);
    }

    100% {
        transform: rotate(-405deg);
    }
}

@keyframes icon-line-tip {
    0% {
        width: 0;
        left: 1px;
        top: 15px;
    }

    54% {
        width: 0;
        left: 1px;
        top: 15px;
    }

    70% {
        width: 40px;
        left: -6px;
        top: 30px;
    }

    84% {
        width: 14px;
        left: 17px;
        top: 38px;
    }

    100% {
        width: 12px;
        left: 8px;
        top: 22px;
    }
}

@keyframes icon-line-long {
    0% {
        width: 0;
        right: 37px;
        top: 43px;
    }

    65% {
        width: 0;
        right: 37px;
        top: 43px;
    }

    84% {
        width: 44px;
        right: 0px;
        top: 28px;
    }

    100% {
        width: 20px;
        right: 8px;
        top: 20px;
    }
}

/* 移动端适配 */
@media (max-width: 480px) {
    .dialog-container {
        padding: 16px;

        .dialog-title {
            font-size: 1.1rem;
            margin-bottom: 16px;
        }

        .dialog-message {
            font-size: 0.95rem;
            margin: 16px 0;
        }

        .warning-message {
            font-size: 0.85rem;
            padding: 10px;
        }

        .input-group {
            margin-bottom: 20px;

            label {
                font-size: 0.9rem;
            }

            .tag-input {
                padding: 8px 10px;
                font-size: 0.95rem;
            }
        }

        .button-group {
            gap: 8px;

            .cancel-button,
            .confirm-button {
                padding: 4px 16px;
                font-size: 0.95rem;
            }
        }

        .loading-animation .spinner {
            width: 36px;
            height: 36px;
        }
    }
}
</style>