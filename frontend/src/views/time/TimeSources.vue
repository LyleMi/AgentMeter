<script setup lang="ts">
import { computed, nextTick, onMounted, watch } from 'vue'
import { BarChartOutlined, ClockCircleOutlined } from '@ant-design/icons-vue'
import { formatDuration, formatNumber, type AgentTimeUsage } from '../../api'
import { chartPalette } from '../../chartPalette'
import { useEChart } from '../../composables/useEChart'
import { useMessages } from '../../i18n'
import { sourceDisplay, sourceInstanceKey } from '../../presentation/sourceIdentity'
import { useTimeContext } from './timeContext'

interface SourceMetricCard {
  label: string
  value: string
  note: string
}

const { t, locale, createNumberFormatter } = useMessages({
  en: {
    'hero.title': 'Source time comparison',
    'hero.kicker': 'Compare wall time, active time, model time, tool time, and idle gaps for each indexed source',
    'chart.title': 'Time by source',
    'chart.kicker': 'Stacked by model, tool, and idle duration; sorted by total wall time',
    'table.title': 'Source detail',
    'table.kicker': 'Per-source totals and normalized averages',
    'metric.sources': 'Sources',
    'metric.sourcesNote': '{count} indexed sessions',
    'metric.wall': 'Wall time',
    'metric.wallNote': 'across selected scope',
    'metric.topSource': 'Top source',
    'metric.topSourceNote': '{share} of wall time',
    'metric.avgSession': 'Avg per session',
    'metric.avgSessionNote': 'wall time / session',
    'series.model': 'Model',
    'series.tool': 'Tool',
    'series.idle': 'Idle',
    'empty.title': 'No source time rows',
    'empty.text': 'Indexed sessions with wall-time data will appear here.',
    'fallback.unknown': 'unknown',
    'column.source': 'Source',
    'column.sessions': 'Sessions',
    'column.calls': 'Calls',
    'column.wall': 'Wall',
    'column.active': 'Active',
    'column.activeShare': 'Active %',
    'column.avg': 'Avg / session',
    'column.modelTime': 'Model',
    'column.tool': 'Tool',
    'column.network': 'Network',
    'column.idle': 'Idle'
  },
  'zh-CN': {
    'hero.title': '来源耗时对比',
    'hero.kicker': '按每个已索引来源对比墙钟、活跃、模型、工具和空闲耗时',
    'chart.title': '按来源的耗时构成',
    'chart.kicker': '按模型、工具和空闲耗时堆叠展示，并按总墙钟时间排序',
    'table.title': '来源明细',
    'table.kicker': '每个来源的总量与标准化平均值',
    'metric.sources': '来源数',
    'metric.sourcesNote': '{count} 个已索引会话',
    'metric.wall': '墙钟耗时',
    'metric.wallNote': '当前筛选范围合计',
    'metric.topSource': '最高来源',
    'metric.topSourceNote': '占墙钟耗时 {share}',
    'metric.avgSession': '单会话平均',
    'metric.avgSessionNote': '墙钟耗时 / 会话',
    'series.model': '模型',
    'series.tool': '工具',
    'series.idle': '空闲',
    'empty.title': '暂无来源耗时行',
    'empty.text': '索引包含墙钟耗时数据的会话后会显示在这里。',
    'fallback.unknown': '未知',
    'column.source': '来源',
    'column.sessions': '会话',
    'column.calls': '调用',
    'column.wall': '墙钟',
    'column.active': '活跃',
    'column.activeShare': '活跃占比',
    'column.avg': '单会话平均',
    'column.modelTime': '模型',
    'column.tool': '工具',
    'column.network': '网络',
    'column.idle': '空闲'
  }
})

const { rankedAgentTimeUsage: rows, wallDurationMs, formatPercent } = useTimeContext()
const { chartEl, getChart, disposeChart } = useEChart()

const hasRows = computed(() => rows.value.length > 0)
const topSource = computed(() => rows.value[0])
const totalSessions = computed(() => rows.value.reduce((sum, item) => sum + item.sessionCount, 0))
const totalWall = computed(() => rows.value.reduce((sum, item) => sum + item.wallDurationMs, 0))
const averageSessionMs = computed(() => (totalSessions.value > 0 ? totalWall.value / totalSessions.value : 0))
const topShare = computed(() => (wallDurationMs.value > 0 && topSource.value ? topSource.value.wallDurationMs / wallDurationMs.value : 0))

