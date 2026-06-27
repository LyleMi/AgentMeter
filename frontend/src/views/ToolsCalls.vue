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
import ToolInputInline from '../components/ToolInputInline.vue'
import { api, formatDateTime, formatDuration, formatNumber, sessionLabel, type AgentUsage, type ToolCall, type ToolStat } from '../api'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const DEFAULT_SORT = 'recent'
type ToolCallSort = typeof DEFAULT_SORT | 'duration_desc' | 'duration_asc'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const callLoading = ref(true)
const tools = ref<ToolStat[]>([])
const agents = ref<AgentUsage[]>([])
const toolCalls = ref<ToolCall[]>([])
const toolFilter = ref<string | undefined>(routeStringQuery('tool'))
const agentFilter = ref<string | undefined>(routeStringQuery('agent'))
const fromFilter = ref(routeDateTimeQuery('from'))
const toFilter = ref(routeDateTimeQuery('to'))
const sortFilter = ref<ToolCallSort>(routeSortQuery())
const selectedToolCall = ref<ToolCall | null>(null)
let applyingRouteUpdate = false

const callColumns = [
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 140 },
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName', width: 140 },
  { title: 'Status', dataIndex: 'status', key: 'status', width: 105 },
  { title: 'Duration', dataIndex: 'durationMs', key: 'duration', width: 96, align: 'right' },
  { title: 'Session', dataIndex: 'sessionId', key: 'session', width: 96 },
  { title: 'Input', dataIndex: 'inputSummary', key: 'input', width: 440 },
  { title: 'Output', dataIndex: 'outputSummary', key: 'output', width: 280 },
  { title: '', key: 'detail', width: 56, align: 'right' }
]

const toolOptions = computed(() => tools.value.map((item) => ({ value: item.toolName, label: item.toolName || 'unknown' })))
const agentOptions = computed(() => {
  const values = new Map<string, string>()
  for (const item of agents.value) {
    if (item.agentKind) values.set(item.agentKind, item.agentName || item.agentKind)
  }
  return [...values.entries()].sort((left, right) => left[1].localeCompare(right[1])).map(([value, label]) => ({ value, label }))
})
const sortOptions = [
  { value: DEFAULT_SORT, label: 'Recent first' },
  { value: 'duration_desc', label: 'Duration high to low' },
  { value: 'duration_asc', label: 'Duration low to high' }
]
const hasActiveFilters = computed(() => Boolean(toolFilter.value || agentFilter.value || fromFilter.value || toFilter.value))
const toolCallCountText = computed(() => {
  const visible = formatNumber(toolCalls.value.length)
  if (hasActiveFilters.value) return `${visible} matching calls`
  if (sortFilter.value !== DEFAULT_SORT) return `${visible} sorted calls`
  return `${visible} recent calls`
})

async function load() {
  loading.value = true
  callLoading.value = true
  try {
    const [nextTools, overview, nextCalls] = await Promise.all([api.getTools(), api.getOverview(), api.listToolCalls(currentToolCallFilters())])
    tools.value = nextTools || []
    agents.value = overview?.agentUsage || []
    toolCalls.value = nextCalls || []
  } finally {
    loading.value = false
    callLoading.value = false
  }
}

async function loadToolCalls() {
  callLoading.value = true
  try {
    toolCalls.value = (await api.listToolCalls(currentToolCallFilters())) || []
  } finally {
    callLoading.value = false
  }
}

function routeStringQuery(key: string) {
  const value = route.query[key]
  return typeof value === 'string' && value ? value : undefined
}

function routeDateTimeQuery(key: string) {
  const value = routeStringQuery(key)
  if (!value) return ''
  return value.endsWith('Z') ? toLocalDateTimeInputValue(value) : value
}

function routeSortQuery(): ToolCallSort {
  const value = routeStringQuery('sort')
  if (value === 'duration_desc' || value === 'duration_asc') return value
  return DEFAULT_SORT
}

