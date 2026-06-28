import type { AuditFinding, Session } from '../api/types'
import { api } from '../api/client'

export interface AuditFindingQuery {
  agent?: string
  category?: string
  severity?: string
  shell?: string
  search?: string
  limit?: number
  offset?: number
}

export interface AuditSummaryQuery {
  agent?: string
}

export interface AuditFindingDetail {
  finding: AuditFinding
  session?: Session
  relatedFindings: AuditFinding[]
}

export function cleanQueryValue(value: unknown): string {
  const nextValue = Array.isArray(value) ? value[0] : value
  return typeof nextValue === 'string' ? nextValue.trim() : ''
}

export function cleanRouteQuery(values: Record<string, unknown>): Record<string, string> {
  const query: Record<string, string> = {}
  Object.entries(values).forEach(([key, value]) => {
    const cleanValue = cleanQueryValue(value)
    if (cleanValue) query[key] = cleanValue
  })
  return query
}

export function buildQueryString(values: Record<string, string | number | undefined | null>): string {
  const params = new URLSearchParams()
  Object.entries(values).forEach(([key, value]) => {
    if (value !== undefined && value !== null && String(value).trim()) params.set(key, String(value))
  })
  const query = params.toString()
  return query ? `?${query}` : ''
}

export function auditPath(path: string, query: Record<string, unknown>): string {
  return `${path}${buildQueryString(cleanRouteQuery(query))}`
}

export function getAuditSummary(filters: AuditSummaryQuery = {}) {
  return api.getAuditSummary({ agent: filters.agent })
}

export function listAuditFindings(filters: AuditFindingQuery = {}) {
  return api.listAuditFindings({
    agent: filters.agent,
    category: filters.category,
    severity: filters.severity,
    shell: filters.shell,
    search: filters.search,
    limit: filters.limit,
    offset: filters.offset
  })
}

export async function getAuditFinding(id: number, filters: AuditFindingQuery = {}): Promise<AuditFindingDetail> {
  try {
    const finding = await api.getAuditFinding(id)
    if (!matchesAgent(finding, filters.agent)) {
      throw new Error('Audit finding does not match the selected agent filter')
    }
    return { finding, relatedFindings: [] }
  } catch (detailError) {
    const findings = await listAuditFindings({ agent: filters.agent, limit: filters.limit || 1000, offset: 0 })
    const finding = findings.filter((item) => matchesAgent(item, filters.agent)).find((item) => item.id === id)
    if (finding) return { finding, relatedFindings: [] }
    throw detailError
  }
}

export function normalized(value?: string | null) {
  return (value || '').trim().toLowerCase()
}

export function titleCaseFallback(value?: string | null, fallback = 'unknown') {
  const text = (value || '').trim()
  if (!text) return fallback
  return text.replace(/[_-]+/g, ' ').replace(/\b\w/g, (match) => match.toUpperCase())
}

export function severityColor(value?: string | null) {
  const severity = normalized(value)
  if (severity === 'critical') return 'red'
  if (severity === 'high') return 'orange'
  if (severity === 'medium') return 'gold'
  if (severity === 'low') return 'blue'
  return 'default'
}

export function categoryColor(value?: string | null) {
  const category = normalized(value)
  if (category === 'command') return 'processing'
  if (category === 'privacy') return 'purple'
  if (category === 'egress') return 'cyan'
  if (category === 'file') return 'geekblue'
  return 'default'
}

export function matchesAgent(record: Pick<AuditFinding, 'agentKind' | 'agentName'>, agent?: string) {
  const filter = normalized(agent)
  if (!filter) return true
  return [record.agentKind, record.agentName].some((value) => normalized(value) === filter)
}
