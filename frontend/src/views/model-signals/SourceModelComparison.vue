<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import ASegmented from 'ant-design-vue/es/segmented'
import ASelect from 'ant-design-vue/es/select'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import { BarChartOutlined } from '@ant-design/icons-vue'
import {
  formatNumber,
  type ModelSignalMatrixCell,
  type ModelSignalMatrixRow,
  type ModelSignalMetricSet
} from '../../api'
import { chartPalette } from '../../chartPalette'
import { useEChart } from '../../composables/useEChart'
import { useMessages } from '../../i18n'
import {
  formatModelSignalPercent as formatPercent,
  formatModelSignalRate as formatRate
} from '../../presentation/modelSignals'
import { sourceDisplay } from '../../presentation/sourceIdentity'
import Panel from '../../components/ui/Panel.vue'

type ComparisonMetricKey = 'latency' | 'throughput' | 'outputThroughput' | 'failurePressure' | 'toolFailureRate' | 'retryPressure'

interface ComparisonMetric {
  key: ComparisonMetricKey
  label: string
  description: string
  color: string
  direction: 'lower' | 'higher'
  value: (metric?: ModelSignalMetricSet) => number | undefined
  format: (value?: number) => string
}

interface ComparisonRow {
  key: string
  sourceKey: string
  sourceLabel: string
  sourceTitle: string
  sourceSecondary: string
  model: string
  modelProvider: string
  cell: ModelSignalMatrixCell
  value?: number
  baselineValue?: number
}

const props = withDefaults(defineProps<{
  rows?: ModelSignalMatrixRow[]
  loading?: boolean
}>(), {
  rows: () => [],
  loading: false
})

const messages = {
  en: {
    'title': 'Source and Model Comparison',
    'kicker': 'Compare the same operational metric across sources, then inspect models inside one source',
    'control.metric': 'Metric',
    'control.source': 'Source',
    'chart.crossSource': 'Across sources and models',
    'chart.withinSource': 'Models in selected source',
    'tag.direction.lower': 'lower is better',
    'tag.direction.higher': 'higher is better',
    'metric.latency': 'P90 latency',
    'metric.latencyDesc': 'Tail latency normalized per 1k output tokens',
    'metric.throughput': 'P10 throughput',
    'metric.throughputDesc': 'Slow-floor total-token throughput',
    'metric.outputThroughput': 'Output throughput',
    'metric.outputThroughputDesc': 'Observed output-token throughput',
    'metric.failurePressure': 'Failure pressure',
    'metric.failurePressureDesc': 'Failed model and tool calls per session',
    'metric.toolFailureRate': 'Tool failure rate',
    'metric.toolFailureRateDesc': 'Failed tool calls divided by tool calls',
    'metric.retryPressure': 'Retry pressure',
    'metric.retryPressureDesc': 'Model calls per session',
    'tooltip.current': 'Current',
    'tooltip.baseline': 'Baseline',
    'tooltip.sessions': 'Sessions',
    'tooltip.calls': 'Model calls',
    'tooltip.reason': 'Reason',
    'tooltip.unavailable': 'Unavailable',
    'empty.title': 'No comparable source/model data',
    'empty.text': 'Broaden the current source, model, project, or date scope.',
    'fallback.unknown': 'unknown',
    'fallback.noReason': 'No drift reason'
  },
  'zh-CN': {
    'title': '来源与模型横向对比',
    'kicker': '用同一个运营指标横向比较不同来源，再查看某个来源内的模型差异',
    'control.metric': '指标',
    'control.source': '来源',
    'chart.crossSource': '跨来源与模型',
    'chart.withinSource': '选中来源内的模型',
    'tag.direction.lower': '越低越好',
    'tag.direction.higher': '越高越好',
    'metric.latency': 'P90 延迟',
    'metric.latencyDesc': '按 1k 输出 token 归一化的尾部延迟',
    'metric.throughput': 'P10 吞吐',
    'metric.throughputDesc': '低谷总 token 吞吐',
    'metric.outputThroughput': '输出吞吐',
    'metric.outputThroughputDesc': '观测输出 token 吞吐',
    'metric.failurePressure': '失败压力',
    'metric.failurePressureDesc': '每会话失败模型与工具调用数',
    'metric.toolFailureRate': '工具失败率',
    'metric.toolFailureRateDesc': '失败工具调用占工具调用比例',
    'metric.retryPressure': '重试压力',
    'metric.retryPressureDesc': '每会话模型调用数',
    'tooltip.current': '当前',
    'tooltip.baseline': '基线',
    'tooltip.sessions': '会话',
    'tooltip.calls': '模型调用',
    'tooltip.reason': '原因',
    'tooltip.unavailable': '不可用',
    'empty.title': '没有可对比的来源/模型数据',
    'empty.text': '可以放宽来源、模型、项目或日期范围。',
    'fallback.unknown': '未知',
    'fallback.noReason': '无漂移原因'
  }
} as const

