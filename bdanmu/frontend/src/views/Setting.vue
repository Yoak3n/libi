<template>
  <div class="setting-wrapper">
    <n-config-provider :theme="darkTheme">
      <n-card title="设置" style="width: 512px;color: white" :bordered="false" size="huge" role="dialog" embedded
        :segmented="{ content: true, footer: 'soft' }">

        <n-form style="color: white;">
          <n-form-item label="直播间ID">
            <n-input v-model:value="roomId" placeholder="请输入将连接的直播间ID" />
          </n-form-item>
          <n-form-item label="WS端口">
            <n-input v-model:value="wsPort" placeholder="WebSocket 服务端口" />
          </n-form-item>
          <n-form-item label="WS服务">
            <n-button @click="toggleWS" :type="wsRunning ? 'error' : 'success'" style="width: 50%;margin: 0 auto;">
              {{ wsRunning ? '停止' : '启动' }}
            </n-button>
          </n-form-item>
          <n-form-item>
            <n-button @click="saveSettingAndRestart" type="primary" style="width: 50%;margin: 0 auto;">保存并连接</n-button>
          </n-form-item>
        </n-form>
      </n-card>
    </n-config-provider>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoomStore } from '@/store';
import { Events } from '@wailsio/runtime';
import { useRouter } from 'vue-router'
import * as AuthService from '../../bindings/bdanmu/app/service/authservice'
import {
  NConfigProvider,
  NCard,
  NForm,
  NFormItem,
  NInput,
  NButton,
  NSwitch,
  darkTheme
} from 'naive-ui';

const $router = useRouter()
const roomStore = useRoomStore()
let roomId = ref('')
let wsPort = ref('10421')
let wsRunning = ref(false)

onMounted(async () => {
  if (roomStore.room_id && roomStore.room_id > 0) {
    roomId.value = String(roomStore.room_id)
  }
  const savedPort = localStorage.getItem('ws_port')
  if (savedPort) {
    wsPort.value = savedPort
  }
  try {
    wsRunning.value = await AuthService.IsWSRunning()
  } catch {}
})

const toggleWS = async () => {
  if (wsRunning.value) {
    await AuthService.StopWS()
    wsRunning.value = false
  } else {
    const port = Number(wsPort.value)
    if (port > 0) {
      localStorage.setItem('ws_port', String(port))
      await AuthService.StartWS(port)
      wsRunning.value = true
    }
  }
}

const saveSettingAndRestart = () => {
  const port = Number(wsPort.value)
  if (port > 0) {
    localStorage.setItem('ws_port', String(port))
  }
  $router.push({ name: 'Dashboard', query: { from: 'setting' } })
  Events.Emit("change", Number(roomId.value))
}
</script>

<style scoped lang="less">
.setting-wrapper {
  padding-top: 3rem;
}
</style>
