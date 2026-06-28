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
import { api, formatDateTime, formatDuration, formatNumber, sessionLabel, shortPath, type AgentUsage, type ToolCall, type ToolStat } from '../api'
import { useMessages } from '../i18n'
import { sourceDisplay, sourceFilterOptions } from '../presentation/sourceIdentity'
import { statusClass, statusColor } from '../presentation/status'
import { parseToolInput, type ToolInputField } from '../toolInput'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const DEFAULT_SORT = 'recent'
const TOOL_CALL_LIMIT = 500
type ToolCallSort = typeof DEFAULT_SORT | 'duration_desc' | 'duration_asc'

const SHELL_TOOL_NAMES = new Set([
  'bash',
  'cmd',
  'cmd.exe',
  'powershell',
  'powershell.exe',
  'pwsh',
  'pwsh.exe',
  'sh',
  'shell',
  'shell_command',
  'terminal',
  'zsh'
])
const COMMAND_FIELD_KEYS = new Set(['arguments', 'cmd', 'command', 'input', 'script'])

const route = useRoute()
const router = useRouter()
const { t } = useMessages({
  en: {
    'title': 'Shell Commands',
    'kicker': 'Shell and terminal tool calls by source, command, session and project',
    'count.matching': '{count} matching shell commands',
    'count.sorted': '{count} sorted shell commands',
    'count.recent': '{count} recent shell commands',
    'column.started': 'Started',
    'column.agentTool': 'Source / Tool',
    'column.command': 'Command / Input',
    'column.status': 'Status',
    'column.duration': 'Duration',
    'column.session': 'Session',
    'column.project': 'Project',
    'filter.agent': 'Source',
    'filter.tool': 'Shell tool',
    'filter.from': 'From',
    'filter.to': 'To',
    'filter.fromAria': 'Started from',
    'filter.toAria': 'Started to',
    'sort.recent': 'Recent first',
    'sort.durationDesc': 'Duration high to low',
    'sort.durationAsc': 'Duration low to high',
    'action.reset': 'Reset',
    'action.refresh': 'Refresh',
    'label.rawSource': 'raw',
    'empty.loading': 'Loading shell commands...',
    'empty.none': 'No shell command calls indexed',
    'tooltip.viewDetails': 'View details',
    'fallback.unknown': 'unknown',
    'fallback.none': '-'
  },
  'zh-CN': {
    'title': 'Shell \u547d\u4ee4',
    'kicker': '\u6309来源\u3001\u547d\u4ee4\u3001\u4f1a\u8bdd\u548c\u9879\u76ee\u67e5\u770b Shell \u4e0e\u7ec8\u7aef\u5de5\u5177\u8c03\u7528',
    'count.matching': '{count} \u4e2a\u5339\u914d Shell \u547d\u4ee4',
    'count.sorted': '{count} \u4e2a\u5df2\u6392\u5e8f Shell \u547d\u4ee4',
    'count.recent': '{count} \u4e2a\u6700\u8fd1 Shell \u547d\u4ee4',
    'column.started': '\u5f00\u59cb',
    'column.agentTool': '来源 / \u5de5\u5177',
    'column.command': '\u547d\u4ee4 / \u8f93\u5165',
    'column.status': '\u72b6\u6001',
    'column.duration': '\u8017\u65f6',
    'column.session': '\u4f1a\u8bdd',
    'column.project': '\u9879\u76ee',
    'filter.agent': '来源',
    'filter.tool': 'Shell \u5de5\u5177',
    'filter.from': '\u4ece',
    'filter.to': '\u5230',
    'filter.fromAria': '\u5f00\u59cb\u65f6\u95f4\u4ece',
    'filter.toAria': '\u5f00\u59cb\u65f6\u95f4\u5230',
    'sort.recent': '\u6700\u8fd1\u4f18\u5148',
    'sort.durationDesc': '\u8017\u65f6\u4ece\u9ad8\u5230\u4f4e',
    'sort.durationAsc': '\u8017\u65f6\u4ece\u4f4e\u5230\u9ad8',
    'action.reset': '\u91cd\u7f6e',
    'action.refresh': '\u5237\u65b0',
    'label.rawSource': '\u539f\u59cb',
    'empty.loading': '\u6b63\u5728\u52a0\u8f7d Shell \u547d\u4ee4...',
    'empty.none': '\u6682\u65e0\u5df2\u7d22\u5f15 Shell \u547d\u4ee4\u8c03\u7528',
    'tooltip.viewDetails': '\u67e5\u770b\u8be6\u60c5',
    'fallback.unknown': '\u672a\u77e5',
    'fallback.none': '-'
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
  { title: t('column.started'), dataIndex: 'startedAt', key: 'startedAt', width: 132 },
  { title: t('column.agentTool'), dataIndex: 'agentName', key: 'agentTool', width: 180 },
  { title: t('column.command'), dataIndex: 'inputSummary', key: 'command', width: 470 },
  { title: t('column.status'), dataIndex: 'status', key: 'status', width: 105 },
  { title: t('column.duration'), dataIndex: 'durationMs', key: 'duration', width: 96, align: 'right' },
  { title: t('column.session'), dataIndex: 'sessionId', key: 'session', width: 120 },
  { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 260 },
  { title: '', key: 'detail', width: 56, align: 'right' }
])

const shellTools = computed(() => tools.value.filter((item) => isShellToolName(item.toolName)))
const toolOptions = computed(() =>
  [...shellTools.value]
    .sort((left, right) => left.toolName.localeCompare(right.toolName))
    .map((item) => ({
      value: item.toolName,
      label: `${item.toolName || t('fallback.unknown')} (${formatNumber(item.calls)})`
    }))
)
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
const shellCallCountText = computed(() => {
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
    toolCalls.value = await fetchShellToolCalls()
  } finally {
    loading.value = false
    callLoading.value = false
  }
}

async function loadToolCalls() {
  callLoading.value = true
  try {
    toolCalls.value = await fetchShellToolCalls()
  } finally {
    callLoading.value = false
  }
}

async function fetchShellToolCalls() {
  const selectedTools = toolFilter.value ? [toolFilter.value] : shellTools.value.map((item) => item.toolName).filter(Boolean)
  if (!selectedTools.length) return []

  const callGroups = await Promise.all(
    selectedTools.map((tool) =>
      api.listToolCalls({
        ...currentToolCallFilters(),
        tool,
        limit: TOOL_CALL_LIMIT
      })
    )
  )
  return sortedCalls(uniqueCalls(callGroups.flat()).filter((call) => isShellToolName(call.toolName))).slice(0, TOOL_CALL_LIMIT)
}

function uniqueCalls(calls: ToolCall[]) {
  const values = new Map<number, ToolCall>()
  for (const call of calls) values.set(call.id, call)
  return [...values.values()]
}

function sortedCalls(calls: ToolCall[]) {
  return [...calls].sort((left, right) => {
    if (sortFilter.value === 'duration_desc') return (right.durationMs || 0) - (left.durationMs || 0)
    if (sortFilter.value === 'duration_asc') return (left.durationMs || 0) - (right.durationMs || 0)
    return timestampMs(right.startedAt) - timestampMs(left.startedAt)
  })
}

function timestampMs(value?: string) {
  if (!value) return 0
  const parsed = Date.parse(value)
  return Number.isNaN(parsed) ? 0 : parsed
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
    agent: agentFilter.value,
    from: toQueryDateTime(fromFilter.value),
    to: toQueryDateTime(toFilter.value, 'end'),
    sort: sortFilter.value === DEFAULT_SORT ? undefined : sortFilter.value
  }
}

