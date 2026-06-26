export interface Usage {
  model: string
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  totalTokens: number
  source: string
  costUsd?: number
  unpriced: boolean
}

export interface Session {
  id: number
  agentKind: string
  agentName: string
  sessionKey: string
  codexSessionId?: string
  projectPath: string
  model: string
  modelProvider: string
  originator: string
  threadSource: string
  startedAt: string
  endedAt: string
  wallDurationMs: number
  activeDurationMs: number
  modelDurationMs: number
  toolDurationMs: number
  idleDurationMs: number
  eventCount: number
  parseStatus: string
  tokenUsage: Usage
  estimatedCostUsd?: number
  unpriced: boolean
  toolCallCount: number
  rawSourcePath: string
  lastIndexedScanStatus: string
  lastIndexedScanMessage: string
}

export interface DailyUsage {
  date: string
  sessionCount: number
  totalTokens: number
  inputTokens: number
  outputTokens: number
  toolCalls: number
  estimatedCostUsd?: number
}

export interface ModelUsage {
  model: string
  sessionCount: number
  totalTokens: number
  inputTokens: number
  outputTokens: number
  estimatedCostUsd?: number
  unpriced: boolean
}

export interface AgentUsage {
  agentKind: string
  agentName: string
  sessionCount: number
  totalTokens: number
  inputTokens: number
  outputTokens: number
  toolCalls: number
  estimatedCostUsd?: number
  unpriced: boolean
}

export interface Overview {
  totalSessions: number
  totalInputTokens: number
  totalCachedInputTokens: number
  totalOutputTokens: number
  totalReasoningTokens: number
  totalTokens: number
  estimatedCostUsd?: number
  unpricedSessions: number
  totalWallDurationMs: number
  totalActiveDurationMs: number
  totalToolCalls: number
  dailyUsage: DailyUsage[]
  modelUsage: ModelUsage[]
  agentUsage: AgentUsage[]
  recentSessions: Session[]
}

export interface EventItem {
  id: number
  sourceLine: number
  timestamp: string
  kind: string
  rawType: string
  summary: string
  rawJson?: string
}

export interface ModelCall {
  id: number
  startedAt: string
  endedAt: string
  durationMs: number
  model: string
  provider: string
  status: string
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  totalTokens: number
  costUsd?: number
  unpriced: boolean
}

export interface ToolCall {
  id: number
  startedAt: string
  endedAt: string
  durationMs: number
  toolName: string
  status: string
  inputSummary: string
  outputSummary: string
  error: string
  rawEventId: number
}

export interface SessionDetail {
  session: Session
  events: EventItem[]
  modelCalls: ModelCall[]
  toolCalls: ToolCall[]
}

export interface PricingModel {
  id: number
  model: string
  normalizedModel: string
  inputPer1m: number
  cachedInputPer1m: number
  outputPer1m: number
  source: string
  effectiveFrom: string
}

export interface IndexResult {
  sourcePath: string
  sourcePaths: string[]
  database: string
  filesSeen: number
  indexed: number
  skipped: number
  failed: number
  sessions: number
  warnings: string[]
  durationMs: number
  rebuild: boolean
}

export interface Settings {
  sourcePath: string
  sourcePaths: string[]
  defaultSourcePath: string
  defaultSourcePaths: string[]
  databasePath: string
  pricingModels: PricingModel[]
  lastIndexStartedAt?: string
  lastIndexResult?: IndexResult
}

export interface SessionFilters {
  search?: string
  model?: string
  agent?: string
  limit?: number
  offset?: number
}

async function call<T>(method: string, args: unknown[], http: () => Promise<T>): Promise<T> {
  const bridge = window.go?.app?.App
  const action = bridge?.[method]
  if (action) {
    return (await action(...args)) as T
  }
  return http()
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
    ...init
  })
  const payload = await response.json()
  if (!response.ok) {
    throw new Error(payload.error || `Request failed: ${response.status}`)
  }
  return payload as T
}

export const api = {
  getSettings: () => call<Settings>('GetSettings', [], () => request('/api/settings')),
  saveSettings: (sourcePath: string) =>
    call<Settings>('SaveSettings', [sourcePath], () =>
      request('/api/settings', { method: 'POST', body: JSON.stringify({ sourcePath }) })
    ),
  indexNow: (rebuild = false) =>
    call<IndexResult>('IndexNow', [rebuild], () =>
      request('/api/index', { method: 'POST', body: JSON.stringify({ rebuild }) })
    ),
  getOverview: () => call<Overview>('GetOverview', [], () => request('/api/overview')),
  listSessions: (filters: SessionFilters = {}) => {
    const params = new URLSearchParams()
    if (filters.search) params.set('search', filters.search)
    if (filters.model) params.set('model', filters.model)
    if (filters.agent) params.set('agent', filters.agent)
    if (filters.limit) params.set('limit', String(filters.limit))
    if (filters.offset) params.set('offset', String(filters.offset))
    return call<Session[]>('ListSessions', [filters], () => request(`/api/sessions?${params}`))
  },
  getSessionDetail: (id: number) => call<SessionDetail>('GetSessionDetail', [id], () => request(`/api/sessions/${id}`)),
  getTools: () => call<ToolStat[]>('GetTools', [], () => request('/api/tools')),
  getPricingModels: () => call<PricingModel[]>('GetPricingModels', [], () => request('/api/pricing'))
}

export interface ToolStat {
  toolName: string
  calls: number
  successCalls: number
  failedCalls: number
  totalDurationMs: number
  avgDurationMs: number
}

export function formatNumber(value: number | undefined): string {
  return new Intl.NumberFormat().format(value || 0)
}

export function formatCost(value?: number): string {
  if (value === undefined || value === null) return 'unpriced'
  return new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: 4 }).format(value)
}

export function formatDuration(ms: number | undefined): string {
  const total = Math.max(0, Math.round((ms || 0) / 1000))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const seconds = total % 60
  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

export function formatDateTime(value?: string): string {
  if (!value) return '-'
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}

export function shortPath(value: string): string {
  if (!value) return 'unknown'
  const parts = value.split(/[\\/]/).filter(Boolean)
  if (parts.length <= 3) return value
  return `.../${parts.slice(-3).join('/')}`
}

export function sessionLabel(session: Pick<Session, 'id' | 'sessionKey' | 'codexSessionId'>): string {
  return session.sessionKey || session.codexSessionId || `#${session.id}`
}
