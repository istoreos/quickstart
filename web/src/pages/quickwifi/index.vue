<template>
    <div id="page" class="page-container">
        <!-- 移动端标签栏 (仅移动端显示) -->
        <div class="mobile-tags-container">
            <div class="tags-wrapper">
                <div class="tags-scroll">
                    <div v-for="item in ifaces" :key="getGenIfaceTabName(item)"
                        :class="activeTab === getGenIfaceTabName(item) ? 'tag-item active' : 'tag-item'"
                        @click.stop="hanldeSwitchBand(item)">
                        {{ getIfaceName(item) }} {{ item.isGuest ? ` ${$gettext("访客")}` : "" }} Wi-Fi {{ item.ifaceIndex? `[${item.ifaceIndex}]` : '' }}
                    </div>
                </div>
                <div class="more-btn-wrapper">
                    <div class="fade-overlay"></div>
                    <div class="more-btn" @click="showPopup = true">
                        <div class="line"></div>
                        <div class="line"></div>
                        <div class="line"></div>
                    </div>
                </div>
            </div>
        </div>

        <!-- PC端布局 -->
        <div class="page-flex">
            <!-- PC端侧边栏 (仅PC端显示) -->
            <div class="page-sidebar">
                <template v-for="item in ifaces" :key="getGenIfaceTabName(item)">
                    <div :class="activeTab === getGenIfaceTabName(item) ? 'item activeItem' : 'item'"
                        @click.stop="hanldeSwitchBand(item)">
                        {{ getIfaceName(item) }} {{ item.isGuest ? ` ${$gettext("访客")}` : "" }} Wi-Fi {{ item.ifaceIndex? `[${item.ifaceIndex}]` : '' }}
                    </div>
                </template>
            </div>

            <!-- 主内容区 -->
            <div class="page-main">
                <template v-for="item in ifaces" :key="getGenIfaceTabName(item)">
                    <FormVue :data="item" v-if="activeTab === getGenIfaceTabName(item)" @getData="getData" />
                </template>
            </div>
        </div>

        <!-- 移动端弹出框 -->
        <div class="popup-overlay" v-if="showPopup" @click.self="showPopup = false">
            <div class="popup-content">
                <div class="popup-tags">
                    <div v-for="item in ifaces" :key="getGenIfaceTabName(item)"
                        :class="activeData === getGenIfaceTabName(item) ? 'popup-tag-item active' : 'popup-tag-item'"
                        @click.stop="hanldeSwitchBand1(item)">
                        {{ getIfaceName(item) }} {{ item.isGuest ? ` ${$gettext("访客")}` : "" }} Wi-Fi {{ item.ifaceIndex? `[${item.ifaceIndex}]` : '' }}
                    </div>
                </div>

                <div class="popup-footer">
                    <button class="cancel-btn" @click="showPopup = false">{{ $gettext("取消") }}</button>
                    <button class="confirm-btn" @click="confirm">{{ $gettext("确定") }}</button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { provide, ref,watch } from 'vue'
import request from '/@/request';
import { useRoute, useRouter } from 'vue-router';
import FormVue from "./components/form.vue"
import { useGettext } from '/@/plugins/i18n'
// import { watch } from 'node:fs';

const { $gettext } = useGettext()
const disabled = ref(false)
provide("disabled", disabled)
const ifaces = ref<Iface[]>([])
const route = useRoute()
const router = useRouter()
const activeTab = ref<string | undefined>(String(route?.query?.tab))
// const activeTab = ref<string | undefined>(String(''))
const showPopup = ref(false)

const getIfaceName = (iface: Iface) => {
    return iface.band?.toUpperCase()
}

const getGenIfaceTabName = (value: Iface) => {
    if (value.isGuest) {
        return value.band + "_Guest"
    }
    return value.ssid
    // return value.band
}

const hanldeSwitchBand = (value: Iface) => {
    if (disabled.value) {
        return
    }
    activeTab.value = getGenIfaceTabName(value)
    showPopup.value = false

    if (activeTab.value !== route.query.tab) {
        router.push({
            query: {
                tab: activeTab.value
            }
        })
    }
}
const activeData = ref()
const hanldeSwitchBand1 = (value: Iface) => {
    if (disabled.value) {
        return
    }
    activeData.value = getGenIfaceTabName(value)
}

const confirm = () => {
    showPopup.value = false
    if (activeData.value !== route.query.tab) {
        router.push({
            query: {
                tab: activeData.value
            }
        })
    }
}

watch(() => showPopup.value, (newValue) => {
    if(newValue) {
        activeData.value = activeTab.value
    } else {
       activeTab.value = activeData.value 
    }
})

const getData = async () => {
    try {
        const { data } = await request.Quickwifi.List.GET()
        const { error, result } = data
        if (error) {
            throw error
        }
        if (result?.ifaces) {
            ifaces.value = result.ifaces.map((item): Iface => {
                return {
                    ...item,
                    hidden: item.hidden || false,
                    disabled: item.disabled || false,
                    isGuest: item.isGuest || false,
                    channel: item.channel || 0,
                    txpower: item.txpower || 0,
                    ifaceIndex: item.ifaceIndex || 0,
                }
            })
        }
        let isTab = false
        for (let i = 0; i < ifaces.value.length; i++) {
            if (getGenIfaceTabName(ifaces.value[i]) === activeTab.value) {
                hanldeSwitchBand(ifaces.value[i])
                isTab = true
                break
            }
        }
        if (!isTab && ifaces.value.length > 0) {
            hanldeSwitchBand(ifaces.value[0])
        }
    } catch (error) {
        console.log(error)
    }
}

