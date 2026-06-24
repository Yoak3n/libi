<template>

</template>


<script setup lang="ts">
import { useRouter } from "vue-router";
import { onMounted } from "vue";
import * as AuthService from '../../bindings/bdanmu/app/service/authservice'
import { useRoomStore } from '@/store'

const $router = useRouter()
const roomStore = useRoomStore()
onMounted(async () => {
    const authed = await AuthService.CheckLogin()
    if (!authed) {
        localStorage.removeItem("cookie")
        localStorage.removeItem("token")
        await $router.push("/login")
        return
    }
    await roomStore.syncRoomId()
    if (roomStore.room_id && roomStore.room_id > 0) {
        await $router.push({name:"Dashboard",query:{from:"home"}})
    } else {
        await $router.push({name:"Setting",query:{from:"home"}})
    }
}
)
</script>