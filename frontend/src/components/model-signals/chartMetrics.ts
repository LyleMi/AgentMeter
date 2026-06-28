import {
  formatCost,
  projectDisplay,
  type ModelSignalMetricSet,
  type ModelSignalProjectHotspot,
  type ModelSignalsDailyMetric,
  type ModelSignalsProjectMetric
} from '../../api'
import { chartPalette } from '../../chartPalette'
import {
  formatModelSignalPercent as formatPercent,
  formatModelSignalRate as formatRate
} from '../../presentation/modelSignals'
import type {
  ModelSignalsMetricChartMessageKey,
  ModelSignalsMetricChartTranslate
} from './chartMessages'

export type ChartMode = 'daily' | 'projects'
export type ChartKind = 'bar' | 'line'
export type MetricKind = 'cost' | 'latency' | 'throughput' | 'percent' | 'pressure' | 'ratio'
export type MetricGroupKey = 'performance' | 'cost' | 'pressure' | 'shape'
export type MetricWindow = 'current' | 'baseline' | 'total'
export type MetricDirection = 'lower' | 'higher' | 'context'
export type MetricKey =
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
  | 'degradationRisk'
  | 'retryPressure'
  | 'modelFailureRate'
  | 'toolFailureRate'
  | 'cacheMiss'
  | 'reasoningShare'
  | 'outputExpansion'
  | 'toolDependency'

export type ProjectChartRow = ModelSignalsProjectMetric | ModelSignalProjectHotspot
export type ChartRow = ModelSignalsDailyMetric | ProjectChartRow

export interface MetricDefinition {
  key: MetricKey
  label: string
  description: string
  group: MetricGroupKey
  kind: MetricKind
  color: string
  chart: ChartKind
  direction: MetricDirection
  value: (metric?: ModelSignalMetricSet) => number | undefined
}

export interface MetricGroup {
  key: MetricGroupKey
  label: string
}

export function buildMetricGroups(t: ModelSignalsMetricChartTranslate): MetricGroup[] {
  return [
    { key: 'performance', label: t('group.performance') },
    { key: 'cost', label: t('group.cost') },
    { key: 'pressure', label: t('group.pressure') },
    { key: 'shape', label: t('group.shape') }
  ]
}

export function buildMetricDefinitions(t: ModelSignalsMetricChartTranslate): MetricDefinition[] {
  return [
    {
      key: 'p90Latency',
      label: t('metric.p90Latency'),
      description: t('metric.p90LatencyDesc'),
      group: 'performance',
      kind: 'latency',
      color: chartPalette.danger,
      chart: 'line',
      direction: 'lower',
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
      direction: 'lower',
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
      direction: 'higher',
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
      direction: 'higher',
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
      direction: 'lower',
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
      direction: 'lower',
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
      direction: 'lower',
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
      direction: 'lower',
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
      direction: 'higher',
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
      direction: 'lower',
      value: (metric) => finiteNumber(metric?.failurePressure)
    },
    {
      key: 'degradationRisk',
      label: t('metric.degradationRisk'),
      description: t('metric.degradationRiskDesc'),
      group: 'pressure',
      kind: 'percent',
      color: '#be123c',
      chart: 'line',
      direction: 'lower',
      value: (metric) => finiteNumber(metric?.degradationRiskScore)
    },
    {
      key: 'retryPressure',
      label: t('metric.retryPressure'),
      description: t('metric.retryPressureDesc'),
      group: 'pressure',
      kind: 'pressure',
      color: '#9333ea',
      chart: 'line',
      direction: 'lower',
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
      direction: 'lower',
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
      direction: 'lower',
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
      direction: 'lower',
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
      direction: 'context',
      value: (metric) => reasoningOverhead(metric)
    },
    {
      key: 'outputExpansion',
      label: t('metric.outputExpansion'),
      description: t('metric.outputExpansionDesc'),
      group: 'shape',
      kind: 'ratio',
      color: chartPalette.primary,
      chart: 'line',
      direction: 'context',
      value: (metric) => generationOverhead(metric)
    },
    {
      key: 'toolDependency',
      label: t('metric.toolDependency'),
      description: t('metric.toolDependencyDesc'),
      group: 'shape',
      kind: 'percent',
      color: chartPalette.axis,
      chart: 'line',
      direction: 'lower',
      value: (metric) => finiteNumber(metric?.toolDependencyRate)
    }
  ]
}

export function defaultMetricsForMode(mode: ChartMode): MetricKey[] {
  return mode === 'projects'
    ? ['degradationRisk', 'costPer1kTokens', 'failurePressure']
    : ['degradationRisk', 'p90Latency', 'p10Throughput']
}

export function resolveSelectedMetrics(keys: MetricKey[], definitions: MetricDefinition[]) {
  const selected = keys
    .map((key) => definitions.find((item) => item.key === key))
    .filter((item): item is MetricDefinition => Boolean(item))
  return selected.length ? selected : definitions.slice(0, 1)
}

export function plottedRowsForMode(
  mode: ChartMode,
  dailyRows: ModelSignalsDailyMetric[],
  projectRows: ProjectChartRow[],
  primaryMetric: MetricDefinition
): ChartRow[] {
  if (mode === 'daily') {
    return [...dailyRows].sort((left, right) => left.date.localeCompare(right.date))
  }
  return [...projectRows]
    .sort((left, right) => metricSortValue(right, primaryMetric) - metricSortValue(left, primaryMetric))
    .slice(0, 10)
}

