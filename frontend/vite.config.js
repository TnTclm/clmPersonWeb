import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    // vueDevTools 已移除：调试用的悬浮面板不需要了（想恢复就 npm i 后把它加回 plugins）
  ],

  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',      // 凡是 /api 开头的请求，转发给 Go 后端
      '/uploads': 'http://localhost:8080',  // 用户上传的图片也存后端，同样转发
      '/pictures': 'http://localhost:8080', // 主页图片墙的图片（D:\pictures）也由后端递送
    },
  },
})
