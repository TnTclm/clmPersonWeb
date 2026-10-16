
<!-- 样式和注册页共用一份（改一处两边一起变） -->
<style scoped src="@/views/css/RegisterView.css"></style>


<template>
  <div class="auth-card">
    <h1>登录</h1>

    <!-- v-model：双向绑定。打字 → 变量变；变量变 → 输入框变 -->
    <input v-model="username" placeholder="用户名" />
    <!-- :type 动态切换明文/圆点；眼睛是自己的，永远在，不靠浏览器赏脸 -->
    <div class="pwd-wrap">
      <input v-model="password" :type="showPwd ? 'text' : 'password'" placeholder="密码" @keyup.enter="handleLogin" />
      <EyeToggle v-model="showPwd" />
    </div>

    <div class="captcha-row">
      <!-- @keyup.enter：按回车也能登录 -->
      <input v-model="captchaCode" placeholder="验证码" @keyup.enter="handleLogin" />
      <!-- :src 绑定变量（没冒号只会找字面量）；@click 点击换一张 -->
      <img :src="captchaImg" title="看不清？点我刷新" @click="refreshCaptcha" />
    </div>

    <button @click="handleLogin">登 录</button>

    <p class="tip">没有账号？<RouterLink to="/register">去注册</RouterLink></p>
  </div>
</template>



<script setup>
// ============================================================
// LoginView —— 登录页
// 流程：填表单 → 验证码 → 提交 → 存令牌 → 问一次"我是谁" → 跳首页
// ============================================================

import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router' // setup 里做页面跳转要用它
import { api } from '@/api'
import { fetchMe } from '@/stores/me'  // 登录成功后同步 header
import { ElMessage } from 'element-plus'
import EyeToggle from '@/components/EyeToggle.vue' // 密码显隐小眼睛

const router = useRouter()

// ---------- 表单状态（ref 化 = 响应式，值一变页面自动更新） ----------
const username = ref('')
const password = ref('')
const showPwd = ref(false)   // 小眼睛：true=明文显示密码
const captchaCode = ref('')  // 用户输入的验证码
const captchaImg = ref('')   // 后端返回的 base64 图片
const captchaId = ref('')    // 验证码"号牌"，提交时要带上

// 取一张新验证码：记号牌 + 显示图片
async function refreshCaptcha() {
  const res = await api.get('/api/captcha')
  captchaId.value = res.data.captchaId
  captchaImg.value = res.data.img
}

// 进页面自动来一张
onMounted(refreshCaptcha)

// ---------- 登录 ----------
async function handleLogin() {
  try {
    const res = await api.post('/api/login', {
      username: username.value.trim(), // .trim() 去首尾空格
      password: password.value,
      captcha_id: captchaId.value,
      captcha_code: captchaCode.value.trim(),
    })
    // 成功：门禁卡存进 localStorage，之后 api.js 每个请求自动带上
    localStorage.setItem('token', res.data.token)
    await fetchMe()  // 先问清"我是谁"，到首页 header 立刻显示"你好，xx"
    router.push('/') // 跳首页（SPA 内跳转，不刷新）
  } catch (e) {
    ElMessage.error(e.message)
    refreshCaptcha() // 旧验证码已作废，必须换一张
  }
}
</script>








