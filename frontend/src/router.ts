import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/overview' },
    { path: '/overview', component: () => import('./views/Overview.vue') },
    { path: '/sessions', component: () => import('./views/Sessions.vue') },
    { path: '/sessions/:id', component: () => import('./views/SessionDetail.vue'), props: true },
    { path: '/tools', component: () => import('./views/Tools.vue') },
    { path: '/settings', component: () => import('./views/Settings.vue') }
  ]
})

export default router
