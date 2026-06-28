import { shortPath } from './formatters'

export interface SourceIdentityLike {
  sourceId?: number
  sourceKey?: string
  sourceLabel?: string
  sourceRootPath?: string
  sourceSessionsPath?: string
  agentKind?: string
  agentName?: string
  rawSourcePath?: string
  projectPath?: string
}

export interface SourceDisplay {
  key: string
  filterValue: string
  label: string
  family: string
  path: string
  shortPath: string
  secondary: string
  title: string
}

export interface SourceFilterOption {
  value: string
  label: string
  title: string
}

function clean(value?: string | null): string {
  return (value || '').trim()
}

function sourceKeyFromId(value?: number): string {
  return typeof value === 'number' && Number.isFinite(value) ? `source:${value}` : ''
}

export function sourceInstanceKey(record: SourceIdentityLike, fallback = 'unknown'): string {
  return clean(record.sourceKey) || sourceKeyFromId(record.sourceId) || `${clean(record.agentKind) || fallback}:${clean(record.agentName)}`
}

export function sourceFilterValue(record: SourceIdentityLike): string {
  return clean(record.sourceKey) || sourceKeyFromId(record.sourceId) || clean(record.agentKind) || clean(record.agentName)
}

export function sourceFamily(record: SourceIdentityLike): string {
  return clean(record.agentKind)
}

export function sourceName(record: SourceIdentityLike, fallback = 'unknown'): string {
  return clean(record.sourceLabel) || clean(record.agentName) || clean(record.agentKind) || fallback
}

export function sourcePath(record: SourceIdentityLike): string {
  return clean(record.sourceRootPath) || clean(record.sourceSessionsPath) || clean(record.rawSourcePath) || clean(record.projectPath)
}

export function sourceDisplay(record: SourceIdentityLike, fallback = 'unknown'): SourceDisplay {
  const label = sourceName(record, fallback)
  const family = sourceFamily(record)
  const path = sourcePath(record)
  const pathLabel = path ? shortPath(path) : ''
  const secondary = [family, pathLabel].filter(Boolean).join(' · ')
  const title = [label, family, clean(record.sourceRootPath), clean(record.sourceSessionsPath), clean(record.rawSourcePath), clean(record.projectPath)]
    .filter(Boolean)
    .join('\n')

  return {
    key: sourceInstanceKey(record, fallback),
    filterValue: sourceFilterValue(record),
    label,
    family,
    path,
    shortPath: pathLabel,
    secondary,
    title
  }
}

export function sourceFilterOptions(records: SourceIdentityLike[], fallback = 'unknown'): SourceFilterOption[] {
  const values = new Map<string, SourceFilterOption>()
  for (const record of records) {
    const display = sourceDisplay(record, fallback)
    if (!display.filterValue) continue
    if (!values.has(display.filterValue)) {
      values.set(display.filterValue, {
        value: display.filterValue,
        label: display.secondary ? `${display.label} · ${display.secondary}` : display.label,
        title: display.title
      })
    }
  }
  return [...values.values()].sort((left, right) => left.label.localeCompare(right.label))
}

export function matchesSourceFilter(record: SourceIdentityLike, filter?: string): boolean {
  const normalizedFilter = clean(filter).toLowerCase()
  if (!normalizedFilter) return true
  return [
    record.sourceKey,
    sourceKeyFromId(record.sourceId),
    record.sourceLabel,
    record.agentKind,
    record.agentName
  ].some((value) => clean(value).toLowerCase() === normalizedFilter)
}
