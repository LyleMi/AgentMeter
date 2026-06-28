import type {
  AgentTimeUsage,
  AgentUsage,
  AuditFinding,
  AuditFindingFilters,
  AuditSummary,
  CacheHitTrendPoint,
  DailyUsage,
  EventItem,
  IndexResult,
  ModelSignalAnomalySession,
  ModelSignalBreakdown,
  ModelSignalCohort,
  ModelSignalDrift,
  ModelSignalDriftMetric,
  ModelSignalMatrixCell,
  ModelSignalMatrixRow,
  ModelSignalMetricSet,
  ModelSignalsDailyMetric,
  ModelSignalsProjectMetric,
  ModelSignalProjectHotspot,
  ModelSignalRates,
  ModelSignalsWindow,
  ModelSignals,
  ModelSignalsHealthSummary,
  ModelSignalsTrendPoint,
  ModelCall,
  ModelTimeUsage,
  ModelUsage,
  Overview,
  PricingModel,
  PrivacyConfigApplyResult,
  PrivacyConfigChange,
  PrivacyConfigProfile,
  PrivacyConfigSetting,
  PrivacyConfigStatus,
  PrivacyProfileId,
  PrivacyTarget,
  Session,
  SessionDetail,
  SessionFilters,
  Settings,
  SourceIdentity,
  SourceEntry,
  TokenAnalytics,
  ToolCall,
  ToolCallFilters,
  ToolFilters,
  ToolStat,
  ToolTimeUsage,
  UsageBreakdown,
  UsageBreakdownBucket,
  UsageBreakdownFilters,
  UsageScopeFilters
} from './types'

interface DemoSource {
  sourceId: number
  sourceKey: string
  sourceLabel: string
  sourceRootPath: string
  sourceSessionsPath: string
  agentKind: string
  agentName: string
}

type DemoApi = {
  getSettings: () => Promise<Settings>
  saveSourceSettings: (sourceEntries: SourceEntry[]) => Promise<Settings>
  getAgentPrivacy: (target: PrivacyTarget) => Promise<PrivacyConfigStatus>
  applyAgentPrivacyChanges: (target: PrivacyTarget, changes: PrivacyConfigChange[]) => Promise<PrivacyConfigApplyResult>
  applyAgentPrivacyProfile: (target: PrivacyTarget, profile: PrivacyProfileId) => Promise<PrivacyConfigApplyResult>
  indexNow: (rebuild?: boolean) => Promise<IndexResult>
  getOverview: (filters?: UsageScopeFilters) => Promise<Overview>
  getTokenAnalytics: (filters?: UsageScopeFilters) => Promise<TokenAnalytics>
  getModelSignals: (filters?: UsageScopeFilters) => Promise<ModelSignals>
  getUsageBreakdown: (filters: UsageBreakdownFilters) => Promise<UsageBreakdown>
  listSessions: (filters?: SessionFilters) => Promise<Session[]>
  getSessionDetail: (id: number) => Promise<SessionDetail>
  getTools: (filters?: ToolFilters) => Promise<ToolStat[]>
  listToolCalls: (filters?: ToolCallFilters) => Promise<ToolCall[]>
  getAuditSummary: (filters?: Pick<AuditFindingFilters, 'agent'>) => Promise<AuditSummary>
  listAuditFindings: (filters?: AuditFindingFilters) => Promise<AuditFinding[]>
  getAuditFinding: (id: number) => Promise<AuditFinding>
  getPricingModels: () => Promise<PricingModel[]>
}

const sources: DemoSource[] = [
  {
    sourceId: 1,
    sourceKey: 'source:1',
    sourceLabel: 'Codex CLI',
    sourceRootPath: 'C:\\Users\\demo\\.codex',
    sourceSessionsPath: 'C:\\Users\\demo\\.codex\\sessions',
    agentKind: 'codex',
    agentName: 'Codex CLI'
  },
  {
    sourceId: 2,
    sourceKey: 'source:2',
    sourceLabel: 'Gemini CLI',
    sourceRootPath: 'C:\\Users\\demo\\.gemini',
    sourceSessionsPath: 'C:\\Users\\demo\\.gemini\\tmp',
    agentKind: 'gemini',
    agentName: 'Gemini CLI'
  },
  {
    sourceId: 3,
    sourceKey: 'source:3',
    sourceLabel: 'Claude Code',
    sourceRootPath: 'C:\\Users\\demo\\.claude',
    sourceSessionsPath: 'C:\\Users\\demo\\.claude\\projects',
    agentKind: 'claude',
    agentName: 'Claude Code'
  }
]

const pricingModels: PricingModel[] = [
  {
    id: 1,
    model: 'gpt-5-codex',
    normalizedModel: 'gpt-5-codex',
    inputPer1m: 1.25,
    cachedInputPer1m: 0.125,
    outputPer1m: 10,
    source: 'demo',
    effectiveFrom: '2026-06-01T00:00:00Z'
  },
  {
    id: 2,
    model: 'gemini-2.5-pro',
    normalizedModel: 'gemini-2.5-pro',
    inputPer1m: 1.25,
    cachedInputPer1m: 0.31,
    outputPer1m: 10,
    source: 'demo',
    effectiveFrom: '2026-06-01T00:00:00Z'
  },
  {
    id: 3,
    model: 'claude-sonnet-4',
    normalizedModel: 'claude-sonnet-4',
    inputPer1m: 3,
    cachedInputPer1m: 0.3,
    outputPer1m: 15,
    source: 'demo',
    effectiveFrom: '2026-06-01T00:00:00Z'
  }
]

function source(index: number): DemoSource {
  return sources[index]
}

function costUsd(model: string, inputTokens: number, cachedInputTokens: number, outputTokens: number): number | undefined {
  const pricing = pricingModels.find((item) => item.normalizedModel === model)
  if (!pricing) return undefined
  return Number(
    (
      ((inputTokens - cachedInputTokens) * pricing.inputPer1m) / 1_000_000 +
      (cachedInputTokens * pricing.cachedInputPer1m) / 1_000_000 +
      (outputTokens * pricing.outputPer1m) / 1_000_000
    ).toFixed(4)
  )
}

function makeSession(
  id: number,
  sourceIndex: number,
  startedAt: string,
  durationMinutes: number,
  model: string,
  projectPath: string,
  inputTokens: number,
  cachedInputTokens: number,
  outputTokens: number,
  reasoningOutputTokens: number,
  toolCallCount: number,
  eventCount: number
): Session {
  const agent = source(sourceIndex)
  const startedMs = Date.parse(startedAt)
  const wallDurationMs = durationMinutes * 60 * 1000
  const modelDurationMs = Math.round(wallDurationMs * 0.46)
  const toolDurationMs = Math.round(wallDurationMs * 0.28)
  const idleDurationMs = Math.max(0, wallDurationMs - modelDurationMs - toolDurationMs)
  const totalTokens = inputTokens + outputTokens
  const estimatedCostUsd = costUsd(model, inputTokens, cachedInputTokens, outputTokens)
  return {
    ...agent,
    id,
    sessionKey: `demo-session-${String(id).padStart(3, '0')}`,
    codexSessionId: agent.agentKind === 'codex' ? `codex-demo-${id}` : undefined,
    projectPath,
    model,
    modelProvider: agent.agentKind === 'gemini' ? 'google' : agent.agentKind === 'claude' ? 'anthropic' : 'openai',
    originator: 'demo',
    threadSource: id % 2 === 0 ? 'resume' : 'new',
    startedAt,
    endedAt: new Date(startedMs + wallDurationMs).toISOString(),
    wallDurationMs,
    activeDurationMs: modelDurationMs + toolDurationMs,
    modelDurationMs,
    toolDurationMs,
    idleDurationMs,
    eventCount,
    parseStatus: 'ok',
    tokenUsage: {
      model,
      inputTokens,
      cachedInputTokens,
      outputTokens,
      reasoningOutputTokens,
      totalTokens,
      source: 'demo transcript',
      costUsd: estimatedCostUsd,
      unpriced: estimatedCostUsd === undefined
    },
    estimatedCostUsd,
    unpriced: estimatedCostUsd === undefined,
    toolCallCount,
    rawSourcePath: `${agent.sourceSessionsPath}\\${String(id).padStart(3, '0')}.jsonl`,
    lastIndexedScanStatus: 'ok',
    lastIndexedScanMessage: 'Demo session indexed from synthetic transcript data'
  }
}

const sessions: Session[] = [
  makeSession(101, 0, '2026-06-28T01:12:00Z', 34, 'gpt-5-codex', 'D:\\work\\checkout\\agentmeter', 128400, 38400, 22100, 6200, 16, 74),
  makeSession(102, 1, '2026-06-27T18:44:00Z', 22, 'gemini-2.5-pro', 'D:\\work\\demo\\pricing-audit', 73200, 14600, 18400, 3100, 11, 49),
  makeSession(103, 0, '2026-06-27T10:05:00Z', 47, 'gpt-5-codex', 'D:\\work\\checkout\\docs-site', 201500, 96100, 35600, 12200, 23, 103),
  makeSession(104, 2, '2026-06-26T15:30:00Z', 18, 'claude-sonnet-4', 'D:\\work\\client\\privacy-review', 52200, 8700, 10100, 0, 8, 38),
  makeSession(105, 1, '2026-06-25T22:18:00Z', 61, 'gemini-2.5-pro', 'D:\\work\\research\\tool-latency', 176300, 44100, 28700, 5200, 31, 128),
  makeSession(106, 0, '2026-06-24T07:42:00Z', 14, 'experimental-local-model', 'D:\\work\\scratch\\offline-index', 31800, 0, 6400, 0, 5, 26)
]

