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
  PrivacyTarget,
  Session,
  SessionDetail,
  SessionFilters,
  Settings,
  SourceEntry,
  ToolCall,
  ToolCallFilters,
  ToolFilters,
  ToolStat
} from './types'

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

export const api = {
  getSettings: () => request<Settings>('/api/settings'),
  saveSourceSettings: (sourceEntries: SourceEntry[]) =>
    request<Settings>('/api/settings', { method: 'POST', body: JSON.stringify({ sourceEntries }) }),
  getAgentPrivacy: (target: PrivacyTarget) => request<PrivacyConfigStatus>(`/api/privacy/${target}`),
  applyAgentPrivacyChanges: (target: PrivacyTarget, changes: PrivacyConfigChange[]) =>
    request<PrivacyConfigApplyResult>(`/api/privacy/${target}/changes`, {
      method: 'POST',
      body: JSON.stringify({ changes })
    }),
  indexNow: (rebuild = false) =>
    request<IndexResult>('/api/index', { method: 'POST', body: JSON.stringify({ rebuild }) }),
  getOverview: () => request<Overview>('/api/overview'),
  listSessions: (filters: SessionFilters = {}) => {
    const params = new URLSearchParams()
    if (filters.search) params.set('search', filters.search)
    if (filters.model) params.set('model', filters.model)
    if (filters.agent) params.set('agent', filters.agent)
    if (filters.limit) params.set('limit', String(filters.limit))
    if (filters.offset) params.set('offset', String(filters.offset))
    return request<Session[]>(`/api/sessions?${params}`)
  },
  getSessionDetail: (id: number) => request<SessionDetail>(`/api/sessions/${id}`),
  getTools: (filters: ToolFilters = {}) => {
    const params = new URLSearchParams()
    if (filters.agent) params.set('agent', filters.agent)
    const query = params.toString()
    return request<ToolStat[]>(`/api/tools${query ? `?${query}` : ''}`)
  },
  listToolCalls: (filters: ToolCallFilters = {}) => {
    const params = new URLSearchParams()
    if (filters.tool) params.set('tool', filters.tool)
    if (filters.agent) params.set('agent', filters.agent)
    if (filters.from) params.set('from', filters.from)
    if (filters.to) params.set('to', filters.to)
    if (filters.sort) params.set('sort', filters.sort)
    if (filters.limit) params.set('limit', String(filters.limit))
    if (filters.offset) params.set('offset', String(filters.offset))
    return request<ToolCall[]>(`/api/tool-calls?${params}`)
  },
  getAuditSummary: () => request<AuditSummary>('/api/audit/summary'),
  listAuditFindings: (filters: AuditFindingFilters = {}) => {
    const params = new URLSearchParams()
    if (filters.category) params.set('category', filters.category)
    if (filters.severity) params.set('severity', filters.severity)
    if (filters.shell) params.set('shell', filters.shell)
    if (filters.search) params.set('search', filters.search)
    if (filters.limit) params.set('limit', String(filters.limit))
    if (filters.offset) params.set('offset', String(filters.offset))
    return request<AuditFinding[]>(`/api/audit/findings?${params}`)
  },
  getPricingModels: () => request<PricingModel[]>('/api/pricing')
}
