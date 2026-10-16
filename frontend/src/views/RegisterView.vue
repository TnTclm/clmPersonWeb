<script setup>
// ============================================================
// RegisterView —— 注册页
// 流程：填表单 → 验证码 → 提交注册 → 跳登录页
// 和 LoginView 的两大区别：
//   1. 表单多一个昵称（nickname），是文章作者位置显示给别人看的名字
//   2. 注册成功后端【不发 token】，所以不能自动登录，只能跳去登录页
// ============================================================

import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router' // 注册成功要跳页面，所以这回用得上它
import { api } from '@/api'
import { ElMessage } from 'element-plus'
import EyeToggle from '@/components/EyeToggle.vue' // 密码显隐小眼睛

const router = useRouter()

// ---------- 表单状态（ref 化 = 响应式，值一变页面自动更新） ----------
const nickname = ref('')        // 昵称：1-20 字，显示给别人的
const username = ref('')        // 用户名：3-20 字，登录用的账号
const password = ref('')        // 密码：6-32 位
const confirmPassword = ref('') // 确认密码：再输一遍，防手滑打错
const showPwd = ref(false)      // 小眼睛：true=明文显示密码
const showConfirm = ref(false)  // 确认密码那栏自己的眼睛
const captchaCode = ref('')     // 用户输入的验证码答案
const captchaImg = ref('')      // 后端返回的 base64 验证码图片
const captchaId = ref('')       // 验证码"号牌"，提交时要带上，证明是同一张图

// 取一张新验证码：记号牌 + 显示图片（和 LoginView 一字不差）
async function refreshCaptcha() {
  const res = await api.get('/api/captcha')
  captchaId.value = res.data.captchaId
  captchaImg.value = res.data.img
}

// 进页面自动来一张
onMounted(refreshCaptcha)

// ---------- 注册 ----------
async function handleRegister() {
  // 前端先拦一道：两次密码要一致（后端只收一次密码，"确认一遍"是纯前端体验设计）
  if (password.value !== confirmPassword.value) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }
  try {
    // 长度校验后端都做了（用户名3-20、密码6-32、昵称1-20）
    // 不合法会返回错误 msg，被 catch 住弹出来，前端不用自己写
    await api.post('/api/register', {
      nickname: nickname.value.trim(), // .trim() 去首尾空格
      username: username.value.trim(),
      password: password.value,
      captcha_id: captchaId.value,     // 注意：后端字段是下划线命名
      captcha_code: captchaCode.value.trim(),
    })
    // 走到这里 = 注册成功
    // ⚠️ 和登录不同：后端不返回 token，所以这里【没有】localStorage.setItem
    // 注册之后让用户去登录页面登录获取 token
    ElMessage.success('注册成功，请登录')
    router.push('/login') // 跳去登录页，让用户用新账号登录
  } catch (e) {
    // 失败：可能是"用户名已被注册"、"验证码错误"等，后端会给出 msg
    ElMessage.error(e.message)
    refreshCaptcha() // 验证码是一次性的（查完就删），失败了必须换一张新的
  }
}
</script>

<style scoped src="@/views/css/RegisterView.css"></style>

<template>
  <div class="auth-card">
    <h1>注册</h1>

    <!-- v-model 双向绑定：打字 → 变量变；变量变 → 输入框变 -->
    <input v-model="username" placeholder="用户名（3-20字）" />
    <input v-model="nickname" placeholder="昵称（1-20字）" />
    <!-- 密码栏自带小眼睛：:type 动态切换明文/圆点，永远在，不靠浏览器赏脸 -->
    <div class="pwd-wrap">
      <input v-model="password" :type="showPwd ? 'text' : 'password'" placeholder="密码（6-32位）" />
      <EyeToggle v-model="showPwd" />
    </div>
    <div class="pwd-wrap">
      <input v-model="confirmPassword" :type="showConfirm ? 'text' : 'password'" placeholder="确认密码（再输一遍）" @keyup.enter="handleRegister" />
      <EyeToggle v-model="showConfirm" />
    </div>

    <div class="captcha-row">
      <!-- @keyup.enter：按回车也能提交注册 -->
      <input v-model="captchaCode" placeholder="验证码" @keyup.enter="handleRegister" />
      <!-- :src 绑定变量（没冒号只会找字面量）；@click 点击换一张 -->
      <img :src="captchaImg" title="看不清？点我刷新" @click="refreshCaptcha" />
    </div>

    <button @click="handleRegister">注 册</button>

    <p class="tip">已有账号？<RouterLink to="/login">去登录</RouterLink></p>
  </div>
</template>
