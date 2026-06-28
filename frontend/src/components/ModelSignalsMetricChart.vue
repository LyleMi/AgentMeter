<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import ASegmented from 'ant-design-vue/es/segmented'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import { BarChartOutlined, LineChartOutlined } from '@ant-design/icons-vue'
import {
  formatCost,
  formatNumber,
  projectDisplay,
  type ModelSignalMetricSet,
  type ModelSignalProjectHotspot,
  type ModelSignalsDailyMetric,
  type ModelSignalsProjectMetric
} from '../api'
import { chartPalette } from '../chartPalette'
import { useEChart } from '../composables/useEChart'
import { useMessages } from '../i18n'

type ChartMode = 'daily' | 'projects'
type ChartKind = 'bar' | 'line'
type MetricKind = 'cost' | 'latency' | 'throughput' | 'percent' | 'pressure' | 'ratio'
type MetricGroupKey = 'performance' | 'cost' | 'pressure' | 'shape'
type MetricWindow = 'current' | 'baseline' | 'total'
type MetricKey =
  | 'p90Latency'
  | 'p50Latency'
  | 'p10Throughput'
  | 'outputThroughput'
  | 'costBurn'
  | 'costPerSession'
  | 'costPerActiveHour'
  | 'costPer1kTokens'
  | 'cacheSavings'
  | 'failurePressure'
  | 'retryPressure'
  | 'modelFailureRate'
  | 'toolFailureRate'
  | 'cacheMiss'
  | 'reasoningShare'
  | 'outputExpansion'
  | 'toolDependency'
type ProjectChartRow = ModelSignalsProjectMetric | ModelSignalProjectHotspot
type ChartRow = ModelSignalsDailyMetric | ProjectChartRow

interface MetricDefinition {
  key: MetricKey
  label: string
  description: string
  group: MetricGroupKey
  kind: MetricKind
  color: string
  chart: ChartKind
  lowerIsBetter: boolean
  value: (metric?: ModelSignalMetricSet) => number | undefined
}

interface MetricGroup {
  key: MetricGroupKey
  label: string
}

const props = withDefaults(
  defineProps<{
    dailyRows?: ModelSignalsDailyMetric[]
    projectRows?: ProjectChartRow[]
    loading?: boolean
    initialMode?: ChartMode
    allowModeSwitch?: boolean
  }>(),
  {
    dailyRows: () => [],
    projectRows: () => [],
    loading: false,
    initialMode: 'daily',
    allowModeSwitch: true
  }
)

