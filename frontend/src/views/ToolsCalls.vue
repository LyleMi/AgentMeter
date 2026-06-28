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
import { useMessages } from '../i18n'
import { sourceDisplay, sourceFilterOptions } from '../presentation/sourceIdentity'
import { statusClass, statusColor } from '../presentation/status'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const DEFAULT_SORT = 'recent'
type ToolCallSort = typeof DEFAULT_SORT | 'duration_desc' | 'duration_asc'

const route = useRoute()
const router = useRouter()
const { t } = useMessages({
  en: {
    'title': 'Recent Tool Calls',
    'kicker': 'Individual calls with parsed input, output, raw events and session context',
    'count.matching': '{count} matching calls',
    'count.sorted': '{count} sorted calls',
    'count.recent': '{count} recent calls',
    'column.started': 'Started',
    'column.tool': 'Tool',
    'column.status': 'Status',
    'column.duration': 'Duration',
    'column.session': 'Session',
    'column.input': 'Input',
    'column.output': 'Output',
    'filter.agent': 'Source',
    'filter.tool': 'Tool',
    'filter.from': 'From',
    'filter.to': 'To',
    'filter.fromAria': 'Started from',
    'filter.toAria': 'Started to',
    'sort.recent': 'Recent first',
    'sort.durationDesc': 'Duration high to low',
    'sort.durationAsc': 'Duration low to high',
    'action.reset': 'Reset',
    'action.refresh': 'Refresh',
    'empty.loading': 'Loading tool calls...',
    'empty.none': 'No tool calls indexed',
    'tooltip.viewDetails': 'View details',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    'title': '最近工具调用',
    'kicker': '包含解析输入、输出、原始事件和会话上下文的单次调用',
    'count.matching': '{count} 个匹配调用',
    'count.sorted': '{count} 个已排序调用',
    'count.recent': '{count} 个最近调用',
    'column.started': '开始',
    'column.tool': '工具',
    'column.status': '状态',
    'column.duration': '耗时',
    'column.session': '会话',
    'column.input': '输入',
    'column.output': '输出',
    'filter.agent': '来源',
    'filter.tool': '工具',
    'filter.from': '从',
    'filter.to': '到',
    'filter.fromAria': '开始时间从',
    'filter.toAria': '开始时间到',
    'sort.recent': '最近优先',
    'sort.durationDesc': '耗时从高到低',
    'sort.durationAsc': '耗时从低到高',
    'action.reset': '重置',
    'action.refresh': '刷新',
    'empty.loading': '正在加载工具调用...',
    'empty.none': '暂无已索引工具调用',
    'tooltip.viewDetails': '查看详情',
    'fallback.unknown': '未知'
  }
})
const loading = ref(true)
const callLoading = ref(true)
const toolLoading = ref(true)
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

const callColumns = computed(() => [
  { title: t('column.started'), dataIndex: 'startedAt', key: 'startedAt', width: 140 },
  { title: t('column.tool'), dataIndex: 'toolName', key: 'toolName', width: 140 },
  { title: t('column.status'), dataIndex: 'status', key: 'status', width: 105 },
  { title: t('column.duration'), dataIndex: 'durationMs', key: 'duration', width: 96, align: 'right' },
  { title: t('column.session'), dataIndex: 'sessionId', key: 'session', width: 96 },
  { title: t('column.input'), dataIndex: 'inputSummary', key: 'input', width: 440 },
  { title: t('column.output'), dataIndex: 'outputSummary', key: 'output', width: 280 },
  { title: '', key: 'detail', width: 56, align: 'right' }
])

const toolOptions = computed(() => tools.value.map((item) => ({ value: item.toolName, label: item.toolName || t('fallback.unknown') })))
const agentOptions = computed(() => {
  return sourceFilterOptions(agents.value, t('fallback.unknown'))
})
const sortOptions = computed(() => [
  { value: DEFAULT_SORT, label: t('sort.recent') },
  { value: 'duration_desc', label: t('sort.durationDesc') },
  { value: 'duration_asc', label: t('sort.durationAsc') }
])
const tableLocale = computed(() => ({ emptyText: callLoading.value ? t('empty.loading') : t('empty.none') }))
const hasActiveFilters = computed(() => Boolean(toolFilter.value || agentFilter.value || fromFilter.value || toFilter.value))
const toolCallCountText = computed(() => {
  const visible = formatNumber(toolCalls.value.length)
  if (hasActiveFilters.value) return t('count.matching', { count: visible })
  if (sortFilter.value !== DEFAULT_SORT) return t('count.sorted', { count: visible })
  return t('count.recent', { count: visible })
})

