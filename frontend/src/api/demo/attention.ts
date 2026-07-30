import type { AttentionResponse, UsageScopeFilters } from '../types'
import { overview } from './usage'

function parsedDate(value: string | undefined, fallback: Date) {
  if (!value) return fallback
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? fallback : date
}

export function attention(filters: UsageScopeFilters = {}): AttentionResponse {
  const toDate = parsedDate(filters.to, new Date())
  const fromDate = parsedDate(filters.from, new Date(toDate.getTime() - 7 * 24 * 60 * 60 * 1000))
  const durationMs = Math.max(toDate.getTime() - fromDate.getTime(), 24 * 60 * 60 * 1000)
  const baselineFromDate = new Date(fromDate.getTime() - durationMs)
  const filteredCurrent = overview(filters)
  const current = filteredCurrent.totalSessions ? filteredCurrent : overview()
  const filteredBaseline = overview({
    ...filters,
    from: baselineFromDate.toISOString(),
    to: fromDate.toISOString()
  })
  const baseline = filteredBaseline.totalSessions ? filteredBaseline : undefined
  const to = toDate.toISOString()
  const from = fromDate.toISOString()
  const baselineFrom = baselineFromDate.toISOString()
  return {
    window: {
      current: { from, to },
      baseline: { from: baselineFrom, to: from }
    },
    snapshot: {
      sessions: metric(current.totalSessions, baseline?.totalSessions ?? syntheticBaseline(current.totalSessions)),
      tokens: metric(current.totalTokens, baseline?.totalTokens ?? syntheticBaseline(current.totalTokens)),
      costUsd: metric(current.estimatedCostUsd || 0, baseline?.estimatedCostUsd ?? syntheticBaseline(current.estimatedCostUsd || 0)),
      activeTime: metric(current.totalActiveDurationMs, baseline?.totalActiveDurationMs ?? syntheticBaseline(current.totalActiveDurationMs)),
      toolCalls: metric(current.totalToolCalls, baseline?.totalToolCalls ?? syntheticBaseline(current.totalToolCalls))
    },
    counts: { critical: 1, warning: 2, total: 3 },
    items: [
      {
        key: 'demo:audit:1',
        kind: 'audit_finding',
        severity: 'critical',
        confidence: 0.96,
        subject: 'Credential-shaped value passed to a shell command',
        reason: 'A high-confidence local audit rule matched command evidence.',
        occurredAt: current.recentSessions[0]?.startedAt,
        destination: { kind: 'audit_finding', auditFindingId: 1 }
      },
      {
        key: 'demo:model:drift',
        kind: 'model_cohort',
        severity: 'warning',
        confidence: 0.82,
        subject: 'Model cohort drift',
        reason: 'Tool failure pressure increased versus the preceding equal-length period.',
        destination: { kind: 'model_analysis' }
      },
      {
        key: 'demo:pricing',
        kind: 'unpriced_sessions',
        severity: 'warning',
        confidence: 1,
        subject: 'Unpriced sessions',
        reason: 'Some sessions use models without pricing.',
        metric: { label: 'sessions', value: Math.max(current.unpricedSessions, 1) },
        destination: { kind: 'settings', settingsPanel: 'pricing' }
      }
    ]
  }
}

function syntheticBaseline(current: number) {
  if (!current) return 0
  return Math.max(current * 0.82, current < 10 ? 1 : 0)
}

function metric(current: number, baseline: number) {
  return {
    current,
    baseline,
    changePct: baseline ? (current - baseline) / baseline : undefined
  }
}
