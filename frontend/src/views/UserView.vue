<script setup>
// ============================================================
// UserView —— 个人空间页
// 网址 /user/3 = "3号用户的个人主页"
// 显示：头像、昵称、个性签名、加入时间、活跃度热力图
//       说说列表只挂在博主的空间下（其他人发不了说说，列表永远是空的）
// 特权：看自己的空间 → 上传头像 + 编辑资料（昵称/个性签名）
// ============================================================

import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api'
import { me, fetchMe } from '@/stores/me'
import { ElMessage } from 'element-plus'
import defaultAvatar from '@/assets/images/avatar-default.svg' // 默认灰头像

const route = useRoute()
const router = useRouter()

// 空间主人的信息（后端 /api/users/:id 返回）
const user = ref(null)     // { id, nickname, avatar, joinDate, talkCount }
const talks = ref([])      // TA 最近的说说
const error = ref('')

// 头像上传相关
const fileInput = ref(null)  // 指向模板里那个隐藏的 <input type="file">
const uploading = ref(false) // 上传中标记

// 网址 /user/3 里的 3
const userId = route.params.id

// 拉取个人空间数据
async function loadProfile() {
  try {
    const res = await api.get('/api/users/' + userId)
    user.value = res.data.user
    talks.value = res.data.talks
  } catch (e) {
    error.value = e.message
  }
}

// ---------- 活跃度热力图（LeetCode式） ----------
const weeks = ref([])        // 53列格子，一列=一周7天：[{date, count, future}, ...]
const monthLabels = ref([])  // 月份标签 [{col: 第几列, text: '10月'}]

// 日期 → "2026-10-14"（必须和后端存的格式一致，否则对不上格子）
function fmtDate(d) {
  return d.getFullYear() + '-' +
    String(d.getMonth() + 1).padStart(2, '0') + '-' +
    String(d.getDate()).padStart(2, '0')
}

// 次数 → 颜色档位：0灰 / 1-2浅绿 / 3-5绿 / 6-9深绿 / 10+墨绿
function actLevel(cell) {
  if (cell.future || !cell.count) return 0
  if (cell.count <= 2) return 1
  if (cell.count <= 5) return 2
  if (cell.count <= 9) return 3
  return 4
}

// 用后端数据铺出整张热力图
function buildHeatmap(list) {
  const countMap = {} // { '2026-10-14': 3 } 查表用
  for (const it of list) {
    countMap[it.date] = it.count
  }

  // 起点：364 天前，再退回到那个星期的周日（一列=一周，和 GitHub 一个规矩）
  const today = new Date()
  const start = new Date(today)
  start.setDate(start.getDate() - 364)
  start.setDate(start.getDate() - start.getDay())
  // 终点：补齐到本周六，让最后一列也是完整的 7 格
  const end = new Date(today)
  end.setDate(end.getDate() + (6 - end.getDay()))

  const cols = []
  const labels = []
  const cursor = new Date(start)
  let week = []
  while (cursor <= end) {
    const dateStr = fmtDate(cursor)
    const future = cursor > today // 本周还没过的天：占位但不上色
    // 某列里撞见"1号" = 一个月开始了 → 在这列上方贴月份标签
    if (cursor.getDate() === 1 && !future) {
      labels.push({ col: cols.length, text: (cursor.getMonth() + 1) + '月' })
    }
    week.push({ date: dateStr, count: countMap[dateStr] || 0, future })
    if (week.length === 7) {
      cols.push(week)
      week = []
    }
    cursor.setDate(cursor.getDate() + 1)
  }
  weeks.value = cols
  monthLabels.value = labels
}

// 拉热力图数据（公开接口，游客也能看任何人的）
async function loadActivities() {
  try {
    const res = await api.get('/api/users/' + userId + '/activities')
    buildHeatmap(res.data)
  } catch (e) {
    // 热力图挂了不能拖累页面主体，静默降级
  }
}

// 点"上传头像"按钮 → 让隐藏的文件选择框弹出来
function pickFile() {
  fileInput.value.click() // 程序替我们"点"那个 input
}

// 用户选完文件后自动触发（change 事件）
async function handleFileChange(e) {
  const file = e.target.files[0]
  if (!file) return
  // FormData：浏览器上传文件的标准"信封"
  // 'avatar' 这个名字必须和后端 FormFile("avatar") 一致
  const fd = new FormData()
  fd.append('avatar', file)
  uploading.value = true
  try {
    await api.upload('/api/avatar', fd)
    await fetchMe()     // 共享登录态里的头像也刷新 → header 立刻换新
    await loadProfile() // 重新拉资料 → 页面头像立刻换新
    alert('头像更新成功')
  } catch (err) {
    alert(err.message)
  } finally {
    uploading.value = false
    e.target.value = '' // 清空选择，下次选同一个文件也能触发 change
  }
}

