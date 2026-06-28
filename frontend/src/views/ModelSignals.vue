<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import {
  ArrowRightOutlined,
  BranchesOutlined,
  DashboardOutlined,
  ExperimentOutlined,
  LineChartOutlined,
  TableOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatDateTime,
  formatDisplayNumber,
  formatDuration,
  formatNumber,
  projectDisplay,
  sessionDisplay,
  type ModelSignalAnomalySession,
  type ModelSignalBreakdown,
  type ModelSignals,
  type Overview,
  type Session,
  type UsageBreakdownBucket
} from '../api'
import ModelSignalsTrendChart from '../components/ModelSignalsTrendChart.vue'
import PageHeader from '../components/PageHeader.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { sourceDisplay } from '../presentation/sourceIdentity'
import { useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import { buildUsageAgentOptions, buildUsageModelOptions, buildUsageProjectOptions } from './useUsageScopeOptions'

const ATable = AntTable as unknown as DefineComponent

const router = useRouter()
const resource = useAsyncResource<ModelSignals | null>(null)
const signals = computed(() => resource.data.value)
const loading = resource.loading
const error = resource.error
const optionOverview = ref<Overview | null>(null)
const projectOptionRows = ref<UsageBreakdownBucket[]>([])
const scope = useUsageScopeRoute(() => {
  void load()
})

const { t } = useMessages({
  en: {
    'title': 'Model Signals',
    'subtitle': 'Proxy operational signals for model-heavy sessions; these trends help triage behavior but do not prove model quality',
    'metric.sessions': 'Sessions / Calls',
    'metric.sessionsNote': '{sessions} sessions, {calls} model calls, {avg}/session',
    'metric.throughput': 'Model Throughput',
    'metric.throughputNote': 'Total tokens per model-second',
    'metric.toolFailure': 'Tool Failure Rate',
    'metric.toolFailureNote': '{failed} failed of {total} tool calls',
    'metric.toolDependency': 'Tool Dependency',
    'metric.toolDependencyNote': 'Sessions that relied on tools',
    'metric.reasoningShare': 'Reasoning Share',
    'metric.reasoningShareNote': 'Reasoning tokens as output share',
    'metric.cacheMiss': 'Cache Miss Rate',
    'metric.cacheMissNote': 'Input tokens not served from cache',
    'metric.outputExpansion': 'Output Expansion',
    'metric.outputExpansionNote': 'Output tokens relative to input',
    'breakdown.title': 'Model Breakdown',
    'breakdown.kicker': 'Per-model proxy signals for volume, cache behavior, tool failures, and throughput',
    'anomaly.title': 'Anomaly Sessions',
    'anomaly.kicker': 'Sessions with unusual operational signals for review, not automatic quality judgments',
    'column.model': 'Model',
    'column.sessions': 'Sessions',
    'column.modelCalls': 'Model calls',
    'column.toolCalls': 'Tools',
    'column.failedTools': 'Failed',
    'column.tokens': 'Tokens',
    'column.outputExpansion': 'Output/input',
    'column.reasoning': 'Reasoning',
    'column.cacheMiss': 'Cache miss',
    'column.throughput': 'Tok/s',
    'column.session': 'Session',
    'column.source': 'Source',
    'column.project': 'Project',
    'column.signal': 'Signals',
    'column.started': 'Started',
    'column.wall': 'Model time',
    'empty.loading': 'Loading model signals...',
    'empty.breakdown': 'No model breakdown rows match the current scope',
    'empty.anomalies': 'No anomaly sessions match the current scope',
    'fallback.unknown': 'unknown',
    'error.title': 'Model signals failed to load',
    'action.openSession': 'Open session'
  },
  'zh-CN': {
    'title': '模型信号',
    'subtitle': '面向模型密集会话的代理运营信号；这些趋势用于排查行为，不证明模型质量',
    'metric.sessions': '会话 / 调用',
    'metric.sessionsNote': '{sessions} 个会话，{calls} 次模型调用，平均 {avg}/会话',
    'metric.throughput': '模型吞吐',
    'metric.throughputNote': '每秒模型耗时产生的总 Token',
    'metric.toolFailure': '工具失败率',
    'metric.toolFailureNote': '{total} 次工具调用中 {failed} 次失败',
    'metric.toolDependency': '工具依赖',
    'metric.toolDependencyNote': '依赖工具的会话占比',
    'metric.reasoningShare': '推理占比',
    'metric.reasoningShareNote': '推理 Token 在输出中的占比',
    'metric.cacheMiss': '缓存未命中',
    'metric.cacheMissNote': '未由缓存提供的输入 Token 占比',
    'metric.outputExpansion': '输出膨胀',
    'metric.outputExpansionNote': '输出 Token 相对输入的比例',
    'breakdown.title': '模型拆分',
    'breakdown.kicker': '按模型查看用量、缓存行为、工具失败和吞吐等代理信号',
    'anomaly.title': '异常会话',
    'anomaly.kicker': '列出运营信号异常的会话供复核，不自动判断质量',
    'column.model': '模型',
    'column.sessions': '会话',
    'column.modelCalls': '模型调用',
    'column.toolCalls': '工具',
    'column.failedTools': '失败',
    'column.tokens': 'Token',
    'column.outputExpansion': '输出/输入',
    'column.reasoning': '推理',
    'column.cacheMiss': '缓存未命中',
    'column.throughput': 'Token/秒',
    'column.session': '会话',
    'column.source': '来源',
    'column.project': '项目',
    'column.signal': '信号',
    'column.started': '开始',
    'column.wall': '模型耗时',
    'empty.loading': '正在加载模型信号...',
    'empty.breakdown': '当前范围内没有模型拆分行',
    'empty.anomalies': '当前范围内没有异常会话',
    'fallback.unknown': '未知',
    'error.title': '模型信号加载失败',
    'action.openSession': '打开会话'
  }
})

const hasData = computed(() => Boolean(signals.value?.totalSessions))

const agentOptions = computed(() =>
  buildUsageAgentOptions({
    sources: [
      optionOverview.value?.agentUsage,
      optionOverview.value?.recentSessions,
      optionOverview.value?.slowSessions,
      normalizedAnomalies.value
    ],
    selected: scope.filters.value.agent,
    fallback: t('fallback.unknown')
  })
)

const modelOptions = computed(() =>
  buildUsageModelOptions({
    modelUsage: [
      signals.value?.modelBreakdown,
      optionOverview.value?.modelUsage
    ],
    sessions: [
      normalizedAnomalies.value,
      optionOverview.value?.recentSessions,
      optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.model
  })
)

const projectOptions = computed(() =>
  buildUsageProjectOptions({
    projects: [
      projectOptionRows.value,
      normalizedAnomalies.value,
      optionOverview.value?.recentSessions,
      optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.project,
    fallback: t('fallback.unknown')
  })
)

const metricCards = computed(() => {
  const item = signals.value
  return [
    {
      label: t('metric.sessions'),
      value: displayPair(item?.totalSessions, item?.totalModelCalls),
      note: t('metric.sessionsNote', {
        sessions: formatDisplayNumber(item?.totalSessions).main,
        calls: formatDisplayNumber(item?.totalModelCalls).main,
        avg: formatRate(item?.avgModelCallsPerSession, 1)
      }),
      icon: BranchesOutlined,
      tone: 'metric-primary'
    },
    {
      label: t('metric.throughput'),
      value: displayRate(item?.modelThroughputTokensPerSecond, ' tok/s', 1),
      note: t('metric.throughputNote'),
      icon: DashboardOutlined,
      tone: 'metric-info'
    },
    {
      label: t('metric.toolFailure'),
      value: displayPercent(item?.toolFailureRate),
      note: t('metric.toolFailureNote', {
        failed: formatDisplayNumber(item?.failedToolCalls).main,
        total: formatDisplayNumber(item?.totalToolCalls).main
      }),
      icon: WarningOutlined,
      tone: item?.failedToolCalls ? 'metric-warning' : 'metric-success'
    },
    {
      label: t('metric.toolDependency'),
      value: displayPercent(item?.toolDependencyRate),
      note: t('metric.toolDependencyNote'),
      icon: TableOutlined,
      tone: 'metric-neutral'
    },
    {
      label: t('metric.reasoningShare'),
      value: displayPercent(item?.reasoningTokenShare),
      note: t('metric.reasoningShareNote'),
      icon: ExperimentOutlined,
      tone: 'metric-neutral'
    },
    {
      label: t('metric.cacheMiss'),
      value: displayPercent(item?.cacheMissRate),
      note: t('metric.cacheMissNote'),
      icon: LineChartOutlined,
      tone: 'metric-neutral'
    },
    {
      label: t('metric.outputExpansion'),
      value: displayPercent(item?.outputExpansionRate),
      note: t('metric.outputExpansionNote'),
      icon: BranchesOutlined,
      tone: 'metric-neutral'
    }
  ]
})

const breakdownColumns = computed(() => [
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 210 },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 90, align: 'right' },
  { title: t('column.modelCalls'), dataIndex: 'modelCalls', key: 'modelCalls', width: 110, align: 'right' },
  { title: t('column.toolCalls'), dataIndex: 'toolCalls', key: 'toolCalls', width: 90, align: 'right' },
  { title: t('column.failedTools'), dataIndex: 'failedToolCalls', key: 'failedTools', width: 90, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 110, align: 'right' },
  { title: t('column.outputExpansion'), dataIndex: 'outputExpansionRate', key: 'outputExpansion', width: 110, align: 'right' },
  { title: t('column.reasoning'), dataIndex: 'reasoningTokenShare', key: 'reasoning', width: 100, align: 'right' },
  { title: t('column.cacheMiss'), dataIndex: 'cacheMissRate', key: 'cacheMiss', width: 120, align: 'right' },
  { title: t('column.throughput'), dataIndex: 'modelThroughputTokensPerSecond', key: 'throughput', width: 100, align: 'right' }
])

const anomalyColumns = computed(() => [
  { title: t('column.session'), dataIndex: 'sessionKey', key: 'session', width: 170 },
  { title: t('column.source'), dataIndex: 'sourceLabel', key: 'source', width: 170 },
  { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 220 },
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 170 },
  { title: t('column.signal'), dataIndex: 'reasons', key: 'signal', width: 260 },
  { title: t('column.outputExpansion'), dataIndex: 'outputExpansionRate', key: 'outputExpansion', width: 112, align: 'right' },
  { title: t('column.reasoning'), dataIndex: 'reasoningTokenShare', key: 'reasoning', width: 100, align: 'right' },
  { title: t('column.cacheMiss'), dataIndex: 'cacheMissRate', key: 'cacheMiss', width: 120, align: 'right' },
  { title: t('column.failedTools'), dataIndex: 'failedToolCalls', key: 'failedTools', width: 90, align: 'right' },
  { title: t('column.throughput'), dataIndex: 'modelThroughputTokensPerSecond', key: 'throughput', width: 100, align: 'right' },
  { title: t('column.started'), dataIndex: 'startedAt', key: 'started', width: 136 },
  { title: t('column.wall'), dataIndex: 'modelDurationMs', key: 'duration', width: 98, align: 'right' },
  { title: '', key: 'open', width: 48, align: 'right' }
])

const tableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.breakdown') }))
const anomalyTableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.anomalies') }))
const normalizedAnomalies = computed(() => (signals.value?.anomalySessions || []).map(normalizeAnomaly))

async function load() {
  return resource.run(async () => {
    const filters = scope.apiFilters.value
    const [nextSignals, nextOptionOverview, projectBreakdown] = await Promise.all([
      api.getModelSignals(filters),
      api.getOverview(),
      api.getUsageBreakdown({ groupBy: 'project' }).catch(() => null)
    ])
    optionOverview.value = nextOptionOverview
    projectOptionRows.value = projectBreakdown?.buckets || []
    return nextSignals
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

function displayPair(left?: number, right?: number) {
  const leftDisplay = formatDisplayNumber(left)
  const rightDisplay = formatDisplayNumber(right)
  return {
    main: `${leftDisplay.main} / ${rightDisplay.main}`,
    full: `${leftDisplay.full} / ${rightDisplay.full}`
  }
}

function displayPercent(value?: number) {
  const text = formatPercent(value)
  return { main: text, full: text }
}

function displayRate(value?: number, suffix = '', digits = 0) {
  const text = `${formatRate(value, digits)}${suffix}`
  return { main: text, full: text }
}

function formatPercent(value?: number) {
  if (!Number.isFinite(value)) return '0%'
  const percent = Math.max(0, value || 0) * 100
  if (percent > 0 && percent < 1) return '<1%'
  return `${percent.toLocaleString(undefined, { maximumFractionDigits: percent >= 10 ? 0 : 1 })}%`
}

function formatRate(value?: number, digits = 0) {
  if (!Number.isFinite(value)) return '0'
  return (value || 0).toLocaleString(undefined, { maximumFractionDigits: digits })
}

function rowClass(record: ModelSignalBreakdown) {
  return { class: record.failedToolCalls > 0 ? 'model-signals-warning-row' : '' }
}

function anomalyRowClass(record: NormalizedAnomalySession) {
  return { class: record.failedToolCalls > 0 || record.severity === 'high' ? 'model-signals-warning-row' : '' }
}

function anomalyReasons(row: ModelSignalAnomalySession): string[] {
  const candidates = [row.reasons, row.reasonLabels, row.signalReasons, row.reason, row.signal]
  const values = candidates.flatMap((value) => {
    if (Array.isArray(value)) return value
    if (typeof value === 'string') return [value]
    return []
  })
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))]
}

