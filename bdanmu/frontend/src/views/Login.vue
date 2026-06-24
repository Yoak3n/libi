<template>
  <div class="login-wrapper">
    <n-card v-if="text !== ''" class="qrcode" title="请使用哔哩哔哩App扫码登录">
      <n-qr-code :value="text"  :size="250" color="#f69"/>
    </n-card>
  </div>
</template>
<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { NCard, NQrCode,useMessage } from 'naive-ui';
import {useRouter} from  'vue-router'
import { useRoomStore,useAppStore } from '@/store';
import { Events } from '@wailsio/runtime'
import * as AuthService from '../../bindings/bdanmu/app/service/authservice'

let text = ref("")
const $router = useRouter()
const $message = useMessage()
const appStore = useAppStore()
const roomStore = useRoomStore()
onMounted(async() => {
  Events.Once("auth:qr-url", (ev) => {
    text.value = ev.data as string
  })
  Events.Once("auth:login-result", auth)
  await AuthService.StartLogin()
})

const start = async () => {
  if (roomStore.room_id !== 0) {
    Events.Emit("change",Number(roomStore.room_id))
    $router.push({name: 'Dashboard', query: {from: 'login'}})
  }else{
    $router.push({name: 'Setting', query: {from: 'login'}})
    window.$message.error("未找到直播间",{keepAliveOnHover: true})
  }
}

const auth = (ev: any) => {
  const success = ev.data as boolean
  if (success) {
    appStore.syncAuth()
    $message.success("登录成功",{keepAliveOnHover: true})
    start()
  } else {
    $message.error("登录失败",{keepAliveOnHover: true})
  }
}

</script>

<style scoped lang="less">
.login-wrapper {
  color: #fff;
  margin-top: 3rem;
  .qrcode {
    width: 63%;
    margin: 0 auto;
  }

}
</style>