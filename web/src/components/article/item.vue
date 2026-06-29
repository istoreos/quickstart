<template>
    <article-container>
        <a @click="onClick">
            <div class="cover">
                <div class="thumbnail">
                    <slot></slot>
                </div>
            </div>
        </a>
    </article-container>
</template>
<script setup lang="ts">
import { App, getCurrentInstance, PropType } from "vue"
import ArticleContainer from "./container.vue"
const emits = defineEmits([
    "click",
])
const onClick = () => {
    emits('click')
}
</script>
<style lang="scss" scoped>
a {
    position: relative;
    display: block;
    width: 100%;
    transition: 0.4s;
    cursor: pointer;
    &:hover {
        transform: scale(1.07);
        transition: 0.4s;
        position: relative;
        .cover {
            .thumbnail {
                box-shadow: 0px 6px 40px 0px #1c67f2;
            }
        }
    }
    .cover {
        position: relative;
        padding-top: 130%;
        z-index: 1;
        :deep(.thumbnail) {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            object-fit: contain;
            border-radius: 6px;
            overflow: hidden;
            z-index: 1;
            background-color: #2dc8fd;
            display: flex;
            flex-wrap: wrap;
            align-items: center;
            align-content: center;
            justify-content: center;
            i {
                display: block;
                font-size: 100px;
                color: #eee;
            }
            span {
                display: block;
                text-align: center;
                width: 100%;
                color: #eeee;
                font-size: 28px;
                margin: 1rem 0;
            }
        }
    }
}
article {
    &:nth-child(9n + 1) {
        a {
            .cover {
                .thumbnail {
                    background-color: #ff9100;
                }
            }
        }
    }
    &:nth-child(9n + 2) {
        a {
            .cover {
                .thumbnail {
                    background-color: #2dc8fd;
                }
            }
        }
    }
    &:nth-child(9n + 3) {
        a {
            .cover {
                .thumbnail {
                    background-color: #f66a2c;
                }
            }
        }
    }
    &:nth-child(9n + 4) {
        a {
            .cover {
                .thumbnail {
                    background-color: #9b58de;
                }
            }
        }
    }
    &:nth-child(9n + 5) {
        a {
            .cover {
                .thumbnail {
                    background-color: #297ff3;
                }
            }
        }
    }
    &:nth-child(9n + 6) {
        a {
            .cover {
                .thumbnail {
                    background-color: #27aa8f;
                }
            }
        }
    }
    &:nth-child(9n + 7) {
        a {
            .cover {
                .thumbnail {
                    background-color: #f15a4a;
                }
            }
        }
    }
    &:nth-child(9n + 8) {
        a {
            .cover {
                .thumbnail {
                    background-color: #439c07;
                }
            }
        }
    }
    &:nth-child(9n + 9) {
    }
    &:nth-child(9n + 10) {
    }
}
</style>