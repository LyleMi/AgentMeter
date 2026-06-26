<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import * as echarts from 'echarts'
import {
  BarChartOutlined,
  ClockCircleOutlined,
  DollarCircleOutlined,
  FunctionOutlined,
  TableOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatCost,
  formatDateTime,
  formatDuration,
  formatNumber,
  shortPath,
  type ModelUsage,
  type Overview,
  type Session
} from '../api'
import { chartPalette, usageChartColors } from '../chartPalette'

const router = useRouter()
const loading = ref(true)
const overview = ref<Overview | null>(null)
const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

const hasIndexedData = computed(() => (overview.value?.totalSessions || 0) > 0)
const hasDailyUsage = computed(() => (overview.value?.dailyUsage?.length || 0) > 0)
const rankedModelUsage = computed(() =>
  [...(overview.value?.modelUsage || [])].sort((left, right) => right.totalTokens - left.totalTokens)
)
const hasModelUsage = computed(() => rankedModelUsage.value.length > 0)
const hasRecentSessions = computed(() => (overview.value?.recentSessions?.length || 0) > 0)
const unpricedModelCount = computed(() => rankedModelUsage.value.filter((item) => item.unpriced).length)

const modelColumns = [
  { title: 'Model', dataIndex: 'model', key: 'model' },
  { title: 'Sessions', dataIndex: 'sessionCount', key: 'sessionCount', width: 96, align: 'right' },
  { title: 'Tokens', dataIndex: 'totalTokens', key: 'totalTokens', width: 132, align: 'right' },
  { title: 'Cost', dataIndex: 'estimatedCostUsd', key: 'cost', width: 118, align: 'right' }
]

const recentColumns = [
  { title: 'Project', dataIndex: 'projectPath', key: 'projectPath' },
  { title: 'Model', dataIndex: 'model', key: 'model', width: 132 },
  { title: 'Tokens', dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120, align: 'right' },
  { title: 'Tools', dataIndex: 'toolCallCount', key: 'tools', width: 80, align: 'right' },
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 150 }
]

async function load() {
  loading.value = true
  try {
    overview.value = await api.getOverview()
  } finally {
    loading.value = false
  }
  await nextTick()
  renderChart()
}

function renderChart() {
  const dailyUsage = overview.value?.dailyUsage || []
  if (!dailyUsage.length) {
    chart?.dispose()
    chart = null
    return
  }
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value)
  const days = dailyUsage.map((item) => item.date.slice(5))
  chart.setOption({
    color: usageChartColors,
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'shadow', shadowStyle: { color: chartPalette.pointer } },
      valueFormatter: (value: string | number) => formatNumber(Number(value))
    },
    grid: { left: 56, right: 44, top: 50, bottom: 36 },
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
      data: days,
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.axis, fontSize: 11 }
    },
    yAxis: [
      {
        type: 'value',
        axisLabel: { color: chartPalette.axis, fontSize: 11 },
        splitLine: { lineStyle: { color: chartPalette.grid } }
      },
      {
        type: 'value',
        axisLabel: { color: chartPalette.axis, fontSize: 11 },
        splitLine: { show: false }
      }
    ],
    series: [
      {
        name: 'Input',
        type: 'bar',
        stack: 'tokens',
        data: dailyUsage.map((item) => item.inputTokens),
        barWidth: 16,
        itemStyle: { borderRadius: [0, 0, 4, 4] },
        emphasis: { focus: 'series' }
      },
      {
        name: 'Output',
        type: 'bar',
        stack: 'tokens',
        data: dailyUsage.map((item) => item.outputTokens),
        barWidth: 16,
        itemStyle: { borderRadius: [4, 4, 0, 0] },
        emphasis: { focus: 'series' }
      },
      {
        name: 'Tools',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        symbolSize: 6,
        lineStyle: { width: 2 },
        data: dailyUsage.map((item) => item.toolCalls)
      }
    ]
  }, true)
}

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function recentRow(record: Session) {
  return { class: 'overview-session-row is-clickable-row', onClick: () => openSession(record.id) }
}

function modelRow(record: ModelUsage) {
  return { class: record.unpriced ? 'overview-model-row is-unpriced-row' : 'overview-model-row' }
}

function resize() {
  chart?.resize()
}

