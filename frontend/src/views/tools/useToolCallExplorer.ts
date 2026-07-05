import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import { api, type AgentUsage, type ToolCall, type ToolCallFilters, type ToolCallRiskSummary, type ToolStat } from '../../api'
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
export type ToolCallSort = typeof DEFAULT_SORT | 'duration_desc' | 'duration_asc' | 'risk_desc' | 'risk_asc'
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
  const toolCallRisks = ref<ToolCallRiskSummary[]>([])
  const commandOptions = ref<ShellCommandStat[]>([])
  const toolFilter = ref<string | undefined>(stringRouteQueryValue(route.query.tool))
  const commandFilter = ref<string | undefined>(mode === 'shell' ? stringRouteQueryValue(route.query.command) : undefined)
  const riskOnlyFilter = ref(mode === 'shell' && stringRouteQueryValue(route.query.risk) === '1')
  const agentFilter = ref<string | undefined>(stringRouteQueryValue(route.query.agent))
  const fromFilter = ref(routeDateTimeInputValue(route.query, 'from'))
  const toFilter = ref(routeDateTimeInputValue(route.query, 'to'))
  const sortFilter = ref<ToolCallSort>(routeSortQuery(route.query))
  const selectedToolCall = ref<ToolCall | null>(null)
  const routePath = mode === 'shell' ? '/tools/shell' : '/tools/calls'
  const availableTools = computed(() => (mode === 'shell' ? tools.value.filter((item) => isShellToolName(item.toolName)) : tools.value))
  const riskByToolCallId = computed(() => riskMapFrom(toolCallRisks.value))
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
    toolCallRisks.value = []
    return (await api.listToolCalls(currentToolCallFilters())) || []
  }

  async function fetchShellToolCalls() {
    let calls = ((await api.listToolCalls(currentToolCallFilters())) || []).filter((call) => isShellToolName(call.toolName))
    toolCallRisks.value = calls
      .filter((call) => call.riskScore || call.riskSeverity || call.riskCount || call.riskRuleIds?.length)
      .map((call) => ({
        toolCallId: call.id,
        severity: call.riskSeverity || '',
        riskScore: call.riskScore || 1,
        riskCount: call.riskCount || 0,
        ruleIds: call.riskRuleIds || []
      }))
    commandOptions.value = shellCommandStats(calls)
    calls = filteredByCommand(calls, commandFilter.value)
    return calls.slice(0, TOOL_CALL_LIMIT)
  }

  function currentToolCallFilters(): ToolCallFilters {
    return {
      tool: toolFilter.value,
      agent: agentFilter.value,
      from: dateTimeInputToQueryIso(fromFilter.value),
      to: dateTimeInputToQueryIso(toFilter.value, 'end'),
      sort: sortFilter.value === DEFAULT_SORT ? undefined : sortFilter.value,
      shell: mode === 'shell',
      riskOnly: mode === 'shell' && riskOnlyFilter.value,
      includeRisk: mode === 'shell',
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
    setRouteQueryValue(query, 'risk', mode === 'shell' && riskOnlyFilter.value ? '1' : undefined)
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
    riskOnlyFilter.value = mode === 'shell' && stringRouteQueryValue(route.query.risk) === '1'
    agentFilter.value = stringRouteQueryValue(route.query.agent)
    fromFilter.value = routeDateTimeInputValue(route.query, 'from')
    toFilter.value = routeDateTimeInputValue(route.query, 'to')
    sortFilter.value = routeSortQuery(route.query)
  }

  function resetFilters() {
    toolFilter.value = undefined
    commandFilter.value = undefined
    riskOnlyFilter.value = false
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
    () => [route.query.tool, route.query.command, route.query.risk, route.query.agent, route.query.from, route.query.to, route.query.sort],
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
    toolCallRisks,
    riskByToolCallId,
    commandOptions,
    toolFilter,
    commandFilter,
    riskOnlyFilter,
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

function riskMapFrom(risks: ToolCallRiskSummary[]) {
  return new Map(risks.map((risk) => [risk.toolCallId, risk]))
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

function routeSortQuery(query: LocationQuery): ToolCallSort {
  const value = stringRouteQueryValue(query.sort)
  if (value === 'duration_desc' || value === 'duration_asc' || value === 'risk_desc' || value === 'risk_asc') return value
  return DEFAULT_SORT
}
