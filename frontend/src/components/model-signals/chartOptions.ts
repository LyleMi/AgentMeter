import {
  formatNumber,
  type ModelSignalsDailyMetric
} from '../../api'
import { chartPalette } from '../../chartPalette'
import type { ModelSignalsMetricChartTranslate } from './chartMessages'
import {
  axisLabelForKind,
  axisName,
  baselineSeriesName,
  formatMetricValue,
  metricValueForProject,
  metricValueForRow,
  projectInfo,
  projectMetricMax,
  projectMetricSet,
  projectPlotValue,
  valueOrNull,
  type ChartRow,
  type MetricDefinition,
  type MetricKind,
  type ProjectChartRow
} from './chartMetrics'

interface DailyChartOptionInput {
  rows: ModelSignalsDailyMetric[]
  selectedMetrics: MetricDefinition[]
  primaryMetric: MetricDefinition
  activeMetricKinds: MetricKind[]
  showBaselineComparison: boolean
  t: ModelSignalsMetricChartTranslate
}

interface ProjectChartOptionInput {
  rows: ProjectChartRow[]
  selectedMetrics: MetricDefinition[]
  primaryMetric: MetricDefinition
  activeMetricKinds: MetricKind[]
  normalizeProjectScale: boolean
  showBaselineComparison: boolean
  t: ModelSignalsMetricChartTranslate
}

interface TooltipInput {
  rows: ChartRow[]
  selectedMetrics: MetricDefinition[]
  showBaselineComparison: boolean
  t: ModelSignalsMetricChartTranslate
}

export function buildDailyChartOption(input: DailyChartOptionInput) {
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'cross', lineStyle: { color: chartPalette.pointer }, shadowStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => dailyTooltipMarkup(params, input)
    },
    grid: dailyGrid(input.activeMetricKinds),
    legend: legendOptions(),
    xAxis: {
      type: 'category',
      data: input.rows.map((row) => row.date.slice(5)),
      boundaryGap: true,
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.axis, fontSize: 11, hideOverlap: true }
    },
    yAxis: input.activeMetricKinds.map((kind, index) => valueAxisOptions(input.t, kind, index)),
    series: [
      ...input.selectedMetrics.flatMap((metric) => dailyMetricSeries(metric, input)),
      lowSampleSeries(input)
    ].filter(Boolean)
  }
}

export function buildProjectChartOption(input: ProjectChartOptionInput) {
  const axisKind = input.activeMetricKinds[0] || input.primaryMetric.kind

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'shadow', shadowStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => projectTooltipMarkup(params, input)
    },
    grid: { left: 136, right: 56, top: 68, bottom: 38 },
    legend: legendOptions(),
    xAxis: {
      type: 'value',
      name: input.normalizeProjectScale ? input.t('axis.relative') : axisName(input.t, axisKind),
      nameTextStyle: { color: chartPalette.axis, fontSize: 11, padding: [0, 0, 0, 4] },
      axisLabel: {
        color: chartPalette.axis,
        fontSize: 11,
        formatter: (value: number) => input.normalizeProjectScale ? `${Math.round(value)}%` : axisLabelForKind(axisKind, value)
      },
      splitLine: { lineStyle: { color: chartPalette.grid } }
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: input.rows.map((row) => projectInfo(row).main),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.text, fontSize: 11, overflow: 'truncate', width: 110 }
    },
    series: input.selectedMetrics.flatMap((metric) => projectMetricSeries(metric, input))
  }
}

