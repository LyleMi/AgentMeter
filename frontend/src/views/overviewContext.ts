import { inject, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import type { Overview, Settings } from '../api'

export interface OverviewContext {
  overview: Ref<Overview | null>
  settings: Ref<Settings | null>
  loading: Ref<boolean>
  startupIndexing: Ref<boolean>
  hasIndexedData: ComputedRef<boolean>
  sourcePathDisplay: ComputedRef<string>
  load: () => Promise<void>
  indexFromOverview: () => Promise<void>
}

export const overviewContextKey: InjectionKey<OverviewContext> = Symbol('overviewContext')

export function useOverviewContext() {
  const context = inject(overviewContextKey)
  if (!context) {
    throw new Error('Overview context is unavailable')
  }
  return context
}
