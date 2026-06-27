<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import ASelect from 'ant-design-vue/es/select'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import { BarChartOutlined, EyeOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import ToolCallDetailDrawer from '../components/ToolCallDetailDrawer.vue'
import { api, formatDateTime, formatDuration, formatNumber, sessionLabel, shortPath, type ToolCall, type ToolStat } from '../api'
import { chartPalette, toolChartColors } from '../chartPalette'
import { init, type ECharts } from '../chartRuntime'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const router = useRouter()
const loading = ref(true)
const callLoading = ref(true)
const tools = ref<ToolStat[]>([])
const toolCalls = ref<ToolCall[]>([])
const toolFilter = ref<string | undefined>()
const selectedToolCall = ref<ToolCall | null>(null)
const chartEl = ref<HTMLDivElement | null>(null)
let chart: ECharts | null = null

const statColumns = [
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName' },
  { title: 'Calls', dataIndex: 'calls', key: 'calls', width: 120, align: 'right' },
  { title: 'Success', dataIndex: 'successCalls', key: 'success', width: 140, align: 'right' },
  { title: 'Failed / Pending', dataIndex: 'failedCalls', key: 'failed', width: 160, align: 'right' },
  { title: 'Total Duration', dataIndex: 'totalDurationMs', key: 'totalDuration', width: 150, align: 'right' },
  { title: 'Average', dataIndex: 'avgDurationMs', key: 'average', width: 120, align: 'right' }
]

const callColumns = [
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 150 },
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName', width: 150 },
  { title: 'Status', dataIndex: 'status', key: 'status', width: 110 },
  { title: 'Duration', dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
  { title: 'Session', dataIndex: 'sessionKey', key: 'session', width: 190 },
  { title: 'Input', dataIndex: 'inputSummary', key: 'input' },
  { title: 'Output', dataIndex: 'outputSummary', key: 'output' },
  { title: '', key: 'detail', width: 56, align: 'right' }
]

const totalCalls = computed(() => tools.value.reduce((sum, item) => sum + item.calls, 0))
const toolsUsed = computed(() => tools.value.length)
const failedPendingCalls = computed(() => tools.value.reduce((sum, item) => sum + item.failedCalls, 0))
const totalDurationMs = computed(() => tools.value.reduce((sum, item) => sum + item.totalDurationMs, 0))
const averageDurationMs = computed(() => (totalCalls.value > 0 ? totalDurationMs.value / totalCalls.value : 0))
const toolOptions = computed(() => tools.value.map((item) => ({ value: item.toolName, label: item.toolName || 'unknown' })))
const toolCallCountText = computed(() => {
  const visible = formatNumber(toolCalls.value.length)
  if (toolFilter.value) return `${visible} ${toolFilter.value} calls`
  return `${visible} recent calls`
})

async function load() {
  loading.value = true
  callLoading.value = true
  try {
    const [nextTools, nextCalls] = await Promise.all([api.getTools(), api.listToolCalls({ tool: toolFilter.value, limit: 500 })])
    tools.value = nextTools || []
    toolCalls.value = nextCalls || []
    setTimeout(renderChart)
  } finally {
    loading.value = false
    callLoading.value = false
  }
}

async function loadToolCalls() {
  callLoading.value = true
  try {
    toolCalls.value = (await api.listToolCalls({ tool: toolFilter.value, limit: 500 })) || []
  } finally {
    callLoading.value = false
  }
}

function renderChart() {
  if (!chartEl.value || tools.value.length === 0) {
    chart?.dispose()
    chart = null
    return
  }
  if (!chart) chart = init(chartEl.value)
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

function successRate(record: ToolStat) {
  if (!record.calls) return 0
  return Math.round((record.successCalls / record.calls) * 100)
}

function successStatus(record: ToolStat) {
  const rate = successRate(record)
  if (!record.calls) return { color: 'default', label: 'No calls' }
  if (rate >= 99) return { color: 'success', label: `${rate}% ok` }
  if (rate >= 90) return { color: 'warning', label: `${rate}% ok` }
  return { color: 'error', label: `${rate}% ok` }
}

function failureStatus(record: ToolStat) {
  if (!record.failedCalls) return { color: 'success', label: 'Clear' }
  const rate = Math.round((record.failedCalls / Math.max(record.calls, 1)) * 100)
  return {
    color: rate >= 10 ? 'error' : 'warning',
    label: `${formatNumber(record.failedCalls)} affected`
  }
}

function normalizedStatus(status?: string) {
  return (status || 'unknown').toLowerCase()
}

function statusClass(status?: string) {
  const normalized = normalizedStatus(status)
  if (['completed', 'ok', 'indexed', 'success'].includes(normalized)) return 'status-ok'
  if (['pending', 'warning', 'scanning', 'unknown', 'started'].includes(normalized)) return 'status-warning'
  return 'status-error'
}

function statusColor(status?: string) {
  const normalized = normalizedStatus(status)
  if (['completed', 'ok', 'indexed', 'success'].includes(normalized)) return 'success'
  if (normalized === 'scanning') return 'processing'
  if (['pending', 'warning', 'unknown', 'started'].includes(normalized)) return 'warning'
  return 'error'
}

function durationSignal() {
  if (!totalCalls.value) return { color: 'default', label: 'No calls' }
  if (averageDurationMs.value >= 60000) return { color: 'warning', label: 'Long average' }
  if (averageDurationMs.value >= 10000) return { color: 'processing', label: 'Moderate average' }
  return { color: 'success', label: 'Fast average' }
}

function resize() {
  chart?.resize()
}

function selectTool(toolName: string) {
  toolFilter.value = toolName || undefined
  loadToolCalls()
}

function resetToolFilter() {
  toolFilter.value = undefined
  loadToolCalls()
}

function toolStatRow(record: ToolStat) {
  return { class: 'is-clickable-row', onClick: () => selectTool(record.toolName) }
}

function callSessionLabel(call: ToolCall) {
  return sessionLabel({ id: call.sessionId, sessionKey: call.sessionKey || '', codexSessionId: call.codexSessionId })
}

function openToolCall(call: ToolCall) {
  selectedToolCall.value = call
}

function closeToolCall() {
  selectedToolCall.value = null
}

function openSession(id: number) {
  router.push(`/sessions/${id}`)
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
      <a-button @click="load">
        <template #icon>
          <ReloadOutlined />
        </template>
        Refresh
      </a-button>
    </div>

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
            <BarChartOutlined class="panel-header-icon" />
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

        <section class="panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">Tool Summary</h2>
              <div class="panel-kicker">Status and duration by tool name</div>
            </div>
            <span class="row-count">{{ formatNumber(tools.length) }} tools</span>
          </div>
          <a-table
            class="dense-table tools-table"
            :columns="statColumns"
            :data-source="tools"
            row-key="toolName"
            size="middle"
            :locale="{ emptyText: loading ? 'Loading tools...' : 'No tool calls indexed' }"
            :pagination="{ pageSize: 20, showSizeChanger: true }"
            :scroll="{ x: 900 }"
            :custom-row="toolStatRow"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'toolName'">
                <a-typography-text :ellipsis="{ tooltip: record.toolName }">
                  {{ record.toolName || 'unknown' }}
                </a-typography-text>
              </template>
              <template v-else-if="column.key === 'calls'">
                <span class="number-cell">{{ formatNumber(record.calls) }}</span>
              </template>
              <template v-else-if="column.key === 'success'">
                <div class="status-number-cell">
                  <a-tag :color="successStatus(record).color" class="status-tag">
                    {{ successStatus(record).label }}
                  </a-tag>
                  <span class="number-cell muted">{{ formatNumber(record.successCalls) }}</span>
                </div>
              </template>
              <template v-else-if="column.key === 'failed'">
                <div class="status-number-cell">
                  <a-tag :color="failureStatus(record).color" class="status-tag">
                    {{ failureStatus(record).label }}
                  </a-tag>
                  <span v-if="record.failedCalls" class="number-cell status-error">
                    {{ formatNumber(record.failedCalls) }}
                  </span>
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

        <section class="panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">Recent Tool Calls</h2>
              <div class="panel-kicker">Individual calls with input, output, raw events and session context</div>
            </div>
            <span class="row-count">{{ toolCallCountText }}</span>
          </div>
          <div class="panel-body">
            <div class="toolbar toolbar-compact">
              <div class="toolbar-left">
                <a-select
                  v-model:value="toolFilter"
                  class="control-medium"
                  allow-clear
                  placeholder="Tool"
                  :options="toolOptions"
                  @change="loadToolCalls"
                />
                <a-button @click="resetToolFilter">Reset</a-button>
              </div>
              <div class="toolbar-right">
                <a-button @click="loadToolCalls">
                  <template #icon>
                    <ReloadOutlined />
                  </template>
                  Refresh Calls
                </a-button>
              </div>
            </div>
            <a-table
              class="dense-table tool-call-detail-table"
              :columns="callColumns"
              :data-source="toolCalls"
              row-key="id"
              size="small"
              :loading="callLoading"
              :locale="{ emptyText: callLoading ? 'Loading tool calls...' : 'No tool calls indexed' }"
              :pagination="{ pageSize: 20, showSizeChanger: true }"
              :scroll="{ x: 1250 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'startedAt'">{{ formatDateTime(record.startedAt) }}</template>
                <template v-else-if="column.key === 'toolName'">
                  <a-typography-text :ellipsis="{ tooltip: record.toolName }">
                    {{ record.toolName || 'unknown' }}
                  </a-typography-text>
                </template>
                <template v-else-if="column.key === 'status'">
                  <a-tooltip :title="record.error || record.status || 'unknown'">
                    <a-tag class="status-tag call-status-tag" :class="statusClass(record.status)" :color="statusColor(record.status)">
                      {{ record.status || 'unknown' }}
                    </a-tag>
                  </a-tooltip>
                </template>
                <template v-else-if="column.key === 'duration'">
                  <span class="number-cell">{{ formatDuration(record.durationMs) }}</span>
                </template>
                <template v-else-if="column.key === 'session'">
                  <a-tooltip :title="record.projectPath || record.rawSourcePath" placement="topLeft">
                    <span class="mono path-cell">{{ callSessionLabel(record) }}</span>
                  </a-tooltip>
                  <div class="timeline-event-raw">{{ shortPath(record.projectPath || record.rawSourcePath || '') }}</div>
                </template>
                <template v-else-if="column.key === 'input'">
                  <a-typography-text :ellipsis="{ tooltip: record.inputSummary }">
                    {{ record.inputSummary || '-' }}
                  </a-typography-text>
                </template>
                <template v-else-if="column.key === 'output'">
                  <a-typography-text :ellipsis="{ tooltip: record.outputSummary || record.error }">
                    {{ record.outputSummary || record.error || '-' }}
                  </a-typography-text>
                </template>
                <template v-else-if="column.key === 'detail'">
                  <a-tooltip title="View details">
                    <a-button type="text" size="small" @click="openToolCall(record)">
                      <template #icon>
                        <EyeOutlined />
                      </template>
                    </a-button>
                  </a-tooltip>
                </template>
              </template>
            </a-table>
          </div>
        </section>
      </div>
    </a-spin>

    <ToolCallDetailDrawer
      :open="Boolean(selectedToolCall)"
      :call="selectedToolCall"
      @close="closeToolCall"
      @open-session="openSession"
    />
  </div>
</template>
