<script setup>
// ============================================================
// TalksView —— 说说流（全站唯一内容形态，文章已并入说说）
// 发布：仅博主（其他人只能看 + 点赞 + 评论）
// 博主：发布 + "执法权"（能删任何人的说说）
// ============================================================

import { ref, reactive, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { me } from '@/stores/me'
import { ElMessage, ElMessageBox } from 'element-plus'
import defaultAvatar from '@/assets/images/avatar-default.svg'

const router = useRouter()

// ---------- 动态列表 ----------
const talks = ref([])

// 库里 images 是 JSON 字符串，转回数组；脏数据兜底成空数组
function parseImages(talk) {
  try {
    const arr = JSON.parse(talk.images || '[]')
    return Array.isArray(arr) ? arr : []
  } catch {
    return []
  }
}

// 按图片数量决定网格布局：1 张大图、2/4 张两列、其余三列，铺满内容区
function gridClass(n) {
  if (n === 1) return 'single'
  if (n === 2 || n === 4) return 'two'
  return 'three'
}

// 点卡片进详情页（卡片上的按钮/图片/评论区都 stop 了，只有空白处能点穿）
function goDetail(id) {
  router.push('/talks/' + id)
}

// 点头像/昵称去看 TA 的个人空间
function goUser(id) {
  router.push('/user/' + id)
}

// ---------- 分页状态 ----------
const page = ref(1)            // 当前第几页
const hasMore = ref(false)     // 后端说"后面还有没有"
const loadingMore = ref(false) // 正在加载下一页（防止连点）

// reset=true：从第 1 页重新拉（首次进页面 / 发完新说说后）
// reset=false：只把"下一页"追加到列表后面
async function loadTalks(reset = true) {
  try {
    if (reset) page.value = 1
    const res = await api.get('/api/talks?page=' + page.value)
    // ...t 展开原字段，再补一个 _imgs 数组（下划线 = 前端自己加的，不是后端给的）
    const list = res.data.map(t => ({ ...t, _imgs: parseImages(t) }))
    talks.value = reset ? list : talks.value.concat(list)
    hasMore.value = res.has_more
  } catch (e) {
    ElMessage.error(e.message)
  }
}

// 点"加载更多"：页码 +1，拿到新一页追加到后面
async function loadMoreTalks() {
  if (loadingMore.value || !hasMore.value) return // 正在加载 / 没有更多，直接不理
  loadingMore.value = true
  page.value += 1
  await loadTalks(false)
  loadingMore.value = false
}

// ---------- 发布 ----------
const content = ref('')
const pics = ref([])         // 已上传成功的图片地址（最多9个）
const uploading = ref(false) // 有图正在上传
const publishing = ref(false)
const fileInput = ref(null)  // 指向隐藏的多选文件框

function pickPics() {
  if (pics.value.length >= 9) {
    ElMessage.warning('最多9张图哦')
    return
  }
  fileInput.value.click() // 让隐藏的文件框弹出来
}

// 选完文件（支持多选）：一张一张立刻上传，拿到地址就进预览
async function handleFilesChange(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = '' // 清空选择，同一个文件下次还能再选
  if (!files.length) return
  const remain = 9 - pics.value.length
  const toUpload = files.slice(0, remain) // 超出9张的部分直接忽略
  if (toUpload.length < files.length) {
    ElMessage.warning('最多9张，多出来的没传')
  }
  uploading.value = true
  try {
    for (const f of toUpload) {
      const fd = new FormData()
      fd.append('file', f) // 字段名必须叫 file，和后端 FormFile("file") 对应
      const res = await api.upload('/api/images', fd)
      pics.value.push(res.data.url)
    }
  } catch (err) {
    ElMessage.error(err.message)
  } finally {
    uploading.value = false
  }
}

// 从预览里移除某张图（只是不发了，服务器文件不删）
function removePic(idx) {
  pics.value.splice(idx, 1)
}

async function publish() {
  if (!content.value.trim() && pics.value.length === 0) {
    ElMessage.warning('总得写点什么或传张图吧')
    return
  }
  publishing.value = true
  try {
    await api.post('/api/talks', {
      content: content.value.trim(),
      images: pics.value, // 数组直接给，axios...哦不，我们的 api 会自动 JSON.stringify
    })
    ElMessage.success('发表成功')
    content.value = ''
    pics.value = []
    loadTalks() // 重拉列表，新发的在最上
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    publishing.value = false
  }
}

// ---------- 删除（自己的随便删；博主是"执法者"，谁的都能删） ----------
function isMine(t) {
  return me.value && me.value.user_id === t.user_id
}

const isOwner = () => me.value && me.value.is_owner

// ---------- 评论删除：自己的评论自己删，博主谁的都能删 ----------
const canDelete = (cm) => me.value && (me.value.user_id === cm.user_id || me.value.is_owner)

async function removeComment(t, cm) {
  try {
    await ElMessageBox.confirm('确定删除这条评论吗？', '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return // 点了取消
  }
  try {
    await api.del('/api/talks/' + t.id + '/comments/' + cm.id)
    // 本地移除，不用重新拉列表
    commentsMap[t.id] = (commentsMap[t.id] || []).filter(x => x.id !== cm.id)
    t.comment_count-- // 计数器同步 -1
    ElMessage.success('删除成功')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

async function removeTalk(t) {
  try {
    await ElMessageBox.confirm('确定删除这条说说吗？配图文件会一并删除', '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return // 点了取消
  }
  try {
    await api.del('/api/talks/' + t.id)
    ElMessage.success('删除成功')
    loadTalks()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

// ---------- 点赞（Redis Set，切换式） ----------
const likedSet = ref(new Set()) // 我赞过的说说ID集合（登录才有数据）

// 拉一次"我赞过哪些"，恢复小红心
async function loadMyLikes() {
  if (!me.value) return
  try {
    const res = await api.get('/api/me/liked-talks')
    likedSet.value = new Set(res.data)
  } catch {
    // 拿不到就不亮心，不影响页面主体
  }
}

// watch：盯着 me，它一有值就补拉一次点赞记录
// 为什么不能只在 onMounted 里调？硬刷新时那一刻 me 还没加载完，
// 会被上面那行 return 挡掉，小红心就永远不亮了。
// watch 会一直盯着，me 从 null 变成用户对象的那一刻自动再跑一次
watch(me, (val) => {
  if (val) loadMyLikes() // 有值（登录了）才拉；登出变回 null 就不管
}, { immediate: true })  // immediate：建立侦听时立刻按当前值跑一次
                         //（SPA 内部跳转进来时 me 已有值，这一次就够了）

async function toggleLike(t) {
  if (!me.value) {
    ElMessage.warning('登录后才能点赞哦')
    return
  }
  try {
    const res = await api.post('/api/talks/' + t.id + '/like')
    t.likes = res.data.likes // 后端算好最新数直接用，不用自己±1
    if (res.data.liked) likedSet.value.add(t.id)
    else likedSet.value.delete(t.id)
  } catch (e) {
    ElMessage.error(e.message)
  }
}

// ---------- 评论（每条说说一个内嵌评论区，一次只展开一个） ----------
const openId = ref(null)         // 正在展开评论区的说说ID（null=全收起）
const commentsMap = reactive({}) // { [说说ID]: [评论...] }，展开过就缓存住
const drafts = reactive({})      // { [说说ID]: 输入框草稿 }

async function toggleComments(t) {
  if (openId.value === t.id) {
    openId.value = null // 再点一下收起
    return
  }
  openId.value = t.id
  // 第一次展开才拉数据，收起再展开用缓存
  if (!commentsMap[t.id]) {
    try {
      const res = await api.get('/api/talks/' + t.id + '/comments')
      commentsMap[t.id] = res.data
    } catch (e) {
      ElMessage.error(e.message)
    }
  }
}

async function sendComment(t) {
  if (!me.value) {
    ElMessage.warning('登录后才能评论哦')
    return
  }
  const text = (drafts[t.id] || '').trim()
  if (!text) {
    ElMessage.warning('写点啥再发吧')
    return
  }
  try {
    const res = await api.post('/api/talks/' + t.id + '/comments', { content: text })
    ;(commentsMap[t.id] = commentsMap[t.id] || []).push(res.data) // 本地追加，不用重新拉
    drafts[t.id] = ''
    t.comment_count++ // 计数器 +1，按钮上的数字立刻变
  } catch (e) {
    ElMessage.error(e.message)
  }
}

onMounted(loadTalks) // 点赞记录不在这儿拉——交给上面的 watch，它要等 me 加载完
</script>

<style scoped src="@/views/css/TalksView.css"></style>

<template>
  <div class="talks-page">
    <!-- 发布框（只有博主显示） -->
    <div v-if="isOwner()" class="talk-composer">
      <img class="composer-avatar" :src="me.avatar || defaultAvatar" alt="头像" />
      <div class="composer-main">
        <textarea
          v-model="content"
          class="composer-input"
          rows="3"
          maxlength="1000"
          placeholder="此刻的想法……（可长可短，最多1000字）"
        ></textarea>

        <!-- 九宫格预览：已上传的图（el-image 点击可放大，本次传的全部图可以左右翻看） -->
        <div v-if="pics.length" class="pic-grid small">
          <div v-for="(p, i) in pics" :key="p" class="pic-cell">
            <el-image
              class="pic-thumb"
              :src="p"
              :preview-src-list="pics"
              :initial-index="i"
              fit="cover"
              :hide-on-click-modal="true"
            />
            <span class="pic-remove" title="移除" @click.stop="removePic(i)">×</span>
          </div>
          <!-- 没满9张时显示"加号"格子，点击加图 -->
          <div v-if="pics.length < 9" class="pic-cell pic-add" @click="pickPics">
            {{ uploading ? '···' : '+' }}
          </div>
        </div>
        <!-- 一张都没有时也要有入口 -->
        <div v-if="pics.length === 0" class="composer-tools">
          <button class="btn-add" :disabled="uploading" @click="pickPics">🖼 添加图片（最多9张）</button>
        </div>

        <div class="composer-foot">
          <span class="pic-count">{{ pics.length }}/9 图</span>
          <button class="btn-publish" :disabled="publishing || uploading" @click="publish">
            {{ publishing ? '发表中……' : '发 表' }}
          </button>
        </div>
      </div>
      <!-- 隐藏的多选文件框 -->
      <input ref="fileInput" type="file" accept="image/*" multiple class="hidden-input" @change="handleFilesChange" />
    </div>

    <!-- 动态流：点卡片进详情页；卡上的按钮/图/评论区都 stop，不会误触跳转 -->
    <div v-for="t in talks" :key="t.id" class="talk-card" @click="goDetail(t.id)">
      <img class="talk-avatar" :src="t.avatar || defaultAvatar" alt="头像" @click.stop="goUser(t.user_id)" />
      <div class="talk-main">
        <div class="talk-head">
          <span class="talk-name" @click.stop="goUser(t.user_id)">{{ t.nickname }}</span>
          <span class="talk-time">{{ (t.created_at || '').slice(0, 16).replace('T', ' ') }}</span>
        </div>
        <p class="talk-content">{{ t.content }}</p>

        <!-- 九宫格：el-image 自带"点击放大看原图"（preview-src-list） -->
        <div v-if="t._imgs.length" class="pic-grid" :class="gridClass(t._imgs.length)" @click.stop>
          <el-image
            v-for="(img, idx) in t._imgs"
            :key="img"
            class="pic-thumb"
            :src="img"
            :preview-src-list="t._imgs"
            :initial-index="idx"
            fit="cover"
            :hide-on-click-modal="true"
          />
        </div>

        <!-- 底部操作行：互动按钮靠左，删除靠右（博主可删任何人的） -->
        <div class="talk-foot" @click.stop>
          <span class="foot-actions">
            <button class="tk-like" :class="{ on: likedSet.has(t.id) }" @click="toggleLike(t)">
              {{ likedSet.has(t.id) ? '❤️' : '🤍' }} {{ t.likes }}
            </button>
            <button class="tk-cbtn" @click="toggleComments(t)">
              💬 {{ t.comment_count }} 条评论
            </button>
          </span>
          <a
            v-if="isMine(t) || isOwner()"
            href="#"
            class="talk-delete"
            @click.prevent="removeTalk(t)"
          >删除</a>
        </div>

        <!-- 内嵌评论区：点💬展开，再点收起 -->
        <div v-if="openId === t.id" class="tk-panel" @click.stop>
          <div v-for="cm in commentsMap[t.id] || []" :key="cm.id" class="tk-comment">
            <img class="tk-cavatar" :src="cm.avatar || defaultAvatar" alt="头像" @click="goUser(cm.user_id)" />
            <div class="tk-cmain">
              <span class="tk-cname" @click="goUser(cm.user_id)">{{ cm.nickname }}</span>
              <p class="tk-ctext">{{ cm.content }}</p>
            </div>
            <button v-if="canDelete(cm)" class="tk-cdel" @click="removeComment(t, cm)">删除</button>
          </div>
          <p v-if="!(commentsMap[t.id] || []).length" class="tk-cempty">还没有评论，来抢沙发～</p>

          <!-- 评论输入框：登录才显示；游客给提示 -->
          <div v-if="me" class="tk-cinput">
            <input
              v-model="drafts[t.id]"
              class="tk-input"
              maxlength="500"
              placeholder="友善评论，最多500字"
              @keyup.enter="sendComment(t)"
            />
            <button class="tk-send" @click="sendComment(t)">发送</button>
          </div>
          <p v-else class="tk-cempty">
            <RouterLink to="/login" class="tk-login">登录</RouterLink>后才能评论哦
          </p>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <p v-if="talks.length === 0" class="empty-tip">还没有动态，来发第一条吧～</p>

    <!-- 分页：后端说还有 → 显示"加载更多" -->
    <div v-if="hasMore" class="load-more-wrap">
      <button class="load-more" :disabled="loadingMore" @click="loadMoreTalks">
        {{ loadingMore ? '加载中…' : '加载更多' }}
      </button>
    </div>
  </div>
</template>
