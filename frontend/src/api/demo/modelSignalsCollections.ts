import type {
  ModelSignalAnomalySession,
  ModelSignalBreakdown,
  ModelSignalCohort,
  ModelSignalMatrixCell,
  ModelSignalMatrixRow,
  ModelSignalRates,
  ModelSignalsDailyMetric,
  ModelSignalsHealthSummary,
  ModelSignalsProjectMetric,
  ModelSignalProjectHotspot,
  ModelSignalsTrendPoint,
  ModelSignalsWindow,
  Session,
  SourceIdentity,
  ToolCall
} from '../types'
import { modelSignalDriftFor, reasoningOverhead, severityRank } from './modelSignalsDrift'
import {
  clampRate,
  combineMetricSets,
  isSuccessfulToolStatus,
  metricSetFor,
  modelCallsForSession,
  safeRate,
  signalRatesFor,
  syntheticBaselineFor
} from './modelSignalsMetrics'
import { groupedBy, projectPathKey, sum } from './utils'

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

export function modelSignalCohortsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalCohort[] {
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

export function modelSignalMatrixFor(cohorts: ModelSignalCohort[]): ModelSignalMatrixRow[] {
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

export function modelSignalProjectHotspotsFor(cohorts: ModelSignalCohort[]): ModelSignalProjectHotspot[] {
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

export function modelSignalsDailyMetricsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsDailyMetric[] {
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

export function modelSignalsProjectMetricsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsProjectMetric[] {
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

export function modelSignalsHealthSummaryFor(items: Session[], cohorts: ModelSignalCohort[]): ModelSignalsHealthSummary {
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

export function modelSignalsTrendFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsTrendPoint[] {
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

export function modelSignalsBreakdownFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalBreakdown[] {
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

export function anomalySessionsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalAnomalySession[] {
  const anomalies: ModelSignalAnomalySession[] = []
  for (const session of items) {
    const sessionToolCalls = scopedToolCalls.filter((call) => call.sessionId === session.id)
    const rates = signalRatesFor([session], sessionToolCalls)
    const reasons = anomalyReasonsFor(rates)
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

function anomalyReasonsFor(rates: ModelSignalRates): string[] {
  return [
    { reason: 'Tool failure in session', active: rates.toolFailureRate > 0 },
    { reason: 'High reasoning overhead', active: reasoningOverhead(rates) >= 0.25 },
    { reason: 'Generation overhead relative to input', active: rates.outputExpansionRate >= 0.2 },
    { reason: 'Low cache reuse', active: rates.cacheMissRate >= 0.85 },
    {
      reason: 'Low model token throughput',
      active: rates.modelThroughputTokensPerSecond > 0 && rates.modelThroughputTokensPerSecond < 85
    }
  ]
    .filter((item) => item.active)
    .map((item) => item.reason)
}
