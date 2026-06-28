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

export interface ToolTimeUsage {
  toolName: string
  calls: number
  successCalls: number
  failedCalls: number
  totalDurationMs: number
  avgDurationMs: number
  maxDurationMs: number
  suspectedNetwork: boolean
}

export interface AgentTimeUsage {
  agentKind: string
  agentName: string
  sessionCount: number
  toolCalls: number
  wallDurationMs: number
  activeDurationMs: number
  modelDurationMs: number
  toolDurationMs: number
  idleDurationMs: number
  suspectedNetworkToolDurationMs: number
}

export interface ModelTimeUsage {
  model: string
  sessionCount: number
  totalTokens: number
  wallDurationMs: number
  activeDurationMs: number
  modelDurationMs: number
  toolDurationMs: number
  idleDurationMs: number
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
  totalModelDurationMs: number
  totalToolDurationMs: number
  totalIdleDurationMs: number
  suspectedNetworkToolDurationMs: number
  suspectedNetworkToolCalls: number
  totalToolCalls: number
  dailyUsage: DailyUsage[]
  modelUsage: ModelUsage[]
  agentUsage: AgentUsage[]
  toolTimeLeaders: ToolTimeUsage[]
  agentTimeUsage: AgentTimeUsage[]
  modelTimeUsage: ModelTimeUsage[]
  slowSessions: Session[]
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

export type PrivacyConfigValueType = 'bool' | 'string' | 'stringArray' | 'number'
export type PrivacyProfileId = 'default' | 'recommended' | 'strict'
export type PrivacyConfigProfileValueOp = 'set' | 'unset' | 'none'

export interface PrivacyConfigProfile {
  id: PrivacyProfileId
  title: string
  description: string
}

export interface PrivacyConfigProfileValue {
  profile: PrivacyProfileId
  op: PrivacyConfigProfileValueOp
  value?: unknown
}

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
  profileValues?: PrivacyConfigProfileValue[]
}

export interface PrivacyConfigStatus {
  target: string
  name: string
  configPath: string
  exists: boolean
  summary: PrivacyConfigSummary
  profiles?: PrivacyConfigProfile[]
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

export type PrivacyTarget = 'codex' | 'gemini' | 'claude' | 'codebuddy'

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

export interface ToolStat {
  toolName: string
  calls: number
  successCalls: number
  failedCalls: number
  totalDurationMs: number
  avgDurationMs: number
}
