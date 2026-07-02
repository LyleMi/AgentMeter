import type {
  ModelSignalAnomalySession,
  ModelSignalBreakdown,
  ModelSignalCohort,
  ModelSignalDrift,
  ModelSignalDriftMetric,
  ModelSignalMatrixCell,
  ModelSignalMatrixRow,
  ModelSignalMetricSet,
  ModelSignalsDailyMetric,
  ModelSignalsProjectMetric,
  ModelSignalProjectHotspot,
  ModelSignalRates,
  ModelSignalsWindow,
  ModelSignals,
  ModelSignalsHealthSummary,
  ModelSignalsTrendPoint,
  Session,
  SourceIdentity,
  ToolCall,
  UsageScopeFilters
} from '../types'
import { filteredSessions, filteredToolCalls } from './sessions'
import { cacheSavingsUsdFor, costSum } from './usageMetrics'
import { groupedBy, projectPathKey, sum } from './utils'

function modelCallsForSession(session: Session): number {
  return Math.max(1, Math.ceil(session.eventCount / 55))
}

function safeRate(numerator: number, denominator: number): number {
  return denominator > 0 ? numerator / denominator : 0
}

function clampRate(value: number): number {
  if (!Number.isFinite(value) || value < 0) return 0
  if (value > 1) return 1
  return value
}

function percentile(values: number[], percentileRank: number): number | undefined {
  const sorted = values.filter((value) => Number.isFinite(value)).sort((left, right) => left - right)
  if (!sorted.length) return undefined
  const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil(sorted.length * percentileRank) - 1))
  return sorted[index]
}

function modelSignalDegradationRiskScore(metric: {
  sessionCount: number
  modelCalls: number
  failurePressure?: number
  toolFailureRate?: number
  cacheMissRate?: number
  avgModelCallsPerSession?: number
  outputExpansionRate?: number
  reasoningOverheadRate?: number
  modelLatencyMsPer1kOutputTokens?: number
  p90ModelLatencyMsPer1kOutputTokens?: number
  modelThroughputTokensPerSecond?: number
  modelThroughputOutputTokensPerSecond?: number
  p10ModelThroughputTokensPerSecond?: number
}): number {
  if (!metric.sessionCount || !metric.modelCalls) return 0
  const latency = firstPositiveNumber(metric.p90ModelLatencyMsPer1kOutputTokens, metric.modelLatencyMsPer1kOutputTokens)
  const throughput = firstPositiveNumber(
    metric.p10ModelThroughputTokensPerSecond,
    metric.modelThroughputOutputTokensPerSecond,
    metric.modelThroughputTokensPerSecond
  )
  return clampRate(
    thresholdScore(latency, 8_000, 20_000) * 0.24 +
    inverseThresholdScore(throughput, 40, 12) * 0.24 +
    rangeScore(metric.failurePressure || 0, 0.05, 0.95) * 0.18 +
    rangeScore(metric.toolFailureRate || 0, 0.08, 0.42) * 0.10 +
    rangeScore(metric.cacheMissRate || 0, 0.70, 0.30) * 0.08 +
    rangeScore(metric.avgModelCallsPerSession || 0, 1.5, 2.5) * 0.07 +
    rangeScore(metric.outputExpansionRate || 0, 3.0, 5.0) * 0.05 +
    rangeScore(metric.reasoningOverheadRate || 0, 1.0, 4.0) * 0.04
  )
}

function thresholdScore(value: number, warning: number, critical: number): number {
  if (value <= warning || warning >= critical) return 0
  if (value >= critical) return 1
  return clampRate((value - warning) / (critical - warning))
}

function inverseThresholdScore(value: number, warning: number, critical: number): number {
  if (value <= 0 || warning <= critical) return 0
  if (value >= warning) return 0
  if (value <= critical) return 1
  return clampRate((warning - value) / (warning - critical))
}

function rangeScore(value: number, start: number, span: number): number {
  if (value <= start || span <= 0) return 0
  return clampRate((value - start) / span)
}

function firstPositiveNumber(...values: Array<number | undefined>): number {
  return values.find((value) => Number.isFinite(value) && (value || 0) > 0) || 0
}

function sessionLatencyMsPer1kOutputTokens(session: Session): number {
  return safeRate(session.modelDurationMs, session.tokenUsage.outputTokens / 1000)
}

function sessionThroughputTokensPerSecond(session: Session): number {
  return safeRate(session.tokenUsage.totalTokens, session.modelDurationMs / 1000)
}

function isSuccessfulToolStatus(status: string): boolean {
  return status === 'completed' || status === 'success'
}

function signalRatesFor(group: Session[], groupToolCalls: ToolCall[]): ModelSignalRates {
  const inputTokens = sum(group, (session) => session.tokenUsage.inputTokens)
  const cachedInputTokens = sum(group, (session) => session.tokenUsage.cachedInputTokens)
  const outputTokens = sum(group, (session) => session.tokenUsage.outputTokens)
  const reasoningOutputTokens = sum(group, (session) => session.tokenUsage.reasoningOutputTokens)
  const totalTokens = sum(group, (session) => session.tokenUsage.totalTokens)
  const modelDurationSeconds = sum(group, (session) => session.modelDurationMs) / 1000
  const failedToolCalls = groupToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length

  return {
    outputExpansionRate: safeRate(outputTokens, inputTokens),
    reasoningTokenShare: safeRate(reasoningOutputTokens, outputTokens),
    reasoningOverheadRate: safeRate(reasoningOutputTokens, Math.max(0, outputTokens - reasoningOutputTokens)),
    cacheMissRate: clampRate(safeRate(inputTokens - cachedInputTokens, inputTokens)),
    modelThroughputTokensPerSecond: safeRate(totalTokens, modelDurationSeconds),
    modelThroughputOutputTokensPerSecond: safeRate(outputTokens, modelDurationSeconds),
    toolFailureRate: safeRate(failedToolCalls, groupToolCalls.length),
    toolDependencyRate: safeRate(group.filter((session) => session.toolCallCount > 0).length, group.length)
  }
}