onMounted(() => {
  load()
  window.addEventListener('resize', resize)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', resize)
  chart?.dispose()
})
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Overview</h1>
        <div class="page-subtitle">Indexed Codex usage across local JSONL sessions</div>
      </div>
      <a-button @click="load">Refresh</a-button>
    </div>

    <a-spin :spinning="loading">
      <section class="metric-strip overview-summary-strip">
        <div class="metric-strip-item metric-primary">
          <div class="metric-strip-head">
            <span class="metric-label">Sessions</span>
            <ClockCircleOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(overview?.totalSessions) }}</div>
          <div class="metric-strip-note">{{ formatDuration(overview?.totalWallDurationMs) }} wall time</div>
        </div>
        <div class="metric-strip-item metric-success">
          <div class="metric-strip-head">
            <span class="metric-label">Tokens</span>
            <FunctionOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(overview?.totalTokens) }}</div>
          <div class="metric-strip-note">
            {{ formatNumber(overview?.totalInputTokens) }} in · {{ formatNumber(overview?.totalOutputTokens) }} out ·
            {{ formatNumber(overview?.totalCachedInputTokens) }} cached
          </div>
        </div>
        <div class="metric-strip-item metric-warning">
          <div class="metric-strip-head">
            <span class="metric-label">Estimated Cost</span>
            <DollarCircleOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatCost(overview?.estimatedCostUsd) }}</div>
          <div class="metric-strip-note" :class="{ 'metric-note-warning': (overview?.unpricedSessions || 0) > 0 }">
            {{ formatNumber(overview?.unpricedSessions) }} sessions missing pricing
          </div>
        </div>
        <div class="metric-strip-item metric-info">
          <div class="metric-strip-head">
            <span class="metric-label">Tool Calls</span>
            <ToolOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(overview?.totalToolCalls) }}</div>
          <div class="metric-strip-note">Across indexed sessions</div>
        </div>
        <div class="metric-strip-item metric-neutral">
          <div class="metric-strip-head">
            <span class="metric-label">Active Time</span>
            <ClockCircleOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatDuration(overview?.totalActiveDurationMs) }}</div>
          <div class="metric-strip-note">Measured model and tool time</div>
        </div>
      </section>

      <div v-if="!loading && !hasIndexedData" class="empty-callout overview-empty-callout">
        <div>
          <div class="empty-callout-title">No indexed sessions yet</div>
          <div class="empty-callout-text">
            Configure a local source path, then run Index Now to populate usage, cost, and tool-call telemetry.
          </div>
        </div>
        <a-button type="primary" @click="$router.push('/settings')">Open Settings</a-button>
      </div>

      <div class="content-grid overview-primary-grid">
        <section class="panel overview-chart-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">Daily Usage</h2>
              <div class="panel-kicker">Input, output, and tool activity by day</div>
            </div>
            <BarChartOutlined class="panel-header-icon" />
          </div>
          <div class="panel-body">
            <div v-if="hasDailyUsage" ref="chartEl" class="chart"></div>
            <div v-else class="empty-state">
              <BarChartOutlined class="empty-state-icon" />
              <div class="empty-state-title">No daily usage to chart</div>
              <div class="empty-state-text">Indexed sessions will appear here as input, output, and tool activity by day.</div>
            </div>
          </div>
        </section>

        <section class="panel overview-model-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">Model Usage</h2>
              <div class="panel-kicker">Ranked by token volume</div>
            </div>
            <TableOutlined class="panel-header-icon" />
          </div>
          <a-table
            v-if="hasModelUsage"
            class="overview-model-table"
            size="small"
            :columns="modelColumns"
            :data-source="rankedModelUsage"
            :pagination="false"
            row-key="model"
            :custom-row="modelRow"
          >
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.key === 'model'">
                <div class="model-rank-cell">
                  <span class="model-rank">{{ index + 1 }}</span>
                  <span class="model-name">{{ record.model || 'unknown' }}</span>
                  <a-tag v-if="record.unpriced" class="model-status-tag" color="warning">unpriced</a-tag>
                </div>
              </template>
              <template v-else-if="column.key === 'sessionCount'">
                <span class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
              </template>
              <template v-else-if="column.key === 'totalTokens'">
                <span class="number-cell">{{ formatNumber(record.totalTokens) }}</span>
              </template>
              <template v-else-if="column.key === 'cost'">
                <span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span>
              </template>
            </template>
          </a-table>
          <div v-else class="empty-state empty-state-compact">
            <TableOutlined class="empty-state-icon" />
            <div class="empty-state-title">No model usage yet</div>
            <div class="empty-state-text">Model rankings will appear after at least one session is indexed.</div>
          </div>
          <div v-if="unpricedModelCount > 0" class="panel-footer-note status-warning">
            {{ formatNumber(unpricedModelCount) }} model entries need pricing coverage.
          </div>
        </section>
      </div>

      <section class="panel overview-recent-panel panel-spaced">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Recent Sessions</h2>
            <div class="panel-kicker">Open a row to inspect timeline and calls</div>
          </div>
          <a-button type="link" @click="$router.push('/sessions')">View all</a-button>
        </div>
        <a-table
          v-if="hasRecentSessions"
          class="overview-session-table"
          size="middle"
          :columns="recentColumns"
          :data-source="overview?.recentSessions || []"
          :pagination="false"
          row-key="id"
          :custom-row="recentRow"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'startedAt'">
              {{ formatDateTime(record.startedAt) }}
            </template>
            <template v-else-if="column.key === 'projectPath'">
              <div class="overview-session-identity">
                <a-typography-text class="overview-session-project" :ellipsis="{ tooltip: record.projectPath }">
                  {{ shortPath(record.projectPath) }}
                </a-typography-text>
                <span class="overview-session-meta mono">{{ record.codexSessionId }}</span>
              </div>
            </template>
            <template v-else-if="column.key === 'model'">
              <a-tag class="model-lite-tag">{{ record.model || 'unknown' }}</a-tag>
            </template>
            <template v-else-if="column.key === 'tokens'">
              <span class="number-cell">{{ formatNumber(record.tokenUsage.totalTokens) }}</span>
            </template>
            <template v-else-if="column.key === 'tools'">
              <span class="number-cell">{{ formatNumber(record.toolCallCount) }}</span>
            </template>
          </template>
        </a-table>
        <div v-else class="empty-state empty-state-compact">
          <ClockCircleOutlined class="empty-state-icon" />
          <div class="empty-state-title">No recent sessions</div>
          <div class="empty-state-text">Recently indexed sessions will be listed here for quick inspection.</div>
        </div>
      </section>
    </a-spin>
  </div>
</template>
