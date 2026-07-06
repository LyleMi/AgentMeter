import { computed, onMounted, ref, watch, type ComputedRef } from 'vue'
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
  const state = createToolCallExplorerState(route.query, mode)
  const availableTools = computed(() => (mode === 'shell' ? state.tools.value.filter((item) => isShellToolName(item.toolName)) : state.tools.value))
  const riskByToolCallId = computed(() => riskMapFrom(state.toolCallRisks.value))
  const ctx: ToolCallExplorerContext = {
    mode,
    route,
    router,
    routePath: mode === 'shell' ? '/tools/shell' : '/tools/calls',
    routeUpdate: { applying: false },
    state,
    availableTools
  }

  registerRouteSync(ctx)
  onMounted(() => loadExplorer(ctx))

  return {
    ...state,
    availableTools,
    riskByToolCallId,
    load: () => loadExplorer(ctx),
    updateFilters: (changedFilter?: 'agent') => updateFilters(ctx, changedFilter),
    resetFilters: () => resetFilters(ctx),
    openToolCall: (call: ToolCall) => openToolCall(ctx, call),
    closeToolCall: () => closeToolCall(ctx),
    openSession: (id: number) => openSession(ctx, id)
  }
}

function createToolCallExplorerState(query: LocationQuery, mode: ToolCallExplorerMode) {
  return {
    loading: ref(true),
    callLoading: ref(true),
    toolLoading: ref(true),
    tools: ref<ToolStat[]>([]),
    agents: ref<AgentUsage[]>([]),
    toolCalls: ref<ToolCall[]>([]),
    toolCallRisks: ref<ToolCallRiskSummary[]>([]),
    commandOptions: ref<ShellCommandStat[]>([]),
    toolFilter: ref<string | undefined>(stringRouteQueryValue(query.tool)),
    commandFilter: ref<string | undefined>(mode === 'shell' ? stringRouteQueryValue(query.command) : undefined),
    riskOnlyFilter: ref(mode === 'shell' && stringRouteQueryValue(query.risk) === '1'),
    agentFilter: ref<string | undefined>(stringRouteQueryValue(query.agent)),
    fromFilter: ref(routeDateTimeInputValue(query, 'from')),
    toFilter: ref(routeDateTimeInputValue(query, 'to')),
    sortFilter: ref<ToolCallSort>(routeSortQuery(query)),
    selectedToolCall: ref<ToolCall | null>(null)
  }
}

type ToolCallExplorerState = ReturnType<typeof createToolCallExplorerState>

interface ToolCallExplorerContext {
  mode: ToolCallExplorerMode
  route: ReturnType<typeof useRoute>
  router: ReturnType<typeof useRouter>
  routePath: string
  routeUpdate: { applying: boolean }
  state: ToolCallExplorerState
  availableTools: ComputedRef<ToolStat[]>
}

async function loadExplorer(ctx: ToolCallExplorerContext) {
  ctx.state.loading.value = true
  ctx.state.callLoading.value = true
  try {
    const overviewRequest = api.getOverview()
    const clearedTool = await loadToolOptions(ctx, true)
    const overview = await overviewRequest
    if (clearedTool) await replaceRouteQuery(ctx)
    ctx.state.agents.value = overview?.agentUsage || []
    ctx.state.toolCalls.value = await fetchToolCalls(ctx)
  } finally {
    ctx.state.loading.value = false
    ctx.state.callLoading.value = false
  }
}

async function loadToolCalls(ctx: ToolCallExplorerContext) {
  ctx.state.callLoading.value = true
  try {
    ctx.state.toolCalls.value = await fetchToolCalls(ctx)
  } finally {
    ctx.state.callLoading.value = false
  }
}

async function fetchToolCalls(ctx: ToolCallExplorerContext) {
  if (ctx.mode === 'shell') return fetchShellToolCalls(ctx)
  ctx.state.commandOptions.value = []
  ctx.state.toolCallRisks.value = []
  return (await api.listToolCalls(currentToolCallFilters(ctx))) || []
}

async function fetchShellToolCalls(ctx: ToolCallExplorerContext) {
  const calls = ((await api.listToolCalls(currentToolCallFilters(ctx))) || []).filter((call) => isShellToolName(call.toolName))
  ctx.state.toolCallRisks.value = riskSummariesFromCalls(calls)
  ctx.state.commandOptions.value = shellCommandStats(calls)
  return filteredByCommand(calls, ctx.state.commandFilter.value).slice(0, TOOL_CALL_LIMIT)
}

function riskSummariesFromCalls(calls: ToolCall[]) {
  return calls
    .filter(hasRiskFields)
    .map((call) => ({
      toolCallId: call.id,
      severity: call.riskSeverity || '',
      riskScore: call.riskScore || 1,
      riskCount: call.riskCount || 0,
      ruleIds: call.riskRuleIds || []
    }))
}

function hasRiskFields(call: ToolCall) {
  return Boolean(call.riskScore || call.riskSeverity || call.riskCount || call.riskRuleIds?.length)
}

