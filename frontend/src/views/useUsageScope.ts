import { computed, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import type { UsageScopeFilters } from '../api'

export interface UsageScopeForm {
  agent?: string
  model?: string
  range?: string
  from: string
  to: string
}

const scopeKeys = ['agent', 'model', 'range', 'from', 'to'] as const
const dateOnlyPattern = /^\d{4}-\d{2}-\d{2}$/
const quickRangeDays: Record<string, number> = {
  day: 1,
  week: 7,
  month: 30
}

function cleanQueryValue(value: unknown) {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

function setQueryValue(query: Record<string, string>, key: string, value?: string) {
  const next = value?.trim()
  if (next) query[key] = next
  else delete query[key]
}

export function normalizeUsageScope(filters: Partial<UsageScopeForm>): UsageScopeForm {
  return {
    agent: filters.agent?.trim() || undefined,
    model: filters.model?.trim() || undefined,
    range: normalizeQuickRange(filters.range),
    from: filters.from?.trim() || '',
    to: filters.to?.trim() || ''
  }
}

export function readUsageScopeQuery(query: LocationQuery): UsageScopeForm {
  return normalizeUsageScope({
    agent: cleanQueryValue(query.agent),
    model: cleanQueryValue(query.model),
    range: cleanQueryValue(query.range),
    from: cleanQueryValue(query.from) || '',
    to: cleanQueryValue(query.to) || ''
  })
}

export function usageScopeToApiFilters(filters: UsageScopeForm): UsageScopeFilters {
  const normalized = normalizeUsageScope(filters)
  if (normalized.range) {
    return {
      agent: normalized.agent,
      model: normalized.model,
      from: quickRangeFrom(normalized.range),
      to: undefined
    }
  }
  return {
    agent: normalized.agent,
    model: normalized.model,
    from: toApiDateBoundary(normalized.from, 'start'),
    to: toApiDateBoundary(normalized.to, 'end')
  }
}

function normalizeQuickRange(value?: string) {
  const normalized = value?.trim()
  return normalized && normalized in quickRangeDays ? normalized : undefined
}

function quickRangeFrom(value: string) {
  const days = quickRangeDays[value]
  if (!days) return undefined
  return new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString()
}

function toApiDateBoundary(value: string, boundary: 'start' | 'end') {
  const normalized = value.trim()
  if (!normalized) return undefined
  if (dateOnlyPattern.test(normalized)) {
    const date = new Date(`${normalized}T00:00:00`)
    if (Number.isNaN(date.getTime())) return undefined
    if (boundary === 'end') date.setHours(23, 59, 59, 999)
    return date.toISOString()
  }
  const date = new Date(normalized)
  if (Number.isNaN(date.getTime())) return normalized
  return date.toISOString()
}

export function applyUsageScopeToQuery(
  sourceQuery: LocationQuery,
  filters: UsageScopeForm,
  extra: Record<string, string | undefined> = {}
) {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(sourceQuery)) {
    if (typeof value === 'string') query[key] = value
  }

  const normalized = normalizeUsageScope(filters)
  for (const key of scopeKeys) setQueryValue(query, key, normalized[key])
  for (const [key, value] of Object.entries(extra)) setQueryValue(query, key, value)
  return query
}

export function useUsageScopeRoute(onRouteScopeChange?: () => void | Promise<void>) {
  const route = useRoute()
  const router = useRouter()
  const filters = ref(readUsageScopeQuery(route.query))
  let applyingRouteUpdate = false

  const apiFilters = computed(() => usageScopeToApiFilters(filters.value))
  const hasActiveFilters = computed(() =>
    Boolean(filters.value.agent || filters.value.model || filters.value.range || filters.value.from || filters.value.to)
  )

  async function updateFilters(nextFilters: UsageScopeForm) {
    filters.value = normalizeUsageScope(nextFilters)
    applyingRouteUpdate = true
    try {
      await router.replace({
        path: route.path,
        query: applyUsageScopeToQuery(route.query, filters.value)
      })
    } finally {
      applyingRouteUpdate = false
    }
  }

  async function clearFilters() {
    await updateFilters({ from: '', to: '' })
  }

  watch(
    () => [route.query.agent, route.query.model, route.query.range, route.query.from, route.query.to],
    () => {
      if (applyingRouteUpdate) return
      filters.value = readUsageScopeQuery(route.query)
      void onRouteScopeChange?.()
    }
  )

  return {
    filters,
    apiFilters,
    hasActiveFilters,
    updateFilters,
    clearFilters
  }
}
