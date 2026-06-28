import { projectDisplay, shortPath, type ToolCall } from '../../api'
import { parseToolInput, type ToolInputField } from '../../toolInput'

const SHELL_TOOL_NAMES = new Set([
  'bash',
  'cmd',
  'cmd.exe',
  'powershell',
  'powershell.exe',
  'pwsh',
  'pwsh.exe',
  'sh',
  'shell',
  'shell_command',
  'terminal',
  'zsh'
])
const COMMAND_FIELD_KEYS = new Set(['arguments', 'cmd', 'command', 'input', 'script'])

export function isShellToolName(toolName?: string) {
  const normalized = normalizeToolName(toolName)
  if (!normalized) return false
  if (SHELL_TOOL_NAMES.has(normalized)) return true
  if (normalized.endsWith('.shell_command') || normalized.includes('shell_command')) return true
  const tokens = normalized.split(/[^a-z0-9]+/).filter(Boolean)
  return tokens.some((token) => SHELL_TOOL_NAMES.has(token))
}

export function commandSummary(call: ToolCall) {
  const parsed = parseToolInput(call)
  const field = parsed.fields.find((item) => isCommandField(item))
  return (field?.value || parsed.rawText || call.inputSummary || '').trim()
}

export function commandTooltip(call: ToolCall, fallback: string) {
  const parsed = parseToolInput(call)
  return parsed.tooltip || parsed.rawText || call.inputSummary || fallback
}

export function inputContext(call: ToolCall) {
  const parsed = parseToolInput(call)
  if (!parsed.fields.length) return ''
  return parsed.fields
    .filter((field) => !isCommandField(field))
    .slice(0, 4)
    .map((field) => `${field.key}: ${field.preview || field.value}`)
    .join('  ')
}

export function projectContext(call: ToolCall, fallback: string) {
  if (call.projectPath) return projectDisplay(call.projectPath).main
  return call.rawSourcePath ? shortPath(call.rawSourcePath) : fallback
}

export function projectTooltip(call: ToolCall, fallback: string) {
  return [call.projectPath, call.rawSourcePath].filter(Boolean).join('\n') || fallback
}

export function rawSourceContext(call: ToolCall, rawLabel: string) {
  if (!call.rawSourcePath || call.rawSourcePath === call.projectPath) return ''
  return `${rawLabel}: ${shortPath(call.rawSourcePath)}`
}

function normalizeToolName(toolName?: string) {
  return (toolName || '').trim().toLowerCase()
}

function isCommandField(field: ToolInputField) {
  const compactKey = field.key.replace(/[-_]/g, '').toLowerCase()
  return COMMAND_FIELD_KEYS.has(compactKey) || COMMAND_FIELD_KEYS.has(field.key.toLowerCase())
}