const toolCalls: ToolCall[] = [
  makeToolCall(1001, 101, 0, 4, 'shell_command', 'success', 'rg exported API methods', '18 matching lines', ''),
  makeToolCall(1002, 101, 8, 11, 'apply_patch', 'success', 'add demo API module', 'patch applied', ''),
  makeToolCall(1003, 101, 15, 28, 'npm', 'success', 'npm run build', 'vite build completed', ''),
  makeToolCall(1004, 102, 3, 8, 'read_file', 'success', 'open pricing table', '3 models parsed', ''),
  makeToolCall(1005, 102, 9, 17, 'web_fetch', 'failed', 'fetch vendor pricing page', '', 'network disabled by demo policy'),
  makeToolCall(1006, 103, 5, 13, 'shell_command', 'success', 'list docs routes', '7 markdown files', ''),
  makeToolCall(1007, 103, 19, 41, 'apply_patch', 'success', 'rewrite validation docs', '2 files changed', ''),
  makeToolCall(1008, 104, 4, 9, 'read_file', 'success', 'inspect privacy config', 'config loaded', ''),
  makeToolCall(1009, 105, 2, 21, 'shell_command', 'success', 'run smoke-api script', 'all read-only checks passed', ''),
  makeToolCall(1010, 105, 30, 54, 'browser_screenshot', 'success', 'capture tools chart', 'screenshot stored', ''),
  makeToolCall(1011, 106, 3, 7, 'shell_command', 'success', 'scan local jsonl files', '5 files discovered', '')
]

sessions.forEach((session) => {
  session.toolCallCount = toolCalls.filter((call) => call.sessionId === session.id).length
})

function makeToolCall(
  id: number,
  sessionId: number,
  startMinute: number,
  endMinute: number,
  toolName: string,
  status: string,
  inputSummary: string,
  outputSummary: string,
  error: string
): ToolCall {
  const session = sessions.find((item) => item.id === sessionId)
  if (!session) throw new Error(`Missing demo session ${sessionId}`)
  const startedAt = new Date(Date.parse(session.startedAt) + startMinute * 60 * 1000).toISOString()
  const endedAt = new Date(Date.parse(session.startedAt) + endMinute * 60 * 1000).toISOString()
  return {
    id,
    sessionId,
    startedAt,
    endedAt,
    durationMs: Math.max(1, endMinute - startMinute) * 60 * 1000,
    toolName,
    status,
    inputSummary,
    outputSummary,
    error,
    callId: `call-${id}`,
    rawEventId: id + 3000,
    rawStartEventId: id + 2000,
    rawEndEventId: id + 3000,
    rawEventLine: 20 + id - 1000,
    rawStartEventLine: 19 + id - 1000,
    rawEndEventLine: 20 + id - 1000,
    rawStartEventType: 'tool_call',
    rawEndEventType: 'tool_result',
    rawStartEventSummary: inputSummary,
    rawEndEventSummary: outputSummary || error,
    rawStartEventJson: JSON.stringify({ type: 'tool_call', toolName, inputSummary }, null, 2),
    rawEndEventJson: JSON.stringify({ type: 'tool_result', status, outputSummary, error }, null, 2),
    sessionKey: session.sessionKey,
    codexSessionId: session.codexSessionId,
    projectPath: session.projectPath,
    agentKind: session.agentKind,
    agentName: session.agentName,
    rawSourcePath: session.rawSourcePath,
    sourceId: session.sourceId,
    sourceKey: session.sourceKey,
    sourceLabel: session.sourceLabel,
    sourceRootPath: session.sourceRootPath,
    sourceSessionsPath: session.sourceSessionsPath
  }
}

const auditFindings: AuditFinding[] = [
  makeFinding(501, 101, 1001, 'command', 'medium', 'shell.powershell.concatenated-delete', 'Review destructive shell composition', 'Remove command was composed with string interpolation.', 'Remove-Item $target -Recurse', 'powershell'),
  makeFinding(502, 102, 1005, 'egress', 'low', 'network.fetch.failed', 'Network access attempted', 'A documentation lookup was attempted while offline demo policy was active.', 'Invoke-WebRequest https://example.invalid/pricing', 'powershell'),
  makeFinding(503, 104, 1008, 'privacy', 'high', 'privacy.telemetry.enabled', 'Telemetry setting needs review', 'Demo privacy config shows a setting that is not hardened.', 'telemetry.enabled = true', 'json'),
  makeFinding(504, 105, 1009, 'file', 'medium', 'file.output.screenshot', 'Generated artifact requires retention decision', 'A browser screenshot artifact was created during validation.', 'browser_screenshot tools chart', 'browser')
]

function makeFinding(
  id: number,
  sessionId: number,
  toolCallId: number,
  category: string,
  severity: string,
  ruleId: string,
  title: string,
  description: string,
  command: string,
  shellFamily: string
): AuditFinding {
  const session = sessions.find((item) => item.id === sessionId)
  if (!session) throw new Error(`Missing demo session ${sessionId}`)
  const tool = toolCalls.find((item) => item.id === toolCallId)
  return {
    id,
    sessionId,
    toolCallId,
    sourceFileId: session.sourceId || 0,
    rawEventId: tool?.rawEventId || id + 5000,
    sourceLine: tool?.rawEventLine || 1,
    timestamp: tool?.endedAt || session.endedAt,
    source: 'demo audit',
    eventType: tool?.rawEndEventType || 'tool_result',
    category,
    severity,
    ruleId,
    title,
    description,
    evidence: tool?.rawEndEventSummary || description,
    command,
    shellFamily,
    platform: 'windows',
    decision: 'review',
    createdAt: session.endedAt,
    sessionKey: session.sessionKey,
    codexSessionId: session.codexSessionId,
    projectPath: session.projectPath,
    agentKind: session.agentKind,
    agentName: session.agentName,
    rawSourcePath: session.rawSourcePath,
    sourceId: session.sourceId,
    sourceKey: session.sourceKey,
    sourceLabel: session.sourceLabel,
    sourceRootPath: session.sourceRootPath,
    sourceSessionsPath: session.sourceSessionsPath
  }
}

const profiles: PrivacyConfigProfile[] = [
  { id: 'default', title: 'Default', description: 'Leave vendor defaults in place.' },
  { id: 'recommended', title: 'Recommended', description: 'Disable telemetry while preserving local productivity features.' },
  { id: 'strict', title: 'Strict', description: 'Disable telemetry, network helpers, memory, and extended local retention.' }
]

function privacySetting(
  id: string,
  group: string,
  title: string,
  key: string,
  desiredValue: unknown,
  strictValue: unknown,
  currentValue: unknown,
  configured: boolean,
  status: string,
  valueType: PrivacyConfigSetting['valueType'] = 'bool'
): PrivacyConfigSetting {
  return {
    id,
    group,
    title,
    description: `Demo status for ${title.toLowerCase()}.`,
    key,
    desiredValue,
    strictValue,
    currentValue,
    valueType,
    configured,
    supportsUnset: true,
    status,
    impact: `Controls ${title.toLowerCase()} behavior for the selected agent.`,
    canApply: true,
    profileValues: [
      { profile: 'default', op: 'unset' },
      { profile: 'recommended', op: 'set', value: desiredValue },
      { profile: 'strict', op: 'set', value: strictValue }
    ]
  }
}

