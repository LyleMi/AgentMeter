import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/attention' },
    { path: '/attention', component: () => import('./views/Attention.vue') },
    {
      path: '/analysis/usage',
      component: () => import('./views/Tokens.vue'),
      children: [
        { path: '', component: () => import('./views/analysis/UsageSummary.vue') },
        { path: 'projects', component: () => import('./views/tokens/TokensProjects.vue') },
        { path: 'breakdown', component: () => import('./views/tokens/TokensBreakdown.vue') },
        { path: 'sessions', component: () => import('./views/tokens/TokensSessions.vue') }
      ]
    },
    {
      path: '/analysis/time',
      component: () => import('./views/OverviewTime.vue'),
      children: [
        { path: '', component: () => import('./views/time/TimeSummary.vue') },
        { path: 'sources', component: () => import('./views/time/TimeSources.vue') },
        { path: 'tools', component: () => import('./views/time/ToolDurationLeaders.vue') },
        { path: 'sessions', component: () => import('./views/time/SlowSessionsTable.vue') }
      ]
    },
    { path: '/analysis/models', component: () => import('./views/ModelSignals.vue') },
    { path: '/analysis/models/trends', component: () => import('./views/ModelSignals.vue') },
    { path: '/analysis/models/compare', component: () => import('./views/ModelSignals.vue') },
    {
      path: '/analysis/tools',
      component: () => import('./views/Tools.vue'),
      children: [
        { path: '', component: () => import('./views/analysis/ToolsSummary.vue') },
        { path: 'shell', component: () => import('./views/ToolsShell.vue') },
        { path: 'calls', component: () => import('./views/ToolsCalls.vue') }
      ]
    },
    { path: '/sessions', component: () => import('./views/Sessions.vue') },
    { path: '/sessions/:id', component: () => import('./views/SessionDetail.vue'), props: true },
    {
      path: '/safety/audit',
      component: () => import('./views/Audit.vue'),
      children: [
        { path: '', component: () => import('./views/AuditSummary.vue') },
        { path: 'findings', component: () => import('./views/AuditFindings.vue') },
        { path: 'findings/:id', component: () => import('./views/AuditDetail.vue') }
      ]
    },
    { path: '/safety/privacy', component: () => import('./views/AgentPrivacy.vue') },
    { path: '/resources/prompts', component: () => import('./views/Prompts.vue') },
    { path: '/resources/agents', component: () => import('./views/AgentResources.vue') },
    {
      path: '/settings',
      component: () => import('./views/Settings.vue'),
      redirect: '/settings/sources',
      children: [
        { path: 'sources', component: () => import('./views/SettingsSource.vue') },
        { path: 'data', component: () => import('./views/SettingsDatabase.vue') },
        { path: 'pricing', component: () => import('./views/SettingsPrice.vue') },
        { path: 'display', component: () => import('./views/SettingsDisplay.vue') }
      ]
    },
    { path: '/:pathMatch(.*)*', component: () => import('./views/NotFound.vue') }
  ]
})

export default router