const { t, locale } = useMessages(messages)
const selectedMetricKey = ref<ComparisonMetricKey>('latency')
const selectedSourceKey = ref('')
const crossSourceChart = useEChart()
const sourceModelChart = useEChart()

const metrics = computed<ComparisonMetric[]>(() => [
  {
    key: 'latency',
    label: t('metric.latency'),
    description: t('metric.latencyDesc'),
    color: chartPalette.danger,
    direction: 'lower',
    value: (metric) => firstFinite(metric?.p90ModelLatencyMsPer1kOutputTokens, metric?.modelLatencyMsPer1kOutputTokens),
    format: (value) => `${formatRate(value, 0)} ms/1k`
  },
  {
    key: 'throughput',
    label: t('metric.throughput'),
    description: t('metric.throughputDesc'),
    color: chartPalette.success,
    direction: 'higher',
    value: (metric) => firstFinite(metric?.p10ModelThroughputTokensPerSecond, metric?.modelThroughputTokensPerSecond),
    format: (value) => `${formatRate(value, 1)} tok/s`
  },
  {
    key: 'outputThroughput',
    label: t('metric.outputThroughput'),
    description: t('metric.outputThroughputDesc'),
    color: chartPalette.info,
    direction: 'higher',
    value: (metric) => finiteNumber(metric?.modelThroughputOutputTokensPerSecond),
    format: (value) => `${formatRate(value, 1)} tok/s`
  },
  {
    key: 'failurePressure',
    label: t('metric.failurePressure'),
    description: t('metric.failurePressureDesc'),
    color: chartPalette.warning,
    direction: 'lower',
    value: (metric) => finiteNumber(metric?.failurePressure),
    format: (value) => `${formatRate(value, 2)}/session`
  },
  {
    key: 'toolFailureRate',
    label: t('metric.toolFailureRate'),
    description: t('metric.toolFailureRateDesc'),
    color: '#c2410c',
    direction: 'lower',
    value: (metric) => finiteNumber(metric?.toolFailureRate),
    format: (value) => formatPercent(value)
  },
  {
    key: 'retryPressure',
    label: t('metric.retryPressure'),
    description: t('metric.retryPressureDesc'),
    color: chartPalette.indigo,
    direction: 'lower',
    value: (metric) => finiteNumber(metric?.avgModelCallsPerSession),
    format: (value) => `${formatRate(value, 2)}/session`
  }
])

const metricOptions = computed(() => metrics.value.map((metric) => ({
  label: metric.label,
  value: metric.key,
  title: metric.description
})))
const selectedMetric = computed(() => metrics.value.find((metric) => metric.key === selectedMetricKey.value) || metrics.value[0])
const flattenedRows = computed(() => flattenRows(props.rows, selectedMetric.value, t('fallback.unknown')))
const sourceOptions = computed(() => {
  const values = new Map<string, { value: string; label: string; title: string }>()
  flattenedRows.value.forEach((row) => {
    if (values.has(row.sourceKey)) return
    values.set(row.sourceKey, {
      value: row.sourceKey,
      label: row.sourceSecondary ? `${row.sourceLabel} · ${row.sourceSecondary}` : row.sourceLabel,
      title: row.sourceTitle
    })
  })
  return [...values.values()].sort((left, right) => left.label.localeCompare(right.label))
})
const crossSourceRows = computed(() => sortedRows(flattenedRows.value, selectedMetric.value).slice(0, 12))
const sourceModelRows = computed(() => sortedRows(
  flattenedRows.value.filter((row) => row.sourceKey === selectedSourceKey.value),
  selectedMetric.value
))
const hasCrossSourceRows = computed(() => crossSourceRows.value.length > 0)
const hasSourceModelRows = computed(() => sourceModelRows.value.length > 0)
const directionLabel = computed(() => t(`tag.direction.${selectedMetric.value.direction}`))

watch(sourceOptions, (options) => {
  if (!options.length) {
    selectedSourceKey.value = ''
    return
  }
  if (!options.some((option) => option.value === selectedSourceKey.value)) {
    selectedSourceKey.value = options[0].value
  }
}, { immediate: true })