export function hasChartData(rows: ChartRow[], metrics: MetricDefinition[], mode: ChartMode) {
  return metrics.some((metric) =>
    rows.some((row) => metricValueForRow(row, metric, mode, 'current') !== undefined)
  )
}

export function hasBaselineComparison(rows: ChartRow[], metrics: MetricDefinition[], mode: ChartMode) {
  return metrics.some((metric) =>
    rows.some((row) => metricValueForRow(row, metric, mode, 'baseline') !== undefined)
  )
}

export function metricKindsFor(metrics: MetricDefinition[]): MetricKind[] {
  const kinds: MetricKind[] = []
  metrics.forEach((metric) => {
    if (!kinds.includes(metric.kind)) kinds.push(metric.kind)
  })
  return kinds
}

export function shouldNormalizeProjectScale(mode: ChartMode, kinds: MetricKind[]) {
  return mode === 'projects' && kinds.length > 1
}

export function metricSortValue(row: ProjectChartRow, primaryMetric: MetricDefinition) {
  return metricValueForProject(row, primaryMetric, 'current') ?? metricValueForProject(row, primaryMetric, 'total') ?? -1
}

export function metricValueForRow(
  row: ChartRow,
  metric: MetricDefinition,
  mode: ChartMode,
  window: MetricWindow = 'current'
) {
  if (mode === 'projects') return metricValueForProject(row as ProjectChartRow, metric, window)
  const dailyRow = row as ModelSignalsDailyMetric
  if (window === 'baseline') return hasMetricSetSamples(dailyRow.baseline) ? metric.value(dailyRow.baseline) : undefined
  return metric.value(dailyRow)
}

export function metricValueForProject(row: ProjectChartRow, metric: MetricDefinition, window: MetricWindow) {
  const set = projectMetricSet(row, window)
  if (window === 'baseline' && !hasMetricSetSamples(set)) return undefined
  return metric.value(set)
}

export function projectMetricSet(row: ProjectChartRow, window: MetricWindow): ModelSignalMetricSet | undefined {
  if (window === 'current') return row.current || row
  if (window === 'baseline') return row.baseline
  return row
}

export function projectPlotValue(
  row: ProjectChartRow,
  metric: MetricDefinition,
  window: MetricWindow,
  normalized: boolean,
  max: number
) {
  const value = metricValueForProject(row, metric, window)
  if (value === undefined || !normalized) return value
  if (max <= 0) return 0
  return value / max * 100
}

export function projectMetricMax(rows: ProjectChartRow[], metric: MetricDefinition, showBaselineComparison: boolean) {
  const values: number[] = []
  rows.forEach((row) => {
    const current = metricValueForProject(row, metric, 'current')
    if (current !== undefined) values.push(Math.abs(current))
    if (showBaselineComparison) {
      const baseline = metricValueForProject(row, metric, 'baseline')
      if (baseline !== undefined) values.push(Math.abs(baseline))
    }
  })
  return Math.max(...values, 0)
}

export function valueOrNull(value?: number) {
  return value === undefined ? null : value
}

export function baselineSeriesName(t: ModelSignalsMetricChartTranslate, metric: MetricDefinition) {
  return `${metric.label} ${t('series.baseline')}`
}

export function projectInfo(row: { projectPath?: string }) {
  return projectDisplay(row.projectPath)
}

export function formatMetricValue(
  t: ModelSignalsMetricChartTranslate,
  metric: MetricDefinition,
  value?: number
) {
  if (value === undefined) return t('tooltip.unavailable')
  if (metric.kind === 'cost') return formatCost(value)
  if (metric.kind === 'latency') return `${formatRate(value, 0)} ms/1k`
  if (metric.kind === 'throughput') return `${formatRate(value, 1)} tok/s`
  if (metric.kind === 'percent') return formatPercent(value)
  if (metric.kind === 'pressure') return `${formatRate(value, 2)}/session`
  if (metric.kind === 'ratio') return `${formatRate(value, 2)}x`
  return formatRate(value, 2)
}

export function axisName(t: ModelSignalsMetricChartTranslate, kind: MetricKind) {
  return t(`axis.${kind}` as ModelSignalsMetricChartMessageKey)
}

export function axisLabelForKind(kind: MetricKind, value: number) {
  if (kind === 'cost') return compactCost(value)
  if (kind === 'percent') return formatPercent(value)
  if (kind === 'ratio') return `${formatRate(value, 1)}x`
  return compactNumber(value)
}

function firstFinite(...values: Array<number | undefined>) {
  return values.find((value) => Number.isFinite(value))
}

function finiteNumber(value?: number) {
  return Number.isFinite(value) ? value : undefined
}

function reasoningOverhead(metric?: ModelSignalMetricSet) {
  return firstFinite(metric?.reasoningOverheadRate, metric?.reasoningTokenOverhead, metric?.reasoningOutputShare, metric?.reasoningTokenShare)
}

function generationOverhead(metric?: ModelSignalMetricSet) {
  return firstFinite(metric?.generationTokenOverhead, metric?.outputExpansionRate)
}

function safeRate(numerator?: number, denominator?: number) {
  if (!Number.isFinite(denominator) || !denominator) return undefined
  return (Number.isFinite(numerator) ? numerator || 0 : 0) / denominator
}

function hasMetricSetSamples(metric?: ModelSignalMetricSet) {
  return Boolean(metric && (metric.sessionCount > 0 || metric.modelCalls > 0))
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
