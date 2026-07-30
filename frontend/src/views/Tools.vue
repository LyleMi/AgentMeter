<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { BarChartOutlined, CodeOutlined, HistoryOutlined } from '@ant-design/icons-vue'
import PageHeader from '../components/PageHeader.vue'
import PageTabs from '../components/PageTabs.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import { useMessages } from '../i18n'
import { useRouteTabKey } from './routeTabs'
import { applyUsageScopeToQuery, useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import {
  buildUsageAgentOptions,
  buildUsageModelOptions,
  buildUsageProjectOptions,
  useUsageScopeOptionData
} from './useUsageScopeOptions'
import { routePathWithQuery } from './routeQuery'

const toolsTabMatches = [
  { key: 'shell', pathPrefix: '/analysis/tools/shell' },
  { key: 'calls', pathPrefix: '/analysis/tools/calls' }
] as const
const route = useRoute()
const scope = useUsageScopeRoute()
const scopeOptions = useUsageScopeOptionData()

const { t } = useMessages({
  en: {
    pageTitle: 'Tools',
    pageSubtitle: 'Aggregated tool-call counts, status, duration and raw call records',
    tabOverview: 'Overview',
    tabSummary: 'Summary',
    tabShell: 'Shell',
    tabCalls: 'Calls'
  },
  'zh-CN': {
    pageTitle: '\u5de5\u5177',
    pageSubtitle: '\u6c47\u603b\u5de5\u5177\u8c03\u7528\u6b21\u6570\u3001\u72b6\u6001\u3001\u8017\u65f6\u548c\u539f\u59cb\u8c03\u7528\u8bb0\u5f55',
    tabOverview: '\u6982\u89c8',
    tabSummary: '\u6c47\u603b',
    tabShell: 'Shell \u547d\u4ee4',
    tabCalls: '\u8c03\u7528'
  }
})

const tabs = computed(() => [
  { key: 'overview', label: t('tabOverview'), path: scopedPath('/analysis/tools'), icon: BarChartOutlined },
  { key: 'shell', label: t('tabShell'), path: scopedPath('/analysis/tools/shell'), icon: CodeOutlined },
  { key: 'calls', label: t('tabCalls'), path: scopedPath('/analysis/tools/calls'), icon: HistoryOutlined }
])

const activeKey = useRouteTabKey(toolsTabMatches, 'overview')
const agentOptions = computed(() => buildUsageAgentOptions({
  sources: [scopeOptions.optionOverview.value?.agentUsage],
  selected: scope.filters.value.agent,
  fallback: 'unknown'
}))
const modelOptions = computed(() => buildUsageModelOptions({
  modelUsage: [scopeOptions.optionOverview.value?.modelUsage],
  selected: scope.filters.value.model
}))
const projectOptions = computed(() => buildUsageProjectOptions({
  projects: [scopeOptions.projectOptionRows.value],
  selected: scope.filters.value.project,
  fallback: 'unknown'
}))

function scopedPath(path: string) {
  return routePathWithQuery(path, applyUsageScopeToQuery(route.query, scope.filters.value))
}

async function loadOptions() {
  scopeOptions.applyUsageScopeOptionData(await scopeOptions.loadUsageScopeOptionData())
}

async function updateFilters(filters: UsageScopeForm) {
  await scope.updateFilters(filters)
}

onMounted(loadOptions)
</script>

<template>
  <div class="page">
    <PageHeader :title="t('pageTitle')" :subtitle="t('pageSubtitle')" />
    <UsageScopeBar
      :filters="scope.filters.value"
      :agent-options="agentOptions"
      :model-options="modelOptions"
      :project-options="projectOptions"
      @update:filters="updateFilters"
      @refresh="loadOptions"
      @clear="scope.clearFilters"
    />
    <PageTabs :tabs="tabs" :active-key="activeKey" />
    <RouterView />
  </div>
</template>
