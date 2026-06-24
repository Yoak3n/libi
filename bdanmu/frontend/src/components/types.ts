export interface User{
    uid:number
    name:string
    sex:number
    guard:boolean
    avatar:string
    fans_count?:number
    medal?:Medal
}

interface Medal {
    name:string,
    owner_id:number,
    level:number,
    target_id:number
}

export interface Room{
    short_id:number,
    user?:User,
    title:string,
    cover:string,
    long_id:number,
    follower_count:number,
}

export const MessageType = {
    SuperChat: "super_chat",
    UserEntry: "user_entry",
    Danmu: "danmu",
    User: "user",
} as const

export type MessageType = typeof MessageType[keyof typeof MessageType]

import type {Danmu} from './Danmu/danmu'
import type {SuperChat} from './SuperChat/super_chat'
export interface Message {
    type: MessageType,
    data: Danmu|SuperChat|User,
}