<script setup lang="ts">
import { computed, nextTick, onMounted, watch } from 'vue'
import ASpin from 'ant-design-vue/es/spin'
import { LineChartOutlined } from '@ant-design/icons-vue'
import { formatNumber, type ModelSignalsTrendPoint } from '../api'
import { chartPalette } from '../chartPalette'
import { useEChart } from '../composables/useEChart'
import { useMessages } from '../i18n'

const props = withDefaults(
  defineProps<{
    points: ModelSignalsTrendPoint[]
    loading?: boolean
  }>(),
  {
    loading: false
  }
)

const { chartEl, getChart, disposeChart } = useEChart()
const { t, locale } = useMessages({
  en: {
    'title': 'Signal Trend',
    'kicker': 'Operational proxy signals over time; low-sample days are marked separately',
    'series.throughput': 'Throughput',
    'series.rollingThroughput': '7-day throughput',
    'series.failureRate': 'Tool failure rate',
    'series.lowSample': 'Low sample',
    'tooltip.sessions': 'Sessions',
    'tooltip.modelCalls': 'Model calls',
    'tooltip.toolCalls': 'Tool calls',
    'tooltip.failedTools': 'Failed tools',
    'tooltip.tokens': 'Tokens',
    'tooltip.lowSample': 'Low sample day',
    'empty.title': 'No signal trend yet',
    'empty.text': 'Model signal trends appear after indexed sessions match the current scope.'
  },
  'zh-CN': {
    'title': '信号趋势',
    'kicker': '按时间查看运营代理信号；低样本日期会单独标记',
    'series.throughput': '吞吐',
    'series.rollingThroughput': '7 天吞吐',
    'series.failureRate': '工具失败率',
    'series.lowSample': '低样本',
    'tooltip.sessions': '会话',
    'tooltip.modelCalls': '模型调用',
    'tooltip.toolCalls': '工具调用',
    'tooltip.failedTools': '失败工具',
    'tooltip.tokens': 'Token',
    'tooltip.lowSample': '低样本日期',
    'empty.title': '暂无信号趋势',
    'empty.text': '当前范围内有已索引会话后，这里会显示模型信号趋势。'
  }
})

const hasTrend = computed(() => props.points.some((point) => point.sessionCount > 0 || point.modelCalls > 0))

async function renderAfterUpdate() {
  await nextTick()
  renderChart()
}

function renderChart() {
  const points = props.points || []
  if (!points.length || !hasTrend.value) {
    disposeChart()
    return
  }
  const chart = getChart()
  if (!chart) return

  chart.setOption({
    color: [chartPalette.info, chartPalette.primary, chartPalette.danger, chartPalette.warning],
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'cross', lineStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => tooltipMarkup(params)
    },
    grid: { left: 58, right: 58, top: 48, bottom: 38 },
    legend: {
      top: 4,
      right: 8,
      itemGap: 16,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: chartPalette.axis, fontSize: 12 }
    },
    xAxis: {
      type: 'category',
      data: points.map((point) => point.date.slice(5)),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.axis, fontSize: 11 }
    },
    yAxis: [
      {
        type: 'value',
        axisLabel: {
          color: chartPalette.axis,
          fontSize: 11,
          formatter: (value: number) => compactNumber(value)
        },
        splitLine: { lineStyle: { color: chartPalette.grid } }
      },
      {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: '{value}%' },
        splitLine: { show: false }
      }
    ],
    series: [
      {
        name: t('series.throughput'),
        type: 'line',
        smooth: true,
        symbolSize: 7,
        lineStyle: { width: 2, color: chartPalette.info },
        itemStyle: { color: chartPalette.info },
        data: points.map((point) => point.modelThroughputTokensPerSecond)
      },
      {
        name: t('series.rollingThroughput'),
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 2, color: chartPalette.primary },
        data: points.map((point) => point.rollingModelThroughputTokensPerSecond)
      },
      {
        name: t('series.failureRate'),
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        symbolSize: 6,
        lineStyle: { width: 2, color: chartPalette.danger },
        itemStyle: { color: chartPalette.danger },
        data: points.map((point) => toPercent(point.toolFailureRate))
      },
      {
        name: t('series.lowSample'),
        type: 'scatter',
        symbol: 'diamond',
        symbolSize: 11,
        itemStyle: { color: chartPalette.warning },
        data: points.map((point) => (point.lowSample ? point.modelThroughputTokensPerSecond : null))
      }
    ]
  }, true)
}

function tooltipMarkup(params: unknown) {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as { dataIndex?: number; axisValue?: string } | undefined
  const point = props.points[first?.dataIndex ?? 0]
  if (!point) return ''

  const rows = [
    `${t('series.throughput')}: ${formatRate(point.modelThroughputTokensPerSecond, 1)} tok/s`,
    `${t('series.rollingThroughput')}: ${formatRate(point.rollingModelThroughputTokensPerSecond, 1)} tok/s`,
    `${t('series.failureRate')}: ${formatPercent(point.toolFailureRate)}`,
    `${t('tooltip.sessions')}: ${formatNumber(point.sessionCount)}`,
    `${t('tooltip.modelCalls')}: ${formatNumber(point.modelCalls)}`,
    `${t('tooltip.toolCalls')}: ${formatNumber(point.toolCalls)}`,
    `${t('tooltip.failedTools')}: ${formatNumber(point.failedToolCalls)}`,
    `${t('tooltip.tokens')}: ${formatNumber(point.totalTokens)}`
  ]
  if (point.lowSample) rows.push(t('tooltip.lowSample'))
  return [
    `<strong>${point.date || first?.axisValue || ''}</strong>`,
    ...rows.map((row) => `<div>${row}</div>`)
  ].join('')
}

function toPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(100, value * 100))
}

function formatPercent(value: number) {
  return `${Math.round(toPercent(value))}%`
}

function formatRate(value: number, digits = 0) {
  if (!Number.isFinite(value)) return '0'
  return value.toLocaleString(undefined, { maximumFractionDigits: digits })
}

function compactNumber(value: number) {
  const normalized = Number(value || 0)
  if (Math.abs(normalized) >= 1_000_000) return `${Math.round(normalized / 1_000_000)}M`
  if (Math.abs(normalized) >= 1_000) return `${Math.round(normalized / 1_000)}K`
  return String(Math.round(normalized))
}

watch(() => [props.points, locale.value], renderAfterUpdate, { deep: true })

onMounted(() => {
  renderAfterUpdate()
})
</script>

<template>
  <section class="panel model-signals-trend-panel">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ t('title') }}</h2>
        <div class="panel-kicker">{{ t('kicker') }}</div>
      </div>
      <LineChartOutlined class="panel-header-icon" />
    </div>
    <a-spin :spinning="loading">
      <div class="panel-body">
        <div v-if="hasTrend" ref="chartEl" class="chart model-signals-trend-chart"></div>
        <div v-else class="empty-state model-signals-trend-empty">
          <LineChartOutlined class="empty-state-icon" />
          <div class="empty-state-title">{{ t('empty.title') }}</div>
          <div class="empty-state-text">{{ t('empty.text') }}</div>
        </div>
      </div>
    </a-spin>
  </section>
</template>

<style scoped>
.model-signals-trend-chart {
  height: 330px;
}

.model-signals-trend-empty {
  min-height: 260px;
}
</style>
