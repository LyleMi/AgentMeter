import type {
  AgentTimeUsage,
  ModelTimeUsage,
  Overview,
  Session,
  TokenAnalytics,
  ToolTimeUsage,
  UsageBreakdown,
  UsageBreakdownBucket,
  UsageBreakdownFilters,
  UsageScopeFilters
} from '../types'
import { filteredSessions, filteredToolCalls, toolStatsFor } from './sessions'
import {
  agentUsageFor,
  cacheHitTrendFor,
  costSum,
  dailyUsageFor,
  modelUsageFor
} from './usageMetrics'
import { groupedBy, matchesAgent, projectPathKey, sum } from './utils'

export function overview(filters: UsageScopeFilters = {}): Overview {
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
    totalContextCompressionTokens: sum(scoped, (session) => session.tokenUsage.contextCompressionTokens || 0),
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

export function breakdown(filters: UsageBreakdownFilters): UsageBreakdown {
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
    contextCompressionTokens: sum(group, (session) => session.tokenUsage.contextCompressionTokens || 0),
    cacheUtilizationRate: inputTokens > 0 ? cachedInputTokens / inputTokens : 0,
    estimatedCostUsd: costSum(group),
    unpriced: group.some((session) => session.unpriced)
  }
}

export function tokenAnalytics(filters: UsageScopeFilters = {}): TokenAnalytics {
  const scoped = filteredSessions(filters)
  const inputTokens = sum(scoped, (session) => session.tokenUsage.inputTokens)
  const cachedInputTokens = sum(scoped, (session) => session.tokenUsage.cachedInputTokens)
  return {
    totalSessions: scoped.length,
    totalInputTokens: inputTokens,
    totalCachedInputTokens: cachedInputTokens,
    totalOutputTokens: sum(scoped, (session) => session.tokenUsage.outputTokens),
    totalReasoningTokens: sum(scoped, (session) => session.tokenUsage.reasoningOutputTokens),
    totalContextCompressionTokens: sum(scoped, (session) => session.tokenUsage.contextCompressionTokens || 0),
    totalTokens: sum(scoped, (session) => session.tokenUsage.totalTokens),
    cacheUtilizationRate: inputTokens > 0 ? cachedInputTokens / inputTokens : 0,
    estimatedCostUsd: costSum(scoped),
    unpricedCount: scoped.filter((session) => session.unpriced).length,
    cacheHitTrend: cacheHitTrendFor(scoped),
    modelUsage: modelUsageFor(scoped),
    agentUsage: agentUsageFor(scoped),
    recentSessions: scoped.slice(0, 5),
    highTokenSessions: [...scoped].sort((left, right) => right.tokenUsage.totalTokens - left.tokenUsage.totalTokens).slice(0, 5)
  }
}