function privacyStatus(target: PrivacyTarget): PrivacyConfigStatus {
  const targetName: Record<PrivacyTarget, string> = {
    codex: 'Codex',
    gemini: 'Gemini CLI',
    claude: 'Claude Code',
    codebuddy: 'CodeBuddy'
  }
  const settings = [
    privacySetting('analytics.enabled', 'Telemetry', 'Analytics', 'analytics.enabled', false, false, false, true, 'hardened'),
    privacySetting('telemetry.enabled', 'Telemetry', 'Telemetry export', 'telemetry.enabled', false, false, target === 'claude', target !== 'claude', target === 'claude' ? 'attention' : 'hardened'),
    privacySetting('web_search', 'Network', 'Web search', 'web_search', false, false, false, target === 'codex', target === 'codex' ? 'hardened' : 'implicit'),
    privacySetting('history.persistence', 'Local history', 'Conversation history', 'history.persistence', false, false, true, false, 'implicit'),
    privacySetting('retention.days', 'Local retention', 'Retention days', 'retention.days', 14, 7, 14, true, 'hardened', 'number')
  ]
  const hardened = settings.filter((setting) => setting.status === 'hardened').length
  const attention = settings.filter((setting) => setting.status === 'attention').length
  const implicit = settings.filter((setting) => setting.status === 'implicit').length
  return {
    target,
    name: targetName[target],
    configPath: `C:\\Users\\demo\\.${target}\\${target === 'codex' ? 'config.toml' : 'settings.json'}`,
    exists: true,
    summary: {
      score: Math.round((hardened / settings.length) * 100),
      total: settings.length,
      hardened,
      attention,
      implicit
    },
    profiles,
    settings,
    warnings: ['Static demo mode is read-only. No local agent config will be changed.']
  }
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function matchesAgent(record: { sourceId?: number; sourceKey?: string; sourceLabel?: string; agentKind?: string; agentName?: string }, agent?: string): boolean {
  const normalized = (agent || '').trim().toLowerCase()
  if (!normalized) return true
  return [record.sourceKey, record.sourceId !== undefined ? `source:${record.sourceId}` : '', record.sourceLabel, record.agentKind, record.agentName]
    .some((value) => (value || '').toLowerCase() === normalized)
}

function matchesDateRange(value: string, filters: UsageScopeFilters | ToolCallFilters): boolean {
  const timestamp = Date.parse(value)
  if (filters.from && timestamp < Date.parse(filters.from)) return false
  if (filters.to && timestamp > Date.parse(filters.to)) return false
  return true
}

function matchesProject(record: { projectPath?: string; rawSourcePath?: string }, project?: string): boolean {
  const normalized = (project || '').trim()
  if (!normalized) return true
  const projectKey = projectPathKey(normalized)
  return [record.projectPath, record.rawSourcePath]
    .some((value) => {
      const candidate = (value || '').trim()
      return candidate === normalized || projectPathKey(candidate) === projectKey
    })
}

function filteredSessions(filters: UsageScopeFilters & SessionFilters = {}): Session[] {
  const search = (filters.search || '').trim().toLowerCase()
  return sessions
    .filter((session) => matchesAgent(session, filters.agent))
    .filter((session) => !filters.model || session.model === filters.model)
    .filter((session) => matchesProject(session, filters.project))
    .filter((session) => matchesDateRange(session.startedAt, filters))
    .filter((session) => {
      if (!search) return true
      return [session.sessionKey, session.codexSessionId, session.projectPath, session.model, session.agentName, session.sourceLabel]
        .some((value) => (value || '').toLowerCase().includes(search))
    })
    .sort((left, right) => Date.parse(right.startedAt) - Date.parse(left.startedAt))
}

function paginate<T>(items: T[], limit?: number, offset?: number): T[] {
  const start = Math.max(0, offset || 0)
  const end = limit ? start + Math.max(0, limit) : undefined
  return items.slice(start, end)
}

function sum(items: Session[], selector: (session: Session) => number): number {
  return items.reduce((total, session) => total + selector(session), 0)
}

function groupedBy<T>(items: T[], keyFor: (item: T) => string): Map<string, T[]> {
  const groups = new Map<string, T[]>()
  items.forEach((item) => {
    const key = keyFor(item)
    groups.set(key, [...(groups.get(key) || []), item])
  })
  return groups
}

function projectPathKey(value: string): string {
  const normalized = value.trim().replace(/[\\/]\.$/, '').replace(/[\\/]+$/, '')
  return normalized ? normalized.toLowerCase() : 'unknown'
}

function modelUsageFor(items: Session[]): ModelUsage[] {
  return [...groupedBy(items, (session) => session.model)].map(([model, group]) => ({
    model,
    sessionCount: group.length,
    totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
    inputTokens: sum(group, (session) => session.tokenUsage.inputTokens),
    cachedInputTokens: sum(group, (session) => session.tokenUsage.cachedInputTokens),
    outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
    reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
    estimatedCostUsd: costSum(group),
    unpriced: group.some((session) => session.unpriced)
  })).sort((left, right) => right.totalTokens - left.totalTokens)
}

function agentUsageFor(items: Session[]): AgentUsage[] {
  return [...groupedBy(items, (session) => session.sourceKey || session.agentKind)].map(([, group]) => {
    const first = group[0]
    return {
      sourceId: first.sourceId,
      sourceKey: first.sourceKey,
      sourceLabel: first.sourceLabel,
      sourceRootPath: first.sourceRootPath,
      sourceSessionsPath: first.sourceSessionsPath,
      agentKind: first.agentKind,
      agentName: first.agentName,
      sessionCount: group.length,
      totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
      inputTokens: sum(group, (session) => session.tokenUsage.inputTokens),
      cachedInputTokens: sum(group, (session) => session.tokenUsage.cachedInputTokens),
      outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
      reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
      toolCalls: sum(group, (session) => session.toolCallCount),
      estimatedCostUsd: costSum(group),
      unpriced: group.some((session) => session.unpriced)
    }
  }).sort((left, right) => right.totalTokens - left.totalTokens)
}

function costSum(items: Session[]): number | undefined {
  const priced = items.filter((session) => !session.unpriced)
  if (priced.length !== items.length) return undefined
  return Number(priced.reduce((total, session) => total + (session.estimatedCostUsd || 0), 0).toFixed(4))
}

function cacheSavingsUsdFor(items: Session[]): number | undefined {
  let total = 0
  let hasSavings = false
  for (const session of items) {
    const pricing = pricingModels.find((item) => item.normalizedModel === session.model)
    if (!pricing) continue
    const cachedInputTokens = session.tokenUsage.cachedInputTokens || 0
    const savings = (cachedInputTokens * Math.max(0, pricing.inputPer1m - pricing.cachedInputPer1m)) / 1_000_000
    if (savings > 0) {
      total += savings
      hasSavings = true
    }
  }
  return hasSavings ? Number(total.toFixed(4)) : undefined
}

function dailyUsageFor(items: Session[]): DailyUsage[] {
  return [...groupedBy(items, (session) => session.startedAt.slice(0, 10))].map(([date, group]) => {
    const inputTokens = sum(group, (session) => session.tokenUsage.inputTokens)
    const cachedInputTokens = sum(group, (session) => session.tokenUsage.cachedInputTokens)
    return {
      date,
      sessionCount: group.length,
      totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
      inputTokens,
      cachedInputTokens,
      outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
      cacheUtilizationRate: inputTokens > 0 ? cachedInputTokens / inputTokens : 0,
      toolCalls: sum(group, (session) => session.toolCallCount),
      estimatedCostUsd: costSum(group)
    }
  }).sort((left, right) => left.date.localeCompare(right.date))
}

function cacheHitTrendFor(items: Session[]): CacheHitTrendPoint[] {
  const days = dailyUsageFor(items)
  return days.map((day, index) => {
    const window = days.slice(Math.max(0, index - 6), index + 1)
    const rollingInputTokens = window.reduce((total, item) => total + item.inputTokens, 0)
    const rollingCachedInputTokens = window.reduce((total, item) => total + item.cachedInputTokens, 0)
    return {
      date: day.date,
      sessionCount: day.sessionCount,
      totalTokens: day.totalTokens,
      inputTokens: day.inputTokens,
      cachedInputTokens: day.cachedInputTokens,
      cacheUtilizationRate: day.cacheUtilizationRate,
      rollingCacheUtilizationRate: rollingInputTokens > 0 ? rollingCachedInputTokens / rollingInputTokens : 0,
      lowInputVolume: day.inputTokens > 0 && day.inputTokens < 60_000,
      hasUsage: day.sessionCount > 0
    }
  })
}

function modelCallsForSession(session: Session): number {
  return Math.max(1, Math.ceil(session.eventCount / 55))
}

function safeRate(numerator: number, denominator: number): number {
  return denominator > 0 ? numerator / denominator : 0
}

function clampRate(value: number): number {
  if (!Number.isFinite(value) || value < 0) return 0
  if (value > 1) return 1
  return value
}

function percentile(values: number[], percentileRank: number): number | undefined {
  const sorted = values.filter((value) => Number.isFinite(value)).sort((left, right) => left - right)
  if (!sorted.length) return undefined
  const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil(sorted.length * percentileRank) - 1))
  return sorted[index]
}

function sessionLatencyMsPer1kOutputTokens(session: Session): number {
  return safeRate(session.modelDurationMs, session.tokenUsage.outputTokens / 1000)
}

function sessionThroughputTokensPerSecond(session: Session): number {
  return safeRate(session.tokenUsage.totalTokens, session.modelDurationMs / 1000)
}

function isSuccessfulToolStatus(status: string): boolean {
  return status === 'completed' || status === 'success'
}

function signalRatesFor(group: Session[], groupToolCalls: ToolCall[]): ModelSignalRates {
  const inputTokens = sum(group, (session) => session.tokenUsage.inputTokens)
  const cachedInputTokens = sum(group, (session) => session.tokenUsage.cachedInputTokens)
  const outputTokens = sum(group, (session) => session.tokenUsage.outputTokens)
  const reasoningOutputTokens = sum(group, (session) => session.tokenUsage.reasoningOutputTokens)
  const totalTokens = sum(group, (session) => session.tokenUsage.totalTokens)
  const modelDurationSeconds = sum(group, (session) => session.modelDurationMs) / 1000
  const failedToolCalls = groupToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length

  return {
    outputExpansionRate: safeRate(outputTokens, inputTokens),
    reasoningTokenShare: safeRate(reasoningOutputTokens, outputTokens),
    cacheMissRate: clampRate(safeRate(inputTokens - cachedInputTokens, inputTokens)),
    modelThroughputTokensPerSecond: safeRate(totalTokens, modelDurationSeconds),
    modelThroughputOutputTokensPerSecond: safeRate(outputTokens, modelDurationSeconds),
    toolFailureRate: safeRate(failedToolCalls, groupToolCalls.length),
    toolDependencyRate: safeRate(group.filter((session) => session.toolCallCount > 0).length, group.length)
  }
}

interface MetricTotals {
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
  latencySamples?: number[]
  throughputSamples?: number[]
  toolDependencyRate?: number
}

