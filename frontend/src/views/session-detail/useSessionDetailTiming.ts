import { computed, type Ref } from 'vue'
import { formatDuration, type ModelCall, type SessionDetail, type ToolCall } from '../../api'

type TimingMessageKey =
  | 'timing.model'
  | 'timing.tools'
  | 'timing.idle'
  | 'timing.status.unknown'
  | 'timing.status.idle'
  | 'timing.noCalls'
  | 'timing.kind.model'
  | 'timing.kind.tool'
  | 'timing.kind.gap'
  | 'fallback.unknown'

type Translate = (key: TimingMessageKey) => string

export type TimingKind = 'model' | 'tool' | 'gap'

export interface TimingCompositionSegment {
  key: 'model' | 'tool' | 'idle'
  label: string
  durationMs: number
  percent: number
  width: number
}

interface TimingCallSegment {
  id: string
  kind: Exclude<TimingKind, 'gap'>
  label: string
  status: string
  startMs: number
  endMs: number
  durationMs: number
}

export interface TimingRow {
  id: string
  kind: TimingKind
  label: string
  status: string
  startMs: number
  endMs: number
  durationMs: number
  left: number
  width: number
}

function safeDurationMs(value: number | undefined): number {
  return Number.isFinite(value) ? Math.max(0, value || 0) : 0
}

function safeTimestampMs(value: string | undefined): number | null {
  if (!value) return null
  const parsed = Date.parse(value)
  return Number.isFinite(parsed) ? parsed : null
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}

function percentOf(value: number, total: number): number {
  if (total <= 0) return 0
  return clamp((value / total) * 100, 0, 100)
}

function normalizeCallBounds(
  startedAt: string,
  endedAt: string,
  durationMs: number,
  sessionStartMs: number | null,
  totalMs: number
): Pick<TimingCallSegment, 'startMs' | 'endMs' | 'durationMs'> {
  const declaredDurationMs = safeDurationMs(durationMs)
  const startTimestampMs = safeTimestampMs(startedAt)
  const endTimestampMs = safeTimestampMs(endedAt)
  let startMs: number | null = null
  let endMs: number | null = null

  if (sessionStartMs !== null) {
    if (startTimestampMs !== null) startMs = startTimestampMs - sessionStartMs
    if (endTimestampMs !== null) endMs = endTimestampMs - sessionStartMs
  } else if (startTimestampMs !== null && endTimestampMs !== null) {
    startMs = 0
    endMs = endTimestampMs - startTimestampMs
  }

  if (startMs === null && endMs !== null) startMs = endMs - declaredDurationMs
  if (endMs === null && startMs !== null) endMs = startMs + declaredDurationMs
  if (startMs === null && endMs === null) {
    startMs = 0
    endMs = declaredDurationMs
  }
  const resolvedStartMs = startMs ?? 0
  let resolvedEndMs = endMs ?? resolvedStartMs
  if (resolvedEndMs < resolvedStartMs) resolvedEndMs = resolvedStartMs

  const boundedStartMs = clamp(resolvedStartMs, 0, totalMs)
  const boundedEndMs = clamp(Math.max(resolvedEndMs, resolvedStartMs), 0, totalMs)
  const normalizedEndMs = Math.max(boundedStartMs, boundedEndMs)
  return {
    startMs: boundedStartMs,
    endMs: normalizedEndMs,
    durationMs: normalizedEndMs - boundedStartMs
  }
}

function createTimingRow(id: string, kind: TimingKind, label: string, status: string, startMs: number, endMs: number, totalMs: number): TimingRow {
  const boundedStartMs = clamp(startMs, 0, totalMs)
  const boundedEndMs = Math.max(boundedStartMs, clamp(endMs, 0, totalMs))
  const durationMs = boundedEndMs - boundedStartMs
  return {
    id,
    kind,
    label,
    status,
    startMs: boundedStartMs,
    endMs: boundedEndMs,
    durationMs,
    left: percentOf(boundedStartMs, totalMs),
    width: durationMs > 0 ? Math.max(percentOf(durationMs, totalMs), 0.75) : 0
  }
}

export function formatPreciseDuration(ms: number): string {
  const duration = safeDurationMs(ms)
  if (duration > 0 && duration < 1000) return `${Math.round(duration)}ms`
  return formatDuration(duration)
}

export function formatOffset(ms: number): string {
  return `+${formatPreciseDuration(ms)}`
}

export function timingKindColor(kind: TimingKind) {
  if (kind === 'model') return 'blue'
  if (kind === 'tool') return 'purple'
  return 'default'
}