interface MetricTotals {
  sessionCount: number
  modelCalls: number
  failedModelCalls?: number
  toolCalls: number
  failedToolCalls: number
  totalTokens: number
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  modelDurationMs: number
  wallDurationMs?: number
  activeDurationMs?: number
  toolDurationMs?: number
  idleDurationMs?: number
  estimatedCostUsd?: number
  unpricedSessionCount?: number
  cacheSavingsUsd?: number
  latencySamples?: number[]
  throughputSamples?: number[]
  toolDependencyRate?: number
}

function metricSetFromTotals(totals: MetricTotals): ModelSignalMetricSet {
  const wallDurationMs = totals.wallDurationMs ?? totals.modelDurationMs
  const toolDurationMs = totals.toolDurationMs ?? 0
  const activeDurationMs = totals.activeDurationMs ?? totals.modelDurationMs + toolDurationMs
  const idleDurationMs = totals.idleDurationMs ?? Math.max(0, wallDurationMs - activeDurationMs)
  const modelDurationSeconds = totals.modelDurationMs / 1000
  const activeDurationHours = activeDurationMs / 3_600_000
  const modelLatencyMsPer1kOutputTokens = safeRate(totals.modelDurationMs, totals.outputTokens / 1000)
  const modelThroughputTokensPerSecond = safeRate(totals.totalTokens, modelDurationSeconds)
  const unpricedSessionCount = totals.unpricedSessionCount || 0
  const estimatedCostUsd = totals.estimatedCostUsd
  const hasCompletePricing = estimatedCostUsd !== undefined && unpricedSessionCount === 0
  const costPerSession = hasCompletePricing ? safeRate(estimatedCostUsd, totals.sessionCount) : undefined
  const costPerActiveHour = hasCompletePricing ? safeRate(estimatedCostUsd, activeDurationHours) : undefined
  const costPer1kTokens = hasCompletePricing ? safeRate(estimatedCostUsd, totals.totalTokens / 1000) : undefined
  const failurePressure = safeRate((totals.failedModelCalls || 0) + totals.failedToolCalls, totals.sessionCount)
  const avgModelCallsPerSession = safeRate(totals.modelCalls, totals.sessionCount)
  const outputExpansionRate = safeRate(totals.outputTokens, totals.inputTokens)
  const visibleOutputTokens = Math.max(0, totals.outputTokens - totals.reasoningOutputTokens)
  const billableOutputTokens = totals.outputTokens
  const reasoningOverheadRate = safeRate(totals.reasoningOutputTokens, Math.max(0, totals.outputTokens - totals.reasoningOutputTokens))
  const cacheMissRate = clampRate(safeRate(totals.inputTokens - totals.cachedInputTokens, totals.inputTokens))
  const modelThroughputOutputTokensPerSecond = safeRate(totals.outputTokens, modelDurationSeconds)
  const p90ModelLatencyMsPer1kOutputTokens = percentile(totals.latencySamples || [], 0.9) ?? modelLatencyMsPer1kOutputTokens
  const p10ModelThroughputTokensPerSecond = percentile(totals.throughputSamples || [], 0.1) ?? modelThroughputTokensPerSecond
  const toolFailureRate = safeRate(totals.failedToolCalls, totals.toolCalls)
  return {
    sessionCount: totals.sessionCount,
    modelCalls: totals.modelCalls,
    failedModelCalls: totals.failedModelCalls || 0,
    toolCalls: totals.toolCalls,
    failedToolCalls: totals.failedToolCalls,
    totalTokens: totals.totalTokens,
    inputTokens: totals.inputTokens,
    cachedInputTokens: totals.cachedInputTokens,
    outputTokens: totals.outputTokens,
    reasoningOutputTokens: totals.reasoningOutputTokens,
    visibleOutputTokens,
    billableOutputTokens,
    modelDurationMs: totals.modelDurationMs,
    wallDurationMs,
    activeDurationMs,
    toolDurationMs,
    idleDurationMs,
    estimatedCostUsd: totals.estimatedCostUsd,
    unpricedSessionCount,
    cacheSavingsUsd: totals.cacheSavingsUsd,
    costPerSession,
    costPerActiveHour,
    costPer1kTokens,
    failurePressure,
    degradationRiskScore: modelSignalDegradationRiskScore({
      sessionCount: totals.sessionCount,
      modelCalls: totals.modelCalls,
      failurePressure,
      toolFailureRate,
      cacheMissRate,
      avgModelCallsPerSession,
      outputExpansionRate,
      reasoningOverheadRate,
      modelLatencyMsPer1kOutputTokens,
      p90ModelLatencyMsPer1kOutputTokens,
      modelThroughputTokensPerSecond,
      modelThroughputOutputTokensPerSecond,
      p10ModelThroughputTokensPerSecond
    }),
    avgModelCallsPerSession,
    outputExpansionRate,
    reasoningTokenShare: safeRate(totals.reasoningOutputTokens, totals.outputTokens),
    reasoningOverheadRate,
    cacheMissRate,
    modelThroughputTokensPerSecond,
    modelThroughputOutputTokensPerSecond,
    modelLatencyMsPer1kOutputTokens,
    p50ModelLatencyMsPer1kOutputTokens: percentile(totals.latencySamples || [], 0.5) ?? modelLatencyMsPer1kOutputTokens,
    p90ModelLatencyMsPer1kOutputTokens,
    p50ModelThroughputTokensPerSecond: percentile(totals.throughputSamples || [], 0.5) ?? modelThroughputTokensPerSecond,
    p10ModelThroughputTokensPerSecond,
    toolFailureRate,
    toolDependencyRate: totals.toolDependencyRate ?? safeRate(totals.toolCalls, totals.sessionCount)
  }
}

