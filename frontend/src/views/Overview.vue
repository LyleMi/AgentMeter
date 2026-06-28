<script setup lang="ts">
import { computed, onMounted, provide, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import message from 'ant-design-vue/es/message'
import {
  BarChartOutlined,
  ClockCircleOutlined,
  DatabaseOutlined,
  HistoryOutlined
} from '@ant-design/icons-vue'
import { api, isStaticDemo, type Overview, type Settings } from '../api'
import PageHeader from '../components/PageHeader.vue'
import PageTabs from '../components/PageTabs.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import { notifyAppDataChanged } from '../events'
import { useMessages } from '../i18n'
import { overviewContextKey, type OverviewContext } from './overviewContext'
import { applyUsageScopeToQuery, useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import {
  buildUsageAgentOptions,
  buildUsageModelOptions,
  buildUsageProjectOptions,
  useUsageScopeOptionData
} from './useUsageScopeOptions'

const route = useRoute()
const loading = ref(true)
const startupIndexing = ref(false)
const overview = ref<Overview | null>(null)
const settings = ref<Settings | null>(null)
const scope = useUsageScopeRoute(() => load())
const scopeOptionData = useUsageScopeOptionData()
const { t } = useMessages({
  en: {
    'title': 'Overview',
    'subtitle': 'Indexed coding-agent usage across local JSONL sessions',
    'tab.summary': 'Summary',
    'tab.trends': 'Trends',
    'tab.breakdown': 'Breakdown',
    'tab.recent': 'Recent',
    'message.indexed': '{indexed} indexed, {skipped} skipped, {failed} failed',
    'message.indexFailed': 'Index failed',
    'message.demoReadOnly': 'Static demo mode is read-only.'
  },
  'zh-CN': {
    'title': '概览',
    'subtitle': '基于本地 JSONL 会话索引的编码代理用量',
    'tab.summary': '汇总',
    'tab.trends': '趋势',
    'tab.breakdown': '拆分',
    'tab.recent': '最近',
    'message.indexed': '已索引 {indexed}，已跳过 {skipped}，失败 {failed}',
    'message.indexFailed': '索引失败',
    'message.demoReadOnly': '静态演示模式为只读。'
  }
})

const hasIndexedData = computed(() => (overview.value?.totalSessions || 0) > 0)
const sourcePathDisplay = computed(() => settings.value?.sourcePath || settings.value?.defaultSourcePath || '')

const tabs = computed(() => [
  { key: 'summary', label: t('tab.summary'), path: overviewPath('/overview/summary'), icon: BarChartOutlined },
  { key: 'trends', label: t('tab.trends'), path: overviewPath('/overview/trends'), icon: ClockCircleOutlined },
  { key: 'breakdown', label: t('tab.breakdown'), path: overviewPath('/overview/breakdown'), icon: DatabaseOutlined },
  { key: 'recent', label: t('tab.recent'), path: overviewPath('/overview/recent'), icon: HistoryOutlined }
])

const agentOptions = computed(() =>
  buildUsageAgentOptions({
    sources: [
      overview.value?.agentUsage,
      scopeOptionData.optionOverview.value?.agentUsage,
      overview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions,
      overview.value?.slowSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.agent,
    fallback: 'unknown'
  })
)

const modelOptions = computed(() =>
  buildUsageModelOptions({
    modelUsage: [
      overview.value?.modelUsage,
      scopeOptionData.optionOverview.value?.modelUsage
    ],
    sessions: [
      overview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions,
      overview.value?.slowSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.model
  })
)

const projectOptions = computed(() =>
  buildUsageProjectOptions({
    projects: [
      scopeOptionData.projectOptionRows.value,
      overview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions,
      overview.value?.slowSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.project,
    fallback: 'unknown'
  })
)

const activeKey = computed(() => {
  if (route.path.startsWith('/overview/trends')) return 'trends'
  if (route.path.startsWith('/overview/breakdown')) return 'breakdown'
  if (route.path.startsWith('/overview/recent')) return 'recent'
  return 'summary'
})

async function load() {
  loading.value = true
  try {
    const [settingsValue, overviewValue, optionData] = await Promise.all([
      api.getSettings(),
      api.getOverview(scope.apiFilters.value),
      scopeOptionData.loadUsageScopeOptionData({ includeOverview: scope.hasActiveFilters.value })
    ])
    settings.value = settingsValue
    overview.value = overviewValue
    scopeOptionData.applyUsageScopeOptionData(optionData, overviewValue)
  } finally {
    loading.value = false
  }
}

async function indexFromOverview() {
  if (isStaticDemo) {
    message.info(t('message.demoReadOnly'))
    return
  }
  startupIndexing.value = true
  try {
    const result = await api.indexNow(false)
    message.success(
      t('message.indexed', {
        indexed: result.indexed,
        skipped: result.skipped,
        failed: result.failed
      })
    )
    await load()
    notifyAppDataChanged('index')
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('message.indexFailed'))
  } finally {
    startupIndexing.value = false
  }
}

async function updateScopeFilters(nextFilters: UsageScopeForm) {
  await scope.updateFilters(nextFilters)
  await load()
}

async function clearScopeFilters() {
  await scope.clearFilters()
  await load()
}

function overviewPath(path: string) {
  const query = applyUsageScopeToQuery(route.query, scope.filters.value)
  const params = new URLSearchParams(query)
  const encoded = params.toString()
  return encoded ? `${path}?${encoded}` : path
}

const context: OverviewContext = {
  overview,
  settings,
  loading,
  startupIndexing,
  hasIndexedData,
  sourcePathDisplay,
  load,
  indexFromOverview
}

provide(overviewContextKey, context)

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')" />

    <UsageScopeBar
      :filters="scope.filters.value"
      :agent-options="agentOptions"
      :model-options="modelOptions"
      :project-options="projectOptions"
      :loading="loading"
      @update:filters="updateScopeFilters"
      @refresh="load"
      @clear="clearScopeFilters"
    />

    <PageTabs class="overview-subnav" :tabs="tabs" :active-key="activeKey" />

    <RouterView />
  </div>
</template>
