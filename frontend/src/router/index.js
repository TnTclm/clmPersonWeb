import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // 主页=纯展示门面（轮播图+入口卡片）；说说流住在自己的标签页 /talks
    { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
    { path: '/talks', name: 'talks', component: () => import('@/views/TalksView.vue') },
    { path: '/talks/:id', name: 'talk-detail', component: () => import('@/views/TalkDetailView.vue') },
    { path: '/login', name: 'login', component: () => import('@/views/LoginView.vue') },
    { path: '/register', name: 'register', component: () => import('@/views/RegisterView.vue') },
    // meta.requiresAuth = 给这条路由贴个"需要登录"的标签
    // 守卫（下面beforeEach）看到这个标签就查令牌
    { path: '/admin', name: 'admin', component: () => import('@/views/AdminView.vue'), meta: { requiresAuth: true } },
    { path: '/user/:id', name: 'user', component: () => import('@/views/UserView.vue') },
    { path: '/guestbook', name: 'guestbook', component: () => import('@/views/GuestbookView.vue') },
    { path: '/album', name: 'album', component: () => import('@/views/AlbumView.vue') },
    { path: '/heart', name: 'heart', component: () => import('@/views/HeartView.vue') },
  ],
})

// 全局前置守卫：每次跳转页面前都会经过这里
// to = 要去哪；meta.requiresAuth = 刚才贴的标签
router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  // 要去的页面需要登录，但兜里没卡 → 踢去登录页
  if (to.meta.requiresAuth && !token) {
    return '/login'
  }
  // return undefined（啥都不返回）= 放行
})

export default router
