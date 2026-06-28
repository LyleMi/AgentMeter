<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import ASelect from 'ant-design-vue/es/select'
import ASegmented from 'ant-design-vue/es/segmented'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import { BarChartOutlined, ControlOutlined, LineChartOutlined } from '@ant-design/icons-vue'
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
type MetricKey =
  | 'p90Latency'
  | 'p10Throughput'
  | 'costBurn'
  | 'costPerActiveHour'
  | 'costPer1kTokens'
  | 'cacheSavings'
  | 'failurePressure'
  | 'retryPressure'
  | 'cacheMiss'
  | 'reasoningShare'
  | 'outputExpansion'
type ProjectChartRow = ModelSignalsProjectMetric | ModelSignalProjectHotspot
type ChartRow = ModelSignalsDailyMetric | ProjectChartRow

interface MetricDefinition {
  key: MetricKey
  label: string
  description: string
  kind: MetricKind
  color: string
  chart: ChartKind
  lowerIsBetter: boolean
  value: (metric?: ModelSignalMetricSet) => number | undefined
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
const selectedMetric = ref<MetricKey>(props.initialMode === 'projects' ? 'costBurn' : 'p90Latency')
const { chartEl, getChart, disposeChart } = useEChart()
const { t, locale } = useMessages({
  en: {
    'title.daily': 'Daily Signal Lens',
    'title.projects': 'Project Signal Lens',
    'kicker.daily': 'Switch the metric to see the shape of model service behavior before inspecting rows',
    'kicker.projects': 'Rank projects by the signal that matters for the current investigation',
    'mode.daily': 'Time',
    'mode.projects': 'Projects',
    'control.mode': 'Chart view',
    'control.metric': 'Metric',
    'direction.lower': 'lower is better',
    'direction.higher': 'higher is better',
    'series.current': 'Current',
    'series.baseline': 'Baseline',
    'series.lowSample': 'Low sample',
    'metric.p90Latency': 'P90 latency',
    'metric.p90LatencyDesc': 'Tail model latency per 1k output tokens',
    'metric.p10Throughput': 'P10 throughput',
    'metric.p10ThroughputDesc': 'Slow-floor observed token throughput',
    'metric.costBurn': 'Cost burn',
    'metric.costBurnDesc': 'Observed estimated cost in the row',
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
    'metric.cacheMiss': 'Cache miss rate',
    'metric.cacheMissDesc': 'Uncached input share',
    'metric.reasoningShare': 'Reasoning share',
    'metric.reasoningShareDesc': 'Reasoning tokens divided by output tokens',
    'metric.outputExpansion': 'Output expansion',
    'metric.outputExpansionDesc': 'Output tokens divided by input tokens',
    'tooltip.sessions': 'Sessions',
    'tooltip.modelCalls': 'Model calls',
    'tooltip.tokens': 'Tokens',
    'tooltip.confidence': 'Confidence',
    'tooltip.reason': 'Reason',
    'tooltip.unavailable': 'Unavailable',
    'empty.title': 'No chartable signal values',
    'empty.text': 'Try another metric or broaden the source, model, project, or date scope.',
    'fallback.unknown': 'unknown',
    'fallback.noReason': 'No drift reason'
  },
  'zh-CN': {
    'title.daily': '每日信号镜头',
    'title.projects': '项目信号镜头',
    'kicker.daily': '先切换指标查看模型服务行为的形状，再进入明细行',
    'kicker.projects': '按当前排查最关心的信号给项目排序',
    'mode.daily': '时间',
    'mode.projects': '项目',
    'control.mode': '图表视图',
    'control.metric': '指标',
    'direction.lower': '越低越好',
    'direction.higher': '越高越好',
    'series.current': '当前',
    'series.baseline': '基线',
    'series.lowSample': '低样本',
    'metric.p90Latency': 'P90 延迟',
    'metric.p90LatencyDesc': '按 1k 输出 token 归一化的尾部模型延迟',
    'metric.p10Throughput': 'P10 吞吐',
    'metric.p10ThroughputDesc': '低谷观测 token 吞吐',
    'metric.costBurn': '费用消耗',
    'metric.costBurnDesc': '当前行的观测估算费用',
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
    'metric.cacheMiss': '缓存未命中率',
    'metric.cacheMissDesc': '未缓存输入占比',
    'metric.reasoningShare': '推理占比',
    'metric.reasoningShareDesc': '推理 token 占输出 token 的比例',
    'metric.outputExpansion': '输出扩张',
    'metric.outputExpansionDesc': '输出 token 与输入 token 的比例',
    'tooltip.sessions': '会话',
    'tooltip.modelCalls': '模型调用',
    'tooltip.tokens': 'Token',
    'tooltip.confidence': '置信度',
    'tooltip.reason': '原因',
    'tooltip.unavailable': '不可用',
    'empty.title': '没有可绘制的信号值',
    'empty.text': '可以切换指标，或放宽来源、模型、项目、日期范围。',
    'fallback.unknown': '未知',
    'fallback.noReason': '无漂移原因'
  }
})

const metricDefinitions = computed<MetricDefinition[]>(() => [
  {
    key: 'p90Latency',
    label: t('metric.p90Latency'),
    description: t('metric.p90LatencyDesc'),
    kind: 'latency',
    color: chartPalette.danger,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => firstFinite(metric?.p90ModelLatencyMsPer1kOutputTokens, metric?.modelLatencyMsPer1kOutputTokens)
  },
  {
    key: 'p10Throughput',
    label: t('metric.p10Throughput'),
    description: t('metric.p10ThroughputDesc'),
    kind: 'throughput',
    color: chartPalette.success,
    chart: 'line',
    lowerIsBetter: false,
    value: (metric) => firstFinite(metric?.p10ModelThroughputTokensPerSecond, metric?.modelThroughputTokensPerSecond)
  },
  {
    key: 'costBurn',
    label: t('metric.costBurn'),
    description: t('metric.costBurnDesc'),
    kind: 'cost',
    color: chartPalette.primary,
    chart: 'bar',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.estimatedCostUsd)
  },
  {
    key: 'costPerActiveHour',
    label: t('metric.costPerActiveHour'),
    description: t('metric.costPerActiveHourDesc'),
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
    kind: 'cost',
    color: chartPalette.success,
    chart: 'bar',
    lowerIsBetter: false,
    value: (metric) => finiteNumber(metric?.cacheSavingsUsd)
  },
  {
    key: 'failurePressure',
    label: t('metric.failurePressure'),
    description: t('metric.failurePressureDesc'),
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
    kind: 'pressure',
    color: chartPalette.info,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.avgModelCallsPerSession)
  },
  {
    key: 'cacheMiss',
    label: t('metric.cacheMiss'),
    description: t('metric.cacheMissDesc'),
    kind: 'percent',
    color: chartPalette.warning,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.cacheMissRate)
  },
  {
    key: 'reasoningShare',
    label: t('metric.reasoningShare'),
    description: t('metric.reasoningShareDesc'),
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
    kind: 'ratio',
    color: chartPalette.primary,
    chart: 'line',
    lowerIsBetter: true,
    value: (metric) => finiteNumber(metric?.outputExpansionRate)
  }
])