function dailyMetricSeries(metric: MetricDefinition, input: DailyChartOptionInput) {
  const series: Array<Record<string, unknown>> = [{
    name: metric.label,
    type: metric.chart,
    yAxisIndex: metricAxisIndex(metric, input.activeMetricKinds),
    smooth: metric.chart === 'line',
    barMaxWidth: 16,
    symbolSize: 7,
    lineStyle: { width: 2, color: metric.color },
    itemStyle: { color: metric.color, borderRadius: metric.chart === 'bar' ? [3, 3, 0, 0] : 0 },
    emphasis: { focus: 'series' },
    data: input.rows.map((row) => valueOrNull(metricValueForRow(row, metric, 'daily', 'current')))
  }]
  if (input.showBaselineComparison) {
    series.push({
      name: baselineSeriesName(input.t, metric),
      type: metric.chart,
      yAxisIndex: metricAxisIndex(metric, input.activeMetricKinds),
      smooth: metric.chart === 'line',
      barMaxWidth: 12,
      symbolSize: 5,
      lineStyle: { width: 2, type: 'dashed', color: metric.color, opacity: 0.5 },
      itemStyle: { color: metric.color, opacity: 0.28, borderRadius: metric.chart === 'bar' ? [3, 3, 0, 0] : 0 },
      emphasis: { focus: 'series' },
      data: input.rows.map((row) => valueOrNull(metricValueForRow(row, metric, 'daily', 'baseline')))
    })
  }
  return series
}

function lowSampleSeries(input: DailyChartOptionInput) {
  const metric = input.primaryMetric
  return {
    name: input.t('series.lowSample'),
    type: 'scatter',
    yAxisIndex: metricAxisIndex(metric, input.activeMetricKinds),
    symbol: 'diamond',
    symbolSize: 11,
    itemStyle: { color: chartPalette.warning },
    data: input.rows.map((row) => row.lowSample ? valueOrNull(metricValueForRow(row, metric, 'daily', 'current')) : null)
  }
}

function projectMetricSeries(metric: MetricDefinition, input: ProjectChartOptionInput) {
  const max = projectMetricMax(input.rows, metric, input.showBaselineComparison)
  const currentSeries: Record<string, unknown> = {
    name: metric.label,
    type: 'bar',
    barMaxWidth: 12,
    itemStyle: { color: metric.color, borderRadius: [0, 3, 3, 0] },
    emphasis: { focus: 'series' },
    data: input.rows.map((row) => valueOrNull(projectPlotValue(row, metric, 'current', input.normalizeProjectScale, max)))
  }
  if (!input.showBaselineComparison) return [currentSeries]

  return [
    currentSeries,
    {
      name: baselineSeriesName(input.t, metric),
      type: 'bar',
      barMaxWidth: 10,
      itemStyle: { color: metric.color, opacity: 0.28, borderRadius: [0, 3, 3, 0] },
      emphasis: { focus: 'series' },
      data: input.rows.map((row) => valueOrNull(projectPlotValue(row, metric, 'baseline', input.normalizeProjectScale, max)))
    }
  ]
}

function dailyTooltipMarkup(params: unknown, input: TooltipInput) {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as { dataIndex?: number; axisValue?: string } | undefined
  const row = input.rows[first?.dataIndex ?? 0] as ModelSignalsDailyMetric | undefined
  if (!row) return ''
  return [
    `<strong>${escapeHtml(row.date || first?.axisValue || '')}</strong>`,
    ...input.selectedMetrics.map((metric) => metricTooltipLine(row, metric, 'daily', input)),
    `<div>${input.t('tooltip.sessions')}: ${formatNumber(row.sessionCount)}</div>`,
    `<div>${input.t('tooltip.modelCalls')}: ${formatNumber(row.modelCalls)}</div>`,
    `<div>${input.t('tooltip.tokens')}: ${formatNumber(row.totalTokens)}</div>`,
    `<div>${input.t('tooltip.confidence')}: ${escapeHtml(row.drift?.confidence || input.t('fallback.unknown'))}</div>`,
    `<div>${input.t('tooltip.reason')}: ${escapeHtml(row.keyReason || row.drift?.sampleNote || row.drift?.reasons?.[0] || input.t('fallback.noReason'))}</div>`
  ].join('')
}