function metricSetFor(group: Session[], groupToolCalls: ToolCall[]): ModelSignalMetricSet {
  const toolSessions = new Set(groupToolCalls.map((call) => call.sessionId)).size
  return metricSetFromTotals({
    sessionCount: group.length,
    modelCalls: sum(group, modelCallsForSession),
    toolCalls: groupToolCalls.length,
    failedToolCalls: groupToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length,
    totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
    inputTokens: sum(group, (session) => session.tokenUsage.inputTokens),
    cachedInputTokens: sum(group, (session) => session.tokenUsage.cachedInputTokens),
    outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
    reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
    modelDurationMs: sum(group, (session) => session.modelDurationMs),
    wallDurationMs: sum(group, (session) => session.wallDurationMs),
    activeDurationMs: sum(group, (session) => session.activeDurationMs),
    toolDurationMs: sum(group, (session) => session.toolDurationMs),
    idleDurationMs: sum(group, (session) => session.idleDurationMs),
    estimatedCostUsd: costSum(group),
    unpricedSessionCount: group.filter((session) => session.unpriced).length,
    cacheSavingsUsd: cacheSavingsUsdFor(group),
    latencySamples: group.map(sessionLatencyMsPer1kOutputTokens),
    throughputSamples: group.map(sessionThroughputTokensPerSecond),
    toolDependencyRate: safeRate(toolSessions, group.length)
  })
}

function stableHash(value: string): number {
  let hash = 0
  for (let index = 0; index < value.length; index += 1) {
    hash = ((hash * 31) + value.charCodeAt(index)) >>> 0
  }
  return hash
}

function syntheticBaselineFor(current: ModelSignalMetricSet, key: string): ModelSignalMetricSet {
  const profile = stableHash(key) % 5
  const durationFactors = [0.64, 0.76, 0.88, 0.96, 1.08]
  const outputFactors = [0.92, 0.96, 1.02, 1.05, 0.98]
  const reasoningFactors = [0.72, 0.82, 0.94, 1.04, 0.9]
  const cacheLift = [0.1, 0.07, 0.04, 0.02, -0.01]

  const inputTokens = Math.max(0, Math.round(current.inputTokens * (profile === 4 ? 1.03 : 0.98)))
  const outputTokens = Math.max(0, Math.round(current.outputTokens * outputFactors[profile]))
  const reasoningOutputTokens = Math.max(0, Math.min(outputTokens, Math.round(current.reasoningOutputTokens * reasoningFactors[profile])))
  const cachedInputTokens = Math.max(
    0,
    Math.min(inputTokens, Math.round((current.cachedInputTokens * 0.95) + (inputTokens * cacheLift[profile])))
  )
  const modelDurationMs = Math.max(1, Math.round(current.modelDurationMs * durationFactors[profile]))
  const wallDurationMs = Math.max(1, Math.round((current.wallDurationMs || current.modelDurationMs) * durationFactors[profile]))
  const toolDurationMs = Math.max(0, Math.round((current.toolDurationMs || 0) * (profile === 0 ? 0.78 : profile === 1 ? 0.86 : 0.96)))
  const activeDurationMs = Math.max(1, modelDurationMs + toolDurationMs)
  const idleDurationMs = Math.max(0, wallDurationMs - activeDurationMs)
  const failedToolCalls = current.failedToolCalls > 0 ? Math.max(0, current.failedToolCalls - 1) : 0
  const estimatedCostUsd = current.estimatedCostUsd === undefined
    ? undefined
    : Number((current.estimatedCostUsd * [0.88, 0.94, 0.99, 1.03, 0.96][profile]).toFixed(4))
  const cacheSavingsUsd = current.cacheSavingsUsd === undefined
    ? undefined
    : Number((current.cacheSavingsUsd * [1.18, 1.12, 1.04, 1, 0.96][profile]).toFixed(4))

  return metricSetFromTotals({
    sessionCount: current.sessionCount + (profile === 3 ? 1 : 0),
    modelCalls: current.modelCalls + (profile === 3 ? 1 : 0),
    failedModelCalls: current.failedModelCalls || 0,
    toolCalls: current.toolCalls,
    failedToolCalls,
    totalTokens: inputTokens + outputTokens,
    inputTokens,
    cachedInputTokens,
    outputTokens,
    reasoningOutputTokens,
    modelDurationMs,
    wallDurationMs,
    activeDurationMs,
    toolDurationMs,
    idleDurationMs,
    estimatedCostUsd,
    unpricedSessionCount: current.unpricedSessionCount,
    cacheSavingsUsd,
    latencySamples: [
      current.p50ModelLatencyMsPer1kOutputTokens || current.modelLatencyMsPer1kOutputTokens,
      current.p90ModelLatencyMsPer1kOutputTokens || current.modelLatencyMsPer1kOutputTokens
    ].map((value) => value * durationFactors[profile]),
    throughputSamples: [
      current.p10ModelThroughputTokensPerSecond || current.modelThroughputTokensPerSecond,
      current.p50ModelThroughputTokensPerSecond || current.modelThroughputTokensPerSecond
    ].map((value) => safeRate(value, durationFactors[profile])),
    toolDependencyRate: current.toolDependencyRate
  })
}

