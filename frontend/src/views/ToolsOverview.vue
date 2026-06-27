<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import { BarChartOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDuration, formatNumber, type ToolStat } from '../api'
import { chartPalette, toolChartColors } from '../chartPalette'
import { useEChart } from '../composables/useEChart'

const loading = ref(true)
const tools = ref<ToolStat[]>([])
const { chartEl, getChart, disposeChart } = useEChart()

const totalCalls = computed(() => tools.value.reduce((sum, item) => sum + item.calls, 0))
const toolsUsed = computed(() => tools.value.length)
const failedPendingCalls = computed(() => tools.value.reduce((sum, item) => sum + item.failedCalls, 0))
const totalDurationMs = computed(() => tools.value.reduce((sum, item) => sum + item.totalDurationMs, 0))
const averageDurationMs = computed(() => (totalCalls.value > 0 ? totalDurationMs.value / totalCalls.value : 0))

async function load() {
  loading.value = true
  try {
    tools.value = (await api.getTools()) || []
    setTimeout(renderChart)
  } finally {
    loading.value = false
  }
}

function renderChart() {
  if (!chartEl.value || tools.value.length === 0) {
    disposeChart()
    return
  }
  const chart = getChart()
  if (!chart) return
  const top = [...tools.value].sort((a, b) => b.calls - a.calls).slice(0, 12).reverse()
  chart.setOption({
    color: toolChartColors,
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'shadow', shadowStyle: { color: chartPalette.pointer } },
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
      textStyle: { color: chartPalette.axis, fontSize: 12 }
    },
    grid: { left: 136, right: 28, top: 38, bottom: 28 },
    xAxis: {
      type: 'value',
      axisLabel: { color: chartPalette.axis, fontSize: 11 },
      splitLine: { lineStyle: { color: chartPalette.grid } }
    },
    yAxis: {
      type: 'category',
      data: top.map((item) => item.toolName),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.text, fontSize: 11, overflow: 'truncate', width: 120 }
    },
    series: [
      {
        name: 'Successful',
        type: 'bar',
        stack: 'calls',
        data: top.map((item) => item.successCalls),
        barWidth: 16,
        itemStyle: { borderRadius: [0, 3, 3, 0] },
        emphasis: { focus: 'series' }
      },
      {
        name: 'Failed / Pending',
        type: 'bar',
        stack: 'calls',
        data: top.map((item) => item.failedCalls),
        barWidth: 16,
        itemStyle: { borderRadius: [0, 3, 3, 0] },
        emphasis: { focus: 'series' }
      }
    ]
  })
}

function durationSignal() {
  if (!totalCalls.value) return { color: 'default', label: 'No calls' }
  if (averageDurationMs.value >= 60000) return { color: 'warning', label: 'Long average' }
  if (averageDurationMs.value >= 10000) return { color: 'processing', label: 'Moderate average' }
  return { color: 'success', label: 'Fast average' }
}

onMounted(() => {
  load()
})
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="info-block">
        <div class="info-block-title">Tool activity summary</div>
        <div class="info-block-grid">
          <div class="info-stat">
            <div class="info-stat-label">Total calls</div>
            <div class="info-stat-value">{{ formatNumber(totalCalls) }}</div>
          </div>
          <div class="info-stat">
            <div class="info-stat-label">Tools used</div>
            <div class="info-stat-value">{{ formatNumber(toolsUsed) }}</div>
          </div>
          <div class="info-stat">
            <div class="info-stat-label">Failed / pending</div>
            <div class="info-stat-value" :class="failedPendingCalls ? 'status-error' : 'status-ok'">
              {{ formatNumber(failedPendingCalls) }}
            </div>
          </div>
          <div class="info-stat">
            <div class="info-stat-label">Average duration</div>
            <div class="info-stat-value">{{ formatDuration(averageDurationMs) }}</div>
            <div class="metric-note">
              <a-tag :color="durationSignal().color" class="status-tag">{{ durationSignal().label }}</a-tag>
            </div>
          </div>
        </div>
      </section>

      <section class="panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Top Tools</h2>
            <div class="panel-kicker">Successful calls stacked against failed or pending outcomes</div>
          </div>
          <div class="panel-actions">
            <BarChartOutlined class="panel-header-icon" />
            <a-button @click="load">
              <template #icon>
                <ReloadOutlined />
              </template>
              Refresh
            </a-button>
          </div>
        </div>
        <div class="panel-body">
          <div v-if="tools.length" ref="chartEl" class="chart tools-chart"></div>
          <div v-else class="empty-state empty-state-compact">
            <BarChartOutlined class="empty-state-icon" />
            <div class="empty-state-title">No tool activity to chart</div>
            <div class="empty-state-text">Indexed tool calls will appear here as successful and failed or pending totals.</div>
          </div>
        </div>
      </section>
    </div>
  </a-spin>
</template>
