import type {
  ModelSignalDrift,
  ModelSignalDriftMetric,
  ModelSignalMetricSet,
  ModelSignalRates
} from '../types'

type DriftSeverity = 'critical' | 'warning'

interface DriftSignal {
  severity: DriftSeverity
  key: string
  label: string
  direction: string
  reason: string
  currentValue: number
  baselineValue: number
}

interface ThresholdSignalInput extends Omit<DriftSignal, 'severity'> {
  change: number
  criticalAt: number
  warningAt: number
  enabled?: boolean
}

interface ConditionalSignalInput extends DriftSignal {
  condition: boolean
}

function relativeIncrease(current: number, baseline: number): number {
  if (baseline <= 0) return current > 0 ? 1 : 0
  return (current - baseline) / baseline
}

function relativeDecrease(current: number, baseline: number): number {
  if (baseline <= 0) return 0
  return (baseline - current) / baseline
}

export function reasoningOverhead(metric: Pick<ModelSignalRates, 'reasoningOverheadRate'>): number {
  return metric.reasoningOverheadRate
}

export function modelSignalDriftFor(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): ModelSignalDrift {
  const signals = driftSignalsFor(current, baseline)
  const severity = signals.reduce((highest, signal) =>
    severityRank(signal.severity) > severityRank(highest) ? signal.severity : highest, 'healthy'
  )

  return {
    severity,
    confidence: current.sessionCount < 2 || current.modelCalls < 3 ? 'low' : 'high',
    sampleNote: current.sessionCount < 2 || current.modelCalls < 3 ? 'Low sample' : undefined,
    reasons: [...new Set(signals.map((signal) => signal.reason))],
    metrics: signals.map(signalMetric)
  }
}

function driftSignalsFor(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): DriftSignal[] {
  const reasoningOverheadDelta = reasoningOverhead(current) - reasoningOverhead(baseline)
  const degradationRiskDelta = current.degradationRiskScore - baseline.degradationRiskScore
  return [
    latencyDriftSignal(current, baseline),
    throughputDriftSignal(current, baseline),
    outputThroughputDriftSignal(current, baseline),
    toolFailureDriftSignal(current, baseline),
    cacheMissDriftSignal(current, baseline),
    reasoningOverheadDriftSignal(current, baseline, reasoningOverheadDelta),
    degradationRiskDriftSignal(current, baseline, degradationRiskDelta)
  ].filter((signal): signal is DriftSignal => signal !== undefined)
}

function latencyDriftSignal(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): DriftSignal | undefined {
  return thresholdSignal({
    change: relativeIncrease(current.modelLatencyMsPer1kOutputTokens, baseline.modelLatencyMsPer1kOutputTokens),
    criticalAt: 0.55,
    warningAt: 0.22,
    key: 'modelLatencyMsPer1kOutputTokens',
    label: 'model latency per 1k output tokens',
    direction: 'higher_worse',
    reason: 'Latency rose vs baseline',
    currentValue: current.modelLatencyMsPer1kOutputTokens,
    baselineValue: baseline.modelLatencyMsPer1kOutputTokens
  })
}

function throughputDriftSignal(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): DriftSignal | undefined {
  return thresholdSignal({
    change: relativeDecrease(current.modelThroughputTokensPerSecond, baseline.modelThroughputTokensPerSecond),
    criticalAt: 0.42,
    warningAt: 0.2,
    key: 'modelThroughputTokensPerSecond',
    label: 'model throughput',
    direction: 'lower_worse',
    reason: 'Throughput fell vs baseline',
    currentValue: current.modelThroughputTokensPerSecond,
    baselineValue: baseline.modelThroughputTokensPerSecond
  })
}

function outputThroughputDriftSignal(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): DriftSignal | undefined {
  return thresholdSignal({
    change: relativeDecrease(current.modelThroughputOutputTokensPerSecond, baseline.modelThroughputOutputTokensPerSecond),
    criticalAt: 0.5,
    warningAt: 0.24,
    key: 'modelThroughputOutputTokensPerSecond',
    label: 'model output throughput',
    direction: 'lower_worse',
    reason: 'Output throughput fell',
    currentValue: current.modelThroughputOutputTokensPerSecond,
    baselineValue: baseline.modelThroughputOutputTokensPerSecond
  })
}

