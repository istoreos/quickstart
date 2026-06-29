<template>
    <Card :title='$gettext("远程域名")' :showSettings="true" style="width: 100%;height: 100%; display: block;">
        <template #icon>
            <earthIcon color="#00a63e" class="icon" />
        </template>
        <template #settings>
            <div class="btn_settings" @click="onSetting">
                <earthIcon color="#0a0a0a" class="icon1 earthIcon" style="margin-right: 6px;" />
                {{ $gettext("域名配置") }}
            </div>
        </template>
        <div class="content">
            <div class="tab">
                <div class="item cloud" :class="{ active: activeIndex == 0 }" @click="tabChange(0)">
                    <earthIcon color="#155dfc" class="icon2" />
                    <div class="title">DDNSTO</div>
                    <span v-if="DdnsStatus?.ddnstoDomain">{{ $gettext("正常") }}</span>
                    <span v-else>{{ $gettext("未启动") }}</span>
                </div>
                <div class="item memory" v-if="ipv4Domain" :class="{ active: activeIndex == 1 }" @click="tabChange(1)">
                    <topologyIcon color="#00a63e" class="icon2" />
                    <div class="title">IPv4</div>
                    <span v-if="ipv4Domain == 'Stopped' || ipv4Domain == 'Disabled'">{{ $gettext("离线") }}</span>
                    <span v-else>{{ $gettext("正常") }}</span>
                </div>
                <div class="item network" v-if="ipv6Domain" :class="{ active: activeIndex == 2 }" @click="tabChange(2)">
                    <routerIcon class="icon2" />
                    <div class="title">IPv6</div>
                    <span v-if="ipv6Domain == 'Stopped' || ipv6Domain == 'Disabled'">{{ $gettext("离线") }}</span>
                    <span v-else>{{ $gettext("正常") }}</span>
                </div>
            </div>
            <div class="info" v-if="activeIndex == 0">
                <div class="status">
                    <div>DDNSTO</div>
                    <span v-if="DdnsStatus?.ddnstoDomain">{{ $gettext("正常") }}</span>
                    <span v-else class="offline">{{ $gettext("未启动") }}</span>
                </div>
                <div class="title_box">
                    <div class="title">{{ $gettext("智能内网穿透服务") }}</div>
                    <div class="path">
                        <a v-if="DdnsStatus?.ddnstoDomain" class="configure" :href="DdnsStatus?.ddnstoDomain"
                            target="_blank" rel="noopener noreferrer" :title="DdnsStatus?.ddnstoDomain">
                            {{ DdnsStatus?.ddnstoDomain }}
                        </a>
                        <div v-else>{{ $gettext("未安装或未配置") }}</div>
                        <span><a class="item_btn" href="https://www.kooldns.cn/app/#/devices" target="_blank">{{
                            $gettext("控制台")
                                }}</a></span>
                    </div>
                </div>
            </div>
            <div class="info" v-if="activeIndex == 1">
                <div class="status">
                    <div>{{ $gettext("当前状态:") }}</div>
                    <span v-if="ipv4Domain == 'Stopped' || ipv4Domain == 'Disabled'" class="offline">{{ $gettext("离线")
                        }}</span>
                    <span v-else>{{ $gettext("正常") }}</span>
                </div>
                <div class="title_box">
                    <div class="title">IPv4 {{ $gettext("动态域名解析") }}</div>
                    <div class="path">
                        <div v-if="ipv4Domain == 'Stopped' || ipv4Domain == 'Disabled'">
                            {{ ipv4Domain }}
                        </div>
                        <a class="configure" :href="ipv4Domain" target="_blank" rel="noopener noreferrer" v-else>
                            {{ ipv4Domain }}
                        </a>
                        <a href="/cgi-bin/luci/admin/services/ddns" v-if="ipv4Domain">
                            <editIcon class="icon3" />
                        </a>
                    </div>
                </div>
            </div>
            <div class="info" v-if="activeIndex == 2">
                <div class="status">
                    <div>{{ $gettext("当前状态:") }}</div>
                    <span v-if="ipv6Domain == 'Stopped' || ipv6Domain == 'Disabled'" class="offline">{{ $gettext("离线")
                        }}</span>
                    <span v-else>{{ $gettext("正常") }}</span>
                </div>
                <div class="title_box">
                    <div class="title">IPv6 {{ $gettext("动态域名解析") }}</div>
                    <div class="path">
                        <div v-if="ipv6Domain == 'Stopped' || ipv6Domain == 'Disabled'">
                            {{ ipv6Domain }}
                        </div>
                        <a class="configure" :href="ipv6Domain" target="_blank" rel="noopener noreferrer" v-else>
                            {{ ipv6Domain }}
                        </a>
                        <a href="/cgi-bin/luci/admin/services/ddns" v-if="ipv6Domain">
                            <editIcon class="icon3" />
                        </a>
                    </div>
                </div>
            </div>
        </div>
    </Card>
</template>

<script lang="ts" setup>
import Card from "../components/Card.vue"
import earthIcon from "/@/components/svg/earth.vue"
import topologyIcon from "/@/components/svg/topology.vue"
import routerIcon from "/@/components/svg/router1.vue"
import editIcon from "/@/components/svg/edit.vue"

import { computed, ref, onMounted, onUnmounted } from "vue";
import { useGettext } from '/@/plugins/i18n'
import request from '/@/request';
import ActionDomain from "/@/components/action-domain"

const { $gettext } = useGettext()

