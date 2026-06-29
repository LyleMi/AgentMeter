import type { AuditFinding, Session } from '../api/types'
import { api } from '../api/client'
import { matchesSourceFilter, type SourceIdentityLike } from '../presentation/sourceIdentity'
import { cleanFirstRouteQuery, firstTrimmedRouteQueryValue, routePathWithQuery } from './routeQuery'

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

export const cleanQueryValue = firstTrimmedRouteQueryValue
export const cleanRouteQuery = cleanFirstRouteQuery

export function auditPath(path: string, query: Record<string, unknown>): string {
  return routePathWithQuery(path, cleanRouteQuery(query))
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

export function matchesAgent(record: SourceIdentityLike, agent?: string) {
  return matchesSourceFilter(record, agent)
}
