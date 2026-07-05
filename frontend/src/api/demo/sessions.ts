import type {
  EventItem,
  ModelCall,
  Session,
  SessionDetail,
  SessionFilters,
  ToolCall,
  ToolCallFilters,
  ToolStat,
  UsageScopeFilters
} from '../types'
import { costUsd } from './pricing'
import { source } from './sources'
import { groupedBy, matchesAgent, matchesDateRange, matchesProject } from './utils'

type DemoSessionSpec = {
  id: number
  sourceIndex: number
  startedAt: string
  durationMinutes: number
  model: string
  projectPath: string
  inputTokens: number
  cachedInputTokens: number
  outputTokens: number
  reasoningOutputTokens: number
  toolCallCount: number
  eventCount: number
}

function makeSession(spec: DemoSessionSpec): Session {
  const agent = source(spec.sourceIndex)
  const startedMs = Date.parse(spec.startedAt)
  const wallDurationMs = spec.durationMinutes * 60 * 1000
  const modelDurationMs = Math.round(wallDurationMs * 0.46)
  const toolDurationMs = Math.round(wallDurationMs * 0.28)
  const idleDurationMs = Math.max(0, wallDurationMs - modelDurationMs - toolDurationMs)
  const contextCompressionTokens = agent.agentKind === 'codex' && spec.inputTokens >= 100_000 ? Math.round(spec.inputTokens * 0.025) : 0
  const totalTokens = spec.inputTokens + spec.outputTokens + contextCompressionTokens
  const estimatedCostUsd = costUsd(spec.model, spec.inputTokens, spec.cachedInputTokens, spec.outputTokens)
  return {
    ...agent,
    id: spec.id,
    sessionKey: `demo-session-${String(spec.id).padStart(3, '0')}`,
    codexSessionId: agent.agentKind === 'codex' ? `codex-demo-${spec.id}` : undefined,
    projectPath: spec.projectPath,
    model: spec.model,
    modelProvider: agent.agentKind === 'gemini' ? 'google' : agent.agentKind === 'claude' ? 'anthropic' : 'openai',
    originator: 'demo',
    threadSource: spec.id % 2 === 0 ? 'resume' : 'new',
    startedAt: spec.startedAt,
    endedAt: new Date(startedMs + wallDurationMs).toISOString(),
    wallDurationMs,
    activeDurationMs: modelDurationMs + toolDurationMs,
    modelDurationMs,
    toolDurationMs,
    idleDurationMs,
    eventCount: spec.eventCount,
    parseStatus: 'ok',
    tokenUsage: {
      model: spec.model,
      inputTokens: spec.inputTokens,
      cachedInputTokens: spec.cachedInputTokens,
      outputTokens: spec.outputTokens,
      reasoningOutputTokens: spec.reasoningOutputTokens,
      contextCompressionTokens,
      totalTokens,
      source: 'demo transcript',
      costUsd: estimatedCostUsd,
      unpriced: estimatedCostUsd === undefined
    },
    estimatedCostUsd,
    unpriced: estimatedCostUsd === undefined,
    toolCallCount: spec.toolCallCount,
    rawSourcePath: `${agent.sourceSessionsPath}\\${String(spec.id).padStart(3, '0')}.jsonl`,
    lastIndexedScanStatus: 'ok',
    lastIndexedScanMessage: 'Demo session indexed from synthetic transcript data'
  }
}