const selectedMode = ref<ChartMode>(props.initialMode)
const selectedMetricKeys = ref<MetricKey[]>(defaultMetricsForMode(props.initialMode))
const showBaselineComparison = ref(false)
const { chartEl, getChart, disposeChart } = useEChart()
const { t, locale } = useMessages({
  en: {
    'title.daily': 'Daily Signal Lens',
    'title.projects': 'Project Signal Lens',
    'kicker.daily': 'Compare service speed, failure pressure, cost, and token shape before inspecting rows',
    'kicker.projects': 'Scan project hotspots across the signals that matter for the current investigation',
    'mode.daily': 'Time',
    'mode.projects': 'Projects',
    'control.mode': 'Chart view',
    'control.metrics': 'Metrics',
    'control.baseline': 'Compare baseline',
    'control.baselineUnavailable': 'No baseline values for the selected metrics',
    'direction.lower': 'lower is better',
    'direction.higher': 'higher is better',
    'direction.mixed': 'mixed directions',
    'selection.count': '{count} metrics',
    'series.current': 'Current',
    'series.baseline': 'Baseline',
    'series.lowSample': 'Low sample',
    'axis.cost': 'cost',
    'axis.latency': 'latency',
    'axis.throughput': 'throughput',
    'axis.percent': 'rate',
    'axis.pressure': 'pressure',
    'axis.ratio': 'ratio',
    'axis.relative': 'relative scale',
    'group.performance': 'Performance',
    'group.cost': 'Cost',
    'group.pressure': 'Pressure',
    'group.shape': 'Token shape',
    'metric.p90Latency': 'P90 latency',
    'metric.p90LatencyDesc': 'Tail model latency per 1k output tokens',
    'metric.p50Latency': 'P50 latency',
    'metric.p50LatencyDesc': 'Typical model latency per 1k output tokens',
    'metric.p10Throughput': 'P10 throughput',
    'metric.p10ThroughputDesc': 'Slow-floor observed total-token throughput',
    'metric.outputThroughput': 'Output throughput',
    'metric.outputThroughputDesc': 'Observed output-token throughput',
    'metric.costBurn': 'Cost burn',
    'metric.costBurnDesc': 'Observed estimated cost in the row',
    'metric.costPerSession': 'Cost / session',
    'metric.costPerSessionDesc': 'Estimated cost normalized by session count',
    'metric.costPerActiveHour': 'Cost / active hour',
    'metric.costPerActiveHourDesc': 'Estimated cost normalized by active time',
    'metric.costPer1kTokens': 'Cost / 1k tokens',
    'metric.costPer1kTokensDesc': 'Estimated cost normalized by total token volume',
    'metric.cacheSavings': 'Cache savings',
    'metric.cacheSavingsDesc': 'Estimated avoided cost from cached input tokens',
    'metric.failurePressure': 'Failure pressure',
    'metric.failurePressureDesc': 'Failed model and tool calls per session',
    'metric.retryPressure': 'Retry pressure',
    'metric.retryPressureDesc': 'Model calls per session as a repair-loop proxy',
    'metric.modelFailureRate': 'Model failure rate',
    'metric.modelFailureRateDesc': 'Failed model calls divided by model calls',
    'metric.toolFailureRate': 'Tool failure rate',
    'metric.toolFailureRateDesc': 'Failed tool calls divided by tool calls',
    'metric.cacheMiss': 'Cache miss rate',
    'metric.cacheMissDesc': 'Uncached input share',
    'metric.reasoningShare': 'Reasoning share',
    'metric.reasoningShareDesc': 'Reasoning tokens divided by output tokens',
    'metric.outputExpansion': 'Output expansion',
    'metric.outputExpansionDesc': 'Output tokens divided by input tokens',
    'metric.toolDependency': 'Tool dependency',
    'metric.toolDependencyDesc': 'Sessions with tool calls divided by sessions',
    'tooltip.sessions': 'Sessions',
    'tooltip.modelCalls': 'Model calls',
    'tooltip.tokens': 'Tokens',
    'tooltip.confidence': 'Confidence',
    'tooltip.reason': 'Reason',
    'tooltip.unavailable': 'Unavailable',
    'empty.title': 'No chartable signal values',
    'empty.text': 'Select another metric or broaden the source, model, project, or date scope.',
    'fallback.unknown': 'unknown',
    'fallback.noReason': 'No drift reason'
  },
  'zh-CN': {
    'title.daily': '每日信号镜头',
    'title.projects': '项目信号镜头',
    'kicker.daily': '先对比服务速度、失败压力、费用和 token 形态，再进入明细行',
    'kicker.projects': '按当前排查关心的信号扫描项目热点',
    'mode.daily': '时间',
    'mode.projects': '项目',
    'control.mode': '图表视图',
    'control.metrics': '指标',
    'control.baseline': '对比基线',
    'control.baselineUnavailable': '所选指标没有可用基线值',
    'direction.lower': '越低越好',
    'direction.higher': '越高越好',
    'direction.mixed': '方向混合',
    'selection.count': '{count} 个指标',
    'series.current': '当前',
    'series.baseline': '基线',
    'series.lowSample': '低样本',
    'axis.cost': '费用',
    'axis.latency': '延迟',
    'axis.throughput': '吞吐',
    'axis.percent': '比例',
    'axis.pressure': '压力',
    'axis.ratio': '倍率',
    'axis.relative': '相对刻度',
    'group.performance': '性能',
    'group.cost': '费用',
    'group.pressure': '压力',
    'group.shape': 'Token 形态',
    'metric.p90Latency': 'P90 延迟',
    'metric.p90LatencyDesc': '按 1k 输出 token 归一化的尾部模型延迟',
    'metric.p50Latency': 'P50 延迟',
    'metric.p50LatencyDesc': '按 1k 输出 token 归一化的典型模型延迟',
    'metric.p10Throughput': 'P10 吞吐',
    'metric.p10ThroughputDesc': '低谷观测总 token 吞吐',
    'metric.outputThroughput': '输出吞吐',
    'metric.outputThroughputDesc': '观测输出 token 吞吐',
    'metric.costBurn': '费用消耗',
    'metric.costBurnDesc': '当前行的观测估算费用',
    'metric.costPerSession': '每会话费用',
    'metric.costPerSessionDesc': '按会话数归一化的估算费用',
    'metric.costPerActiveHour': '每活跃小时费用',
    'metric.costPerActiveHourDesc': '按活跃时间归一化的估算费用',
    'metric.costPer1kTokens': '每 1k token 费用',
    'metric.costPer1kTokensDesc': '按总 token 量归一化的估算费用',
    'metric.cacheSavings': '缓存节省',
    'metric.cacheSavingsDesc': '缓存输入 token 带来的估算节省',
    'metric.failurePressure': '失败压力',
    'metric.failurePressureDesc': '每个会话的失败模型与工具调用压力',
    'metric.retryPressure': '重试压力',
    'metric.retryPressureDesc': '用每会话模型调用数代理修复循环',
    'metric.modelFailureRate': '模型失败率',
    'metric.modelFailureRateDesc': '失败模型调用占模型调用的比例',
    'metric.toolFailureRate': '工具失败率',
    'metric.toolFailureRateDesc': '失败工具调用占工具调用的比例',
    'metric.cacheMiss': '缓存未命中率',
    'metric.cacheMissDesc': '未缓存输入占比',
    'metric.reasoningShare': '推理占比',
    'metric.reasoningShareDesc': '推理 token 占输出 token 的比例',
    'metric.outputExpansion': '输出扩张',
    'metric.outputExpansionDesc': '输出 token 与输入 token 的比例',
    'metric.toolDependency': '工具依赖',
    'metric.toolDependencyDesc': '使用工具的会话占比',
    'tooltip.sessions': '会话',
    'tooltip.modelCalls': '模型调用',
    'tooltip.tokens': 'Token',
    'tooltip.confidence': '置信度',
    'tooltip.reason': '原因',
    'tooltip.unavailable': '不可用',
    'empty.title': '没有可绘制的信号值',
    'empty.text': '可以选择其他指标，或放宽来源、模型、项目、日期范围。',
    'fallback.unknown': '未知',
    'fallback.noReason': '无漂移原因'
  }
})

