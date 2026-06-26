<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import * as echarts from 'echarts'
import { api, formatDuration, formatNumber, type ToolStat } from '../api'
import { chartPalette, toolChartColors } from '../chartPalette'

const loading = ref(true)
const tools = ref<ToolStat[]>([])
const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

const columns = [
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName' },
  { title: 'Calls', dataIndex: 'calls', key: 'calls', width: 120, align: 'right' },
  { title: 'Success', dataIndex: 'successCalls', key: 'success', width: 140, align: 'right' },
  { title: 'Failed / Pending', dataIndex: 'failedCalls', key: 'failed', width: 160, align: 'right' },
  { title: 'Total Duration', dataIndex: 'totalDurationMs', key: 'totalDuration', width: 150, align: 'right' },
  { title: 'Average', dataIndex: 'avgDurationMs', key: 'average', width: 120, align: 'right' }
]

async function load() {
  loading.value = true
  try {
    tools.value = await api.getTools()
    setTimeout(renderChart)
  } finally {
    loading.value = false
  }
}

function renderChart() {
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value)
  const top = tools.value.slice(0, 12).reverse()
  chart.setOption({
    color: toolChartColors,
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      valueFormatter: (value: unknown) => formatNumber(Number(value))
    },
    legend: {
      top: 0,
      right: 4,
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: chartPalette.axis }
    },
    grid: { left: 132, right: 24, top: 36, bottom: 24 },
    xAxis: {
      type: 'value',
      axisLabel: { color: chartPalette.axis },
      splitLine: { lineStyle: { color: chartPalette.grid } }
    },
    yAxis: {
      type: 'category',
      data: top.map((item) => item.toolName),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.text }
    },
    series: [
      { name: 'Calls', type: 'bar', data: top.map((item) => item.calls), barWidth: 14 },
      { name: 'Failed / Pending', type: 'bar', data: top.map((item) => item.failedCalls), barWidth: 14 }
    ]
  })
}

function successRate(record: ToolStat) {
  if (!record.calls) return 0
  return Math.round((record.successCalls / record.calls) * 100)
}

function failureStatus(record: ToolStat) {
  if (!record.failedCalls) return { color: 'success', label: 'Clear' }
  const rate = Math.round((record.failedCalls / Math.max(record.calls, 1)) * 100)
  return {
    color: rate >= 10 ? 'error' : 'warning',
    label: `${formatNumber(record.failedCalls)} affected`
  }
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
        <h1 class="page-title">Tools</h1>
        <div class="page-subtitle">Aggregated tool-call counts, status and duration</div>
      </div>
      <a-button @click="load">Refresh</a-button>
    </div>

    <a-spin :spinning="loading">
      <section class="panel" style="margin-bottom: 18px">
        <div class="panel-header">
          <h2 class="panel-title">Top Tools</h2>
          <span class="muted">Calls with failed or pending outcomes separated</span>
        </div>
        <div class="panel-body">
          <div ref="chartEl" class="chart tools-chart"></div>
        </div>
      </section>

      <section class="panel">
        <a-table
          class="dense-table tools-table"
          :columns="columns"
          :data-source="tools"
          row-key="toolName"
          size="middle"
          :pagination="{ pageSize: 20, showSizeChanger: true }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'calls'">
              <span class="number-cell">{{ formatNumber(record.calls) }}</span>
            </template>
            <template v-else-if="column.key === 'success'">
              <div class="status-number-cell">
                <a-tag color="success" class="status-tag">{{ successRate(record) }}% ok</a-tag>
                <span class="number-cell muted">{{ formatNumber(record.successCalls) }}</span>
              </div>
            </template>
            <template v-else-if="column.key === 'failed'">
              <div class="status-number-cell">
                <a-tag :color="failureStatus(record).color" class="status-tag">
                  {{ failureStatus(record).label }}
                </a-tag>
              </div>
            </template>
            <template v-else-if="column.key === 'totalDuration'">
              <span class="number-cell duration-cell">{{ formatDuration(record.totalDurationMs) }}</span>
            </template>
            <template v-else-if="column.key === 'average'">
              <span class="number-cell duration-cell">{{ formatDuration(record.avgDurationMs) }}</span>
            </template>
          </template>
        </a-table>
      </section>
    </a-spin>
  </div>
</template>
