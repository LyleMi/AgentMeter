import type { AgentUsage, CacheHitTrendPoint, DailyUsage, ModelUsage, Session } from '../types'
import { pricingModels } from './pricing'
import { groupedBy, sum } from './utils'

export function modelUsageFor(items: Session[]): ModelUsage[] {
  return [...groupedBy(items, (session) => session.model)].map(([model, group]) => ({
    model,
    sessionCount: group.length,
    totalTokens: sum(group, (session) => session.tokenUsage.totalTokens),
    inputTokens: sum(group, (session) => session.tokenUsage.inputTokens),
    cachedInputTokens: sum(group, (session) => session.tokenUsage.cachedInputTokens),
    outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
    reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
    contextCompressionTokens: sum(group, (session) => session.tokenUsage.contextCompressionTokens || 0),
    estimatedCostUsd: costSum(group),
    unpriced: group.some((session) => session.unpriced)
  })).sort((left, right) => right.totalTokens - left.totalTokens)
}

export function agentUsageFor(items: Session[]): AgentUsage[] {
  return [...groupedBy(items, (session) => session.sourceKey || session.agentKind)].map(([, group]) => {
    const first = group[0]
    const inputTokens = sum(group, (session) => session.tokenUsage.inputTokens)
    const cachedInputTokens = sum(group, (session) => session.tokenUsage.cachedInputTokens)
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
      inputTokens,
      cachedInputTokens,
      outputTokens: sum(group, (session) => session.tokenUsage.outputTokens),
      reasoningOutputTokens: sum(group, (session) => session.tokenUsage.reasoningOutputTokens),
      contextCompressionTokens: sum(group, (session) => session.tokenUsage.contextCompressionTokens || 0),
      cacheUtilizationRate: inputTokens > 0 ? cachedInputTokens / inputTokens : 0,
      toolCalls: sum(group, (session) => session.toolCallCount),
      estimatedCostUsd: costSum(group),
      unpriced: group.some((session) => session.unpriced)
    }
  }).sort((left, right) => right.totalTokens - left.totalTokens)
}

export function costSum(items: Session[]): number | undefined {
  const priced = items.filter((session) => !session.unpriced)
  if (priced.length !== items.length) return undefined
  return Number(priced.reduce((total, session) => total + (session.estimatedCostUsd || 0), 0).toFixed(4))
}

export function cacheSavingsUsdFor(items: Session[]): number | undefined {
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

export function dailyUsageFor(items: Session[]): DailyUsage[] {
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
      contextCompressionTokens: sum(group, (session) => session.tokenUsage.contextCompressionTokens || 0),
      cacheUtilizationRate: inputTokens > 0 ? cachedInputTokens / inputTokens : 0,
      toolCalls: sum(group, (session) => session.toolCallCount),
      estimatedCostUsd: costSum(group)
    }
  }).sort((left, right) => left.date.localeCompare(right.date))
}

export function cacheHitTrendFor(items: Session[]): CacheHitTrendPoint[] {
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
