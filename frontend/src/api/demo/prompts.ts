import type {
  PromptExample,
  PromptSuggestion,
  PromptSuggestionFilters,
  PromptVariant,
  SavedPrompt,
  SavedPromptInput
} from '../types'
import { sessions } from './sessions'
import { matchesAgent, matchesProject } from './utils'

interface DemoPromptObservation {
  text: string
  sessionId: number
  timestamp: string
}

const promptObservations: DemoPromptObservation[] = [
  { text: 'Review the current diff, find correctness risks first, then suggest focused fixes.', sessionId: 101, timestamp: '2026-06-28T01:14:00Z' },
  { text: 'Review the current diff and call out correctness risks before style notes.', sessionId: 103, timestamp: '2026-06-27T10:07:00Z' },
  { text: 'Review the current diff, find correctness risks first, then suggest focused fixes.', sessionId: 104, timestamp: '2026-06-26T15:33:00Z' },
  { text: 'Run the relevant tests and fix any failing cases without broad refactors.', sessionId: 101, timestamp: '2026-06-28T01:22:00Z' },
  { text: 'Run relevant tests and fix failing cases without broad refactors.', sessionId: 106, timestamp: '2026-06-24T07:46:00Z' },
  { text: 'Summarize this session into concrete next actions and unresolved risks.', sessionId: 102, timestamp: '2026-06-27T18:46:00Z' },
  { text: 'Summarize this session into concrete next actions and unresolved risks.', sessionId: 105, timestamp: '2026-06-25T22:23:00Z' },
  { text: 'Turn this rough requirement into a v1 implementation plan with backend, frontend, and validation tasks.', sessionId: 101, timestamp: '2026-06-28T01:18:00Z' },
  { text: 'Turn this rough requirement into a v1 implementation plan with backend, frontend and validation tasks.', sessionId: 103, timestamp: '2026-06-27T10:11:00Z' }
]

const ignoredPromptSuggestionKeys = new Set<string>()
const savedPrompts: SavedPrompt[] = [
  {
    id: 1,
    title: 'Review current diff',
    content: 'Review the current diff, find correctness risks first, then suggest focused fixes.',
    sourceSuggestionKey: 'prompt-demo-review-diff',
    copyCount: 4,
    lastCopiedAt: '2026-06-28T02:30:00Z',
    createdAt: '2026-06-27T09:00:00Z',
    updatedAt: '2026-06-28T02:30:00Z'
  }
]

function promptKey(text: string) {
  const normalized = text.trim().toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]+/g, '').slice(0, 48)
  if (text.startsWith('Review the current diff')) return 'prompt-demo-review-diff'
  if (text.startsWith('Run')) return 'prompt-demo-run-tests'
  if (text.startsWith('Summarize')) return 'prompt-demo-session-summary'
  if (text.startsWith('Turn this rough')) return 'prompt-demo-v1-plan'
  return `prompt-demo-${normalized || 'untitled'}`
}

function promptExample(observation: DemoPromptObservation): PromptExample {
  const session = sessions.find((item) => item.id === observation.sessionId) || sessions[0]
  return {
    sessionId: session.id,
    sessionKey: session.sessionKey,
    codexSessionId: session.codexSessionId,
    projectPath: session.projectPath,
    timestamp: observation.timestamp,
    rawSourcePath: session.rawSourcePath,
    agentKind: session.agentKind,
    agentName: session.agentName,
    sourceId: session.sourceId,
    sourceKey: session.sourceKey,
    sourceLabel: session.sourceLabel,
    sourceRootPath: session.sourceRootPath,
    sourceSessionsPath: session.sourceSessionsPath
  }
}

function demoPromptSuggestions(): PromptSuggestion[] {
  const groups = new Map<string, DemoPromptObservation[]>()
  for (const observation of promptObservations) {
    const key = promptKey(observation.text)
    groups.set(key, [...(groups.get(key) || []), observation])
  }

  return [...groups.entries()].map(([key, observations]) => {
    const variantGroups = new Map<string, DemoPromptObservation[]>()
    for (const observation of observations) {
      variantGroups.set(observation.text, [...(variantGroups.get(observation.text) || []), observation])
    }
    const variants: PromptVariant[] = [...variantGroups.entries()]
      .map(([text, group]) => ({
        text,
        count: group.length,
        lastUsedAt: group.map((item) => item.timestamp).sort().at(-1) || ''
      }))
      .sort((left, right) => right.count - left.count || Date.parse(right.lastUsedAt) - Date.parse(left.lastUsedAt))
    const examples = observations.map(promptExample).sort((left, right) => Date.parse(right.timestamp) - Date.parse(left.timestamp))
    return {
      key,
      text: variants[0]?.text || observations[0]?.text || '',
      count: observations.length,
      sessionCount: new Set(observations.map((item) => item.sessionId)).size,
      variantCount: variants.length,
      firstUsedAt: observations.map((item) => item.timestamp).sort()[0] || '',
      lastUsedAt: observations.map((item) => item.timestamp).sort().at(-1) || '',
      matchKind: variants.length > 1 ? 'near' : 'exact',
      confidence: variants.length > 1 ? 0.86 : 1,
      examples: examples.slice(0, 4),
      variants: variants.slice(0, 5)
    }
  })
}

