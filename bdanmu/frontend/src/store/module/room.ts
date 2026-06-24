import { defineStore } from 'pinia'
import * as AuthService from '../../../bindings/bdanmu/app/service/authservice'
import type{ Room } from '../../components/types'

// 第一个参数是应用程序中 store 的唯一 id
export const useRoomStore = defineStore('room', {
  state:()=> {
      return {
          room_id: 0 as number || null,
          room_title: '',
      }
  },
  actions: {
      // 初始化
      async syncRoomId(){
        const saved = localStorage.getItem('room_id')
        if (saved) {
          this.room_id = Number(saved)
          return
        }
        const id = await AuthService.SyncRoomId()
        if (id > 0) {
          this.room_id = id
        }
      },
      setRoomId(id: number) {
          this.room_id = id
          localStorage.setItem('room_id', id.toString())
      },
      setRoomTitle(title: string){
        this.room_title = title
        localStorage.setItem('room_title', title)
      },
      setRoom(room:Room){
        localStorage.setItem('room', room.toString())
        this.room_id = room.short_id
        this.room_title = room.title
      }
  },
  getters: {
      getRoom: (state) => state.room_id
  }
})