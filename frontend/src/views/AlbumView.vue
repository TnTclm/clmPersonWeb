<script setup>
// ============================================================
// AlbumView —— 相册（博主专属，博客式纵向卡片流）
// 一条内容 = 一段说明 + 多个照片/视频（多选一次上传）
// 发布/编辑说明/删除 = 博主；浏览公开
// ============================================================

import { ref, onMounted, onUnmounted } from 'vue'
import { api } from '@/api'
import { me } from '@/stores/me'
import { ElMessage, ElMessageBox } from 'element-plus'

// ---------- 相册内容列表 ----------
const posts = ref([])      // [{ id, title, created_at, _imgs: [...], _vids: [...] }]
const loading = ref(true)

// 把每条内容的 medias 拆成 图片数组 / 视频数组，模板好渲染
function decorate(p) {
  return {
    ...p,
    _imgs: p.medias.filter(m => m.type === 'image').map(m => m.url),
    _vids: p.medias.filter(m => m.type === 'video').map(m => m.url),
  }
}

// 卡片头的计数文案："📷 3 张照片 · 🎬 2 个视频"
// 放在卡片最顶上，一眼看出这几个文件是同一次上传的
function countLabel(p) {
  const parts = []
  if (p._imgs.length) parts.push('📷 ' + p._imgs.length + ' 张照片')
  if (p._vids.length) parts.push('🎬 ' + p._vids.length + ' 个视频')
  return parts.join(' · ')
}

// 按图片数量决定网格布局：1 张大图、2/4 张两列（4 张正好 2×2 铺满）、其余三列
// 铺满卡片宽度，格子跟着变大，右边就不会空一大块
function imgGridClass(p) {
  const n = p._imgs.length
  if (n === 1) return 'single'
  if (n === 2 || n === 4) return 'two'
  return 'three'
}