export function filteredPromptSuggestions(filters: PromptSuggestionFilters = {}): PromptSuggestion[] {
  const savedKeys = new Set(savedPrompts.map((item) => item.sourceSuggestionKey).filter(Boolean))
  const search = (filters.search || '').trim().toLowerCase()
  const minCount = Math.max(1, filters.minCount || 2)
  return demoPromptSuggestions()
    .filter((suggestion) => suggestion.count >= minCount)
    .filter((suggestion) => !ignoredPromptSuggestionKeys.has(suggestion.key) && !savedKeys.has(suggestion.key))
    .filter((suggestion) => {
      if (!search) return true
      return [suggestion.text, ...suggestion.variants.map((variant) => variant.text)]
        .some((value) => value.toLowerCase().includes(search))
    })
    .filter((suggestion) => {
      if (!filters.agent) return true
      return suggestion.examples.some((example) => matchesAgent(example, filters.agent))
    })
    .filter((suggestion) => {
      if (!filters.project) return true
      return suggestion.examples.some((example) => matchesProject(example, filters.project))
    })
    .sort((left, right) => right.count - left.count || Date.parse(right.lastUsedAt) - Date.parse(left.lastUsedAt))
    .slice(0, filters.limit || 50)
}

function promptTitleFromContent(content: string) {
  const firstLine = content.trim().split(/\r?\n/)[0] || 'Prompt'
  return firstLine.length > 58 ? `${firstLine.slice(0, 57)}...` : firstLine
}

export function saveDemoPrompt(input: SavedPromptInput): SavedPrompt {
  const content = input.content.trim()
  if (!content) throw new Error('Prompt content is required')
  const now = new Date().toISOString()
  const saved: SavedPrompt = {
    id: Math.max(0, ...savedPrompts.map((item) => item.id)) + 1,
    title: input.title.trim() || promptTitleFromContent(content),
    content,
    sourceSuggestionKey: input.sourceSuggestionKey?.trim(),
    copyCount: 0,
    createdAt: now,
    updatedAt: now
  }
  savedPrompts.unshift(saved)
  return saved
}

export function updateDemoPrompt(id: number, input: SavedPromptInput): SavedPrompt {
  const index = savedPrompts.findIndex((item) => item.id === id)
  if (index < 0) throw new Error('Saved prompt not found')
  const content = input.content.trim()
  if (!content) throw new Error('Prompt content is required')
  const updated: SavedPrompt = {
    ...savedPrompts[index],
    title: input.title.trim() || promptTitleFromContent(content),
    content,
    sourceSuggestionKey: input.sourceSuggestionKey?.trim(),
    updatedAt: new Date().toISOString()
  }
  savedPrompts.splice(index, 1, updated)
  return updated
}

export function recordDemoPromptCopy(id: number): SavedPrompt {
  const index = savedPrompts.findIndex((item) => item.id === id)
  if (index < 0) throw new Error('Saved prompt not found')
  const copied: SavedPrompt = {
    ...savedPrompts[index],
    copyCount: savedPrompts[index].copyCount + 1,
    lastCopiedAt: new Date().toISOString()
  }
  savedPrompts.splice(index, 1, copied)
  return copied
}

export function listSavedPrompts(): SavedPrompt[] {
  return savedPrompts
}

export function deleteDemoPrompt(id: number): { ok: boolean } {
  const index = savedPrompts.findIndex((item) => item.id === id)
  if (index < 0) throw new Error('Saved prompt not found')
  savedPrompts.splice(index, 1)
  return { ok: true }
}

export function ignoreDemoPromptSuggestion(suggestionKey: string): { ok: boolean } {
  ignoredPromptSuggestionKeys.add(suggestionKey)
  return { ok: true }
}

export function unignoreDemoPromptSuggestion(suggestionKey: string): { ok: boolean } {
  ignoredPromptSuggestionKeys.delete(suggestionKey)
  return { ok: true }
}
