import type {
  AgentMemoryLookup,
  AgentMemoryResource,
  AgentMemoryUpdateInput,
  AgentResourceOperationResult,
  AgentResourceOverview,
  AgentResourceToggleInput,
  AuditFinding,
  AuditFindingFilters,
  AuditSummary,
  IndexResult,
  ModelSignals,
  Overview,
  PricingModelInput,
  PricingModel,
  PromptSuggestion,
  PromptSuggestionFilters,
  PrivacyConfigApplyResult,
  PrivacyConfigChange,
  PrivacyConfigStatus,
  PrivacyProfileId,
  PrivacyTarget,
  SavedPrompt,
  SavedPromptInput,
  Session,
  SessionDetail,
  SessionFilters,
  Settings,
  SourceStorage,
  SourceEntry,
  TokenAnalytics,
  ToolCall,
  ToolCallFilters,
  ToolCallRiskFilters,
  ToolCallRiskSummary,
  ToolFilters,
  ToolStat,
  UsageBreakdown,
  UsageBreakdownFilters,
  UsageScopeFilters
} from './types'
import { demoApi } from './demo'

declare global {
  interface ImportMeta {
    readonly env: {
      readonly VITE_AGENTMETER_STATIC_DEMO?: string
    }
  }
}

export const isStaticDemo = import.meta.env.VITE_AGENTMETER_STATIC_DEMO === 'true'

type QueryParamValue = string | number | undefined
type QueryParamValues = Record<string, QueryParamValue>

function arrayOrEmpty<T>(value: T[] | null | undefined): T[] {
  return Array.isArray(value) ? value : []
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init
  })
  const raw = await response.text()
  let payload: unknown = null
  if (raw) {
    try {
      payload = JSON.parse(raw)
    } catch {
      const contentType = response.headers.get('Content-Type') || 'unknown content type'
      throw new Error(`Expected JSON from ${path}, got ${contentType}`)
    }
  }
  if (!response.ok) {
    const error = payload && typeof payload === 'object' && 'error' in payload ? String(payload.error) : ''
    throw new Error(error || `Request failed: ${response.status}`)
  }
  return payload as T
}

function queryParams(values: QueryParamValues) {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(values)) {
    if (value) params.set(key, String(value))
  }
  return params
}

function usageScopeParamValues(filters: UsageScopeFilters = {}): QueryParamValues {
  return {
    agent: filters.agent,
    model: filters.model,
    project: filters.project,
    from: filters.from,
    to: filters.to
  }
}

function queryPath(path: string, values: QueryParamValues = {}) {
  const params = queryParams(values)
  const query = params.toString()
  return query ? `${path}?${query}` : path
}

function normalizeAgentResourceOverview(value: Partial<AgentResourceOverview> | null | undefined): AgentResourceOverview {
  return {
    agents: arrayOrEmpty(value?.agents).map((agent) => ({
      ...agent,
      warnings: arrayOrEmpty(agent.warnings),
      supports: arrayOrEmpty(agent.supports),
      unsupported: arrayOrEmpty(agent.unsupported)
    })),
    skills: arrayOrEmpty(value?.skills),
    mcpServers: arrayOrEmpty(value?.mcpServers).map((server) => ({
      ...server,
      args: arrayOrEmpty(server.args),
      envKeys: arrayOrEmpty(server.envKeys)
    })),
    memories: arrayOrEmpty(value?.memories),
    warnings: arrayOrEmpty(value?.warnings)
  }
}

function agentResourceOverview(result: AgentResourceOverview | AgentResourceOperationResult | null | undefined) {
  if (result && typeof result === 'object' && 'overview' in result) {
    return normalizeAgentResourceOverview(result.overview)
  }
  return normalizeAgentResourceOverview(result)
}

