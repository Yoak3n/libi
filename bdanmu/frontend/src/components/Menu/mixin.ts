import { Browser, Application, Window } from '@wailsio/runtime'

export const JumpToLiveRoom =(e:MouseEvent)=> {
    e.preventDefault()
    const room_id:string = localStorage.getItem('room_id')!
    Browser.OpenURL('https://live.bilibili.com/' + room_id)
}

export const AppQuit = ()=>{
    Application.Quit()
}

export const HideWindow =(e:MouseEvent)=> {
    e.preventDefault()
    Application.Hide()
}

export const TopWindow = (e:MouseEvent,flag:boolean)=> {
    e.preventDefault();
    Window.SetAlwaysOnTop(!flag);
}
