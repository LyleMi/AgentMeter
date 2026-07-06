import {
  formatNumber,
  type ModelSignalDrift,
  type ModelSignalMatrixCell,
  type ModelSignalMatrixRow,
  type ModelSignalMetricSet
} from '../../api'
import {
  formatModelSignalPercent as formatPercent,
  formatModelSignalRate as formatRate
} from '../../presentation/modelSignals'
import {
  clampRiskScore,
  inverseThresholdRiskScore,
  rangeRiskScore,
  thresholdRiskScore
} from '../../presentation/riskScore'
import { sourceDisplay } from '../../presentation/sourceIdentity'

export type QualityRiskLevel = 'low' | 'watch' | 'elevated' | 'high'

export interface QualityRiskDriver {
  key: string
  label: string
  value: number
  formattedValue: string
  contribution: number
  weight: number
  severity: QualityRiskLevel
  explanation: string
}

export interface QualityRiskRow {
  key: string
  sourceKey: string
  sourceLabel: string
  sourceSecondary: string
  model: string
  modelProvider: string
  score: number
  level: QualityRiskLevel
  current: ModelSignalMetricSet
  baseline?: ModelSignalMetricSet
  drift?: ModelSignalDrift
  cell: ModelSignalMatrixCell
  drivers: QualityRiskDriver[]
  primaryReason: string
  sampleNote: string
  sessionCount: number
  modelCalls: number
  totalTokens: number
}

export interface QualityRiskTranslate {
  (key: QualityRiskMessageKey, params?: Record<string, string | number | boolean | null | undefined>): string
}

export const qualityRiskMessages = {
  en: {
    'risk.level.low': 'low',
    'risk.level.watch': 'watch',
    'risk.level.elevated': 'elevated',
    'risk.level.high': 'high',
    'risk.driver.latency': 'Tail latency',
    'risk.driver.latencyExplain': 'Tail responses are slow after normalizing by generated output.',
    'risk.driver.throughput': 'Slow-floor throughput',
    'risk.driver.throughputExplain': 'Observed token throughput is below the expected floor.',
    'risk.driver.failurePressure': 'Failure pressure',
    'risk.driver.failurePressureExplain': 'Model or tool failures are concentrated per session.',
    'risk.driver.toolFailureRate': 'Tool failures',
    'risk.driver.toolFailureRateExplain': 'Tool failures are taking a larger share of tool calls.',
    'risk.driver.cacheMiss': 'Cache misses',
    'risk.driver.cacheMissExplain': 'A high uncached input share can make the same work slower or more expensive.',
    'risk.driver.retryPressure': 'Retry pressure',
    'risk.driver.retryPressureExplain': 'More model calls per session can indicate repair loops or unstable responses.',
    'risk.driver.outputExpansion': 'Output expansion',
    'risk.driver.outputExpansionExplain': 'Generated output is large relative to input, which can change latency and cost.',
    'risk.driver.reasoningOverhead': 'Reasoning overhead',
    'risk.driver.reasoningOverheadExplain': 'Hidden reasoning output is high relative to visible output.',
    'risk.fallbackReason': 'No dominant risk driver',
    'risk.sampleFallback': 'Sample confidence is normal',
    'risk.rowSummary': '{sessions} sessions, {calls} model calls, {tokens} tokens'
  },
  'zh-CN': {
    'risk.level.low': '低',
    'risk.level.watch': '观察',
    'risk.level.elevated': '升高',
    'risk.level.high': '高',
    'risk.driver.latency': '尾部延迟',
    'risk.driver.latencyExplain': '按生成输出归一化后，尾部响应明显变慢。',
    'risk.driver.throughput': '低谷吞吐',
    'risk.driver.throughputExplain': '观测 token 吞吐低于预期下沿。',
    'risk.driver.failurePressure': '失败压力',
    'risk.driver.failurePressureExplain': '模型或工具失败集中出现在每个会话里。',
    'risk.driver.toolFailureRate': '工具失败',
    'risk.driver.toolFailureRateExplain': '工具失败在工具调用中的占比更高。',
    'risk.driver.cacheMiss': '缓存未命中',
    'risk.driver.cacheMissExplain': '未缓存输入占比过高，会让同类工作更慢或更贵。',
    'risk.driver.retryPressure': '重试压力',
    'risk.driver.retryPressureExplain': '每会话模型调用数偏高，可能来自修复循环或响应不稳定。',
    'risk.driver.outputExpansion': '输出膨胀',
    'risk.driver.outputExpansionExplain': '生成输出相对输入过大，会改变延迟和费用。',
    'risk.driver.reasoningOverhead': '推理开销',
    'risk.driver.reasoningOverheadExplain': '隐藏推理输出相对可见输出偏高。',
    'risk.fallbackReason': '没有主导风险因子',
    'risk.sampleFallback': '样本置信度正常',
    'risk.rowSummary': '{sessions} 个会话，{calls} 次模型调用，{tokens} token'
  }
} as const

