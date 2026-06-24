import { createRouter, createWebHashHistory } from "vue-router";
import * as AuthService from '../../bindings/bdanmu/app/service/authservice'

import routes from './routes'
const router =  createRouter({ 
    history: createWebHashHistory(),
    routes,
})

router.beforeEach(async(to, _, next) => {
    if (to.meta.requireAuth) {
        // 判断该路由是否需要登录权限
        const auth = await AuthService.CheckLogin()
        if (!auth) {
            console.log("need login")
            next({
                path: '/login',
                query: {
                    redirect: to.fullPath
                }
            })
        }else{
            console.log("auth passed")
            next()
        }
    }else{
        next()
    }
})


export default router