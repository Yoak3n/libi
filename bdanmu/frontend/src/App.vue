<script setup lang="ts">
import { onMounted } from 'vue'
import { NFloatButton, NIcon, NDrawer, NDrawerContent, NScrollbar } from 'naive-ui'
import { MenuOutline } from '@vicons/ionicons5'
import NaiveProvider from './components/NaiveProvider/index.vue'
import Menu from './components/Menu/index.vue'
import { useRoute } from 'vue-router'
import { useRoomStore, useAppStore } from './store'
import { storeToRefs } from 'pinia'

const roomsStore = useRoomStore()
const appStore = useAppStore()
const $route = useRoute()

let { room_title: title } = storeToRefs(roomsStore)
let { drawer_open: drawer_open } = storeToRefs(appStore)
onMounted(() => {
  appStore.syncAuth()
  roomsStore.syncRoomId()
})
</script>

<template>
  <naive-provider>
    <router-view v-slot="{ Component }">
      <keep-alive>
        <component :is="Component" v-if="$route.meta.keepAlive" />
      </keep-alive>
      <component :is="Component" v-if="!$route.meta.keepAlive" />
    </router-view>
  </naive-provider>
  <div>
    <n-float-button :right="0" :bottom="0" shape="square" @click="appStore.setDrawer(true)" class="menu">
      <n-icon>
        <menu-outline />
      </n-icon>
    </n-float-button>
    <n-drawer v-model:show="drawer_open" show-mask="transparent"
      style="background-color: rgba(30,30,30,0.6);margin: 3rem 0;border: 1px solid rgba(240,240,240,0.6);border-radius:  10px 0 0 10px"
      width="40%">
      <n-drawer-content body-content-style="padding: 0;">
        <template #header>
          <div style="color:rgb(240,240,240);cursor: default;user-select: none" onselectstart="return false"
            unselectable="on">
            {{ title }}
          </div>
        </template>
        <n-scrollbar style="max-height: 500px">
          <Menu />
        </n-scrollbar>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<style scoped lang="less">
.menu {
  color: transparent;
  background-color: transparent;
  border: none;
  box-shadow: none;
}

.menu:hover {
  animation-duration: .5s;
  animation-name: fadeIn;
  color: #1c1c1c;
  background-color: rgba(240, 240, 240, 1);
}

.menu:hover div {
  animation-duration: .5s;
  animation-name: fadeIn;
  background-color: rgba(240, 240, 240, 1);
}

@keyframes fadeIn {
  from {
    background-color: rgba(6, 7, 15, 0);
  }

  to {
    background-color: rgba(240, 240, 240, 1);
  }
}
</style>
