import type { Session, ToolCallFilters, UsageScopeFilters } from '../types'

export function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

export function matchesAgent(record: { sourceId?: number; sourceKey?: string; sourceLabel?: string; agentKind?: string; agentName?: string }, agent?: string): boolean {
  const normalized = (agent || '').trim().toLowerCase()
  if (!normalized) return true
  return [record.sourceKey, record.sourceId !== undefined ? `source:${record.sourceId}` : '', record.sourceLabel, record.agentKind, record.agentName]
    .some((value) => (value || '').toLowerCase() === normalized)
}

export function matchesDateRange(value: string, filters: UsageScopeFilters | ToolCallFilters): boolean {
  const timestamp = Date.parse(value)
  if (filters.from && timestamp < Date.parse(filters.from)) return false
  if (filters.to && timestamp > Date.parse(filters.to)) return false
  return true
}

export function matchesProject(record: { projectPath?: string; rawSourcePath?: string }, project?: string): boolean {
  const normalized = (project || '').trim()
  if (!normalized) return true
  const projectKey = projectPathKey(normalized)
  return [record.projectPath, record.rawSourcePath]
    .some((value) => {
      const candidate = (value || '').trim()
      return candidate === normalized || projectPathKey(candidate) === projectKey
    })
}

export function paginate<T>(items: T[], limit?: number, offset?: number): T[] {
  const start = Math.max(0, offset || 0)
  const end = limit ? start + Math.max(0, limit) : undefined
  return items.slice(start, end)
}

export function sum(items: Session[], selector: (session: Session) => number): number {
  return items.reduce((total, session) => total + selector(session), 0)
}

export function groupedBy<T>(items: T[], keyFor: (item: T) => string): Map<string, T[]> {
  const groups = new Map<string, T[]>()
  items.forEach((item) => {
    const key = keyFor(item)
    groups.set(key, [...(groups.get(key) || []), item])
  })
  return groups
}

export function projectPathKey(value: string): string {
  const normalized = value.trim().replace(/[\\/]\.$/, '').replace(/[\\/]+$/, '')
  return normalized ? normalized.toLowerCase() : 'unknown'
}