const metricGroups = computed<MetricGroup[]>(() => [
  { key: 'performance', label: t('group.performance') },
  { key: 'cost', label: t('group.cost') },
  { key: 'pressure', label: t('group.pressure') },
  { key: 'shape', label: t('group.shape') }
])

const metricDefinitions = computed<MetricDefinition[]>(() => [
  {
    key: 'p90Latency',
    label: t('metric.p90Latency'),
    description: t('metric.p90LatencyDesc'),
    group: 'performance',
    kind: 'latency',
    color: chartPalette.danger,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => firstFinite(metric?.p90ModelLatencyMsPer1kOutputTokens, metric?.modelLatencyMsPer1kOutputTokens)
  },
  {
    key: 'p50Latency',
    label: t('metric.p50Latency'),
    description: t('metric.p50LatencyDesc'),
    group: 'performance',
    kind: 'latency',
    color: '#ea580c',
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => firstFinite(metric?.p50ModelLatencyMsPer1kOutputTokens, metric?.modelLatencyMsPer1kOutputTokens)
  },
  {
    key: 'p10Throughput',
    label: t('metric.p10Throughput'),
    description: t('metric.p10ThroughputDesc'),
    group: 'performance',
    kind: 'throughput',
    color: chartPalette.success,
    chart: 'line',
    lowerIsBetter: false,
    value: (metric) => firstFinite(metric?.p10ModelThroughputTokensPerSecond, metric?.modelThroughputTokensPerSecond)
  },
  {
    key: 'outputThroughput',
    label: t('metric.outputThroughput'),
    description: t('metric.outputThroughputDesc'),
    group: 'performance',
    kind: 'throughput',
    color: chartPalette.info,
    chart: 'line',
    lowerIsBetter: false,
    value: (metric) => finiteNumber(metric?.modelThroughputOutputTokensPerSecond)
  },
  {
    key: 'costBurn',
    label: t('metric.costBurn'),
    description: t('metric.costBurnDesc'),
    group: 'cost',
    kind: 'cost',
    color: chartPalette.primary,
    chart: 'bar',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.estimatedCostUsd)
  },
  {
    key: 'costPerSession',
    label: t('metric.costPerSession'),
    description: t('metric.costPerSessionDesc'),
    group: 'cost',
    kind: 'cost',
    color: '#7c3aed',
    chart: 'bar',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.costPerSession)
  },
  {
    key: 'costPerActiveHour',
    label: t('metric.costPerActiveHour'),
    description: t('metric.costPerActiveHourDesc'),
    group: 'cost',
    kind: 'cost',
    color: chartPalette.indigo,
    chart: 'bar',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.costPerActiveHour)
  },
  {
    key: 'costPer1kTokens',
    label: t('metric.costPer1kTokens'),
    description: t('metric.costPer1kTokensDesc'),
    group: 'cost',
    kind: 'cost',
    color: chartPalette.sky,
    chart: 'bar',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.costPer1kTokens)
  },
  {
    key: 'cacheSavings',
    label: t('metric.cacheSavings'),
    description: t('metric.cacheSavingsDesc'),
    group: 'cost',
    kind: 'cost',
    color: '#059669',
    chart: 'bar',
    lowerIsBetter: false,
    value: (metric) => finiteNumber(metric?.cacheSavingsUsd)
  },
  {
    key: 'failurePressure',
    label: t('metric.failurePressure'),
    description: t('metric.failurePressureDesc'),
    group: 'pressure',
    kind: 'pressure',
    color: chartPalette.warning,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.failurePressure)
  },
  {
    key: 'retryPressure',
    label: t('metric.retryPressure'),
    description: t('metric.retryPressureDesc'),
    group: 'pressure',
    kind: 'pressure',
    color: '#9333ea',
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.avgModelCallsPerSession)
  },
  {
    key: 'modelFailureRate',
    label: t('metric.modelFailureRate'),
    description: t('metric.modelFailureRateDesc'),
    group: 'pressure',
    kind: 'percent',
    color: chartPalette.danger,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => safeRate(metric?.failedModelCalls, metric?.modelCalls)
  },
  {
    key: 'toolFailureRate',
    label: t('metric.toolFailureRate'),
    description: t('metric.toolFailureRateDesc'),
    group: 'pressure',
    kind: 'percent',
    color: '#c2410c',
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.toolFailureRate)
  },
  {
    key: 'cacheMiss',
    label: t('metric.cacheMiss'),
    description: t('metric.cacheMissDesc'),
    group: 'shape',
    kind: 'percent',
    color: '#d97706',
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.cacheMissRate)
  },
  {
    key: 'reasoningShare',
    label: t('metric.reasoningShare'),
    description: t('metric.reasoningShareDesc'),
    group: 'shape',
    kind: 'percent',
    color: chartPalette.indigo,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.reasoningTokenShare)
  },
  {
    key: 'outputExpansion',
    label: t('metric.outputExpansion'),
    description: t('metric.outputExpansionDesc'),
    group: 'shape',
    kind: 'ratio',
    color: chartPalette.primary,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.outputExpansionRate)
  },
  {
    key: 'toolDependency',
    label: t('metric.toolDependency'),
    description: t('metric.toolDependencyDesc'),
    group: 'shape',
    kind: 'percent',
    color: chartPalette.axis,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.toolDependencyRate)
  }
])

