import type { ModelSignalsMessageKey, ModelSignalsTranslate } from './messages'

export function buildOverviewColumns(t: ModelSignalsTranslate) {
  return [
    { title: t('column.source'), dataIndex: 'sourceLabel', key: 'source', width: 170 },
    { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 210 },
    { title: t('column.model'), dataIndex: 'model', key: 'model', width: 190 },
    { title: t('column.severity'), dataIndex: 'severity', key: 'severity', width: 104 },
    { title: t('column.latency'), key: 'latency', width: 126, align: 'right' },
    { title: t('column.throughput'), key: 'throughput', width: 126, align: 'right' },
    { title: t('column.confidence'), key: 'confidence', width: 110, align: 'right' },
    { title: t('column.reasons'), key: 'reasons', width: 280 }
  ]
}

export function buildDailyColumns(t: ModelSignalsTranslate) {
  return [
    { title: t('column.date'), dataIndex: 'date', key: 'date', width: 112 },
    { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 88, align: 'right' },
    { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 108, align: 'right' },
    { title: t('column.costPerSession'), dataIndex: 'costPerSession', key: 'costPerSession', width: 124, align: 'right' },
    { title: t('column.costPerActiveHour'), dataIndex: 'costPerActiveHour', key: 'costPerActiveHour', width: 138, align: 'right' },
    { title: t('column.cacheSavings'), dataIndex: 'cacheSavingsUsd', key: 'cacheSavings', width: 124, align: 'right' },
    { title: t('column.p90Latency'), key: 'p90Latency', width: 124, align: 'right' },
    { title: t('column.p10Throughput'), key: 'p10Throughput', width: 128, align: 'right' },
    { title: t('column.retryPressure'), key: 'retryPressure', width: 130, align: 'right' },
    { title: t('column.failurePressure'), key: 'failurePressure', width: 132, align: 'right' },
    { title: t('column.confidence'), key: 'confidence', width: 220 }
  ]
}

export function buildCohortColumns(t: ModelSignalsTranslate) {
  return [
    { title: t('column.source'), dataIndex: 'sourceLabel', key: 'source', width: 180 },
    { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 220 },
    { title: t('column.model'), dataIndex: 'model', key: 'model', width: 190 },
    { title: t('column.samples'), key: 'samples', width: 128, align: 'right' },
    { title: t('column.latency'), key: 'latency', width: 136, align: 'right' },
    { title: t('column.throughput'), key: 'throughput', width: 136, align: 'right' },
    { title: t('column.outputThroughput'), key: 'outputThroughput', width: 118, align: 'right' },
    { title: t('column.toolFailure'), key: 'toolFailure', width: 108, align: 'right' },
    { title: t('column.severity'), key: 'severity', width: 104 },
    { title: t('column.confidence'), key: 'confidence', width: 104, align: 'right' },
    { title: t('column.reasons'), key: 'reasons', width: 280 }
  ]
}

export function buildMatrixColumns(t: ModelSignalsTranslate) {
  return [
    { title: t('column.source'), dataIndex: 'sourceLabel', key: 'source', width: 230 },
    { title: t('column.models'), dataIndex: 'cells', key: 'models' }
  ]
}

export function buildProjectColumns(t: ModelSignalsTranslate, hasProjectMetrics: boolean) {
  if (!hasProjectMetrics) {
    return [
      { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 260 },
      { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 92, align: 'right' },
      { title: t('column.sources'), dataIndex: 'sourceCount', key: 'sources', width: 88, align: 'right' },
      { title: t('column.models'), dataIndex: 'modelCount', key: 'models', width: 88, align: 'right' },
      { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 118, align: 'right' },
      { title: t('column.latency'), key: 'latency', width: 136, align: 'right' },
      { title: t('column.throughput'), key: 'throughput', width: 136, align: 'right' },
      { title: t('column.severity'), key: 'severity', width: 104 },
      { title: t('column.confidence'), key: 'confidence', width: 104, align: 'right' },
      { title: t('column.reasons'), key: 'reasons', width: 280 }
    ]
  }

  return [
    { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 260 },
    { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 92, align: 'right' },
    { title: t('column.mix'), key: 'mix', width: 210 },
    { title: t('column.costBurn'), key: 'costBurn', width: 132, align: 'right' },
    { title: t('column.cacheSavings'), key: 'cacheSavings', width: 124, align: 'right' },
    { title: t('column.health'), key: 'health', width: 142 },
    { title: t('column.p90Latency'), key: 'latency', width: 136, align: 'right' },
    { title: t('column.p10Throughput'), key: 'throughput', width: 136, align: 'right' },
    { title: t('column.failurePressure'), key: 'pressure', width: 136, align: 'right' },
    { title: t('column.reasons'), key: 'reasons', width: 280 }
  ]
}

export function buildAnomalyColumns(t: ModelSignalsTranslate) {
  return [
    { title: t('column.session'), dataIndex: 'sessionKey', key: 'session', width: 170 },
    { title: t('column.source'), dataIndex: 'sourceLabel', key: 'source', width: 170 },
    { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 220 },
    { title: t('column.model'), dataIndex: 'model', key: 'model', width: 170 },
    { title: t('column.signal'), dataIndex: 'reasons', key: 'signal', width: 260 },
    { title: t('column.outputExpansion'), dataIndex: 'outputExpansionRate', key: 'outputExpansion', width: 112, align: 'right' },
    { title: t('column.reasoning'), dataIndex: 'reasoningOverheadRate', key: 'reasoning', width: 100, align: 'right' },
    { title: t('column.cacheMiss'), dataIndex: 'cacheMissRate', key: 'cacheMiss', width: 120, align: 'right' },
    { title: t('column.failedTools'), dataIndex: 'failedToolCalls', key: 'failedTools', width: 90, align: 'right' },
    { title: t('column.throughput'), dataIndex: 'modelThroughputTokensPerSecond', key: 'throughput', width: 100, align: 'right' },
    { title: t('column.started'), dataIndex: 'startedAt', key: 'started', width: 136 },
    { title: t('column.wall'), dataIndex: 'modelDurationMs', key: 'duration', width: 98, align: 'right' },
    { title: '', key: 'open', width: 48, align: 'right' }
  ]
}

export function buildTableLocale(
  t: ModelSignalsTranslate,
  loading: boolean,
  emptyKey: ModelSignalsMessageKey
) {
  return { emptyText: loading ? t('empty.loading') : t(emptyKey) }
}
