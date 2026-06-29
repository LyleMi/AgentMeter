import { computed } from 'vue'
import type { EventItem } from '../../api'

type TableColumnMessageKey =
  | 'column.ended'
  | 'column.model'
  | 'column.status'
  | 'column.duration'
  | 'column.input'
  | 'column.cached'
  | 'column.output'
  | 'column.reasoning'
  | 'column.contextCompression'
  | 'column.total'
  | 'column.cost'
  | 'column.started'
  | 'column.tool'
  | 'column.rawEvent'
  | 'column.line'
  | 'column.time'
  | 'column.kind'
  | 'column.type'
  | 'column.summary'

type Translate = (key: TableColumnMessageKey) => string

export function useSessionDetailTableColumns(t: Translate) {
  const modelColumns = computed(() => [
    { title: t('column.ended'), dataIndex: 'endedAt', key: 'endedAt', width: 150 },
    { title: t('column.model'), dataIndex: 'model', key: 'model', width: 220 },
    { title: t('column.status'), dataIndex: 'status', key: 'status', width: 130 },
    { title: t('column.duration'), dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
    { title: t('column.input'), dataIndex: 'inputTokens', key: 'input', width: 100, align: 'right' },
    { title: t('column.cached'), dataIndex: 'cachedInputTokens', key: 'cached', width: 100, align: 'right' },
    { title: t('column.output'), dataIndex: 'outputTokens', key: 'output', width: 100, align: 'right' },
    { title: t('column.reasoning'), dataIndex: 'reasoningOutputTokens', key: 'reasoning', width: 110, align: 'right' },
    { title: t('column.contextCompression'), dataIndex: 'contextCompressionTokens', key: 'contextCompression', width: 110, align: 'right' },
    { title: t('column.total'), dataIndex: 'totalTokens', key: 'total', width: 110, align: 'right' },
    { title: t('column.cost'), dataIndex: 'costUsd', key: 'cost', width: 120, align: 'right' }
  ])

  const toolColumns = computed(() => [
    { title: t('column.started'), dataIndex: 'startedAt', key: 'startedAt', width: 150 },
    { title: t('column.ended'), dataIndex: 'endedAt', key: 'endedAt', width: 150 },
    { title: t('column.tool'), dataIndex: 'toolName', key: 'toolName', width: 160 },
    { title: t('column.status'), dataIndex: 'status', key: 'status', width: 110 },
    { title: t('column.duration'), dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
    { title: t('column.rawEvent'), dataIndex: 'rawEventId', key: 'rawEvent', width: 100, align: 'right' },
    { title: t('column.input'), dataIndex: 'inputSummary', key: 'input' },
    { title: t('column.output'), dataIndex: 'outputSummary', key: 'output' },
    { title: '', key: 'detail', width: 56, align: 'right' }
  ])

  const rawColumns = computed(() => [
    { title: t('column.line'), dataIndex: 'sourceLine', key: 'line', width: 80, align: 'right' },
    { title: t('column.time'), dataIndex: 'timestamp', key: 'time', width: 150 },
    { title: t('column.kind'), dataIndex: 'kind', key: 'kind', width: 100 },
    { title: t('column.type'), dataIndex: 'rawType', key: 'rawType', width: 150 },
    { title: t('column.summary'), dataIndex: 'summary', key: 'summary' }
  ])

  return {
    modelColumns,
    toolColumns,
    rawColumns
  }
}

export const rawEventsExpandable = {
  rowExpandable: (record: EventItem) => Boolean(record.rawJson)
}