async function load() {
  loading.value = true
  callLoading.value = true
  try {
    const overviewRequest = api.getOverview()
    const clearedTool = await loadToolOptions(true)
    const overview = await overviewRequest
    if (clearedTool) await replaceRouteQuery()
    agents.value = overview?.agentUsage || []
    toolCalls.value = (await api.listToolCalls(currentToolCallFilters())) || []
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

function currentToolFilters() {
  return {
    agent: agentFilter.value
  }
}

function clearMissingToolFilter() {
  if (!toolFilter.value) return false
  if (tools.value.some((item) => item.toolName === toolFilter.value)) return false
  toolFilter.value = undefined
  return true
}

async function loadToolOptions(clearInvalidTool = false) {
  toolLoading.value = true
  try {
    tools.value = (await api.getTools(currentToolFilters())) || []
    return clearInvalidTool ? clearMissingToolFilter() : false
  } finally {
    toolLoading.value = false
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

async function replaceRouteQuery() {
  applyingRouteUpdate = true
  try {
    await router.replace({ path: '/tools/calls', query: currentRouteQuery() })
  } finally {
    applyingRouteUpdate = false
  }
}

async function updateFilters(changedFilter?: 'agent') {
  if (changedFilter === 'agent') {
    await loadToolOptions(true)
  }
  await replaceRouteQuery()
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
  updateFilters('agent')
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

function sourceInfo(call: ToolCall) {
  return sourceDisplay(call, t('fallback.unknown'))
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
  async () => {
    if (applyingRouteUpdate) return
    syncFiltersFromRoute()
    const clearedTool = await loadToolOptions(true)
    if (clearedTool) await replaceRouteQuery()
    loadToolCalls()
  }
)

onMounted(load)
</script>

<template>
  <section class="panel">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ t('title') }}</h2>
        <div class="panel-kicker">{{ t('kicker') }}</div>
      </div>
      <span class="row-count">{{ toolCallCountText }}</span>
    </div>
    <div class="panel-body">
      <div class="toolbar toolbar-compact">
        <div class="toolbar-left">
          <a-select
            v-model:value="agentFilter"
            class="control-medium"
            allow-clear
            :placeholder="t('filter.agent')"
            :options="agentOptions"
            :loading="loading"
            @change="updateFilters('agent')"
          />
          <a-select
            v-model:value="toolFilter"
            class="control-medium"
            allow-clear
            :placeholder="t('filter.tool')"
            :options="toolOptions"
            :loading="loading || toolLoading"
            @change="() => updateFilters()"
          />
          <label class="inline-field tool-time-filter">
            <span>{{ t('filter.from') }}</span>
            <input v-model="fromFilter" class="native-date-input" type="datetime-local" :aria-label="t('filter.fromAria')" @change="() => updateFilters()" />
          </label>
          <label class="inline-field tool-time-filter">
            <span>{{ t('filter.to') }}</span>
            <input v-model="toFilter" class="native-date-input" type="datetime-local" :aria-label="t('filter.toAria')" @change="() => updateFilters()" />
          </label>
          <a-select
            v-model:value="sortFilter"
            class="control-medium"
            :options="sortOptions"
            @change="() => updateFilters()"
          />
          <a-button @click="resetFilters">{{ t('action.reset') }}</a-button>
        </div>
        <div class="toolbar-right">
          <a-button @click="load">
            <template #icon>
              <ReloadOutlined />
            </template>
            {{ t('action.refresh') }}
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
        :locale="tableLocale"
        :pagination="{ pageSize: 20, showSizeChanger: true }"
        :scroll="{ x: 1350 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'startedAt'">{{ formatDateTime(record.startedAt) }}</template>
          <template v-else-if="column.key === 'toolName'">
            <a-typography-text :ellipsis="{ tooltip: record.toolName }">
              {{ record.toolName || t('fallback.unknown') }}
            </a-typography-text>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tooltip :title="record.error || record.status || t('fallback.unknown')">
              <a-tag class="status-tag call-status-tag" :class="statusClass(record.status)" :color="statusColor(record.status)">
                {{ record.status || t('fallback.unknown') }}
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
            <div class="tool-call-session-meta">{{ sourceInfo(record).label }}</div>
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
            <a-tooltip :title="t('tooltip.viewDetails')">
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
