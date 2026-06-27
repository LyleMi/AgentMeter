import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/overview' },
    {
      path: '/overview',
      component: () => import('./views/Overview.vue'),
      redirect: '/overview/summary',
      children: [
        { path: 'summary', component: () => import('./views/OverviewSummary.vue') },
        { path: 'trends', component: () => import('./views/OverviewTrends.vue') },
        { path: 'breakdown', component: () => import('./views/OverviewBreakdown.vue') },
        { path: 'recent', component: () => import('./views/OverviewRecent.vue') }
      ]
    },
    { path: '/sessions', component: () => import('./views/Sessions.vue') },
    { path: '/sessions/:id', component: () => import('./views/SessionDetail.vue'), props: true },
    { path: '/audit', component: () => import('./views/Audit.vue') },
    {
      path: '/tools',
      component: () => import('./views/Tools.vue'),
      redirect: '/tools/overview',
      children: [
        { path: 'overview', component: () => import('./views/ToolsOverview.vue') },
        { path: 'summary', component: () => import('./views/ToolsSummary.vue') },
        { path: 'calls', component: () => import('./views/ToolsCalls.vue') }
      ]
    },
    {
      path: '/settings',
      component: () => import('./views/Settings.vue'),
      redirect: '/settings/source',
      children: [
        { path: 'source', component: () => import('./views/SettingsSource.vue') },
        { path: 'database', component: () => import('./views/SettingsDatabase.vue') },
        { path: 'price', component: () => import('./views/SettingsPrice.vue') }
      ]
    }
  ]
})

export default router
