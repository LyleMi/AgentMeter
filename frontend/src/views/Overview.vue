<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import * as echarts from 'echarts'
import { api, formatCost, formatDateTime, formatDuration, formatNumber, shortPath, type Overview, type Session } from '../api'

const router = useRouter()
const loading = ref(true)
const overview = ref<Overview | null>(null)
const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

const modelColumns = [
  { title: 'Model', dataIndex: 'model', key: 'model' },
  { title: 'Sessions', dataIndex: 'sessionCount', key: 'sessionCount', width: 110 },
  { title: 'Tokens', dataIndex: 'totalTokens', key: 'totalTokens', width: 140 },
  { title: 'Cost', dataIndex: 'estimatedCostUsd', key: 'cost', width: 130 }
]

const recentColumns = [
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 150 },
  { title: 'Project', dataIndex: 'projectPath', key: 'projectPath' },
  { title: 'Model', dataIndex: 'model', key: 'model', width: 110 },
  { title: 'Tokens', dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120 },
  { title: 'Tools', dataIndex: 'toolCallCount', key: 'tools', width: 80 }
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
    color: ['#1677ff', '#13a8a8', '#fa8c16'],
    tooltip: { trigger: 'axis' },
    grid: { left: 48, right: 20, top: 28, bottom: 34 },
    legend: { top: 0, right: 0 },
    xAxis: { type: 'category', data: days, axisTick: { show: false } },
    yAxis: [
      { type: 'value', name: 'Tokens', splitLine: { lineStyle: { color: '#eef0f4' } } },
      { type: 'value', name: 'Tools', splitLine: { show: false } }
    ],
    series: [
      {
        name: 'Input',
        type: 'bar',
        stack: 'tokens',
        data: overview.value.dailyUsage.map((item) => item.inputTokens),
        barWidth: 14
      },
      {
        name: 'Output',
        type: 'bar',
        stack: 'tokens',
        data: overview.value.dailyUsage.map((item) => item.outputTokens),
        barWidth: 14
      },
      {
        name: 'Tools',
        type: 'line',
        yAxisIndex: 1,
        smooth: true,
        data: overview.value.dailyUsage.map((item) => item.toolCalls)
      }
    ]
  })
}

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function recentRow(record: Session) {
  return { onClick: () => openSession(record.id) }
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
        <a-card class="metric-card" :bordered="false">
          <div class="metric-label">Sessions</div>
          <div class="metric-value">{{ formatNumber(overview?.totalSessions) }}</div>
          <div class="metric-note">{{ formatDuration(overview?.totalWallDurationMs) }} wall time</div>
        </a-card>
        <a-card class="metric-card" :bordered="false">
          <div class="metric-label">Tokens</div>
          <div class="metric-value">{{ formatNumber(overview?.totalTokens) }}</div>
          <div class="metric-note">
            {{ formatNumber(overview?.totalCachedInputTokens) }} cached input
          </div>
        </a-card>
        <a-card class="metric-card" :bordered="false">
          <div class="metric-label">Estimated Cost</div>
          <div class="metric-value">{{ formatCost(overview?.estimatedCostUsd) }}</div>
          <div class="metric-note">{{ formatNumber(overview?.unpricedSessions) }} unpriced sessions</div>
        </a-card>
        <a-card class="metric-card" :bordered="false">
          <div class="metric-label">Tool Calls</div>
          <div class="metric-value">{{ formatNumber(overview?.totalToolCalls) }}</div>
          <div class="metric-note">{{ formatDuration(overview?.totalActiveDurationMs) }} active time</div>
        </a-card>
      </div>

      <div class="content-grid">
        <section class="panel">
          <div class="panel-header">
            <h2 class="panel-title">Daily Usage</h2>
          </div>
          <div class="panel-body">
            <div ref="chartEl" class="chart"></div>
          </div>
        </section>

        <section class="panel">
          <div class="panel-header">
            <h2 class="panel-title">Model Usage</h2>
          </div>
          <a-table
            size="small"
            :columns="modelColumns"
            :data-source="overview?.modelUsage || []"
            :pagination="false"
            row-key="model"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'totalTokens'">
                {{ formatNumber(record.totalTokens) }}
              </template>
              <template v-else-if="column.key === 'cost'">
                {{ formatCost(record.estimatedCostUsd) }}
              </template>
            </template>
          </a-table>
        </section>
      </div>

      <section class="panel" style="margin-top: 18px">
        <div class="panel-header">
          <h2 class="panel-title">Recent Sessions</h2>
          <a-button type="link" @click="$router.push('/sessions')">View all</a-button>
        </div>
        <a-table
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
              {{ formatNumber(record.tokenUsage.totalTokens) }}
            </template>
          </template>
        </a-table>
      </section>
    </a-spin>
  </div>
</template>