getData()
</script>

<style lang="scss" scoped>
/* 基础样式 */
.page-container {
    width: 100%;
    background-color: var(--card-bg-color);
    border-radius: 6px;
    padding: 3rem;
    margin-top: 50px;
}

/* 移动端标签样式 */
.mobile-tags-container {
    display: none;
    /* 默认隐藏 */
}

/* PC端样式 */
.page-flex {
    display: flex;

    .page-sidebar {
        flex: 0 0 200px;
        border-right: 1px solid #eee;

        .item {
            width: 100%;
            height: 42px;
            line-height: 42px;
            font-size: 16px;
            cursor: pointer;
            color: var(--item-label_key-span-color);
            display: block;
            user-select: none;
            position: relative;
            display: flex;
            flex-wrap: wrap;
            align-items: center;

            &:hover {
                transition: 0.3s;
                color: #418cff;
            }

            &.activeItem {
                transition: 0.3s;
                color: #418cff;

                &:before {
                    content: "";
                    position: absolute;
                    left: -1rem;
                    width: 3px;
                    height: 20px;
                    background-color: #4388ff;
                }
            }
        }
    }

    .page-main {
        flex: 1;
        padding-left: 24px;
    }
}

/* 移动端样式 - 基于827px设计图 */
@media (max-width: 827px) {
    .page-container {
        padding: 12px 12px 0 8px;
        margin-top: 0;
    }

    /* 隐藏PC端侧边栏 */
    .page-flex {
        display: block;

        .page-sidebar {
            display: none;
        }

        .page-main {
            padding-left: 0;
            padding-top: 16px;
        }
    }

    /* 显示移动端标签栏 */
    .mobile-tags-container {
        display: block;
        width: 100%;
        margin-bottom: 16px;
        position: relative;
    }

    .tags-wrapper {
        display: flex;
        align-items: center;
        height: 40px;
        position: relative;
        // overflow: hidden;
    }

    .tags-scroll {
        flex: 1;
        display: flex;
        overflow-x: auto;
        scrollbar-width: none;
        -ms-overflow-style: none;
        height: 100%;
        align-items: center;
        white-space: nowrap;
        padding-right: 40px;

        &::-webkit-scrollbar {
            display: none;
        }
    }

    .tag-item {
        flex-shrink: 0;
        padding: 7px 12px;
        margin-right: 8px;
        border-radius: 4px;
        background-color: var(--tag-bg-color);
        color: var(--item-label_key-span-color);
        font-size: 12px;
        line-height: 18px;
        cursor: pointer;
        white-space: nowrap;

        &.active {
            background-color: #5279F7;
            color: #fff;
        }
    }

    /* 更多按钮容器 */
    .more-btn-wrapper {
        position: absolute;
        right: -6px;
        top: 0;
        height: 100%;
        width: 40px;
        display: flex;
        align-items: center;
        justify-content: flex-end;
        pointer-events: none;
    }

    /* 渐变遮罩 */
    .fade-overlay {
        position: absolute;
        right: 0;
        top: 50%;
        transform: translateY(-50%);
        width: 100px;
        height: 32px;
        background: var(--gradient-bg-color);
    }

    /* 更多按钮 */
    .more-btn {
        width: 28px;
        height: 28px;
        border-radius: 4px;
        display: flex;
        flex-direction: column;
        justify-content: center;
        align-items: center;
        cursor: pointer;
        pointer-events: auto;
        position: relative;
        z-index: 1;

        .line {
            width: 14px;
            height: 2px;
            background-color: #5279F7;
            margin: 2px 0;
            border-radius: 1px;
        }
    }
    :deep(.showSide) {
        z-index: 1 !important;
    }
    /* 弹出框样式 */
    .popup-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background-color: rgba(0, 0, 0, 0.5);
        z-index: 1000;
        display: flex;
        justify-content: center;
        align-items: flex-start;
    }

    .popup-content {
        width: 100%;
        // max-width: 827px;
        max-height: 85vh;
        background-color: var(--popup-bg-color);
        border-radius: 0 0 4px 4px;
        animation: slideDown 0.3s ease-out;
        // overflow: hidden;
        overflow-y: auto;
        padding-top: 25px;

        .popup-tag-item, .active{
            text-align: center;
            padding: 8px 12px 5px;
            width: calc((100% - 24px) / 3);
        }
    }

    @keyframes slideDown {
        from {
            transform: translateY(-100%);
        }

        to {
            transform: translateY(0);
        }
    }

    .popup-tags {
        padding: 12px;
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        max-height: 70vh;
        overflow-y: auto;
    }

    .popup-tag-item {
        padding: 7px 12px;
        border-radius: 4px;
        background-color: #f5f5f5;
        color: #333;
        font-size: 12px;
        line-height: 18px;
        cursor: pointer;
        white-space: nowrap;

        &.active {
            background-color: #5279F7;
            color: #fff;
        }
    }

    .popup-footer {
        display: flex;
        padding: 12px;
        border-top: 1px solid #f0f0f0;

        button {
            flex: 1;
            height: 36px;
            border-radius: 23px;
            font-size: 14px;
            cursor: pointer;
        }

        .cancel-btn {
            background-color: #f5f5f5;
            color: #000;
            border: none;
            margin-right: 12px;
        }

        .confirm-btn {
            background-color: #5279F7;
            color: #fff;
            border: none;
        }
    }
}
</style>
