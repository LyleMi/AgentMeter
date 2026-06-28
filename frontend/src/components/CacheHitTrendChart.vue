<script setup lang="ts">
import { computed, nextTick, onMounted, watch } from 'vue'
import ASpin from 'ant-design-vue/es/spin'
import { LineChartOutlined } from '@ant-design/icons-vue'
import { formatNumber, formatPercent, type CacheHitTrendPoint } from '../api'
import { chartPalette } from '../chartPalette'
import { useEChart } from '../composables/useEChart'
import { useMessages } from '../i18n'

const props = withDefaults(
  defineProps<{
    points: CacheHitTrendPoint[]
    title?: string
    kicker?: string
    compact?: boolean
    loading?: boolean
  }>(),
  {
    title: undefined,
    kicker: undefined,
    compact: false,
    loading: false
  }
)

const { chartEl, getChart, disposeChart } = useEChart()
const { t, locale } = useMessages({
  en: {
    'title': 'Cache Hit Trend',
    'kicker': 'Daily cache reuse with input-weighted 7-day trend',
    'series.daily': 'Daily hit rate',
    'series.rolling': '7-day weighted',
    'series.input': 'Input tokens',
    'tooltip.sessions': 'Sessions',
    'tooltip.cached': 'Cached input',
    'tooltip.input': 'Input tokens',
    'tooltip.low': 'Low input volume',
    'empty.title': 'No cache trend to chart',
    'empty.text': 'Cache hit rate appears after indexed sessions include input token usage.'
  },
  'zh-CN': {
    'title': '缓存命中趋势',
    'kicker': '按天展示缓存复用，并叠加按输入 Token 加权的 7 天趋势',
    'series.daily': '每日命中率',
    'series.rolling': '7 天加权',
    'series.input': '输入 Token',
    'tooltip.sessions': '会话',
    'tooltip.cached': '缓存输入',
    'tooltip.input': '输入 Token',
    'tooltip.low': '低输入量',
    'empty.title': '暂无缓存趋势可绘制',
    'empty.text': '索引到包含输入 Token 的会话后，这里会显示缓存命中率。'
  }
})

const chartTitle = computed(() => props.title || t('title'))
const chartKicker = computed(() => props.kicker || t('kicker'))
const hasTrend = computed(() => props.points.some((point) => point.hasUsage || point.inputTokens > 0))

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
    color: [chartPalette.success, chartPalette.primary, chartPalette.warning],
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'cross', lineStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => tooltipMarkup(params)
    },
    grid: {
      left: props.compact ? 44 : 54,
      right: props.compact ? 42 : 58,
      top: props.compact ? 26 : 48,
      bottom: props.compact ? 30 : 38
    },
    legend: {
      show: !props.compact,
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
        min: 0,
        max: 100,
        axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: '{value}%' },
        splitLine: { lineStyle: { color: chartPalette.grid } }
      },
      {
        type: 'value',
        axisLabel: {
          color: chartPalette.axis,
          fontSize: 11,
          formatter: (value: number) => compactNumber(value)
        },
        splitLine: { show: false }
      }
    ],
    series: [
      {
        name: t('series.input'),
        type: 'bar',
        yAxisIndex: 1,
        data: points.map((point) => point.inputTokens),
        barWidth: props.compact ? 10 : 14,
        itemStyle: { color: 'rgba(100, 116, 139, 0.22)', borderRadius: [3, 3, 0, 0] },
        emphasis: { focus: 'series' }
      },
      {
        name: t('series.daily'),
        type: 'line',
        smooth: true,
        connectNulls: false,
        symbolSize: props.compact ? 5 : 7,
        lineStyle: { width: 2, color: chartPalette.success },
        itemStyle: { color: chartPalette.success },
        data: points.map((point) => ({
          value: point.inputTokens > 0 ? toPercent(point.cacheUtilizationRate) : null,
          symbolSize: point.lowInputVolume ? (props.compact ? 7 : 9) : (props.compact ? 5 : 7),
          itemStyle: { color: point.lowInputVolume ? chartPalette.warning : chartPalette.success }
        }))
      },
      {
        name: t('series.rolling'),
        type: 'line',
        smooth: true,
        symbol: 'none',
        lineStyle: { width: 2, color: chartPalette.primary },
        data: points.map((point) => toPercent(point.rollingCacheUtilizationRate))
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
    `${t('series.daily')}: ${formatPercent(point.cacheUtilizationRate, { clamp: true })}`,
    `${t('series.rolling')}: ${formatPercent(point.rollingCacheUtilizationRate, { clamp: true })}`,
    `${t('tooltip.input')}: ${formatNumber(point.inputTokens)}`,
    `${t('tooltip.cached')}: ${formatNumber(point.cachedInputTokens)}`,
    `${t('tooltip.sessions')}: ${formatNumber(point.sessionCount)}`
  ]
  if (point.lowInputVolume) rows.push(t('tooltip.low'))
  return [
    `<strong>${point.date || first?.axisValue || ''}</strong>`,
    ...rows.map((row) => `<div>${row}</div>`)
  ].join('')
}

function toPercent(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(100, value * 100))
}

function compactNumber(value: number) {
  const normalized = Number(value || 0)
  if (Math.abs(normalized) >= 1_000_000) return `${Math.round(normalized / 1_000_000)}M`
  if (Math.abs(normalized) >= 1_000) return `${Math.round(normalized / 1_000)}K`
  return String(Math.round(normalized))
}

watch(() => [props.points, props.compact, locale.value], renderAfterUpdate, { deep: true })

onMounted(() => {
  renderAfterUpdate()
})
</script>

<template>
  <section class="panel cache-hit-trend-panel" :class="{ 'is-compact': compact }">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ chartTitle }}</h2>
        <div class="panel-kicker">{{ chartKicker }}</div>
      </div>
      <LineChartOutlined class="panel-header-icon" />
    </div>
    <a-spin :spinning="loading">
      <div class="panel-body">
        <div v-if="hasTrend" ref="chartEl" class="chart cache-hit-trend-chart"></div>
        <div v-else class="empty-state cache-hit-trend-empty">
          <LineChartOutlined class="empty-state-icon" />
          <div class="empty-state-title">{{ t('empty.title') }}</div>
          <div class="empty-state-text">{{ t('empty.text') }}</div>
        </div>
      </div>
    </a-spin>
  </section>
</template>

<style scoped>
.cache-hit-trend-panel.is-compact .panel-header {
  margin-bottom: 6px;
}

.cache-hit-trend-chart {
  height: 320px;
}

.cache-hit-trend-panel.is-compact .cache-hit-trend-chart {
  height: 220px;
}

.cache-hit-trend-empty {
  min-height: 220px;
}
</style>