const selectedMetrics = computed(() => {
  const selected = selectedMetricKeys.value
    .map((key) => metricDefinitions.value.find((item) => item.key === key))
    .filter((item): item is MetricDefinition => Boolean(item))
  return selected.length ? selected : metricDefinitions.value.slice(0, 1)
})
const primaryMetric = computed(() => selectedMetrics.value[0] || metricDefinitions.value[0])
const modeOptions = computed(() => [
  { label: t('mode.daily'), value: 'daily' },
  { label: t('mode.projects'), value: 'projects' }
])
const chartTitle = computed(() => selectedMode.value === 'projects' ? t('title.projects') : t('title.daily'))
const chartKicker = computed(() => selectedMode.value === 'projects' ? t('kicker.projects') : t('kicker.daily'))
const directionLabel = computed(() => {
  const lower = selectedMetrics.value.filter((metric) => metric.lowerIsBetter).length
  if (lower === selectedMetrics.value.length) return t('direction.lower')
  if (lower === 0) return t('direction.higher')
  return t('direction.mixed')
})
const selectionCountLabel = computed(() => t('selection.count', { count: selectedMetrics.value.length }))
const plottedRows = computed<ChartRow[]>(() => {
  if (selectedMode.value === 'daily') {
    return [...props.dailyRows].sort((left, right) => left.date.localeCompare(right.date))
  }
  return [...props.projectRows]
    .sort((left, right) => metricSortValue(right) - metricSortValue(left))
    .slice(0, 10)
})
const hasChart = computed(() =>
  selectedMetrics.value.some((metric) =>
    plottedRows.value.some((row) => metricValueForRow(row, metric, 'current') !== undefined)
  )
)
const canCompareBaseline = computed(() =>
  selectedMetrics.value.some((metric) =>
    plottedRows.value.some((row) => metricValueForRow(row, metric, 'baseline') !== undefined)
  )
)
const activeMetricKinds = computed<MetricKind[]>(() => {
  const keys: MetricKind[] = []
  selectedMetrics.value.forEach((metric) => {
    if (!keys.includes(metric.kind)) keys.push(metric.kind)
  })
  return keys
})
const normalizeProjectScale = computed(() => selectedMode.value === 'projects' && activeMetricKinds.value.length > 1)

watch(() => props.initialMode, (mode) => {
  selectedMode.value = mode
  if (!selectedMetricKeys.value.length) selectedMetricKeys.value = defaultMetricsForMode(mode)
})

watch(selectedMode, (mode, previous) => {
  if (mode !== previous && selectedMetricKeys.value.length === 0) {
    selectedMetricKeys.value = defaultMetricsForMode(mode)
  }
})

watch(canCompareBaseline, (available) => {
  if (!available) showBaselineComparison.value = false
})

watch(() => [selectedMode.value, selectedMetricKeys.value, showBaselineComparison.value, props.dailyRows, props.projectRows, locale.value], renderAfterUpdate, { deep: true })

onMounted(() => {
  renderAfterUpdate()
})

async function renderAfterUpdate() {
  await nextTick()
  renderChart()
}

function renderChart() {
  if (!hasChart.value) {
    disposeChart()
    return
  }
  if (selectedMode.value === 'projects') {
    renderProjectChart()
  } else {
    renderDailyChart()
  }
}

function renderDailyChart() {
  const rows = plottedRows.value as ModelSignalsDailyMetric[]
  const chart = getChart()
  if (!chart) return

  chart.setOption({
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'cross', lineStyle: { color: chartPalette.pointer }, shadowStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => dailyTooltipMarkup(params)
    },
    grid: dailyGrid(),
    legend: legendOptions(),
    xAxis: {
      type: 'category',
      data: rows.map((row) => row.date.slice(5)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.axis, fontSize: 11 }
    },
    yAxis: activeMetricKinds.value.map((kind, index) => valueAxisOptions(kind, index)),
    series: [
      ...selectedMetrics.value.flatMap((metric) => dailyMetricSeries(metric, rows)),
      lowSampleSeries(rows)
    ].filter(Boolean)
  }, true)
}

