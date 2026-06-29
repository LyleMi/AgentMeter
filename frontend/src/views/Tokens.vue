<script setup lang="ts">
import { computed, onMounted, provide, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import {
  BarChartOutlined,
  DatabaseOutlined,
  HistoryOutlined,
  LineChartOutlined
} from '@ant-design/icons-vue'
import {
  api,
  type TokenAnalytics,
  type UsageBreakdownBucket,
  type UsageBreakdownGroupBy,
  type UsageScopeFilters
} from '../api'
import PageHeader from '../components/PageHeader.vue'
import PageTabs from '../components/PageTabs.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { applyUsageScopeToQuery, useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import {
  buildUsageAgentOptions,
  buildUsageModelOptions,
  buildUsageProjectOptions,
  useUsageScopeOptionData
} from './useUsageScopeOptions'
import {
  DEFAULT_BREAKDOWN_GROUP,
  tokensContextKey,
  type TokenBreakdownGroup,
  type TokensContext
} from './tokens/tokensContext'

const route = useRoute()
const router = useRouter()
const resource = useAsyncResource<TokenAnalytics | null>(null)
const analytics = computed(() => resource.data.value)
const loading = resource.loading
const error = resource.error
const breakdownRows = ref<UsageBreakdownBucket[]>([])
const scope = useUsageScopeRoute(() => {
  void load()
})
const scopeOptionData = useUsageScopeOptionData()
const breakdownGroup = ref<TokenBreakdownGroup>(routeBreakdownGroup())
let applyingBreakdownRouteUpdate = false
let loadRequestId = 0

const { t } = useMessages({
  en: {
    'title': 'Tokens',
    'subtitle': 'Token usage, cache reuse, trend volatility, and estimated price consumption',
    'tab.summary': 'Summary',
    'tab.trends': 'Trends',
    'tab.breakdown': 'Breakdown',
    'tab.sessions': 'Sessions',
    'fallback.unknown': 'unknown',
    'error.title': 'Token analytics failed to load'
  },
  'zh-CN': {
    'title': 'Token',
    'subtitle': '查看 Token 用量、缓存复用、趋势波动和预估价格消耗',
    'tab.summary': '汇总',
    'tab.trends': '趋势',
    'tab.breakdown': '拆分',
    'tab.sessions': '会话',
    'fallback.unknown': '未知',
    'error.title': 'Token 分析加载失败'
  }
})

const tabs = computed(() => [
  { key: 'summary', label: t('tab.summary'), path: tokenPath('/tokens'), icon: DatabaseOutlined },
  { key: 'trends', label: t('tab.trends'), path: tokenPath('/tokens/trends'), icon: LineChartOutlined },
  { key: 'breakdown', label: t('tab.breakdown'), path: tokenPath('/tokens/breakdown', true), icon: BarChartOutlined },
  { key: 'sessions', label: t('tab.sessions'), path: tokenPath('/tokens/sessions'), icon: HistoryOutlined }
])

const activeKey = computed(() => {
  if (route.path.startsWith('/tokens/trends')) return 'trends'
  if (route.path.startsWith('/tokens/breakdown')) return 'breakdown'
  if (route.path.startsWith('/tokens/sessions')) return 'sessions'
  return 'summary'
})

