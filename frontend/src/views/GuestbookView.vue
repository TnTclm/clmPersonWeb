<script setup>
// ============================================================
// GuestbookView —— 留言板（樱花弹幕版 v2）
// 画面策略：一张动漫级背景插画（public/images/sakura-bg.jpg）打底，
//          代码只负责"动"的部分：花瓣雨、弹幕、纸飞机
//          图没放进来之前，用手绘 SVG 场景自动顶替（bgReady 机制）
// 弹幕规则：每条只飞一次（飞走就算了），鼠标悬停暂停、可删除
// ============================================================

import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { me } from '@/stores/me'
import { ElMessage, ElMessageBox } from 'element-plus'
import defaultAvatar from '@/assets/images/avatar-default.svg'

const router = useRouter()

// ---------- 背景：视频/图片/手绘 三层接力 ----------
// 文件放 public/images/ 下即生效：sakura-bg.mp4（最佳）→ sakura-bg.jpg → 手绘SVG
const videoFailed = ref(false) // 没有视频 → 退级用图片
const imgFailed = ref(false)   // 没有图片 → 退级用手绘SVG

// 背景视频控制：默认【不自动播放 + 静音】，右下角两个小按钮管播放和声音
const bgVideo = ref(null)      // 指向 <video> 元素本身
const bgPlaying = ref(false)   // 当前是否在播放
const bgMuted = ref(true)      // 当前是否静音（默认静音）

function toggleBgPlay() {
  const v = bgVideo.value
  if (!v) return
  if (v.paused) {
    v.play()                   // 用户点了播放才加载/播放，不浪费流量
    bgPlaying.value = true
  } else {
    v.pause()
    bgPlaying.value = false
  }
}

function toggleBgSound() {
  const v = bgVideo.value
  if (!v) return
  v.muted = !v.muted           // 直接改 video 属性（:muted 绑定在部分浏览器不生效）
  bgMuted.value = v.muted
}

// ---------- 花瓣雨（叠在插画上，比插画里的花瓣更"近"） ----------
const petals = ref([])

function makePetals() {
  return Array.from({ length: 22 }, (_, i) => ({
    id: i,
    left: Math.random() * 100,
    size: 8 + Math.random() * 10,
    fallDur: 8 + Math.random() * 9,
    swayDur: 2.6 + Math.random() * 2.4,
    delay: -Math.random() * 18,          // 负延迟：打开页面就"正在飘"
    opacity: 0.4 + Math.random() * 0.45, // 有插画了，花瓣调淡一点不抢戏
    color: Math.random() < 0.5 ? '#ffc2dc' : '#ffd9e8',
  }))
}

// ---------- 弹幕 ----------
const messages = ref([])
const TRACKS = 8
const replayRound = ref(0) // 点"再放一遍"就 +1；:key 变化强制重建 DOM → 动画重播

// 轨道从 22vh 到 78vh：避开 header，也不贴地
function decorate(list) {
  return list.map((m, i) => {
    const dur = 18 + ((i * 7) % 13)              // 18~30 秒飞完全程
    return {
      ...m,
      dur,
      top: 22 + (i % TRACKS) * 8,                // 8条轨道：22/30/38...78 vh
      delay: -((i * 3.7) % dur),                 // 负延迟：进场时就已经飞在天上
    }
  })
}

async function loadMessages() {
  try {
    const res = await api.get('/api/guestbook')
    messages.value = decorate(res.data)
  } catch (e) {
    ElMessage.error(e.message)
  }
}

// ---------- 发布 ----------
const content = ref('')
const publishing = ref(false)

async function publish() {
  if (!me.value) {
    ElMessage.warning('登录后才能留言哦')
    router.push('/login')
    return
  }
  const text = content.value.trim()
  if (!text) {
    ElMessage.warning('一定要留下点什么～')
    return
  }
  publishing.value = true
  try {
    const res = await api.post('/api/guestbook', { content: text })
    // 新留言 delay=0：立刻从右侧起飞（只飞一次）
    messages.value = [
      ...messages.value,
      {
        ...res.data,
        avatar: me.value.avatar || '',
        dur: 18,
        top: 22 + (messages.value.length % TRACKS) * 8,
        delay: 0,
      },
    ]
    content.value = ''
    ElMessage.success('已乘着花瓣飞走啦')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    publishing.value = false
  }
}

// ---------- 删除权限：本人删自己的，博主删任何人的 ----------
// 用普通函数不用 computed：me 是异步加载的，
// 函数在每次渲染时都会重新跑一遍，fetchMe() 一回来，
// 博主那边的 × 按钮就自己"长出来"，不用手动刷新
function canDelete(m) {
  // 满足其一就放行：这条是我的 || 我是博主
  return me.value && (me.value.user_id === m.user_id || me.value.is_owner)
}