export type QualityRiskMessageKey = keyof typeof qualityRiskMessages.en

export function buildQualityRiskRows(
  matrixRows: ModelSignalMatrixRow[],
  t: QualityRiskTranslate,
  fallback: string
): QualityRiskRow[] {
  return matrixRows
    .flatMap((row) => buildQualityRiskRowsForSource(row, t, fallback))
    .sort((left, right) => {
    if (left.score !== right.score) return right.score - left.score
    if (left.sessionCount !== right.sessionCount) return right.sessionCount - left.sessionCount
    return left.sourceLabel.localeCompare(right.sourceLabel)
  })
}

function buildQualityRiskRowsForSource(row: ModelSignalMatrixRow, t: QualityRiskTranslate, fallback: string) {
  const source = sourceDisplay(row, fallback)
  return (row.cells || []).map((cell) => buildQualityRiskRow(source, cell, t, fallback))
}

function buildQualityRiskRow(
  source: ReturnType<typeof sourceDisplay>,
  cell: ModelSignalMatrixCell,
  t: QualityRiskTranslate,
  fallback: string
): QualityRiskRow {
  const current = cell.current
  const drivers = sortedQualityRiskDrivers(current, t)
  const score = clampRiskScore(current?.degradationRiskScore || 0)
  const drift = cell.drift
  return {
    key: `${source.key}:${cell.modelProvider}:${cell.model}`,
    sourceKey: source.key,
    sourceLabel: source.label,
    sourceSecondary: source.secondary,
    model: cell.model || fallback,
    modelProvider: cell.modelProvider || fallback,
    score,
    level: qualityRiskLevel(score),
    current,
    baseline: cell.baseline,
    drift,
    cell,
    drivers: drivers.slice(0, 4),
    primaryReason: primaryQualityRiskReason(cell, drivers, t),
    sampleNote: drift?.sampleNote || t('risk.sampleFallback'),
    sessionCount: cell.sessionCount || current?.sessionCount || 0,
    modelCalls: cell.modelCalls || current?.modelCalls || 0,
    totalTokens: cell.totalTokens || current?.totalTokens || 0
  }
}

function sortedQualityRiskDrivers(metric: ModelSignalMetricSet | undefined, t: QualityRiskTranslate) {
  return buildQualityRiskDrivers(metric, t)
    .filter((driver) => driver.contribution > 0)
    .sort((left, right) => right.contribution - left.contribution)
}

function primaryQualityRiskReason(cell: ModelSignalMatrixCell, drivers: QualityRiskDriver[], t: QualityRiskTranslate) {
  return cell.drift?.reasons?.[0] || cell.keyReason || drivers[0]?.label || t('risk.fallbackReason')
}

export function buildQualityRiskDrivers(metric: ModelSignalMetricSet | undefined, t: QualityRiskTranslate): QualityRiskDriver[] {
  if (!metric) return []
  return qualityRiskDriverInputs(metric, t).map(riskDriver)
}

export function qualityRiskLevel(score: number): QualityRiskLevel {
  if (score >= 0.75) return 'high'
  if (score >= 0.45) return 'elevated'
  if (score >= 0.2) return 'watch'
  return 'low'
}

export function formatQualityRiskScore(score?: number) {
  return formatPercent(clampRiskScore(score || 0))
}

export function formatQualityRiskContribution(value: number) {
  return `${formatPercent(value)} of score`
}

export function qualityRiskRowSummary(row: QualityRiskRow, t: QualityRiskTranslate) {
  return t('risk.rowSummary', {
    sessions: formatNumber(row.sessionCount),
    calls: formatNumber(row.modelCalls),
    tokens: formatNumber(row.totalTokens)
  })
}

