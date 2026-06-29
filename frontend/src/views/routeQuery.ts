import type { LocationQuery } from 'vue-router'

export type RouteQueryRecord = Record<string, string>
export type RouteQueryDateTimeBoundary = 'start' | 'end'

export function stringRouteQueryValue(value: unknown) {
  return typeof value === 'string' && value ? value : undefined
}

export function trimmedRouteQueryValue(value: unknown) {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

export function firstTrimmedRouteQueryValue(value: unknown): string {
  const nextValue = Array.isArray(value) ? value[0] : value
  return typeof nextValue === 'string' ? nextValue.trim() : ''
}

export function optionalFirstTrimmedRouteQueryValue(value: unknown) {
  return firstTrimmedRouteQueryValue(value) || undefined
}

export function copyStringRouteQuery(sourceQuery: LocationQuery | Record<string, unknown>): RouteQueryRecord {
  const query: RouteQueryRecord = {}
  for (const [key, value] of Object.entries(sourceQuery)) {
    if (typeof value === 'string') query[key] = value
  }
  return query
}

export function cleanFirstRouteQuery(values: LocationQuery | Record<string, unknown>): RouteQueryRecord {
  const query: RouteQueryRecord = {}
  Object.entries(values).forEach(([key, value]) => {
    const cleanValue = firstTrimmedRouteQueryValue(value)
    if (cleanValue) query[key] = cleanValue
  })
  return query
}

export function setRouteQueryValue(query: RouteQueryRecord, key: string, value?: string) {
  if (value) query[key] = value
  else delete query[key]
}

export function setTrimmedRouteQueryValue(query: RouteQueryRecord, key: string, value?: string) {
  const next = value?.trim()
  if (next) query[key] = next
  else delete query[key]
}

export function routePathWithQuery(path: string, query: RouteQueryRecord): string {
  const encoded = new URLSearchParams(query).toString()
  return encoded ? `${path}?${encoded}` : path
}

export function routeDateTimeInputValue(query: LocationQuery, key: string) {
  const value = stringRouteQueryValue(query[key])
  if (!value) return ''
  return value.endsWith('Z') ? toLocalDateTimeInputValue(value) : value
}

export function toLocalDateTimeInputValue(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function dateTimeInputToQueryIso(value: string, boundary: RouteQueryDateTimeBoundary = 'start') {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  if (boundary === 'end' && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) {
    date.setSeconds(59, 999)
  }
  return date.toISOString()
}