const agentOptions = computed(() =>
  buildUsageAgentOptions({
    sources: [
      analytics.value?.agentUsage,
      scopeOptionData.optionOverview.value?.agentUsage,
      analytics.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions,
      analytics.value?.highTokenSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.agent,
    fallback: t('fallback.unknown')
  })
)

const modelOptions = computed(() =>
  buildUsageModelOptions({
    modelUsage: [
      analytics.value?.modelUsage,
      scopeOptionData.optionOverview.value?.modelUsage
    ],
    sessions: [
      analytics.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions,
      analytics.value?.highTokenSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.model
  })
)

const projectOptions = computed(() =>
  buildUsageProjectOptions({
    projects: [
      scopeOptionData.projectOptionRows.value,
      analytics.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions,
      analytics.value?.highTokenSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.project,
    fallback: t('fallback.unknown')
  })
)

function tokenPath(path: string, keepBreakdownGroup = false) {
  const query = applyUsageScopeToQuery(route.query, scope.filters.value, {
    groupBy: keepBreakdownGroup && breakdownGroup.value !== DEFAULT_BREAKDOWN_GROUP ? breakdownGroup.value : undefined
  })
  const params = new URLSearchParams(query)
  const encoded = params.toString()
  return encoded ? `${path}?${encoded}` : path
}

function load() {
  const requestId = ++loadRequestId
  return resource.run(async () => {
    const filters = scope.apiFilters.value
    const [nextAnalytics, optionData] = await Promise.all([
      api.getTokenAnalytics(filters),
      scopeOptionData.loadUsageScopeOptionData()
    ])
    const nextBreakdownRows = await loadBreakdownRows(nextAnalytics, filters)
    if (requestId === loadRequestId) {
      scopeOptionData.applyUsageScopeOptionData(optionData)
      breakdownRows.value = nextBreakdownRows
    }
    return nextAnalytics
  }, { onErrorData: null })
}

async function loadBreakdownRows(item: TokenAnalytics, filters: UsageScopeFilters) {
  if (breakdownGroup.value === DEFAULT_BREAKDOWN_GROUP) return [globalBreakdownRow(item)]
  const breakdown = await api.getUsageBreakdown({
    ...filters,
    groupBy: breakdownGroup.value
  })
  return breakdown.buckets || []
}

function globalBreakdownRow(item: TokenAnalytics): UsageBreakdownBucket {
  return {
    sessionCount: item.totalSessions,
    totalTokens: item.totalTokens,
    inputTokens: item.totalInputTokens,
    cachedInputTokens: item.totalCachedInputTokens,
    outputTokens: item.totalOutputTokens,
    reasoningOutputTokens: item.totalReasoningTokens,
    contextCompressionTokens: item.totalContextCompressionTokens,
    cacheUtilizationRate: item.cacheUtilizationRate,
    estimatedCostUsd: item.estimatedCostUsd,
    unpriced: item.unpricedCount > 0
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

async function updateBreakdownGroup(value: unknown) {
  const nextGroup = normalizeBreakdownGroup(value)
  if (nextGroup === breakdownGroup.value) return
  breakdownGroup.value = nextGroup
  applyingBreakdownRouteUpdate = true
  try {
    await router.replace({
      path: route.path,
      query: applyUsageScopeToQuery(route.query, scope.filters.value, {
        groupBy: nextGroup === DEFAULT_BREAKDOWN_GROUP ? undefined : nextGroup
      })
    })
  } finally {
    applyingBreakdownRouteUpdate = false
  }
  await load()
}

function routeBreakdownGroup(): TokenBreakdownGroup {
  return normalizeBreakdownGroup(route.query.groupBy)
}

function normalizeBreakdownGroup(value: unknown): TokenBreakdownGroup {
  if (value === 'agent' || value === 'model' || value === 'agent,model' || value === 'day' || value === 'project') {
    return value as UsageBreakdownGroupBy
  }
  return DEFAULT_BREAKDOWN_GROUP
}

watch(
  () => route.query.groupBy,
  () => {
    if (applyingBreakdownRouteUpdate) return
    breakdownGroup.value = routeBreakdownGroup()
    void load()
  }
)

const context: TokensContext = {
  analytics,
  optionOverview: scopeOptionData.optionOverview,
  loading,
  error,
  breakdownRows,
  breakdownGroup,
  agentOptions,
  modelOptions,
  projectOptions,
  load,
  updateScopeFilters,
  clearScopeFilters,
  updateBreakdownGroup
}

provide(tokensContextKey, context)

onMounted(load)
</script>

<template>
  <div class="page tokens-page">
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

    <PageTabs class="tokens-subnav" :tabs="tabs" :active-key="activeKey" />

    <a-alert
      v-if="error"
      class="tokens-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <RouterView />
  </div>
</template>

<style scoped>
.tokens-page {
  max-width: 1560px;
}

.tokens-error {
  margin-bottom: var(--am-section-gap);
}

.tokens-subnav {
  margin-bottom: var(--am-section-gap);
}
</style>
