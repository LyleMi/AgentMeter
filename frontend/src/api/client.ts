import type {
  AuditFinding,
  AuditFindingFilters,
  AuditSummary,
  IndexResult,
  ModelSignals,
  Overview,
  PricingModelInput,
  PricingModel,
  PrivacyConfigApplyResult,
  PrivacyConfigChange,
  PrivacyConfigStatus,
  PrivacyProfileId,
  PrivacyTarget,
  Session,
  SessionDetail,
  SessionFilters,
  Settings,
  SourceEntry,
  TokenAnalytics,
  ToolCall,
  ToolCallFilters,
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

const fetchApi = {
  getSettings: () => request<Settings>('/api/settings'),
  saveSourceSettings: (sourceEntries: SourceEntry[]) =>
    request<Settings>('/api/settings', { method: 'POST', body: JSON.stringify({ sourceEntries }) }),
  getAgentPrivacy: (target: PrivacyTarget) => request<PrivacyConfigStatus>(`/api/privacy/${target}`),
  applyAgentPrivacyChanges: (target: PrivacyTarget, changes: PrivacyConfigChange[]) =>
    request<PrivacyConfigApplyResult>(`/api/privacy/${target}/changes`, {
      method: 'POST',
      body: JSON.stringify({ changes })
    }),
  applyAgentPrivacyProfile: (target: PrivacyTarget, profile: PrivacyProfileId) =>
    request<PrivacyConfigApplyResult>(`/api/privacy/${target}/profile`, {
      method: 'POST',
      body: JSON.stringify({ profile })
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
      limit: filters.limit,
      offset: filters.offset
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
