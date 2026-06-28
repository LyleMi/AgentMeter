import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, type AgentUsage, type ToolCall, type ToolCallFilters, type ToolStat } from '../../api'
import { isShellToolName } from './shellTool'

export const DEFAULT_SORT = 'recent'
export const TOOL_CALL_LIMIT = 500
export type ToolCallExplorerMode = 'all' | 'shell'
export type ToolCallSort = typeof DEFAULT_SORT | 'duration_desc' | 'duration_asc'

export function useToolCallExplorer(mode: ToolCallExplorerMode) {
  const route = useRoute()
  const router = useRouter()
  const loading = ref(true)
  const callLoading = ref(true)
  const toolLoading = ref(true)
  const tools = ref<ToolStat[]>([])
  const agents = ref<AgentUsage[]>([])
  const toolCalls = ref<ToolCall[]>([])
  const toolFilter = ref<string | undefined>(routeStringQuery(route, 'tool'))
  const agentFilter = ref<string | undefined>(routeStringQuery(route, 'agent'))
  const fromFilter = ref(routeDateTimeQuery(route, 'from'))
  const toFilter = ref(routeDateTimeQuery(route, 'to'))
  const sortFilter = ref<ToolCallSort>(routeSortQuery(route))
  const selectedToolCall = ref<ToolCall | null>(null)
  const routePath = mode === 'shell' ? '/tools/shell' : '/tools/calls'
  const availableTools = computed(() => (mode === 'shell' ? tools.value.filter((item) => isShellToolName(item.toolName)) : tools.value))
  let applyingRouteUpdate = false

  async function load() {
    loading.value = true
    callLoading.value = true
    try {
      const overviewRequest = api.getOverview()
      const clearedTool = await loadToolOptions(true)
      const overview = await overviewRequest
      if (clearedTool) await replaceRouteQuery()
      agents.value = overview?.agentUsage || []
      toolCalls.value = await fetchToolCalls()
    } finally {
      loading.value = false
      callLoading.value = false
    }
  }

  async function loadToolCalls() {
    callLoading.value = true
    try {
      toolCalls.value = await fetchToolCalls()
    } finally {
      callLoading.value = false
    }
  }

  async function fetchToolCalls() {
    return mode === 'shell' ? fetchShellToolCalls() : (await api.listToolCalls(currentToolCallFilters())) || []
  }

  async function fetchShellToolCalls() {
    const selectedTools = toolFilter.value ? [toolFilter.value] : availableTools.value.map((item) => item.toolName).filter(Boolean)
    if (!selectedTools.length) return []

    const callGroups = await Promise.all(
      selectedTools.map((tool) =>
        api.listToolCalls({
          ...currentToolCallFilters(),
          tool,
          limit: TOOL_CALL_LIMIT
        })
      )
    )
    return sortedCalls(uniqueCalls(callGroups.flat()).filter((call) => isShellToolName(call.toolName)), sortFilter.value).slice(0, TOOL_CALL_LIMIT)
  }

  function currentToolCallFilters(): ToolCallFilters {
    return {
      tool: mode === 'all' ? toolFilter.value : undefined,
      agent: agentFilter.value,
      from: toQueryDateTime(fromFilter.value),
      to: toQueryDateTime(toFilter.value, 'end'),
      sort: sortFilter.value === DEFAULT_SORT ? undefined : sortFilter.value,
      limit: TOOL_CALL_LIMIT
    }
  }

  function clearMissingToolFilter() {
    if (!toolFilter.value) return false
    if (availableTools.value.some((item) => item.toolName === toolFilter.value)) return false
    toolFilter.value = undefined
    return true
  }

  async function loadToolOptions(clearInvalidTool = false) {
    toolLoading.value = true
    try {
      tools.value = (await api.getTools({ agent: agentFilter.value })) || []
      return clearInvalidTool ? clearMissingToolFilter() : false
    } finally {
      toolLoading.value = false
    }
  }

  function currentRouteQuery() {
    const query: Record<string, string> = {}
    for (const [key, value] of Object.entries(route.query)) {
      if (typeof value === 'string') query[key] = value
    }
    setQueryValue(query, 'tool', toolFilter.value)
    setQueryValue(query, 'agent', agentFilter.value)
    setQueryValue(query, 'from', fromFilter.value || undefined)
    setQueryValue(query, 'to', toFilter.value || undefined)
    setQueryValue(query, 'sort', sortFilter.value === DEFAULT_SORT ? undefined : sortFilter.value)
    return query
  }

  async function replaceRouteQuery() {
    applyingRouteUpdate = true
    try {
      await router.replace({ path: routePath, query: currentRouteQuery() })
    } finally {
      applyingRouteUpdate = false
    }
  }

  async function updateFilters(changedFilter?: 'agent') {
    if (changedFilter === 'agent') await loadToolOptions(true)
    await replaceRouteQuery()
    loadToolCalls()
  }

  function syncFiltersFromRoute() {
    toolFilter.value = routeStringQuery(route, 'tool')
    agentFilter.value = routeStringQuery(route, 'agent')
    fromFilter.value = routeDateTimeQuery(route, 'from')
    toFilter.value = routeDateTimeQuery(route, 'to')
    sortFilter.value = routeSortQuery(route)
  }

  function resetFilters() {
    toolFilter.value = undefined
    agentFilter.value = undefined
    fromFilter.value = ''
    toFilter.value = ''
    sortFilter.value = DEFAULT_SORT
    updateFilters('agent')
  }

  function openToolCall(call: ToolCall) {
    selectedToolCall.value = call
  }

  function closeToolCall() {
    selectedToolCall.value = null
  }

  function openSession(id: number) {
    router.push(`/sessions/${id}`)
  }

  watch(
    () => [route.query.tool, route.query.agent, route.query.from, route.query.to, route.query.sort],
    async () => {
      if (applyingRouteUpdate) return
      syncFiltersFromRoute()
      const clearedTool = await loadToolOptions(true)
      if (clearedTool) await replaceRouteQuery()
      loadToolCalls()
    }
  )

  onMounted(load)

  return {
    loading,
    callLoading,
    toolLoading,
    tools,
    availableTools,
    agents,
    toolCalls,
    toolFilter,
    agentFilter,
    fromFilter,
    toFilter,
    sortFilter,
    selectedToolCall,
    load,
    updateFilters,
    resetFilters,
    openToolCall,
    closeToolCall,
    openSession
  }
}

