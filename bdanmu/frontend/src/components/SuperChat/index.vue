<script setup lang="ts">
import { ref, PropType} from 'vue';
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
const superChatRef = ref<undefined | HTMLElement>(undefined)
const color = computeSuperChatBackground(props.data.price)
const long = props.data.end_time - props.data.timestamp
let height = 30
</script>

<template>
  <div class="super-chat-wrapper" ref="superChatRef" style="z-index: 99999;" >
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
  animation: in 1s ease-in-out;
  .super-chat-box {
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