<script setup>
// ============================================================
// HomeView —— 主页 · 图片墙（极简画廊）
// 结构：整页容器(米白纸墙 + 顶部柔光) → 图片列表(一张一行)
// 这一页不打任何动态特效：主角就是照片本身，安静地挂满一墙
// 清单来自后端 /api/pictures（后端已排好序），图片本体来自 /pictures/xxx.jpg
// ============================================================

import { ref, onMounted } from 'vue'
import { api } from '@/api'
import AvatarBurst from '@/components/AvatarBurst.vue' // 点了会迸发小图案的头像（可复用组件）
import avatarImg from '@/assets/images/clmqq.jpg'        // 信息卡上的头像图片

// 后端给的图片清单，形如 [{ name: '1.jpg', url: '/pictures/1.jpg' }, ...]
const pictures = ref([])
// 是否还在加载（控制"加载中…"提示的显示）
const loading = ref(true)
// 是否加载失败（最常见原因：后端没启动）
const failed = ref(false)

// onMounted：组件挂载完成后跑一次 —— 页面一打开就去拉清单
onMounted(async () => {
  try {
    const res = await api.get('/api/pictures')
    pictures.value = res.data
  } catch (err) {
    failed.value = true
  } finally {
    // 不管成功还是失败，都别再显示"加载中"了
    loading.value = false
  }
})

// ---------- 邮箱气泡：点"邮箱"弹出一小块显示地址，方便看和复制 ----------
const mailOpen = ref(false) // 气泡是否展开
const copyTip = ref('复制') // 复制按钮上的文字（成功/失败会短暂变化，给个反馈）

async function copyMail() {
  try {
    // navigator.clipboard 是浏览器自带的剪贴板 API
    // （只在 https 或 localhost 下可用，本地开发正好满足）
    await navigator.clipboard.writeText('2169616905@qq.com')
    copyTip.value = '已复制 ✓'
  } catch (err) {
    // 浏览器不给用剪贴板时（少见，比如将来部署在 http 上）：
    // 退一步 —— 把地址整段选中，用户按 Ctrl+C 也能拿走，不让人干瞪眼
    const el = document.querySelector('.mail-addr')
    if (el) {
      const range = document.createRange()
      range.selectNodeContents(el)   // 选中范围 = 地址元素的全部内容
      const sel = window.getSelection()
      sel.removeAllRanges()          // 先清掉可能存在的旧选区
      sel.addRange(range)            // 再放上新选区
    }
    copyTip.value = '已选中，按 Ctrl+C'
  }
  // 1.8 秒后按钮文字变回"复制"
  setTimeout(() => {
    copyTip.value = '复制'
  }, 1800)
}
</script>

<style scoped src="@/views/css/HomeView.css"></style>

<template>
  <!-- 整页容器：一面安静的米白展墙 -->
  <div class="home">
    <!-- 个人信息卡：贴在屏幕左侧（屏幕不够宽时 CSS 会自动隐藏它） -->
    <aside class="profile">
      <!-- 头像：点击迸发小图案
           （特效逻辑住在 AvatarBurst 组件里，和左上角头部 logo 共用同一份代码） -->
      <AvatarBurst :src="avatarImg" :size="64" hint="点我试试 ✨" />
      <p class="profile-name">clm</p>
      <p class="profile-desc">即兴的誓，烂尾的诗</p>

      <div class="profile-links">
        <!-- LeetCode：图标是"代码尖括号"（线性描边，风格和另外两个统一） -->
        <a
          class="profile-link"
          href="https://leetcode.cn/u/tnt_clm/"
          target="_blank"
          rel="noopener"
        >
          <svg
            viewBox="0 0 24 24"
            width="17"
            height="17"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path d="M9 6 3 12l6 6" />
            <path d="M15 6l6 6-6 6" />
          </svg>
          <span>LeetCode</span>
        </a>

        <!-- GitHub：target="_blank" 新窗口打开；rel="noopener" 是安全习惯——
             防止新页面通过 window.opener 反向控制你的原页面 -->
        <a
          class="profile-link"
          href="https://github.com/TnTclm"
          target="_blank"
          rel="noopener"
        >
          <!-- 图标用内联 SVG 画：不用下载图片、不会糊、颜色跟着文字走（currentColor） -->
          <svg viewBox="0 0 16 16" width="17" height="17" fill="currentColor" aria-hidden="true">
            <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8z" />
          </svg>
          <span>GitHub</span>
        </a>

        <!-- 邮箱：点击不是"发邮件"，而是弹出一个小框显示地址（方便看和复制） -->
        <div class="mail-wrap">
          <button class="profile-link" @click="mailOpen = !mailOpen">
            <svg
              viewBox="0 0 24 24"
              width="17"
              height="17"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <rect x="3" y="5" width="18" height="14" rx="3" />
              <path d="m4 7.5 8 5.5 8-5.5" />
            </svg>
            <span>邮箱</span>
          </button>

          <!-- 全屏透明遮罩：点页面上别的地方 = 关闭气泡（和头部用户菜单同一招） -->
          <div v-if="mailOpen" class="mail-overlay" @click="mailOpen = false"></div>

          <!-- 气泡：地址 + 一键复制 -->
          <div v-if="mailOpen" class="mail-pop">
            <span class="mail-addr">2169616905@qq.com</span>
            <button class="mail-copy" @click="copyMail">{{ copyTip }}</button>
          </div>
        </div>
      </div>
    </aside>

    <!-- 内容区：定宽居中（1080，和头部对齐） -->
    <div class="gallery">
      <!-- ① 加载中 -->
      <p v-if="loading" class="gallery-tip">正在加载图片…</p>

      <!-- ② 加载失败（先检查后端有没有启动） -->
      <p v-else-if="failed" class="gallery-tip">图片加载失败了，看看后端是否已启动～</p>

      <!-- ③ 文件夹是空的 -->
      <p v-else-if="pictures.length === 0" class="gallery-tip">图片文件夹里还没有图片哦</p>

      <!-- ④ 正常：一张图一行，宽度统一、高度随原图比例 -->
      <div v-else class="gallery-list">
        <figure v-for="p in pictures" :key="p.name" class="shot">
          <!-- loading="lazy"：浏览器只加载"快滚动到"的图片，图再多也不卡 -->
          <img :src="p.url" :alt="p.name" loading="lazy" />
        </figure>
      </div>
    </div>
  </div>
</template>
