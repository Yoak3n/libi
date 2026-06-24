<template>
    <div class="medal" :style="{background:color}">
        <span class="medal-name">
            {{ medal_name }}
        </span>
        <span class="medal-level" :style="{ color: level>20?level<29? '#444':selectColor(level-20):color }">
            {{ props.level }}
        </span>
    </div>
</template>
<script setup lang="ts">
import { ref } from 'vue';
import {danmuLevelColor,selectColor} from '../Danmu/danmu'
const props = defineProps({
    name: {
        type: String,
        required: true
    },
    level: {
        type: Number,
        required: true
    }
})
let color = ref<danmuLevelColor>()
let medal_name= ref(props.name)
color.value = selectColor(props.level)

if (props.name.length > 3) {
    medal_name.value = props.name.slice(0,3)
}
</script>

<style scoped lang="less">
.medal{
    min-width: 15%;
    text-align: center;
    padding: 3px;
    color: white;
    display:flex;
    font-size: 15px;
    align-items: center;
    height: 1rem;
    justify-content: space-between;
    margin: 0.1rem 0.5rem 0  0 ;
    border-radius: 5px;
    line-height: 1rem;
    .medal-level{
        border-radius:3px ;
        width:auto;
        margin: 0 1px 0 3px;
        padding: 0 1px;
        font-size: 13px;
        background-color: white;
    }
    .medal-name{
        margin: 0 auto;
        border-radius: 5px 0 0 5px;
        width: auto;

    }
}

</style>