const activeIndex = ref(0)
const hasUserSelectedTab = ref(false)
const tabChange = (index: number) => {
    activeIndex.value = index
    hasUserSelectedTab.value = true
}

const isDomainOnline = (domain?: string) => {
    return Boolean(domain && domain !== "Stopped" && domain !== "Disabled")
}
const setActiveByStatus = () => {
    if (hasUserSelectedTab.value) return
    const candidates = [
        { index: 0, active: Boolean(DdnsStatus.value.ddnstoDomain) },
        { index: 1, active: isDomainOnline(DdnsStatus.value.ipv4Domain) },
        { index: 2, active: isDomainOnline(DdnsStatus.value.ipv6Domain) },
    ]
    const target = candidates.find(item => item.active)
    activeIndex.value = target ? target.index : 0
}

let mounted = false
let timerId: number | undefined = undefined
const DdnsStatus = ref<GuideDdns>({} as GuideDdns)
const getDdns = function () {
    if (!mounted)
        return
    (document.hidden ? Promise.resolve() : request.Guide.GetDdns.GET().then(res => {
        if (res?.data) {
            if ((res?.data?.success || 0) == 0) {
                if (res.data.result) {
                    DdnsStatus.value = res.data.result
                    setActiveByStatus()
                }
            }

        }
    })).then(() => {
        if (!mounted)
            return
        timerId = window.setTimeout(getDdns, 3000)
    })
}
onMounted(() => {
    mounted = true
    timerId = window.setTimeout(getDdns, 1100)
})
onUnmounted(() => {
    if (timerId !== undefined)
        window.clearTimeout(timerId)
    mounted = false
})
const onSetting = () => {
    ActionDomain({
        url: DdnsStatus.value.ddnstoDomain
    })
}
const ipv4Domain = computed(() => {
    const domain = DdnsStatus.value.ipv4Domain
    if (!domain || domain == "Stopped" || domain == "Disabled") {
        return domain
    }
    return `http://${domain}`
})
const ipv6Domain = computed(() => {
    const domain = DdnsStatus.value.ipv6Domain
    if (!domain || domain == "Stopped" || domain == "Disabled") {
        return domain
    }
    return `http://${domain}`
})
</script>

<style lang="scss" scoped>
:deep(.footer-btn) {
    background: var(--card-bg-color);
    border: 1px solid var(--btn-border-color);
}

:deep(.reusable-card) {
    background: #fff5ee !important;
    border: 1px solid #ffd6a7 !important;
}

:deep(.earthIcon) {
    path {
        fill: var(--app-container_title-color) !important;
    }
}

.icon {
    width: 1.3rem;
    height: 1.3rem;
}

.icon1 {
    width: 1rem;
    height: 1rem;
}

.icon2 {
    width: 1.5rem;
    height: 1.5rem;
    margin-bottom: 8px;
}

.icon3 {
    width: 1.5rem;
    height: 1.5rem;
    cursor: pointer;
}

a {
    text-decoration: none;
    cursor: pointer;
    font-size: 14px;
    display: block;
}

.content {
    color: #333;
    margin-top: 20px;
    margin-bottom: 20px;
    font-weight: normal;
    padding: 0 12px;

    .tab {
        display: flex;
        gap: 8px;

        .item {
            flex: 1;
            padding: 16px;
            display: flex;
            flex-direction: column;
            align-items: center;
            border-radius: 10px;
            cursor: pointer;
            border: 2px solid transparent;
            box-sizing: border-box;

            .title {
                margin-bottom: 8px;
            }

            >span {
                font-size: 12px;
            }
        }

        .active {
            border: 2px solid #6d6d6d;
        }

        .cloud {
            background-color: #eff6ff;
            color: #1447e6;
        }

        .memory {
            background-color: #f0fdf4;
            color: #008236;
        }

        .network {
            background-color: #f9fafb;
            color: #4a5565;
        }
    }

    .info {
        margin-top: 20px;

        .status {
            padding: 20px 0 0;
            margin-top: 16px;
            display: flex;
            justify-content: space-between;
            border-top: 1px solid var(--btn-border-color);

            .offline {
                background: #eceef2;
                color: #030213;
            }

            >div {
                color: var(--app-container_title-color);
                font-size: 16px;
            }

            >span {
                color: #fff;
                padding: 4px 8px;
                background: #030213;
                border-radius: 6px;
                font-size: 12px;
            }
        }

        .title_box {
            margin: 20px 0;

            .title {
                color: var(--item-label_key-span-color);
                margin-bottom: 10px;
            }

            .path {
                display: flex;
                align-items: center;
                justify-content: space-between;
                border: 1px solid #e0e1e1;
                background: #f9fafb;
                border-radius: 4px;
                padding: 8px 10px;

                >span {
                    display: inline-block;
                    padding: 4px 8px;
                    border: 1px solid #553afb;
                    font-size: 12px;
                    border-radius: 4px;
                    cursor: pointer;
                    flex-shrink: 0;

                    >a {
                        color: #553afb;
                    }
                }
            }
        }
    }
}
.btn_settings {
    padding: 6px 18px;
    border-radius: 4px;
    border: 1px solid var(--btn-border-color);
    line-height: 1;
    display: flex;
    align-items: center;
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
    .content {
        margin: 10px 0 0;
        padding: 4px;

        .info {
            margin-top: 10px;

            .status {
                padding-top: 10px;
            }

            .title_box {
                margin: 10px 0;
            }
        }
    }
}
</style>