function relativeIncrease(current: number, baseline: number): number {
  if (baseline <= 0) return current > 0 ? 1 : 0
  return (current - baseline) / baseline
}

function relativeDecrease(current: number, baseline: number): number {
  if (baseline <= 0) return 0
  return (baseline - current) / baseline
}

function reasoningOverhead(metric: Pick<ModelSignalRates, 'reasoningOverheadRate'>): number {
  return metric.reasoningOverheadRate
}

function modelSignalDriftFor(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): ModelSignalDrift {
  const reasons: string[] = []
  const metrics: ModelSignalDriftMetric[] = []
  let severity = 'healthy'

  const mark = (nextSeverity: string, key: string, label: string, direction: string, reason: string, currentValue: number, baselineValue: number) => {
    if (severityRank(nextSeverity) > severityRank(severity)) severity = nextSeverity
    metrics.push({
      key,
      label,
      direction,
      severity: nextSeverity,
      current: currentValue,
      baseline: baselineValue,
      delta: currentValue - baselineValue,
      deltaPct: baselineValue > 0 ? (currentValue - baselineValue) / baselineValue : 0
    })
    reasons.push(reason)
  }

  const latencyIncrease = relativeIncrease(current.modelLatencyMsPer1kOutputTokens, baseline.modelLatencyMsPer1kOutputTokens)
  if (latencyIncrease >= 0.55) {
    mark('critical', 'modelLatencyMsPer1kOutputTokens', 'model latency per 1k output tokens', 'higher_worse', 'Latency rose vs baseline', current.modelLatencyMsPer1kOutputTokens, baseline.modelLatencyMsPer1kOutputTokens)
  } else if (latencyIncrease >= 0.22) {
    mark('warning', 'modelLatencyMsPer1kOutputTokens', 'model latency per 1k output tokens', 'higher_worse', 'Latency rose vs baseline', current.modelLatencyMsPer1kOutputTokens, baseline.modelLatencyMsPer1kOutputTokens)
  }

  const throughputDrop = relativeDecrease(current.modelThroughputTokensPerSecond, baseline.modelThroughputTokensPerSecond)
  if (throughputDrop >= 0.42) {
    mark('critical', 'modelThroughputTokensPerSecond', 'model throughput', 'lower_worse', 'Throughput fell vs baseline', current.modelThroughputTokensPerSecond, baseline.modelThroughputTokensPerSecond)
  } else if (throughputDrop >= 0.2) {
    mark('warning', 'modelThroughputTokensPerSecond', 'model throughput', 'lower_worse', 'Throughput fell vs baseline', current.modelThroughputTokensPerSecond, baseline.modelThroughputTokensPerSecond)
  }

  const outputThroughputDrop = relativeDecrease(current.modelThroughputOutputTokensPerSecond, baseline.modelThroughputOutputTokensPerSecond)
  if (outputThroughputDrop >= 0.24) {
    mark(outputThroughputDrop >= 0.5 ? 'critical' : 'warning', 'modelThroughputOutputTokensPerSecond', 'model output throughput', 'lower_worse', 'Output throughput fell', current.modelThroughputOutputTokensPerSecond, baseline.modelThroughputOutputTokensPerSecond)
  }

  if (current.failedToolCalls > baseline.failedToolCalls && current.toolFailureRate >= 0.08) {
    mark(current.toolFailureRate >= 0.2 ? 'critical' : 'warning', 'toolFailureRate', 'tool failure rate', 'higher_downstream_symptom', 'Tool failures above baseline', current.toolFailureRate, baseline.toolFailureRate)
  }

  if (current.cacheMissRate - baseline.cacheMissRate >= 0.12) {
    mark('warning', 'cacheMissRate', 'cache miss rate', 'higher_symptom', 'Cache misses above baseline', current.cacheMissRate, baseline.cacheMissRate)
  }

  const reasoningOverheadDelta = reasoningOverhead(current) - reasoningOverhead(baseline)
  if (reasoningOverheadDelta >= 0.12) {
    mark('warning', 'reasoningOverheadRate', 'reasoning overhead', 'cost_shape_review', 'Reasoning overhead rose', reasoningOverhead(current), reasoningOverhead(baseline))
  }

  const degradationRiskDelta = current.degradationRiskScore - baseline.degradationRiskScore
  if (current.degradationRiskScore >= 0.3 && degradationRiskDelta >= 0.3) {
    mark('critical', 'degradationRiskScore', 'model quality risk score', 'higher_worse', 'Model quality risk rose', current.degradationRiskScore, baseline.degradationRiskScore)
  } else if (current.degradationRiskScore >= 0.3 && degradationRiskDelta >= 0.15) {
    mark('warning', 'degradationRiskScore', 'model quality risk score', 'higher_worse', 'Model quality risk rose', current.degradationRiskScore, baseline.degradationRiskScore)
  }

  const uniqueReasons = [...new Set(reasons)]
  return {
    severity,
    confidence: current.sessionCount < 2 || current.modelCalls < 3 ? 'low' : 'high',
    sampleNote: current.sessionCount < 2 || current.modelCalls < 3 ? 'Low sample' : undefined,
    reasons: uniqueReasons,
    metrics
  }
}

