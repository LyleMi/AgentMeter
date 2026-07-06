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

interface TimingRowInput {
  id: string
  kind: TimingKind
  label: string
  status: string
  startMs: number
  endMs: number
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

function createTimingRow(input: TimingRowInput, totalMs: number): TimingRow {
  const boundedStartMs = clamp(input.startMs, 0, totalMs)
  const boundedEndMs = Math.max(boundedStartMs, clamp(input.endMs, 0, totalMs))
  const durationMs = boundedEndMs - boundedStartMs
  return {
    id: input.id,
    kind: input.kind,
    label: input.label,
    status: input.status,
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

function timingTotal(detail: SessionDetail | null): number {
  if (!detail) return 1
  const session = detail.session
  const wallMs = safeDurationMs(session.wallDurationMs)
  if (wallMs > 0) return wallMs

  return Math.max(
    safeDurationMs(session.activeDurationMs),
    safeDurationMs(session.modelDurationMs) + safeDurationMs(session.toolDurationMs) + safeDurationMs(session.idleDurationMs),
    ...callBounds(detail),
    1
  )
}

function callBounds(detail: SessionDetail): number[] {
  const sessionStartMs = safeTimestampMs(detail.session.startedAt)
  return [...detail.modelCalls, ...detail.toolCalls].map((call) => callBound(call, sessionStartMs))
}

function callBound(call: ModelCall | ToolCall, sessionStartMs: number | null): number {
  const startTimestampMs = safeTimestampMs(call.startedAt)
  const endTimestampMs = safeTimestampMs(call.endedAt)
  if (sessionStartMs !== null && endTimestampMs !== null) return Math.max(0, endTimestampMs - sessionStartMs)
  if (sessionStartMs !== null && startTimestampMs !== null) return Math.max(0, startTimestampMs - sessionStartMs + safeDurationMs(call.durationMs))
  if (startTimestampMs !== null && endTimestampMs !== null) return Math.max(0, endTimestampMs - startTimestampMs)
  return safeDurationMs(call.durationMs)
}

function timingCompositionSegments(detail: SessionDetail | null, t: Translate): TimingCompositionSegment[] {
  if (!detail) return []
  const session = detail.session
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
}

function timedCallSegments(detail: SessionDetail | null, totalMs: number, t: Translate): TimingCallSegment[] {
  if (!detail) return []
  const sessionStartMs = safeTimestampMs(detail.session.startedAt)
  const modelSegments = detail.modelCalls.map((call) => modelCallSegment(call, sessionStartMs, totalMs, t))
  const toolSegments = detail.toolCalls.map((call) => toolCallSegment(call, sessionStartMs, totalMs, t))
  return [...modelSegments, ...toolSegments].sort(compareTimingCallSegments)
}

function modelCallSegment(call: ModelCall, sessionStartMs: number | null, totalMs: number, t: Translate): TimingCallSegment {
  return {
    id: `model-${call.id}`,
    kind: 'model',
    label: call.model || t('fallback.unknown'),
    status: call.status || t('timing.status.unknown'),
    ...normalizeCallBounds(call.startedAt, call.endedAt, call.durationMs, sessionStartMs, totalMs)
  }
}

function toolCallSegment(call: ToolCall, sessionStartMs: number | null, totalMs: number, t: Translate): TimingCallSegment {
  return {
    id: `tool-${call.id}`,
    kind: 'tool',
    label: call.toolName || t('fallback.unknown'),
    status: call.status || t('timing.status.unknown'),
    ...normalizeCallBounds(call.startedAt, call.endedAt, call.durationMs, sessionStartMs, totalMs)
  }
}

function compareTimingCallSegments(a: TimingCallSegment, b: TimingCallSegment): number {
  return a.startMs - b.startMs || a.endMs - b.endMs || a.id.localeCompare(b.id)
}

function timingRowsFor(detail: SessionDetail | null, totalMs: number, calls: TimingCallSegment[], t: Translate): TimingRow[] {
  const rows: TimingRow[] = []
  let cursorMs = 0

  calls.forEach((call) => {
    if (call.startMs > cursorMs) {
      rows.push(createGapRow(rows.length, cursorMs, call.startMs, totalMs, t))
    }
    rows.push(createTimingRow(call, totalMs))
    cursorMs = Math.max(cursorMs, call.endMs)
  })

  if (calls.length === 0 && safeDurationMs(detail?.session.wallDurationMs) > 0) {
    rows.push(createTimingRow({ id: 'gap-session', kind: 'gap', label: t('timing.noCalls'), status: t('timing.status.idle'), startMs: 0, endMs: totalMs }, totalMs))
  } else if (cursorMs < totalMs && totalMs > 1) {
    rows.push(createGapRow(rows.length, cursorMs, totalMs, totalMs, t))
  }

  return rows
}

function createGapRow(index: number, startMs: number, endMs: number, totalMs: number, t: Translate): TimingRow {
  return createTimingRow({ id: `gap-${index}`, kind: 'gap', label: t('timing.idle'), status: t('timing.status.idle'), startMs, endMs }, totalMs)
}

export function useSessionDetailTiming(detail: Ref<SessionDetail | null>, t: Translate) {
  const timingTotalMs = computed(() => timingTotal(detail.value))
  const timingComposition = computed<TimingCompositionSegment[]>(() => timingCompositionSegments(detail.value, t))
  const timedCalls = computed<TimingCallSegment[]>(() => timedCallSegments(detail.value, timingTotalMs.value, t))
  const timingRows = computed<TimingRow[]>(() => timingRowsFor(detail.value, timingTotalMs.value, timedCalls.value, t))

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