function renderProjectChart() {
  const rows = plottedRows.value as ProjectChartRow[]
  const chart = getChart()
  if (!chart) return
  const normalized = normalizeProjectScale.value
  const axisKind = activeMetricKinds.value[0] || primaryMetric.value.kind

  chart.setOption({
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'shadow', shadowStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => projectTooltipMarkup(params)
    },
    grid: { left: 128, right: 42, top: 42, bottom: 34 },
    legend: legendOptions(),
    xAxis: {
      type: 'value',
      name: normalized ? t('axis.relative') : axisName(axisKind),
      nameTextStyle: { color: chartPalette.axis, fontSize: 11, padding: [0, 0, 0, 4] },
      axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: (value: number) => normalized ? `${Math.round(value)}%` : axisLabelForKind(axisKind, value) },
      splitLine: { lineStyle: { color: chartPalette.grid } }
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: rows.map((row) => projectInfo(row).main),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.text, fontSize: 11, overflow: 'truncate', width: 110 }
    },
    series: selectedMetrics.value.flatMap((metric) => projectMetricSeries(metric, rows, normalized))
  }, true)
}

function dailyMetricSeries(metric: MetricDefinition, rows: ModelSignalsDailyMetric[]) {
  const series: Array<Record<string, unknown>> = [{
    name: metric.label,
    type: metric.chart,
    yAxisIndex: metricAxisIndex(metric),
    smooth: metric.chart === 'line',
    barMaxWidth: 16,
    symbolSize: 7,
    lineStyle: { width: 2, color: metric.color },
    itemStyle: { color: metric.color, borderRadius: metric.chart === 'bar' ? [3, 3, 0, 0] : 0 },
    emphasis: { focus: 'series' },
    data: rows.map((row) => valueOrNull(metricValueForRow(row, metric, 'current')))
  }]
  if (showBaselineComparison.value) {
    series.push({
      name: baselineSeriesName(metric),
      type: metric.chart,
      yAxisIndex: metricAxisIndex(metric),
      smooth: metric.chart === 'line',
      barMaxWidth: 12,
      symbolSize: 5,
      lineStyle: { width: 2, type: 'dashed', color: metric.color, opacity: 0.5 },
      itemStyle: { color: metric.color, opacity: 0.28, borderRadius: metric.chart === 'bar' ? [3, 3, 0, 0] : 0 },
      emphasis: { focus: 'series' },
      data: rows.map((row) => valueOrNull(metricValueForRow(row, metric, 'baseline')))
    })
  }
  return series
}

function lowSampleSeries(rows: ModelSignalsDailyMetric[]) {
  const metric = primaryMetric.value
  return {
    name: t('series.lowSample'),
    type: 'scatter',
    yAxisIndex: metricAxisIndex(metric),
    symbol: 'diamond',
    symbolSize: 11,
    itemStyle: { color: chartPalette.warning },
    data: rows.map((row) => row.lowSample ? valueOrNull(metricValueForRow(row, metric, 'current')) : null)
  }
}

function projectMetricSeries(metric: MetricDefinition, rows: ProjectChartRow[], normalized: boolean) {
  const currentSeries: Record<string, unknown> = {
    name: metric.label,
    type: 'bar',
    barMaxWidth: 12,
    itemStyle: { color: metric.color, borderRadius: [0, 3, 3, 0] },
    emphasis: { focus: 'series' },
    data: rows.map((row) => valueOrNull(projectPlotValue(row, metric, 'current', normalized)))
  }
  if (!showBaselineComparison.value) return [currentSeries]

  return [
    currentSeries,
    {
      name: baselineSeriesName(metric),
      type: 'bar',
      barMaxWidth: 10,
      itemStyle: { color: metric.color, opacity: 0.28, borderRadius: [0, 3, 3, 0] },
      emphasis: { focus: 'series' },
      data: rows.map((row) => valueOrNull(projectPlotValue(row, metric, 'baseline', normalized)))
    }
  ]
}

function dailyTooltipMarkup(params: unknown) {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as { dataIndex?: number; axisValue?: string } | undefined
  const row = plottedRows.value[first?.dataIndex ?? 0] as ModelSignalsDailyMetric | undefined
  if (!row) return ''
  return [
    `<strong>${escapeHtml(row.date || first?.axisValue || '')}</strong>`,
    ...selectedMetrics.value.map((metric) => metricTooltipLine(row, metric)),
    `<div>${t('tooltip.sessions')}: ${formatNumber(row.sessionCount)}</div>`,
    `<div>${t('tooltip.modelCalls')}: ${formatNumber(row.modelCalls)}</div>`,
    `<div>${t('tooltip.tokens')}: ${formatNumber(row.totalTokens)}</div>`,
    `<div>${t('tooltip.confidence')}: ${escapeHtml(row.drift?.confidence || t('fallback.unknown'))}</div>`,
    `<div>${t('tooltip.reason')}: ${escapeHtml(row.keyReason || row.drift?.sampleNote || row.drift?.reasons?.[0] || t('fallback.noReason'))}</div>`
  ].join('')
}