function metricSetFromTotals(totals: MetricTotals): ModelSignalMetricSet {
  const wallDurationMs = totals.wallDurationMs ?? totals.modelDurationMs
  const toolDurationMs = totals.toolDurationMs ?? 0
  const activeDurationMs = totals.activeDurationMs ?? totals.modelDurationMs + toolDurationMs
  const idleDurationMs = totals.idleDurationMs ?? Math.max(0, wallDurationMs - activeDurationMs)
  const modelDurationSeconds = totals.modelDurationMs / 1000
  const activeDurationHours = activeDurationMs / 3_600_000
  const modelLatencyMsPer1kOutputTokens = safeRate(totals.modelDurationMs, totals.outputTokens / 1000)
  const modelThroughputTokensPerSecond = safeRate(totals.totalTokens, modelDurationSeconds)
  const unpricedSessionCount = totals.unpricedSessionCount || 0
  const estimatedCostUsd = totals.estimatedCostUsd
  const hasCompletePricing = estimatedCostUsd !== undefined && unpricedSessionCount === 0
  const costPerSession = hasCompletePricing ? safeRate(estimatedCostUsd, totals.sessionCount) : undefined
  const costPerActiveHour = hasCompletePricing ? safeRate(estimatedCostUsd, activeDurationHours) : undefined
  return {
    sessionCount: totals.sessionCount,
    modelCalls: totals.modelCalls,
    failedModelCalls: totals.failedModelCalls || 0,
    toolCalls: totals.toolCalls,
    failedToolCalls: totals.failedToolCalls,
    totalTokens: totals.totalTokens,
    inputTokens: totals.inputTokens,
    cachedInputTokens: totals.cachedInputTokens,
    outputTokens: totals.outputTokens,
    reasoningOutputTokens: totals.reasoningOutputTokens,
    modelDurationMs: totals.modelDurationMs,
    wallDurationMs,
    activeDurationMs,
    toolDurationMs,
    idleDurationMs,
    estimatedCostUsd: totals.estimatedCostUsd,
    unpricedSessionCount,
    cacheSavingsUsd: totals.cacheSavingsUsd,
    costPerSession,
    costPerActiveHour,
    failurePressure: safeRate((totals.failedModelCalls || 0) + totals.failedToolCalls, totals.sessionCount),
    avgModelCallsPerSession: safeRate(totals.modelCalls, totals.sessionCount),
    outputExpansionRate: safeRate(totals.outputTokens, totals.inputTokens),
    reasoningTokenShare: safeRate(totals.reasoningOutputTokens, totals.outputTokens),
    cacheMissRate: clampRate(safeRate(totals.inputTokens - totals.cachedInputTokens, totals.inputTokens)),
    modelThroughputTokensPerSecond,
    modelThroughputOutputTokensPerSecond: safeRate(totals.outputTokens, modelDurationSeconds),
    modelLatencyMsPer1kOutputTokens,
    p50ModelLatencyMsPer1kOutputTokens: percentile(totals.latencySamples || [], 0.5) ?? modelLatencyMsPer1kOutputTokens,
    p90ModelLatencyMsPer1kOutputTokens: percentile(totals.latencySamples || [], 0.9) ?? modelLatencyMsPer1kOutputTokens,
    p50ModelThroughputTokensPerSecond: percentile(totals.throughputSamples || [], 0.5) ?? modelThroughputTokensPerSecond,
    p10ModelThroughputTokensPerSecond: percentile(totals.throughputSamples || [], 0.1) ?? modelThroughputTokensPerSecond,
    toolFailureRate: safeRate(totals.failedToolCalls, totals.toolCalls),
    toolDependencyRate: totals.toolDependencyRate ?? safeRate(totals.toolCalls, totals.sessionCount)
  }
}

function metricSetFor(group: Session[], groupToolCalls: ToolCall[]): ModelSignalMetricSet {
  const toolSessions = new Set(groupToolCalls.map((call) => call.sessionId)).size
  return metricSetFromTotals({
    sessionCount: group.length,
    modelCalls: sum(group, modelCallsForSession),
    toolCalls: groupToolCalls.length,
    failedToolCalls: groupToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length,
    totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
    inputTokens: sum(group, (session) => session.tokenUsage.inputTokens),
    cachedInputTokens: sum(group, (session) => session.tokenUsage.cachedInputTokens),
    outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
    reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
    modelDurationMs: sum(group, (session) => session.modelDurationMs),
    wallDurationMs: sum(group, (session) => session.wallDurationMs),
    activeDurationMs: sum(group, (session) => session.activeDurationMs),
    toolDurationMs: sum(group, (session) => session.toolDurationMs),
    idleDurationMs: sum(group, (session) => session.idleDurationMs),
    estimatedCostUsd: costSum(group),
    unpricedSessionCount: group.filter((session) => session.unpriced).length,
    cacheSavingsUsd: cacheSavingsUsdFor(group),
    latencySamples: group.map(sessionLatencyMsPer1kOutputTokens),
    throughputSamples: group.map(sessionThroughputTokensPerSecond),
    toolDependencyRate: safeRate(toolSessions, group.length)
  })
}

function stableHash(value: string): number {
  let hash = 0
  for (let index = 0; index < value.length; index += 1) {
    hash = ((hash * 31) + value.charCodeAt(index)) >>> 0
  }
  return hash
}

function syntheticBaselineFor(current: ModelSignalMetricSet, key: string): ModelSignalMetricSet {
  const profile = stableHash(key) % 5
  const durationFactors = [0.64, 0.76, 0.88, 0.96, 1.08]
  const outputFactors = [0.92, 0.96, 1.02, 1.05, 0.98]
  const reasoningFactors = [0.72, 0.82, 0.94, 1.04, 0.9]
  const cacheLift = [0.1, 0.07, 0.04, 0.02, -0.01]

  const inputTokens = Math.max(0, Math.round(current.inputTokens * (profile === 4 ? 1.03 : 0.98)))
  const outputTokens = Math.max(0, Math.round(current.outputTokens * outputFactors[profile]))
  const reasoningOutputTokens = Math.max(0, Math.min(outputTokens, Math.round(current.reasoningOutputTokens * reasoningFactors[profile])))
  const cachedInputTokens = Math.max(
    0,
    Math.min(inputTokens, Math.round((current.cachedInputTokens * 0.95) + (inputTokens * cacheLift[profile])))
  )
  const modelDurationMs = Math.max(1, Math.round(current.modelDurationMs * durationFactors[profile]))
  const wallDurationMs = Math.max(1, Math.round((current.wallDurationMs || current.modelDurationMs) * durationFactors[profile]))
  const toolDurationMs = Math.max(0, Math.round((current.toolDurationMs || 0) * (profile === 0 ? 0.78 : profile === 1 ? 0.86 : 0.96)))
  const activeDurationMs = Math.max(1, modelDurationMs + toolDurationMs)
  const idleDurationMs = Math.max(0, wallDurationMs - activeDurationMs)
  const failedToolCalls = current.failedToolCalls > 0 ? Math.max(0, current.failedToolCalls - 1) : 0
  const estimatedCostUsd = current.estimatedCostUsd === undefined
    ? undefined
    : Number((current.estimatedCostUsd * [0.88, 0.94, 0.99, 1.03, 0.96][profile]).toFixed(4))
  const cacheSavingsUsd = current.cacheSavingsUsd === undefined
    ? undefined
    : Number((current.cacheSavingsUsd * [1.18, 1.12, 1.04, 1, 0.96][profile]).toFixed(4))

  return metricSetFromTotals({
    sessionCount: current.sessionCount + (profile === 3 ? 1 : 0),
    modelCalls: current.modelCalls + (profile === 3 ? 1 : 0),
    failedModelCalls: current.failedModelCalls || 0,
    toolCalls: current.toolCalls,
    failedToolCalls,
    totalTokens: inputTokens + outputTokens,
    inputTokens,
    cachedInputTokens,
    outputTokens,
    reasoningOutputTokens,
    modelDurationMs,
    wallDurationMs,
    activeDurationMs,
    toolDurationMs,
    idleDurationMs,
    estimatedCostUsd,
    unpricedSessionCount: current.unpricedSessionCount,
    cacheSavingsUsd,
    latencySamples: [
      current.p50ModelLatencyMsPer1kOutputTokens || current.modelLatencyMsPer1kOutputTokens,
      current.p90ModelLatencyMsPer1kOutputTokens || current.modelLatencyMsPer1kOutputTokens
    ].map((value) => value * durationFactors[profile]),
    throughputSamples: [
      current.p10ModelThroughputTokensPerSecond || current.modelThroughputTokensPerSecond,
      current.p50ModelThroughputTokensPerSecond || current.modelThroughputTokensPerSecond
    ].map((value) => safeRate(value, durationFactors[profile])),
    toolDependencyRate: current.toolDependencyRate
  })
}

function relativeIncrease(current: number, baseline: number): number {
  if (baseline <= 0) return current > 0 ? 1 : 0
  return (current - baseline) / baseline
}

function relativeDecrease(current: number, baseline: number): number {
  if (baseline <= 0) return 0
  return (baseline - current) / baseline
}

function modelSignalDriftFor(current: ModelSignalMetricSet, baseline: ModelSignalMetricSet): ModelSignalDrift {
  const reasons: string[] = []
  const metrics: ModelSignalDriftMetric[] = []
  let severity = 'healthy'

  const mark = (nextSeverity: string, key: string, label: string, direction: string, reason: string, currentValue: number, baselineValue: number) => {
    if (severityRank(nextSeverity) > severityRank(severity)) severity = nextSeverity
    metrics.push({
      key,
      label,
      direction,
      severity: nextSeverity,
      current: currentValue,
      baseline: baselineValue,
      delta: currentValue - baselineValue,
      deltaPct: baselineValue > 0 ? (currentValue - baselineValue) / baselineValue : 0
    })
    reasons.push(reason)
  }

  const latencyIncrease = relativeIncrease(current.modelLatencyMsPer1kOutputTokens, baseline.modelLatencyMsPer1kOutputTokens)
  if (latencyIncrease >= 0.55) {
    mark('critical', 'modelLatencyMsPer1kOutputTokens', 'model latency per 1k output tokens', 'higher_worse', 'Latency rose vs baseline', current.modelLatencyMsPer1kOutputTokens, baseline.modelLatencyMsPer1kOutputTokens)
  } else if (latencyIncrease >= 0.22) {
    mark('warning', 'modelLatencyMsPer1kOutputTokens', 'model latency per 1k output tokens', 'higher_worse', 'Latency rose vs baseline', current.modelLatencyMsPer1kOutputTokens, baseline.modelLatencyMsPer1kOutputTokens)
  }

  const throughputDrop = relativeDecrease(current.modelThroughputTokensPerSecond, baseline.modelThroughputTokensPerSecond)
  if (throughputDrop >= 0.42) {
    mark('critical', 'modelThroughputTokensPerSecond', 'model throughput', 'lower_worse', 'Throughput fell vs baseline', current.modelThroughputTokensPerSecond, baseline.modelThroughputTokensPerSecond)
  } else if (throughputDrop >= 0.2) {
    mark('warning', 'modelThroughputTokensPerSecond', 'model throughput', 'lower_worse', 'Throughput fell vs baseline', current.modelThroughputTokensPerSecond, baseline.modelThroughputTokensPerSecond)
  }

  const outputThroughputDrop = relativeDecrease(current.modelThroughputOutputTokensPerSecond, baseline.modelThroughputOutputTokensPerSecond)
  if (outputThroughputDrop >= 0.24) {
    mark(outputThroughputDrop >= 0.5 ? 'critical' : 'warning', 'modelThroughputOutputTokensPerSecond', 'model output throughput', 'lower_worse', 'Output throughput fell', current.modelThroughputOutputTokensPerSecond, baseline.modelThroughputOutputTokensPerSecond)
  }

  if (current.failedToolCalls > baseline.failedToolCalls && current.toolFailureRate >= 0.08) {
    mark(current.toolFailureRate >= 0.2 ? 'critical' : 'warning', 'toolFailureRate', 'tool failure rate', 'higher_downstream_symptom', 'Tool failures above baseline', current.toolFailureRate, baseline.toolFailureRate)
  }

  if (current.cacheMissRate - baseline.cacheMissRate >= 0.12) {
    mark('warning', 'cacheMissRate', 'cache miss rate', 'higher_symptom', 'Cache misses above baseline', current.cacheMissRate, baseline.cacheMissRate)
  }

  if (current.reasoningTokenShare - baseline.reasoningTokenShare >= 0.12) {
    mark('warning', 'reasoningTokenShare', 'reasoning token share', 'behavior_higher', 'Reasoning share rose', current.reasoningTokenShare, baseline.reasoningTokenShare)
  }

  const uniqueReasons = [...new Set(reasons)]
  return {
    severity,
    confidence: current.sessionCount < 2 || current.modelCalls < 3 ? 'low' : 'high',
    sampleNote: current.sessionCount < 2 || current.modelCalls < 3 ? 'Low sample' : undefined,
    reasons: uniqueReasons,
    metrics
  }
}

