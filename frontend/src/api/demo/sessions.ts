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
  const contextCompressionTokens = agent.agentKind === 'codex' && inputTokens >= 100_000 ? Math.round(inputTokens * 0.025) : 0
  const totalTokens = inputTokens + outputTokens + contextCompressionTokens
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
      contextCompressionTokens,
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

export const sessions: Session[] = [
  makeSession(101, 0, '2026-06-28T01:12:00Z', 34, 'gpt-5-codex', 'D:\\work\\checkout\\agentmeter', 128400, 38400, 22100, 6200, 16, 74),
  makeSession(102, 1, '2026-06-27T18:44:00Z', 22, 'gemini-2.5-pro', 'D:\\work\\demo\\pricing-audit', 73200, 14600, 18400, 3100, 11, 49),
  makeSession(103, 0, '2026-06-27T10:05:00Z', 47, 'gpt-5-codex', 'D:\\work\\checkout\\docs-site', 201500, 96100, 35600, 12200, 23, 103),
  makeSession(104, 2, '2026-06-26T15:30:00Z', 18, 'claude-sonnet-4', 'D:\\work\\client\\privacy-review', 52200, 8700, 10100, 0, 8, 38),
  makeSession(105, 1, '2026-06-25T22:18:00Z', 61, 'gemini-2.5-pro', 'D:\\work\\research\\tool-latency', 176300, 44100, 28700, 5200, 31, 128),
  makeSession(106, 0, '2026-06-24T07:42:00Z', 14, 'experimental-local-model', 'D:\\work\\scratch\\offline-index', 31800, 0, 6400, 0, 5, 26)
]

export const toolCalls: ToolCall[] = [
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