async function loadAlbum() {
  loading.value = true
  try {
    const res = await api.get('/api/album')
    posts.value = res.data.map(decorate)
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

// ---------- 发布（博主）：说明 + 多选文件，一次提交 ----------
const title = ref('')        // 说明文字（可空）
const picked = ref([])       // 选中的文件数组
const pickedNames = ref('')  // 选中的文件名清单（展示用）
const fileInput = ref(null)
const uploading = ref(false)

function pickFiles() {
  fileInput.value.click()
}

// 多选文件：最多9个，超出的忽略
function handleFilesChange(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  picked.value = files.slice(0, 9)
  if (files.length > 9) ElMessage.warning('一次最多9个文件，多出来的没选')
  // 文件名清单太长就截断展示
  const names = picked.value.map(f => f.name)
  pickedNames.value = names.length > 3 ? names.slice(0, 3).join('、') + ` 等${names.length}个文件` : names.join('、')
}

async function publish() {
  if (!picked.value.length) {
    ElMessage.warning('先选几个照片或视频吧')
    return
  }
  uploading.value = true
  try {
    // 一个请求带全部文件：多个同名字段 files，后端 MultipartForm.File["files"] 全收
    const fd = new FormData()
    fd.append('title', title.value.trim())
    for (const f of picked.value) {
      fd.append('files', f)
    }
    await api.upload('/api/album', fd)
    ElMessage.success('发布成功')
    title.value = ''
    picked.value = []
    pickedNames.value = ''
    loadAlbum() // 重拉列表，新内容在最上
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    uploading.value = false
  }
}

// ---------- 编辑说明（博主） ----------
async function editTitle(p) {
  let value
  try {
    const result = await ElMessageBox.prompt('改一下这条内容的说明吧（100字内）', '编辑说明', {
      confirmButtonText: '保存',
      cancelButtonText: '取消',
      inputValue: p.title || '',
      inputPattern: /^[\s\S]{0,100}$/,
      inputErrorMessage: '最多100字',
    })
    value = result.value.trim()
  } catch (e) {
    return // 取消不算错
  }
  try {
    const res = await api.put('/api/album/posts/' + p.id, { title: value })
    p.title = res.data.title // 本地同步
    ElMessage.success('说明已保存')
  } catch (e) {
    ElMessage.error(e.message)
  }
}

// ---------- 删除（博主）：整条内容连文件一起删 ----------
async function removePost(p) {
  try {
    await ElMessageBox.confirm('确定删除这条内容吗？里面的文件也会一并删除', '删除确认', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.del('/api/album/posts/' + p.id)
    ElMessage.success('删除成功')
    loadAlbum()
  } catch (e) {
    ElMessage.error(e.message)
  }
}

const isOwner = () => me.value && me.value.is_owner

// ---------- 视频预览：卡片里是小缩略样，点击弹出大播放器 ----------
const previewVid = ref('')   // 正在预览的视频 url，空字符串 = 弹层关闭

function openPreview(url) {
  previewVid.value = url
}

function closePreview() {
  previewVid.value = ''      // video 随 v-if 一起卸载，播放自动停止
}

// 按 ESC 也能关弹层
function onKeydown(e) {
  if (e.key === 'Escape') closePreview()
}

onMounted(() => {
  loadAlbum()
  window.addEventListener('keydown', onKeydown)
})
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped src="@/views/css/AlbumView.css"></style>

<template>
  <div class="album-page">
    <!-- 页头 -->
    <header class="album-head">
      <div>
        <h1 class="album-title">📷 相册</h1>
        <p class="album-sub">一次可以上传多个照片/视频，它们会同属一条内容</p>
      </div>
    </header>

    <!-- 发布框（仅博主）：说明 + 多选文件 -->
    <div v-if="isOwner()" class="album-composer">
      <textarea
        v-model="title"
        class="composer-input"
        rows="2"
        maxlength="100"
        placeholder="这次的内容想说什么？（可空，最多100字）"
      ></textarea>
      <div class="composer-foot">
        <button class="btn-pick" :disabled="uploading" @click="pickFiles">
          📎 选择文件（可多选，最多9个）
        </button>
        <span class="pick-note">{{ pickedNames || '还没选文件' }}</span>
        <button class="btn-publish" :disabled="uploading" @click="publish">
          {{ uploading ? '发布中……' : '发 布' }}
        </button>
      </div>
      <!-- 隐藏的多选文件框 -->
      <input
        ref="fileInput"
        type="file"
        accept="image/*,video/*"
        multiple
        class="hidden-input"
        @change="handleFilesChange"
      />
    </div>

    <!-- 加载中 / 空状态 -->
    <p v-if="loading" class="state-tip">加载中……</p>
    <p v-else-if="posts.length === 0" class="state-tip">相册还是空的，发布第一条吧～</p>

    <!-- 纵向卡片流：一条内容 = 一个通栏卡片 -->
    <div v-for="p in posts" :key="p.id" class="album-card">
      <!-- 卡片头：文件计数 + 时间，开头就标明"这些文件是同一次发的" -->
      <div class="card-head">
        <span class="card-count">{{ countLabel(p) }}</span>
        <span class="card-time">{{ (p.created_at || '').slice(0, 16).replace('T', ' ') }}</span>
      </div>

      <!-- 说明（可空）：有字才显示 -->
      <p v-if="p.title" class="card-title">{{ p.title }}</p>

      <!-- 图片：按数量决定布局，点击放大 -->
      <div v-if="p._imgs.length" class="card-img-grid" :class="imgGridClass(p)">
        <el-image
          v-for="(img, idx) in p._imgs"
          :key="img"
          class="card-img"
          :src="img"
          :preview-src-list="p._imgs"
          :initial-index="idx"
          fit="cover"
          :hide-on-click-modal="true"
        />
      </div>

      <!-- 视频：缩略小样（第一帧当封面 + 播放按钮），点击弹出大播放器 -->
      <div v-if="p._vids.length" class="card-vid-row">
        <div
          v-for="v in p._vids"
          :key="v"
          class="vid-thumb"
          @click="openPreview(v)"
        >
          <video class="vid-cover" :src="v" preload="metadata" muted></video>
          <span class="vid-play">▶</span>
        </div>
      </div>

      <!-- 底部：博主操作（时间已挪到卡片头，访客看不到这行） -->
      <div v-if="isOwner()" class="card-foot">
        <span class="card-actions">
          <button class="card-edit" @click="editTitle(p)">编辑说明</button>
          <button class="card-del" @click="removePost(p)">删除</button>
        </span>
      </div>
    </div>

    <!-- 视频大播放器弹层：点遮罩空白处 / 右上角× / ESC 关闭 -->
    <div v-if="previewVid" class="vid-mask" @click.self="closePreview">
      <video class="vid-player" :src="previewVid" controls autoplay></video>
      <button class="vid-close" @click="closePreview">×</button>
    </div>
  </div>
</template>