export function useSessionDetailTiming(detail: Ref<SessionDetail | null>, t: Translate) {
  const timingTotalMs = computed(() => {
    if (!detail.value) return 1
    const session = detail.value.session
    const wallMs = safeDurationMs(session.wallDurationMs)
    if (wallMs > 0) return wallMs

    const sessionStartMs = safeTimestampMs(session.startedAt)
    const callBounds = [...detail.value.modelCalls, ...detail.value.toolCalls].map((call) => {
      const startTimestampMs = safeTimestampMs(call.startedAt)
      const endTimestampMs = safeTimestampMs(call.endedAt)
      if (sessionStartMs !== null && endTimestampMs !== null) return Math.max(0, endTimestampMs - sessionStartMs)
      if (sessionStartMs !== null && startTimestampMs !== null) return Math.max(0, startTimestampMs - sessionStartMs + safeDurationMs(call.durationMs))
      if (startTimestampMs !== null && endTimestampMs !== null) return Math.max(0, endTimestampMs - startTimestampMs)
      return safeDurationMs(call.durationMs)
    })

    return Math.max(
      safeDurationMs(session.activeDurationMs),
      safeDurationMs(session.modelDurationMs) + safeDurationMs(session.toolDurationMs) + safeDurationMs(session.idleDurationMs),
      ...callBounds,
      1
    )
  })

  const timingComposition = computed<TimingCompositionSegment[]>(() => {
    if (!detail.value) return []
    const session = detail.value.session
    const modelMs = safeDurationMs(session.modelDurationMs)
    const toolMs = safeDurationMs(session.toolDurationMs)
    const knownIdleMs = safeDurationMs(session.idleDurationMs)
    const wallMs = safeDurationMs(session.wallDurationMs)
    const unclassifiedMs = Math.max(0, wallMs - modelMs - toolMs - knownIdleMs)
    const idleMs = knownIdleMs + unclassifiedMs
    const totalMs = Math.max(wallMs, modelMs + toolMs + idleMs, 1)
    const segments: Array<Omit<TimingCompositionSegment, 'percent' | 'width'>> = [
      { key: 'model', label: t('timing.model'), durationMs: modelMs },
      { key: 'tool', label: t('timing.tools'), durationMs: toolMs },
      { key: 'idle', label: t('timing.idle'), durationMs: idleMs }
    ]

    return segments.map((segment) => {
      const percent = percentOf(segment.durationMs, totalMs)
      return {
        ...segment,
        percent,
        width: segment.durationMs > 0 ? Math.max(percent, 1.5) : 0
      }
    })
  })

  const timedCallSegments = computed<TimingCallSegment[]>(() => {
    if (!detail.value) return []
    const sessionStartMs = safeTimestampMs(detail.value.session.startedAt)
    const totalMs = timingTotalMs.value
    const modelSegments = detail.value.modelCalls.map((call: ModelCall): TimingCallSegment => ({
      id: `model-${call.id}`,
      kind: 'model',
      label: call.model || t('fallback.unknown'),
      status: call.status || t('timing.status.unknown'),
      ...normalizeCallBounds(call.startedAt, call.endedAt, call.durationMs, sessionStartMs, totalMs)
    }))
    const toolSegments = detail.value.toolCalls.map((call: ToolCall): TimingCallSegment => ({
      id: `tool-${call.id}`,
      kind: 'tool',
      label: call.toolName || t('fallback.unknown'),
      status: call.status || t('timing.status.unknown'),
      ...normalizeCallBounds(call.startedAt, call.endedAt, call.durationMs, sessionStartMs, totalMs)
    }))

    return [...modelSegments, ...toolSegments].sort((a, b) => a.startMs - b.startMs || a.endMs - b.endMs || a.id.localeCompare(b.id))
  })

  const timingRows = computed<TimingRow[]>(() => {
    const totalMs = timingTotalMs.value
    const rows: TimingRow[] = []
    let cursorMs = 0

    timedCallSegments.value.forEach((call) => {
      if (call.startMs > cursorMs) {
        rows.push(createTimingRow(`gap-${rows.length}`, 'gap', t('timing.idle'), t('timing.status.idle'), cursorMs, call.startMs, totalMs))
      }
      rows.push(createTimingRow(call.id, call.kind, call.label, call.status, call.startMs, call.endMs, totalMs))
      cursorMs = Math.max(cursorMs, call.endMs)
    })

    if (timedCallSegments.value.length === 0 && safeDurationMs(detail.value?.session.wallDurationMs) > 0) {
      rows.push(createTimingRow('gap-session', 'gap', t('timing.noCalls'), t('timing.status.idle'), 0, totalMs, totalMs))
    } else if (cursorMs < totalMs && totalMs > 1) {
      rows.push(createTimingRow(`gap-${rows.length}`, 'gap', t('timing.idle'), t('timing.status.idle'), cursorMs, totalMs, totalMs))
    }

    return rows
  })

  function timingKindLabel(kind: TimingKind) {
    if (kind === 'model') return t('timing.kind.model')
    if (kind === 'tool') return t('timing.kind.tool')
    return t('timing.kind.gap')
  }

  return {
    timingTotalMs,
    timingComposition,
    timingRows,
    timingKindLabel
  }
}