function currentToolFilters() {
  return {
    agent: agentFilter.value
  }
}

function clearMissingToolFilter() {
  if (!toolFilter.value) return false
  if (shellTools.value.some((item) => item.toolName === toolFilter.value)) return false
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
    await router.replace({ path: '/tools/shell', query: currentRouteQuery() })
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

function isShellToolName(toolName?: string) {
  const normalized = normalizeToolName(toolName)
  if (!normalized) return false
  if (SHELL_TOOL_NAMES.has(normalized)) return true
  if (normalized.endsWith('.shell_command') || normalized.includes('shell_command')) return true
  const tokens = normalized.split(/[^a-z0-9]+/).filter(Boolean)
  return tokens.some((token) => SHELL_TOOL_NAMES.has(token))
}

function normalizeToolName(toolName?: string) {
  return (toolName || '').trim().toLowerCase()
}

function commandSummary(call: ToolCall) {
  const parsed = parseToolInput(call)
  const field = parsed.fields.find((item) => isCommandField(item))
  return (field?.value || parsed.rawText || call.inputSummary || '').trim()
}

function commandTooltip(call: ToolCall) {
  const parsed = parseToolInput(call)
  return parsed.tooltip || parsed.rawText || call.inputSummary || t('fallback.none')
}

function inputContext(call: ToolCall) {
  const parsed = parseToolInput(call)
  if (!parsed.fields.length) return ''
  return parsed.fields
    .filter((field) => !isCommandField(field))
    .slice(0, 4)
    .map((field) => `${field.key}: ${field.preview || field.value}`)
    .join('  ')
}

function isCommandField(field: ToolInputField) {
  return COMMAND_FIELD_KEYS.has(field.key.replace(/[-_]/g, '').toLowerCase()) || COMMAND_FIELD_KEYS.has(field.key.toLowerCase())
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

function projectContext(call: ToolCall) {
  const value = call.projectPath || call.rawSourcePath || ''
  return value ? shortPath(value) : t('fallback.none')
}

function projectTooltip(call: ToolCall) {
  return [call.projectPath, call.rawSourcePath].filter(Boolean).join('\n') || t('fallback.none')
}

function rawSourceContext(call: ToolCall) {
  if (!call.rawSourcePath || call.rawSourcePath === call.projectPath) return ''
  return `${t('label.rawSource')}: ${shortPath(call.rawSourcePath)}`
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
      <span class="row-count">{{ shellCallCountText }}</span>
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
        class="dense-table tool-shell-table"
        :columns="callColumns"
        :data-source="toolCalls"
        row-key="id"
        size="small"
        :loading="callLoading"
        :locale="tableLocale"
        :pagination="{ pageSize: 20, showSizeChanger: true }"
        :scroll="{ x: 1450 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'startedAt'">{{ formatDateTime(record.startedAt) }}</template>

          <template v-else-if="column.key === 'agentTool'">
            <a-typography-text class="source-identity-name" :ellipsis="{ tooltip: sourceInfo(record).title }">
              {{ sourceInfo(record).label }}
            </a-typography-text>
            <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
            <div class="tool-shell-meta">
              <a-tag class="model-lite-tag">{{ record.toolName || t('fallback.unknown') }}</a-tag>
            </div>
          </template>

          <template v-else-if="column.key === 'command'">
            <a-tooltip :title="commandTooltip(record)" placement="topLeft">
              <pre class="tool-shell-command mono">{{ commandSummary(record) || t('fallback.none') }}</pre>
            </a-tooltip>
            <div v-if="inputContext(record)" class="tool-shell-input">{{ inputContext(record) }}</div>
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
            <div class="tool-call-session-meta">{{ callSessionLabel(record) }}</div>
          </template>

          <template v-else-if="column.key === 'project'">
            <a-tooltip :title="projectTooltip(record)" placement="topLeft">
              <a-typography-text class="path-cell" :ellipsis="{ tooltip: projectTooltip(record) }">
                {{ projectContext(record) }}
              </a-typography-text>
            </a-tooltip>
            <div v-if="rawSourceContext(record)" class="tool-shell-meta mono">{{ rawSourceContext(record) }}</div>
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

<style scoped>
.tool-shell-table :deep(.ant-table-tbody > tr > td) {
  vertical-align: top;
}

.tool-shell-command {
  display: -webkit-box;
  overflow: hidden;
  max-width: 100%;
  min-height: 34px;
  max-height: 52px;
  margin: 0;
  padding: 6px 7px;
  color: var(--am-text);
  font-size: 11px;
  line-height: 16px;
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--am-surface);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.tool-shell-input,
.tool-shell-meta {
  max-width: 100%;
  margin-top: 4px;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 11px;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
