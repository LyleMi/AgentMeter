<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import { BarChartOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDuration, formatNumber, type ToolStat } from '../api'
import { chartPalette, toolChartColors } from '../chartPalette'
import { useEChart } from '../composables/useEChart'
import { useMessages } from '../i18n'

const loading = ref(true)
const tools = ref<ToolStat[]>([])
const { chartEl, getChart, disposeChart } = useEChart()
const { t, locale } = useMessages({
  en: {
    'series.success': 'Successful',
    'series.failed': 'Failed / Pending',
    'signal.none': 'No calls',
    'signal.long': 'Long average',
    'signal.moderate': 'Moderate average',
    'signal.fast': 'Fast average',
    'summary.title': 'Tool activity summary',
    'summary.totalCalls': 'Total calls',
    'summary.toolsUsed': 'Tools used',
    'summary.failedPending': 'Failed / pending',
    'summary.averageDuration': 'Average duration',
    'chart.title': 'Top Tools',
    'chart.kicker': 'Successful calls stacked against failed or pending outcomes',
    'action.refresh': 'Refresh',
    'empty.title': 'No tool activity to chart',
    'empty.text': 'Indexed tool calls will appear here as successful and failed or pending totals.'
  },
  'zh-CN': {
    'series.success': '成功',
    'series.failed': '失败 / 未完成',
    'signal.none': '暂无调用',
    'signal.long': '平均较长',
    'signal.moderate': '平均适中',
    'signal.fast': '平均较快',
    'summary.title': '工具活动汇总',
    'summary.totalCalls': '总调用',
    'summary.toolsUsed': '已使用工具',
    'summary.failedPending': '失败 / 未完成',
    'summary.averageDuration': '平均耗时',
    'chart.title': '热门工具',
    'chart.kicker': '成功调用与失败或未完成结果堆叠展示',
    'action.refresh': '刷新',
    'empty.title': '暂无工具活动可绘制',
    'empty.text': '索引工具调用后，这里会显示成功、失败或未完成的总数。'
  }
})

const totalCalls = computed(() => tools.value.reduce((sum, item) => sum + item.calls, 0))
const toolsUsed = computed(() => tools.value.length)
const failedPendingCalls = computed(() => tools.value.reduce((sum, item) => sum + item.failedCalls, 0))
const totalDurationMs = computed(() => tools.value.reduce((sum, item) => sum + item.totalDurationMs, 0))
const averageDurationMs = computed(() => (totalCalls.value > 0 ? totalDurationMs.value / totalCalls.value : 0))

async function load() {
  loading.value = true
  try {
    tools.value = (await api.getTools()) || []
    renderAfterUpdate()
  } finally {
    loading.value = false
  }
}

async function renderAfterUpdate() {
  await nextTick()
  renderChart()
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
        name: t('series.success'),
        type: 'bar',
        stack: 'calls',
        data: top.map((item) => item.successCalls),
        barWidth: 16,
        itemStyle: { borderRadius: [0, 3, 3, 0] },
        emphasis: { focus: 'series' }
      },
      {
        name: t('series.failed'),
        type: 'bar',
        stack: 'calls',
        data: top.map((item) => item.failedCalls),
        barWidth: 16,
        itemStyle: { borderRadius: [0, 3, 3, 0] },
        emphasis: { focus: 'series' }
      }
    ]
  }, true)
}

function durationSignal() {
  if (!totalCalls.value) return { color: 'default', label: t('signal.none') }
  if (averageDurationMs.value >= 60000) return { color: 'warning', label: t('signal.long') }
  if (averageDurationMs.value >= 10000) return { color: 'processing', label: t('signal.moderate') }
  return { color: 'success', label: t('signal.fast') }
}

watch(locale, renderAfterUpdate)

onMounted(() => {
  load()
})
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="info-block">
        <div class="info-block-title">{{ t('summary.title') }}</div>
        <div class="info-block-grid">
          <div class="info-stat">
            <div class="info-stat-label">{{ t('summary.totalCalls') }}</div>
            <div class="info-stat-value">{{ formatNumber(totalCalls) }}</div>
          </div>
          <div class="info-stat">
            <div class="info-stat-label">{{ t('summary.toolsUsed') }}</div>
            <div class="info-stat-value">{{ formatNumber(toolsUsed) }}</div>
          </div>
          <div class="info-stat">
            <div class="info-stat-label">{{ t('summary.failedPending') }}</div>
            <div class="info-stat-value" :class="failedPendingCalls ? 'status-error' : 'status-ok'">
              {{ formatNumber(failedPendingCalls) }}
            </div>
          </div>
          <div class="info-stat">
            <div class="info-stat-label">{{ t('summary.averageDuration') }}</div>
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
            <h2 class="panel-title">{{ t('chart.title') }}</h2>
            <div class="panel-kicker">{{ t('chart.kicker') }}</div>
          </div>
          <div class="panel-actions">
            <BarChartOutlined class="panel-header-icon" />
            <a-button @click="load">
              <template #icon>
                <ReloadOutlined />
              </template>
              {{ t('action.refresh') }}
            </a-button>
          </div>
        </div>
        <div class="panel-body">
          <div v-if="tools.length" ref="chartEl" class="chart tools-chart"></div>
          <div v-else class="empty-state empty-state-compact">
            <BarChartOutlined class="empty-state-icon" />
            <div class="empty-state-title">{{ t('empty.title') }}</div>
            <div class="empty-state-text">{{ t('empty.text') }}</div>
          </div>
        </div>
      </section>
    </div>
  </a-spin>
</template>