function uniqueCalls(calls: ToolCall[]) {
  const values = new Map<number, ToolCall>()
  for (const call of calls) values.set(call.id, call)
  return [...values.values()]
}

function sortedCalls(calls: ToolCall[], sort: ToolCallSort) {
  return [...calls].sort((left, right) => {
    if (sort === 'duration_desc') return (right.durationMs || 0) - (left.durationMs || 0)
    if (sort === 'duration_asc') return (left.durationMs || 0) - (right.durationMs || 0)
    return timestampMs(right.startedAt) - timestampMs(left.startedAt)
  })
}

function timestampMs(value?: string) {
  if (!value) return 0
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? 0 : parsed
}

function routeStringQuery(route: ReturnType<typeof useRoute>, key: string) {
  const value = route.query[key]
  return typeof value === 'string' && value ? value : undefined
}

function routeDateTimeQuery(route: ReturnType<typeof useRoute>, key: string) {
  const value = routeStringQuery(route, key)
  if (!value) return ''
  return value.endsWith('Z') ? toLocalDateTimeInputValue(value) : value
}

function routeSortQuery(route: ReturnType<typeof useRoute>): ToolCallSort {
  const value = routeStringQuery(route, 'sort')
  if (value === 'duration_desc' || value === 'duration_asc') return value
  return DEFAULT_SORT
}

function toLocalDateTimeInputValue(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function toQueryDateTime(value: string, boundary: 'start' | 'end' = 'start') {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  if (boundary === 'end' && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) {
    date.setSeconds(59, 999)
  }
  return date.toISOString()
}

function setQueryValue(query: Record<string, string>, key: string, value?: string) {
  if (value) query[key] = value
  else delete query[key]
}
