<script setup lang="ts">
import { computed, onMounted, provide, type Component } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import {
  BarChartOutlined,
  ClockCircleOutlined,
  FieldTimeOutlined,
  ProfileOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatDuration,
  formatNumber,
  formatPercent,
  sessionLabel,
  type Overview
} from '../api'
import PageHeader from '../components/PageHeader.vue'
import PageTabs from '../components/PageTabs.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { routePathWithQuery } from './routeQuery'
import { routeTabKey } from './routeTabs'
import { applyUsageScopeToQuery, useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import {
  buildUsageAgentOptions,
  buildUsageModelOptions,
  buildUsageProjectOptions,
  useUsageScopeOptionData
} from './useUsageScopeOptions'
import { timeContextKey, type TimeContext, type TimeKpiCard, type TimeSegment } from './time/timeContext'

const timeTabMatches = [
  { key: 'sources', pathPrefix: '/time/sources' },
  { key: 'tools', pathPrefix: '/time/tools' },
  { key: 'sessions', pathPrefix: '/time/sessions' }
] as const

const route = useRoute()
const resource = useAsyncResource<Overview | null>(null)
const overview = computed(() => resource.data.value)
const loading = resource.loading
const error = resource.error
const scope = useUsageScopeRoute(() => {
  void load()
})
const scopeOptionData = useUsageScopeOptionData()
let loadRequestId = 0

const { t } = useMessages({
  en: {
    'title': 'Time',
    'subtitle': 'Wall-time attribution across model, tool, source, and slow session activity',
    'action.refresh': 'Refresh',
    'tab.summary': 'Summary',
    'tab.sources': 'Sources',
    'tab.tools': 'Tools',
    'tab.sessions': 'Slow Sessions',
    'composition.model': 'Model',
    'composition.network': 'Suspected network tools',
    'composition.tools': 'Other tools',
    'composition.idle': 'Idle / unclassified',
    'kpi.wall': 'Wall Time',
    'kpi.wallNote': '{count} indexed sessions',
    'kpi.activeShare': 'Active Share',
    'kpi.activeShareNote': '{duration} measured model and tool time',
    'kpi.toolShare': 'Tool Share',
    'kpi.toolShareNote': '{duration} total tool time',
    'kpi.networkShare': 'Network-Likely Share',
    'kpi.networkShareNote': '{duration} across {count} calls',
    'kpi.slowest': 'Slowest Session',
    'kpi.slowestNote': '{duration} wall time',
    'empty.sessions': 'No slow sessions yet',
    'empty.title': 'No time analysis yet',
    'empty.text': 'Time attribution appears after sessions with model, tool, and wall durations are indexed.',
    'fallback.unknown': 'unknown',
    'error.title': 'Time analytics failed to load'
  },
  'zh-CN': {
    'title': '耗时',
    'subtitle': '按模型、工具、来源和慢会话归因墙钟耗时',
    'action.refresh': '刷新',
    'tab.summary': '汇总',
    'tab.sources': '来源对比',
    'tab.tools': '工具耗时',
    'tab.sessions': '慢会话',
    'composition.model': '模型',
    'composition.network': '疑似网络工具',
    'composition.tools': '其他工具',
    'composition.idle': '空闲 / 未分类',
    'kpi.wall': '墙钟时间',
    'kpi.wallNote': '{count} 个已索引会话',
    'kpi.activeShare': '活跃占比',
    'kpi.activeShareNote': '{duration} 已测量模型和工具时间',
    'kpi.toolShare': '工具占比',
    'kpi.toolShareNote': '{duration} 工具总耗时',
    'kpi.networkShare': '疑似网络占比',
    'kpi.networkShareNote': '{duration}，共 {count} 次调用',
    'kpi.slowest': '最慢会话',
    'kpi.slowestNote': '墙钟时间 {duration}',
    'empty.sessions': '暂无慢会话',
    'empty.title': '暂无时间分析',
    'empty.text': '索引包含模型、工具和墙钟时长的会话后，会显示时间归因。',
    'fallback.unknown': '未知',
    'error.title': '耗时分析加载失败'
  }
})

const tabs = computed(() => [
  { key: 'summary', label: t('tab.summary'), path: timePath('/time'), icon: FieldTimeOutlined },
  { key: 'sources', label: t('tab.sources'), path: timePath('/time/sources'), icon: BarChartOutlined },
  { key: 'tools', label: t('tab.tools'), path: timePath('/time/tools'), icon: ToolOutlined },
  { key: 'sessions', label: t('tab.sessions'), path: timePath('/time/sessions'), icon: ProfileOutlined }
])

const activeKey = computed(() => routeTabKey(route.path, timeTabMatches, 'summary'))

const hasIndexedData = computed(() => (overview.value?.totalSessions || 0) > 0)
const wallDurationMs = computed(() => Math.max(0, overview.value?.totalWallDurationMs || 0))
const activeDurationMs = computed(() => Math.max(0, overview.value?.totalActiveDurationMs || 0))
const toolDurationMs = computed(() => Math.max(0, overview.value?.totalToolDurationMs || 0))
const suspectedNetworkDurationMs = computed(() =>
  Math.max(0, Math.min(overview.value?.suspectedNetworkToolDurationMs || 0, toolDurationMs.value))
)
const slowSessions = computed(() => overview.value?.slowSessions || [])
const slowestSession = computed(() => slowSessions.value[0])

const compositionSegments = computed<TimeSegment[]>(() => {
  const wall = wallDurationMs.value
  let remaining = wall
  const modelDuration = takeDuration(overview.value?.totalModelDurationMs || 0, remaining)
  remaining -= modelDuration
  const networkDuration = takeDuration(suspectedNetworkDurationMs.value, remaining)
  remaining -= networkDuration
  const otherToolDuration = takeDuration(Math.max(0, toolDurationMs.value - suspectedNetworkDurationMs.value), remaining)
  remaining -= otherToolDuration
  const idleDuration = Math.max(0, remaining)

  return [
    buildSegment('model', t('composition.model'), modelDuration, wall, 'is-model'),
    buildSegment('network', t('composition.network'), networkDuration, wall, 'is-network'),
    buildSegment('tools', t('composition.tools'), otherToolDuration, wall, 'is-tools'),
    buildSegment('idle', t('composition.idle'), idleDuration, wall, 'is-idle')
  ]
})

const kpiCards = computed<TimeKpiCard[]>(() => [
  {
    label: t('kpi.wall'),
    value: formatDuration(wallDurationMs.value),
    note: t('kpi.wallNote', { count: formatNumber(overview.value?.totalSessions) }),
    icon: FieldTimeOutlined as Component
  },
  {
    label: t('kpi.activeShare'),
    value: formatTimePercent(activeDurationMs.value / Math.max(wallDurationMs.value, 1)),
    note: t('kpi.activeShareNote', { duration: formatDuration(activeDurationMs.value) }),
    icon: ClockCircleOutlined as Component
  },
  {
    label: t('kpi.toolShare'),
    value: formatTimePercent(toolDurationMs.value / Math.max(wallDurationMs.value, 1)),
    note: t('kpi.toolShareNote', { duration: formatDuration(toolDurationMs.value) }),
    icon: ToolOutlined as Component
  },
  {
    label: t('kpi.networkShare'),
    value: formatTimePercent(suspectedNetworkDurationMs.value / Math.max(wallDurationMs.value, 1)),
    note: t('kpi.networkShareNote', {
      duration: formatDuration(suspectedNetworkDurationMs.value),
      count: formatNumber(overview.value?.suspectedNetworkToolCalls)
    }),
    icon: ToolOutlined as Component
  },
  {
    label: t('kpi.slowest'),
    value: slowestSession.value ? sessionLabel(slowestSession.value) : '-',
    note: slowestSession.value ? t('kpi.slowestNote', { duration: formatDuration(slowestSession.value.wallDurationMs) }) : t('empty.sessions'),
    icon: ClockCircleOutlined as Component
  }
])

const rankedToolLeaders = computed(() =>
  [...(overview.value?.toolTimeLeaders || [])].sort((left, right) => right.totalDurationMs - left.totalDurationMs)
)
const rankedAgentTimeUsage = computed(() =>
  [...(overview.value?.agentTimeUsage || [])].sort((left, right) => right.wallDurationMs - left.wallDurationMs)
)
const rankedModelTimeUsage = computed(() =>
  [...(overview.value?.modelTimeUsage || [])].sort((left, right) => right.wallDurationMs - left.wallDurationMs)
)

const agentOptions = computed(() =>
  buildUsageAgentOptions({
    sources: [
      overview.value?.agentTimeUsage,
      overview.value?.agentUsage,
      scopeOptionData.optionOverview.value?.agentTimeUsage,
      scopeOptionData.optionOverview.value?.agentUsage,
      overview.value?.slowSessions,
      scopeOptionData.optionOverview.value?.slowSessions,
      overview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions
    ],
    selected: scope.filters.value.agent,
    fallback: t('fallback.unknown')
  })
)

const modelOptions = computed(() =>
  buildUsageModelOptions({
    modelUsage: [
      overview.value?.modelUsage,
      scopeOptionData.optionOverview.value?.modelUsage,
      overview.value?.modelTimeUsage,
      scopeOptionData.optionOverview.value?.modelTimeUsage
    ],
    sessions: [
      overview.value?.slowSessions,
      scopeOptionData.optionOverview.value?.slowSessions,
      overview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions
    ],
    selected: scope.filters.value.model
  })
)

const projectOptions = computed(() =>
  buildUsageProjectOptions({
    projects: [
      scopeOptionData.projectOptionRows.value,
      overview.value?.slowSessions,
      scopeOptionData.optionOverview.value?.slowSessions,
      overview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.recentSessions
    ],
    selected: scope.filters.value.project,
    fallback: t('fallback.unknown')
  })
)

function timePath(path: string) {
  return routePathWithQuery(path, applyUsageScopeToQuery(route.query, scope.filters.value))
}

function takeDuration(value: number, remaining: number) {
  return Math.min(Math.max(0, value), Math.max(0, remaining))
}

function buildSegment(key: string, label: string, value: number, wall: number, tone: string): TimeSegment {
  const share = wall > 0 ? value / wall : 0
  return {
    key,
    label,
    value,
    share,
    width: share > 0 ? `${Math.max(1, share * 100)}%` : '0%',
    tone
  }
}

function formatTimePercent(value: number) {
  return formatPercent(value, { lessThanOne: true })
}

function load() {
  const requestId = ++loadRequestId
  return resource.run(async () => {
    const [nextOverview, optionData] = await Promise.all([
      api.getOverview(scope.apiFilters.value),
      scopeOptionData.loadUsageScopeOptionData({ includeOverview: scope.hasActiveFilters.value })
    ])
    if (requestId === loadRequestId) {
      scopeOptionData.applyUsageScopeOptionData(optionData, nextOverview)
    }
    return nextOverview
  }, { onErrorData: null })
}

async function updateScopeFilters(nextFilters: UsageScopeForm) {
  await scope.updateFilters(nextFilters)
  await load()
}

async function clearScopeFilters() {
  await scope.clearFilters()
  await load()
}

const context: TimeContext = {
  overview,
  optionOverview: scopeOptionData.optionOverview,
  loading,
  error,
  hasIndexedData,
  wallDurationMs,
  activeDurationMs,
  toolDurationMs,
  suspectedNetworkDurationMs,
  slowSessions,
  compositionSegments,
  kpiCards,
  rankedToolLeaders,
  rankedAgentTimeUsage,
  rankedModelTimeUsage,
  agentOptions,
  modelOptions,
  projectOptions,
  formatPercent: formatTimePercent,
  load,
  updateScopeFilters,
  clearScopeFilters
}

provide(timeContextKey, context)

onMounted(load)
</script>

<template>
  <div class="page time-page">
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

    <PageTabs class="time-subnav" :tabs="tabs" :active-key="activeKey" />

    <a-alert
      v-if="error"
      class="time-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <RouterView />
  </div>
</template>

<style scoped>
.time-page {
  max-width: 1560px;
}

.time-error {
  margin-bottom: var(--am-section-gap);
}

.time-subnav {
  margin-bottom: var(--am-section-gap);
}
</style>