export const sessions: Session[] = [
  makeSession({ id: 101, sourceIndex: 0, startedAt: '2026-06-28T01:12:00Z', durationMinutes: 34, model: 'gpt-5-codex', projectPath: 'D:\\work\\checkout\\agentmeter', inputTokens: 128400, cachedInputTokens: 38400, outputTokens: 22100, reasoningOutputTokens: 6200, toolCallCount: 16, eventCount: 74 }),
  makeSession({ id: 102, sourceIndex: 1, startedAt: '2026-06-27T18:44:00Z', durationMinutes: 22, model: 'gemini-2.5-pro', projectPath: 'D:\\work\\demo\\pricing-audit', inputTokens: 73200, cachedInputTokens: 14600, outputTokens: 18400, reasoningOutputTokens: 3100, toolCallCount: 11, eventCount: 49 }),
  makeSession({ id: 103, sourceIndex: 0, startedAt: '2026-06-27T10:05:00Z', durationMinutes: 47, model: 'gpt-5-codex', projectPath: 'D:\\work\\checkout\\docs-site', inputTokens: 201500, cachedInputTokens: 96100, outputTokens: 35600, reasoningOutputTokens: 12200, toolCallCount: 23, eventCount: 103 }),
  makeSession({ id: 104, sourceIndex: 2, startedAt: '2026-06-26T15:30:00Z', durationMinutes: 18, model: 'claude-sonnet-4', projectPath: 'D:\\work\\client\\privacy-review', inputTokens: 52200, cachedInputTokens: 8700, outputTokens: 10100, reasoningOutputTokens: 0, toolCallCount: 8, eventCount: 38 }),
  makeSession({ id: 105, sourceIndex: 1, startedAt: '2026-06-25T22:18:00Z', durationMinutes: 61, model: 'gemini-2.5-pro', projectPath: 'D:\\work\\research\\tool-latency', inputTokens: 176300, cachedInputTokens: 44100, outputTokens: 28700, reasoningOutputTokens: 5200, toolCallCount: 31, eventCount: 128 }),
  makeSession({ id: 106, sourceIndex: 0, startedAt: '2026-06-24T07:42:00Z', durationMinutes: 14, model: 'experimental-local-model', projectPath: 'D:\\work\\scratch\\offline-index', inputTokens: 31800, cachedInputTokens: 0, outputTokens: 6400, reasoningOutputTokens: 0, toolCallCount: 5, eventCount: 26 })
]

export const toolCalls: ToolCall[] = [
  makeToolCall({ id: 1001, sessionId: 101, startMinute: 0, endMinute: 4, toolName: 'shell_command', status: 'success', inputSummary: 'rg exported API methods', outputSummary: '18 matching lines', error: '' }),
  makeToolCall({ id: 1002, sessionId: 101, startMinute: 8, endMinute: 11, toolName: 'apply_patch', status: 'success', inputSummary: 'add demo API module', outputSummary: 'patch applied', error: '' }),
  makeToolCall({ id: 1003, sessionId: 101, startMinute: 15, endMinute: 28, toolName: 'npm', status: 'success', inputSummary: 'npm run build', outputSummary: 'vite build completed', error: '' }),
  makeToolCall({ id: 1004, sessionId: 102, startMinute: 3, endMinute: 8, toolName: 'read_file', status: 'success', inputSummary: 'open pricing table', outputSummary: '3 models parsed', error: '' }),
  makeToolCall({ id: 1005, sessionId: 102, startMinute: 9, endMinute: 17, toolName: 'web_fetch', status: 'failed', inputSummary: 'fetch vendor pricing page', outputSummary: '', error: 'network disabled by demo policy' }),
  makeToolCall({ id: 1006, sessionId: 103, startMinute: 5, endMinute: 13, toolName: 'shell_command', status: 'success', inputSummary: 'list docs routes', outputSummary: '7 markdown files', error: '' }),
  makeToolCall({ id: 1007, sessionId: 103, startMinute: 19, endMinute: 41, toolName: 'apply_patch', status: 'success', inputSummary: 'rewrite validation docs', outputSummary: '2 files changed', error: '' }),
  makeToolCall({ id: 1008, sessionId: 104, startMinute: 4, endMinute: 9, toolName: 'read_file', status: 'success', inputSummary: 'inspect privacy config', outputSummary: 'config loaded', error: '' }),
  makeToolCall({ id: 1009, sessionId: 105, startMinute: 2, endMinute: 21, toolName: 'shell_command', status: 'success', inputSummary: 'run smoke-api script', outputSummary: 'all read-only checks passed', error: '' }),
  makeToolCall({ id: 1010, sessionId: 105, startMinute: 30, endMinute: 54, toolName: 'browser_screenshot', status: 'success', inputSummary: 'capture tools chart', outputSummary: 'screenshot stored', error: '' }),
  makeToolCall({ id: 1011, sessionId: 106, startMinute: 3, endMinute: 7, toolName: 'shell_command', status: 'success', inputSummary: 'scan local jsonl files', outputSummary: '5 files discovered', error: '' })
]

