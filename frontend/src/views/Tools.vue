<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import * as echarts from 'echarts'
import { api, formatDuration, formatNumber, type ToolStat } from '../api'

const loading = ref(true)
const tools = ref<ToolStat[]>([])
const chartEl = ref<HTMLDivElement | null>(null)
let chart: echarts.ECharts | null = null

const columns = [
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName' },
  { title: 'Calls', dataIndex: 'calls', key: 'calls', width: 120 },
  { title: 'Success', dataIndex: 'successCalls', key: 'success', width: 120 },
  { title: 'Failed/Pending', dataIndex: 'failedCalls', key: 'failed', width: 140 },
  { title: 'Total Duration', dataIndex: 'totalDurationMs', key: 'totalDuration', width: 150 },
  { title: 'Average', dataIndex: 'avgDurationMs', key: 'average', width: 120 }
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
    color: ['#1677ff', '#fa8c16'],
    tooltip: { trigger: 'axis' },
    grid: { left: 130, right: 24, top: 20, bottom: 24 },
    xAxis: { type: 'value', splitLine: { lineStyle: { color: '#eef0f4' } } },
    yAxis: { type: 'category', data: top.map((item) => item.toolName), axisTick: { show: false } },
    series: [
      { name: 'Calls', type: 'bar', data: top.map((item) => item.calls), barWidth: 14 },
      { name: 'Failed/Pending', type: 'bar', data: top.map((item) => item.failedCalls), barWidth: 14 }
    ]
  })
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
        </div>
        <div class="panel-body">
          <div ref="chartEl" class="chart"></div>
        </div>
      </section>

      <section class="panel">
        <a-table :columns="columns" :data-source="tools" row-key="toolName" :pagination="{ pageSize: 20 }">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'calls'">
              {{ formatNumber(record.calls) }}
            </template>
            <template v-else-if="column.key === 'success'">
              {{ formatNumber(record.successCalls) }}
            </template>
            <template v-else-if="column.key === 'failed'">
              {{ formatNumber(record.failedCalls) }}
            </template>
            <template v-else-if="column.key === 'totalDuration'">
              {{ formatDuration(record.totalDurationMs) }}
            </template>
            <template v-else-if="column.key === 'average'">
              {{ formatDuration(record.avgDurationMs) }}
            </template>
          </template>
        </a-table>
      </section>
    </a-spin>
  </div>
</template>
