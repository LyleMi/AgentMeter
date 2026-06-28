import { sourceFilterOptions, type SourceFilterOption, type SourceIdentityLike } from '../presentation/sourceIdentity'

export interface UsageScopeOption {
  value: string
  label: string
  title?: string
}

interface UsageAgentOptionsInput {
  sources: Array<readonly SourceIdentityLike[] | null | undefined>
  selected?: string
  fallback: string
}

interface ModelLike {
  model?: string | null
}

interface UsageModelOptionsInput {
  modelUsage?: Array<readonly ModelLike[] | null | undefined>
  sessions?: Array<readonly ModelLike[] | null | undefined>
  selected?: string
}

export function buildUsageAgentOptions(input: UsageAgentOptionsInput): SourceFilterOption[] {
  return ensureSelectedOption(
    sourceFilterOptions(flattenNullable(input.sources), input.fallback, { includeSecondaryInLabel: false }),
    input.selected
  )
}

export function buildUsageModelOptions(input: UsageModelOptionsInput): UsageScopeOption[] {
  const values = new Set<string>()
  collectModelValues(values, input.modelUsage)
  collectModelValues(values, input.sessions)
  return ensureSelectedOption(
    [...values].sort().map((value) => ({ value, label: value, title: value })),
    input.selected
  )
}

export function ensureSelectedOption<T extends UsageScopeOption>(options: T[], selected?: string): T[] {
  if (!selected || options.some((item) => item.value === selected)) return options
  return [{ value: selected, label: selected, title: selected } as T, ...options]
}

function flattenNullable<T>(collections: Array<readonly T[] | null | undefined>): T[] {
  return collections.flatMap((items) => [...(items || [])])
}

function collectModelValues(values: Set<string>, collections?: Array<readonly ModelLike[] | null | undefined>) {
  for (const item of flattenNullable(collections || [])) {
    if (item.model) values.add(item.model)
  }
}
