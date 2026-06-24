import type{User} from '../types'
export interface Danmu{
    user:User
    content:string
    type:danmuType
    room_id:number
    message_id:string
}

export enum danmuType{
    UserEntry = 1,
    Danmu = 2,
    EmoticonDanmu = 3
}


export function selectColor(level:number):danmuLevelColor{
    if (level >0 &&level < 5) {
        return danmuLevelColor.green
    } else if (level < 9) {
        return danmuLevelColor.blue
    } else if (level < 13) {
        return danmuLevelColor.purple
    }else if (level < 17) {
        return danmuLevelColor.pink
    } else if (level < 21){
        return danmuLevelColor.yellow
    }else if (level < 25){
        return danmuLevelColor.deep_green
    }else if (level <29){
        return danmuLevelColor.deep_blue
    }else if (level < 33){
        return danmuLevelColor.deep_purple
    }else if (level < 37){
        return danmuLevelColor.deep_pink
    }else{
        return danmuLevelColor.orange
    }
}

export enum danmuLevelColor{
    green = "#42b983",
    blue = "#5d7c9b",
    purple = "#8d7aaf",
    pink = "#bc6786",
    yellow= "#c89d24",
    deep_green = "linear-gradient(to right,#215b4f,#579588)",
    deep_blue = "linear-gradient(to right,#102052,#7791c2)",
    deep_purple = "linear-gradient(to right,#351b63,#7c68bd)",
    deep_pink = "linear-gradient(to right,#861436,#c25884)",
    orange = "linear-gradient(to right,#fe7016,#fea757)",
}