sessions.forEach((session) => {
  session.toolCallCount = toolCalls.filter((call) => call.sessionId === session.id).length
})

type DemoToolCallSpec = {
  id: number
  sessionId: number
  startMinute: number
  endMinute: number
  toolName: string
  status: string
  inputSummary: string
  outputSummary: string
  error: string
}

function makeToolCall(spec: DemoToolCallSpec): ToolCall {
  const session = sessions.find((item) => item.id === spec.sessionId)
  if (!session) throw new Error(`Missing demo session ${spec.sessionId}`)
  const startedAt = new Date(Date.parse(session.startedAt) + spec.startMinute * 60 * 1000).toISOString()
  const endedAt = new Date(Date.parse(session.startedAt) + spec.endMinute * 60 * 1000).toISOString()
  return {
    id: spec.id,
    sessionId: spec.sessionId,
    startedAt,
    endedAt,
    durationMs: Math.max(1, spec.endMinute - spec.startMinute) * 60 * 1000,
    toolName: spec.toolName,
    status: spec.status,
    inputSummary: spec.inputSummary,
    outputSummary: spec.outputSummary,
    error: spec.error,
    callId: `call-${spec.id}`,
    rawEventId: spec.id + 3000,
    rawStartEventId: spec.id + 2000,
    rawEndEventId: spec.id + 3000,
    rawEventLine: 20 + spec.id - 1000,
    rawStartEventLine: 19 + spec.id - 1000,
    rawEndEventLine: 20 + spec.id - 1000,
    rawStartEventType: 'tool_call',
    rawEndEventType: 'tool_result',
    rawStartEventSummary: spec.inputSummary,
    rawEndEventSummary: spec.outputSummary || spec.error,
    rawStartEventJson: JSON.stringify({ type: 'tool_call', toolName: spec.toolName, inputSummary: spec.inputSummary }, null, 2),
    rawEndEventJson: JSON.stringify({ type: 'tool_result', status: spec.status, outputSummary: spec.outputSummary, error: spec.error }, null, 2),
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

export function filteredSessions(filters: UsageScopeFilters & SessionFilters = {}): Session[] {
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

export function filteredToolCalls(filters: ToolCallFilters & Pick<UsageScopeFilters, 'project'> = {}): ToolCall[] {
  return toolCalls
    .filter((call) => matchesAgent(call, filters.agent))
    .filter((call) => !filters.tool || call.toolName === filters.tool)
    .filter((call) => !filters.shell || isShellToolName(call.toolName))
    .filter((call) => matchesProject(call, filters.project))
    .filter((call) => matchesDateRange(call.startedAt, filters))
    .sort((left, right) => {
      let direction = Date.parse(right.startedAt) - Date.parse(left.startedAt)
      if (filters.sort === 'duration_desc' || filters.sort === 'duration') direction = right.durationMs - left.durationMs
      if (filters.sort === 'duration_asc') direction = left.durationMs - right.durationMs
      return direction || right.id - left.id
    })
}

function isShellToolName(toolName?: string) {
  const normalized = (toolName || '').trim().toLowerCase()
  if (!normalized) return false
  if (normalized.endsWith('.shell_command') || normalized.includes('shell_command')) return true
  return ['bash', 'cmd', 'cmd.exe', 'powershell', 'powershell.exe', 'pwsh', 'pwsh.exe', 'sh', 'shell', 'terminal', 'zsh'].includes(normalized)
}

export function toolStatsFor(calls: ToolCall[]): ToolStat[] {
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

export function sessionDetail(id: number): SessionDetail {
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
    contextCompressionTokens: session.tokenUsage.contextCompressionTokens || 0,
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
