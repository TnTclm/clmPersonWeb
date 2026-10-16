<script setup>
// ============================================================
// AdminView —— 内容管理（博主专属）
// 全站只剩说说一种内容：这里看全部说说 + 删除任何违规内容
// 访问控制：路由守卫拦"没登录的"；博主身份前后端双重校验
// ============================================================

import { ref, onMounted } from 'vue'
import { api } from '@/api'
import { me } from '@/stores/me'
import { ElMessage, ElMessageBox } from 'element-plus' // Element 的消息气泡 + 确认弹窗

// ---------- 全部说说列表 ----------
const talks = ref([])

// images 是 JSON 字符串，数一下有几张图
function imgCount(images) {
  try {
    const arr = JSON.parse(images || '[]')
    return Array.isArray(arr) ? arr.length : 0
  } catch {
    return 0
  }
}

// 内容摘要：无文字的纯图说说给个占位词
function summary(t) {
  const text = (t.content || '').trim()
  if (!text) return '（纯图说说）'
  return text.length > 50 ? text.slice(0, 50) + '……' : text
}

// 博主身份：me 是异步加载的，函数写法每次渲染都会重算
const isOwner = () => me.value && me.value.is_owner

async function loadTalks() {
  const res = await api.get('/api/talks')
  talks.value = res.data
}

onMounted(loadTalks)

// 点"删除"：Element 确认弹窗防手滑；后端还有博主校验兜底
async function handleDelete(talk) {
  try {
    await ElMessageBox.confirm(
      '确定删除这条说说吗？配图文件会一并删除！',
      '删除确认',
      { confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning' }
    )
  } catch {
    return // 用户点了"取消"，什么都不做
  }
  try {
    await api.del('/api/talks/' + talk.id)
    ElMessage.success('删除成功')
    loadTalks()
  } catch (e) {
    ElMessage.error(e.message)
  }
}
</script>

<style scoped src="@/views/css/AdminView.css"></style>

<template>
  <div class="admin-page">
    <!-- 非博主：温和拦截（后端也会 403 兜底） -->
    <div v-if="!isOwner()" class="admin-card">
      <p class="empty-tip">这个页面只有博主能使用哦</p>
    </div>

    <template v-else>
      <!-- 说说管理列表 -->
      <div class="admin-card">
        <h2>说说管理（{{ talks.length }} 条）</h2>
        <p class="admin-tip">发现垃圾内容直接删——记录、评论、配图文件会一起清掉</p>
        <p v-if="talks.length === 0" class="empty-tip">还没有任何说说～</p>
        <div v-for="t in talks" :key="t.id" class="manage-row">
          <div class="row-info">
            <span class="row-title">{{ summary(t) }}</span>
            <span class="row-meta">
              {{ t.nickname }} · {{ (t.created_at || '').slice(0, 16).replace('T', ' ') }}
              · {{ imgCount(t.images) ? '🖼 ' + imgCount(t.images) + ' 图' : '无图' }}
            </span>
          </div>
          <div class="row-actions">
            <button class="btn-danger" @click="handleDelete(t)">删除</button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
