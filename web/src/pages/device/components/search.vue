<template>
    <div style="display: flex;align-items: center;">
        <button class="del-button add-button--danger" style="" v-if="showAdd" @click="handleAdd">
            <span>{{ $gettext('添加') }}</span>
        </button>
        <button class="del-button del-button--danger" v-if="showBatchDelete" @click="handleBatchDelete">
            <span>{{ $gettext('批量删除') }}</span>
        </button>
        <div class="search_box">
            <div class="search_container">
                <div class="search_input_wrapper">
                    <input type="text" @keyup.enter="handleSearch" v-model.trim="search" class="search_input"
                        :placeholder="$gettext('请输入名称/IP/MAC…')">
                    <svg class="search_icon" viewBox="0 0 24 24" width="24" height="24" @click="handleSearch">
                        <path
                            d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 0 0 1.48-5.34c-.47-2.78-2.79-5-5.59-5.34a6.505 6.505 0 0 0-7.27 7.27c.34 2.8 2.56 5.12 5.34 5.59a6.5 6.5 0 0 0 5.34-1.48l.27.28v.79l4.25 4.25c.41.41 1.08.41 1.49 0 .41-.41.41-1.08 0-1.49L15.5 14zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z" />
                    </svg>
                </div>
                <button class="refresh_button" :class="{ rotate: isRefreshing }" @click="handleRefresh">
                    <svg class="refresh_icon" viewBox="0 0 24 24" width="26" height="26">
                        <path
                            d="M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z" />
                    </svg>
                </button>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, } from 'vue'
import Toast from "/@/components/toast";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()
const search = ref('')
const emit = defineEmits(['refresh', 'batch-delete', 'search','handleAdd'])
const isRefreshing = ref(false)

const props = defineProps({
    showBatchDelete: {
        type: Boolean,
        default: true
    },
    showAdd: {
        type: Boolean,
        default: true
    }
})
const showBatchDelete = ref(props.showBatchDelete)
const showAdd = ref(props.showAdd)

// 刷新
const handleRefresh = () => {
    emit('refresh', { data: '这是子组件的数据' })
}

// 搜索
const handleSearch = () => {
    // if (!search.value) {
    //     return Toast.Warning($gettext("请输入搜索词"))
    // }
    emit('search', String(search.value))
}

// 批量删除
const handleBatchDelete = () => {
    emit('batch-delete', { data: '这是子组件的数据' })
}
const handleAdd = () => {
    emit('handleAdd')
}
</script>

<style lang="scss" scoped>
.del-button {
    display: inline-flex;
    justify-content: center;
    align-items: center;
    line-height: 1;
    white-space: nowrap;
    cursor: pointer;
    background: #fff;
    border: 1px solid #dcdfe6;
    color: #606266;
    text-align: center;
    box-sizing: border-box;
    outline: none;
    margin: 0;
    transition: all 0.1s;
    font-weight: 500;
    user-select: none;
    padding: 8px 15px;
    font-size: 14px;
    border-radius: 4px;
    margin-right: 8px;
}

.add-button--danger {
    color: #fff;
    background-color: #553afe;
    border-color: #553afe;
}

/* 悬停效果 */
.add-button--danger:hover {
    background: #5c44f8;
    border-color: #5c44f8;
    color: #fff;
}

/* 激活效果 */
.add-button--danger:active {
    background: #553AFE;
    border-color: #553AFE;
    color: #fff;
}

/* 禁用状态 */
.add-button.is-disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

/* 删除按钮特定样式 */
.del-button--danger {
    color: #fff;
    background-color: #f56c6c;
    border-color: #f56c6c;
}

/* 悬停效果 */
.del-button--danger:hover {
    background: #f78989;
    border-color: #f78989;
    color: #fff;
}

/* 激活效果 */
.del-button--danger:active {
    background: #dd6161;
    border-color: #dd6161;
    color: #fff;
}

/* 禁用状态 */
.del-button.is-disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.search_box {
    width: 350px;

    .search_container {
        display: flex;
        align-items: center;
        gap: 8px;

        .search_input_wrapper {
            position: relative;
            flex-grow: 1;

            .search_icon {
                position: absolute;
                right: 10px;
                top: 50%;
                transform: translateY(-50%);
                fill: rgba(0, 0, 0, 0.8);
                cursor: pointer;
            }

            .search_input {
                width: 100%;
                padding: 4px 35px 4px 12px;
                border-radius: 4px;
                border: 1px solid rgba(0, 0, 0, 0.6);
                font-size: 14px;
                outline: none;
                transition: border-color 0.3s;
                color: #222;
                background: transparent;

                &:focus {
                    border-color: #4a90e2;
                }
            }
        }

        .refresh_button {
            background: none;
            border: none;
            cursor: pointer;
            padding: 8px;
            border-radius: 50%;
            transition: background-color 0.3s;
            display: flex;
            align-items: center;
            justify-content: center;

            &:hover {
                background-color: #f0f0f0;
            }

            .refresh_icon {
                fill: rgba(0, 0, 0, 0.8);
                transition: transform 0.3s;
            }

            &.rotate .refresh_icon {
                animation: spin 1s linear infinite;
            }
        }
    }
}

@keyframes spin {
    from {
        transform: rotate(0deg);
    }

    to {
        transform: rotate(360deg);
    }
}
</style>

<style lang="scss" scoped>
/* 移动端样式 - 基于827px设计图 */
@media (max-width: 827px) {
    .search_box {
        width: 80%;
    }
    .del-button {
        padding: 6px 8px;
    }
}
</style>