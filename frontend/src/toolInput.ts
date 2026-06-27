import type { ToolCall } from './api'

export interface ToolInputField {
  key: string
  label: string
  value: string
  preview: string
  isLong: boolean
}

export interface ParsedToolInput {
  hasInput: boolean
  isStructured: boolean
  fields: ToolInputField[]
  preview: string
  tooltip: string
  rawText: string
}

const GENERIC_FIELD_PRIORITY = [
  'command',
  'cmd',
  'script',
  'query',
  'prompt',
  'path',
  'file_path',
  'filePath',
  'workdir',
  'cwd',
  'url',
  'ref_id',
  'refId',
  'id',
  'timeout_ms',
  'timeoutMs',
  'sandbox_permissions',
  'sandboxPermissions'
]

const TOOL_FIELD_PRIORITY: Record<string, string[]> = {
  shell_command: ['command', 'workdir', 'timeout_ms', 'sandbox_permissions', 'login'],
  'functions.shell_command': ['command', 'workdir', 'timeout_ms', 'sandbox_permissions', 'login'],
  tool_search_tool: ['query', 'limit'],
  imagegen: ['prompt'],
  imagegen__imagegen: ['prompt'],
  apply_patch: ['patch'],
  web_search: ['query', 'search_query', 'open'],
  tool_search: ['query', 'limit']
}

const EMPTY_INPUT: ParsedToolInput = {
  hasInput: false,
  isStructured: false,
  fields: [],
  preview: '',
  tooltip: '',
  rawText: ''
}

export function parseToolInput(call: ToolCall | null | undefined): ParsedToolInput {
  if (!call) return EMPTY_INPUT

  const candidate = normalizeJsonish(extractRawInput(call) ?? call.inputSummary)
  if (candidate === undefined || candidate === null || candidate === '') return EMPTY_INPUT

  if (isPlainRecord(candidate)) {
    const fields = fieldsFromRecord(candidate, call.toolName)
    if (fields.length > 0) {
      return {
        hasInput: true,
        isStructured: true,
        fields,
        preview: compactFieldPreview(fields),
        tooltip: tooltipForFields(fields),
        rawText: stringifyValue(candidate, true)
      }
    }
  }

  if (Array.isArray(candidate)) {
    const rawText = stringifyValue(candidate, true)
    return {
      hasInput: rawText.length > 0,
      isStructured: false,
      fields: [],
      preview: compactText(rawText, 180),
      tooltip: rawText,
      rawText
    }
  }

  const rawText = stringifyValue(candidate, false)
  return {
    hasInput: rawText.length > 0,
    isStructured: false,
    fields: [],
    preview: compactText(rawText, 180),
    tooltip: rawText,
    rawText
  }
}

function extractRawInput(call: ToolCall): unknown {
  const raw = parseJsonObject(call.rawStartEventJson)
  if (!raw) return undefined

  const payload = recordValue(raw.payload)
  const providerData = recordValue(raw.providerData)
  const message = recordValue(raw.message)

  const payloadInput = firstInputValue(payload)
  if (payloadInput !== undefined) return payloadInput

  const topLevelInput = firstInputValue(raw)
  if (topLevelInput !== undefined) return topLevelInput

  const providerInput = firstInputValue(providerData)
  if (providerInput !== undefined) return providerInput

  const messageInput = inputFromMessage(message, call.callId)
  if (messageInput !== undefined) return messageInput

  return undefined
}

function firstInputValue(record: Record<string, unknown> | undefined): unknown {
  if (!record) return undefined
  for (const key of ['arguments', 'input', 'query', 'action']) {
    if (record[key] !== undefined && record[key] !== null && record[key] !== '') {
      return record[key]
    }
  }
  return undefined
}

function inputFromMessage(message: Record<string, unknown> | undefined, callId?: string): unknown {
  if (!message) return undefined
  const content = message.content
  const items = Array.isArray(content) ? content : [content]
  for (const item of items) {
    const record = recordValue(item)
    if (!record || record.type !== 'tool_use') continue
    const itemID = String(record.id || record.call_id || record.tool_use_id || '')
    if (callId && itemID && itemID !== callId) continue
    if (record.input !== undefined) return record.input
  }
  return undefined
}

function fieldsFromRecord(record: Record<string, unknown>, toolName: string): ToolInputField[] {
  const orderedKeys = orderKeys(Object.keys(record), toolName)
  return orderedKeys
    .filter((key) => record[key] !== undefined)
    .map((key) => {
      const value = stringifyValue(normalizeJsonish(record[key]), true)
      return {
        key,
        label: labelForKey(key),
        value,
        preview: compactText(value, key === 'command' || key === 'query' || key === 'prompt' ? 160 : 110),
        isLong: value.includes('\n') || value.length > 120
      }
    })
}

function orderKeys(keys: string[], toolName: string): string[] {
  const normalizedTool = (toolName || '').trim()
  const priority = [...(TOOL_FIELD_PRIORITY[normalizedTool] || []), ...GENERIC_FIELD_PRIORITY]
  return [...keys].sort((left, right) => {
    const leftRank = rankKey(left, priority)
    const rightRank = rankKey(right, priority)
    if (leftRank !== rightRank) return leftRank - rightRank
    return left.localeCompare(right)
  })
}

function rankKey(key: string, priority: string[]): number {
  const exact = priority.indexOf(key)
  if (exact >= 0) return exact
  const lowerKey = key.toLowerCase()
  const lower = priority.findIndex((item) => item.toLowerCase() === lowerKey)
  if (lower >= 0) return lower
  return priority.length + 1
}

function labelForKey(key: string): string {
  return key
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[_-]+/g, ' ')
    .replace(/\b\w/g, (match) => match.toUpperCase())
}

function compactFieldPreview(fields: ToolInputField[]): string {
  return fields
    .slice(0, 3)
    .map((field) => `${field.key}: ${field.preview}`)
    .join('  ')
}

function tooltipForFields(fields: ToolInputField[]): string {
  return fields.map((field) => `${field.label}: ${field.value}`).join('\n')
}

function normalizeJsonish(value: unknown): unknown {
  if (typeof value !== 'string') return value
  const trimmed = value.trim()
  if (!trimmed) return ''
  if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) return trimmed
  try {
    return JSON.parse(trimmed)
  } catch {
    return trimmed
  }
}

function parseJsonObject(value?: string): Record<string, unknown> | undefined {
  if (!value) return undefined
  try {
    const parsed: unknown = JSON.parse(value)
    return recordValue(parsed)
  } catch {
    return undefined
  }
}

function recordValue(value: unknown): Record<string, unknown> | undefined {
  return isPlainRecord(value) ? value : undefined
}

function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function stringifyValue(value: unknown, pretty: boolean): string {
  if (value === undefined || value === null) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  try {
    return JSON.stringify(value, null, pretty ? 2 : 0)
  } catch {
    return String(value)
  }
}

function compactText(value: string, limit: number): string {
  const compact = value.replace(/\s+/g, ' ').trim()
  if (compact.length <= limit) return compact
  return `${compact.slice(0, Math.max(0, limit - 1))}...`
}
