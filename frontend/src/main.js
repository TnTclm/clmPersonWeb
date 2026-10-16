import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import ElementPlus from 'element-plus'          // Element Plus 组件库本体
import 'element-plus/dist/index.css'            // 它的全套样式（要跟着一起引）
import './assets/main.css'                      // 自己的全局样式放它后面，能覆盖它

const app = createApp(App)

app.use(router)
app.use(ElementPlus) // 注册后，所有页面都能直接用 <el-xxx> 组件，不用逐个 import

app.mount('#app')
