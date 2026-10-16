<script setup>
// ============================================================
// SiteHeader —— 公共头部（小米风：最左头像，最右导航+登录区）
// 逻辑："我是谁"存在全站共享的 stores/me.js 里
//       组件挂载时问一次；登录/登出时由对应页面调 fetchMe() 同步
// ============================================================

import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { me, fetchMe } from '@/stores/me'   // 共享状态：登录人信息
import defaultAvatar from '@/assets/images/avatar-default.svg' // 默认灰头像（没传头像的人用）
import AvatarBurst from '@/components/AvatarBurst.vue' // 点了会迸发小图案的头像（可复用组件）
import logoImg from '@/assets/images/clmqq.jpg'        // 左上角 logo 用的头像图片

const router = useRouter()
const route = useRoute()   // 读当前路由：留言页要让 header 换装成透明模式

// 留言页 / 爱心页：头部要换成"透明浮层"模式（这两个页面下面是全屏背景，不需要头部分层）
const isGhost = computed(() => route.path === '/guestbook' || route.path === '/heart')

// ---------- 滚动感知：页面在顶部时头部"隐形"，滚下去才浮出阴影 ----------
// 目的：顶部时头部和纯白页面融为一体（没有那条"横带"感），
//      滚动后内容从底下穿过，再浮出阴影把头部"托"起来
const scrolled = ref(false)   // 页面是否已经滚动离开顶部

function onScroll() {
  scrolled.value = window.scrollY > 8   // 滚过 8px 就算"离开顶部"（留一点容错）
}

onMounted(() => {
  fetchMe()   // 问一次"我是谁"（me 是共享 ref，值一变 header 自动更新）
  // passive: true = 告诉浏览器"这监听器不会阻止滚动"，滚动性能更好
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()  // 万一是刷新在页面中间打开的，先手动算一次
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)  // 销毁时摘掉监听，防止内存泄漏
})

// ---------- 用户菜单（LeetCode 式下拉面板） ----------
const menuOpen = ref(false)

// 面板里的跳转：点了顺手关面板
function go(path) {
  menuOpen.value = false
  router.push(path)
}

// 退出登录：扔掉门禁卡 + 清空共享状态 + 关面板 + 回登录页
function logout() {
  localStorage.removeItem('token')
  me.value = null        // header 立刻变回"登录|注册"，不用整页刷新
  menuOpen.value = false
  router.push('/login')
}
</script>

<!-- 样式外置：scoped 隔离照样生效 -->
<style scoped src="@/views/css/SiteHeader.css"></style>

<template>
  <!-- 留言页 / 爱心页：header-ghost 透明浮层模式
       爱心页再叠加 header-dark：深空底上的导航文字换浅色 -->
  <header
    class="site-header"
    :class="{
      'header-ghost': isGhost,
      'header-dark': route.path === '/heart',
      'is-scrolled': scrolled && !isGhost,
    }"
  >
    <div class="header-inner">
      <!-- 最左：头像当 logo，点击回首页；顺手带个"迸发小图案"的特效
           （已经在首页时点击不跳转，正好把特效看全；
             在别的页面点击会边放特效边跳回首页，跳转不受影响） -->
      <RouterLink class="logo" to="/">
        <AvatarBurst :src="logoImg" :size="40" alt="本站作者头像" />
      </RouterLink>

      <!-- 右侧：导航 + 登录区 挨在一起 -->
      <div class="header-right">
        <!-- 导航：每项前面带一个小图标（emoji），和"💖 爱心"保持同一套视觉语言 -->
        <nav class="nav">
          <RouterLink to="/">🏠 主页</RouterLink>
          <RouterLink to="/talks">💬 说说</RouterLink>
          <RouterLink to="/album">📷 相册</RouterLink>
          <RouterLink to="/guestbook">🌸 留言</RouterLink>
          <RouterLink to="/heart">💖 爱心</RouterLink>
        </nav>

        <!-- 竖分割线：导航和登录区的视觉分隔（学小米的 登录|注册） -->
        <span class="header-divider"></span>

        <!-- 登录区：v-if/v-else 二选一 -->
        <div class="auth">
          <template v-if="me">
            <!-- 点头像：弹出 LeetCode 式用户菜单（不再直接跳个人空间） -->
            <div class="user-menu-wrap">
              <button class="header-user" @click="menuOpen = !menuOpen">
                <img class="header-avatar" :src="me.avatar || defaultAvatar" alt="头像" />
                <span class="hello">{{ me.nickname || me.username }}</span>
                <span class="menu-caret">▾</span>
              </button>

              <!-- 全屏透明遮罩：点面板外面任意一处 = 关闭面板 -->
              <div v-if="menuOpen" class="menu-overlay" @click="menuOpen = false"></div>

              <!-- 用户菜单面板 -->
              <div v-if="menuOpen" class="user-menu">
                <!-- 顶部：头像 + 昵称，点击进个人空间 -->
                <button class="um-head" @click="go('/user/' + me.user_id)">
                  <img class="um-avatar" :src="me.avatar || defaultAvatar" alt="头像" />
                  <span class="um-info">
                    <span class="um-name">{{ me.nickname || me.username }}</span>
                    <span class="um-go">查看个人空间 ›</span>
                  </span>
                </button>

                <!-- 底部：退出（页面入口都在导航栏，菜单里就不重复摆了） -->
                <div class="um-footer">
                  <button class="um-logout" @click="logout">⏻ 退出登录</button>
                </div>
              </div>
            </div>
          </template>
          <template v-else>
            <RouterLink to="/login">登录</RouterLink>
            <span class="auth-sep">|</span>
            <RouterLink to="/register">注册</RouterLink>
          </template>
        </div>
      </div>
    </div>
  </header>
</template>
