import { sourceFilterOptions, type SourceFilterOption, type SourceIdentityLike } from '../presentation/sourceIdentity'
import { projectDisplay } from '../presentation/formatters'

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

interface ProjectLike {
  projectPath?: string | null
  rawSourcePath?: string | null
}

interface UsageProjectOptionsInput {
  projects?: Array<readonly ProjectLike[] | null | undefined>
  selected?: string
  fallback: string
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

export function buildUsageProjectOptions(input: UsageProjectOptionsInput): UsageScopeOption[] {
  const optionsByValue = new Map<string, UsageScopeOption>()
  for (const item of flattenNullable(input.projects || [])) {
    const value = (item.projectPath || item.rawSourcePath || '').trim()
    const key = projectOptionKey(value)
    if (!value || !key || optionsByValue.has(key)) continue
    const display = projectDisplay(value)
    optionsByValue.set(key, {
      value,
      label: display.main || input.fallback,
      title: display.full
    })
  }

  const options = [...optionsByValue.values()].sort((left, right) => left.label.localeCompare(right.label))
  const selectedKey = projectOptionKey(input.selected || '')
  if (input.selected && !options.some((item) => projectOptionKey(item.value) === selectedKey)) {
    const display = projectDisplay(input.selected)
    options.unshift({
      value: input.selected,
      label: display.main || input.fallback,
      title: display.full
    })
  }
  return options
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

function projectOptionKey(value: string) {
  const normalized = value.trim().replace(/\\/g, '/').replace(/[/.]+$/g, '')
  return /^[a-z]:/i.test(value.trim()) || value.includes('\\') ? normalized.toLowerCase() : normalized
}