function projectTooltipMarkup(params: unknown) {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as { dataIndex?: number; axisValue?: string } | undefined
  const row = plottedRows.value[first?.dataIndex ?? 0] as ProjectChartRow | undefined
  if (!row) return ''
  const info = projectInfo(row)
  const current = projectMetricSet(row, 'current')
  const drift = row.drift
  return [
    `<strong>${escapeHtml(info.full || first?.axisValue || '')}</strong>`,
    ...selectedMetrics.value.map((metric) => metricTooltipLine(row, metric)),
    `<div>${t('tooltip.sessions')}: ${formatNumber(current?.sessionCount || row.sessionCount)}</div>`,
    `<div>${t('tooltip.tokens')}: ${formatNumber(current?.totalTokens || row.totalTokens)}</div>`,
    `<div>${t('tooltip.confidence')}: ${escapeHtml(drift?.confidence || t('fallback.unknown'))}</div>`,
    `<div>${t('tooltip.reason')}: ${escapeHtml(drift?.reasons?.[0] || drift?.sampleNote || t('fallback.noReason'))}</div>`
  ].join('')
}

function metricTooltipLine(row: ChartRow, metric: MetricDefinition) {
  const currentValue = metricValueForRow(row, metric, 'current')
  const baselineValue = metricValueForRow(row, metric, 'baseline')
  const baseline = showBaselineComparison.value
    ? ` / ${t('series.baseline')} ${escapeHtml(formatMetricValue(metric, baselineValue))}`
    : ''
  return `<div><span style="color:${metric.color}">●</span> ${escapeHtml(metric.label)}: ${escapeHtml(formatMetricValue(metric, currentValue))}${baseline}</div>`
}

function metricSortValue(row: ProjectChartRow) {
  return metricValueForProject(row, primaryMetric.value, 'current') ?? metricValueForProject(row, primaryMetric.value, 'total') ?? -1
}

function metricValueForRow(row: ChartRow, metric: MetricDefinition, window: MetricWindow = 'current') {
  if (selectedMode.value === 'projects') return metricValueForProject(row as ProjectChartRow, metric, window)
  const dailyRow = row as ModelSignalsDailyMetric
  if (window === 'baseline') return hasMetricSetSamples(dailyRow.baseline) ? metric.value(dailyRow.baseline) : undefined
  return metric.value(dailyRow)
}

function metricValueForProject(row: ProjectChartRow, metric: MetricDefinition, window: MetricWindow) {
  const set = projectMetricSet(row, window)
  if (window === 'baseline' && !hasMetricSetSamples(set)) return undefined
  return metric.value(set)
}

function projectMetricSet(row: ProjectChartRow, window: MetricWindow): ModelSignalMetricSet | undefined {
  if (window === 'current') return row.current || row
  if (window === 'baseline') return row.baseline
  return row
}

function projectPlotValue(row: ProjectChartRow, metric: MetricDefinition, window: MetricWindow, normalized: boolean) {
  const value = metricValueForProject(row, metric, window)
  if (value === undefined || !normalized) return value
  const max = projectMetricMax(metric)
  if (max <= 0) return 0
  return value / max * 100
}

function projectMetricMax(metric: MetricDefinition) {
  const values: number[] = []
  ;(plottedRows.value as ProjectChartRow[]).forEach((row) => {
    const current = metricValueForProject(row, metric, 'current')
    if (current !== undefined) values.push(Math.abs(current))
    if (showBaselineComparison.value) {
      const baseline = metricValueForProject(row, metric, 'baseline')
      if (baseline !== undefined) values.push(Math.abs(baseline))
    }
  })
  return Math.max(...values, 0)
}

function metricAxisIndex(metric: MetricDefinition) {
  const index = activeMetricKinds.value.indexOf(metric.kind)
  return index >= 0 ? index : 0
}

function valueAxisOptions(kind: MetricKind, index: number) {
  const position = index % 2 === 0 ? 'left' : 'right'
  const sameSideOffset = Math.floor(index / 2) * 54
  return {
    type: 'value',
    name: axisName(kind),
    position,
    offset: sameSideOffset,
    nameTextStyle: { color: chartPalette.axis, fontSize: 11, padding: [0, 0, 0, 4] },
    axisLine: { show: index > 0, lineStyle: { color: chartPalette.border } },
    axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: (value: number) => axisLabelForKind(kind, value) },
    splitLine: { show: index === 0, lineStyle: { color: chartPalette.grid } }
  }
}

