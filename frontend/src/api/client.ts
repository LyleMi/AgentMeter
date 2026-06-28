import type {
  AuditFinding,
  AuditFindingFilters,
  AuditSummary,
  IndexResult,
  Overview,
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

function setTextParam(params: URLSearchParams, key: string, value?: string) {
  if (value) params.set(key, value)
}

function setNumberParam(params: URLSearchParams, key: string, value?: number) {
  if (value) params.set(key, String(value))
}

function usageScopeParams(filters: UsageScopeFilters = {}) {
  const params = new URLSearchParams()
  setTextParam(params, 'agent', filters.agent)
  setTextParam(params, 'model', filters.model)
  setTextParam(params, 'from', filters.from)
  setTextParam(params, 'to', filters.to)
  return params
}

function queryPath(path: string, params: URLSearchParams) {
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
    request<Overview>(queryPath('/api/overview', usageScopeParams(filters))),
  getTokenAnalytics: (filters: UsageScopeFilters = {}) =>
    request<TokenAnalytics>(queryPath('/api/tokens', usageScopeParams(filters))),
  getUsageBreakdown: (filters: UsageBreakdownFilters) => {
    const params = usageScopeParams(filters)
    params.set('groupBy', filters.groupBy)
    return request<UsageBreakdown>(queryPath('/api/usage/breakdown', params))
  },
  listSessions: (filters: SessionFilters = {}) => {
    const params = new URLSearchParams()
    setTextParam(params, 'search', filters.search)
    setTextParam(params, 'model', filters.model)
    setTextParam(params, 'agent', filters.agent)
    setNumberParam(params, 'limit', filters.limit)
    setNumberParam(params, 'offset', filters.offset)
    return request<Session[]>(queryPath('/api/sessions', params))
  },
  getSessionDetail: (id: number) => request<SessionDetail>(`/api/sessions/${id}`),
  getTools: (filters: ToolFilters = {}) => {
    const params = new URLSearchParams()
    setTextParam(params, 'agent', filters.agent)
    return request<ToolStat[]>(queryPath('/api/tools', params))
  },
  listToolCalls: (filters: ToolCallFilters = {}) => {
    const params = new URLSearchParams()
    setTextParam(params, 'tool', filters.tool)
    setTextParam(params, 'agent', filters.agent)
    setTextParam(params, 'from', filters.from)
    setTextParam(params, 'to', filters.to)
    setTextParam(params, 'sort', filters.sort)
    setNumberParam(params, 'limit', filters.limit)
    setNumberParam(params, 'offset', filters.offset)
    return request<ToolCall[]>(queryPath('/api/tool-calls', params))
  },
  getAuditSummary: (filters: Pick<AuditFindingFilters, 'agent'> = {}) => {
    const params = new URLSearchParams()
    setTextParam(params, 'agent', filters.agent)
    return request<AuditSummary>(queryPath('/api/audit/summary', params))
  },
  listAuditFindings: (filters: AuditFindingFilters = {}) => {
    const params = new URLSearchParams()
    setTextParam(params, 'agent', filters.agent)
    setTextParam(params, 'category', filters.category)
    setTextParam(params, 'severity', filters.severity)
    setTextParam(params, 'shell', filters.shell)
    setTextParam(params, 'search', filters.search)
    setNumberParam(params, 'limit', filters.limit)
    setNumberParam(params, 'offset', filters.offset)
    return request<AuditFinding[]>(queryPath('/api/audit/findings', params))
  },
  getAuditFinding: (id: number) => request<AuditFinding>(`/api/audit/findings/${id}`),
  getPricingModels: () => request<PricingModel[]>('/api/pricing')
}

export const api = isStaticDemo ? demoApi : fetchApi
