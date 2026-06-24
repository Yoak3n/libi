import type{ MenuOption} from 'naive-ui'
import { NIcon} from 'naive-ui'
import {  h, Component } from 'vue'
import {JumpToLiveRoom} from './mixin'
import {
    BookOutline as BookIcon,
    PersonOutline as PersonIcon,
    LogOutOutline as Logout
} from '@vicons/ionicons5'

export const menuOptions: MenuOption[] = [
    {
        label: ()=>h('a',{
            onClick:JumpToLiveRoom
        }, {
            default: () => "直播间信息"}),
        key: 'room_info',
        icon: renderIcon(PersonIcon)
    },
    {
        label: '重启弹幕机',
        key: 'restart',
        icon: renderIcon(BookIcon),
    },
    {
        label: '退出',
        key: 'quit',
        icon: renderIcon(Logout),
    },
]


function renderIcon (icon: Component) {
    return () => h(NIcon, null, { default: () => h(icon) })
}