function dailyGrid() {
  const leftCount = activeMetricKinds.value.filter((_, index) => index % 2 === 0).length
  const rightCount = activeMetricKinds.value.length - leftCount
  return {
    left: 64 + Math.max(0, leftCount - 1) * 54,
    right: 34 + Math.max(0, rightCount - 1) * 58,
    top: 42,
    bottom: 42
  }
}

function legendOptions() {
  return {
    show: true,
    top: 0,
    right: 8,
    type: 'scroll',
    itemGap: 12,
    itemWidth: 10,
    itemHeight: 10,
    textStyle: { color: chartPalette.axis, fontSize: 12 }
  }
}

function toggleMetric(key: MetricKey) {
  const selected = selectedMetricKeys.value
  if (selected.includes(key)) {
    if (selected.length <= 1) return
    selectedMetricKeys.value = selected.filter((item) => item !== key)
    return
  }
  selectedMetricKeys.value = [...selected, key]
}

function isMetricSelected(key: MetricKey) {
  return selectedMetricKeys.value.includes(key)
}

function isLastSelectedMetric(key: MetricKey) {
  return selectedMetricKeys.value.length === 1 && selectedMetricKeys.value[0] === key
}

function metricsForGroup(group: MetricGroupKey) {
  return metricDefinitions.value.filter((metric) => metric.group === group)
}

function baselineSeriesName(metric: MetricDefinition) {
  return `${metric.label} ${t('series.baseline')}`
}

function defaultMetricsForMode(mode: ChartMode): MetricKey[] {
  return mode === 'projects'
    ? ['costBurn', 'costPer1kTokens', 'failurePressure']
    : ['p90Latency', 'p10Throughput', 'modelFailureRate']
}

function valueOrNull(value?: number) {
  return value === undefined ? null : value
}

function firstFinite(...values: Array<number | undefined>) {
  return values.find((value) => Number.isFinite(value))
}

function finiteNumber(value?: number) {
  return Number.isFinite(value) ? value : undefined
}

function safeRate(numerator?: number, denominator?: number) {
  if (!Number.isFinite(denominator) || !denominator) return undefined
  return (Number.isFinite(numerator) ? numerator || 0 : 0) / denominator
}

function hasMetricSetSamples(metric?: ModelSignalMetricSet) {
  return Boolean(metric && (metric.sessionCount > 0 || metric.modelCalls > 0))
}

function projectInfo(row: { projectPath?: string }) {
  return projectDisplay(row.projectPath)
}

function formatMetricValue(metric: MetricDefinition, value?: number) {
  if (value === undefined) return t('tooltip.unavailable')
  if (metric.kind === 'cost') return formatCost(value)
  if (metric.kind === 'latency') return `${formatRate(value, 0)} ms/1k`
  if (metric.kind === 'throughput') return `${formatRate(value, 1)} tok/s`
  if (metric.kind === 'percent') return formatPercent(value)
  if (metric.kind === 'pressure') return `${formatRate(value, 2)}/session`
  if (metric.kind === 'ratio') return `${formatRate(value, 2)}x`
  return formatRate(value, 2)
}

function axisName(kind: MetricKind) {
  return t(`axis.${kind}`)
}

function axisLabelForKind(kind: MetricKind, value: number) {
  if (kind === 'cost') return compactCost(value)
  if (kind === 'percent') return formatPercent(value)
  if (kind === 'ratio') return `${formatRate(value, 1)}x`
  return compactNumber(value)
}

function compactCost(value: number) {
  if (Math.abs(value) >= 1) return formatCost(value)
  if (Math.abs(value) >= 0.01) return `$${formatRate(value, 2)}`
  return `$${formatRate(value, 4)}`
}

function compactNumber(value: number) {
  const normalized = Number(value || 0)
  if (Math.abs(normalized) >= 1_000_000) return `${formatRate(normalized / 1_000_000, 1)}M`
  if (Math.abs(normalized) >= 1_000) return `${formatRate(normalized / 1_000, 1)}K`
  return formatRate(normalized, Math.abs(normalized) >= 10 ? 0 : 1)
}

function formatPercent(value: number) {
  const percent = Math.max(0, value || 0) * 100
  if (percent > 0 && percent < 1) return '<1%'
  return `${percent.toLocaleString(undefined, { maximumFractionDigits: percent >= 10 ? 0 : 1 })}%`
}

function formatRate(value: number, digits = 0) {
  if (!Number.isFinite(value)) return '0'
  return (value || 0).toLocaleString(undefined, { maximumFractionDigits: digits })
}

function escapeHtml(value: string | number | undefined) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
</script>

