import type {
  ModelSignals,
  UsageScopeFilters
} from '../types'
import { filteredSessions, filteredToolCalls } from './sessions'
import {
  anomalySessionsFor,
  modelSignalCohortsFor,
  modelSignalMatrixFor,
  modelSignalProjectHotspotsFor,
  modelSignalsBreakdownFor,
  modelSignalsDailyMetricsFor,
  modelSignalsHealthSummaryFor,
  modelSignalsProjectMetricsFor,
  modelSignalsTrendFor
} from './modelSignalsCollections'
import {
  isSuccessfulToolStatus,
  modelCallsForSession,
  safeRate,
  signalRatesFor
} from './modelSignalsMetrics'
import { sum } from './utils'

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