function severityRank(value?: string): number {
  const normalized = (value || '').toLowerCase()
  if (normalized === 'critical' || normalized === 'high') return 3
  if (normalized === 'warning' || normalized === 'medium') return 2
  if (normalized === 'watch' || normalized === 'low') return 1
  return 0
}

function sourceIdentityKey(record: SourceIdentity): string {
  return record.sourceKey || (record.sourceId !== undefined ? `source:${record.sourceId}` : '')
}

function cohortKeyFor(session: Session): string {
  return [
    session.modelProvider || 'unknown',
    session.model || 'unknown',
    sourceIdentityKey(session) || session.agentKind || session.agentName || 'unknown',
    projectPathKey(session.projectPath || session.rawSourcePath)
  ].join('|')
}

function modelSignalCohortsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalCohort[] {
  return [...groupedBy(items, cohortKeyFor)].map(([cohortKey, group]) => {
    const first = group[0]
    const sessionIds = new Set(group.map((session) => session.id))
    const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
    const current = metricSetFor(group, groupToolCalls)
    const baseline = syntheticBaselineFor(current, cohortKey)
    const drift = modelSignalDriftFor(current, baseline)
    return {
      sourceId: first.sourceId,
      sourceKey: first.sourceKey,
      sourceLabel: first.sourceLabel,
      sourceRootPath: first.sourceRootPath,
      sourceSessionsPath: first.sourceSessionsPath,
      agentKind: first.agentKind,
      agentName: first.agentName,
      modelProvider: first.modelProvider,
      model: first.model,
      projectPath: first.projectPath,
      cohortKey,
      sessionCount: current.sessionCount,
      modelCalls: current.modelCalls,
      toolCalls: current.toolCalls,
      failedToolCalls: current.failedToolCalls,
      totalTokens: current.totalTokens,
      current,
      baseline,
      drift
    }
  }).sort((left, right) =>
    severityRank(right.drift.severity) - severityRank(left.drift.severity) ||
    right.totalTokens - left.totalTokens
  )
}

function combineMetricSets(items: ModelSignalMetricSet[]): ModelSignalMetricSet {
  const sessionCount = items.reduce((total, item) => total + item.sessionCount, 0)
  const modelCalls = items.reduce((total, item) => total + item.modelCalls, 0)
  const allPriced = items.every((item) => item.estimatedCostUsd !== undefined)
  const allSavingsPriced = items.every((item) => item.cacheSavingsUsd !== undefined)
  const latencySamples = items.flatMap((item) => [
    item.p50ModelLatencyMsPer1kOutputTokens,
    item.p90ModelLatencyMsPer1kOutputTokens,
    item.modelLatencyMsPer1kOutputTokens
  ].filter((value): value is number => typeof value === 'number' && Number.isFinite(value)))
  const throughputSamples = items.flatMap((item) => [
    item.p10ModelThroughputTokensPerSecond,
    item.p50ModelThroughputTokensPerSecond,
    item.modelThroughputTokensPerSecond
  ].filter((value): value is number => typeof value === 'number' && Number.isFinite(value)))
  const toolDependencyRate = safeRate(
    items.reduce((total, item) => total + item.toolDependencyRate * item.sessionCount, 0),
    sessionCount
  )
  return metricSetFromTotals({
    sessionCount,
    modelCalls,
    failedModelCalls: items.reduce((total, item) => total + (item.failedModelCalls || 0), 0),
    toolCalls: items.reduce((total, item) => total + item.toolCalls, 0),
    failedToolCalls: items.reduce((total, item) => total + item.failedToolCalls, 0),
    totalTokens: items.reduce((total, item) => total + item.totalTokens, 0),
    inputTokens: items.reduce((total, item) => total + item.inputTokens, 0),
    cachedInputTokens: items.reduce((total, item) => total + item.cachedInputTokens, 0),
    outputTokens: items.reduce((total, item) => total + item.outputTokens, 0),
    reasoningOutputTokens: items.reduce((total, item) => total + item.reasoningOutputTokens, 0),
    modelDurationMs: items.reduce((total, item) => total + item.modelDurationMs, 0),
    wallDurationMs: items.reduce((total, item) => total + (item.wallDurationMs || item.modelDurationMs), 0),
    activeDurationMs: items.reduce((total, item) => total + (item.activeDurationMs || item.modelDurationMs), 0),
    toolDurationMs: items.reduce((total, item) => total + (item.toolDurationMs || 0), 0),
    idleDurationMs: items.reduce((total, item) => total + (item.idleDurationMs || 0), 0),
    estimatedCostUsd: allPriced ? Number(items.reduce((total, item) => total + (item.estimatedCostUsd || 0), 0).toFixed(4)) : undefined,
    unpricedSessionCount: items.reduce((total, item) => total + (item.unpricedSessionCount || 0), 0),
    cacheSavingsUsd: allSavingsPriced ? Number(items.reduce((total, item) => total + (item.cacheSavingsUsd || 0), 0).toFixed(4)) : undefined,
    latencySamples,
    throughputSamples,
    toolDependencyRate
  })
}

function modelSignalMatrixFor(cohorts: ModelSignalCohort[]): ModelSignalMatrixRow[] {
  return [...groupedBy(cohorts, (cohort) => sourceIdentityKey(cohort) || cohort.agentKind || cohort.agentName || 'unknown')]
    .map(([, group]) => {
      const first = group[0]
      const cells: ModelSignalMatrixCell[] = [...groupedBy(group, (cohort) => `${cohort.modelProvider}:${cohort.model}`)]
        .map(([, cellCohorts]) => {
          const cellFirst = cellCohorts[0]
          const current = combineMetricSets(cellCohorts.map((cohort) => cohort.current))
          const baseline = combineMetricSets(cellCohorts.map((cohort) => cohort.baseline))
          const drift = modelSignalDriftFor(current, baseline)
          return {
            model: cellFirst.model,
            modelProvider: cellFirst.modelProvider,
            cohortCount: cellCohorts.length,
            sessionCount: current.sessionCount,
            modelCalls: current.modelCalls,
            totalTokens: current.totalTokens,
            severity: drift.severity,
            confidence: drift.confidence,
            keyReason: drift.reasons[0],
            current,
            baseline
          }
        })
        .sort((left, right) => severityRank(right.severity) - severityRank(left.severity) || right.totalTokens - left.totalTokens)
      return {
        sourceId: first.sourceId,
        sourceKey: first.sourceKey,
        sourceLabel: first.sourceLabel,
        sourceRootPath: first.sourceRootPath,
        sourceSessionsPath: first.sourceSessionsPath,
        agentKind: first.agentKind,
        agentName: first.agentName,
        cells
      }
    })
    .sort((left, right) =>
      Math.max(...right.cells.map((cell) => severityRank(cell.severity)), 0) -
      Math.max(...left.cells.map((cell) => severityRank(cell.severity)), 0)
    )
}

function modelSignalProjectHotspotsFor(cohorts: ModelSignalCohort[]): ModelSignalProjectHotspot[] {
  return [...groupedBy(cohorts, (cohort) => projectPathKey(cohort.projectPath || 'unknown'))].map(([, group]) => {
    const current = combineMetricSets(group.map((cohort) => cohort.current))
    const baseline = combineMetricSets(group.map((cohort) => cohort.baseline))
    const drift = modelSignalDriftFor(current, baseline)
    return {
      projectPath: group[0].projectPath || 'unknown',
      sessionCount: current.sessionCount,
      modelCount: new Set(group.map((cohort) => `${cohort.modelProvider}:${cohort.model}`)).size,
      sourceCount: new Set(group.map((cohort) => sourceIdentityKey(cohort) || cohort.agentKind || cohort.agentName)).size,
      totalTokens: current.totalTokens,
      current,
      baseline,
      drift
    }
  }).sort((left, right) =>
    severityRank(right.drift.severity) - severityRank(left.drift.severity) ||
    right.totalTokens - left.totalTokens
  )
}

function modelSignalsDailyMetricsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsDailyMetric[] {
  return [...groupedBy(items, (session) => session.startedAt.slice(0, 10))]
    .map(([date, group]) => {
      const sessionIds = new Set(group.map((session) => session.id))
      const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
      const current = metricSetFor(group, groupToolCalls)
      const baseline = syntheticBaselineFor(current, `daily:${date}`)
      const drift = modelSignalDriftFor(current, baseline)
      return {
        date,
        ...current,
        lowSample: group.length < 2 || current.modelCalls < 3 || current.totalTokens < 60_000,
        drift,
        keyReason: drift.reasons[0] || drift.sampleNote
      }
    })
    .sort((left, right) => right.date.localeCompare(left.date))
}

function modelSignalsProjectMetricsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsProjectMetric[] {
  return [...groupedBy(items, (session) => projectPathKey(session.projectPath || session.rawSourcePath))]
    .map(([, group]) => {
      const first = group[0]
      const sessionIds = new Set(group.map((session) => session.id))
      const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
      const current = metricSetFor(group, groupToolCalls)
      const baseline = syntheticBaselineFor(current, `project:${first.projectPath || first.rawSourcePath}`)
      const drift = modelSignalDriftFor(current, baseline)
      const dominantModelGroup = [...groupedBy(group, (session) => `${session.modelProvider}:${session.model}`)]
        .map(([, modelGroup]) => ({
          modelProvider: modelGroup[0].modelProvider,
          model: modelGroup[0].model,
          sessionCount: modelGroup.length,
          totalTokens: sum(modelGroup, (session) => session.tokenUsage.totalTokens)
        }))
        .sort((left, right) => right.sessionCount - left.sessionCount || right.totalTokens - left.totalTokens)[0]

      return {
        ...current,
        projectPath: first.projectPath || first.rawSourcePath || 'unknown',
        modelCount: new Set(group.map((session) => `${session.modelProvider}:${session.model}`)).size,
        sourceCount: new Set(group.map((session) => sourceIdentityKey(session) || session.agentKind || session.agentName)).size,
        dominantModelProvider: dominantModelGroup?.modelProvider,
        dominantModel: dominantModelGroup?.model,
        dominantModelShare: safeRate(dominantModelGroup?.sessionCount || 0, group.length),
        current,
        baseline,
        drift
      }
    })
    .sort((left, right) =>
      severityRank(right.drift.severity) - severityRank(left.drift.severity) ||
      (right.estimatedCostUsd || 0) - (left.estimatedCostUsd || 0) ||
      right.totalTokens - left.totalTokens
    )
}

function modelSignalsHealthSummaryFor(items: Session[], cohorts: ModelSignalCohort[]): ModelSignalsHealthSummary {
  const reasonCounts = new Map<string, number>()
  for (const cohort of cohorts) {
    for (const reason of cohort.drift.reasons) {
      reasonCounts.set(reason, (reasonCounts.get(reason) || 0) + 1)
    }
  }
  const criticalCohorts = cohorts.filter((cohort) => severityRank(cohort.drift.severity) >= 3).length
  const warningCohorts = cohorts.filter((cohort) => severityRank(cohort.drift.severity) === 2).length
  const lowConfidenceCohorts = cohorts.filter((cohort) => cohort.drift.confidence === 'low').length
  return {
    currentWindow: dateWindow(items),
    baselineWindow: baselineWindow(items),
    severity: criticalCohorts > 0 ? 'critical' : warningCohorts > 0 ? 'warning' : lowConfidenceCohorts > 0 ? 'unknown' : 'healthy',
    cohortCount: cohorts.length,
    warningCohorts,
    criticalCohorts,
    lowConfidenceCohorts,
    topReasons: [...reasonCounts.entries()]
      .sort((left, right) => right[1] - left[1] || left[0].localeCompare(right[0]))
      .map(([reason]) => reason)
      .slice(0, 5)
  }
}

function dateWindow(items: Session[]): ModelSignalsWindow {
  const dates = [...new Set(items.map((session) => session.startedAt.slice(0, 10)))].sort()
  return {
    from: dates[0] ? `${dates[0]}T00:00:00Z` : '',
    to: dates[dates.length - 1] ? `${dates[dates.length - 1]}T23:59:59Z` : '',
    sessionCount: items.length,
    modelCalls: sum(items, modelCallsForSession)
  }
}

function baselineWindow(items: Session[]): ModelSignalsWindow {
  const dates = [...new Set(items.map((session) => session.startedAt.slice(0, 10)))].sort()
  if (!dates.length) {
    return { from: '', to: '', sessionCount: 0, modelCalls: 0 }
  }
  const first = new Date(`${dates[0]}T00:00:00Z`)
  const last = new Date(`${dates[dates.length - 1]}T00:00:00Z`)
  const spanDays = Math.max(1, Math.round((last.getTime() - first.getTime()) / 86_400_000) + 1)
  const baselineEnd = new Date(first.getTime() - 86_400_000)
  const baselineStart = new Date(first.getTime() - spanDays * 86_400_000)
  return {
    from: baselineStart.toISOString(),
    to: baselineEnd.toISOString(),
    sessionCount: items.length,
    modelCalls: sum(items, modelCallsForSession)
  }
}

function modelSignalsTrendFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalsTrendPoint[] {
  const days = [...groupedBy(items, (session) => session.startedAt.slice(0, 10))]
    .map(([date, group]) => signalTrendPoint(date, group, scopedToolCalls))
    .sort((left, right) => left.date.localeCompare(right.date))

  return days.map((day, index) => {
    const window = days.slice(Math.max(0, index - 6), index + 1)
    const windowToolCalls = window.reduce((total, item) => total + item.toolCalls, 0)
    const windowFailedTools = window.reduce((total, item) => total + item.failedToolCalls, 0)
    const windowTokens = window.reduce((total, item) => total + item.totalTokens, 0)
    const windowDurationSeconds = window.reduce((total, item) => total + item.modelDurationMs, 0) / 1000
    return {
      ...day,
      rollingModelThroughputTokensPerSecond: safeRate(windowTokens, windowDurationSeconds),
      rollingToolFailureRate: safeRate(windowFailedTools, windowToolCalls)
    }
  })
}

function signalTrendPoint(date: string, group: Session[], scopedToolCalls: ToolCall[]): ModelSignalsTrendPoint {
  const sessionIds = new Set(group.map((session) => session.id))
  const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
  const inputTokens = sum(group, (session) => session.tokenUsage.inputTokens)
  const cachedInputTokens = sum(group, (session) => session.tokenUsage.cachedInputTokens)
  const outputTokens = sum(group, (session) => session.tokenUsage.outputTokens)
  const reasoningOutputTokens = sum(group, (session) => session.tokenUsage.reasoningOutputTokens)
  const totalTokens = sum(group, (session) => session.tokenUsage.totalTokens)
  const modelDurationMs = sum(group, (session) => session.modelDurationMs)
  const modelCalls = sum(group, modelCallsForSession)
  const failedToolCalls = groupToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length
  return {
    date,
    sessionCount: group.length,
    modelCalls,
    toolCalls: groupToolCalls.length,
    failedToolCalls,
    totalTokens,
    inputTokens,
    cachedInputTokens,
    outputTokens,
    reasoningOutputTokens,
    modelDurationMs,
    ...signalRatesFor(group, groupToolCalls),
    rollingModelThroughputTokensPerSecond: 0,
    rollingToolFailureRate: 0,
    lowSample: group.length < 2 || modelCalls < 2 || totalTokens < 60_000
  }
}

function modelSignalsBreakdownFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalBreakdown[] {
  return [...groupedBy(items, (session) => session.model)].map(([model, group]) => {
    const sessionIds = new Set(group.map((session) => session.id))
    const groupToolCalls = scopedToolCalls.filter((call) => sessionIds.has(call.sessionId))
    return {
      model,
      sessionCount: group.length,
      modelCalls: sum(group, modelCallsForSession),
      toolCalls: groupToolCalls.length,
      failedToolCalls: groupToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length,
      totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
      inputTokens: sum(group, (session) => session.tokenUsage.inputTokens),
      cachedInputTokens: sum(group, (session) => session.tokenUsage.cachedInputTokens),
      outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
      reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
      modelDurationMs: sum(group, (session) => session.modelDurationMs),
      ...signalRatesFor(group, groupToolCalls)
    }
  }).sort((left, right) => right.totalTokens - left.totalTokens)
}

function anomalySessionsFor(items: Session[], scopedToolCalls: ToolCall[]): ModelSignalAnomalySession[] {
  const anomalies: ModelSignalAnomalySession[] = []
  for (const session of items) {
    const sessionToolCalls = scopedToolCalls.filter((call) => call.sessionId === session.id)
    const rates = signalRatesFor([session], sessionToolCalls)
    const reasons: string[] = []
    if (rates.toolFailureRate > 0) reasons.push('Tool failure in session')
    if (rates.reasoningTokenShare >= 0.25) reasons.push('High reasoning token share')
    if (rates.outputExpansionRate >= 0.2) reasons.push('Output expanded relative to input')
    if (rates.cacheMissRate >= 0.85) reasons.push('Low cache reuse')
    if (rates.modelThroughputTokensPerSecond > 0 && rates.modelThroughputTokensPerSecond < 85) reasons.push('Low model token throughput')
    if (!reasons.length) continue
    anomalies.push({
      session,
      id: session.id,
      sessionId: session.id,
      sessionKey: session.sessionKey,
      codexSessionId: session.codexSessionId,
      startedAt: session.startedAt,
      projectPath: session.projectPath,
      rawSourcePath: session.rawSourcePath,
      agentKind: session.agentKind,
      agentName: session.agentName,
      sourceId: session.sourceId,
      sourceKey: session.sourceKey,
      sourceLabel: session.sourceLabel,
      sourceRootPath: session.sourceRootPath,
      sourceSessionsPath: session.sourceSessionsPath,
      model: session.model,
      totalTokens: session.tokenUsage.totalTokens,
      inputTokens: session.tokenUsage.inputTokens,
      outputTokens: session.tokenUsage.outputTokens,
      reasoningOutputTokens: session.tokenUsage.reasoningOutputTokens,
      toolCalls: sessionToolCalls.length,
      failedToolCalls: sessionToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length,
      modelDurationMs: session.modelDurationMs,
      severity: reasons.length > 1 ? 'high' : 'medium',
      signal: reasons[0],
      reasons,
      ...rates
    })
  }
  return anomalies
    .sort((left, right) => {
      const leftReasons = Array.isArray(left.reasons) ? left.reasons.length : 0
      const rightReasons = Array.isArray(right.reasons) ? right.reasons.length : 0
      return rightReasons - leftReasons || (right.totalTokens || 0) - (left.totalTokens || 0)
    })
    .slice(0, 6)
}

