<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
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
import { api, formatCost, formatDateTime, formatDuration, formatNumber, shortPath, type Overview, type Session } from '../api'
import { chartPalette, usageChartColors } from '../chartPalette'

const router = useRouter()
const loading = ref(true)
const overview = ref<Overview | null>(null)
const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

const modelColumns = [
  { title: 'Model', dataIndex: 'model', key: 'model' },
  { title: 'Sessions', dataIndex: 'sessionCount', key: 'sessionCount', width: 96, align: 'right' },
  { title: 'Tokens', dataIndex: 'totalTokens', key: 'totalTokens', width: 132, align: 'right' },
  { title: 'Cost', dataIndex: 'estimatedCostUsd', key: 'cost', width: 118, align: 'right' }
]

const recentColumns = [
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 150 },
  { title: 'Project', dataIndex: 'projectPath', key: 'projectPath' },
  { title: 'Model', dataIndex: 'model', key: 'model', width: 110 },
  { title: 'Tokens', dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120, align: 'right' },
  { title: 'Tools', dataIndex: 'toolCallCount', key: 'tools', width: 80, align: 'right' }
]

async function load() {
  loading.value = true
  try {
    overview.value = await api.getOverview()
    setTimeout(renderChart)
  } finally {
    loading.value = false
  }
}

function renderChart() {
  if (!chartEl.value || !overview.value) return
  if (!chart) chart = echarts.init(chartEl.value)
  const days = overview.value.dailyUsage.map((item) => item.date.slice(5))
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
    grid: { left: 56, right: 44, top: 42, bottom: 36 },
    legend: {
      top: 2,
      right: 0,
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
        name: 'Tokens',
        nameTextStyle: { color: chartPalette.axis, fontSize: 11, padding: [0, 0, 8, 0] },
        axisLabel: { color: chartPalette.axis, fontSize: 11 },
        splitLine: { lineStyle: { color: chartPalette.grid } }
      },
      {
        type: 'value',
        name: 'Tools',
        nameTextStyle: { color: chartPalette.axis, fontSize: 11, padding: [0, 0, 8, 0] },
        axisLabel: { color: chartPalette.axis, fontSize: 11 },
        splitLine: { show: false }
      }
    ],
    series: [
      {
        name: 'Input',
        type: 'bar',
        stack: 'tokens',
        data: overview.value.dailyUsage.map((item) => item.inputTokens),
        barWidth: 16,
        emphasis: { focus: 'series' }
      },
      {
        name: 'Output',
        type: 'bar',
        stack: 'tokens',
        data: overview.value.dailyUsage.map((item) => item.outputTokens),
        barWidth: 16,
        emphasis: { focus: 'series' }
      },
      {
        name: 'Tools',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        symbolSize: 6,
        lineStyle: { width: 2 },
        data: overview.value.dailyUsage.map((item) => item.toolCalls)
      }
    ]
  })
}

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function recentRow(record: Session) {
  return { class: 'overview-session-row is-clickable-row', onClick: () => openSession(record.id) }
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
      <div class="metric-grid">
        <a-card class="metric-card overview-metric-card overview-metric-sessions" :bordered="false">
          <div class="metric-card-topline">
            <div class="metric-label">Sessions</div>
            <ClockCircleOutlined class="metric-icon" />
          </div>
          <div class="metric-value">{{ formatNumber(overview?.totalSessions) }}</div>
          <div class="metric-note">Indexed sessions · {{ formatDuration(overview?.totalWallDurationMs) }} wall time</div>
        </a-card>
        <a-card class="metric-card overview-metric-card overview-metric-tokens" :bordered="false">
          <div class="metric-card-topline">
            <div class="metric-label">Tokens</div>
            <FunctionOutlined class="metric-icon" />
          </div>
          <div class="metric-value">{{ formatNumber(overview?.totalTokens) }}</div>
          <div class="metric-note">
            {{ formatNumber(overview?.totalInputTokens) }} input · {{ formatNumber(overview?.totalOutputTokens) }} output ·
            {{ formatNumber(overview?.totalCachedInputTokens) }} cached
          </div>
        </a-card>
        <a-card class="metric-card overview-metric-card overview-metric-cost" :bordered="false">
          <div class="metric-card-topline">
            <div class="metric-label">Estimated Cost</div>
            <DollarCircleOutlined class="metric-icon" />
          </div>
          <div class="metric-value">{{ formatCost(overview?.estimatedCostUsd) }}</div>
          <div class="metric-note metric-note-warning">
            {{ formatNumber(overview?.unpricedSessions) }} sessions missing pricing
          </div>
        </a-card>
        <a-card class="metric-card overview-metric-card overview-metric-tools" :bordered="false">
          <div class="metric-card-topline">
            <div class="metric-label">Tool Calls</div>
            <ToolOutlined class="metric-icon" />
          </div>
          <div class="metric-value">{{ formatNumber(overview?.totalToolCalls) }}</div>
          <div class="metric-note">Across active work · {{ formatDuration(overview?.totalActiveDurationMs) }} active time</div>
        </a-card>
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
            <div ref="chartEl" class="chart"></div>
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
            class="overview-model-table"
            size="small"
            :columns="modelColumns"
            :data-source="overview?.modelUsage || []"
            :pagination="false"
            row-key="model"
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
        </section>
      </div>

      <section class="panel overview-recent-panel" style="margin-top: 18px">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Recent Sessions</h2>
            <div class="panel-kicker">Open a row to inspect timeline and calls</div>
          </div>
          <a-button type="link" @click="$router.push('/sessions')">View all</a-button>
        </div>
        <a-table
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
              <a-typography-text :ellipsis="{ tooltip: record.projectPath }">
                {{ shortPath(record.projectPath) }}
              </a-typography-text>
            </template>
            <template v-else-if="column.key === 'tokens'">
              <span class="number-cell">{{ formatNumber(record.tokenUsage.totalTokens) }}</span>
            </template>
            <template v-else-if="column.key === 'tools'">
              <span class="number-cell">{{ formatNumber(record.toolCallCount) }}</span>
            </template>
          </template>
        </a-table>
      </section>
    </a-spin>
  </div>
</template>