function severityRank(value?: string): number {
  const normalized = (value || '').toLowerCase()
  if (normalized === 'critical' || normalized === 'high') return 3
  if (normalized === 'warning' || normalized === 'medium') return 2
  if (normalized === 'watch' || normalized === 'low') return 1
  return 0
}

function sourceIdentityKey(record: SourceIdentity): string {
  return record.sourceKey || (record.sourceId !== undefined ? `source:${record.sourceId}` : '')
}

function cohortKeyFor(session: Session): string {
  return [
    session.modelProvider || 'unknown',
    session.model || 'unknown',
    sourceIdentityKey(session) || session.agentKind || session.agentName || 'unknown',
    projectPathKey(session.projectPath || session.rawSourcePath)
  ].join('|')
}

function modelSignalCohortsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalCohort[] {
  return [...groupedBy(items, cohortKeyFor)].map(([cohortKey, group]) => {
    const first = group[0]
    const sessionIds = new Set(group.map((session) => session.id))
    const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
    const current = metricSetFor(group, groupToolCalls)
    const baseline = syntheticBaselineFor(current, cohortKey)
    const drift = modelSignalDriftFor(current, baseline)
    return {
      sourceId: first.sourceId,
      sourceKey: first.sourceKey,
      sourceLabel: first.sourceLabel,
      sourceRootPath: first.sourceRootPath,
      sourceSessionsPath: first.sourceSessionsPath,
      agentKind: first.agentKind,
      agentName: first.agentName,
      modelProvider: first.modelProvider,
      model: first.model,
      projectPath: first.projectPath,
      cohortKey,
      sessionCount: current.sessionCount,
      modelCalls: current.modelCalls,
      toolCalls: current.toolCalls,
      failedToolCalls: current.failedToolCalls,
      totalTokens: current.totalTokens,
      current,
      baseline,
      drift
    }
  }).sort((left, right) =>
    severityRank(right.drift.severity) - severityRank(left.drift.severity) ||
    right.totalTokens - left.totalTokens
  )
}

function combineMetricSets(items: ModelSignalMetricSet[]): ModelSignalMetricSet {
  const sessionCount = items.reduce((total, item) => total + item.sessionCount, 0)
  const modelCalls = items.reduce((total, item) => total + item.modelCalls, 0)
  const allPriced = items.every((item) => item.estimatedCostUsd !== undefined)
  const allSavingsPriced = items.every((item) => item.cacheSavingsUsd !== undefined)
  const latencySamples = items.flatMap((item) => [
    item.p50ModelLatencyMsPer1kOutputTokens,
    item.p90ModelLatencyMsPer1kOutputTokens,
    item.modelLatencyMsPer1kOutputTokens
  ].filter((value): value is number => typeof value === 'number' && Number.isFinite(value)))
  const throughputSamples = items.flatMap((item) => [
    item.p10ModelThroughputTokensPerSecond,
    item.p50ModelThroughputTokensPerSecond,
    item.modelThroughputTokensPerSecond
  ].filter((value): value is number => typeof value === 'number' && Number.isFinite(value)))
  const toolDependencyRate = safeRate(
    items.reduce((total, item) => total + item.toolDependencyRate * item.sessionCount, 0),
    sessionCount
  )
  return metricSetFromTotals({
    sessionCount,
    modelCalls,
    failedModelCalls: items.reduce((total, item) => total + (item.failedModelCalls || 0), 0),
    toolCalls: items.reduce((total, item) => total + item.toolCalls, 0),
    failedToolCalls: items.reduce((total, item) => total + item.failedToolCalls, 0),
    totalTokens: items.reduce((total, item) => total + item.totalTokens, 0),
    inputTokens: items.reduce((total, item) => total + item.inputTokens, 0),
    cachedInputTokens: items.reduce((total, item) => total + item.cachedInputTokens, 0),
    outputTokens: items.reduce((total, item) => total + item.outputTokens, 0),
    reasoningOutputTokens: items.reduce((total, item) => total + item.reasoningOutputTokens, 0),
    modelDurationMs: items.reduce((total, item) => total + item.modelDurationMs, 0),
    wallDurationMs: items.reduce((total, item) => total + (item.wallDurationMs || item.modelDurationMs), 0),
    activeDurationMs: items.reduce((total, item) => total + (item.activeDurationMs || item.modelDurationMs), 0),
    toolDurationMs: items.reduce((total, item) => total + (item.toolDurationMs || 0), 0),
    idleDurationMs: items.reduce((total, item) => total + (item.idleDurationMs || 0), 0),
    estimatedCostUsd: allPriced ? Number(items.reduce((total, item) => total + (item.estimatedCostUsd || 0), 0).toFixed(4)) : undefined,
    unpricedSessionCount: items.reduce((total, item) => total + (item.unpricedSessionCount || 0), 0),
    cacheSavingsUsd: allSavingsPriced ? Number(items.reduce((total, item) => total + (item.cacheSavingsUsd || 0), 0).toFixed(4)) : undefined,
    latencySamples,
    throughputSamples,
    toolDependencyRate
  })
}

