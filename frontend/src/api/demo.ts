import type {
  AgentTimeUsage,
  AgentUsage,
  AuditFinding,
  AuditFindingFilters,
  AuditSummary,
  DailyUsage,
  EventItem,
  IndexResult,
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

function filteredSessions(filters: UsageScopeFilters & SessionFilters = {}): Session[] {
  const search = (filters.search || '').trim().toLowerCase()
  return sessions
    .filter((session) => matchesAgent(session, filters.agent))
    .filter((session) => !filters.model || session.model === filters.model)
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

function filteredToolCalls(filters: ToolCallFilters = {}): ToolCall[] {
  return toolCalls
    .filter((call) => matchesAgent(call, filters.agent))
    .filter((call) => !filters.tool || call.toolName === filters.tool)
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
  const scopedToolCalls = filteredToolCalls({ agent: filters.agent, from: filters.from, to: filters.to })
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
      modelUsage: modelUsageFor(scoped),
      agentUsage: agentUsageFor(scoped),
      recentSessions: scoped.slice(0, 5),
      highTokenSessions: [...scoped].sort((left, right) => right.tokenUsage.totalTokens - left.tokenUsage.totalTokens).slice(0, 5)
    } satisfies TokenAnalytics)
  },
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
