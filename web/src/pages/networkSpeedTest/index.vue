<template>
    <div id="page">
        <div style="text-align: left;display: flex;align-items: center;margin-bottom: 20px;padding-top: 4px;">
            <router-link to="/"
                style="text-decoration: none;color: var(--breadcrumbs-tit-color);line-height: 1.5em;margin-right: 4px;">{{
                $gettext("首页") }}</router-link>
            <svg width="20" height="20" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
                <path d="M20 30 L50 50 L20 70" stroke="#d6dbf8" stroke-width="8" stroke-linecap="round" fill="none" />
            </svg>
            <a style="text-decoration: none;color: var(--breadcrumbs-tit-color1);line-height: 1.5em;">{{
                $gettext("网络测速") }}</a>
        </div>
    </div>
    <div class="container" style="color: black;">
        <div class="title">
            <div>
                <signalIcon class="icon" />
                <span>网络测速工具</span>
            </div>
            <p>专业的网络连接速度检测工具，支持内网和外网测速</p>
        </div>
        <item />
    </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
import item from "./components/item.vue"
import signalIcon from "/@/components/svg/signal.vue"
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

</script>
<style lang="scss" scoped>
.icon {
    width: 30px;
    height: 30px;
    margin-right: 10px;
    display: block;

    :deep(svg) {
        display: block;
    }
}

.container {
    min-height: 87vh;

    .title {
        font-size: 16px;
        margin-bottom: 20px;
        display: flex;
        justify-content: center;
        flex-direction: column;

        >div {
            display: inline-flex;
            justify-content: center;
            align-items: center;
            line-height: 1;
            margin-bottom: 10px;

            >span {
                color: var(--app-container_title-color);
                position: relative;
                top: 1px;
                font-weight: 600;
            }
        }

        >p {
            text-align: center;
            color: #717182;
        }
    }
}
</style>

<style lang="scss" scoped>
/* 移动端样式 - 基于827px设计图 */
@media (max-width: 827px) {}
</style>