// ============================================================
// me.js —— 全站共享的"当前登录人"状态
// 为什么要有它？
//   SiteHeader 在 App.vue 里，整站只挂载一次，onMounted 只跑一次。
//   登录是 SPA 内部跳转（router.push），header 不会重新创建，
//   也就不会重新问后端"我是谁"——所以登录后 header 还是"登录|注册"。
//   把 me 抽成共享状态：谁登录了，谁调一下 fetchMe()，
//   所有用到 me 的地方（header、以后的页面）瞬间同步。
// ============================================================

import { ref } from 'vue'
import { api } from '@/api'

// ---------- 防热更新"分身"保险 ----------
// Vite 热更新偶尔会把同一个模块执行两次，产生两份独立的 ref
//（症状：header 显示已登录，页面里却提示没登录——两份状态各改各的）。
// 解法：把 ref 挂到 window 上当单例，无论模块被执行几次，
// 所有 import 拿到的永远是同一个 ref。
const store = window.__blogMeStore || (window.__blogMeStore = {})
export const me = store.me || (store.me = ref(null))

// 问后端"我是谁"，结果存进共享的 me
export async function fetchMe() {
  // 兜里没令牌 = 游客，直接置空（登出也靠这行清状态）
  if (!localStorage.getItem('token')) {
    me.value = null
    return
  }
  try {
    const res = await api.get('/api/me')
    me.value = res.data // { user_id, username, nickname }
  } catch (e) {
    // 令牌失效(401) api.js 已统一处理跳转，这里不用管
  }
}