const activeMetric = computed(() => metricDefinitions.value.find((item) => item.key === selectedMetric.value) || metricDefinitions.value[0])
const modeOptions = computed(() => [
  { label: t('mode.daily'), value: 'daily' },
  { label: t('mode.projects'), value: 'projects' }
])
const metricOptions = computed(() => metricDefinitions.value.map((item) => ({
  label: item.label,
  value: item.key,
  title: item.description
})))
const chartTitle = computed(() => selectedMode.value === 'projects' ? t('title.projects') : t('title.daily'))
const chartKicker = computed(() => selectedMode.value === 'projects' ? t('kicker.projects') : t('kicker.daily'))
const directionLabel = computed(() => activeMetric.value.lowerIsBetter ? t('direction.lower') : t('direction.higher'))
const plottedRows = computed<ChartRow[]>(() => {
  if (selectedMode.value === 'daily') {
    return [...props.dailyRows].sort((left, right) => left.date.localeCompare(right.date))
  }
  return [...props.projectRows]
    .sort((left, right) => metricSortValue(right) - metricSortValue(left))
    .slice(0, 10)
})
const hasChart = computed(() => plottedRows.value.some((row) => metricValueForRow(row) !== undefined))

watch(() => props.initialMode, (mode) => {
  selectedMode.value = mode
})