interface NormalizedAnomalySession {
  id: number
  sessionKey?: string
  codexSessionId?: string
  startedAt?: string
  projectPath?: string
  rawSourcePath?: string
  agentKind?: string
  agentName?: string
  sourceId?: number
  sourceKey?: string
  sourceLabel?: string
  sourceRootPath?: string
  sourceSessionsPath?: string
  model?: string
  totalTokens: number
  outputExpansionRate: number
  reasoningTokenShare: number
  cacheMissRate: number
  modelThroughputTokensPerSecond: number
  failedToolCalls: number
  modelDurationMs: number
  severity?: string
  reasons: string[]
}

function normalizeAnomaly(row: ModelSignalAnomalySession): NormalizedAnomalySession {
  const session = row.session || ({} as Partial<Session>)
  return {
    id: numberField(row, ['id', 'sessionId']) || session.id || 0,
    sessionKey: stringField(row, ['sessionKey']) || session.sessionKey,
    codexSessionId: stringField(row, ['codexSessionId']) || session.codexSessionId,
    startedAt: stringField(row, ['startedAt']) || session.startedAt,
    projectPath: stringField(row, ['projectPath']) || session.projectPath,
    rawSourcePath: stringField(row, ['rawSourcePath']) || session.rawSourcePath,
    agentKind: stringField(row, ['agentKind']) || session.agentKind,
    agentName: stringField(row, ['agentName']) || session.agentName,
    sourceId: numberField(row, ['sourceId']) || session.sourceId,
    sourceKey: stringField(row, ['sourceKey']) || session.sourceKey,
    sourceLabel: stringField(row, ['sourceLabel']) || session.sourceLabel,
    sourceRootPath: stringField(row, ['sourceRootPath']) || session.sourceRootPath,
    sourceSessionsPath: stringField(row, ['sourceSessionsPath']) || session.sourceSessionsPath,
    model: stringField(row, ['model']) || session.model,
    totalTokens: numberField(row, ['totalTokens']) || session.tokenUsage?.totalTokens || 0,
    outputExpansionRate: numberField(row, ['outputExpansionRate']),
    reasoningTokenShare: numberField(row, ['reasoningTokenShare']),
    cacheMissRate: numberField(row, ['cacheMissRate']),
    modelThroughputTokensPerSecond: numberField(row, ['modelThroughputTokensPerSecond']),
    failedToolCalls: numberField(row, ['failedToolCalls']),
    modelDurationMs: numberField(row, ['modelDurationMs']) || session.modelDurationMs || 0,
    severity: stringField(row, ['severity']),
    reasons: anomalyReasons(row)
  }
}

