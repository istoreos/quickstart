<template>
    <ul class="disk-item">
        <li class="disk-info" :class="{
            on: disk.venderModel == currDisk?.venderModel,
            nopoint: disk.childrens == null || disk.childrens.length == 0
        }" @click="onChildren(disk)">
            <div class="disk-item_icon">
                <svg t="1642563338465" class="icon" viewBox="0 0 1024 1024" version="1.1"
                    xmlns="http://www.w3.org/2000/svg" p-id="2216" width="128" height="128">
                    <path
                        d="M998.4 711.68l-119.467-512c-6.826-42.667-42.666-75.093-87.04-76.8H232.107c-44.374 1.707-80.214 35.84-87.04 78.507L25.6 711.68c-5.12 13.653-6.827 29.013-6.827 42.667 0 76.8 63.147 139.946 141.654 139.946H865.28c78.507 0 141.653-63.146 141.653-139.946 0-13.654-3.413-29.014-8.533-42.667zM394.24 366.933c1.707-51.2 56.32-92.16 124.587-92.16S640 315.733 640 365.227c44.373-1.707 81.92 23.893 83.627 58.026s-34.134 63.147-78.507 64.854h-6.827l-245.76 1.706c-44.373 0-80.213-27.306-80.213-59.733 0-35.84 37.547-63.147 81.92-63.147z m471.04 459.094H160.427c-39.254 0-69.974-30.72-69.974-69.974s32.427-69.973 69.974-69.973H865.28c39.253 0 69.973 30.72 69.973 69.973 1.707 37.547-30.72 69.974-69.973 69.974z m-35.84-92.16c-11.947 0-22.187 8.533-23.893 20.48 0 11.946 8.533 22.186 20.48 23.893h3.413c11.947 0 22.187-10.24 22.187-22.187 0-13.653-8.534-22.186-22.187-22.186z m-46.08 22.186c0-25.6 20.48-46.08 46.08-46.08s46.08 20.48 46.08 46.08-20.48 46.08-46.08 46.08-46.08-20.48-46.08-46.08z"
                        p-id="2217" />
                </svg>
            </div>
            <div class="disk-item_f">
                <div class="disk-item_venderModel">{{ disk.venderModel }}</div>
                <div class="disk-item_used">{{ disk.used }}/{{ disk.size }}</div>
            </div>
            <div class="auto"></div>
            <div class="disk-item-r">{{ disk.path }}</div>
        </li>
        <div class="disk-children" v-show="childrenShow">
            <li class="disk-children_item" :class="{
                on: ole.uuid == currMountPoint?.uuid && ole.path == currMountPoint?.path
            }" v-for="ole in disk.childrens" @click="onDisk(disk, ole)">
                <div class="disk-item_icon">
                    <svg t="1642563581459" class="icon" viewBox="0 0 1228 1024" version="1.1"
                        xmlns="http://www.w3.org/2000/svg" p-id="7132" width="128" height="128">
                        <path
                            d="M525.2096 145.3568c1.9968-45.568-35.6864-99.1232-57.4976-99.1232H57.4976C15.872 79.9232 17.8176 145.408 17.8176 145.408h507.392z"
                            fill="#ECC049" p-id="7133" />
                        <path
                            d="M21.8112 143.36L19.8144 825.1392c0 75.3152 75.3152 152.576 150.6304 152.576h887.9104c75.264 0 150.6304-75.264 150.6304-152.576V297.984c0-75.264-75.3152-152.576-150.6304-152.576h-434.0224L21.8112 143.36z"
                            fill="#FFD658" p-id="7134" />
                    </svg>
                </div>
                <span v-if="ole.mountPoint">
                    【{{ ole.filesystem }}】
                    {{ ole.mountPoint }}
                    （{{ ole.used }}/{{ ole.total }}）
                    [{{ ole.uuid }}]
                </span>
                <span v-else>
                    【{{ ole.filesystem }}】
                    {{ ole.mountPoint || ole.path || $gettext("未挂载磁盘") }}
                    [{{ ole.uuid }}]
                </span>
            </li>
        </div>
    </ul>
</template>
<script setup lang="ts">
import { PropType, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

const props = defineProps({
    disk: {
        type: Object as PropType<NasDiskModel>,
        required: true
    },
    currDisk: {
        type: Object as PropType<NasDiskModel | null>,
    },
    currMountPoint: {
        type: Object as PropType<MountPoint | null>,
    },
    onDisk: {
        type: Function as PropType<(_disk: NasDiskModel, _mount: MountPoint | null) => void>,
        required: true
    },
})
const childrenShow = ref(false)
if (props.currDisk != null && props.currDisk?.venderModel == props.disk?.venderModel) {
    childrenShow.value = true
}
const onChildren = (item: NasDiskModel) => {
    childrenShow.value = !childrenShow.value
    props.onDisk(item, null)
}

</script>
<style lang="scss" scoped>
ul.disk-item {
    width: 100%;
    margin-bottom: 10px;

    .auto {
        flex: auto;
    }

    .disk-item_icon {
        width: 24px;
        height: 24px;
        margin-right: 0.5rem;

        svg {
            width: 100%;
            height: 100%;

            path {
                // fill: #09aaff;
            }
        }
    }

    li.disk-info {
        display: flex;
        flex-wrap: nowrap;
        align-items: center;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        width: 100%;
        padding: 5px 1rem;
        height: 50px;
        cursor: pointer;
        color: #666;
        font-size: 12px;
        border-left: 3px solid #89897f;

        &:hover {
            background-color: #ecf5ff;
        }

        .disk-item_icon {
            svg {
                path {
                    fill: #09aaff;
                }
            }
        }

        .disk-item_f {
            display: flex;
            flex-wrap: wrap;

            .disk-item_venderModel {
                width: 100%;
            }

            .disk-item_used {
                width: 100%;
            }
        }
    }

    li.disk-info.on {
        border-left: 3px solid #ff9c08;
    }

    li.disk-info.on.nopoint {
        background-color: #ecf5ff;
    }

    .disk-children {
        width: 100%;
        color: #666;

        li.disk-children_item {
            width: 100%;
            height: 40px;
            line-height: 40px;
            padding-left: 2rem;
            font-size: 12px;
            cursor: pointer;
            display: flex;
            flex-wrap: nowrap;
            align-items: center;
            border-left: 3px solid #89897f;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;

            &:hover {
                background-color: #ecf5ff;
            }

            span {
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
                display: inline-block;
            }
        }

        li.disk-children_item.on.on {
            border-left: 3px solid #ff9c08;
            background-color: #ecf5ff;
        }
    }
}
</style>