import type { AuditFinding, AuditFindingFilters, AuditSummary, ToolCallRiskFilters, ToolCallRiskSummary } from '../types'
import { sessions, toolCalls } from './sessions'
import { matchesAgent, matchesDateRange } from './utils'

const auditFindings: AuditFinding[] = [
  makeFinding({ id: 501, sessionId: 101, toolCallId: 1001, category: 'command', severity: 'medium', ruleId: 'shell.powershell.concatenated-delete', title: 'Review destructive shell composition', description: 'Remove command was composed with string interpolation.', command: 'Remove-Item $target -Recurse', shellFamily: 'powershell' }),
  makeFinding({ id: 502, sessionId: 102, toolCallId: 1005, category: 'egress', severity: 'low', ruleId: 'network.fetch.failed', title: 'Network access attempted', description: 'A documentation lookup was attempted while offline demo policy was active.', command: 'Invoke-WebRequest https://example.invalid/pricing', shellFamily: 'powershell' }),
  makeFinding({ id: 503, sessionId: 104, toolCallId: 1008, category: 'privacy', severity: 'high', ruleId: 'privacy.telemetry.enabled', title: 'Telemetry setting needs review', description: 'Demo privacy config shows a setting that is not hardened.', command: 'telemetry.enabled = true', shellFamily: 'json' }),
  makeFinding({ id: 504, sessionId: 105, toolCallId: 1009, category: 'file', severity: 'medium', ruleId: 'file.output.screenshot', title: 'Generated artifact requires retention decision', description: 'A browser screenshot artifact was created during validation.', command: 'browser_screenshot tools chart', shellFamily: 'browser' })
]

type DemoFindingSpec = {
  id: number
  sessionId: number
  toolCallId: number
  category: string
  severity: string
  ruleId: string
  title: string
  description: string
  command: string
  shellFamily: string
}

function makeFinding(spec: DemoFindingSpec): AuditFinding {
  const session = sessions.find((item) => item.id === spec.sessionId)
  if (!session) throw new Error(`Missing demo session ${spec.sessionId}`)
  const tool = toolCalls.find((item) => item.id === spec.toolCallId)
  return {
    id: spec.id,
    sessionId: spec.sessionId,
    toolCallId: spec.toolCallId,
    sourceFileId: session.sourceId || 0,
    rawEventId: tool?.rawEventId || spec.id + 5000,
    sourceLine: tool?.rawEventLine || 1,
    timestamp: tool?.endedAt || session.endedAt,
    source: 'demo audit',
    eventType: tool?.rawEndEventType || 'tool_result',
    category: spec.category,
    severity: spec.severity,
    ruleId: spec.ruleId,
    title: spec.title,
    description: spec.description,
    evidence: tool?.rawEndEventSummary || spec.description,
    command: spec.command,
    shellFamily: spec.shellFamily,
    platform: 'windows',
    decision: 'review',
    createdAt: session.endedAt,
    sessionKey: session.sessionKey,
    codexSessionId: session.codexSessionId,
    projectPath: session.projectPath,
    agentKind: session.agentKind,
    agentName: session.agentName,
    rawSourcePath: session.rawSourcePath,
    sourceId: session.sourceId,
    sourceKey: session.sourceKey,
    sourceLabel: session.sourceLabel,
    sourceRootPath: session.sourceRootPath,
    sourceSessionsPath: session.sourceSessionsPath
  }
}

export function auditSummary(filters: Pick<AuditFindingFilters, 'agent'> = {}): AuditSummary {
  const findings = auditFindings.filter((finding) => matchesAgent(finding, filters.agent))
  return {
    totalFindings: findings.length,
    criticalFindings: findings.filter((finding) => finding.severity === 'critical').length,
    highFindings: findings.filter((finding) => finding.severity === 'high').length,
    mediumFindings: findings.filter((finding) => finding.severity === 'medium').length,
    lowFindings: findings.filter((finding) => finding.severity === 'low').length,
    commandFindings: findings.filter((finding) => finding.category === 'command').length,
    privacyFindings: findings.filter((finding) => finding.category === 'privacy').length,
    egressFindings: findings.filter((finding) => finding.category === 'egress').length,
    fileFindings: findings.filter((finding) => finding.category === 'file').length,
    sessionsWithFindings: new Set(findings.map((finding) => finding.sessionId)).size,
    recentFindings: findings.slice(0, 5)
  }
}

export function filteredFindings(filters: AuditFindingFilters = {}): AuditFinding[] {
  const search = (filters.search || '').trim().toLowerCase()
  return auditFindings
    .filter((finding) => matchesAgent(finding, filters.agent))
    .filter((finding) => !filters.category || finding.category === filters.category)
    .filter((finding) => !filters.severity || finding.severity === filters.severity)
    .filter((finding) => !filters.shell || finding.shellFamily === filters.shell)
    .filter((finding) => {
      if (!search) return true
      return [finding.title, finding.description, finding.evidence, finding.command, finding.ruleId, finding.projectPath]
        .some((value) => (value || '').toLowerCase().includes(search))
    })
    .sort((left, right) => Date.parse(right.timestamp) - Date.parse(left.timestamp))
}

export function filteredToolCallRisks(filters: ToolCallRiskFilters = {}): ToolCallRiskSummary[] {
  const callsById = new Map(toolCalls.map((call) => [call.id, call]))
  const grouped = new Map<number, AuditFinding[]>()
  for (const finding of auditFindings) {
    if (!finding.toolCallId) continue
    const call = callsById.get(finding.toolCallId)
    if (!call) continue
    if (!matchesAgent(finding, filters.agent)) continue
    if (!matchesDateRange(call.startedAt, filters)) continue
    grouped.set(finding.toolCallId, [...(grouped.get(finding.toolCallId) || []), finding])
  }
  return [...grouped.entries()]
    .map(([toolCallId, findings]) => ({
      toolCallId,
      severity: highestSeverity(findings.map((finding) => finding.severity)),
      riskScore: riskScoreFor(findings),
      riskCount: findings.length,
      ruleIds: [...new Set(findings.map((finding) => finding.ruleId).filter(Boolean))].sort()
    }))
    .sort((left, right) => Date.parse(callsById.get(right.toolCallId)?.startedAt || '') - Date.parse(callsById.get(left.toolCallId)?.startedAt || '') || right.toolCallId - left.toolCallId)
    .slice(0, filters.limit || 500)
}

export function auditFinding(id: number): AuditFinding {
  const finding = auditFindings.find((item) => item.id === id)
  if (!finding) throw new Error('Demo audit finding not found')
  return finding
}

function highestSeverity(values: string[]) {
  const rank: Record<string, number> = { low: 1, medium: 2, high: 3, critical: 4 }
  return values.reduce((best, value) => ((rank[value] || 0) > (rank[best] || 0) ? value : best), '')
}

function riskScoreFor(findings: AuditFinding[]) {
  if (!findings.length) return 1
  const base: Record<string, number> = { low: 20, medium: 45, high: 70, critical: 90 }
  const ruleCount = new Set(findings.map((finding) => finding.ruleId).filter(Boolean)).size
  return Math.min(100, (base[highestSeverity(findings.map((finding) => finding.severity))] || 0) + Math.max(0, ruleCount - 1) * 5)
}
