<script setup>
// ============================================================
// AvatarBurst —— "点了会迸发小图案"的头像（可复用组件）
// 用法：<AvatarBurst :src="图片地址" :size="40" alt="头像" />
// 特点：头像外面如果套着 <RouterLink>，点击时"特效照放 + 跳转照走"
// ============================================================

import { ref } from 'vue'

// defineProps：声明"外面可以传什么进来"（父组件传值 = props）
const props = defineProps({
  src: { type: String, required: true }, // 图片地址（父组件 import 图片后传进来）
  size: { type: Number, default: 64 },   // 头像直径，单位 px
  alt: { type: String, default: '头像' }, // 图片描述（读屏软件用，别省）
  hint: { type: String, default: '' },   // 鼠标悬停的小提示，不传就没有
})

const burst = ref([]) // 正在飞的小图案（空数组 = 没特效在播）

// 点一下：造 10 个小图案，方向/距离/旋转/大小/时长全都随机
function play() {
  const icons = ['💖', '✨', '🌸', '💫']
  burst.value = Array.from({ length: 10 }, (_, i) => {
    // 一整圈 360° 均分 10 份，再加点随机偏移（太规整就"一眼假"）
    const angle = (Math.PI * 2 * i) / 10 + Math.random() * 0.6
    // 飞多远：跟着头像大小走（小的头像炸开小一点，才不会飞得满屏都是）
    const dist = props.size * 0.7 + Math.random() * 32
    return {
      id: Date.now() + i,                 // 唯一编号，给 v-for 当 key
      icon: icons[Math.floor(Math.random() * icons.length)],
      dx: Math.cos(angle) * dist,         // 横向飞多远（三角函数把角度翻译成坐标）
      dy: Math.sin(angle) * dist,         // 纵向飞多远
      rot: (Math.random() - 0.5) * 240,   // 边飞边转多少度
      size: 9 + Math.random() * 7,        // 字号（图案大小）
      dur: 0.7 + Math.random() * 0.4,     // 这一颗飞多久
    }
  })
  // 播完就清空 —— 不清的话连点十下 DOM 里会堆一堆元素
  setTimeout(() => {
    burst.value = []
  }, 1200)
}
</script>

<style scoped src="@/views/css/AvatarBurst.css"></style>

<template>
  <!-- 外面这层 span：接住点击 + 给小图案当定位基准 -->
  <span
    class="ab-wrap"
    :style="{ '--ab-size': size + 'px' }"
    :title="hint || undefined"
    @click="play"
  >
    <img class="ab-img" :src="src" :alt="alt" />

    <!-- 点击迸发出来的小图案：飞出去 + 边飞边转 + 渐隐 -->
    <span
      v-for="b in burst"
      :key="b.id"
      class="ab-burst"
      aria-hidden="true"
      :style="{
        '--dx': b.dx + 'px',
        '--dy': b.dy + 'px',
        '--rot': b.rot + 'deg',
        fontSize: b.size + 'px',
        animationDuration: b.dur + 's',
      }"
      >{{ b.icon }}</span
    >
  </span>
</template>