function currentToolCallFilters(ctx: ToolCallExplorerContext): ToolCallFilters {
  return {
    tool: ctx.state.toolFilter.value,
    agent: ctx.state.agentFilter.value,
    from: dateTimeInputToQueryIso(ctx.state.fromFilter.value),
    to: dateTimeInputToQueryIso(ctx.state.toFilter.value, 'end'),
    sort: ctx.state.sortFilter.value === DEFAULT_SORT ? undefined : ctx.state.sortFilter.value,
    shell: ctx.mode === 'shell',
    riskOnly: ctx.mode === 'shell' && ctx.state.riskOnlyFilter.value,
    includeRisk: ctx.mode === 'shell',
    limit: TOOL_CALL_LIMIT
  }
}

function clearMissingToolFilter(ctx: ToolCallExplorerContext) {
  if (!ctx.state.toolFilter.value) return false
  if (ctx.availableTools.value.some((item) => item.toolName === ctx.state.toolFilter.value)) return false
  ctx.state.toolFilter.value = undefined
  return true
}

async function loadToolOptions(ctx: ToolCallExplorerContext, clearInvalidTool = false) {
  ctx.state.toolLoading.value = true
  try {
    ctx.state.tools.value = (await api.getTools({ agent: ctx.state.agentFilter.value })) || []
    return clearInvalidTool ? clearMissingToolFilter(ctx) : false
  } finally {
    ctx.state.toolLoading.value = false
  }
}

function currentRouteQuery(ctx: ToolCallExplorerContext) {
  const query = copyStringRouteQuery(ctx.route.query)
  setRouteQueryValue(query, 'tool', ctx.state.toolFilter.value)
  setRouteQueryValue(query, 'command', ctx.mode === 'shell' ? ctx.state.commandFilter.value : undefined)
  setRouteQueryValue(query, 'risk', ctx.mode === 'shell' && ctx.state.riskOnlyFilter.value ? '1' : undefined)
  setRouteQueryValue(query, 'agent', ctx.state.agentFilter.value)
  setRouteQueryValue(query, 'from', ctx.state.fromFilter.value || undefined)
  setRouteQueryValue(query, 'to', ctx.state.toFilter.value || undefined)
  setRouteQueryValue(query, 'sort', ctx.state.sortFilter.value === DEFAULT_SORT ? undefined : ctx.state.sortFilter.value)
  return query
}

async function replaceRouteQuery(ctx: ToolCallExplorerContext) {
  ctx.routeUpdate.applying = true
  try {
    await ctx.router.replace({ path: ctx.routePath, query: currentRouteQuery(ctx) })
  } finally {
    ctx.routeUpdate.applying = false
  }
}

async function updateFilters(ctx: ToolCallExplorerContext, changedFilter?: 'agent') {
  if (changedFilter === 'agent') await loadToolOptions(ctx, true)
  await replaceRouteQuery(ctx)
  loadToolCalls(ctx)
}

function syncFiltersFromRoute(ctx: ToolCallExplorerContext) {
  ctx.state.toolFilter.value = stringRouteQueryValue(ctx.route.query.tool)
  ctx.state.commandFilter.value = ctx.mode === 'shell' ? stringRouteQueryValue(ctx.route.query.command) : undefined
  ctx.state.riskOnlyFilter.value = ctx.mode === 'shell' && stringRouteQueryValue(ctx.route.query.risk) === '1'
  ctx.state.agentFilter.value = stringRouteQueryValue(ctx.route.query.agent)
  ctx.state.fromFilter.value = routeDateTimeInputValue(ctx.route.query, 'from')
  ctx.state.toFilter.value = routeDateTimeInputValue(ctx.route.query, 'to')
  ctx.state.sortFilter.value = routeSortQuery(ctx.route.query)
}

function resetFilters(ctx: ToolCallExplorerContext) {
  ctx.state.toolFilter.value = undefined
  ctx.state.commandFilter.value = undefined
  ctx.state.riskOnlyFilter.value = false
  ctx.state.agentFilter.value = undefined
  ctx.state.fromFilter.value = ''
  ctx.state.toFilter.value = ''
  ctx.state.sortFilter.value = DEFAULT_SORT
  updateFilters(ctx, 'agent')
}

function openToolCall(ctx: ToolCallExplorerContext, call: ToolCall) {
  ctx.state.selectedToolCall.value = call
}

function closeToolCall(ctx: ToolCallExplorerContext) {
  ctx.state.selectedToolCall.value = null
}

function openSession(ctx: ToolCallExplorerContext, id: number) {
  ctx.router.push(`/sessions/${id}`)
}

function registerRouteSync(ctx: ToolCallExplorerContext) {
  watch(
    () => [ctx.route.query.tool, ctx.route.query.command, ctx.route.query.risk, ctx.route.query.agent, ctx.route.query.from, ctx.route.query.to, ctx.route.query.sort],
    async () => {
      if (ctx.routeUpdate.applying) return
      syncFiltersFromRoute(ctx)
      const clearedTool = await loadToolOptions(ctx, true)
      if (clearedTool) await replaceRouteQuery(ctx)
      loadToolCalls(ctx)
    }
  )
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