watch(() => [selectedMode.value, selectedMetric.value, props.dailyRows, props.projectRows, locale.value], renderAfterUpdate, { deep: true })

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
  const metric = activeMetric.value
  const chart = getChart()
  if (!chart) return

  chart.setOption({
    color: [metric.color, chartPalette.warning],
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: metric.chart === 'bar' ? 'shadow' : 'cross', lineStyle: { color: chartPalette.pointer }, shadowStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => dailyTooltipMarkup(params)
    },
    grid: { left: 64, right: 28, top: 34, bottom: 42 },
    legend: {
      show: true,
      top: 0,
      right: 8,
      itemGap: 14,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: chartPalette.axis, fontSize: 12 }
    },
    xAxis: {
      type: 'category',
      data: rows.map((row) => row.date.slice(5)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.axis, fontSize: 11 }
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: (value: number) => axisLabel(value) },
      splitLine: { lineStyle: { color: chartPalette.grid } }
    },
    series: [
      {
        name: metric.label,
        type: metric.chart,
        smooth: metric.chart === 'line',
        barWidth: 16,
        symbolSize: 7,
        lineStyle: { width: 2, color: metric.color },
        itemStyle: { color: metric.color },
        data: rows.map((row) => valueOrNull(metricValueForRow(row)))
      },
      {
        name: t('series.lowSample'),
        type: 'scatter',
        symbol: 'diamond',
        symbolSize: 11,
        itemStyle: { color: chartPalette.warning },
        data: rows.map((row) => row.lowSample ? valueOrNull(metricValueForRow(row)) : null)
      }
    ]
  }, true)
}

function renderProjectChart() {
  const rows = plottedRows.value as ProjectChartRow[]
  const metric = activeMetric.value
  const chart = getChart()
  if (!chart) return

  const currentValues = rows.map((row) => valueOrNull(metricValueForProject(row, 'current')))
  const baselineValues = rows.map((row) => valueOrNull(metricValueForProject(row, 'baseline')))
  const showBaseline = baselineValues.some((value) => value !== null)

  chart.setOption({
    color: [metric.color, chartPalette.axis],
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'shadow', shadowStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => projectTooltipMarkup(params)
    },
    grid: { left: 128, right: 38, top: 34, bottom: 30 },
    legend: {
      show: showBaseline,
      top: 0,
      right: 8,
      itemGap: 14,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: chartPalette.axis, fontSize: 12 }
    },
    xAxis: {
      type: 'value',
      axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: (value: number) => axisLabel(value) },
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
    series: [
      {
        name: t('series.current'),
        type: 'bar',
        barWidth: showBaseline ? 10 : 14,
        itemStyle: { color: metric.color, borderRadius: [0, 3, 3, 0] },
        data: currentValues
      },
      ...(showBaseline
        ? [{
            name: t('series.baseline'),
            type: 'bar',
            barWidth: 10,
            itemStyle: { color: chartPalette.axis, borderRadius: [0, 3, 3, 0] },
            data: baselineValues
          }]
        : [])
    ]
  }, true)
}

