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
  CalendarOutlined,
  DashboardOutlined,
  ExperimentOutlined,
  LineChartOutlined,
  TableOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatCost,
  formatDateTime,
  formatDisplayNumber,
  formatDuration,
  formatNumber,
  projectDisplay,
  sessionDisplay,
  type ModelSignalAnomalySession,
  type ModelSignalCohort,
  type ModelSignalMatrixCell,
  type ModelSignalMatrixRow,
  type ModelSignalMetricSet,
  type ModelSignalProjectHotspot,
  type ModelSignalsDailyMetric,
  type ModelSignalsProjectMetric,
  type ModelSignalsWindow,
  type ModelSignals,
  type ModelSignalsHealthSummary,
  type Overview,
  type Session,
  type UsageBreakdownBucket
} from '../api'
import ModelSignalsMetricChart from '../components/ModelSignalsMetricChart.vue'
import PageHeader from '../components/PageHeader.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { sourceDisplay } from '../presentation/sourceIdentity'
import { useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import { buildUsageAgentOptions, buildUsageModelOptions, buildUsageProjectOptions } from './useUsageScopeOptions'

const ATable = AntTable as unknown as DefineComponent

type ProjectMetricRow = ModelSignalsProjectMetric | ModelSignalProjectHotspot

const router = useRouter()
const resource = useAsyncResource<ModelSignals | null>(null)
const signals = computed(() => resource.data.value)
const loading = resource.loading
const error = resource.error
const activeTab = ref('charts')
const optionOverview = ref<Overview | null>(null)
const projectOptionRows = ref<UsageBreakdownBucket[]>([])
const scope = useUsageScopeRoute(() => {
  void load()
})

const { t } = useMessages({
  en: {
    'title': 'Model Signals',
    'subtitle': 'Service health and behavior drift across model cohorts, agents, and projects',
    'tab.charts': 'Metric Charts',
    'tab.overview': 'Health Overview',
    'tab.daily': 'Daily Metrics',
    'tab.cohorts': 'Cohorts',
    'tab.matrix': 'Matrix',
    'tab.projects': 'Project Hotspots',
    'tab.anomalies': 'Anomalies',
    'metric.health': 'Health',
    'metric.healthNote': '{current} vs {baseline}',
    'metric.cohorts': 'Cohorts',
    'metric.cohortsNote': '{sessions} sessions, {calls} model calls',
    'metric.criticalWarning': 'Critical / Warning',
    'metric.criticalWarningNote': '{critical} critical, {warning} warning',
    'metric.lowConfidence': 'Low Confidence',
    'metric.lowConfidenceNote': '{count} cohorts below confidence threshold',
    'metric.throughput': 'Model Throughput',
    'metric.throughputNote': 'Current scope total-token throughput',
    'metric.toolFailure': 'Tool Failure Rate',
    'metric.toolFailureNote': '{failed} failed of {total} tool calls',
    'topReasons.label': 'Top reasons',
    'overview.title': 'Top Drift Cohorts',
    'overview.kicker': 'Highest-severity cohort drift in the current scope',
    'daily.title': 'Daily Efficiency',
    'daily.kicker': 'Daily model service cost, latency, throughput, and confidence signals',
    'cohorts.title': 'Cohort Drift',
    'cohorts.kicker': 'Provider, model, source, and project cohorts compared with baseline behavior',
    'matrix.title': 'Source Model Matrix',
    'matrix.kicker': 'Agent/source rows with compact model health cells',
    'projects.title': 'Project Hotspots',
    'projects.kicker': 'Projects with concentrated model service drift or elevated sample risk',
    'anomaly.title': 'Anomaly Sessions',
    'anomaly.kicker': 'Sessions with unusual operational signals for review',
    'column.source': 'Source',
    'column.model': 'Model',
    'column.models': 'Models',
    'column.project': 'Project',
    'column.date': 'Date',
    'column.samples': 'Samples',
    'column.sessions': 'Sessions',
    'column.modelCalls': 'Model calls',
    'column.toolCalls': 'Tools',
    'column.failedTools': 'Failed',
    'column.tokens': 'Tokens',
    'column.cost': 'Cost',
    'column.costBurn': 'Cost burn',
    'column.costPerSession': 'Cost/session',
    'column.costPerActiveHour': 'Cost/active-hour',
    'column.cacheSavings': 'Cache savings',
    'column.latency': 'Latency',
    'column.p90Latency': 'P90 latency',
    'column.throughput': 'Throughput',
    'column.p10Throughput': 'P10 throughput',
    'column.outputThroughput': 'Out tok/s',
    'column.toolFailure': 'Tool fail',
    'column.retryPressure': 'Retry pressure',
    'column.failurePressure': 'Failure pressure',
    'column.mix': 'Mix',
    'column.health': 'Health',
    'column.severity': 'Severity',
    'column.confidence': 'Confidence',
    'column.reasons': 'Reasons',
    'column.sources': 'Sources',
    'column.session': 'Session',
    'column.signal': 'Signals',
    'column.outputExpansion': 'Output/input',
    'column.reasoning': 'Reasoning',
    'column.cacheMiss': 'Cache miss',
    'column.started': 'Started',
    'column.wall': 'Model time',
    'metric.current': 'current',
    'metric.baseline': 'base',
    'label.lowSample': 'low sample',
    'label.unpriced': 'unpriced',
    'empty.loading': 'Loading model signals...',
    'empty.overview': 'No drift cohorts match the current scope',
    'empty.daily': 'No daily metrics match the current scope',
    'empty.cohorts': 'No cohort rows match the current scope',
    'empty.matrix': 'No matrix rows match the current scope',
    'empty.projects': 'No project hotspots match the current scope',
    'empty.anomalies': 'No anomaly sessions match the current scope',
    'fallback.unknown': 'unknown',
    'fallback.noReason': 'No drift reason',
    'error.title': 'Model signals failed to load',
    'action.openSession': 'Open session',
    'severity.ok': 'ok',
    'severity.watch': 'watch',
    'severity.warning': 'warning',
    'severity.critical': 'critical',
    'severity.healthy': 'healthy',
    'severity.unknown': 'unknown',
    'severity.high': 'high',
    'severity.medium': 'medium',
    'severity.low': 'low'
  },
  'zh-CN': {
    'title': '模型表现',
    'subtitle': '按模型、来源和项目查看健康状态、延迟、吞吐和费用变化',
    'tab.charts': '指标图表',
    'tab.overview': '健康概览',
    'tab.daily': '每日指标',
    'tab.cohorts': '分组',
    'tab.matrix': '矩阵',
    'tab.projects': '项目热点',
    'tab.anomalies': '异常',
    'metric.health': '健康',
    'metric.healthNote': '{current} 对比 {baseline}',
    'metric.cohorts': '分组',
    'metric.cohortsNote': '{sessions} 个会话，{calls} 次模型调用',
    'metric.criticalWarning': '严重 / 警告',
    'metric.criticalWarningNote': '{critical} 个严重，{warning} 个警告',
    'metric.lowConfidence': '低置信',
    'metric.lowConfidenceNote': '{count} 个分组低于置信阈值',
    'metric.throughput': '模型吞吐',
    'metric.throughputNote': '当前范围总 Token 吞吐',
    'metric.toolFailure': '工具失败率',
    'metric.toolFailureNote': '{total} 次工具调用中 {failed} 次失败',
    'topReasons.label': '主要原因',
    'overview.title': '主要漂移分组',
    'overview.kicker': '当前范围内严重程度最高的分组漂移',
    'daily.title': '每日效率',
    'daily.kicker': '按天展示模型服务费用、延迟、吞吐与置信度',
    'cohorts.title': '分组漂移',
    'cohorts.kicker': '按供应商、模型、来源和项目对比基线行为',
    'matrix.title': '来源模型矩阵',
    'matrix.kicker': '按 Agent/来源展示紧凑模型健康单元',
    'projects.title': '项目热点',
    'projects.kicker': '展示模型服务漂移集中或样本风险较高的项目',
    'anomaly.title': '异常会话',
    'anomaly.kicker': '列出指标异常的会话供复核',
    'column.source': '来源',
    'column.model': '模型',
    'column.models': '模型',
    'column.project': '项目',
    'column.date': '日期',
    'column.samples': '样本',
    'column.sessions': '会话',
    'column.modelCalls': '模型调用',
    'column.toolCalls': '工具',
    'column.failedTools': '失败',
    'column.tokens': 'Token',
    'column.cost': '费用',
    'column.costBurn': '费用消耗',
    'column.costPerSession': '每会话费用',
    'column.costPerActiveHour': '每活跃小时',
    'column.cacheSavings': '缓存节省',
    'column.latency': '延迟',
    'column.p90Latency': 'P90 延迟',
    'column.throughput': '吞吐',
    'column.p10Throughput': 'P10 吞吐',
    'column.outputThroughput': '输出/秒',
    'column.toolFailure': '工具失败',
    'column.retryPressure': '重试压力',
    'column.failurePressure': '失败压力',
    'column.mix': '模型占比',
    'column.health': '健康',
    'column.severity': '严重度',
    'column.confidence': '置信度',
    'column.reasons': '原因',
    'column.sources': '来源数',
    'column.session': '会话',
    'column.signal': '异常指标',
    'column.outputExpansion': '输出/输入',
    'column.reasoning': '推理',
    'column.cacheMiss': '缓存未命中',
    'column.started': '开始',
    'column.wall': '模型耗时',
    'metric.current': '当前',
    'metric.baseline': '基线',
    'label.lowSample': '低样本',
    'label.unpriced': '未定价',
    'empty.loading': '正在加载模型表现...',
    'empty.overview': '当前范围内没有漂移分组',
    'empty.daily': '当前范围内没有每日指标',
    'empty.cohorts': '当前范围内没有分组行',
    'empty.matrix': '当前范围内没有矩阵行',
    'empty.projects': '当前范围内没有项目热点',
    'empty.anomalies': '当前范围内没有异常会话',
    'fallback.unknown': '未知',
    'fallback.noReason': '无漂移原因',
    'error.title': '模型表现加载失败',
    'action.openSession': '打开会话',
    'severity.ok': '正常',
    'severity.watch': '观察',
    'severity.warning': '警告',
    'severity.critical': '严重',
    'severity.healthy': '健康',
    'severity.unknown': '未知',
    'severity.high': '高',
    'severity.medium': '中',
    'severity.low': '低'
  }
})

const healthSummary = computed(() => signals.value?.healthSummary || fallbackHealthSummary(signals.value))
const cohortRows = computed(() => signals.value?.cohorts || [])
const matrixRows = computed(() => signals.value?.matrix || [])
const matrixCells = computed(() => matrixRows.value.flatMap((row) => row.cells || []))
const projectHotspotRows = computed(() => signals.value?.projectHotspots || [])
const dailyMetricRows = computed(() => signals.value?.dailyMetrics || [])
const projectMetricRows = computed(() => signals.value?.projectMetrics || [])
const hasProjectMetrics = computed(() => projectMetricRows.value.length > 0)
const projectRows = computed<ProjectMetricRow[]>(() => hasProjectMetrics.value ? projectMetricRows.value : projectHotspotRows.value)
const normalizedAnomalies = computed(() => (signals.value?.anomalySessions || []).map(normalizeAnomaly))
const hasData = computed(() => Boolean(signals.value?.totalSessions || healthSummary.value.cohortCount || dailyMetricRows.value.length))

const agentOptions = computed(() =>
  buildUsageAgentOptions({
    sources: [
      cohortRows.value,
      matrixRows.value,
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
      cohortRows.value,
      matrixCells.value,
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
      cohortRows.value,
      projectRows.value,
      projectOptionRows.value,
      normalizedAnomalies.value,
      optionOverview.value?.recentSessions,
      optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.project,
    fallback: t('fallback.unknown')
  })
)

const healthReasonRows = computed(() => (healthSummary.value.topReasons || []).map((reason, index) => ({
  key: `${reasonText(reason)}:${index}`,
  reason: reasonText(reason),
  count: reasonCount(reason),
  severity: reasonSeverity(reason)
})))

const topDriftCohorts = computed(() => {
  const driftRows = cohortRows.value.filter((row) => severityRank(row.drift?.severity) > 0)
  return (driftRows.length ? driftRows : cohortRows.value).slice(0, 8)
})

const metricCards = computed(() => {
  const item = signals.value
  const summary = healthSummary.value
  return [
    {
      label: t('metric.health'),
      value: displayText(severityLabel(summary.severity)),
      note: t('metric.healthNote', {
        current: formatWindow(summary.currentWindow),
        baseline: formatWindow(summary.baselineWindow)
      }),
      icon: DashboardOutlined,
      tone: severityMetricTone(summary.severity)
    },
    {
      label: t('metric.cohorts'),
      value: formatDisplayNumber(summary.cohortCount),
      note: t('metric.cohortsNote', {
        sessions: formatDisplayNumber(item?.totalSessions).main,
        calls: formatDisplayNumber(item?.totalModelCalls).main
      }),
      icon: BranchesOutlined,
      tone: 'metric-primary'
    },
    {
      label: t('metric.criticalWarning'),
      value: displayPair(summary.criticalCohorts, summary.warningCohorts),
      note: t('metric.criticalWarningNote', {
        critical: formatDisplayNumber(summary.criticalCohorts).main,
        warning: formatDisplayNumber(summary.warningCohorts).main
      }),
      icon: WarningOutlined,
      tone: summary.criticalCohorts ? 'metric-danger' : summary.warningCohorts ? 'metric-warning' : 'metric-success'
    },
    {
      label: t('metric.lowConfidence'),
      value: formatDisplayNumber(summary.lowConfidenceCohorts),
      note: t('metric.lowConfidenceNote', { count: formatDisplayNumber(summary.lowConfidenceCohorts).main }),
      icon: ExperimentOutlined,
      tone: summary.lowConfidenceCohorts ? 'metric-warning' : 'metric-neutral'
    },
    {
      label: t('metric.throughput'),
      value: displayRate(item?.modelThroughputTokensPerSecond, ' tok/s', 1),
      note: t('metric.throughputNote'),
      icon: LineChartOutlined,
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
    }
  ]
})

const overviewColumns = computed(() => [
  { title: t('column.source'), dataIndex: 'sourceLabel', key: 'source', width: 170 },
  { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 210 },
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 190 },
  { title: t('column.severity'), dataIndex: 'severity', key: 'severity', width: 104 },
  { title: t('column.latency'), key: 'latency', width: 126, align: 'right' },
  { title: t('column.throughput'), key: 'throughput', width: 126, align: 'right' },
  { title: t('column.confidence'), key: 'confidence', width: 110, align: 'right' },
  { title: t('column.reasons'), key: 'reasons', width: 280 }
])

const dailyColumns = computed(() => [
  { title: t('column.date'), dataIndex: 'date', key: 'date', width: 112 },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 88, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 108, align: 'right' },
  { title: t('column.costPerSession'), dataIndex: 'costPerSession', key: 'costPerSession', width: 124, align: 'right' },
  { title: t('column.costPerActiveHour'), dataIndex: 'costPerActiveHour', key: 'costPerActiveHour', width: 138, align: 'right' },
  { title: t('column.cacheSavings'), dataIndex: 'cacheSavingsUsd', key: 'cacheSavings', width: 124, align: 'right' },
  { title: t('column.p90Latency'), key: 'p90Latency', width: 124, align: 'right' },
  { title: t('column.p10Throughput'), key: 'p10Throughput', width: 128, align: 'right' },
  { title: t('column.retryPressure'), key: 'retryPressure', width: 130, align: 'right' },
  { title: t('column.failurePressure'), key: 'failurePressure', width: 132, align: 'right' },
  { title: t('column.confidence'), key: 'confidence', width: 220 }
])