watch(() => [selectedMetricKey.value, props.rows, selectedSourceKey.value, locale.value], renderAfterUpdate, { deep: true })

onMounted(renderAfterUpdate)

async function renderAfterUpdate() {
  await nextTick()
  renderChart(crossSourceChart, crossSourceRows.value)
  renderChart(sourceModelChart, sourceModelRows.value)
}

function renderChart(target: ReturnType<typeof useEChart>, rows: ComparisonRow[]) {
  if (!rows.length) {
    target.disposeChart()
    return
  }
  const chart = target.getChart()
  if (!chart) return
  chart.setOption(buildChartOption(rows, selectedMetric.value), true)
}

function buildChartOption(rows: ComparisonRow[], metric: ComparisonMetric) {
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'shadow', shadowStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => tooltipMarkup(params, rows, metric)
    },
    grid: { left: 150, right: 34, top: 18, bottom: 34 },
    xAxis: {
      type: 'value',
      axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: (value: number) => axisLabel(metric, value) },
      splitLine: { lineStyle: { color: chartPalette.grid } }
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: rows.map((row) => axisRowLabel(row)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.text, fontSize: 11, overflow: 'truncate', width: 132 }
    },
    series: [{
      name: metric.label,
      type: 'bar',
      barMaxWidth: 14,
      itemStyle: { color: metric.color, borderRadius: [0, 3, 3, 0] },
      data: rows.map((row) => row.value ?? null)
    }]
  }
}

function flattenRows(rows: ModelSignalMatrixRow[], metric: ComparisonMetric, fallback: string): ComparisonRow[] {
  return rows.flatMap((row) => {
    const source = sourceDisplay(row, fallback)
    return (row.cells || []).map((cell) => {
      const model = cell.model || fallback
      const modelProvider = cell.modelProvider || fallback
      const currentValue = hasMetricSetSamples(cell.current) ? metric.value(cell.current) : undefined
      const baselineValue = hasMetricSetSamples(cell.baseline) ? metric.value(cell.baseline) : undefined
      return {
        key: `${source.key}:${modelProvider}:${model}`,
        sourceKey: source.key,
        sourceLabel: source.label,
        sourceTitle: source.title,
        sourceSecondary: source.secondary,
        model,
        modelProvider,
        cell,
        value: currentValue,
        baselineValue
      }
    }).filter((item) => item.value !== undefined)
  })
}

function sortedRows(rows: ComparisonRow[], metric: ComparisonMetric) {
  return [...rows].sort((left, right) => {
    const leftValue = left.value ?? 0
    const rightValue = right.value ?? 0
    if (leftValue !== rightValue) {
      return metric.direction === 'higher' ? leftValue - rightValue : rightValue - leftValue
    }
    if (left.sourceLabel !== right.sourceLabel) return left.sourceLabel.localeCompare(right.sourceLabel)
    return left.model.localeCompare(right.model)
  })
}

function tooltipMarkup(params: unknown, rows: ComparisonRow[], metric: ComparisonMetric) {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as { dataIndex?: number } | undefined
  const row = rows[first?.dataIndex ?? 0]
  if (!row) return ''
  return [
    `<strong>${escapeHtml(row.sourceLabel)} / ${escapeHtml(row.model)}</strong>`,
    `<div>${escapeHtml(row.modelProvider)}</div>`,
    `<div>${escapeHtml(t('tooltip.current'))}: ${escapeHtml(formatComparisonValue(metric, row.value))}</div>`,
    `<div>${escapeHtml(t('tooltip.baseline'))}: ${escapeHtml(formatComparisonValue(metric, row.baselineValue))}</div>`,
    `<div>${escapeHtml(t('tooltip.sessions'))}: ${formatNumber(row.cell.sessionCount)}</div>`,
    `<div>${escapeHtml(t('tooltip.calls'))}: ${formatNumber(row.cell.modelCalls)}</div>`,
    `<div>${escapeHtml(t('tooltip.reason'))}: ${escapeHtml(row.cell.keyReason || t('fallback.noReason'))}</div>`
  ].join('')
}

function axisRowLabel(row: ComparisonRow) {
  return `${row.sourceLabel} / ${row.model}`
}

function axisLabel(metric: ComparisonMetric, value: number) {
  if (metric.key === 'toolFailureRate') return formatPercent(value)
  if (metric.key === 'latency') return compactNumber(value)
  if (metric.key === 'throughput' || metric.key === 'outputThroughput') return compactNumber(value)
  return formatRate(value, 1)
}

