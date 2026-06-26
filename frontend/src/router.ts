import { createRouter, createWebHashHistory } from 'vue-router'
import Overview from './views/Overview.vue'
import Sessions from './views/Sessions.vue'
import SessionDetail from './views/SessionDetail.vue'
import Tools from './views/Tools.vue'
import Settings from './views/Settings.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/overview' },
    { path: '/overview', component: Overview },
    { path: '/sessions', component: Sessions },
    { path: '/sessions/:id', component: SessionDetail, props: true },
    { path: '/tools', component: Tools },
    { path: '/settings', component: Settings }
  ]
})

export default router
