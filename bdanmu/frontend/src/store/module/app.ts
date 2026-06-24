import { defineStore } from "pinia";
import * as AuthService from '../../../bindings/bdanmu/app/service/authservice';
export const useAppStore = defineStore("app", {
    state: () => {
        return {
            cookie: "",
            token: "",
            drawer_open: false,
            on_top:false,
        };
    },
    actions: {
        setCookie(cookie: string) {
            this.cookie = cookie
            localStorage.setItem('cookie', cookie)
        },
        setToken(token: string){
            this.token = token
            localStorage.setItem('token', token)
        },
        setDrawer(open: boolean){
            this.drawer_open = open
        },
        setOnTop(){
            this.on_top = !this.on_top
        },
        async syncAuth (){
          const auth = await AuthService.SyncAuth()
          if (auth[0] != '' || auth[1] != '') {
            this.setCookie(auth[0])
            this.setToken(auth[1])
          }else{
            this.setCookie('')
            this.setToken('')
          }
        }
    }
})