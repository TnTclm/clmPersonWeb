<script setup>
// ============================================================
// TalkDetailView —— 单条说说的详情页
// 从说说流点卡片进来（/talks/:id），评论区常开，看完点"返回"
// 权限：点赞/评论 = 登录用户；删说说 = 自己或博主；删评论 = 自己或博主
// 样式直接复用 TalksView.css（卡片/九宫格/评论区全是一套）
// ============================================================

import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { me } from '@/stores/me'
import { ElMessage, ElMessageBox } from 'element-plus'
import defaultAvatar from '@/assets/images/avatar-default.svg'

const route = useRoute()
const router = useRouter()

// 点头像/昵称去看 TA 的个人空间
function goUser(id) {
  router.push('/user/' + id)
}

const talk = ref(null)      // 这条说说（含 likes / comment_count）
const comments = ref([])    // 评论列表（详情页常开，不用缓存 Map）
const draft = ref('')       // 评论输入框草稿
const likedSet = ref(new Set()) // 我赞过的说说ID（恢复红心）

// 库里 images 是 JSON 字符串，转回数组；脏数据兜底成空数组
function parseImages(t) {
  try {
    const arr = JSON.parse(t.images || '[]')
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

// 按图片数量决定网格布局（和说说流同一套规则）
function gridClass(n) {
  if (n === 1) return 'single'
  if (n === 2 || n === 4) return 'two'
  return 'three'
}

async function loadAll() {
  try {
    const res = await api.get('/api/talks/' + route.params.id)
    talk.value = { ...res.data, _imgs: parseImages(res.data) }
  } catch (e) {
    // 说说没了（被删/ID错）直接送回说说流，别让用户停在空页面
    ElMessage.error(e.message)
    router.replace('/talks')
    return
  }
  try {
    const res = await api.get('/api/talks/' + route.params.id + '/comments')
    comments.value = res.data
  } catch (e) {
    ElMessage.error(e.message)
  }
}

// ---------- 恢复小红心（登录才有） ----------
async function loadMyLikes() {
  if (!me.value) return
  try {
    const res = await api.get('/api/me/liked-talks')
    likedSet.value = new Set(res.data)
  } catch {
    // 拿不到就不亮心，不影响页面主体
  }
}

// 和说说流一个道理：硬刷新时 me 还没加载完，
// 交给 watch 盯着，me 一落地就自动补拉（immediate 保证 SPA 跳进来时立刻生效）
watch(me, (val) => {
  if (val) loadMyLikes()
}, { immediate: true })

// 返回：有来路就回退（保留说说流的滚动位置），直接打开的本页就回说说流
function goBack() {
  if (window.history.length > 1) router.back()
  else router.replace('/talks')
}

// ---------- 权限 ----------
const isMine = () => me.value && talk.value && me.value.user_id === talk.value.user_id
const isOwner = () => me.value && me.value.is_owner
const canDelete = (cm) => me.value && (me.value.user_id === cm.user_id || me.value.is_owner)

// ---------- 点赞（和说说流同一套接口） ----------
async function toggleLike() {
  if (!me.value) {
    ElMessage.warning('登录后才能点赞哦')
    return
  }
  try {
    const res = await api.post('/api/talks/' + talk.value.id + '/like')
    talk.value.likes = res.data.likes
    if (res.data.liked) likedSet.value.add(talk.value.id)
    else likedSet.value.delete(talk.value.id)
  } catch (e) {
    ElMessage.error(e.message)
  }
}

// ---------- 评论 ----------
async function sendComment() {
  if (!me.value) {
    ElMessage.warning('登录后才能评论哦')
    return
  }
  const text = draft.value.trim()
  if (!text) {
    ElMessage.warning('写点啥再发吧')
    return
  }
  try {
    const res = await api.post('/api/talks/' + talk.value.id + '/comments', { content: text })
    comments.value.push(res.data)
    draft.value = ''
    talk.value.comment_count++
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function removeComment(cm) {
  try {
    await ElMessageBox.confirm('确定删除这条评论吗？', '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.del('/api/talks/' + talk.value.id + '/comments/' + cm.id)
    comments.value = comments.value.filter(x => x.id !== cm.id)
    talk.value.comment_count--
    ElMessage.success('删除成功')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

// ---------- 删说说（删完回说说流） ----------
async function removeTalk() {
  try {
    await ElMessageBox.confirm('确定删除这条说说吗？配图文件会一并删除', '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.del('/api/talks/' + talk.value.id)
    ElMessage.success('删除成功')
    router.replace('/talks')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

onMounted(loadAll)
</script>

<style scoped src="@/views/css/TalksView.css"></style>

<style scoped>
/* 详情页独有：顶部返回条 */
.detail-bar {
  margin-bottom: 12px;
}

.btn-back {
  padding: 8px 20px;
  background: #fff;
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 999px;
  font-size: 13px;
  color: #666;
  cursor: pointer;
  transition: border-color 0.2s, color 0.2s;
}

.btn-back:hover {
  border-color: #2563eb;
  color: #2563eb;
}
</style>

<template>
  <div class="talks-page">
    <!-- 返回条 -->
    <div class="detail-bar">
      <button class="btn-back" @click="goBack">← 返回</button>
    </div>

    <!-- 加载中 -->
    <p v-if="!talk" class="empty-tip">加载中……</p>

    <!-- 单条说说（结构/样式和说说流的卡片完全一致） -->
    <div v-else class="talk-card detail-card">
      <img class="talk-avatar" :src="talk.avatar || defaultAvatar" alt="头像" @click="goUser(talk.user_id)" />
      <div class="talk-main">
        <div class="talk-head">
          <span class="talk-name" @click="goUser(talk.user_id)">{{ talk.nickname }}</span>
          <span class="talk-time">{{ (talk.created_at || '').slice(0, 16).replace('T', ' ') }}</span>
        </div>
        <p v-if="talk.content" class="talk-content">{{ talk.content }}</p>

        <!-- 九宫格：点击放大 -->
        <div v-if="talk._imgs.length" class="pic-grid" :class="gridClass(talk._imgs.length)">
          <el-image
            v-for="(img, idx) in talk._imgs"
            :key="img"
            class="pic-thumb"
            :src="img"
            :preview-src-list="talk._imgs"
            :initial-index="idx"
            fit="cover"
            :hide-on-click-modal="true"
          />
        </div>

        <!-- 操作行：评论区常开，💬只是个计数牌 -->
        <div class="talk-foot">
          <span class="foot-actions">
            <button class="tk-like" :class="{ on: likedSet.has(talk.id) }" @click="toggleLike">
              {{ likedSet.has(talk.id) ? '❤️' : '🤍' }} {{ talk.likes }}
            </button>
            <span class="tk-cbtn">💬 {{ talk.comment_count }} 条评论</span>
          </span>
          <a
            v-if="isMine() || isOwner()"
            href="#"
            class="talk-delete"
            @click.prevent="removeTalk"
          >删除</a>
        </div>

        <!-- 评论区（常开） -->
        <div class="tk-panel">
          <div v-for="cm in comments" :key="cm.id" class="tk-comment">
            <img class="tk-cavatar" :src="cm.avatar || defaultAvatar" alt="头像" @click="goUser(cm.user_id)" />
            <div class="tk-cmain">
              <span class="tk-cname" @click="goUser(cm.user_id)">{{ cm.nickname }}</span>
              <p class="tk-ctext">{{ cm.content }}</p>
            </div>
            <button v-if="canDelete(cm)" class="tk-cdel" @click="removeComment(cm)">删除</button>
          </div>
          <p v-if="comments.length === 0" class="tk-cempty">还没有评论，来抢沙发～</p>

          <div v-if="me" class="tk-cinput">
            <input
              v-model="draft"
              class="tk-input"
              maxlength="500"
              placeholder="友善评论，最多500字"
              @keyup.enter="sendComment"
            />
            <button class="tk-send" @click="sendComment">发送</button>
          </div>
          <p v-else class="tk-cempty">
            <RouterLink to="/login" class="tk-login">登录</RouterLink>后才能评论哦
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
