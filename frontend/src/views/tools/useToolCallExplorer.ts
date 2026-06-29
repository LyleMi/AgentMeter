import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import { api, type AgentUsage, type ToolCall, type ToolCallFilters, type ToolStat } from '../../api'
import {
  copyStringRouteQuery,
  dateTimeInputToQueryIso,
  routeDateTimeInputValue,
  setRouteQueryValue,
  stringRouteQueryValue
} from '../routeQuery'
import { invokedCommand, isShellToolName } from './shellTool'

export const DEFAULT_SORT = 'recent'
export const TOOL_CALL_LIMIT = 500
export type ToolCallExplorerMode = 'all' | 'shell'
export type ToolCallSort = typeof DEFAULT_SORT | 'duration_desc' | 'duration_asc'
export interface ShellCommandStat {
  command: string
  calls: number
}

export function useToolCallExplorer(mode: ToolCallExplorerMode) {
  const route = useRoute()
  const router = useRouter()
  const loading = ref(true)
  const callLoading = ref(true)
  const toolLoading = ref(true)
  const tools = ref<ToolStat[]>([])
  const agents = ref<AgentUsage[]>([])
  const toolCalls = ref<ToolCall[]>([])
  const commandOptions = ref<ShellCommandStat[]>([])
  const toolFilter = ref<string | undefined>(stringRouteQueryValue(route.query.tool))
  const commandFilter = ref<string | undefined>(mode === 'shell' ? stringRouteQueryValue(route.query.command) : undefined)
  const agentFilter = ref<string | undefined>(stringRouteQueryValue(route.query.agent))
  const fromFilter = ref(routeDateTimeInputValue(route.query, 'from'))
  const toFilter = ref(routeDateTimeInputValue(route.query, 'to'))
  const sortFilter = ref<ToolCallSort>(routeSortQuery(route.query))
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
    if (mode === 'shell') return fetchShellToolCalls()
    commandOptions.value = []
    return (await api.listToolCalls(currentToolCallFilters())) || []
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
    const calls = sortedCalls(uniqueCalls(callGroups.flat()).filter((call) => isShellToolName(call.toolName)), sortFilter.value)
    commandOptions.value = shellCommandStats(calls)
    return filteredByCommand(calls, commandFilter.value).slice(0, TOOL_CALL_LIMIT)
  }

  function currentToolCallFilters(): ToolCallFilters {
    return {
      tool: mode === 'all' ? toolFilter.value : undefined,
      agent: agentFilter.value,
      from: dateTimeInputToQueryIso(fromFilter.value),
      to: dateTimeInputToQueryIso(toFilter.value, 'end'),
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
    const query = copyStringRouteQuery(route.query)
    setRouteQueryValue(query, 'tool', toolFilter.value)
    setRouteQueryValue(query, 'command', mode === 'shell' ? commandFilter.value : undefined)
    setRouteQueryValue(query, 'agent', agentFilter.value)
    setRouteQueryValue(query, 'from', fromFilter.value || undefined)
    setRouteQueryValue(query, 'to', toFilter.value || undefined)
    setRouteQueryValue(query, 'sort', sortFilter.value === DEFAULT_SORT ? undefined : sortFilter.value)
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
    toolFilter.value = stringRouteQueryValue(route.query.tool)
    commandFilter.value = mode === 'shell' ? stringRouteQueryValue(route.query.command) : undefined
    agentFilter.value = stringRouteQueryValue(route.query.agent)
    fromFilter.value = routeDateTimeInputValue(route.query, 'from')
    toFilter.value = routeDateTimeInputValue(route.query, 'to')
    sortFilter.value = routeSortQuery(route.query)
  }

  function resetFilters() {
    toolFilter.value = undefined
    commandFilter.value = undefined
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
    () => [route.query.tool, route.query.command, route.query.agent, route.query.from, route.query.to, route.query.sort],
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
    commandOptions,
    toolFilter,
    commandFilter,
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

function filteredByCommand(calls: ToolCall[], command?: string) {
  if (!command) return calls
  return calls.filter((call) => invokedCommand(call) === command)
}

function shellCommandStats(calls: ToolCall[]): ShellCommandStat[] {
  const counts = new Map<string, number>()
  for (const call of calls) {
    const command = invokedCommand(call)
    if (!command) continue
    counts.set(command, (counts.get(command) || 0) + 1)
  }
  return [...counts.entries()]
    .map(([command, calls]) => ({ command, calls }))
    .sort((left, right) => right.calls - left.calls || left.command.localeCompare(right.command))
}

function timestampMs(value?: string) {
  if (!value) return 0
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? 0 : parsed
}

function routeSortQuery(query: LocationQuery): ToolCallSort {
  const value = stringRouteQueryValue(query.sort)
  if (value === 'duration_desc' || value === 'duration_asc') return value
  return DEFAULT_SORT
}
