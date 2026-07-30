<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import {
  BarChartOutlined,
  FileSearchOutlined,
  UnorderedListOutlined
} from '@ant-design/icons-vue'
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

const auditTabMatches = [
  { key: 'detail', pathPrefix: '/safety/audit/findings/' },
  { key: 'list', pathPrefix: '/safety/audit/findings' }
] as const

const route = useRoute()
const scope = useUsageScopeRoute()
const scopeOptions = useUsageScopeOptionData()
const { t } = useMessages({
  en: {
    'title': 'Audit',
    'subtitle': 'Command and privacy findings split by summary, finding list, and session-linked detail',
    'filter.agent': 'Source',
    'tab.summary': 'Summary',
    'tab.list': 'Findings',
    'tab.detail': 'Detail'
  },
  'zh-CN': {
    'title': '审计',
    'subtitle': '按汇总、发现列表和会话关联详情拆分命令与隐私发现',
    'filter.agent': '来源',
    'tab.summary': '汇总',
    'tab.list': '发现',
    'tab.detail': '详情'
  }
})

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

const activeKey = useRouteTabKey(auditTabMatches, 'summary')

const tabs = computed(() => {
  const items = [
    { key: 'summary', label: t('tab.summary'), path: scopedPath('/safety/audit'), icon: BarChartOutlined },
    { key: 'list', label: t('tab.list'), path: scopedPath('/safety/audit/findings'), icon: UnorderedListOutlined }
  ]
  if (activeKey.value === 'detail') {
    items.push({ key: 'detail', label: t('tab.detail'), path: route.fullPath, icon: FileSearchOutlined })
  }
  return items
})

function scopedPath(path: string) {
  return routePathWithQuery(path, applyUsageScopeToQuery(route.query, scope.filters.value))
}

async function loadOptions() {
  scopeOptions.applyUsageScopeOptionData(await scopeOptions.loadUsageScopeOptionData())
}

async function updateFilters(filters: UsageScopeForm) {
  await scope.updateFilters(filters)
}

loadOptions()
</script>

<template>
  <div class="page audit-page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')" />

    <UsageScopeBar
      :filters="scope.filters.value"
      :agent-options="agentOptions"
      :model-options="modelOptions"
      :project-options="projectOptions"
      @update:filters="updateFilters"
      @refresh="loadOptions"
      @clear="scope.clearFilters"
    />

    <PageTabs class="audit-subnav" :tabs="tabs" :active-key="activeKey" />

    <RouterView />
  </div>
</template>

<style scoped>
.audit-page {
  max-width: 1560px;
}

</style>
