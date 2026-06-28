import { inject, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import type {
  AgentTimeUsage,
  ModelTimeUsage,
  Overview,
  Session,
  ToolTimeUsage
} from '../../api'
import type { SourceFilterOption } from '../../presentation/sourceIdentity'
import type { UsageScopeForm } from '../useUsageScope'
import type { UsageScopeOption } from '../useUsageScopeOptions'

export interface TimeSegment {
  key: string
  label: string
  value: number
  share: number
  width: string
  tone: string
}

export interface TimeKpiCard {
  label: string
  value: string
  note: string
  icon: unknown
}

export interface TimeContext {
  overview: ComputedRef<Overview | null>
  optionOverview: Ref<Overview | null>
  loading: Ref<boolean>
  error: Ref<string>
  hasIndexedData: ComputedRef<boolean>
  wallDurationMs: ComputedRef<number>
  activeDurationMs: ComputedRef<number>
  toolDurationMs: ComputedRef<number>
  suspectedNetworkDurationMs: ComputedRef<number>
  slowSessions: ComputedRef<Session[]>
  compositionSegments: ComputedRef<TimeSegment[]>
  kpiCards: ComputedRef<TimeKpiCard[]>
  rankedToolLeaders: ComputedRef<ToolTimeUsage[]>
  rankedAgentTimeUsage: ComputedRef<AgentTimeUsage[]>
  rankedModelTimeUsage: ComputedRef<ModelTimeUsage[]>
  agentOptions: ComputedRef<SourceFilterOption[]>
  modelOptions: ComputedRef<UsageScopeOption[]>
  projectOptions: ComputedRef<UsageScopeOption[]>
  formatPercent: (value: number) => string
  load: () => Promise<Overview | null | undefined>
  updateScopeFilters: (filters: UsageScopeForm) => Promise<void>
  clearScopeFilters: () => Promise<void>
}

export const timeContextKey: InjectionKey<TimeContext> = Symbol('timeContext')

export function useTimeContext() {
  const context = inject(timeContextKey)
  if (!context) {
    throw new Error('Time context is not available')
  }
  return context
}
