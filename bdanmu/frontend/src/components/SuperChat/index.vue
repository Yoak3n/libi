<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, PropType} from 'vue';
import { NAvatar,NEllipsis} from 'naive-ui';
import { SuperChat, computeSuperChatBackground } from './super_chat'
import { Browser } from '@wailsio/runtime';
import ProcessBar from '../ProcessBar/index.vue'
const props = defineProps({
  data: {
    type: Object as PropType<SuperChat>,
    required: true,
  }
})
const emit = defineEmits<{ done: [] }>()
const superChatRef = ref<undefined | HTMLElement>(undefined)
const color = computeSuperChatBackground(props.data.price)
const now = Math.floor(Date.now() / 1000)
const long = Math.max(1, props.data.end_time - now)
let height = 30

let timer: ReturnType<typeof setTimeout> | null = null

onMounted(() => {
  // 进度条动画结束后触发清除，多等 1.5s 留给 fade-out 过渡
  timer = setTimeout(() => {
    emit('done')
  }, long * 1000 + 1500) // 多等 1.5s 留给 fade-out 过渡
})

onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
})
</script>

<template>
  <div class="super-chat-wrapper" ref="superChatRef">
      <div class="super-chat-box" v-if="props.data != null" >
      <div class="info">
        <a href="javascript:void(0)" @click="Browser.OpenURL('https://space.bilibili.com/' + props.data.user.uid)">
          <n-avatar round :size="45" :src="props.data.user.avatar"
          fallback-src="https://i0.hdslb.com/bfs/face/member/noface.jpg"
          :img-props="{ class: 'avatar-img', alt: props.data.user.name }">
        </n-avatar>
        </a>

        <div class="name">
          {{ props.data.user.name }}
        </div>
      </div>
      <div class="progress-bar">
        <process-bar :fill-color="color" :height="height" back-color="rgba(28,28,28,0.7)" :progress-time="long" :width="512" />
        <div class="content">
          <n-ellipsis :line-clamp="1"  style="max-width: 450px">
            {{ props.data.content }}
          </n-ellipsis>
        </div>
      </div>
    </div>
  </div>

</template>

<style scoped lang="less">
.super-chat-wrapper {
  width: 100%;
  display: flex;
  align-items: center;
  min-height: 2rem;
  color: rgba(28, 28, 28,1);
  pointer-events: none;
  animation: in 1s ease-in-out;
  .super-chat-box {
    pointer-events: auto;
    display: flex;
    flex-flow: row wrap;
    position: relative;

    .info {
      padding-left: 1rem;
      display: inline-flex;
      width: 512px;
      align-items: center;
      background-color: azure;

      .name {
        font-size: 1.2rem;
        margin-left: 1rem;
        a{
          text-decoration: none;
          color: rgb(28, 28, 28);
        }
      }
    }

    .progress-bar {
      width: 100%;
      display: inline-flex;
      flex-flow: row wrap;
      justify-content: space-between;
      padding-right: 1rem;

      .content {
        padding: 0 1rem;
        position: absolute;
        overflow-wrap: break-word;
        font-size: 1rem;
        color: #fff;
        word-break: break-word;
        width: 480px;
        background-color: transparent;
      }

    }
  }
  @keyframes in {
  from {
    transform: translateY(-100%);
    opacity: 0;
  }
  to{
    opacity: 1;
  }
  
}
}


</style>