function projectTooltipMarkup(params: unknown, input: TooltipInput) {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as { dataIndex?: number; axisValue?: string } | undefined
  const row = input.rows[first?.dataIndex ?? 0] as ProjectChartRow | undefined
  if (!row) return ''
  const info = projectInfo(row)
  const current = projectMetricSet(row, 'current')
  const drift = row.drift
  return [
    `<strong>${escapeHtml(info.full || first?.axisValue || '')}</strong>`,
    ...input.selectedMetrics.map((metric) => metricTooltipLine(row, metric, 'projects', input)),
    `<div>${input.t('tooltip.sessions')}: ${formatNumber(current?.sessionCount || row.sessionCount)}</div>`,
    `<div>${input.t('tooltip.tokens')}: ${formatNumber(current?.totalTokens || row.totalTokens)}</div>`,
    `<div>${input.t('tooltip.confidence')}: ${escapeHtml(drift?.confidence || input.t('fallback.unknown'))}</div>`,
    `<div>${input.t('tooltip.reason')}: ${escapeHtml(drift?.reasons?.[0] || drift?.sampleNote || input.t('fallback.noReason'))}</div>`
  ].join('')
}

function metricTooltipLine(
  row: ChartRow,
  metric: MetricDefinition,
  mode: 'daily' | 'projects',
  input: TooltipInput
) {
  const currentValue = mode === 'projects'
    ? metricValueForProject(row as ProjectChartRow, metric, 'current')
    : metricValueForRow(row, metric, 'daily', 'current')
  const baselineValue = mode === 'projects'
    ? metricValueForProject(row as ProjectChartRow, metric, 'baseline')
    : metricValueForRow(row, metric, 'daily', 'baseline')
  const baseline = input.showBaselineComparison
    ? ` / ${input.t('series.baseline')} ${escapeHtml(formatMetricValue(input.t, metric, baselineValue))}`
    : ''
  return `<div><span style="color:${metric.color}">●</span> ${escapeHtml(metric.label)}: ${escapeHtml(formatMetricValue(input.t, metric, currentValue))}${baseline}</div>`
}

function metricAxisIndex(metric: MetricDefinition, activeMetricKinds: MetricKind[]) {
  const index = activeMetricKinds.indexOf(metric.kind)
  return index >= 0 ? index : 0
}

function valueAxisOptions(t: ModelSignalsMetricChartTranslate, kind: MetricKind, index: number) {
  const position = index % 2 === 0 ? 'left' : 'right'
  const sameSideOffset = Math.floor(index / 2) * 54
  return {
    type: 'value',
    name: axisName(t, kind),
    position,
    offset: sameSideOffset,
    nameTextStyle: { color: chartPalette.axis, fontSize: 11, padding: [0, 0, 0, 4] },
    axisLine: { show: index > 0, lineStyle: { color: chartPalette.border } },
    axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: (value: number) => axisLabelForKind(kind, value) },
    splitLine: { show: index === 0, lineStyle: { color: chartPalette.grid } }
  }
}

function dailyGrid(activeMetricKinds: MetricKind[]) {
  const leftCount = activeMetricKinds.filter((_, index) => index % 2 === 0).length
  const rightCount = activeMetricKinds.length - leftCount
  return {
    left: 76 + Math.max(0, leftCount - 1) * 54,
    right: 50 + Math.max(0, rightCount - 1) * 58,
    top: 68,
    bottom: 44
  }
}

function legendOptions() {
  return {
    show: true,
    top: 2,
    left: 8,
    right: 8,
    type: 'scroll',
    orient: 'horizontal',
    itemGap: 10,
    itemWidth: 10,
    itemHeight: 10,
    pageButtonPosition: 'end',
    pageIconSize: 10,
    textStyle: {
      color: chartPalette.axis,
      fontSize: 12,
      width: 96,
      overflow: 'truncate',
      ellipsis: '...'
    }
  }
}

function escapeHtml(value: string | number | undefined) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
