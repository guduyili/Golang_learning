// 引入Vue并创建应用实例
// 引入全局样式
// 引入根组件App.vue
// 将应用挂载到index.html的 #app元素上

import { createApp } from 'vue'
import './assets/reset/reset.scss'
import App from './App.vue'

createApp(App).mount('#app')

// import './style.css'