function modelSignals(filters: UsageScopeFilters = {}): ModelSignals {
  const scoped = filteredSessions(filters)
  const sessionIds = new Set(scoped.map((session) => session.id))
  const scopedToolCalls = filteredToolCalls({ agent: filters.agent, project: filters.project, from: filters.from, to: filters.to })
    .filter((call) => sessionIds.has(call.sessionId))
  const rates = signalRatesFor(scoped, scopedToolCalls)
  const cohorts = modelSignalCohortsFor(scoped, scopedToolCalls)
  return {
    totalSessions: scoped.length,
    totalModelCalls: sum(scoped, modelCallsForSession),
    totalToolCalls: scopedToolCalls.length,
    failedToolCalls: scopedToolCalls.filter((call) => !isSuccessfulToolStatus(call.status)).length,
    toolFailureRate: rates.toolFailureRate,
    toolDependencyRate: rates.toolDependencyRate,
    avgModelCallsPerSession: safeRate(sum(scoped, modelCallsForSession), scoped.length),
    outputExpansionRate: rates.outputExpansionRate,
    reasoningTokenShare: rates.reasoningTokenShare,
    cacheMissRate: rates.cacheMissRate,
    modelThroughputTokensPerSecond: rates.modelThroughputTokensPerSecond,
    modelThroughputOutputTokensPerSecond: rates.modelThroughputOutputTokensPerSecond,
    trend: modelSignalsTrendFor(scoped, scopedToolCalls),
    modelBreakdown: modelSignalsBreakdownFor(scoped, scopedToolCalls),
    anomalySessions: anomalySessionsFor(scoped, scopedToolCalls),
    healthSummary: modelSignalsHealthSummaryFor(scoped, cohorts),
    cohorts,
    matrix: modelSignalMatrixFor(cohorts),
    projectHotspots: modelSignalProjectHotspotsFor(cohorts),
    dailyMetrics: modelSignalsDailyMetricsFor(scoped, scopedToolCalls),
    projectMetrics: modelSignalsProjectMetricsFor(scoped, scopedToolCalls)
  }
}

function filteredToolCalls(filters: ToolCallFilters & Pick<UsageScopeFilters, 'project'> = {}): ToolCall[] {
  return toolCalls
    .filter((call) => matchesAgent(call, filters.agent))
    .filter((call) => !filters.tool || call.toolName === filters.tool)
    .filter((call) => matchesProject(call, filters.project))
    .filter((call) => matchesDateRange(call.startedAt, filters))
    .sort((left, right) => {
      const direction = filters.sort === 'duration' ? right.durationMs - left.durationMs : Date.parse(right.startedAt) - Date.parse(left.startedAt)
      return direction || right.id - left.id
    })
}

function toolStatsFor(calls: ToolCall[]): ToolStat[] {
  return [...groupedBy(calls, (call) => call.toolName)].map(([toolName, group]) => {
    const totalDurationMs = group.reduce((total, call) => total + call.durationMs, 0)
    return {
      toolName,
      calls: group.length,
      successCalls: group.filter((call) => call.status === 'success').length,
      failedCalls: group.filter((call) => call.status !== 'success').length,
      totalDurationMs,
      avgDurationMs: Math.round(totalDurationMs / group.length)
    }
  }).sort((left, right) => right.calls - left.calls || right.totalDurationMs - left.totalDurationMs)
}

function overview(filters: UsageScopeFilters = {}): Overview {
  const scoped = filteredSessions(filters)
  const scopedToolCalls = filteredToolCalls({ agent: filters.agent, project: filters.project, from: filters.from, to: filters.to })
  const modelUsage = modelUsageFor(scoped)
  const agentUsage = agentUsageFor(scoped)
  const toolTimeLeaders: ToolTimeUsage[] = toolStatsFor(scopedToolCalls).map((tool) => ({
    ...tool,
    maxDurationMs: Math.max(...scopedToolCalls.filter((call) => call.toolName === tool.toolName).map((call) => call.durationMs)),
    suspectedNetwork: ['web_fetch', 'browser_screenshot'].includes(tool.toolName)
  }))
  const agentTimeUsage: AgentTimeUsage[] = agentUsage.map((agent) => {
    const group = scoped.filter((session) => matchesAgent(session, agent.sourceKey))
    return {
      sourceId: agent.sourceId,
      sourceKey: agent.sourceKey,
      sourceLabel: agent.sourceLabel,
      sourceRootPath: agent.sourceRootPath,
      sourceSessionsPath: agent.sourceSessionsPath,
      agentKind: agent.agentKind,
      agentName: agent.agentName,
      sessionCount: group.length,
      toolCalls: sum(group, (session) => session.toolCallCount),
      wallDurationMs: sum(group, (session) => session.wallDurationMs),
      activeDurationMs: sum(group, (session) => session.activeDurationMs),
      modelDurationMs: sum(group, (session) => session.modelDurationMs),
      toolDurationMs: sum(group, (session) => session.toolDurationMs),
      idleDurationMs: sum(group, (session) => session.idleDurationMs),
      suspectedNetworkToolDurationMs: scopedToolCalls
        .filter((call) => matchesAgent(call, agent.sourceKey) && ['web_fetch', 'browser_screenshot'].includes(call.toolName))
        .reduce((total, call) => total + call.durationMs, 0)
    }
  })
  const modelTimeUsage: ModelTimeUsage[] = modelUsage.map((model) => {
    const group = scoped.filter((session) => session.model === model.model)
    return {
      model: model.model,
      sessionCount: group.length,
      totalTokens: model.totalTokens,
      wallDurationMs: sum(group, (session) => session.wallDurationMs),
      activeDurationMs: sum(group, (session) => session.activeDurationMs),
      modelDurationMs: sum(group, (session) => session.modelDurationMs),
      toolDurationMs: sum(group, (session) => session.toolDurationMs),
      idleDurationMs: sum(group, (session) => session.idleDurationMs)
    }
  })
  return {
    totalSessions: scoped.length,
    totalInputTokens: sum(scoped, (session) => session.tokenUsage.inputTokens),
    totalCachedInputTokens: sum(scoped, (session) => session.tokenUsage.cachedInputTokens),
    totalOutputTokens: sum(scoped, (session) => session.tokenUsage.outputTokens),
    totalReasoningTokens: sum(scoped, (session) => session.tokenUsage.reasoningOutputTokens),
    totalTokens: sum(scoped, (session) => session.tokenUsage.totalTokens),
    estimatedCostUsd: costSum(scoped),
    unpricedSessions: scoped.filter((session) => session.unpriced).length,
    totalWallDurationMs: sum(scoped, (session) => session.wallDurationMs),
    totalActiveDurationMs: sum(scoped, (session) => session.activeDurationMs),
    totalModelDurationMs: sum(scoped, (session) => session.modelDurationMs),
    totalToolDurationMs: sum(scoped, (session) => session.toolDurationMs),
    totalIdleDurationMs: sum(scoped, (session) => session.idleDurationMs),
    suspectedNetworkToolDurationMs: scopedToolCalls
      .filter((call) => ['web_fetch', 'browser_screenshot'].includes(call.toolName))
      .reduce((total, call) => total + call.durationMs, 0),
    suspectedNetworkToolCalls: scopedToolCalls.filter((call) => ['web_fetch', 'browser_screenshot'].includes(call.toolName)).length,
    totalToolCalls: scopedToolCalls.length,
    dailyUsage: dailyUsageFor(scoped),
    cacheHitTrend: cacheHitTrendFor(scoped),
    modelUsage,
    agentUsage,
    toolTimeLeaders,
    agentTimeUsage,
    modelTimeUsage,
    slowSessions: [...scoped].sort((left, right) => right.wallDurationMs - left.wallDurationMs).slice(0, 5),
    recentSessions: scoped.slice(0, 5)
  }
}

function breakdown(filters: UsageBreakdownFilters): UsageBreakdown {
  const scoped = filteredSessions(filters)
  const buckets: UsageBreakdownBucket[] = []
  if (filters.groupBy === 'day') {
    dailyUsageFor(scoped).forEach((day) => {
      const group = scoped.filter((session) => session.startedAt.startsWith(day.date))
      buckets.push(bucketFor(group, { date: day.date }))
    })
  } else if (filters.groupBy === 'model') {
    groupedBy(scoped, (session) => session.model).forEach((group, model) => buckets.push(bucketFor(group, { model })))
  } else if (filters.groupBy === 'project') {
    groupedBy(scoped, (session) => projectPathKey(session.projectPath || session.rawSourcePath)).forEach((group) => {
      const projectPath = group[0].projectPath || group[0].rawSourcePath
      buckets.push(bucketFor(group, { projectPath }))
    })
  } else if (filters.groupBy === 'agent') {
    groupedBy(scoped, (session) => session.sourceKey || session.agentKind).forEach((group) => {
      const first = group[0]
      buckets.push(bucketFor(group, {
        sourceId: first.sourceId,
        sourceKey: first.sourceKey,
        sourceLabel: first.sourceLabel,
        sourceRootPath: first.sourceRootPath,
        sourceSessionsPath: first.sourceSessionsPath,
        agentKind: first.agentKind,
        agentName: first.agentName
      }))
    })
  } else {
    groupedBy(scoped, (session) => `${session.sourceKey}:${session.model}`).forEach((group) => {
      const first = group[0]
      buckets.push(bucketFor(group, {
        sourceId: first.sourceId,
        sourceKey: first.sourceKey,
        sourceLabel: first.sourceLabel,
        sourceRootPath: first.sourceRootPath,
        sourceSessionsPath: first.sourceSessionsPath,
        agentKind: first.agentKind,
        agentName: first.agentName,
        model: first.model
      }))
    })
  }
  return { groupBy: filters.groupBy, buckets: buckets.sort((left, right) => right.totalTokens - left.totalTokens) }
}

