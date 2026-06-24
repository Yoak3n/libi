<template>
  <n-list hoverable clickable class="menu-wrapper">
    <n-list-item class="menu-item" @click="() => {
      if ($route.path != '/dashboard') {
        $router.push({ name: 'Dashboard', query: { from: $route.path.split('/')[1] } })
      }
      appStore.setDrawer(false)
    }">
      <template #prefix>
        <n-icon>
          <barcode />
        </n-icon>
      </template>
      弹幕流
    </n-list-item>
    <n-list-item class="menu-item" @click="(e) => {
      JumpToLiveRoom(e)
      appStore.setDrawer(false)
    }">
      <template #prefix>
        <n-icon>
          <book-icon />
        </n-icon>
      </template>
      跳转直播间
    </n-list-item>
    <n-list-item class="menu-item" @click="(e) => {
      HideWindow(e)
      appStore.setDrawer(false)
    }">
      <template #prefix>
        <n-icon>
          <caret-down />
        </n-icon>
      </template>
      隐藏至托盘
    </n-list-item>
    <n-list-item class="menu-item" @click="(e) => {
      console.log(on_top);

      TopWindow(e, on_top);
      appStore.setOnTop();
      appStore.setDrawer(false)
    }">
      <template #prefix>
        <n-icon v-if="on_top">
          <lock-open-outline />
        </n-icon>
        <n-icon v-else>
          <lock-closed-outline />
        </n-icon>
      </template>
      {{ on_top ? '取消置顶' : '窗口置顶' }}

    </n-list-item>
    <n-list-item class="menu-item" @click="() => {
      $router.push('/setting')
      appStore.setDrawer(false)
    }">
      <template #prefix>
        <n-icon>
          <settings />
        </n-icon>
      </template>
      设置
    </n-list-item>

    <n-list-item class="menu-item" @click="AppQuit">
      <template #prefix>
        <n-icon>
          <logout />
        </n-icon>
      </template>
      退出
    </n-list-item>
  </n-list>
  <n-modal>

  </n-modal>

</template>
<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { NList, NListItem, NIcon, NModal } from 'naive-ui'
import {
  BookOutline as BookIcon,
  // PersonOutline as PersonIcon,
  LogOutOutline as Logout,
  SettingsOutline as Settings,
  BarcodeOutline as Barcode,
  CaretDownCircleOutline as CaretDown,
  LockClosedOutline,
  LockOpenOutline

} from '@vicons/ionicons5'

import { JumpToLiveRoom, AppQuit, HideWindow, TopWindow } from './mixin'
import { useAppStore } from '@/store'
const appStore = useAppStore()
let { on_top } = storeToRefs(appStore)


</script>

<style lang="less" scoped>
.menu-wrapper {
  width: 100%;
  font-size: large;
  background-color: transparent;
  color: rgb(240, 240, 240);

  .menu-item {
    width: 100%;
    height: 100%;
    border-radius: 0;
  }

  .menu-item:hover {
    background-color: rgba(0, 0, 0, 0.5);
    box-shadow: 0 0 10px 2px #918f8f;
    transition: all 0.3s;
  }

}
</style>