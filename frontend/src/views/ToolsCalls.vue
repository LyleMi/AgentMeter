<script setup lang="ts">
import { computed, onMounted, ref, watch, type DefineComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import ASelect from 'ant-design-vue/es/select'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import { EyeOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import ToolCallDetailDrawer from '../components/ToolCallDetailDrawer.vue'
import { api, formatDateTime, formatDuration, formatNumber, sessionLabel, shortPath, type ToolCall, type ToolStat } from '../api'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const callLoading = ref(true)
const tools = ref<ToolStat[]>([])
const toolCalls = ref<ToolCall[]>([])
const toolFilter = ref<string | undefined>(routeToolQuery())
const selectedToolCall = ref<ToolCall | null>(null)

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

function routeToolQuery() {
  const value = route.query.tool
  return typeof value === 'string' && value ? value : undefined
}

function updateToolQuery() {
  const nextTool = toolFilter.value || undefined
  if (routeToolQuery() === nextTool) {
    loadToolCalls()
    return
  }
  const query = { ...route.query }
  if (nextTool) query.tool = nextTool
  else delete query.tool
  router.replace({ path: '/tools/calls', query })
}

function resetToolFilter() {
  toolFilter.value = undefined
  updateToolQuery()
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

watch(
  () => route.query.tool,
  () => {
    const nextTool = routeToolQuery()
    if (toolFilter.value === nextTool) return
    toolFilter.value = nextTool
    loadToolCalls()
  }
)

onMounted(load)
</script>

<template>
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
            :loading="loading"
            @change="updateToolQuery"
          />
          <a-button @click="resetToolFilter">Reset</a-button>
        </div>
        <div class="toolbar-right">
          <a-button @click="load">
            <template #icon>
              <ReloadOutlined />
            </template>
            Refresh
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

    <ToolCallDetailDrawer
      :open="Boolean(selectedToolCall)"
      :call="selectedToolCall"
      @close="closeToolCall"
      @open-session="openSession"
    />
  </section>
</template>
