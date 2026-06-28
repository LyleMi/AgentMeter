import { computed, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import type { UsageScopeFilters } from '../api'

export interface UsageScopeForm {
  agent?: string
  model?: string
  from: string
  to: string
}

const scopeKeys = ['agent', 'model', 'from', 'to'] as const

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
    from: filters.from?.trim() || '',
    to: filters.to?.trim() || ''
  }
}

export function readUsageScopeQuery(query: LocationQuery): UsageScopeForm {
  return normalizeUsageScope({
    agent: cleanQueryValue(query.agent),
    model: cleanQueryValue(query.model),
    from: cleanQueryValue(query.from) || '',
    to: cleanQueryValue(query.to) || ''
  })
}

export function usageScopeToApiFilters(filters: UsageScopeForm): UsageScopeFilters {
  const normalized = normalizeUsageScope(filters)
  return {
    agent: normalized.agent,
    model: normalized.model,
    from: normalized.from || undefined,
    to: normalized.to || undefined
  }
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
    Boolean(filters.value.agent || filters.value.model || filters.value.from || filters.value.to)
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
    () => [route.query.agent, route.query.model, route.query.from, route.query.to],
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
