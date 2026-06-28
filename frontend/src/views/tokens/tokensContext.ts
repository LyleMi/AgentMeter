import { inject, type ComputedRef, type Ref } from 'vue'
import type {
  Overview,
  TokenAnalytics,
  UsageBreakdownBucket,
  UsageBreakdownGroupBy
} from '../../api'
import type { SourceFilterOption } from '../../presentation/sourceIdentity'
import type { UsageScopeForm } from '../useUsageScope'
import type { UsageScopeOption } from '../useUsageScopeOptions'

export const DEFAULT_BREAKDOWN_GROUP = 'global'
export type TokenBreakdownGroup = typeof DEFAULT_BREAKDOWN_GROUP | UsageBreakdownGroupBy

export interface TokensContext {
  analytics: ComputedRef<TokenAnalytics | null>
  optionOverview: Ref<Overview | null>
  loading: Ref<boolean>
  error: Ref<string>
  breakdownRows: Ref<UsageBreakdownBucket[]>
  breakdownGroup: Ref<TokenBreakdownGroup>
  agentOptions: ComputedRef<SourceFilterOption[]>
  modelOptions: ComputedRef<UsageScopeOption[]>
  projectOptions: ComputedRef<UsageScopeOption[]>
  load: () => Promise<TokenAnalytics | null | undefined>
  updateScopeFilters: (filters: UsageScopeForm) => Promise<void>
  clearScopeFilters: () => Promise<void>
  updateBreakdownGroup: (value: unknown) => Promise<void>
}

export const tokensContextKey = Symbol('tokensContext')

export function useTokensContext() {
  const context = inject<TokensContext>(tokensContextKey)
  if (!context) {
    throw new Error('Tokens context is not available')
  }
  return context
}
