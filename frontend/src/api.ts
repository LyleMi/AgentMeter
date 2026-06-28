import { createDateTimeFormatter, createNumberFormatter, currentLocale } from './i18n'

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
  sessionId: number
  startedAt: string
  endedAt: string
  durationMs: number
  toolName: string
  status: string
  inputSummary: string
  outputSummary: string
  error: string
  callId?: string
  rawEventId: number
  rawStartEventId?: number
  rawEndEventId?: number
  rawEventLine?: number
  rawStartEventLine?: number
  rawEndEventLine?: number
  rawStartEventType?: string
  rawEndEventType?: string
  rawStartEventSummary?: string
  rawEndEventSummary?: string
  rawStartEventJson?: string
  rawEndEventJson?: string
  sessionKey?: string
  codexSessionId?: string
  projectPath?: string
  agentKind?: string
  agentName?: string
  rawSourcePath?: string
}

export interface AuditFinding {
  id: number
  sessionId: number
  toolCallId: number
  sourceFileId: number
  rawEventId: number
  sourceLine: number
  timestamp: string
  source: string
  eventType: string
  category: string
  severity: string
  ruleId: string
  title: string
  description: string
  evidence: string
  command: string
  shellFamily: string
  platform: string
  decision: string
  createdAt: string
  sessionKey?: string
  codexSessionId?: string
  projectPath?: string
  agentKind?: string
  agentName?: string
  rawSourcePath?: string
}

export interface AuditSummary {
  totalFindings: number
  criticalFindings: number
  highFindings: number
  mediumFindings: number
  lowFindings: number
  commandFindings: number
  privacyFindings: number
  egressFindings: number
  fileFindings: number
  sessionsWithFindings: number
  recentFindings: AuditFinding[]
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

export interface SourceEntry {
  path: string
  enabled: boolean
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
  sourceEntries: SourceEntry[]
  defaultSourcePath: string
  defaultSourcePaths: string[]
  databasePath: string
  pricingModels: PricingModel[]
  lastIndexStartedAt?: string
  lastIndexResult?: IndexResult
}

export interface PrivacyConfigSummary {
  score: number
  total: number
  hardened: number
  attention: number
  implicit: number
}

export type PrivacyConfigValueType = 'bool' | 'string' | 'stringArray'

export interface PrivacyConfigSetting {
  id: string
  group: string
  title: string
  description: string
  key: string
  desiredValue: unknown
  strictValue: unknown
  currentValue: unknown
  valueType: PrivacyConfigValueType
  configured: boolean
  supportsUnset: boolean
  status: string
  impact: string
  canApply: boolean
}

export interface PrivacyConfigStatus {
  target: string
  name: string
  configPath: string
  exists: boolean
  summary: PrivacyConfigSummary
  settings: PrivacyConfigSetting[]
  warnings: string[]
}

export interface PrivacyConfigChanged {
  id: string
  key: string
  before: unknown
  after: unknown
}

export interface PrivacyConfigApplyResult {
  status: PrivacyConfigStatus
  changed: PrivacyConfigChanged[]
  backupPath?: string
  warnings: string[]
}

export interface PrivacyConfigChange {
  id: string
  op: 'set' | 'unset'
  value?: unknown
}

export type PrivacyTarget = 'codex' | 'gemini'

export interface SessionFilters {
  search?: string
  model?: string
  agent?: string
  limit?: number
  offset?: number
}

export interface ToolCallFilters {
  tool?: string
  agent?: string
  from?: string
  to?: string
  sort?: string
  limit?: number
  offset?: number
}

export interface ToolFilters {
  agent?: string
}

export interface AuditFindingFilters {
  category?: string
  severity?: string
  shell?: string
  search?: string
  limit?: number
  offset?: number
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

export interface ToolStat {
  toolName: string
  calls: number
  successCalls: number
  failedCalls: number
  totalDurationMs: number
  avgDurationMs: number
}

function localizedFallback(key: 'unknown' | 'unpriced') {
  if (currentLocale.value === 'zh-CN') return key === 'unknown' ? '未知' : '未定价'
  return key
}

export function formatNumber(value: number | undefined): string {
  return createNumberFormatter().format(value || 0)
}

export function formatCost(value?: number): string {
  if (value === undefined || value === null) return localizedFallback('unpriced')
  return createNumberFormatter({ style: 'currency', currency: 'USD', maximumFractionDigits: 4 }).format(value)
}

export function formatDuration(ms: number | undefined): string {
  const total = Math.max(0, Math.round((ms || 0) / 1000))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const seconds = total % 60
  if (currentLocale.value === 'zh-CN') {
    if (hours > 0) return `${hours}小时 ${minutes}分钟`
    if (minutes > 0) return `${minutes}分钟 ${seconds}秒`
    return `${seconds}秒`
  }
  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

export function formatDateTime(value?: string): string {
  if (!value) return '-'
  return createDateTimeFormatter({
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}

export function shortPath(value: string): string {
  if (!value) return localizedFallback('unknown')
  const parts = value.split(/[\\/]/).filter(Boolean)
  if (parts.length <= 3) return value
  return `.../${parts.slice(-3).join('/')}`
}

export function sessionLabel(session: Pick<Session, 'id' | 'sessionKey' | 'codexSessionId'>): string {
  return session.sessionKey || session.codexSessionId || `#${session.id}`
}
