<template>
    <div class="dashboard-wrapper" ref="containerRef">
        <n-affix :trigger-top="0" :listen-to="() => containerRef" class="super-chat-box">
            <Transition name="sc">
                <div v-if="superChats.length > 0" class="super-chat-list">
                    <SuperChatbox v-for="superChat in superChats" :key="superChat.message_id" :data="superChat"
                        @done="removeSuperChat(superChat.message_id)" />
                </div>
            </Transition>
            <!-- <button @click="testSuperChat">test</button> -->
        </n-affix>
        <n-infinite-scroll ref="danmuBoxRef" class="danmu-box">
            <transition-group name="fade" tag="div">
                <Danmubox v-for="(danmu, index) in danmus" :id="index == danmus.length - 1 ? 'bottom' : ''"
                    :key="danmu.message_id" class="danmu-item" :danmu="danmu" />
            </transition-group>
        </n-infinite-scroll>
    </div>
</template>
<script setup lang="ts">
import { ref, onMounted, nextTick, onActivated, onBeforeUnmount } from 'vue';
import { NAffix, NInfiniteScroll } from 'naive-ui'
import { useRoute, useRouter } from 'vue-router';
import { useRoomStore } from '@/store'
import Danmubox from '../components/Danmu/index.vue'
import SuperChatbox from '../components/SuperChat/index.vue'
import type { Danmu } from '../components/Danmu/danmu'
import type { SuperChat } from '../components/SuperChat/super_chat';
import { type User, type Room, type Message, MessageType } from '../components/types'
import { Events } from '@wailsio/runtime'

const roomsStore = useRoomStore()
const containerRef = ref<HTMLElement | undefined>(undefined)
const danmuBoxRef = ref<InstanceType<typeof NInfiniteScroll> | undefined>(undefined)
const $route = useRoute()
const $router = useRouter()
let danmus = ref<Array<Danmu>>([])
let superChats = ref<Array<SuperChat>>([])

let autoScroll = true
let scrollPauseTimer: ReturnType<typeof setTimeout> | null = null

function onWheel(e: WheelEvent) {
    if (e.deltaY < 0) {
        autoScroll = false
        if (scrollPauseTimer) clearTimeout(scrollPauseTimer)
        scrollPauseTimer = setTimeout(() => { autoScroll = true }, 5000)
    }
}

onActivated(() => {
    if ($route.query.from == "login") {
        danmus.value = []
        superChats.value = []
    } else if ($route.query.from == "setting") {
        danmus.value = []
    } else {
        if (roomsStore.room_id == 0 || roomsStore.room_id == null) {
            window.$message.error("未找到直播间", { keepAliveOnHover: true })
            $router.push({ name: 'Setting', query: { from: 'dashboard' } })
        } else {
            Events.Emit("change", roomsStore.room_id)
        }
    }
})

onMounted(() => {
    const el = danmuBoxRef.value?.$el as HTMLElement | undefined
    if (el) el.addEventListener('wheel', onWheel, { passive: true })

    Events.On('started', function (ev: any) {
        const room: Room = JSON.parse(ev.data as string)
        roomsStore.setRoomTitle(room.title)
        roomsStore.setRoomId(room.short_id)
        window.$message.create("已连接房间：" + room.short_id, { duration: 5000 })
        Events.Off("message")
        Events.On("message", reciveMessage)
    })
    Events.On('error', function (ev: any) {
        window.$message.error(ev.data as string, { keepAliveOnHover: true, duration: 5000 })
        $router.push({ name: 'Setting', query: { from: 'dashboard' } })
    })
})

onBeforeUnmount(() => {
    const el = danmuBoxRef.value?.$el as HTMLElement | undefined
    if (el) el.removeEventListener('wheel', onWheel)
    if (scrollPauseTimer) clearTimeout(scrollPauseTimer)
})


const pushSuperChat = (super_chat: SuperChat) => {
    superChats.value.push(super_chat)
}

const removeSuperChat = (message_id: string) => {
    const idx = superChats.value.findIndex(sc => sc.message_id === message_id)
    if (idx !== -1) superChats.value.splice(idx, 1)
}

const pushDanmu = (danmu: Danmu) => {
    if (danmus.value.length > 200) {
        danmus.value.shift()
    }
    danmus.value.push(danmu)
    if (autoScroll) {
        nextTick(() => {
            const bottom = document.getElementById("bottom")
            bottom?.scrollIntoView({ behavior: "smooth", block: "center", inline: "end" });
        })
    }
}
const reciveMessage = (ev: any) => {
    const message: Message = ev.data
    switch (message.type) {
        case MessageType.Danmu:
            pushDanmu(message.data as Danmu)
            break;
        case MessageType.SuperChat:
            pushSuperChat(message.data as SuperChat)
            break;
        case MessageType.User:
            updateUser(message.data as User)
            break;
        default:
            break;
    }
}

const updateUser = (user: User) => {
    console.log(user.uid,":", user.avatar)
    for (const danmu of danmus.value) {
        if (danmu.user.uid === user.uid) {
            danmu.user.avatar = user.avatar
            danmu.user.sex = user.sex
            danmu.user.fans_count = user.fans_count
        }
    }
}

</script>

<style scoped lang="less">
.fade-move {
    transition: all .5s ease;
}

.fade-leave-active,
.fade-enter-active {
    transition: all .5s ease;
}

.fade-leave-to,
.fade-enter-from {
    opacity: 0;
    transform: translateX(-30px);
}

.fade-leave-active {
    position: absolute;
}

.dashboard-wrapper {
    height: 100%;
    width: 100%;

    // margin: 0 2rem;
    .super-chat-box {
        width: 100%;
        pointer-events: none;
        z-index: 10;
    }

    .super-chat-list {
        pointer-events: none;
    }

    .danmu-box {
        height: 100%;
        width: 100%;
        overflow-y: scroll;
        position: relative;
    }

}

.sc-leave-active {
    transition: opacity 1s ease;
}

.sc-leave-from {
    opacity: 1;
}

.sc-leave-to {
    opacity: 0;
}
</style>