const cohortColumns = computed(() => [
  { title: t('column.source'), dataIndex: 'sourceLabel', key: 'source', width: 180 },
  { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 220 },
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 190 },
  { title: t('column.samples'), key: 'samples', width: 128, align: 'right' },
  { title: t('column.latency'), key: 'latency', width: 136, align: 'right' },
  { title: t('column.throughput'), key: 'throughput', width: 136, align: 'right' },
  { title: t('column.outputThroughput'), key: 'outputThroughput', width: 118, align: 'right' },
  { title: t('column.toolFailure'), key: 'toolFailure', width: 108, align: 'right' },
  { title: t('column.severity'), key: 'severity', width: 104 },
  { title: t('column.confidence'), key: 'confidence', width: 104, align: 'right' },
  { title: t('column.reasons'), key: 'reasons', width: 280 }
])

const matrixColumns = computed(() => [
  { title: t('column.source'), dataIndex: 'sourceLabel', key: 'source', width: 230 },
  { title: t('column.models'), dataIndex: 'cells', key: 'models' }
])

const projectColumns = computed(() => {
  if (!hasProjectMetrics.value) {
    return [
      { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 260 },
      { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 92, align: 'right' },
      { title: t('column.sources'), dataIndex: 'sourceCount', key: 'sources', width: 88, align: 'right' },
      { title: t('column.models'), dataIndex: 'modelCount', key: 'models', width: 88, align: 'right' },
      { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 118, align: 'right' },
      { title: t('column.latency'), key: 'latency', width: 136, align: 'right' },
      { title: t('column.throughput'), key: 'throughput', width: 136, align: 'right' },
      { title: t('column.severity'), key: 'severity', width: 104 },
      { title: t('column.confidence'), key: 'confidence', width: 104, align: 'right' },
      { title: t('column.reasons'), key: 'reasons', width: 280 }
    ]
  }

  return [
    { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 260 },
    { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 92, align: 'right' },
    { title: t('column.mix'), key: 'mix', width: 210 },
    { title: t('column.costBurn'), key: 'costBurn', width: 132, align: 'right' },
    { title: t('column.cacheSavings'), key: 'cacheSavings', width: 124, align: 'right' },
    { title: t('column.health'), key: 'health', width: 142 },
    { title: t('column.p90Latency'), key: 'latency', width: 136, align: 'right' },
    { title: t('column.p10Throughput'), key: 'throughput', width: 136, align: 'right' },
    { title: t('column.failurePressure'), key: 'pressure', width: 136, align: 'right' },
    { title: t('column.reasons'), key: 'reasons', width: 280 }
  ]
})

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

const overviewTableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.overview') }))
const dailyTableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.daily') }))
const cohortTableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.cohorts') }))
const matrixTableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.matrix') }))
const projectTableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.projects') }))
const anomalyTableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.anomalies') }))

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

function fallbackHealthSummary(item: ModelSignals | null | undefined): ModelSignalsHealthSummary {
  const hasToolFailures = Boolean(item?.failedToolCalls)
  return {
    currentWindow: emptyWindow(),
    baselineWindow: emptyWindow(),
    severity: hasToolFailures ? 'warning' : 'ok',
    cohortCount: item?.modelBreakdown?.length || 0,
    warningCohorts: hasToolFailures ? 1 : 0,
    criticalCohorts: 0,
    lowConfidenceCohorts: 0,
    topReasons: hasToolFailures ? ['Tool failures above baseline'] : []
  }
}

function displayText(text: string) {
  return { main: text, full: text }
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

function formatLatency(value?: number) {
  return `${formatRate(value, 0)} ms/1k`
}

function formatOptionalCost(value?: number) {
  if (value === undefined || value === null) return '-'
  return formatCost(value)
}

function formatThroughput(value?: number) {
  return `${formatRate(value, 1)} tok/s`
}

function formatPressure(value?: number) {
  return `${formatRate(value, 2)}/session`
}

function p90Latency(metric?: ModelSignalMetricSet) {
  return metric?.p90ModelLatencyMsPer1kOutputTokens ?? metric?.modelLatencyMsPer1kOutputTokens
}

function p10Throughput(metric?: ModelSignalMetricSet) {
  return metric?.p10ModelThroughputTokensPerSecond ?? metric?.modelThroughputTokensPerSecond
}

function failurePressure(metric?: ModelSignalMetricSet) {
  return metric?.failurePressure ?? safeMetricRate(metric?.failedToolCalls, metric?.sessionCount)
}

function safeMetricRate(numerator?: number, denominator?: number) {
  return denominator && denominator > 0 ? (numerator || 0) / denominator : 0
}

function unpricedNote(metric?: ModelSignalMetricSet) {
  const count = metric?.unpricedSessionCount || 0
  return count > 0 ? `${formatNumber(count)} ${t('label.unpriced')}` : ''
}

function confidenceReason(record: Pick<ModelSignalsDailyMetric, 'keyReason' | 'drift' | 'lowSample'>) {
  return record.keyReason || record.drift?.reasons?.[0] || record.drift?.sampleNote || (record.lowSample ? t('label.lowSample') : t('fallback.noReason'))
}

function formatConfidence(value?: string | number) {
  if (typeof value === 'number') return formatPercent(value)
  const normalized = (value || '').trim().toLowerCase()
  if (!normalized) return t('fallback.unknown')
  return normalized
}

function emptyWindow(): ModelSignalsWindow {
  return {
    from: '',
    to: '',
    sessionCount: 0,
    modelCalls: 0
  }
}

function formatWindow(window?: ModelSignalsWindow) {
  if (!window?.from && !window?.to) return t('fallback.unknown')
  const from = window.from ? window.from.slice(0, 10) : ''
  const to = window.to ? window.to.slice(0, 10) : ''
  const range = from && to && from !== to ? `${from} - ${to}` : from || to
  if (!range) return t('fallback.unknown')
  return `${range}, ${formatNumber(window.sessionCount || 0)} ${t('column.sessions')}`
}

function metricClass(current?: number, baseline?: number, lowerIsBetter = false) {
  if (!Number.isFinite(current) || !Number.isFinite(baseline) || !baseline) return ''
  const degraded = lowerIsBetter ? (current || 0) > (baseline || 0) * 1.15 : (current || 0) < (baseline || 0) * 0.85
  const improved = lowerIsBetter ? (current || 0) < (baseline || 0) * 0.9 : (current || 0) > (baseline || 0) * 1.1
  if (degraded) return 'status-error'
  if (improved) return 'status-ok'
  return ''
}

function severityRank(value?: string): number {
  const normalized = (value || '').toLowerCase()
  if (normalized === 'critical' || normalized === 'high') return 3
  if (normalized === 'warning' || normalized === 'medium') return 2
  if (normalized === 'watch' || normalized === 'low' || normalized === 'unknown') return 1
  return 0
}

function severityLabel(value?: string) {
  const normalized = (value || 'ok').toLowerCase()
  if (normalized === 'critical') return t('severity.critical')
  if (normalized === 'warning') return t('severity.warning')
  if (normalized === 'watch') return t('severity.watch')
  if (normalized === 'healthy') return t('severity.healthy')
  if (normalized === 'unknown') return t('severity.unknown')
  if (normalized === 'high') return t('severity.high')
  if (normalized === 'medium') return t('severity.medium')
  if (normalized === 'low') return t('severity.low')
  if (normalized === 'ok') return t('severity.ok')
  return normalized
}

function severityTagColor(value?: string) {
  const rank = severityRank(value)
  if (rank >= 3) return 'error'
  if (rank === 2) return 'warning'
  if (rank === 1) return 'processing'
  return 'success'
}

function severityMetricTone(value?: string) {
  const rank = severityRank(value)
  if (rank >= 3) return 'metric-danger'
  if (rank === 2) return 'metric-warning'
  if (rank === 1) return 'metric-info'
  return 'metric-success'
}

function severityClass(value?: string) {
  const rank = severityRank(value)
  if (rank >= 3) return 'severity-critical'
  if (rank === 2) return 'severity-warning'
  if (rank === 1) return 'severity-watch'
  return 'severity-ok'
}

function driftRowClass(record: { drift?: { severity?: string } }) {
  const rank = severityRank(record.drift?.severity)
  return { class: rank >= 3 ? 'model-signals-critical-row' : rank === 2 ? 'model-signals-warning-row' : '' }
}

function anomalyRowClass(record: NormalizedAnomalySession) {
  return { class: record.failedToolCalls > 0 || record.severity === 'high' ? 'model-signals-warning-row' : '' }
}

function reasonText(row: string): string {
  return row
}

function reasonCount(_row: string): number | undefined {
  return undefined
}

function reasonSeverity(_row: string): string | undefined {
  return undefined
}

function sourceInfo(record: Parameters<typeof sourceDisplay>[0]) {
  return sourceDisplay(record, t('fallback.unknown'))
}

function projectInfo(record: { projectPath?: string; rawSourcePath?: string }) {
  return projectDisplay(record.projectPath || record.rawSourcePath)
}

function projectMixInfo(record: ProjectMetricRow) {
  const projectMetric = record as Partial<ModelSignalsProjectMetric>
  const model = projectMetric.dominantModel || t('fallback.unknown')
  const provider = projectMetric.dominantModelProvider || ''
  const share = projectMetric.dominantModelShare !== undefined ? formatPercent(projectMetric.dominantModelShare) : ''
  const summary = [
    share,
    `${formatNumber(record.modelCount)} ${t('column.models')}`,
    `${formatNumber(record.sourceCount)} ${t('column.sources')}`
  ].filter(Boolean).join(' · ')
  return {
    model,
    provider,
    summary,
    full: [provider, model, summary].filter(Boolean).join(' / ')
  }
}

function projectHealthTitle(record: ProjectMetricRow) {
  return [
    `${t('column.health')}: ${severityLabel(record.drift?.severity)} (${formatConfidence(record.drift?.confidence)})`,
    `${t('column.p90Latency')}: ${formatLatency(p90Latency(record.current))} / ${t('metric.baseline')} ${formatLatency(p90Latency(record.baseline))}`,
    `${t('column.p10Throughput')}: ${formatThroughput(p10Throughput(record.current))} / ${t('metric.baseline')} ${formatThroughput(p10Throughput(record.baseline))}`,
    `${t('column.failurePressure')}: ${formatPressure(failurePressure(record.current))} / ${t('metric.baseline')} ${formatPressure(failurePressure(record.baseline))}`
  ].join('\n')
}

function cohortRowKey(record: ModelSignalCohort) {
  return record.cohortKey || `${record.modelProvider}:${record.model}:${record.projectPath}`
}

function matrixRowKey(record: ModelSignalMatrixRow) {
  return record.sourceKey || (record.sourceId !== undefined ? `source:${record.sourceId}` : `${record.agentKind}:${record.agentName}`)
}

function matrixCellKey(cell: ModelSignalMatrixCell) {
  return `${cell.modelProvider}:${cell.model}`
}

function matrixCellTitle(cell: ModelSignalMatrixCell) {
  return [
    `${cell.modelProvider || t('fallback.unknown')} / ${cell.model || t('fallback.unknown')}`,
    `${t('column.latency')}: ${formatLatency(cell.current?.modelLatencyMsPer1kOutputTokens)} (${t('metric.baseline')} ${formatLatency(cell.baseline?.modelLatencyMsPer1kOutputTokens)})`,
    `${t('column.throughput')}: ${formatRate(cell.current?.modelThroughputTokensPerSecond, 1)} tok/s (${t('metric.baseline')} ${formatRate(cell.baseline?.modelThroughputTokensPerSecond, 1)})`,
    `${t('column.confidence')}: ${formatConfidence(cell.confidence)}`
  ].join('\n')
}

function dailyRowKey(record: ModelSignalsDailyMetric) {
  return record.date
}

function projectRowKey(record: ProjectMetricRow) {
  return record.projectPath || `${record.modelCount}:${record.sourceCount}:${record.totalTokens}`
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
        <div class="model-signals-tabs" role="tablist" :aria-label="t('title')">
          <a-button :type="activeTab === 'charts' ? 'primary' : 'default'" role="tab" :aria-selected="activeTab === 'charts'" @click="activeTab = 'charts'">
            <template #icon><LineChartOutlined /></template>
            {{ t('tab.charts') }}
          </a-button>
          <a-button :type="activeTab === 'overview' ? 'primary' : 'default'" role="tab" :aria-selected="activeTab === 'overview'" @click="activeTab = 'overview'">
            <template #icon><DashboardOutlined /></template>
            {{ t('tab.overview') }}
          </a-button>
          <a-button :type="activeTab === 'daily' ? 'primary' : 'default'" role="tab" :aria-selected="activeTab === 'daily'" @click="activeTab = 'daily'">
            <template #icon><CalendarOutlined /></template>
            {{ t('tab.daily') }}
          </a-button>
          <a-button :type="activeTab === 'cohorts' ? 'primary' : 'default'" role="tab" :aria-selected="activeTab === 'cohorts'" @click="activeTab = 'cohorts'">
            <template #icon><BranchesOutlined /></template>
            {{ t('tab.cohorts') }}
          </a-button>
          <a-button :type="activeTab === 'matrix' ? 'primary' : 'default'" role="tab" :aria-selected="activeTab === 'matrix'" @click="activeTab = 'matrix'">
            <template #icon><TableOutlined /></template>
            {{ t('tab.matrix') }}
          </a-button>
          <a-button :type="activeTab === 'projects' ? 'primary' : 'default'" role="tab" :aria-selected="activeTab === 'projects'" @click="activeTab = 'projects'">
            <template #icon><LineChartOutlined /></template>
            {{ t('tab.projects') }}
          </a-button>
          <a-button :type="activeTab === 'anomalies' ? 'primary' : 'default'" role="tab" :aria-selected="activeTab === 'anomalies'" @click="activeTab = 'anomalies'">
            <template #icon><WarningOutlined /></template>
            {{ t('tab.anomalies') }}
          </a-button>
        </div>

        <div v-if="activeTab === 'charts'">
          <ModelSignalsMetricChart
            :daily-rows="dailyMetricRows"
            :project-rows="projectRows"
            :loading="loading"
          />
        </div>

        <div v-else-if="activeTab === 'overview'">
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

            <div v-if="healthReasonRows.length" class="model-signals-reason-strip">
              <span class="metric-label">{{ t('topReasons.label') }}</span>
              <div class="model-signals-tags">
                <a-tag
                  v-for="reason in healthReasonRows"
                  :key="reason.key"
                  :color="severityTagColor(reason.severity)"
                >
                  {{ reason.reason }}<span v-if="reason.count"> · {{ formatNumber(reason.count) }}</span>
                </a-tag>
              </div>
            </div>

            <section class="panel">
              <div class="panel-header">
                <div>
                  <h2 class="panel-title">{{ t('overview.title') }}</h2>
                  <div class="panel-kicker">{{ t('overview.kicker') }}</div>
                </div>
                <WarningOutlined class="panel-header-icon" />
              </div>
              <a-table
                class="dense-table model-signals-overview-table"
                :columns="overviewColumns"
                :data-source="topDriftCohorts"
                :loading="loading"
                :locale="overviewTableLocale"
                :pagination="false"
                :row-key="cohortRowKey"
                size="small"
                :custom-row="driftRowClass"
                :scroll="{ x: 1340 }"
              >
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'source'">
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
                    <div class="source-identity-meta">{{ record.modelProvider || '-' }}</div>
                  </template>
                  <template v-else-if="column.key === 'severity'">
                    <a-tag class="status-tag" :color="severityTagColor(record.drift?.severity)">
                      {{ severityLabel(record.drift?.severity) }}
                    </a-tag>
                  </template>
                  <template v-else-if="column.key === 'latency'">
                    <div class="metric-comparison" :class="metricClass(record.current?.modelLatencyMsPer1kOutputTokens, record.baseline?.modelLatencyMsPer1kOutputTokens, true)">
                      <span>{{ formatLatency(record.current?.modelLatencyMsPer1kOutputTokens) }}</span>
                      <span>{{ t('metric.baseline') }} {{ formatLatency(record.baseline?.modelLatencyMsPer1kOutputTokens) }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'throughput'">
                    <div class="metric-comparison" :class="metricClass(record.current?.modelThroughputTokensPerSecond, record.baseline?.modelThroughputTokensPerSecond)">
                      <span>{{ formatRate(record.current?.modelThroughputTokensPerSecond, 1) }} tok/s</span>
                      <span>{{ t('metric.baseline') }} {{ formatRate(record.baseline?.modelThroughputTokensPerSecond, 1) }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'confidence'">
                    <span class="number-cell">{{ formatConfidence(record.drift?.confidence) }}</span>
                    <div v-if="record.drift?.sampleNote" class="source-identity-meta">{{ record.drift.sampleNote }}</div>
                  </template>
                  <template v-else-if="column.key === 'reasons'">
                    <div v-if="record.drift?.reasons?.length" class="model-signals-tags">
                      <a-tag v-for="reason in record.drift.reasons" :key="reason" :color="severityTagColor(record.drift?.severity)">
                        {{ reason }}
                      </a-tag>
                    </div>
                    <span v-else class="muted">{{ t('fallback.noReason') }}</span>
                  </template>
                </template>
              </a-table>
            </section>
          </div>
        </div>

        <div v-else-if="activeTab === 'daily'">
          <div class="section-stack">
            <ModelSignalsMetricChart
              :daily-rows="dailyMetricRows"
              :project-rows="projectRows"
              :loading="loading"
              initial-mode="daily"
              :allow-mode-switch="false"
            />

            <section class="panel">
              <div class="panel-header">
                <div>
                  <h2 class="panel-title">{{ t('daily.title') }}</h2>
                  <div class="panel-kicker">{{ t('daily.kicker') }}</div>
                </div>
                <CalendarOutlined class="panel-header-icon" />
              </div>
              <a-table
                class="dense-table model-signals-daily-table"
                :columns="dailyColumns"
                :data-source="dailyMetricRows"
                :loading="loading"
                :locale="dailyTableLocale"
                :pagination="{ pageSize: 10 }"
                :row-key="dailyRowKey"
                size="small"
                :custom-row="driftRowClass"
                :scroll="{ x: 1540 }"
              >
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'date'">
                    <span class="mono model-signals-date">{{ record.date }}</span>
                  </template>
                  <template v-else-if="column.key === 'sessions'">
                    <div class="metric-comparison">
                      <span>{{ formatNumber(record.sessionCount) }}</span>
                      <span>{{ formatNumber(record.modelCalls) }} {{ t('column.modelCalls') }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'cost'">
                    <div class="metric-comparison">
                      <span>{{ formatCost(record.estimatedCostUsd) }}</span>
                      <span>{{ unpricedNote(record) || formatDuration(record.activeDurationMs) }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'costPerSession'">
                    <span class="number-cell">{{ formatOptionalCost(record.costPerSession) }}</span>
                  </template>
                  <template v-else-if="column.key === 'costPerActiveHour'">
                    <span class="number-cell">{{ formatOptionalCost(record.costPerActiveHour) }}</span>
                  </template>
                  <template v-else-if="column.key === 'cacheSavings'">
                    <span class="number-cell status-ok">{{ formatOptionalCost(record.cacheSavingsUsd) }}</span>
                  </template>
                  <template v-else-if="column.key === 'p90Latency'">
                    <div class="metric-comparison">
                      <span>{{ formatLatency(p90Latency(record)) }}</span>
                      <span>p50 {{ formatLatency(record.p50ModelLatencyMsPer1kOutputTokens) }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'p10Throughput'">
                    <div class="metric-comparison">
                      <span>{{ formatThroughput(p10Throughput(record)) }}</span>
                      <span>p50 {{ formatThroughput(record.p50ModelThroughputTokensPerSecond) }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'retryPressure'">
                    <div class="metric-comparison" :class="{ 'status-error': record.avgModelCallsPerSession > 1.5 }">
                      <span>{{ formatRate(record.avgModelCallsPerSession, 2) }}/session</span>
                      <span>{{ formatNumber(record.modelCalls) }} {{ t('column.modelCalls') }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'failurePressure'">
                    <div class="metric-comparison" :class="{ 'status-error': failurePressure(record) > 0 }">
                      <span>{{ formatPressure(failurePressure(record)) }}</span>
                      <span>{{ formatPercent(record.toolFailureRate) }} {{ t('column.toolFailure') }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'confidence'">
                    <div class="model-signals-confidence-cell">
                      <a-tag v-if="record.lowSample" class="status-tag" color="processing">{{ t('label.lowSample') }}</a-tag>
                      <a-tag class="status-tag" :color="severityTagColor(record.drift?.severity)">
                        {{ severityLabel(record.drift?.severity) }}
                      </a-tag>
                      <a-tooltip :title="confidenceReason(record)" placement="topLeft">
                        <span class="model-signals-reason-text">{{ confidenceReason(record) }}</span>
                      </a-tooltip>
                    </div>
                  </template>
                </template>
              </a-table>
            </section>
          </div>
        </div>

        <div v-else-if="activeTab === 'cohorts'">
          <section class="panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">{{ t('cohorts.title') }}</h2>
                <div class="panel-kicker">{{ t('cohorts.kicker') }}</div>
              </div>
              <BranchesOutlined class="panel-header-icon" />
            </div>
            <a-table
              class="dense-table model-signals-cohort-table"
              :columns="cohortColumns"
              :data-source="cohortRows"
              :loading="loading"
              :locale="cohortTableLocale"
              :pagination="{ pageSize: 10 }"
              :row-key="cohortRowKey"
              size="small"
              :custom-row="driftRowClass"
              :scroll="{ x: 1740 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'source'">
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
                  <div class="source-identity-meta">{{ record.modelProvider || '-' }}</div>
                </template>
                <template v-else-if="column.key === 'samples'">
                  <div class="metric-comparison">
                    <span>{{ formatNumber(record.sessionCount) }} {{ t('column.sessions') }}</span>
                    <span>{{ formatNumber(record.modelCalls) }} {{ t('column.modelCalls') }}</span>
                  </div>
                </template>
                <template v-else-if="column.key === 'latency'">
                  <div class="metric-comparison" :class="metricClass(record.current?.modelLatencyMsPer1kOutputTokens, record.baseline?.modelLatencyMsPer1kOutputTokens, true)">
                    <span>{{ formatLatency(record.current?.modelLatencyMsPer1kOutputTokens) }}</span>
                    <span>{{ t('metric.baseline') }} {{ formatLatency(record.baseline?.modelLatencyMsPer1kOutputTokens) }}</span>
                  </div>
                </template>
                <template v-else-if="column.key === 'throughput'">
                  <div class="metric-comparison" :class="metricClass(record.current?.modelThroughputTokensPerSecond, record.baseline?.modelThroughputTokensPerSecond)">
                    <span>{{ formatRate(record.current?.modelThroughputTokensPerSecond, 1) }} tok/s</span>
                    <span>{{ t('metric.baseline') }} {{ formatRate(record.baseline?.modelThroughputTokensPerSecond, 1) }}</span>
                  </div>
                </template>
                <template v-else-if="column.key === 'outputThroughput'">
                  <div class="metric-comparison" :class="metricClass(record.current?.modelThroughputOutputTokensPerSecond, record.baseline?.modelThroughputOutputTokensPerSecond)">
                    <span>{{ formatRate(record.current?.modelThroughputOutputTokensPerSecond, 1) }}</span>
                    <span>{{ t('metric.baseline') }} {{ formatRate(record.baseline?.modelThroughputOutputTokensPerSecond, 1) }}</span>
                  </div>
                </template>
                <template v-else-if="column.key === 'toolFailure'">
                  <div class="metric-comparison" :class="metricClass(record.current?.toolFailureRate, record.baseline?.toolFailureRate, true)">
                    <span>{{ formatPercent(record.current?.toolFailureRate) }}</span>
                    <span>{{ formatNumber(record.failedToolCalls) }} / {{ formatNumber(record.toolCalls) }}</span>
                  </div>
                </template>
                <template v-else-if="column.key === 'severity'">
                  <a-tag class="status-tag" :color="severityTagColor(record.drift?.severity)">
                    {{ severityLabel(record.drift?.severity) }}
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'confidence'">
                  <span class="number-cell">{{ formatConfidence(record.drift?.confidence) }}</span>
                  <div v-if="record.drift?.sampleNote" class="source-identity-meta">{{ record.drift.sampleNote }}</div>
                </template>
                <template v-else-if="column.key === 'reasons'">
                  <div v-if="record.drift?.reasons?.length" class="model-signals-tags">
                    <a-tag v-for="reason in record.drift.reasons" :key="reason" :color="severityTagColor(record.drift?.severity)">
                      {{ reason }}
                    </a-tag>
                  </div>
                  <span v-else class="muted">{{ t('fallback.noReason') }}</span>
                </template>
              </template>
            </a-table>
          </section>
        </div>

        <div v-else-if="activeTab === 'matrix'">
          <section class="panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">{{ t('matrix.title') }}</h2>
                <div class="panel-kicker">{{ t('matrix.kicker') }}</div>
              </div>
              <TableOutlined class="panel-header-icon" />
            </div>
            <a-table
              class="dense-table model-signals-matrix-table"
              :columns="matrixColumns"
              :data-source="matrixRows"
              :loading="loading"
              :locale="matrixTableLocale"
              :pagination="false"
              :row-key="matrixRowKey"
              size="small"
              :scroll="{ x: 980 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'source'">
                  <span class="source-identity-name">{{ sourceInfo(record).label }}</span>
                  <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
                </template>
                <template v-else-if="column.key === 'models'">
                  <div class="model-signals-matrix-cells">
                    <a-tooltip
                      v-for="cell in record.cells"
                      :key="matrixCellKey(cell)"
                      :title="matrixCellTitle(cell)"
                      placement="topLeft"
                    >
                      <div class="model-signals-matrix-cell" :class="severityClass(cell.severity)">
                        <div class="model-signals-matrix-cell-head">
                          <span class="model-name">{{ cell.model || t('fallback.unknown') }}</span>
                          <a-tag class="status-tag" :color="severityTagColor(cell.severity)">
                            {{ severityLabel(cell.severity) }}
                          </a-tag>
                        </div>
                        <div class="source-identity-meta">{{ cell.modelProvider || '-' }} · {{ formatNumber(cell.cohortCount) }} {{ t('tab.cohorts') }}</div>
                        <div class="model-signals-matrix-metrics">
                          <span>{{ formatLatency(cell.current?.modelLatencyMsPer1kOutputTokens) }}</span>
                          <span>{{ formatRate(cell.current?.modelThroughputTokensPerSecond, 1) }} tok/s</span>
                          <span>{{ formatConfidence(cell.confidence) }}</span>
                        </div>
                        <div class="model-signals-matrix-reason">{{ cell.keyReason || t('fallback.noReason') }}</div>
                      </div>
                    </a-tooltip>
                  </div>
                </template>
              </template>
            </a-table>
          </section>
        </div>

        <div v-else-if="activeTab === 'projects'">
          <div class="section-stack">
            <ModelSignalsMetricChart
              :daily-rows="dailyMetricRows"
              :project-rows="projectRows"
              :loading="loading"
              initial-mode="projects"
              :allow-mode-switch="false"
            />

            <section class="panel">
              <div class="panel-header">
                <div>
                  <h2 class="panel-title">{{ t('projects.title') }}</h2>
                  <div class="panel-kicker">{{ t('projects.kicker') }}</div>
                </div>
                <DashboardOutlined class="panel-header-icon" />
              </div>
              <a-table
                class="dense-table model-signals-project-table"
                :columns="projectColumns"
                :data-source="projectRows"
                :loading="loading"
                :locale="projectTableLocale"
                :pagination="{ pageSize: 10 }"
                :row-key="projectRowKey"
                size="small"
                :custom-row="driftRowClass"
                :scroll="{ x: 1580 }"
              >
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'project'">
                    <a-tooltip :title="projectInfo(record).full" placement="topLeft">
                      <span class="model-signals-project">{{ projectInfo(record).main }}</span>
                    </a-tooltip>
                    <div class="source-identity-meta">{{ projectInfo(record).full }}</div>
                  </template>
                  <template v-else-if="column.key === 'sessions'">
                    <span v-if="!hasProjectMetrics" class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
                    <div v-else class="metric-comparison">
                      <span>{{ formatNumber(record.sessionCount) }}</span>
                      <span>{{ formatNumber(record.totalTokens) }} {{ t('column.tokens') }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'sources'"><span class="number-cell">{{ formatNumber(record.sourceCount) }}</span></template>
                  <template v-else-if="column.key === 'models'"><span class="number-cell">{{ formatNumber(record.modelCount) }}</span></template>
                  <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.totalTokens) }}</span></template>
                  <template v-else-if="column.key === 'mix'">
                    <a-tooltip :title="projectMixInfo(record).full" placement="topLeft">
                      <div class="model-signals-mix-cell">
                        <span class="model-name">{{ projectMixInfo(record).model }}</span>
                        <span class="source-identity-meta">{{ projectMixInfo(record).summary || projectMixInfo(record).provider || '-' }}</span>
                      </div>
                    </a-tooltip>
                  </template>
                  <template v-else-if="column.key === 'costBurn'">
                    <div class="metric-comparison">
                      <span>{{ formatCost(record.current?.estimatedCostUsd) }}</span>
                      <span>{{ unpricedNote(record.current) || formatOptionalCost(record.current?.costPerActiveHour) }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'cacheSavings'">
                    <span class="number-cell status-ok">{{ formatOptionalCost(record.current?.cacheSavingsUsd) }}</span>
                  </template>
                  <template v-else-if="column.key === 'health'">
                    <a-tooltip :title="projectHealthTitle(record)" placement="topLeft">
                      <div class="model-signals-health-cell">
                        <a-tag class="status-tag" :color="severityTagColor(record.drift?.severity)">
                          {{ severityLabel(record.drift?.severity) }}
                        </a-tag>
                        <span class="source-identity-meta">{{ formatConfidence(record.drift?.confidence) }}</span>
                      </div>
                    </a-tooltip>
                  </template>
                  <template v-else-if="column.key === 'latency'">
                    <div class="metric-comparison" :class="metricClass(p90Latency(record.current), p90Latency(record.baseline), true)">
                      <span>{{ formatLatency(p90Latency(record.current)) }}</span>
                      <span>{{ t('metric.baseline') }} {{ formatLatency(p90Latency(record.baseline)) }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'throughput'">
                    <div class="metric-comparison" :class="metricClass(p10Throughput(record.current), p10Throughput(record.baseline))">
                      <span>{{ formatThroughput(p10Throughput(record.current)) }}</span>
                      <span>{{ t('metric.baseline') }} {{ formatThroughput(p10Throughput(record.baseline)) }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'pressure'">
                    <div class="metric-comparison" :class="metricClass(failurePressure(record.current), failurePressure(record.baseline), true)">
                      <span>{{ formatPressure(failurePressure(record.current)) }}</span>
                      <span>{{ formatRate(record.current?.avgModelCallsPerSession, 2) }}/session</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'severity'">
                    <a-tag class="status-tag" :color="severityTagColor(record.drift?.severity)">
                      {{ severityLabel(record.drift?.severity) }}
                    </a-tag>
                  </template>
                  <template v-else-if="column.key === 'confidence'">
                    <span class="number-cell">{{ formatConfidence(record.drift?.confidence) }}</span>
                    <div v-if="record.drift?.sampleNote" class="source-identity-meta">{{ record.drift.sampleNote }}</div>
                  </template>
                  <template v-else-if="column.key === 'reasons'">
                    <div v-if="record.drift?.reasons?.length" class="model-signals-tags">
                      <a-tag v-for="reason in record.drift.reasons" :key="reason" :color="severityTagColor(record.drift?.severity)">
                        {{ reason }}
                      </a-tag>
                    </div>
                    <span v-else class="muted">{{ t('fallback.noReason') }}</span>
                  </template>
                </template>
              </a-table>
            </section>
          </div>
        </div>

        <div v-else-if="activeTab === 'anomalies'">
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

.model-signals-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  padding: 8px;
  background: var(--am-surface);
  border: 1px solid var(--am-border);
  border-radius: var(--am-radius);
  box-shadow: var(--am-shadow);
}

.model-signals-tabs .ant-btn {
  min-width: 126px;
  justify-content: center;
}

.model-signals-metric-strip {
  grid-template-columns: repeat(6, minmax(136px, 1fr));
}

.metric-danger {
  --metric-accent: var(--am-danger);
  --metric-soft: var(--am-danger-soft);
}

.model-signals-reason-strip {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 9px 12px;
  background: var(--am-surface);
  border: 1px solid var(--am-border);
  border-radius: var(--am-radius);
}

.model-signals-session,
.model-signals-date,
.model-signals-project {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: var(--am-text);
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.model-signals-date {
  color: var(--am-text);
  font-variant-numeric: tabular-nums;
}

.model-signals-project + .source-identity-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-confidence-cell,
.model-signals-health-cell,
.model-signals-mix-cell {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.model-signals-confidence-cell {
  grid-template-columns: auto auto minmax(0, 1fr);
  align-items: center;
}

.model-signals-health-cell {
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
}

.model-signals-mix-cell .model-name,
.model-signals-reason-text {
  display: block;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-mix-cell .source-identity-meta,
.model-signals-health-cell .source-identity-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.model-signals-tags .ant-tag {
  max-width: 100%;
  margin-right: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.metric-comparison {
  display: grid;
  justify-items: end;
  gap: 1px;
  min-width: 0;
  font-variant-numeric: tabular-nums;
}

.metric-comparison > span:first-child {
  color: inherit;
  font-weight: 650;
}

.metric-comparison > span:last-child {
  max-width: 100%;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 11px;
  line-height: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-matrix-cells {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.model-signals-matrix-cell {
  width: 244px;
  min-width: 0;
  padding: 8px 9px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-left: 3px solid var(--am-success);
  border-radius: var(--am-radius-sm);
}

.model-signals-matrix-cell.severity-watch {
  border-left-color: var(--am-info);
}

.model-signals-matrix-cell.severity-warning {
  border-left-color: var(--am-warning);
  background: var(--am-warning-soft);
}

.model-signals-matrix-cell.severity-critical {
  border-left-color: var(--am-danger);
  background: var(--am-danger-soft);
}

.model-signals-matrix-cell-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}

.model-signals-matrix-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  margin-top: 6px;
  color: var(--am-text-soft);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  line-height: 15px;
}

.model-signals-matrix-metrics span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-matrix-reason {
  margin-top: 4px;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 11px;
  line-height: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.model-signals-warning-row td) {
  background: rgba(254, 243, 199, 0.36);
}

:deep(.model-signals-critical-row td) {
  background: rgba(254, 226, 226, 0.46);
}

@media (max-width: 1420px) {
  .model-signals-metric-strip {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .model-signals-metric-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .model-signals-tabs .ant-btn {
    min-width: 112px;
  }

  .model-signals-reason-strip {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 640px) {
  .model-signals-tabs {
    flex-wrap: nowrap;
    overflow-x: auto;
  }

  .model-signals-tabs .ant-btn {
    flex: 0 0 auto;
  }

  .model-signals-metric-strip {
    grid-template-columns: 1fr;
  }
}
</style>