// ---------- 删除留言：弹窗文案按"删的是谁的"分开写 ----------
async function removeMessage(m) {
  // 是我的留言 → 平常文案；是别人的（只可能是博主在执法）→ 把对方昵称点出来
  // 文案里带上昵称，博主扫一眼就知道删的是哪条，避免手滑误删
  const confirmText = me.value && me.value.user_id === m.user_id
    ? '确定删除这条留言吗？'
    : '确定删除「' + m.nickname + '」的留言吗？'
  try {
    await ElMessageBox.confirm(confirmText, '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.del('/api/guestbook/' + m.id)
    ElMessage.success('删除成功')
    loadMessages()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

onMounted(() => {
  loadMessages()
  petals.value = makePetals()
})
</script>

<style scoped src="@/views/css/GuestbookView.css"></style>

<template>
  <div class="sakura-page">
    <!-- ① 背景（三层接力）：有视频用视频 → 有图片用图片 → 都没有才用手绘替补 -->
    <video
      v-if="!videoFailed"
      ref="bgVideo"
      class="sakura-bg"
      muted
      loop
      playsinline
      preload="auto"
      @error="videoFailed = true"
    >
      <source :src="'/images/sakura-bg.mp4'" type="video/mp4" />
    </video>
    <img
      v-else-if="!imgFailed"
      class="sakura-bg"
      :src="'/images/sakura-bg.jpg'"
      alt=""
      @error="imgFailed = true"
    />
    <!-- 柔光遮罩：给画面蒙一层薄雾，弹幕文字更清晰 -->
    <div class="sakura-veil" aria-hidden="true"></div>

    <!-- 视频和图片都没有时的替补场景：手绘樱花树 + 少女 -->
    <div v-if="videoFailed && imgFailed" class="scene-wrap" aria-hidden="true">
      <svg class="scene-svg" viewBox="0 0 520 720" xmlns="http://www.w3.org/2000/svg">
        <g class="tree-sway">
          <path d="M118 720 C122 640 110 560 132 480 C150 415 190 380 225 358 L237 374 C207 402 175 442 165 492 C155 552 160 640 158 720 Z" fill="#7a5546"/>
          <path d="M180 430 C220 400 260 380 305 368" stroke="#7a5546" stroke-width="9" fill="none" stroke-linecap="round"/>
          <path d="M212 392 C252 352 302 330 345 320" stroke="#7a5546" stroke-width="6" fill="none" stroke-linecap="round"/>
          <path d="M160 470 C130 442 110 420 92 398" stroke="#7a5546" stroke-width="7" fill="none" stroke-linecap="round"/>
          <path d="M150 440 C150 400 158 370 172 350" stroke="#7a5546" stroke-width="5" fill="none" stroke-linecap="round"/>
          <circle cx="140" cy="220" r="60" fill="#ffd6e7"/>
          <circle cx="230" cy="150" r="72" fill="#ffc2dc"/>
          <circle cx="320" cy="200" r="66" fill="#ffd6e7"/>
          <circle cx="240" cy="280" r="58" fill="#ffadd1"/>
          <circle cx="150" cy="300" r="50" fill="#ffc2dc"/>
          <circle cx="330" cy="300" r="48" fill="#ffadd1"/>
          <circle cx="400" cy="240" r="44" fill="#ffc2dc"/>
          <circle cx="90"  cy="260" r="40" fill="#ffadd1"/>
          <circle cx="190" cy="190" r="36" fill="#f79ac4"/>
          <circle cx="280" cy="120" r="34" fill="#ffadd1"/>
          <circle cx="360" cy="160" r="30" fill="#f79ac4"/>
          <circle cx="420" cy="290" r="26" fill="#ffadd1"/>
          <circle cx="70"  cy="210" r="28" fill="#ffc2dc"/>
          <circle cx="300" cy="250" r="40" fill="#ffd6e7"/>
          <circle cx="200" cy="260" r="30" fill="#ffc2dc"/>
          <circle cx="120" cy="150" r="30" fill="#ffadd1"/>
          <circle cx="380" cy="120" r="26" fill="#ffd6e7"/>
          <circle cx="260" cy="220" r="44" fill="#ffc2dc"/>
        </g>
        <ellipse cx="300" cy="702" rx="60" ry="8" fill="#e8a9c8" opacity="0.3"/>
        <g>
          <path d="M277 496 Q293 486 309 496 Q316 502 315 514 Q318 540 326 560 Q330 580 322 602 Q317 585 311 574 Q313 552 309 536 Q300 528 284 526 Q276 515 277 496 Z" fill="#4a3833"/>
          <path d="M315 545 L318 590" stroke="#ffdcc8" stroke-width="7" stroke-linecap="round"/>
          <path d="M289 642 L287 692" stroke="#ffdcc8" stroke-width="7" stroke-linecap="round"/>
          <path d="M303 642 L306 692" stroke="#ffdcc8" stroke-width="7" stroke-linecap="round"/>
          <ellipse cx="286" cy="698" rx="10" ry="5" fill="#5c4a52"/>
          <ellipse cx="307" cy="698" rx="10" ry="5" fill="#5c4a52"/>
          <path d="M278 592 Q299 584 320 592 L330 645 Q299 653 268 645 Z" fill="#3d4a6b"/>
          <path d="M282 540 Q299 532 316 540 L318 594 Q299 600 280 594 Z" fill="#ffffff"/>
          <path d="M285 546 L264 521 L272 512 L291 538 Z" fill="#ffffff"/>
          <path d="M264 521 L248 503" stroke="#ffdcc8" stroke-width="7" stroke-linecap="round"/>
          <circle cx="245" cy="500" r="5.5" fill="#ffdcc8"/>
          <circle cx="293" cy="508" r="16" fill="#ffe3d0"/>
          <path d="M279 500 Q293 490 308 500 Q306 509 300 506 Q293 502 286 509 Q281 507 279 500 Z" fill="#52403a"/>
          <path d="M294 489 Q299 481 305 485" stroke="#4a3833" stroke-width="3" fill="none" stroke-linecap="round"/>
          <path d="M281 509 Q285 513 289 509" stroke="#5a4642" stroke-width="2" fill="none" stroke-linecap="round"/>
          <ellipse cx="284" cy="516" rx="4" ry="2" fill="#ffb9b9" opacity="0.6"/>
        </g>
      </svg>
    </div>

    <!-- ③ 花瓣雨 -->
    <div class="petal-layer" aria-hidden="true">
      <div
        v-for="p in petals"
        :key="p.id"
        class="petal-fall"
        :style="{ left: p.left + 'vw', animationDuration: p.fallDur + 's', animationDelay: p.delay + 's' }"
      >
        <div
          class="petal-sway"
          :style="{
            width: p.size + 'px',
            height: p.size * 0.72 + 'px',
            background: p.color,
            opacity: p.opacity,
            animationDuration: p.swayDur + 's',
          }"
        ></div>
      </div>
    </div>

    <!-- ④ 弹幕层：每条只飞一次，飞走就算了 -->
    <div class="dm-layer">
      <div
        v-for="m in messages"
        :key="replayRound + '-' + m.id"
        class="dm-item"
        :style="{ top: m.top + 'vh', animationDuration: m.dur + 's', animationDelay: m.delay + 's' }"
      >
        <img class="dm-avatar" :src="m.avatar || defaultAvatar" alt="" />
        <span class="dm-name">{{ m.nickname }}</span>
        <span class="dm-text">{{ m.content }}</span>
        <button v-if="canDelete(m)" class="dm-del" title="删除" @click.stop="removeMessage(m)">×</button>
      </div>
    </div>

    <!-- 背景视频控制：右下角两个小圆钮，半透明不抢戏 -->
    <div v-if="!videoFailed" class="bg-controls">
      <button class="bg-btn" :title="bgPlaying ? '暂停背景视频' : '播放背景视频'" @click="toggleBgPlay">
        {{ bgPlaying ? '⏸' : '▶' }}
      </button>
      <button class="bg-btn" :title="bgMuted ? '开启声音' : '静音'" @click="toggleBgSound">
        {{ bgMuted ? '🔇' : '🔊' }}
      </button>
    </div>

    <!-- ⑤ 标题 + 输入胶囊 -->
    <div class="gb-center">
      <h1 class="gb-title">留　言</h1>
      <p class="gb-sub">樱花落下的速度，是秒速五厘米。想说的话，请乘着花瓣来。</p>

      <div class="gb-capsule">
        <input
          v-model="content"
          class="gb-input"
          maxlength="50"
          :placeholder="me ? '一定要留下点什么～' : '登录之后，才能乘着花瓣留言哦'"
          @keyup.enter="publish"
        />
        <button class="gb-send" :disabled="publishing" title="发送" @click="publish">
          <svg viewBox="0 0 24 24" width="20" height="20">
            <path d="M2 21 L23 12 L2 3 L2 10 L17 12 L2 14 Z" fill="currentColor" />
          </svg>
        </button>
      </div>

      <div class="gb-tools">
        <p v-if="me" class="gb-count">{{ content.length }}/50</p>
        <p v-else class="gb-count"><RouterLink class="gb-login" to="/login">去登录</RouterLink></p>
        <!-- 弹幕飞走就没了；想重温点这里重新放一轮 -->
        <button class="gb-replay" @click="replayRound++">再放一遍弹幕 🌸</button>
      </div>
    </div>
  </div>
</template>