function modelSignalMatrixFor(cohorts: ModelSignalCohort[]): ModelSignalMatrixRow[] {
  return [...groupedBy(cohorts, (cohort) => sourceIdentityKey(cohort) || cohort.agentKind || cohort.agentName || 'unknown')]
    .map(([, group]) => {
      const first = group[0]
      const cells: ModelSignalMatrixCell[] = [...groupedBy(group, (cohort) => `${cohort.modelProvider}:${cohort.model}`)]
        .map(([, cellCohorts]) => {
          const cellFirst = cellCohorts[0]
          const current = combineMetricSets(cellCohorts.map((cohort) => cohort.current))
          const baseline = combineMetricSets(cellCohorts.map((cohort) => cohort.baseline))
          const drift = modelSignalDriftFor(current, baseline)
          return {
            model: cellFirst.model,
            modelProvider: cellFirst.modelProvider,
            cohortCount: cellCohorts.length,
            sessionCount: current.sessionCount,
            modelCalls: current.modelCalls,
            totalTokens: current.totalTokens,
            severity: drift.severity,
            confidence: drift.confidence,
            keyReason: drift.reasons[0],
            drift,
            current,
            baseline
          }
        })
        .sort((left, right) => severityRank(right.severity) - severityRank(left.severity) || right.totalTokens - left.totalTokens)
      return {
        sourceId: first.sourceId,
        sourceKey: first.sourceKey,
        sourceLabel: first.sourceLabel,
        sourceRootPath: first.sourceRootPath,
        sourceSessionsPath: first.sourceSessionsPath,
        agentKind: first.agentKind,
        agentName: first.agentName,
        cells
      }
    })
    .sort((left, right) =>
      Math.max(...right.cells.map((cell) => severityRank(cell.severity)), 0) -
      Math.max(...left.cells.map((cell) => severityRank(cell.severity)), 0)
    )
}

function modelSignalProjectHotspotsFor(cohorts: ModelSignalCohort[]): ModelSignalProjectHotspot[] {
  return [...groupedBy(cohorts, (cohort) => projectPathKey(cohort.projectPath || 'unknown'))].map(([, group]) => {
    const current = combineMetricSets(group.map((cohort) => cohort.current))
    const baseline = combineMetricSets(group.map((cohort) => cohort.baseline))
    const drift = modelSignalDriftFor(current, baseline)
    return {
      ...current,
      projectPath: group[0].projectPath || 'unknown',
      modelCount: new Set(group.map((cohort) => `${cohort.modelProvider}:${cohort.model}`)).size,
      sourceCount: new Set(group.map((cohort) => sourceIdentityKey(cohort) || cohort.agentKind || cohort.agentName)).size,
      current,
      baseline,
      drift
    }
  }).sort((left, right) =>
    severityRank(right.drift.severity) - severityRank(left.drift.severity) ||
    right.totalTokens - left.totalTokens
  )
}

function modelSignalsDailyMetricsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsDailyMetric[] {
  return [...groupedBy(items, (session) => session.startedAt.slice(0, 10))]
    .map(([date, group]) => {
      const sessionIds = new Set(group.map((session) => session.id))
      const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
      const current = metricSetFor(group, groupToolCalls)
      const baseline = syntheticBaselineFor(current, `daily:${date}`)
      const drift = modelSignalDriftFor(current, baseline)
      return {
        date,
        ...current,
        baseline,
        lowSample: group.length < 2 || current.modelCalls < 3 || current.totalTokens < 60_000,
        drift,
        keyReason: drift.reasons[0] || drift.sampleNote
      }
    })
    .sort((left, right) => right.date.localeCompare(left.date))
}

function modelSignalsProjectMetricsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsProjectMetric[] {
  return [...groupedBy(items, (session) => projectPathKey(session.projectPath || session.rawSourcePath))]
    .map(([, group]) => {
      const first = group[0]
      const sessionIds = new Set(group.map((session) => session.id))
      const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
      const current = metricSetFor(group, groupToolCalls)
      const baseline = syntheticBaselineFor(current, `project:${first.projectPath || first.rawSourcePath}`)
      const drift = modelSignalDriftFor(current, baseline)
      const dominantModelGroup = [...groupedBy(group, (session) => `${session.modelProvider}:${session.model}`)]
        .map(([, modelGroup]) => ({
          modelProvider: modelGroup[0].modelProvider,
          model: modelGroup[0].model,
          sessionCount: modelGroup.length,
          totalTokens: sum(modelGroup, (session) => session.tokenUsage.totalTokens)
        }))
        .sort((left, right) => right.sessionCount - left.sessionCount || right.totalTokens - left.totalTokens)[0]

      return {
        ...current,
        projectPath: first.projectPath || first.rawSourcePath || 'unknown',
        modelCount: new Set(group.map((session) => `${session.modelProvider}:${session.model}`)).size,
        sourceCount: new Set(group.map((session) => sourceIdentityKey(session) || session.agentKind || session.agentName)).size,
        dominantModelProvider: dominantModelGroup?.modelProvider,
        dominantModel: dominantModelGroup?.model,
        dominantModelShare: safeRate(dominantModelGroup?.sessionCount || 0, group.length),
        current,
        baseline,
        drift
      }
    })
    .sort((left, right) =>
      severityRank(right.drift.severity) - severityRank(left.drift.severity) ||
      (right.estimatedCostUsd || 0) - (left.estimatedCostUsd || 0) ||
      right.totalTokens - left.totalTokens
    )
}

