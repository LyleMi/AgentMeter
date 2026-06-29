import {
  formatCost,
  formatDisplayNumber,
  formatNumber,
  projectDisplay,
  sessionDisplay,
  type ModelSignalAnomalySession,
  type ModelSignalCohort,
  type ModelSignalMatrixCell,
  type ModelSignalMatrixRow,
  type ModelSignalMetricSet,
  type ModelSignalsDailyMetric,
  type ModelSignals,
  type ModelSignalsHealthSummary,
  type ModelSignalsProjectMetric,
  type ModelSignalsWindow
} from '../../api'
import { sourceDisplay } from '../../presentation/sourceIdentity'
import {
  formatModelSignalPercent as formatPercent,
  formatModelSignalRate as formatRate
} from '../../presentation/modelSignals'
import type { ModelSignalsTranslate } from './messages'
import type { NormalizedAnomalySession, ProjectMetricRow } from './types'

export function createModelSignalsDisplay(t: ModelSignalsTranslate) {
  function fallbackHealthSummary(item: ModelSignals | null | undefined): ModelSignalsHealthSummary {
    const hasToolFailures = Boolean(item?.failedToolCalls)
    return {
      currentWindow: emptyWindow(),
      baselineWindow: emptyWindow(),
      severity: hasToolFailures ? 'warning' : 'ok',
      cohortCount: item?.modelBreakdown?.length || 0,
      warningCohorts: hasToolFailures ? 1 : 0,
      criticalCohorts: 0,
      lowConfidenceCohorts: 0,
      topReasons: hasToolFailures ? ['Tool failures above baseline'] : []
    }
  }

  function displayText(text: string) {
    return { main: text, full: text }
  }

  function displayPair(left?: number, right?: number) {
    const leftDisplay = formatDisplayNumber(left)
    const rightDisplay = formatDisplayNumber(right)
    return {
      main: `${leftDisplay.main} / ${rightDisplay.main}`,
      full: `${leftDisplay.full} / ${rightDisplay.full}`
    }
  }

  function displayPercent(value?: number) {
    const text = formatPercent(value)
    return { main: text, full: text }
  }

  function displayRate(value?: number, suffix = '', digits = 0) {
    const text = `${formatRate(value, digits)}${suffix}`
    return { main: text, full: text }
  }

  function formatLatency(value?: number) {
    return `${formatRate(value, 0)} ms/1k`
  }

  function formatOptionalCost(value?: number) {
    if (value === undefined || value === null) return '-'
    return formatCost(value)
  }

  function formatThroughput(value?: number) {
    return `${formatRate(value, 1)} tok/s`
  }

  function formatPressure(value?: number) {
    return `${formatRate(value, 2)}/session`
  }

  function p90Latency(metric?: ModelSignalMetricSet) {
    return metric?.p90ModelLatencyMsPer1kOutputTokens ?? metric?.modelLatencyMsPer1kOutputTokens
  }

  function p10Throughput(metric?: ModelSignalMetricSet) {
    return metric?.p10ModelThroughputTokensPerSecond ?? metric?.modelThroughputTokensPerSecond
  }

  function failurePressure(metric?: ModelSignalMetricSet) {
    return metric?.failurePressure ?? safeMetricRate(metric?.failedToolCalls, metric?.sessionCount)
  }

  function safeMetricRate(numerator?: number, denominator?: number) {
    return denominator && denominator > 0 ? (numerator || 0) / denominator : 0
  }

  function unpricedNote(metric?: ModelSignalMetricSet) {
    const count = metric?.unpricedSessionCount || 0
    return count > 0 ? `${formatNumber(count)} ${t('label.unpriced')}` : ''
  }

  function confidenceReason(record: Pick<ModelSignalsDailyMetric, 'keyReason' | 'drift' | 'lowSample'>) {
    return record.keyReason || record.drift?.reasons?.[0] || record.drift?.sampleNote || (record.lowSample ? t('label.lowSample') : t('fallback.noReason'))
  }

  function formatConfidence(value?: string | number) {
    if (typeof value === 'number') return formatPercent(value)
    const normalized = (value || '').trim().toLowerCase()
    if (!normalized) return t('fallback.unknown')
    return normalized
  }

  function emptyWindow(): ModelSignalsWindow {
    return {
      from: '',
      to: '',
      sessionCount: 0,
      modelCalls: 0
    }
  }

  function formatWindow(window?: ModelSignalsWindow) {
    if (!window?.from && !window?.to) return t('fallback.unknown')
    const from = window.from ? window.from.slice(0, 10) : ''
    const to = window.to ? window.to.slice(0, 10) : ''
    const range = from && to && from !== to ? `${from} - ${to}` : from || to
    if (!range) return t('fallback.unknown')
    return `${range}, ${formatNumber(window.sessionCount || 0)} ${t('column.sessions')}`
  }

  function metricClass(current?: number, baseline?: number, lowerIsBetter = false) {
    if (!Number.isFinite(current) || !Number.isFinite(baseline) || !baseline) return ''
    const degraded = lowerIsBetter ? (current || 0) > (baseline || 0) * 1.15 : (current || 0) < (baseline || 0) * 0.85
    const improved = lowerIsBetter ? (current || 0) < (baseline || 0) * 0.9 : (current || 0) > (baseline || 0) * 1.1
    if (degraded) return 'status-error'
    if (improved) return 'status-ok'
    return ''
  }

  function severityRank(value?: string): number {
    const normalized = (value || '').toLowerCase()
    if (normalized === 'critical' || normalized === 'high') return 3
    if (normalized === 'warning' || normalized === 'medium') return 2
    if (normalized === 'watch' || normalized === 'low' || normalized === 'unknown') return 1
    return 0
  }

  function severityLabel(value?: string) {
    const normalized = (value || 'ok').toLowerCase()
    if (normalized === 'critical') return t('severity.critical')
    if (normalized === 'warning') return t('severity.warning')
    if (normalized === 'watch') return t('severity.watch')
    if (normalized === 'healthy') return t('severity.healthy')
    if (normalized === 'unknown') return t('severity.unknown')
    if (normalized === 'high') return t('severity.high')
    if (normalized === 'medium') return t('severity.medium')
    if (normalized === 'low') return t('severity.low')
    if (normalized === 'ok') return t('severity.ok')
    return normalized
  }

  function severityTagColor(value?: string) {
    const rank = severityRank(value)
    if (rank >= 3) return 'error'
    if (rank === 2) return 'warning'
    if (rank === 1) return 'processing'
    return 'success'
  }

  function severityMetricTone(value?: string) {
    const rank = severityRank(value)
    if (rank >= 3) return 'metric-danger'
    if (rank === 2) return 'metric-warning'
    if (rank === 1) return 'metric-info'
    return 'metric-success'
  }

  function severityClass(value?: string) {
    const rank = severityRank(value)
    if (rank >= 3) return 'severity-critical'
    if (rank === 2) return 'severity-warning'
    if (rank === 1) return 'severity-watch'
    return 'severity-ok'
  }

  function driftRowClass(record: { drift?: { severity?: string } }) {
    const rank = severityRank(record.drift?.severity)
    return { class: rank >= 3 ? 'model-signals-critical-row' : rank === 2 ? 'model-signals-warning-row' : '' }
  }

  function anomalyRowClass(record: NormalizedAnomalySession) {
    return { class: record.failedToolCalls > 0 || record.score >= 0.45 ? 'model-signals-warning-row' : '' }
  }

  function reasonText(row: string): string {
    return row
  }

  function reasonCount(_row: string): number | undefined {
    return undefined
  }

  function reasonSeverity(_row: string): string | undefined {
    return undefined
  }

  function sourceInfo(record: Parameters<typeof sourceDisplay>[0]) {
    return sourceDisplay(record, t('fallback.unknown'))
  }

  function projectInfo(record: { projectPath?: string; rawSourcePath?: string }) {
    return projectDisplay(record.projectPath || record.rawSourcePath)
  }

  function projectMixInfo(record: ProjectMetricRow) {
    const projectMetric = record as Partial<ModelSignalsProjectMetric>
    const model = projectMetric.dominantModel || t('fallback.unknown')
    const provider = projectMetric.dominantModelProvider || ''
    const share = projectMetric.dominantModelShare !== undefined ? formatPercent(projectMetric.dominantModelShare) : ''
    const summary = [
      share,
      `${formatNumber(record.modelCount)} ${t('column.models')}`,
      `${formatNumber(record.sourceCount)} ${t('column.sources')}`
    ].filter(Boolean).join(' · ')
    return {
      model,
      provider,
      summary,
      full: [provider, model, summary].filter(Boolean).join(' / ')
    }
  }

  function projectHealthTitle(record: ProjectMetricRow) {
    return [
      `${t('column.health')}: ${severityLabel(record.drift?.severity)} (${formatConfidence(record.drift?.confidence)})`,
      `${t('column.p90Latency')}: ${formatLatency(p90Latency(record.current))} / ${t('metric.baseline')} ${formatLatency(p90Latency(record.baseline))}`,
      `${t('column.p10Throughput')}: ${formatThroughput(p10Throughput(record.current))} / ${t('metric.baseline')} ${formatThroughput(p10Throughput(record.baseline))}`,
      `${t('column.failurePressure')}: ${formatPressure(failurePressure(record.current))} / ${t('metric.baseline')} ${formatPressure(failurePressure(record.baseline))}`
    ].join('\n')
  }

  function cohortRowKey(record: ModelSignalCohort) {
    return record.cohortKey || `${record.modelProvider}:${record.model}:${record.projectPath}`
  }

  function matrixRowKey(record: ModelSignalMatrixRow) {
    return record.sourceKey || (record.sourceId !== undefined ? `source:${record.sourceId}` : `${record.agentKind}:${record.agentName}`)
  }

  function matrixCellKey(cell: ModelSignalMatrixCell) {
    return `${cell.modelProvider}:${cell.model}`
  }

  function matrixCellTitle(cell: ModelSignalMatrixCell) {
    return [
      `${cell.modelProvider || t('fallback.unknown')} / ${cell.model || t('fallback.unknown')}`,
      `${t('column.latency')}: ${formatLatency(cell.current?.modelLatencyMsPer1kOutputTokens)} (${t('metric.baseline')} ${formatLatency(cell.baseline?.modelLatencyMsPer1kOutputTokens)})`,
      `${t('column.throughput')}: ${formatRate(cell.current?.modelThroughputTokensPerSecond, 1)} tok/s (${t('metric.baseline')} ${formatRate(cell.baseline?.modelThroughputTokensPerSecond, 1)})`,
      `${t('column.confidence')}: ${formatConfidence(cell.confidence)}`
    ].join('\n')
  }

  function dailyRowKey(record: ModelSignalsDailyMetric) {
    return record.date
  }

  function projectRowKey(record: ProjectMetricRow) {
    return record.projectPath || `${record.modelCount}:${record.sourceCount}:${record.totalTokens}`
  }

  function normalizeAnomaly(row: ModelSignalAnomalySession): NormalizedAnomalySession {
    return {
      id: row.sessionId,
      sessionKey: row.sessionKey,
      codexSessionId: row.codexSessionId,
      startedAt: row.startedAt,
      projectPath: row.projectPath,
      rawSourcePath: row.rawSourcePath,
      agentKind: row.agentKind,
      agentName: row.agentName,
      sourceId: row.sourceId,
      sourceKey: row.sourceKey,
      sourceLabel: row.sourceLabel,
      sourceRootPath: row.sourceRootPath,
      sourceSessionsPath: row.sourceSessionsPath,
      model: row.model,
      totalTokens: row.totalTokens,
      outputExpansionRate: row.outputExpansionRate,
      reasoningOverheadRate: row.reasoningOverheadRate,
      cacheMissRate: row.cacheMissRate,
      modelThroughputTokensPerSecond: row.modelThroughputTokensPerSecond,
      failedToolCalls: row.failedToolCalls,
      modelDurationMs: row.modelDurationMs,
      score: row.score,
      reasons: row.reasons
    }
  }

  function sessionInfo(record: NormalizedAnomalySession) {
    return sessionDisplay({
      id: record.id,
      sessionKey: record.sessionKey || '',
      codexSessionId: record.codexSessionId
    })
  }

  function anomalyRowKey(record: NormalizedAnomalySession) {
    return record.id || record.sessionKey || record.codexSessionId || `${record.model}:${record.startedAt}`
  }

  return {
    anomalyRowClass,
    anomalyRowKey,
    cohortRowKey,
    confidenceReason,
    dailyRowKey,
    displayPair,
    displayPercent,
    displayRate,
    displayText,
    driftRowClass,
    failurePressure,
    fallbackHealthSummary,
    formatConfidence,
    formatLatency,
    formatOptionalCost,
    formatPercent,
    formatPressure,
    formatRate,
    formatThroughput,
    formatWindow,
    matrixCellKey,
    matrixCellTitle,
    matrixRowKey,
    metricClass,
    normalizeAnomaly,
    p10Throughput,
    p90Latency,
    projectHealthTitle,
    projectInfo,
    projectMixInfo,
    projectRowKey,
    reasonCount,
    reasonSeverity,
    reasonText,
    sessionInfo,
    severityClass,
    severityLabel,
    severityMetricTone,
    severityRank,
    severityTagColor,
    sourceInfo,
    unpricedNote
  }
}
