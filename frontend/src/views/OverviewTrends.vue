<script setup lang="ts">
import { computed, nextTick, onMounted, watch } from 'vue'
import ASpin from 'ant-design-vue/es/spin'
import { BarChartOutlined } from '@ant-design/icons-vue'
import { formatNumber } from '../api'
import { chartPalette, usageChartColors } from '../chartPalette'
import { useEChart } from '../composables/useEChart'
import { useOverviewContext } from './overviewContext'

const { overview, loading } = useOverviewContext()
const { chartEl, getChart, disposeChart } = useEChart()

const hasDailyUsage = computed(() => (overview.value?.dailyUsage?.length || 0) > 0)

async function renderAfterUpdate() {
  await nextTick()
  renderChart()
}

function renderChart() {
  const dailyUsage = overview.value?.dailyUsage || []
  if (!dailyUsage.length) {
    disposeChart()
    return
  }
  const chart = getChart()
  if (!chart) return
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

watch(() => overview.value?.dailyUsage, renderAfterUpdate, { deep: true })

onMounted(() => {
  renderAfterUpdate()
})
</script>

<template>
  <a-spin :spinning="loading">
    <section class="panel overview-chart-panel overview-trend-panel">
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
  </a-spin>
</template>