const fetchApi = {
  getSettings: () => request<Settings>('/api/settings'),
  getSourceStorage: () => request<SourceStorage>('/api/settings/storage'),
  getAgentResources: () => request<AgentResourceOverview>('/api/agent-resources').then(normalizeAgentResourceOverview),
  setAgentSkillEnabled: (input: AgentResourceToggleInput) =>
    request<AgentResourceOverview | AgentResourceOperationResult>('/api/agent-resources/skills/enabled', {
      method: 'POST',
      body: JSON.stringify(input)
    }).then(agentResourceOverview),
  setAgentMCPServerEnabled: (input: AgentResourceToggleInput) =>
    request<AgentResourceOverview | AgentResourceOperationResult>('/api/agent-resources/mcp/enabled', {
      method: 'POST',
      body: JSON.stringify(input)
    }).then(agentResourceOverview),
  getAgentMemory: (input: AgentMemoryLookup) =>
    request<AgentMemoryResource>(queryPath('/api/agent-resources/memories/detail', {
      agentKind: input.agentKind,
      path: input.path,
      relativePath: input.relativePath
    })),
  saveAgentMemory: (input: AgentMemoryUpdateInput) =>
    request<AgentMemoryResource>('/api/agent-resources/memories/detail', {
      method: 'POST',
      body: JSON.stringify(input)
    }),
  saveSourceSettings: (sourceEntries: SourceEntry[]) =>
    request<Settings>('/api/settings', { method: 'POST', body: JSON.stringify({ sourceEntries }) }),
  getAgentPrivacy: (target: PrivacyTarget, sourceKey?: string) =>
    request<PrivacyConfigStatus>(queryPath(`/api/privacy/${target}`, { sourceKey })),
  applyAgentPrivacyChanges: (target: PrivacyTarget, changes: PrivacyConfigChange[], sourceKey?: string) =>
    request<PrivacyConfigApplyResult>(`/api/privacy/${target}/changes`, {
      method: 'POST',
      body: JSON.stringify({ changes, sourceKey })
    }),
  applyAgentPrivacyProfile: (target: PrivacyTarget, profile: PrivacyProfileId, sourceKey?: string) =>
    request<PrivacyConfigApplyResult>(`/api/privacy/${target}/profile`, {
      method: 'POST',
      body: JSON.stringify({ profile, sourceKey })
    }),
  indexNow: (rebuild = false) =>
    request<IndexResult>('/api/index', { method: 'POST', body: JSON.stringify({ rebuild }) }),
  getOverview: (filters: UsageScopeFilters = {}) =>
    request<Overview>(queryPath('/api/overview', usageScopeParamValues(filters))),
  getTokenAnalytics: (filters: UsageScopeFilters = {}) =>
    request<TokenAnalytics>(queryPath('/api/tokens', usageScopeParamValues(filters))),
  getModelSignals: (filters: UsageScopeFilters = {}) =>
    request<ModelSignals>(queryPath('/api/model-signals', usageScopeParamValues(filters))),
  getUsageBreakdown: (filters: UsageBreakdownFilters) =>
    request<UsageBreakdown>(queryPath('/api/usage/breakdown', {
      ...usageScopeParamValues(filters),
      groupBy: filters.groupBy
    })),
  getPromptSuggestions: (filters: PromptSuggestionFilters = {}) =>
    request<PromptSuggestion[]>(queryPath('/api/prompts/suggestions', {
      agent: filters.agent,
      project: filters.project,
      search: filters.search,
      limit: filters.limit,
      minCount: filters.minCount
    })),
  listSavedPrompts: () => request<SavedPrompt[]>('/api/prompts/saved'),
  savePrompt: (input: SavedPromptInput) =>
    request<SavedPrompt>('/api/prompts/saved', { method: 'POST', body: JSON.stringify(input) }),
  updateSavedPrompt: (id: number, input: SavedPromptInput) =>
    request<SavedPrompt>(`/api/prompts/saved/${id}`, { method: 'PUT', body: JSON.stringify(input) }),
  deleteSavedPrompt: (id: number) =>
    request<{ ok: boolean }>(`/api/prompts/saved/${id}`, { method: 'DELETE' }),
  recordPromptCopy: (id: number) =>
    request<SavedPrompt>(`/api/prompts/saved/${id}/copy`, { method: 'POST' }),
  ignorePromptSuggestion: (suggestionKey: string) =>
    request<{ ok: boolean }>('/api/prompts/ignored', { method: 'POST', body: JSON.stringify({ suggestionKey }) }),
  unignorePromptSuggestion: (suggestionKey: string) =>
    request<{ ok: boolean }>(`/api/prompts/ignored/${encodeURIComponent(suggestionKey)}`, { method: 'DELETE' }),
  listSessions: (filters: SessionFilters = {}) =>
    request<Session[]>(queryPath('/api/sessions', {
      search: filters.search,
      model: filters.model,
      agent: filters.agent,
      limit: filters.limit,
      offset: filters.offset
    })),
  getSessionDetail: (id: number) => request<SessionDetail>(`/api/sessions/${id}`),
  getTools: (filters: ToolFilters = {}) =>
    request<ToolStat[]>(queryPath('/api/tools', { agent: filters.agent })),
  listToolCalls: (filters: ToolCallFilters = {}) =>
    request<ToolCall[]>(queryPath('/api/tool-calls', {
      tool: filters.tool,
      agent: filters.agent,
      from: filters.from,
      to: filters.to,
      sort: filters.sort,
      shell: filters.shell ? '1' : undefined,
      riskOnly: filters.riskOnly ? '1' : undefined,
      includeRisk: filters.includeRisk ? '1' : undefined,
      limit: filters.limit,
      offset: filters.offset
    })),
  listToolCallRisks: (filters: ToolCallRiskFilters = {}) =>
    request<ToolCallRiskSummary[]>(queryPath('/api/tool-call-risks', {
      agent: filters.agent,
      from: filters.from,
      to: filters.to,
      limit: filters.limit
    })),
  getAuditSummary: (filters: Pick<AuditFindingFilters, 'agent'> = {}) =>
    request<AuditSummary>(queryPath('/api/audit/summary', { agent: filters.agent })),
  listAuditFindings: (filters: AuditFindingFilters = {}) =>
    request<AuditFinding[]>(queryPath('/api/audit/findings', {
      agent: filters.agent,
      category: filters.category,
      severity: filters.severity,
      shell: filters.shell,
      search: filters.search,
      limit: filters.limit,
      offset: filters.offset
    })),
  getAuditFinding: (id: number) => request<AuditFinding>(`/api/audit/findings/${id}`),
  getPricingModels: () => request<PricingModel[]>('/api/pricing'),
  savePricingModel: (pricing: PricingModelInput) =>
    request<PricingModel>('/api/pricing', { method: 'POST', body: JSON.stringify(pricing) })
}

export const api = isStaticDemo ? demoApi : fetchApi
