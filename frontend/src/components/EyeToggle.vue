<script setup>
// ============================================================
// EyeToggle —— 密码显隐小眼睛（SVG 版）
// 用法：<EyeToggle v-model="showPwd" />，v-model 双向绑一个布尔值
// true = 明文（睁眼图标），false = 圆点（闭眼划线图标）
// ============================================================

defineProps({ modelValue: Boolean })
defineEmits(['update:modelValue'])
</script>

<style scoped>
.pwd-eye {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  cursor: pointer;
  color: #9ca3af;      /* 灰色，安静不抢戏 */
  user-select: none;   /* 连点不会选中图标 */
  transition: color 0.2s;
}

.pwd-eye:hover {
  color: #2563eb;      /* 悬浮变蓝，暗示可点 */
}
</style>

<template>
  <span class="pwd-eye" title="显示/隐藏密码" @click="$emit('update:modelValue', !modelValue)">
    <!-- 睁眼 = 当前是明文；闭眼划线 = 当前是圆点 -->
    <svg
      v-if="modelValue"
      viewBox="0 0 24 24"
      width="18"
      height="18"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7-11-7-11-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
    <svg
      v-else
      viewBox="0 0 24 24"
      width="18"
      height="18"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
      <line x1="1" y1="1" x2="23" y2="23" />
    </svg>
  </span>
</template>