function stringField(row: ModelSignalAnomalySession, keys: string[]) {
  for (const key of keys) {
    const value = row[key]
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return undefined
}

function numberField(row: ModelSignalAnomalySession, keys: string[]) {
  for (const key of keys) {
    const value = row[key]
    if (typeof value === 'number' && Number.isFinite(value)) return value
  }
  return 0
}

function sourceInfo(record: NormalizedAnomalySession) {
  return sourceDisplay(record, t('fallback.unknown'))
}

function projectInfo(record: NormalizedAnomalySession) {
  return projectDisplay(record.projectPath || record.rawSourcePath)
}

function sessionInfo(record: NormalizedAnomalySession) {
  return sessionDisplay({
    id: record.id,
    sessionKey: record.sessionKey || '',
    codexSessionId: record.codexSessionId
  })
}

function anomalyRowKey(record: NormalizedAnomalySession) {
  return record.id || record.sessionKey || record.codexSessionId || `${record.model}:${record.startedAt}`
}

function openSession(id: number) {
  if (id) router.push(`/sessions/${id}`)
}

onMounted(load)
</script>

<template>
  <div class="page model-signals-page">
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

    <a-alert
      v-if="error"
      class="model-signals-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <a-spin :spinning="loading && !signals">
      <div class="section-stack">
        <section class="metric-strip model-signals-metric-strip" :class="{ 'is-empty': !hasData }">
          <div v-for="item in metricCards" :key="item.label" class="metric-strip-item" :class="item.tone">
            <div class="metric-strip-head">
              <span class="metric-label">{{ item.label }}</span>
              <span class="metric-strip-icon">
                <component :is="item.icon" />
              </span>
            </div>
            <div class="metric-strip-value" :title="item.value.full">{{ item.value.main }}</div>
            <div class="metric-strip-note">{{ item.note }}</div>
          </div>
        </section>

        <ModelSignalsTrendChart :points="signals?.trend || []" :loading="loading" />

        <section class="panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('breakdown.title') }}</h2>
              <div class="panel-kicker">{{ t('breakdown.kicker') }}</div>
            </div>
            <TableOutlined class="panel-header-icon" />
          </div>
          <a-table
            class="dense-table model-signals-breakdown-table"
            :columns="breakdownColumns"
            :data-source="signals?.modelBreakdown || []"
            :loading="loading"
            :locale="tableLocale"
            :pagination="{ pageSize: 10 }"
            row-key="model"
            size="small"
            :custom-row="rowClass"
            :scroll="{ x: 1140 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'model'">
                <span class="model-name">{{ record.model || t('fallback.unknown') }}</span>
              </template>
              <template v-else-if="column.key === 'sessions'"><span class="number-cell">{{ formatNumber(record.sessionCount) }}</span></template>
              <template v-else-if="column.key === 'modelCalls'"><span class="number-cell">{{ formatNumber(record.modelCalls) }}</span></template>
              <template v-else-if="column.key === 'toolCalls'"><span class="number-cell">{{ formatNumber(record.toolCalls) }}</span></template>
              <template v-else-if="column.key === 'failedTools'">
                <span class="number-cell" :class="{ 'status-error': record.failedToolCalls > 0 }">{{ formatNumber(record.failedToolCalls) }}</span>
              </template>
              <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.totalTokens) }}</span></template>
              <template v-else-if="column.key === 'outputExpansion'"><span class="number-cell">{{ formatPercent(record.outputExpansionRate) }}</span></template>
              <template v-else-if="column.key === 'reasoning'"><span class="number-cell">{{ formatPercent(record.reasoningTokenShare) }}</span></template>
              <template v-else-if="column.key === 'cacheMiss'"><span class="number-cell">{{ formatPercent(record.cacheMissRate) }}</span></template>
              <template v-else-if="column.key === 'throughput'"><span class="number-cell">{{ formatRate(record.modelThroughputTokensPerSecond, 1) }}</span></template>
            </template>
          </a-table>
        </section>

        <section class="panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('anomaly.title') }}</h2>
              <div class="panel-kicker">{{ t('anomaly.kicker') }}</div>
            </div>
            <WarningOutlined class="panel-header-icon" />
          </div>
          <a-table
            class="dense-table model-signals-anomaly-table"
            :columns="anomalyColumns"
            :data-source="normalizedAnomalies"
            :loading="loading"
            :locale="anomalyTableLocale"
            :pagination="{ pageSize: 8 }"
            :row-key="anomalyRowKey"
            size="small"
            :custom-row="anomalyRowClass"
            :scroll="{ x: 1600 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'session'">
                <span class="mono model-signals-session" :title="sessionInfo(record).full">{{ sessionInfo(record).main }}</span>
                <div class="source-identity-meta">{{ formatNumber(record.totalTokens) }} {{ t('column.tokens') }}</div>
              </template>
              <template v-else-if="column.key === 'source'">
                <span class="source-identity-name">{{ sourceInfo(record).label }}</span>
                <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
              </template>
              <template v-else-if="column.key === 'project'">
                <a-tooltip :title="projectInfo(record).full" placement="topLeft">
                  <span class="model-signals-project">{{ projectInfo(record).main }}</span>
                </a-tooltip>
              </template>
              <template v-else-if="column.key === 'model'">
                <span class="model-name">{{ record.model || t('fallback.unknown') }}</span>
              </template>
              <template v-else-if="column.key === 'signal'">
                <div class="model-signals-tags">
                  <a-tag v-for="reason in record.reasons" :key="reason" :color="record.severity === 'high' ? 'warning' : 'processing'">
                    {{ reason }}
                  </a-tag>
                </div>
              </template>
              <template v-else-if="column.key === 'outputExpansion'"><span class="number-cell">{{ formatPercent(record.outputExpansionRate) }}</span></template>
              <template v-else-if="column.key === 'reasoning'"><span class="number-cell">{{ formatPercent(record.reasoningTokenShare) }}</span></template>
              <template v-else-if="column.key === 'cacheMiss'"><span class="number-cell">{{ formatPercent(record.cacheMissRate) }}</span></template>
              <template v-else-if="column.key === 'failedTools'">
                <span class="number-cell" :class="{ 'status-error': record.failedToolCalls > 0 }">{{ formatNumber(record.failedToolCalls) }}</span>
              </template>
              <template v-else-if="column.key === 'throughput'"><span class="number-cell">{{ formatRate(record.modelThroughputTokensPerSecond, 1) }}</span></template>
              <template v-else-if="column.key === 'started'">{{ formatDateTime(record.startedAt) }}</template>
              <template v-else-if="column.key === 'duration'"><span class="number-cell">{{ formatDuration(record.modelDurationMs) }}</span></template>
              <template v-else-if="column.key === 'open'">
                <a-tooltip :title="t('action.openSession')">
                  <a-button type="text" size="small" :disabled="!record.id" @click="openSession(record.id)">
                    <template #icon>
                      <ArrowRightOutlined />
                    </template>
                  </a-button>
                </a-tooltip>
              </template>
            </template>
          </a-table>
        </section>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.model-signals-page {
  max-width: 1560px;
}

.model-signals-error {
  margin-bottom: var(--am-section-gap);
}

.model-signals-metric-strip {
  grid-template-columns: repeat(7, minmax(136px, 1fr));
}

.model-signals-session,
.model-signals-project {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: var(--am-text);
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.model-signals-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.model-signals-tags .ant-tag {
  margin-right: 0;
}

:deep(.model-signals-warning-row td) {
  background: rgba(254, 243, 199, 0.36);
}

@media (max-width: 1420px) {
  .model-signals-metric-strip {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .model-signals-metric-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .model-signals-metric-strip {
    grid-template-columns: 1fr;
  }
}
</style>
