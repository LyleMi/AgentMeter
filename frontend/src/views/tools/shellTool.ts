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
const COMMAND_SEPARATORS = new Set(['&&', '||', ';', '|'])
const SKIPPED_COMMANDS = new Set([
  '&',
  '.',
  'alias',
  'call',
  'cd',
  'chdir',
  'echo',
  'export',
  'false',
  'popd',
  'pushd',
  'set',
  'set-location',
  'source',
  'true'
])
const WRAPPER_COMMANDS = new Set(['builtin', 'command', 'doas', 'exec', 'env', 'nice', 'nohup', 'sudo', 'time'])
const POSIX_SHELL_COMMANDS = new Set(['bash', 'sh', 'zsh'])
const CMD_SHELL_COMMANDS = new Set(['cmd'])
const POWERSHELL_COMMANDS = new Set(['powershell', 'pwsh'])

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

export function invokedCommand(call: ToolCall) {
  return commandNameFromText(commandSummary(call))
}

export function commandNameFromText(command: string, depth = 0): string {
  if (depth > 4) return ''
  const tokens = tokenizeCommand(command)
  for (const segment of commandSegments(tokens)) {
    const name = commandNameFromSegment(segment, depth)
    if (name) return name
  }
  return ''
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

function commandNameFromSegment(segment: string[], depth: number): string {
  const tokens = [...segment]
  while (tokens.length > 0) {
    const token = cleanCommandToken(tokens.shift() || '')
    if (!token || isEnvironmentAssignment(token) || isRedirectionToken(token)) continue

    const name = normalizeExecutableName(token)
    if (!name) continue

    if (POSIX_SHELL_COMMANDS.has(name)) {
      const nested = nestedPosixShellCommand(tokens)
      return nested ? commandNameFromText(nested, depth + 1) || name : name
    }
    if (CMD_SHELL_COMMANDS.has(name)) {
      const nested = nestedCmdShellCommand(tokens)
      return nested ? commandNameFromText(nested, depth + 1) || name : name
    }
    if (POWERSHELL_COMMANDS.has(name)) {
      const nested = nestedPowerShellCommand(tokens)
      return nested ? commandNameFromText(nested, depth + 1) || name : name
    }
    if (WRAPPER_COMMANDS.has(name)) {
      stripWrapperPrefix(name, tokens)
      continue
    }
    if (SKIPPED_COMMANDS.has(name)) return ''

    return name
  }
  return ''
}

function commandSegments(tokens: string[]) {
  const segments: string[][] = []
  let current: string[] = []
  for (const token of tokens) {
    if (COMMAND_SEPARATORS.has(token)) {
      if (current.length) segments.push(current)
      current = []
    } else {
      current.push(token)
    }
  }
  if (current.length) segments.push(current)
  return segments
}

function tokenizeCommand(command: string) {
  const tokens: string[] = []
  let current = ''
  let quote = ''
  let escaping = false

  const pushCurrent = () => {
    if (current) tokens.push(current)
    current = ''
  }

  for (let index = 0; index < command.length; index += 1) {
    const char = command[index]
    const next = command[index + 1] || ''

    if (escaping) {
      current += char
      escaping = false
      continue
    }

    if (quote) {
      if (char === '\\' && (next === quote || next === '\\')) {
        escaping = true
      } else if (char === quote) {
        quote = ''
      } else {
        current += char
      }
      continue
    }

    if (char === '"' || char === "'") {
      quote = char
      continue
    }
    if (/\s/.test(char)) {
      pushCurrent()
      continue
    }
    if ((char === '&' && next === '&') || (char === '|' && next === '|')) {
      pushCurrent()
      tokens.push(char + next)
      index += 1
      continue
    }
    if (char === ';' || char === '|') {
      pushCurrent()
      tokens.push(char)
      continue
    }
    current += char
  }
  pushCurrent()
  return tokens
}

function nestedPosixShellCommand(tokens: string[]) {
  for (let index = 0; index < tokens.length; index += 1) {
    const token = cleanCommandToken(tokens[index]).toLowerCase()
    if (token === '-c' || /^-[a-z]*c[a-z]*$/.test(token)) {
      return tokens.slice(index + 1).join(' ').trim()
    }
  }
  return ''
}

function nestedCmdShellCommand(tokens: string[]) {
  for (let index = 0; index < tokens.length; index += 1) {
    const token = cleanCommandToken(tokens[index]).toLowerCase()
    if (token === '/c' || token === '/k') return tokens.slice(index + 1).join(' ').trim()
  }
  return ''
}

function nestedPowerShellCommand(tokens: string[]) {
  for (let index = 0; index < tokens.length; index += 1) {
    const token = cleanCommandToken(tokens[index]).toLowerCase()
    if (token === '-command' || token === '-c' || token === '/c') {
      return tokens.slice(index + 1).join(' ').trim()
    }
  }
  return ''
}

function stripWrapperPrefix(wrapper: string, tokens: string[]) {
  if (wrapper === 'env') {
    while (tokens.length && (cleanCommandToken(tokens[0]).startsWith('-') || isEnvironmentAssignment(cleanCommandToken(tokens[0])))) {
      tokens.shift()
    }
    return
  }

  while (tokens.length && cleanCommandToken(tokens[0]).startsWith('-')) {
    const option = cleanCommandToken(tokens.shift() || '')
    if (wrapper === 'sudo' && ['-g', '-h', '-p', '-u'].includes(option) && tokens.length) tokens.shift()
    if (wrapper === 'nice' && option === '-n' && tokens.length) tokens.shift()
  }
}

function cleanCommandToken(token: string) {
  return token.trim().replace(/^[([{]+|[)\]},]+$/g, '').replace(/^&/, '').trim()
}

function normalizeExecutableName(token: string) {
  let name = cleanCommandToken(token)
  if (!name) return ''
  name = name.replace(/^\.?[\\/]+/, '')
  const parts = name.split(/[\\/]+/).filter(Boolean)
  name = parts[parts.length - 1] || name
  name = name.replace(/\.(exe|cmd|bat|ps1|sh)$/i, '').toLowerCase()
  if (name === 'py' || /^python\d*(?:\.\d+)?$/.test(name)) return 'python'
  if (/^pip\d*(?:\.\d+)?$/.test(name)) return 'pip'
  if (name === 'nodejs') return 'node'
  return name
}

function isEnvironmentAssignment(token: string) {
  return /^[A-Za-z_][A-Za-z0-9_]*=/.test(token) || /^\$env:[A-Za-z_][A-Za-z0-9_]*=/i.test(token)
}

function isRedirectionToken(token: string) {
  return /^(\d*)[<>]/.test(token)
}
