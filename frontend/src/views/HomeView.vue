<script setup>
// ============================================================
// HomeView —— 主页（纯展示门面，不承载内容）
// 结构：Hero 欢迎区（樱花飘浮装饰）→ 轮播图 → 入口卡片
// 正经内容（说说流）住在 /talks；主页只负责"好看 + 指路"
// ============================================================

import { useRouter } from 'vue-router'

const router = useRouter()

// 轮播图幻灯片：渐变底 + 图标 + 一句话，点击整张直达对应页面
const slides = [
  {
    icon: '🌸',
    title: '一个记录生活的小站',
    desc: '没有正经文章，只有朋友们的日常碎片',
    bg: 'linear-gradient(135deg, #fce7f3 0%, #ede9fe 100%)',
    to: '/talks',
  },
  {
    icon: '💬',
    title: '说说流',
    desc: '今天吃了什么、看了什么，随手一条',
    bg: 'linear-gradient(135deg, #dbeafe 0%, #e0f2fe 100%)',
    to: '/talks',
  },
  {
    icon: '📷',
    title: '相册',
    desc: '照片和视频都住在这里',
    bg: 'linear-gradient(135deg, #ecfccb 0%, #d9f99d 100%)',
    to: '/album',
  },
]

// 入口卡片：主页的四个"大门"
const entries = [
  { icon: '💬', title: '说说', desc: '朋友们的日常动态', to: '/talks' },
  { icon: '📷', title: '相册', desc: '照片与视频的存放处', to: '/album' },
  { icon: '🌸', title: '留言板', desc: '樱花雨里留一句话', to: '/guestbook' },
  { icon: '💖', title: '爱心', desc: '粒子爱心特效页', to: '/heart' },
]
</script>

<style scoped src="@/views/css/HomeView.css"></style>

<template>
  <div class="home-page">
    <!-- Hero 欢迎区：樱花 SVG 飘浮装饰 + 站点标语 + 两个入口按钮 -->
    <section class="hero">
      <!-- 飘浮樱花装饰：SVG 花瓣，各自不同的大小/位置/节奏 -->
      <svg class="petal p1" viewBox="0 0 24 24"><path d="M12 2C15 6 20 8 20 13C20 18 16 22 12 22C8 22 4 18 4 13C4 8 9 6 12 2Z" fill="#f9a8d4" /></svg>
      <svg class="petal p2" viewBox="0 0 24 24"><path d="M12 2C15 6 20 8 20 13C20 18 16 22 12 22C8 22 4 18 4 13C4 8 9 6 12 2Z" fill="#fbcfe8" /></svg>
      <svg class="petal p3" viewBox="0 0 24 24"><path d="M12 2C15 6 20 8 20 13C20 18 16 22 12 22C8 22 4 18 4 13C4 8 9 6 12 2Z" fill="#f472b6" /></svg>
      <svg class="petal p4" viewBox="0 0 24 24"><path d="M12 2C15 6 20 8 20 13C20 18 16 22 12 22C8 22 4 18 4 13C4 8 9 6 12 2Z" fill="#fbcfe8" /></svg>

      <h1 class="hero-title">记录生活的小站</h1>
      <p class="hero-sub">说说、相册、留言——朋友们的日常都在这里</p>
      <div class="hero-actions">
        <button class="btn-hero" @click="router.push('/talks')">去逛说说</button>
        <button class="btn-hero ghost" @click="router.push('/album')">翻翻相册</button>
      </div>
    </section>

    <!-- 轮播图：自动播放，点击整张直达 -->
    <div class="banner-card">
      <el-carousel height="300px" :interval="4200" arrow="hover" indicator-position="none">
        <el-carousel-item v-for="s in slides" :key="s.title">
          <div class="slide" :style="{ background: s.bg }" @click="router.push(s.to)">
            <span class="slide-icon">{{ s.icon }}</span>
            <h2 class="slide-title">{{ s.title }}</h2>
            <p class="slide-desc">{{ s.desc }}</p>
            <span class="slide-go">进去看看 ›</span>
          </div>
        </el-carousel-item>
      </el-carousel>
    </div>

    <!-- 入口卡片：四个大门，质感配方全套 -->
    <div class="entry-grid">
      <div v-for="e in entries" :key="e.title" class="entry-card" @click="router.push(e.to)">
        <span class="entry-icon">{{ e.icon }}</span>
        <span class="entry-title">{{ e.title }}</span>
        <span class="entry-desc">{{ e.desc }}</span>
      </div>
    </div>

    <p class="home-foot">朋友们的小站 · 慢慢更新中</p>
  </div>
</template>
