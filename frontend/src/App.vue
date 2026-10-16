<!-- ============================================================
     App.vue —— 根组件：所有页面的"外壳"
     公共部分（header）写在这，各页面差异部分交给 <RouterView>
     ============================================================ -->

<script setup>
import SiteHeader from '@/components/SiteHeader.vue'
</script>

<template>
  <!-- 每个页面都有 header，所以放在根组件里写一次 -->
  <SiteHeader />

  <!-- RouterView：一个"占位坑"
       访问 /login 时 LoginView 填进来；访问 / 时说说流（TalksView）填进来 -->
  <RouterView />
</template>
