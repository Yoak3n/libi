<template>
    <div class="danmu-wrapper">
        <div class="danmu">
            <div class="avatar" @click.prevent.stop="openSpace">
                <n-avatar round :size="45"
                    :src="props.danmu.user.avatar != '' ? props.danmu.user.avatar : 'https://i0.hdslb.com/bfs/face/member/noface.jpg'"
                    fallback-src="https://i0.hdslb.com/bfs/face/member/noface.jpg"
                    :img-props="{ class: 'avatar-img', alt: props.danmu.user.name }">
                </n-avatar>
            </div>
            <div class="content">
                <div class="info">
                    <div v-if="props.danmu.user.guard" class="fleet"></div>
                    <Medal v-if="props.danmu.user.medal?.name" :name="props.danmu.user.medal.name"
                        :level="props.danmu.user.medal?.level" />
                    <span class="name">{{ props.danmu.user.name }}</span>
                </div>
                <div class="message" v-html="props.danmu.content">
                </div>
            </div>
        </div>
    </div>
</template>
<script setup lang="ts">
import type { PropType } from 'vue';
import { NAvatar } from 'naive-ui'
import { Danmu } from './danmu'
import Medal from '../Medal/index.vue'
import { Browser } from '@wailsio/runtime';
const props = defineProps(
    {
        danmu: {
            type: Object as PropType<Danmu>,
            required: true
        }
    }
)

function openSpace() {
    Browser.OpenURL('https://space.bilibili.com/' + props.danmu.user.uid)
}


</script>

<style scoped lang="less">
.fade-enter-active {
    transition: all 0.5s ease;
}

.fade-enter-from,
.fade-leave-to {
    opacity: 0;
    transform: translateY(50%);
}

.danmu-wrapper {
    .danmu {
        padding: 0 .5rem;
        display: flex;
        overflow-wrap: break-word;
        width: 100%;

        .avatar {
            width: 15%;
            height: auto;
            line-height: 100%;
            text-align: center;
            padding-top: 10px;
            cursor: pointer;

            :deep(.n-avatar), :deep(img) {
                pointer-events: none;
            }
        }

        .content {
            width: 85%;
            padding: 1%;

            .message {
                display: flex;
                font-size: 16px;
                align-items: center;
            }

            .info {
                // background-color: bisque;
                border-radius: 5px;
                margin: 0 1%;
                height: 1.5rem;
                line-height: 1.5rem;
                display: flex;
                justify-content: left;

                .name {
                    color: rgb(189, 193, 197);
                    font-size: 1rem;
                    font-weight: bold;
                }
            }

        }
    }
}
</style>