function toLocalDateTimeInputValue(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function toQueryDateTime(value: string, boundary: 'start' | 'end' = 'start') {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  if (boundary === 'end' && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(value)) {
    date.setSeconds(59, 999)
  }
  return date.toISOString()
}

function currentToolCallFilters() {
  return {
    tool: toolFilter.value,
    agent: agentFilter.value,
    from: toQueryDateTime(fromFilter.value),
    to: toQueryDateTime(toFilter.value, 'end'),
    sort: sortFilter.value === DEFAULT_SORT ? undefined : sortFilter.value,
    limit: 500
  }
}

function setQueryValue(query: Record<string, string>, key: string, value?: string) {
  if (value) query[key] = value
  else delete query[key]
}

function currentRouteQuery() {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (typeof value === 'string') query[key] = value
  }
  setQueryValue(query, 'tool', toolFilter.value)
  setQueryValue(query, 'agent', agentFilter.value)
  setQueryValue(query, 'from', fromFilter.value || undefined)
  setQueryValue(query, 'to', toFilter.value || undefined)
  setQueryValue(query, 'sort', sortFilter.value === DEFAULT_SORT ? undefined : sortFilter.value)
  return query
}

async function updateFilters() {
  applyingRouteUpdate = true
  try {
    await router.replace({ path: '/tools/calls', query: currentRouteQuery() })
  } finally {
    applyingRouteUpdate = false
  }
  loadToolCalls()
}

function syncFiltersFromRoute() {
  toolFilter.value = routeStringQuery('tool')
  agentFilter.value = routeStringQuery('agent')
  fromFilter.value = routeDateTimeQuery('from')
  toFilter.value = routeDateTimeQuery('to')
  sortFilter.value = routeSortQuery()
}

function resetFilters() {
  toolFilter.value = undefined
  agentFilter.value = undefined
  fromFilter.value = ''
  toFilter.value = ''
  sortFilter.value = DEFAULT_SORT
  updateFilters()
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

function callSessionShort(call: ToolCall) {
  return `#${formatNumber(call.sessionId)}`
}

function callSessionTooltip(call: ToolCall) {
  const context = call.projectPath || call.rawSourcePath || ''
  return context ? `${callSessionLabel(call)}\n${context}` : callSessionLabel(call)
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
  () => [route.query.tool, route.query.agent, route.query.from, route.query.to, route.query.sort],
  () => {
    if (applyingRouteUpdate) return
    syncFiltersFromRoute()
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
        <div class="panel-kicker">Individual calls with parsed input, output, raw events and session context</div>
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
            @change="updateFilters"
          />
          <a-select
            v-model:value="agentFilter"
            class="control-medium"
            allow-clear
            placeholder="Agent type"
            :options="agentOptions"
            :loading="loading"
            @change="updateFilters"
          />
          <label class="inline-field tool-time-filter">
            <span>From</span>
            <input v-model="fromFilter" class="native-date-input" type="datetime-local" aria-label="Started from" @change="updateFilters" />
          </label>
          <label class="inline-field tool-time-filter">
            <span>To</span>
            <input v-model="toFilter" class="native-date-input" type="datetime-local" aria-label="Started to" @change="updateFilters" />
          </label>
          <a-select
            v-model:value="sortFilter"
            class="control-medium"
            :options="sortOptions"
            @change="updateFilters"
          />
          <a-button @click="resetFilters">Reset</a-button>
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
        :scroll="{ x: 1350 }"
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
            <a-tooltip :title="callSessionTooltip(record)" placement="topLeft">
              <a-button type="link" size="small" class="tool-call-session-link" @click="openSession(record.sessionId)">
                {{ callSessionShort(record) }}
              </a-button>
            </a-tooltip>
            <div class="tool-call-session-meta">{{ record.agentName || record.agentKind || '-' }}</div>
          </template>
          <template v-else-if="column.key === 'input'">
            <ToolInputInline :call="record" />
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
