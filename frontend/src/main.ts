import { createApp } from 'vue'
import 'ant-design-vue/dist/reset.css'
import './styles.css'
import App from './App.vue'
import router from './router'

createApp(App).use(router).mount('#app')