// 看的是不是自己的空间：我登录着 && 主人ID == 我的ID
// 是普通函数不是 computed：me 是异步加载的，函数每次渲染都会重算，
// header 那边 fetchMe 一完成，这里按钮就会"长出来"
const isMine = () => me.value && user.value && me.value.user_id === user.value.id

// ---------- 编辑资料（仅本人）：昵称 + 个性签名 ----------
const editing = ref(false)
const nickInput = ref('')
const bioInput = ref('')
const saving = ref(false)

function startEdit() {
  nickInput.value = user.value.nickname
  bioInput.value = user.value.bio || ''
  editing.value = true
}

async function saveProfile() {
  if (!nickInput.value.trim()) {
    ElMessage.warning('昵称不能是空白')
    return
  }
  saving.value = true
  try {
    const res = await api.put('/api/me', {
      nickname: nickInput.value.trim(),
      bio: bioInput.value.trim(),
    })
    // 后端换了新令牌（旧令牌里还是旧昵称），必须存进去，不然新说说还署旧名
    localStorage.setItem('token', res.data.token)
    await fetchMe() // 共享登录态刷新 → header 昵称立刻换新
    user.value.nickname = res.data.nickname
    user.value.bio = res.data.bio
    editing.value = false
    ElMessage.success('保存成功')
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadProfile()
  loadActivities()
})
</script>

<style scoped src="@/views/css/UserView.css"></style>

<template>
  <div class="user-page">
    <!-- 加载中 / 出错 -->
    <p v-if="!user && !error" class="state-tip">加载中……</p>
    <p v-else-if="error" class="state-tip">{{ error }}</p>

    <template v-else>
      <!-- 资料卡 -->
      <div class="profile-card">
        <!-- 头像：主人传过就显示自己的，没传用默认灰头像 -->
        <img class="profile-avatar" :src="user.avatar || defaultAvatar" alt="头像" />

        <div class="profile-info">
          <!-- 浏览态 -->
          <template v-if="!editing">
            <h1 class="profile-name">{{ user.nickname }}</h1>
            <p v-if="user.bio" class="profile-bio">{{ user.bio }}</p>
            <p class="profile-meta">加入于 {{ user.joinDate }}</p>

            <!-- 只有看自己的空间才显示（v-if 判断 isMine()） -->
            <div v-if="isMine()" class="upload-row">
              <!-- 隐藏的文件选择框：真正干活的是它，按钮只是替它跑腿 -->
              <input ref="fileInput" type="file" accept="image/*" class="hidden-input" @change="handleFileChange" />
              <button class="btn-upload" :disabled="uploading" @click="pickFile">
                {{ uploading ? '上传中……' : '📷 上传头像' }}
              </button>
              <button class="btn-upload" @click="startEdit">✏️ 编辑资料</button>
            </div>
          </template>

          <!-- 编辑态：改昵称 + 写个性签名 -->
          <div v-else class="edit-form">
            <label class="edit-label">昵称</label>
            <input
              v-model="nickInput"
              class="edit-input"
              maxlength="20"
              placeholder="怎么称呼你（最多20字）"
            />
            <label class="edit-label">个性签名</label>
            <input
              v-model="bioInput"
              class="edit-input"
              maxlength="100"
              placeholder="一句话介绍自己（可空，最多100字）"
              @keyup.enter="saveProfile"
            />
            <div class="edit-actions">
              <button class="btn-upload btn-save" :disabled="saving" @click="saveProfile">
                {{ saving ? '保存中……' : '保存' }}
              </button>
              <button class="btn-upload" @click="editing = false">取消</button>
            </div>
          </div>
        </div>
      </div>

      <!-- 活跃度热力图（LeetCode式）：纯视觉，上线=浅绿，创作越多越深 -->
      <div class="heat-card">
        <div class="heat-months">
          <span
            v-for="m in monthLabels"
            :key="m.col"
            class="heat-month"
            :style="{ left: m.col * 18 + 'px' }"
          >{{ m.text }}</span>
        </div>
        <div class="heat-grid">
          <div v-for="(week, wi) in weeks" :key="wi" class="heat-col">
            <span
              v-for="(cell, ci) in week"
              :key="ci"
              class="heat-cell"
              :class="'lv' + actLevel(cell)"
              :title="cell.future ? '' : cell.date + '：活跃 ' + cell.count + ' 次'"
            ></span>
          </div>
        </div>
      </div>

      <!-- 说说列表：只挂在博主的空间下（其他人发不了说说，列表永远是空的） -->
      <template v-if="user.is_owner">
        <h2 class="section-title">TA 的说说</h2>
        <p v-if="talks.length === 0" class="state-tip">还没发表过说说～</p>
        <div
          v-for="t in talks"
          :key="t.id"
          class="user-post"
          @click="router.push('/talks')"
        >
          <span class="user-post-title">{{ (t.content || '').trim() || '（纯图说说）' }}</span>
          <span class="user-post-date">{{ (t.created_at || '').slice(0, 10) }}</span>
        </div>
      </template>
    </template>
  </div>
</template>