const metricCards = computed<SourceMetricCard[]>(() => [
  {
    label: t('metric.sources'),
    value: formatNumber(rows.value.length),
    note: t('metric.sourcesNote', { count: formatNumber(totalSessions.value) })
  },
  {
    label: t('metric.wall'),
    value: formatDuration(totalWall.value),
    note: t('metric.wallNote')
  },
  {
    label: t('metric.topSource'),
    value: topSource.value ? sourceInfo(topSource.value).label : '-',
    note: topSource.value ? t('metric.topSourceNote', { share: formatPercent(topShare.value) }) : t('empty.title')
  },
  {
    label: t('metric.avgSession'),
    value: formatDuration(averageSessionMs.value),
    note: t('metric.avgSessionNote')
  }
])

const columns = computed(() => [
  { title: t('column.source'), key: 'source', fixed: 'left', width: 240 },
  { title: t('column.sessions'), key: 'sessions', dataIndex: 'sessionCount', align: 'right', width: 92 },
  { title: t('column.calls'), key: 'calls', dataIndex: 'toolCalls', align: 'right', width: 92 },
  { title: t('column.wall'), key: 'wall', dataIndex: 'wallDurationMs', align: 'right', width: 120 },
  { title: t('column.active'), key: 'active', dataIndex: 'activeDurationMs', align: 'right', width: 120 },
  { title: t('column.activeShare'), key: 'activeShare', align: 'right', width: 104 },
  { title: t('column.avg'), key: 'avg', align: 'right', width: 120 },
  { title: t('column.modelTime'), key: 'modelTime', dataIndex: 'modelDurationMs', align: 'right', width: 120 },
  { title: t('column.tool'), key: 'tool', dataIndex: 'toolDurationMs', align: 'right', width: 120 },
  { title: t('column.network'), key: 'network', dataIndex: 'suspectedNetworkToolDurationMs', align: 'right', width: 120 },
  { title: t('column.idle'), key: 'idle', dataIndex: 'idleDurationMs', align: 'right', width: 120 }
])

function rowKey(record: AgentTimeUsage) {
  return sourceInstanceKey(record, t('fallback.unknown'))
}

function sourceInfo(record: AgentTimeUsage) {
  return sourceDisplay(record, t('fallback.unknown'))
}

function activeShare(record: AgentTimeUsage) {
  return record.wallDurationMs > 0 ? record.activeDurationMs / record.wallDurationMs : 0
}

function averageDuration(record: AgentTimeUsage) {
  return record.sessionCount > 0 ? record.wallDurationMs / record.sessionCount : 0
}

function chartRows() {
  return rows.value.slice(0, 12).reverse()
}

function chartLabel(record: AgentTimeUsage) {
  return sourceInfo(record).label
}

function durationTooltip(value: unknown) {
  return formatDuration(Number(value || 0))
}

async function renderAfterUpdate() {
  await nextTick()
  renderChart()
}

function renderChart() {
  const top = chartRows()
  if (!top.length) {
    disposeChart()
    return
  }
  const chart = getChart()
  if (!chart) return
  chart.setOption({
    color: [chartPalette.primary, chartPalette.success, chartPalette.warning],
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow', shadowStyle: { color: chartPalette.pointer } },
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      valueFormatter: durationTooltip
    },
    legend: {
      top: 0,
      right: 4,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: chartPalette.axis, fontSize: 12 }
    },
    grid: { left: 150, right: 34, top: 42, bottom: 28 },
    xAxis: {
      type: 'value',
      axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: (value: number) => formatDuration(value) },
      splitLine: { lineStyle: { color: chartPalette.grid } }
    },
    yAxis: {
      type: 'category',
      data: top.map(chartLabel),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.text, fontSize: 11, overflow: 'truncate', width: 132 }
    },
    series: [
      {
        name: t('series.model'),
        type: 'bar',
        stack: 'duration',
        data: top.map((item) => item.modelDurationMs),
        barWidth: 16,
        emphasis: { focus: 'series' }
      },
      {
        name: t('series.tool'),
        type: 'bar',
        stack: 'duration',
        data: top.map((item) => item.toolDurationMs),
        barWidth: 16,
        emphasis: { focus: 'series' }
      },
      {
        name: t('series.idle'),
        type: 'bar',
        stack: 'duration',
        data: top.map((item) => Math.max(0, item.idleDurationMs)),
        barWidth: 16,
        itemStyle: { borderRadius: [0, 3, 3, 0] },
        emphasis: { focus: 'series' }
      }
    ]
  }, true)
}

