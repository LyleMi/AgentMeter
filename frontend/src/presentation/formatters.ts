import { createDateTimeFormatter, createNumberFormatter, currentLocale, numberDisplayMode } from '../i18n'
import type { Session } from '../api/types'

export interface DisplayNumber {
  main: string
  suffix: string
  full: string
}

function localizedFallback(key: 'unknown' | 'unpriced') {
  if (currentLocale.value === 'zh-CN') return key === 'unknown' ? '未知' : '未定价'
  return key
}

export function formatNumber(value: number | undefined): string {
  return createNumberFormatter().format(value || 0)
}

export function formatCost(value?: number): string {
  if (value === undefined || value === null) return localizedFallback('unpriced')
  return createNumberFormatter({ style: 'currency', currency: 'USD', maximumFractionDigits: 4 }).format(value)
}

function shouldUseCompact(value: number) {
  if (numberDisplayMode.value === 'full') return false
  const threshold = numberDisplayMode.value === 'compact' ? 1_000 : 10_000
  return Math.abs(value) >= threshold
}

function compactFractionDigits(value: number) {
  const normalized = Math.abs(value)
  if (normalized >= 100_000_000) return 0
  if (normalized >= 10_000_000) return 1
  if (normalized >= 1_000_000) return 2
  return normalized >= 10_000 ? 1 : 0
}

function displayText(value: string): DisplayNumber {
  return { main: value, suffix: '', full: value }
}

export function formatDisplayNumber(value: number | undefined): DisplayNumber {
  const normalized = Number(value || 0)
  const full = formatNumber(normalized)
  if (!shouldUseCompact(normalized)) return { main: full, suffix: '', full }

  return {
    main: createNumberFormatter({
      notation: 'compact',
      compactDisplay: 'short',
      maximumFractionDigits: compactFractionDigits(normalized)
    }).format(normalized),
    suffix: '',
    full
  }
}

export function formatDisplayCost(value?: number): DisplayNumber {
  if (value === undefined || value === null) return displayText(localizedFallback('unpriced'))
  const full = formatCost(value)
  if (!shouldUseCompact(value)) {
    return {
      main: createNumberFormatter({
        style: 'currency',
        currency: 'USD',
        maximumFractionDigits: value > 0 && value < 1 ? 4 : 2
      }).format(value),
      suffix: '',
      full
    }
  }

  return {
    main: createNumberFormatter({
      style: 'currency',
      currency: 'USD',
      notation: 'compact',
      compactDisplay: 'short',
      maximumFractionDigits: compactFractionDigits(value)
    }).format(value),
    suffix: '',
    full
  }
}

export function formatDuration(ms: number | undefined): string {
  const total = Math.max(0, Math.round((ms || 0) / 1000))
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  const seconds = total % 60
  if (currentLocale.value === 'zh-CN') {
    if (hours > 0) return `${hours}小时 ${minutes}分钟`
    if (minutes > 0) return `${minutes}分钟 ${seconds}秒`
    return `${seconds}秒`
  }
  if (hours > 0) return `${hours}h ${minutes}m`
  if (minutes > 0) return `${minutes}m ${seconds}s`
  return `${seconds}s`
}

export function formatDateTime(value?: string): string {
  if (!value) return '-'
  return createDateTimeFormatter({
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(new Date(value))
}

export function shortPath(value: string): string {
  if (!value) return localizedFallback('unknown')
  const parts = value.split(/[\\/]/).filter(Boolean)
  if (parts.length <= 3) return value
  return `.../${parts.slice(-3).join('/')}`
}

export function sessionLabel(session: Pick<Session, 'id' | 'sessionKey' | 'codexSessionId'>): string {
  return session.sessionKey || session.codexSessionId || `#${session.id}`
}
