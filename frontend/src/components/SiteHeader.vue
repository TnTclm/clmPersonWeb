<script setup>
// ============================================================
// SiteHeader —— 公共头部（小米风：最左头像，最右导航+登录区）
// 逻辑："我是谁"存在全站共享的 stores/me.js 里
//       组件挂载时问一次；登录/登出时由对应页面调 fetchMe() 同步
// ============================================================

import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
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

// ---------- 未读提醒（评论/留言通知，只有博主有）----------
const unread = ref(0)         // 未读条数（红点上的数字）
const notifyOpen = ref(false) // 通知面板是否展开
const notifyItems = ref([])   // 通知列表
const notifyPage = ref(1)     // 当前第几页通知
const notifyHasMore = ref(false)   // 后面还有没有更早的
const notifyLoadingMore = ref(false)

// 拉通知：reset=true 从第 1 页重拉；false 只追加下一页
async function loadNotify(reset = true) {
  if (!me.value || !me.value.is_owner) return // 只有博主才有通知，别人不用请求
  try {
    if (reset) notifyPage.value = 1
    const res = await api.get('/api/notifications?page=' + notifyPage.value)
    const list = res.data.items
    notifyItems.value = reset ? list : notifyItems.value.concat(list)
    unread.value = res.data.unread
    notifyHasMore.value = res.has_more
  } catch (err) {
    // 通知拉不到不是错误（比如后端重启中）：静默处理，别打扰用户
  }
}

// 加载更早的通知
async function loadMoreNotify() {
  if (notifyLoadingMore.value || !notifyHasMore.value) return
  notifyLoadingMore.value = true
  notifyPage.value += 1
  await loadNotify(false)
  notifyLoadingMore.value = false
}

// 点一条通知：关掉面板 + 跳到对应的说说（留言没有 talk_id，就只关面板）
function goNotify(n) {
  notifyOpen.value = false
  if (n.talk_id) router.push('/talks/' + n.talk_id)
}

// 点铃铛：打开面板 → 刷新列表 → 红点清零 → 告诉后端"看过了"
async function openNotify() {
  notifyOpen.value = true
  await loadNotify()
  unread.value = 0 // 界面上的红点先消失（不等请求回来，手感更跟手）
  try {
    await api.post('/api/notifications/read', {})
  } catch (err) {
    // 标记已读失败也不影响看列表
  }
}

// me 是异步加载的：等它到了再拉通知（项目的标准套路）
watch(
  me,
  (val) => {
    if (val && val.is_owner) loadNotify()
  },
  { immediate: true }
)

// 定时轮询：每 30 秒问一次有没有新消息（不轮询的话要刷新页面才看得到）
let notifyTimer = null

onMounted(() => {
  fetchMe()   // 问一次"我是谁"（me 是共享 ref，值一变 header 自动更新）
  // passive: true = 告诉浏览器"这监听器不会阻止滚动"，滚动性能更好
  window.addEventListener('scroll', onScroll, { passive: true })
  onScroll()  // 万一是刷新在页面中间打开的，先手动算一次
  // 面板开着的时候不轮询 —— 否则会把你翻了好几页的列表重置回第 1 页
  notifyTimer = setInterval(() => {
    if (!notifyOpen.value) loadNotify()
  }, 30000)
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)  // 销毁时摘掉监听，防止内存泄漏
  clearInterval(notifyTimer)                      // 定时器也要停，不然会一直跑
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

        <!-- 未读提醒：只有博主看得到（别人没有通知可看，不显示这个铃铛） -->
        <div v-if="me && me.is_owner" class="notify-wrap">
          <button class="notify-btn" title="新的评论 / 留言" @click="openNotify">
            <span class="notify-emoji">💌</span>
            <!-- 红点：未读数（超过 99 就显示 99+，不然会把铃铛撑爆） -->
            <span v-if="unread > 0" class="notify-badge">{{ unread > 99 ? '99+' : unread }}</span>
          </button>

          <!-- 点面板外面任意处 = 关闭 -->
          <div v-if="notifyOpen" class="menu-overlay" @click="notifyOpen = false"></div>

          <!-- 通知面板：最近 20 条 -->
          <div v-if="notifyOpen" class="notify-panel">
            <p v-if="notifyItems.length === 0" class="notify-empty">还没有新消息～</p>
            <!-- 点一条通知：跳到对应的说说（留言没有 talk_id，点了只关面板） -->
            <div
              v-for="(n, i) in notifyItems"
              :key="i"
              class="notify-item"
              :class="{ 'can-go': n.talk_id }"
              @click="goNotify(n)"
            >
              <span class="notify-icon">{{ n.kind === 'comment' ? '💬' : '🌸' }}</span>
              <div class="notify-body">
                <p class="notify-line">
                  <b>{{ n.nickname }}</b>
                  {{ n.kind === 'comment' ? ' 评论了你的说说' : ' 给你留了言' }}
                </p>
                <p class="notify-text">{{ n.content }}</p>
                <p class="notify-time">{{ n.time }}</p>
              </div>
            </div>

            <!-- 还有更早的通知 → 加载更多 -->
            <button
              v-if="notifyHasMore"
              class="notify-more"
              :disabled="notifyLoadingMore"
              @click.stop="loadMoreNotify"
            >
              {{ notifyLoadingMore ? '加载中…' : '加载更多' }}
            </button>
          </div>
        </div>

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