function dailyTooltipMarkup(params: unknown) {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as { dataIndex?: number; axisValue?: string } | undefined
  const row = plottedRows.value[first?.dataIndex ?? 0] as ModelSignalsDailyMetric | undefined
  if (!row) return ''
  return [
    `<strong>${escapeHtml(row.date || first?.axisValue || '')}</strong>`,
    `<div>${escapeHtml(activeMetric.value.label)}: ${escapeHtml(formatMetricValue(metricValueForRow(row)))}</div>`,
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
  const currentValue = metricValueForProject(row, 'current')
  const baselineValue = metricValueForProject(row, 'baseline')
  const drift = row.drift
  return [
    `<strong>${escapeHtml(info.full || first?.axisValue || '')}</strong>`,
    `<div>${t('series.current')}: ${escapeHtml(formatMetricValue(currentValue))}</div>`,
    `<div>${t('series.baseline')}: ${escapeHtml(formatMetricValue(baselineValue))}</div>`,
    `<div>${t('tooltip.sessions')}: ${formatNumber(projectMetricSet(row, 'current')?.sessionCount || row.sessionCount)}</div>`,
    `<div>${t('tooltip.tokens')}: ${formatNumber(projectMetricSet(row, 'current')?.totalTokens || row.totalTokens)}</div>`,
    `<div>${t('tooltip.confidence')}: ${escapeHtml(drift?.confidence || t('fallback.unknown'))}</div>`,
    `<div>${t('tooltip.reason')}: ${escapeHtml(drift?.reasons?.[0] || drift?.sampleNote || t('fallback.noReason'))}</div>`
  ].join('')
}

function metricSortValue(row: ProjectChartRow) {
  return metricValueForProject(row, 'current') ?? metricValueForProject(row, 'total') ?? -1
}

function metricValueForRow(row: ChartRow) {
  if (selectedMode.value === 'projects') return metricValueForProject(row as ProjectChartRow, 'current')
  return activeMetric.value.value(row as ModelSignalsDailyMetric)
}

function metricValueForProject(row: ProjectChartRow, window: 'current' | 'baseline' | 'total') {
  return activeMetric.value.value(projectMetricSet(row, window))
}

function projectMetricSet(row: ProjectChartRow, window: 'current' | 'baseline' | 'total'): ModelSignalMetricSet | undefined {
  if (window === 'current') return row.current || row
  if (window === 'baseline') return row.baseline
  return row
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

function projectInfo(row: { projectPath?: string }) {
  return projectDisplay(row.projectPath)
}

function formatMetricValue(value?: number) {
  if (value === undefined) return t('tooltip.unavailable')
  const metric = activeMetric.value
  if (metric.kind === 'cost') return formatCost(value)
  if (metric.kind === 'latency') return `${formatRate(value, 0)} ms/1k`
  if (metric.kind === 'throughput') return `${formatRate(value, 1)} tok/s`
  if (metric.kind === 'percent') return formatPercent(value)
  if (metric.kind === 'pressure') return `${formatRate(value, 2)}/session`
  if (metric.kind === 'ratio') return `${formatRate(value, 2)}x`
  return formatRate(value, 2)
}

function axisLabel(value: number) {
  const metric = activeMetric.value
  if (metric.kind === 'cost') return compactCost(value)
  if (metric.kind === 'percent') return formatPercent(value)
  if (metric.kind === 'ratio') return `${formatRate(value, 1)}x`
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
        <a-tooltip :title="activeMetric.description">
          <a-tag class="status-tag model-signals-chart-direction" :color="activeMetric.lowerIsBetter ? 'warning' : 'success'">
            {{ directionLabel }}
          </a-tag>
        </a-tooltip>
        <LineChartOutlined v-if="selectedMode === 'daily'" class="panel-header-icon" />
        <BarChartOutlined v-else class="panel-header-icon" />
      </div>
    </div>

    <div class="model-signals-chart-toolbar" :aria-label="t('control.metric')">
      <a-segmented
        v-if="allowModeSwitch"
        v-model:value="selectedMode"
        class="model-signals-chart-segmented"
        :options="modeOptions"
        :aria-label="t('control.mode')"
      />
      <a-select
        v-model:value="selectedMetric"
        class="model-signals-chart-select"
        :options="metricOptions"
        :popup-match-select-width="false"
        :aria-label="t('control.metric')"
      />
      <ControlOutlined class="model-signals-chart-control-icon" />
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

.model-signals-chart-direction {
  margin-right: 0;
  white-space: nowrap;
}

.model-signals-chart-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  margin: 2px 0 14px;
}

.model-signals-chart-segmented {
  flex-shrink: 0;
}

.model-signals-chart-select {
  width: min(320px, 100%);
}

.model-signals-chart-control-icon {
  color: var(--am-muted);
  font-size: 16px;
}

.model-signals-metric-chart {
  height: 340px;
}

.model-signals-metric-empty {
  min-height: 260px;
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

  .model-signals-chart-select {
    width: 100%;
  }

  .model-signals-chart-control-icon {
    display: none;
  }

  .model-signals-metric-chart {
    height: 300px;
  }
}
</style>