<template>
  <section class="panel model-signals-chart-panel">
    <div class="panel-header model-signals-chart-header">
      <div>
        <h2 class="panel-title">{{ chartTitle }}</h2>
        <div class="panel-kicker">{{ chartKicker }}</div>
      </div>
      <div class="model-signals-chart-actions">
        <a-tag class="status-tag model-signals-chart-count" color="processing">
          {{ selectionCountLabel }}
        </a-tag>
        <a-tooltip :title="selectedMetrics.map((metric) => metric.description).join('\n')">
          <a-tag class="status-tag model-signals-chart-direction" :color="directionLabel === t('direction.higher') ? 'success' : 'warning'">
            {{ directionLabel }}
          </a-tag>
        </a-tooltip>
        <LineChartOutlined v-if="selectedMode === 'daily'" class="panel-header-icon" />
        <BarChartOutlined v-else class="panel-header-icon" />
      </div>
    </div>

    <div class="model-signals-chart-toolbar" :aria-label="t('control.metrics')">
      <a-segmented
        v-if="allowModeSwitch"
        v-model:value="selectedMode"
        class="model-signals-chart-segmented"
        :options="modeOptions"
        :aria-label="t('control.mode')"
      />
      <a-tooltip :title="canCompareBaseline ? '' : t('control.baselineUnavailable')">
        <label class="model-signals-baseline-toggle" :class="{ 'is-disabled': !canCompareBaseline }">
          <input v-model="showBaselineComparison" type="checkbox" :disabled="!canCompareBaseline">
          <span>{{ t('control.baseline') }}</span>
        </label>
      </a-tooltip>
    </div>

    <div class="model-signals-metric-picker" role="group" :aria-label="t('control.metrics')">
      <div v-for="group in metricGroups" :key="group.key" class="model-signals-metric-group">
        <div class="model-signals-metric-group-label">{{ group.label }}</div>
        <div class="model-signals-metric-choices">
          <a-tooltip v-for="item in metricsForGroup(group.key)" :key="item.key" :title="item.description">
            <label
              class="model-signals-metric-choice"
              :class="{ 'is-active': isMetricSelected(item.key), 'is-disabled': isLastSelectedMetric(item.key) }"
              :style="{ '--signal-color': item.color }"
            >
              <input
                type="checkbox"
                :checked="isMetricSelected(item.key)"
                :disabled="isLastSelectedMetric(item.key)"
                @change="toggleMetric(item.key)"
              >
              <span class="model-signals-metric-swatch"></span>
              <span class="model-signals-metric-label">{{ item.label }}</span>
            </label>
          </a-tooltip>
        </div>
      </div>
    </div>

    <a-spin :spinning="loading">
      <div class="panel-body">
        <div v-if="hasChart" ref="chartEl" class="chart model-signals-metric-chart"></div>
        <div v-else class="empty-state model-signals-metric-empty">
          <LineChartOutlined class="empty-state-icon" />
          <div class="empty-state-title">{{ t('empty.title') }}</div>
          <div class="empty-state-text">{{ t('empty.text') }}</div>
        </div>
      </div>
    </a-spin>
  </section>
</template>

<style scoped>
.model-signals-chart-header {
  align-items: flex-start;
  gap: 16px;
}

.model-signals-chart-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.model-signals-chart-count,
.model-signals-chart-direction {
  margin-right: 0;
  white-space: nowrap;
}

.model-signals-chart-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  margin: 2px 0 12px;
}

.model-signals-chart-segmented {
  flex-shrink: 0;
}

.model-signals-baseline-toggle {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 32px;
  padding: 0 10px;
  color: var(--am-text);
  font-size: 12px;
  font-weight: 650;
  background: var(--am-surface);
  border: 1px solid var(--am-border);
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
}

.model-signals-baseline-toggle input {
  width: 14px;
  height: 14px;
  margin: 0;
  accent-color: var(--am-primary);
}

.model-signals-baseline-toggle.is-disabled {
  color: var(--am-muted);
  background: var(--am-surface-subtle);
  cursor: not-allowed;
}

.model-signals-metric-picker {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 14px;
}

.model-signals-metric-group {
  min-width: 0;
  padding: 10px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.model-signals-metric-group-label {
  margin-bottom: 8px;
  color: var(--am-muted);
  font-size: 11px;
  font-weight: 750;
  text-transform: uppercase;
}

.model-signals-metric-choices {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.model-signals-metric-choice {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-height: 28px;
  padding: 0 8px;
  color: var(--am-text-soft);
  font-size: 12px;
  font-weight: 600;
  background: var(--am-surface);
  border: 1px solid var(--am-border);
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
}

.model-signals-metric-choice input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.model-signals-metric-choice.is-active {
  color: var(--am-text);
  border-color: color-mix(in srgb, var(--signal-color) 46%, var(--am-border));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--signal-color) 22%, transparent);
}

.model-signals-metric-choice.is-disabled {
  cursor: default;
}

.model-signals-metric-swatch {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  background: var(--signal-color);
  border-radius: 50%;
}

.model-signals-metric-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-metric-chart {
  height: 380px;
}

.model-signals-metric-empty {
  min-height: 260px;
}

@media (max-width: 1180px) {
  .model-signals-metric-picker {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .model-signals-chart-header,
  .model-signals-chart-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .model-signals-chart-actions {
    justify-content: space-between;
  }

  .model-signals-metric-picker {
    grid-template-columns: 1fr;
  }

  .model-signals-metric-chart {
    height: 320px;
  }
}
</style>