watch([rows, locale], renderAfterUpdate, { deep: true })
onMounted(renderAfterUpdate)
</script>

<template>
  <div class="section-stack">
    <section class="source-time-hero">
      <div class="source-time-hero-copy">
        <h2>{{ t('hero.title') }}</h2>
        <p>{{ t('hero.kicker') }}</p>
      </div>
      <div class="source-time-metrics">
        <div v-for="item in metricCards" :key="item.label" class="source-time-metric-card">
          <span class="metric-label">{{ item.label }}</span>
          <strong :title="item.value">{{ item.value }}</strong>
          <span>{{ item.note }}</span>
        </div>
      </div>
    </section>

    <section class="panel overview-time-panel source-time-chart-panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('chart.title') }}</h2>
          <div class="panel-kicker">{{ t('chart.kicker') }}</div>
        </div>
        <BarChartOutlined class="panel-header-icon" />
      </div>
      <div class="panel-body">
        <div v-if="hasRows" ref="chartEl" class="chart source-time-chart"></div>
        <div v-else class="empty-state empty-state-compact">
          <BarChartOutlined class="empty-state-icon" />
          <div class="empty-state-title">{{ t('empty.title') }}</div>
          <div class="empty-state-text">{{ t('empty.text') }}</div>
        </div>
      </div>
    </section>

    <section class="panel overview-time-panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('table.title') }}</h2>
          <div class="panel-kicker">{{ t('table.kicker') }}</div>
        </div>
        <ClockCircleOutlined class="panel-header-icon" />
      </div>
      <a-table
        v-if="hasRows"
        class="dense-table overview-time-table"
        size="small"
        :columns="columns"
        :data-source="rows"
        :pagination="false"
        :row-key="rowKey"
        :scroll="{ x: 1388 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'source'">
            <div class="source-identity-cell">
              <a-typography-text class="source-identity-name" :ellipsis="{ tooltip: sourceInfo(record).title }">
                {{ sourceInfo(record).label }}
              </a-typography-text>
            </div>
            <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
          </template>
          <template v-else-if="column.key === 'sessions'">
            <span class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
          </template>
          <template v-else-if="column.key === 'calls'">
            <span class="number-cell">{{ formatNumber(record.toolCalls) }}</span>
          </template>
          <template v-else-if="column.key === 'activeShare'">
            <span class="number-cell">{{ createNumberFormatter({ style: 'percent', maximumFractionDigits: 0 }).format(activeShare(record)) }}</span>
          </template>
          <template v-else-if="column.key === 'avg'">
            <span class="number-cell duration-cell">{{ formatDuration(averageDuration(record)) }}</span>
          </template>
          <template v-else>
            <span class="number-cell duration-cell">{{ formatDuration(record[column.dataIndex]) }}</span>
          </template>
        </template>
      </a-table>
      <div v-else class="empty-state empty-state-compact">
        <ClockCircleOutlined class="empty-state-icon" />
        <div class="empty-state-title">{{ t('empty.title') }}</div>
        <div class="empty-state-text">{{ t('empty.text') }}</div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.source-time-hero {
  display: grid;
  grid-template-columns: minmax(280px, 0.9fr) minmax(0, 1.4fr);
  gap: var(--am-section-gap);
  align-items: stretch;
}

.source-time-hero-copy,
.source-time-metric-card {
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--am-border);
  border-radius: var(--am-radius);
  background: var(--am-surface);
  box-shadow: 0 1px 0 rgb(255 255 255 / 80%), 0 8px 20px rgb(37 99 235 / 5%);
}

.source-time-hero-copy h2 {
  margin: 0;
  color: var(--am-text);
  font-size: 22px;
  font-weight: 800;
  line-height: 28px;
}

.source-time-hero-copy p {
  margin: 8px 0 0;
  color: var(--am-muted);
  font-size: 13px;
  line-height: 20px;
}

.source-time-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.source-time-metric-card strong {
  display: block;
  min-width: 0;
  margin-top: 8px;
  overflow: hidden;
  color: var(--am-text);
  font-size: 22px;
  font-weight: 800;
  line-height: 28px;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.source-time-metric-card span:last-child {
  display: block;
  margin-top: 5px;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-time-chart-panel .panel-body {
  padding: 0;
}

.source-time-chart {
  height: 360px;
}

.overview-time-panel {
  min-width: 0;
}

.overview-time-table {
  display: block;
}

@media (max-width: 1180px) {
  .source-time-hero {
    grid-template-columns: 1fr;
  }

  .source-time-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .source-time-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
