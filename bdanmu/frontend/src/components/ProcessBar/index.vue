<template>
  <div class="process-bar":style="backStyle">
    <div class="progress-bar-fill" :style="fillStyle"></div>    
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps(
    {
        progressTime:{
            type:Number,
            required:true,
        },
        width:{
          type:Number,
          required:true  
        },
        height:{
            type:Number,
            default:10
        },
        backColor:{
            type:String,
            default:"rgba(28, 28, 28,0.8)"
        },
        fillColor:{
            type:String,
            required:true,
        }
    }
)

const {progressTime,height,width,backColor,fillColor} = props

const backStyle = computed(() => {
  return {
    "background-color":`${backColor}`,
    "height":`${height}px`,
    "width":`${width}%`
  }
})

const fillStyle = computed(() => {
  return {
    "background-color":`${fillColor}`,
    "height":`${height}px`,
    "width":0,
  }
})
const fillWidth = computed(() => {
  return `${width}px`
})
const fillTime = `${progressTime}s`

</script>

<style scoped lang="less">
.process-bar {
  display: inline-block;
  .progress-bar-fill {
    animation-duration: v-bind(fillTime);
    animation-name: progress;
    -webkit-animation-name: progress;
}
@keyframes progress {
    from {
        width: v-bind(fillWidth);
    }
    99%{
      width: 3px;
    }
    to {
        // margin-left:0;
        width:0;
    }
}

}
</style>