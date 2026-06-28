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

export interface SourceIdentity {
  sourceId?: number
  sourceKey?: string
  sourceLabel?: string
  sourceRootPath?: string
  sourceSessionsPath?: string
}

export interface Session extends SourceIdentity {
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
  cachedInputTokens: number
  outputTokens: number
  cacheUtilizationRate: number
  toolCalls: number
  estimatedCostUsd?: number
}

export interface CacheHitTrendPoint {
  date: string
  sessionCount: number
  totalTokens: number
  inputTokens: number
  cachedInputTokens: number
  cacheUtilizationRate: number
  rollingCacheUtilizationRate: number
  lowInputVolume: boolean
  hasUsage: boolean
}

export interface ModelUsage {
  model: string
  sessionCount: number
  totalTokens: number
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  estimatedCostUsd?: number
  unpriced: boolean
}

export interface UsageScopeFilters {
  agent?: string
  model?: string
  project?: string
  from?: string
  to?: string
}

export type UsageBreakdownGroupBy = 'agent' | 'model' | 'agent,model' | 'day' | 'project'

export interface UsageBreakdownFilters extends UsageScopeFilters {
  groupBy: UsageBreakdownGroupBy
}

export interface UsageBreakdown {
  groupBy: UsageBreakdownGroupBy
  buckets: UsageBreakdownBucket[]
}

export interface UsageBreakdownBucket extends SourceIdentity {
  agentKind?: string
  agentName?: string
  model?: string
  date?: string
  projectPath?: string
  sessionCount: number
  totalTokens: number
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  cacheUtilizationRate: number
  estimatedCostUsd?: number
  unpriced: boolean
}