function modelSignalsHealthSummaryFor(items: Session[], cohorts: ModelSignalCohort[]): ModelSignalsHealthSummary {
  const reasonCounts = new Map<string, number>()
  for (const cohort of cohorts) {
    for (const reason of cohort.drift.reasons) {
      reasonCounts.set(reason, (reasonCounts.get(reason) || 0) + 1)
    }
  }
  const criticalCohorts = cohorts.filter((cohort) => severityRank(cohort.drift.severity) >= 3).length
  const warningCohorts = cohorts.filter((cohort) => severityRank(cohort.drift.severity) === 2).length
  const lowConfidenceCohorts = cohorts.filter((cohort) => cohort.drift.confidence === 'low').length
  return {
    currentWindow: dateWindow(items),
    baselineWindow: baselineWindow(items),
    severity: criticalCohorts > 0 ? 'critical' : warningCohorts > 0 ? 'warning' : lowConfidenceCohorts > 0 ? 'unknown' : 'healthy',
    cohortCount: cohorts.length,
    warningCohorts,
    criticalCohorts,
    lowConfidenceCohorts,
    topReasons: [...reasonCounts.entries()]
      .sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0]))
      .map(([reason]) => reason)
      .slice(0, 5)
  }
}

function dateWindow(items: Session[]): ModelSignalsWindow {
  const dates = [...new Set(items.map((session) => session.startedAt.slice(0, 10)))].sort()
  return {
    from: dates[0] ? `${dates[0]}T00:00:00Z` : '',
    to: dates[dates.length - 1] ? `${dates[dates.length - 1]}T23:59:59Z` : '',
    sessionCount: items.length,
    modelCalls: sum(items, modelCallsForSession)
  }
}

function baselineWindow(items: Session[]): ModelSignalsWindow {
  const dates = [...new Set(items.map((session) => session.startedAt.slice(0, 10)))].sort()
  if (!dates.length) {
    return { from: '', to: '', sessionCount: 0, modelCalls: 0 }
  }
  const first = new Date(`${dates[0]}T00:00:00Z`)
  const last = new Date(`${dates[dates.length - 1]}T00:00:00Z`)
  const spanDays = Math.max(1, Math.round((last.getTime() - first.getTime()) / 86_400_000) + 1)
  const baselineEnd = new Date(first.getTime() - 86_400_000)
  const baselineStart = new Date(first.getTime() - spanDays * 86_400_000)
  return {
    from: baselineStart.toISOString(),
    to: baselineEnd.toISOString(),
    sessionCount: items.length,
    modelCalls: sum(items, modelCallsForSession)
  }
}

function modelSignalsTrendFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsTrendPoint[] {
  const days = [...groupedBy(items, (session) => session.startedAt.slice(0, 10))]
    .map(([date, group]) => signalTrendPoint(date, group, scopedToolCalls))
    .sort((left, right) => left.date.localeCompare(right.date))

  return days.map((day, index) => {
    const window = days.slice(Math.max(0, index - 6), index + 1)
    const windowToolCalls = window.reduce((total, item) => total + item.toolCalls, 0)
    const windowFailedTools = window.reduce((total, item) => total + item.failedToolCalls, 0)
    const windowTokens = window.reduce((total, item) => total + item.totalTokens, 0)
    const windowDurationSeconds = window.reduce((total, item) => total + item.modelDurationMs, 0) / 1000
    return {
      ...day,
      rollingModelThroughputTokensPerSecond: safeRate(windowTokens, windowDurationSeconds),
      rollingToolFailureRate: safeRate(windowFailedTools, windowToolCalls)
    }
  })
}

function signalTrendPoint(date: string, group: Session[], scopedToolCalls: ToolCall[]): ModelSignalsTrendPoint {
  const sessionIds = new Set(group.map((session) => session.id))
  const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
  const inputTokens = sum(group, (session) => session.tokenUsage.inputTokens)
  const cachedInputTokens = sum(group, (session) => session.tokenUsage.cachedInputTokens)
  const outputTokens = sum(group, (session) => session.tokenUsage.outputTokens)
  const reasoningOutputTokens = sum(group, (session) => session.tokenUsage.reasoningOutputTokens)
  const totalTokens = sum(group, (session) => session.tokenUsage.totalTokens)
  const modelDurationMs = sum(group, (session) => session.modelDurationMs)
  const modelCalls = sum(group, modelCallsForSession)
  const failedToolCalls = groupToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length
  return {
    date,
    sessionCount: group.length,
    modelCalls,
    toolCalls: groupToolCalls.length,
    failedToolCalls,
    totalTokens,
    inputTokens,
    cachedInputTokens,
    outputTokens,
    reasoningOutputTokens,
    modelDurationMs,
    ...signalRatesFor(group, groupToolCalls),
    rollingModelThroughputTokensPerSecond: 0,
    rollingToolFailureRate: 0,
    lowSample: group.length < 2 || modelCalls < 2 || totalTokens < 60_000
  }
}

function modelSignalsBreakdownFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalBreakdown[] {
  return [...groupedBy(items, (session) => session.model)].map(([model, group]) => {
    const sessionIds = new Set(group.map((session) => session.id))
    const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
    return {
      model,
      sessionCount: group.length,
      modelCalls: sum(group, modelCallsForSession),
      toolCalls: groupToolCalls.length,
      failedToolCalls: groupToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length,
      totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
      inputTokens: sum(group, (session) => session.tokenUsage.inputTokens),
      cachedInputTokens: sum(group, (session) => session.tokenUsage.cachedInputTokens),
      outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
      reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
      modelDurationMs: sum(group, (session) => session.modelDurationMs),
      ...signalRatesFor(group, groupToolCalls)
    }
  }).sort((left, right) => right.totalTokens - left.totalTokens)
}

function anomalySessionsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalAnomalySession[] {
  const anomalies: ModelSignalAnomalySession[] = []
  for (const session of items) {
    const sessionToolCalls = scopedToolCalls.filter((call) => call.sessionId === session.id)
    const rates = signalRatesFor([session], sessionToolCalls)
    const reasons: string[] = []
    if (rates.toolFailureRate > 0) reasons.push('Tool failure in session')
    if (reasoningOverhead(rates) >= 0.25) reasons.push('High reasoning overhead')
    if (rates.outputExpansionRate >= 0.2) reasons.push('Generation overhead relative to input')
    if (rates.cacheMissRate >= 0.85) reasons.push('Low cache reuse')
    if (rates.modelThroughputTokensPerSecond > 0 && rates.modelThroughputTokensPerSecond < 85) reasons.push('Low model token throughput')
    if (!reasons.length) continue
    anomalies.push({
      sessionId: session.id,
      sessionKey: session.sessionKey,
      codexSessionId: session.codexSessionId,
      startedAt: session.startedAt,
      projectPath: session.projectPath,
      rawSourcePath: session.rawSourcePath,
      agentKind: session.agentKind,
      agentName: session.agentName,
      sourceId: session.sourceId || 0,
      sourceKey: session.sourceKey || '',
      sourceLabel: session.sourceLabel || session.agentName,
      sourceRootPath: session.sourceRootPath || '',
      sourceSessionsPath: session.sourceSessionsPath || '',
      model: session.model,
      modelCalls: modelCallsForSession(session),
      totalTokens: session.tokenUsage.totalTokens,
      inputTokens: session.tokenUsage.inputTokens,
      cachedInputTokens: session.tokenUsage.cachedInputTokens,
      outputTokens: session.tokenUsage.outputTokens,
      reasoningOutputTokens: session.tokenUsage.reasoningOutputTokens,
      visibleOutputTokens: Math.max(0, session.tokenUsage.outputTokens - session.tokenUsage.reasoningOutputTokens),
      billableOutputTokens: session.tokenUsage.outputTokens,
      toolCalls: sessionToolCalls.length,
      failedToolCalls: sessionToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length,
      modelDurationMs: session.modelDurationMs,
      reasons,
      score: clampRate(reasons.length / 5),
      ...rates
    })
  }
  return anomalies
    .sort((left, right) => {
      const leftReasons = Array.isArray(left.reasons) ? left.reasons.length : 0
      const rightReasons = Array.isArray(right.reasons) ? right.reasons.length : 0
      return rightReasons - leftReasons || (right.totalTokens || 0) - (left.totalTokens || 0)
    })
    .slice(0, 6)
}

export function modelSignals(filters: UsageScopeFilters = {}): ModelSignals {
  const scoped = filteredSessions(filters)
  const sessionIds = new Set(scoped.map((session) => session.id))
  const scopedToolCalls = filteredToolCalls({ agent: filters.agent, project: filters.project, from: filters.from, to: filters.to })
    .filter((call) => sessionIds.has(call.sessionId))
  const rates = signalRatesFor(scoped, scopedToolCalls)
  const outputTokens = sum(scoped, (session) => session.tokenUsage.outputTokens)
  const reasoningOutputTokens = sum(scoped, (session) => session.tokenUsage.reasoningOutputTokens)
  const cohorts = modelSignalCohortsFor(scoped, scopedToolCalls)
  return {
    totalSessions: scoped.length,
    totalModelCalls: sum(scoped, modelCallsForSession),
    totalToolCalls: scopedToolCalls.length,
    failedToolCalls: scopedToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length,
    toolFailureRate: rates.toolFailureRate,
    toolDependencyRate: rates.toolDependencyRate,
    avgModelCallsPerSession: safeRate(sum(scoped, modelCallsForSession), scoped.length),
    outputExpansionRate: rates.outputExpansionRate,
    reasoningTokenShare: rates.reasoningTokenShare,
    reasoningOverheadRate: rates.reasoningOverheadRate,
    visibleOutputTokens: Math.max(0, outputTokens - reasoningOutputTokens),
    billableOutputTokens: outputTokens,
    cacheMissRate: rates.cacheMissRate,
    modelThroughputTokensPerSecond: rates.modelThroughputTokensPerSecond,
    modelThroughputOutputTokensPerSecond: rates.modelThroughputOutputTokensPerSecond,
    trend: modelSignalsTrendFor(scoped, scopedToolCalls),
    modelBreakdown: modelSignalsBreakdownFor(scoped, scopedToolCalls),
    anomalySessions: anomalySessionsFor(scoped, scopedToolCalls),
    healthSummary: modelSignalsHealthSummaryFor(scoped, cohorts),
    cohorts,
    matrix: modelSignalMatrixFor(cohorts),
    projectHotspots: modelSignalProjectHotspotsFor(cohorts),
    dailyMetrics: modelSignalsDailyMetricsFor(scoped, scopedToolCalls),
    projectMetrics: modelSignalsProjectMetricsFor(scoped, scopedToolCalls)
  }
}