function riskDriver(input: {
  key: string
  label: string
  value: number
  formattedValue: (value: number) => string
  contribution: number
  weight: number
  explanation: string
}): QualityRiskDriver {
  const contribution = clampRiskScore(input.contribution)
  const normalized = input.weight > 0 ? contribution / input.weight : 0
  return {
    key: input.key,
    label: input.label,
    value: input.value,
    formattedValue: input.formattedValue(input.value),
    contribution,
    weight: input.weight,
    severity: qualityRiskLevel(normalized),
    explanation: input.explanation
  }
}

function qualityRiskDriverInputs(metric: ModelSignalMetricSet, t: QualityRiskTranslate) {
  const latency = firstPositive(metric.p90ModelLatencyMsPer1kOutputTokens, metric.modelLatencyMsPer1kOutputTokens)
  const throughput = firstPositive(metric.p10ModelThroughputTokensPerSecond, metric.modelThroughputOutputTokensPerSecond, metric.modelThroughputTokensPerSecond)
  return [
    {
      key: 'latency',
      label: t('risk.driver.latency'),
      value: latency,
      formattedValue: (value: number) => `${formatRate(value, 0)} ms/1k`,
      contribution: thresholdRiskScore({ value: latency, warning: 8_000, critical: 20_000 }) * 0.24,
      weight: 0.24,
      explanation: t('risk.driver.latencyExplain')
    },
    {
      key: 'throughput',
      label: t('risk.driver.throughput'),
      value: throughput,
      formattedValue: (value: number) => `${formatRate(value, 1)} tok/s`,
      contribution: inverseThresholdRiskScore({ value: throughput, warning: 40, critical: 12 }) * 0.24,
      weight: 0.24,
      explanation: t('risk.driver.throughputExplain')
    },
    metricRiskDriverInput(t, {
      key: 'failurePressure',
      value: metric.failurePressure || 0,
      weight: 0.18,
      formattedValue: (value) => `${formatRate(value, 2)}/session`,
      score: rangeRiskScore({ value: metric.failurePressure || 0, start: 0.05, span: 0.95 })
    }),
    metricRiskDriverInput(t, {
      key: 'toolFailureRate',
      value: metric.toolFailureRate || 0,
      weight: 0.10,
      formattedValue: formatPercent,
      score: rangeRiskScore({ value: metric.toolFailureRate || 0, start: 0.08, span: 0.42 })
    }),
    metricRiskDriverInput(t, {
      key: 'cacheMiss',
      value: metric.cacheMissRate || 0,
      weight: 0.08,
      formattedValue: formatPercent,
      score: rangeRiskScore({ value: metric.cacheMissRate || 0, start: 0.70, span: 0.30 })
    }),
    metricRiskDriverInput(t, {
      key: 'retryPressure',
      value: metric.avgModelCallsPerSession || 0,
      weight: 0.07,
      formattedValue: (value) => `${formatRate(value, 2)}/session`,
      score: rangeRiskScore({ value: metric.avgModelCallsPerSession || 0, start: 1.5, span: 2.5 })
    }),
    metricRiskDriverInput(t, {
      key: 'outputExpansion',
      value: metric.outputExpansionRate || 0,
      weight: 0.05,
      formattedValue: (value) => `${formatRate(value, 2)}x`,
      score: rangeRiskScore({ value: metric.outputExpansionRate || 0, start: 3.0, span: 5.0 })
    }),
    metricRiskDriverInput(t, {
      key: 'reasoningOverhead',
      value: metric.reasoningOverheadRate || 0,
      weight: 0.04,
      formattedValue: (value) => `${formatRate(value, 2)}x`,
      score: rangeRiskScore({ value: metric.reasoningOverheadRate || 0, start: 1.0, span: 4.0 })
    })
  ]
}

type MetricRiskDriverKey = 'failurePressure' | 'toolFailureRate' | 'cacheMiss' | 'retryPressure' | 'outputExpansion' | 'reasoningOverhead'

interface MetricRiskDriverInput {
  key: MetricRiskDriverKey
  value: number
  weight: number
  formattedValue: (value: number) => string
  score: number
}

function metricRiskDriverInput(t: QualityRiskTranslate, input: MetricRiskDriverInput) {
  return {
    key: input.key,
    label: t(`risk.driver.${input.key}`),
    value: input.value,
    formattedValue: input.formattedValue,
    contribution: input.score * input.weight,
    weight: input.weight,
    explanation: t(`risk.driver.${input.key}Explain`)
  }
}

function firstPositive(...values: Array<number | undefined>) {
  return values.find((value) => Number.isFinite(value) && (value || 0) > 0) || 0
}