export interface AgentUsage extends SourceIdentity {
  agentKind: string
  agentName: string
  sessionCount: number
  totalTokens: number
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
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

export interface AgentTimeUsage extends SourceIdentity {
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
  cacheHitTrend: CacheHitTrendPoint[]
  modelUsage: ModelUsage[]
  agentUsage: AgentUsage[]
  toolTimeLeaders: ToolTimeUsage[]
  agentTimeUsage: AgentTimeUsage[]
  modelTimeUsage: ModelTimeUsage[]
  slowSessions: Session[]
  recentSessions: Session[]
}

export interface TokenAnalytics {
  totalSessions: number
  totalInputTokens: number
  totalCachedInputTokens: number
  totalOutputTokens: number
  totalReasoningTokens: number
  totalTokens: number
  cacheUtilizationRate: number
  estimatedCostUsd?: number
  unpricedCount: number
  cacheHitTrend: CacheHitTrendPoint[]
  modelUsage: ModelUsage[]
  agentUsage: AgentUsage[]
  recentSessions: Session[]
  highTokenSessions: Session[]
}

export interface ModelSignalRates {
  outputExpansionRate: number
  reasoningTokenShare: number
  cacheMissRate: number
  modelThroughputTokensPerSecond: number
  modelThroughputOutputTokensPerSecond: number
  toolFailureRate: number
  toolDependencyRate: number
}

export interface ModelSignalMetricSet extends ModelSignalRates {
  sessionCount: number
  modelCalls: number
  failedModelCalls?: number
  toolCalls: number
  failedToolCalls: number
  totalTokens: number
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  modelDurationMs: number
  wallDurationMs?: number
  activeDurationMs?: number
  toolDurationMs?: number
  idleDurationMs?: number
  estimatedCostUsd?: number
  unpricedSessionCount?: number
  cacheSavingsUsd?: number
  costPerSession?: number
  costPerActiveHour?: number
  costPer1kTokens?: number
  failurePressure?: number
  avgModelCallsPerSession: number
  modelLatencyMsPer1kOutputTokens: number
  p50ModelLatencyMsPer1kOutputTokens?: number
  p90ModelLatencyMsPer1kOutputTokens?: number
  p50ModelThroughputTokensPerSecond?: number
  p10ModelThroughputTokensPerSecond?: number
}

export interface ModelSignalsWindow {
  from: string
  to: string
  sessionCount: number
  modelCalls: number
}

export interface ModelSignalDriftMetric {
  key: string
  label: string
  direction: string
  severity: string
  current: number
  baseline: number
  delta: number
  deltaPct: number
}

export interface ModelSignalDrift {
  severity: string
  confidence: string
  sampleNote?: string
  reasons: string[]
  metrics: ModelSignalDriftMetric[]
}

export interface ModelSignalsHealthSummary {
  currentWindow: ModelSignalsWindow
  baselineWindow: ModelSignalsWindow
  severity: string
  cohortCount: number
  warningCohorts: number
  criticalCohorts: number
  lowConfidenceCohorts: number
  topReasons: string[]
}

export interface ModelSignalsTrendPoint extends ModelSignalRates {
  date: string
  sessionCount: number
  modelCalls: number
  toolCalls: number
  failedToolCalls: number
  totalTokens: number
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  modelDurationMs: number
  rollingModelThroughputTokensPerSecond: number
  rollingToolFailureRate: number
  lowSample: boolean
}

export interface ModelSignalBreakdown extends ModelSignalRates {
  model: string
  sessionCount: number
  modelCalls: number
  toolCalls: number
  failedToolCalls: number
  totalTokens: number
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  modelDurationMs: number
}

export interface ModelSignalCohort extends SourceIdentity {
  agentKind?: string
  agentName?: string
  modelProvider: string
  model: string
  projectPath?: string
  cohortKey: string
  sessionCount: number
  modelCalls: number
  toolCalls: number
  failedToolCalls: number
  totalTokens: number
  current: ModelSignalMetricSet
  baseline: ModelSignalMetricSet
  drift: ModelSignalDrift
}

export interface ModelSignalMatrixCell {
  model: string
  modelProvider: string
  cohortCount: number
  sessionCount: number
  modelCalls: number
  totalTokens: number
  severity: string
  confidence: string
  keyReason?: string
  current: ModelSignalMetricSet
  baseline: ModelSignalMetricSet
}

export interface ModelSignalMatrixRow extends SourceIdentity {
  agentKind?: string
  agentName?: string
  cells: ModelSignalMatrixCell[]
}

export interface ModelSignalProjectHotspot extends ModelSignalMetricSet {
  projectPath: string
  modelCount: number
  sourceCount: number
  current: ModelSignalMetricSet
  baseline: ModelSignalMetricSet
  drift: ModelSignalDrift
}

export interface ModelSignalsDailyMetric extends ModelSignalMetricSet {
  date: string
  lowSample: boolean
  drift: ModelSignalDrift
  keyReason?: string
}

export interface ModelSignalsProjectMetric extends ModelSignalMetricSet {
  projectPath: string
  modelCount: number
  sourceCount: number
  dominantModelProvider?: string
  dominantModel?: string
  dominantModelShare: number
  current: ModelSignalMetricSet
  baseline: ModelSignalMetricSet
  drift: ModelSignalDrift
}

export interface ModelSignalAnomalySession {
  id?: number
  sessionId?: number
  session?: Session
  sessionKey?: string
  codexSessionId?: string
  startedAt?: string
  projectPath?: string
  rawSourcePath?: string
  agentKind?: string
  agentName?: string
  sourceId?: number
  sourceKey?: string
  sourceLabel?: string
  sourceRootPath?: string
  sourceSessionsPath?: string
  model?: string
  totalTokens?: number
  inputTokens?: number
  outputTokens?: number
  reasoningOutputTokens?: number
  toolCalls?: number
  failedToolCalls?: number
  modelDurationMs?: number
  outputExpansionRate?: number
  reasoningTokenShare?: number
  cacheMissRate?: number
  modelThroughputTokensPerSecond?: number
  toolFailureRate?: number
  toolDependencyRate?: number
  severity?: string
  signal?: string
  reasonLabels?: string[] | string
  reasons?: string[] | string
  signalReasons?: string[] | string
  reason?: string
  [key: string]: unknown
}

export interface ModelSignals {
  totalSessions: number
  totalModelCalls: number
  totalToolCalls: number
  failedToolCalls: number
  toolFailureRate: number
  toolDependencyRate: number
  avgModelCallsPerSession: number
  outputExpansionRate: number
  reasoningTokenShare: number
  cacheMissRate: number
  modelThroughputTokensPerSecond: number
  modelThroughputOutputTokensPerSecond: number
  trend: ModelSignalsTrendPoint[]
  modelBreakdown: ModelSignalBreakdown[]
  anomalySessions: ModelSignalAnomalySession[]
  healthSummary?: ModelSignalsHealthSummary
  cohorts?: ModelSignalCohort[]
  matrix?: ModelSignalMatrixRow[]
  projectHotspots?: ModelSignalProjectHotspot[]
  dailyMetrics?: ModelSignalsDailyMetric[]
  projectMetrics?: ModelSignalsProjectMetric[]
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

export interface ToolCall extends SourceIdentity {
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

export interface AuditFinding extends SourceIdentity {
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
  label?: string
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
  agent?: string
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
