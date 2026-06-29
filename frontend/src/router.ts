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
        { path: 'time', redirect: '/time' },
        { path: 'breakdown', component: () => import('./views/OverviewBreakdown.vue') },
        { path: 'recent', component: () => import('./views/OverviewRecent.vue') }
      ]
    },
    {
      path: '/time',
      component: () => import('./views/OverviewTime.vue'),
      children: [
        { path: '', component: () => import('./views/time/TimeSummary.vue') },
        { path: 'summary', redirect: (to) => ({ path: '/time', query: to.query }) },
        { path: 'sources', component: () => import('./views/time/TimeSources.vue') },
        { path: 'tools', component: () => import('./views/time/ToolDurationLeaders.vue') },
        { path: 'sessions', component: () => import('./views/time/SlowSessionsTable.vue') }
      ]
    },
    {
      path: '/tokens',
      component: () => import('./views/Tokens.vue'),
      children: [
        { path: '', component: () => import('./views/tokens/TokensSummary.vue') },
        { path: 'summary', component: () => import('./views/tokens/TokensSummary.vue') },
        { path: 'trends', component: () => import('./views/tokens/TokensTrends.vue') },
        { path: 'breakdown', component: () => import('./views/tokens/TokensBreakdown.vue') },
        { path: 'sessions', component: () => import('./views/tokens/TokensSessions.vue') }
      ]
    },
    { path: '/model-signals', component: () => import('./views/ModelSignals.vue') },
    { path: '/model-signals/risk', component: () => import('./views/ModelRisk.vue') },
    { path: '/sessions', component: () => import('./views/Sessions.vue') },
    { path: '/sessions/:id', component: () => import('./views/SessionDetail.vue'), props: true },
    { path: '/prompts', component: () => import('./views/Prompts.vue') },
    {
      path: '/audit',
      component: () => import('./views/Audit.vue'),
      redirect: (to) => ({ path: '/audit/summary', query: to.query }),
      children: [
        { path: 'summary', component: () => import('./views/AuditSummary.vue') },
        { path: 'findings', component: () => import('./views/AuditFindings.vue') },
        { path: 'findings/:id', component: () => import('./views/AuditDetail.vue') }
      ]
    },
    { path: '/agent-privacy', component: () => import('./views/AgentPrivacy.vue') },
    {
      path: '/tools',
      component: () => import('./views/Tools.vue'),
      redirect: '/tools/overview',
      children: [
        { path: 'overview', component: () => import('./views/ToolsOverview.vue') },
        { path: 'summary', component: () => import('./views/ToolsSummary.vue') },
        { path: 'shell', component: () => import('./views/ToolsShell.vue') },
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
        { path: 'price', component: () => import('./views/SettingsPrice.vue') },
        { path: 'display', component: () => import('./views/SettingsDisplay.vue') }
      ]
    }
  ]
})

export default router
