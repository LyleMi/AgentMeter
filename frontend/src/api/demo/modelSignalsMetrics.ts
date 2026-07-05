import type {
  ModelSignalMetricSet,
  ModelSignalRates,
  Session,
  ToolCall
} from '../types'
import { cacheSavingsUsdFor, costSum } from './usageMetrics'
import { sum } from './utils'

export function modelCallsForSession(session: Session): number {
  return Math.max(1, Math.ceil(session.eventCount / 55))
}

export function safeRate(numerator: number, denominator: number): number {
  return denominator > 0 ? numerator / denominator : 0
}

export function clampRate(value: number): number {
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
  return weightedRiskScore([
    { score: thresholdScore(latency, 8_000, 20_000), weight: 0.24 },
    { score: inverseThresholdScore(throughput, 40, 12), weight: 0.24 },
    { score: rangeScore(numberOrZero(metric.failurePressure), 0.05, 0.95), weight: 0.18 },
    { score: rangeScore(numberOrZero(metric.toolFailureRate), 0.08, 0.42), weight: 0.10 },
    { score: rangeScore(numberOrZero(metric.cacheMissRate), 0.70, 0.30), weight: 0.08 },
    { score: rangeScore(numberOrZero(metric.avgModelCallsPerSession), 1.5, 2.5), weight: 0.07 },
    { score: rangeScore(numberOrZero(metric.outputExpansionRate), 3.0, 5.0), weight: 0.05 },
    { score: rangeScore(numberOrZero(metric.reasoningOverheadRate), 1.0, 4.0), weight: 0.04 }
  ])
}

function weightedRiskScore(contributions: Array<{ score: number; weight: number }>): number {
  return clampRate(contributions.reduce((total, item) => total + item.score * item.weight, 0))
}

function numberOrZero(value: number | undefined): number {
  return value ?? 0
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
  return values.find((value) => Number.isFinite(value) && (value ?? 0) > 0) ?? 0
}

function sessionLatencyMsPer1kOutputTokens(session: Session): number {
  return safeRate(session.modelDurationMs, session.tokenUsage.outputTokens / 1000)
}

function sessionThroughputTokensPerSecond(session: Session): number {
  return safeRate(session.tokenUsage.totalTokens, session.modelDurationMs / 1000)
}

export function isSuccessfulToolStatus(status: string): boolean {
  return status === 'completed' || status === 'success'
}

export function signalRatesFor(group: Session[], groupToolCalls: ToolCall[]): ModelSignalRates {
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

export function metricSetFor(group: Session[], groupToolCalls: ToolCall[]): ModelSignalMetricSet {
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

export function syntheticBaselineFor(current: ModelSignalMetricSet, key: string): ModelSignalMetricSet {
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

export function combineMetricSets(items: ModelSignalMetricSet[]): ModelSignalMetricSet {
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