function formatComparisonValue(metric: ComparisonMetric, value?: number) {
  return value === undefined ? t('tooltip.unavailable') : metric.format(value)
}

function hasMetricSetSamples(metric?: ModelSignalMetricSet) {
  return Boolean(metric && ((metric.sessionCount || 0) > 0 || (metric.modelCalls || 0) > 0))
}

function firstFinite(...values: Array<number | undefined>) {
  return values.find((value) => Number.isFinite(value))
}

function finiteNumber(value?: number) {
  return Number.isFinite(value) ? value : undefined
}

function compactNumber(value: number) {
  const normalized = Number(value || 0)
  if (Math.abs(normalized) >= 1_000_000) return `${formatRate(normalized / 1_000_000, 1)}M`
  if (Math.abs(normalized) >= 1_000) return `${formatRate(normalized / 1_000, 1)}K`
  return formatRate(normalized, Math.abs(normalized) >= 10 ? 0 : 1)
}

function escapeHtml(value: string | number | undefined) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
</script>

<template>
  <Panel class="model-signals-comparison-panel" :title="t('title')" :kicker="t('kicker')" :icon="BarChartOutlined">
    <div class="model-signals-comparison-toolbar">
      <a-segmented
        v-model:value="selectedMetricKey"
        class="model-signals-comparison-metrics"
        :options="metricOptions"
        :aria-label="t('control.metric')"
      />
      <a-select
        v-model:value="selectedSourceKey"
        class="model-signals-comparison-source"
        :options="sourceOptions"
        :placeholder="t('control.source')"
        :aria-label="t('control.source')"
        show-search
        option-filter-prop="label"
      />
      <a-tag class="status-tag model-signals-comparison-direction" :color="selectedMetric.direction === 'higher' ? 'success' : 'warning'">
        {{ directionLabel }}
      </a-tag>
    </div>

    <a-spin :spinning="loading">
      <div class="model-signals-comparison-grid">
        <section class="model-signals-comparison-block">
          <div class="model-signals-comparison-head">
            <h3>{{ t('chart.crossSource') }}</h3>
            <span>{{ selectedMetric.description }}</span>
          </div>
          <div v-if="hasCrossSourceRows" :ref="crossSourceChart.chartEl" class="chart model-signals-comparison-chart"></div>
          <div v-else class="empty-state model-signals-comparison-empty">
            <BarChartOutlined class="empty-state-icon" />
            <div class="empty-state-title">{{ t('empty.title') }}</div>
            <div class="empty-state-text">{{ t('empty.text') }}</div>
          </div>
        </section>

        <section class="model-signals-comparison-block">
          <div class="model-signals-comparison-head">
            <h3>{{ t('chart.withinSource') }}</h3>
            <span>{{ selectedMetric.description }}</span>
          </div>
          <div v-if="hasSourceModelRows" :ref="sourceModelChart.chartEl" class="chart model-signals-comparison-chart"></div>
          <div v-else class="empty-state model-signals-comparison-empty">
            <BarChartOutlined class="empty-state-icon" />
            <div class="empty-state-title">{{ t('empty.title') }}</div>
            <div class="empty-state-text">{{ t('empty.text') }}</div>
          </div>
        </section>
      </div>
    </a-spin>
  </Panel>
</template>

<style scoped>
.model-signals-comparison-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-width: 0;
  margin-bottom: 14px;
}

.model-signals-comparison-metrics {
  max-width: 100%;
  overflow-x: auto;
}

.model-signals-comparison-source {
  width: min(360px, 100%);
}

.model-signals-comparison-direction {
  margin-right: 0;
  white-space: nowrap;
}

.model-signals-comparison-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 18px;
  min-width: 0;
}

.model-signals-comparison-block {
  min-width: 0;
}

.model-signals-comparison-head {
  display: grid;
  gap: 3px;
  min-width: 0;
  margin-bottom: 8px;
}

.model-signals-comparison-head h3 {
  margin: 0;
  color: var(--am-text);
  font-size: 13px;
  font-weight: 750;
}

.model-signals-comparison-head span {
  color: var(--am-muted);
  font-size: 12px;
  line-height: 18px;
}

.model-signals-comparison-chart {
  height: 340px;
}

.model-signals-comparison-empty {
  min-height: 260px;
}

@media (max-width: 1120px) {
  .model-signals-comparison-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .model-signals-comparison-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .model-signals-comparison-source {
    width: 100%;
  }

  .model-signals-comparison-chart {
    height: 300px;
  }
}
</style>