function toolFailureDriftSignal(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): DriftSignal | undefined {
  return conditionalSignal({
    condition: current.failedToolCalls > baseline.failedToolCalls && current.toolFailureRate >= 0.08,
    severity: current.toolFailureRate >= 0.2 ? 'critical' : 'warning',
    key: 'toolFailureRate',
    label: 'tool failure rate',
    direction: 'higher_downstream_symptom',
    reason: 'Tool failures above baseline',
    currentValue: current.toolFailureRate,
    baselineValue: baseline.toolFailureRate
  })
}

function cacheMissDriftSignal(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): DriftSignal | undefined {
  return thresholdSignal({
    change: current.cacheMissRate - baseline.cacheMissRate,
    criticalAt: Number.POSITIVE_INFINITY,
    warningAt: 0.12,
    key: 'cacheMissRate',
    label: 'cache miss rate',
    direction: 'higher_symptom',
    reason: 'Cache misses above baseline',
    currentValue: current.cacheMissRate,
    baselineValue: baseline.cacheMissRate
  })
}

function reasoningOverheadDriftSignal(
  current: ModelSignalMetricSet,
  baseline: ModelSignalMetricSet,
  change: number
): DriftSignal | undefined {
  return thresholdSignal({
    change,
    criticalAt: Number.POSITIVE_INFINITY,
    warningAt: 0.12,
    key: 'reasoningOverheadRate',
    label: 'reasoning overhead',
    direction: 'cost_shape_review',
    reason: 'Reasoning overhead rose',
    currentValue: reasoningOverhead(current),
    baselineValue: reasoningOverhead(baseline)
  })
}

function degradationRiskDriftSignal(
  current: ModelSignalMetricSet,
  baseline: ModelSignalMetricSet,
  change: number
): DriftSignal | undefined {
  return thresholdSignal({
    change,
    criticalAt: 0.3,
    warningAt: 0.15,
    enabled: current.degradationRiskScore >= 0.3,
    key: 'degradationRiskScore',
    label: 'model quality risk score',
    direction: 'higher_worse',
    reason: 'Model quality risk rose',
    currentValue: current.degradationRiskScore,
    baselineValue: baseline.degradationRiskScore
  })
}

function thresholdSignal(input: ThresholdSignalInput): DriftSignal | undefined {
  if (input.enabled === false || input.change < input.warningAt) return undefined
  return {
    severity: input.change >= input.criticalAt ? 'critical' : 'warning',
    key: input.key,
    label: input.label,
    direction: input.direction,
    reason: input.reason,
    currentValue: input.currentValue,
    baselineValue: input.baselineValue
  }
}

function conditionalSignal(input: ConditionalSignalInput): DriftSignal | undefined {
  if (!input.condition) return undefined
  return {
    severity: input.severity,
    key: input.key,
    label: input.label,
    direction: input.direction,
    reason: input.reason,
    currentValue: input.currentValue,
    baselineValue: input.baselineValue
  }
}

function signalMetric(signal: DriftSignal): ModelSignalDriftMetric {
  return {
    key: signal.key,
    label: signal.label,
    direction: signal.direction,
    severity: signal.severity,
    current: signal.currentValue,
    baseline: signal.baselineValue,
    delta: signal.currentValue - signal.baselineValue,
    deltaPct: signal.baselineValue > 0 ? (signal.currentValue - signal.baselineValue) / signal.baselineValue : 0
  }
}

export function severityRank(value?: string): number {
  const normalized = (value || '').toLowerCase()
  if (normalized === 'critical' || normalized === 'high') return 3
  if (normalized === 'warning' || normalized === 'medium') return 2
  if (normalized === 'watch' || normalized === 'low') return 1
  return 0
}