function bucketFor(group: Session[], fields: Partial<UsageBreakdownBucket>): UsageBreakdownBucket {
  const inputTokens = sum(group, (session) => session.tokenUsage.inputTokens)
  const cachedInputTokens = sum(group, (session) => session.tokenUsage.cachedInputTokens)
  return {
    ...fields,
    sessionCount: group.length,
    totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
    inputTokens,
    cachedInputTokens,
    outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
    reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
    cacheUtilizationRate: inputTokens > 0 ? cachedInputTokens / inputTokens : 0,
    estimatedCostUsd: costSum(group),
    unpriced: group.some((session) => session.unpriced)
  }
}

function sessionDetail(id: number): SessionDetail {
  const session = sessions.find((item) => item.id === id) || sessions[0]
  const calls = toolCalls.filter((call) => call.sessionId === session.id)
  const modelCall: ModelCall = {
    id: session.id + 7000,
    startedAt: session.startedAt,
    endedAt: new Date(Date.parse(session.startedAt) + session.modelDurationMs).toISOString(),
    durationMs: session.modelDurationMs,
    model: session.model,
    provider: session.modelProvider,
    status: 'success',
    inputTokens: session.tokenUsage.inputTokens,
    cachedInputTokens: session.tokenUsage.cachedInputTokens,
    outputTokens: session.tokenUsage.outputTokens,
    reasoningOutputTokens: session.tokenUsage.reasoningOutputTokens,
    totalTokens: session.tokenUsage.totalTokens,
    costUsd: session.estimatedCostUsd,
    unpriced: session.unpriced
  }
  const events: EventItem[] = [
    {
      id: session.id * 10 + 1,
      sourceLine: 1,
      timestamp: session.startedAt,
      kind: 'session',
      rawType: 'session_start',
      summary: `Started ${session.agentName} session for ${session.projectPath}`,
      rawJson: JSON.stringify({ type: 'session_start', sessionKey: session.sessionKey }, null, 2)
    },
    {
      id: session.id * 10 + 2,
      sourceLine: 4,
      timestamp: modelCall.endedAt,
      kind: 'model',
      rawType: 'model_call',
      summary: `${session.model} returned ${session.tokenUsage.outputTokens} output tokens`,
      rawJson: JSON.stringify({ type: 'model_call', model: session.model, usage: session.tokenUsage }, null, 2)
    },
    ...calls.map((call, index) => ({
      id: session.id * 10 + index + 3,
      sourceLine: call.rawEventLine || index + 10,
      timestamp: call.endedAt,
      kind: 'tool',
      rawType: call.rawEndEventType || 'tool_result',
      summary: `${call.toolName}: ${call.outputSummary || call.error}`,
      rawJson: call.rawEndEventJson
    })),
    {
      id: session.id * 10 + 9,
      sourceLine: session.eventCount,
      timestamp: session.endedAt,
      kind: 'session',
      rawType: 'session_end',
      summary: `Finished after ${Math.round(session.wallDurationMs / 60000)} minutes`,
      rawJson: JSON.stringify({ type: 'session_end', durationMs: session.wallDurationMs }, null, 2)
    }
  ]
  return { session, events, modelCalls: [modelCall], toolCalls: calls }
}

function settings(sourceEntries: SourceEntry[] = sources.map((item) => ({ path: item.sourceRootPath, enabled: true, label: item.sourceLabel }))): Settings {
  const result = indexResult(false)
  const enabledEntries = sourceEntries.filter((entry) => entry.enabled)
  return {
    sourcePath: enabledEntries[0]?.path || '',
    sourcePaths: enabledEntries.map((entry) => entry.path),
    sourceEntries,
    defaultSourcePath: sources[0].sourceRootPath,
    defaultSourcePaths: sources.map((item) => item.sourceRootPath),
    databasePath: 'C:\\Users\\demo\\AppData\\Local\\AgentMeter\\agentmeter-demo.db',
    pricingModels,
    lastIndexStartedAt: '2026-06-28T02:00:00Z',
    lastIndexResult: result
  }
}

function indexResult(rebuild: boolean): IndexResult {
  return {
    sourcePath: sources[0].sourceRootPath,
    sourcePaths: sources.map((item) => item.sourceRootPath),
    database: 'C:\\Users\\demo\\AppData\\Local\\AgentMeter\\agentmeter-demo.db',
    filesSeen: 18,
    indexed: sessions.length,
    skipped: 2,
    failed: 0,
    sessions: sessions.length,
    warnings: ['Static demo mode is read-only. Index requests are simulated and no files are scanned.'],
    durationMs: rebuild ? 1420 : 460,
    rebuild
  }
}

function auditSummary(filters: Pick<AuditFindingFilters, 'agent'> = {}): AuditSummary {
  const findings = auditFindings.filter((finding) => matchesAgent(finding, filters.agent))
  return {
    totalFindings: findings.length,
    criticalFindings: findings.filter((finding) => finding.severity === 'critical').length,
    highFindings: findings.filter((finding) => finding.severity === 'high').length,
    mediumFindings: findings.filter((finding) => finding.severity === 'medium').length,
    lowFindings: findings.filter((finding) => finding.severity === 'low').length,
    commandFindings: findings.filter((finding) => finding.category === 'command').length,
    privacyFindings: findings.filter((finding) => finding.category === 'privacy').length,
    egressFindings: findings.filter((finding) => finding.category === 'egress').length,
    fileFindings: findings.filter((finding) => finding.category === 'file').length,
    sessionsWithFindings: new Set(findings.map((finding) => finding.sessionId)).size,
    recentFindings: findings.slice(0, 5)
  }
}

function filteredFindings(filters: AuditFindingFilters = {}): AuditFinding[] {
  const search = (filters.search || '').trim().toLowerCase()
  return auditFindings
    .filter((finding) => matchesAgent(finding, filters.agent))
    .filter((finding) => !filters.category || finding.category === filters.category)
    .filter((finding) => !filters.severity || finding.severity === filters.severity)
    .filter((finding) => !filters.shell || finding.shellFamily === filters.shell)
    .filter((finding) => {
      if (!search) return true
      return [finding.title, finding.description, finding.evidence, finding.command, finding.ruleId, finding.projectPath]
        .some((value) => (value || '').toLowerCase().includes(search))
    })
    .sort((left, right) => Date.parse(right.timestamp) - Date.parse(left.timestamp))
}

export const demoApi: DemoApi = {
  getSettings: async () => clone(settings()),
  saveSourceSettings: async (sourceEntries) => clone(settings(sourceEntries)),
  getAgentPrivacy: async (target) => clone(privacyStatus(target)),
  applyAgentPrivacyChanges: async (target) => ({
    status: clone(privacyStatus(target)),
    changed: [],
    warnings: [
      'Static demo mode is read-only. Privacy changes were accepted for preview but not persisted.'
    ]
  }),
  applyAgentPrivacyProfile: async (target) => ({
    status: clone(privacyStatus(target)),
    changed: [],
    warnings: [
      'Static demo mode is read-only. Privacy profile changes were accepted for preview but not persisted.'
    ]
  }),
  indexNow: async (rebuild = false) => clone(indexResult(rebuild)),
  getOverview: async (filters = {}) => clone(overview(filters)),
  getTokenAnalytics: async (filters = {}) => {
    const scoped = filteredSessions(filters)
    const inputTokens = sum(scoped, (session) => session.tokenUsage.inputTokens)
    const cachedInputTokens = sum(scoped, (session) => session.tokenUsage.cachedInputTokens)
    return clone({
      totalSessions: scoped.length,
      totalInputTokens: inputTokens,
      totalCachedInputTokens: cachedInputTokens,
      totalOutputTokens: sum(scoped, (session) => session.tokenUsage.outputTokens),
      totalReasoningTokens: sum(scoped, (session) => session.tokenUsage.reasoningOutputTokens),
      totalTokens: sum(scoped, (session) => session.tokenUsage.totalTokens),
      cacheUtilizationRate: inputTokens > 0 ? cachedInputTokens / inputTokens : 0,
      estimatedCostUsd: costSum(scoped),
      unpricedCount: scoped.filter((session) => session.unpriced).length,
      cacheHitTrend: cacheHitTrendFor(scoped),
      modelUsage: modelUsageFor(scoped),
      agentUsage: agentUsageFor(scoped),
      recentSessions: scoped.slice(0, 5),
      highTokenSessions: [...scoped].sort((left, right) => right.tokenUsage.totalTokens - left.tokenUsage.totalTokens).slice(0, 5)
    } satisfies TokenAnalytics)
  },
  getModelSignals: async (filters = {}) => clone(modelSignals(filters)),
  getUsageBreakdown: async (filters) => clone(breakdown(filters)),
  listSessions: async (filters = {}) => clone(paginate(filteredSessions(filters), filters.limit, filters.offset)),
  getSessionDetail: async (id) => clone(sessionDetail(id)),
  getTools: async (filters = {}) => clone(toolStatsFor(filteredToolCalls({ agent: filters.agent }))),
  listToolCalls: async (filters = {}) => clone(paginate(filteredToolCalls(filters), filters.limit, filters.offset)),
  getAuditSummary: async (filters = {}) => clone(auditSummary(filters)),
  listAuditFindings: async (filters = {}) => clone(paginate(filteredFindings(filters), filters.limit, filters.offset)),
  getAuditFinding: async (id) => {
    const finding = auditFindings.find((item) => item.id === id)
    if (!finding) throw new Error('Demo audit finding not found')
    return clone(finding)
  },
  getPricingModels: async () => clone(pricingModels)
}
