<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import ASelect from 'ant-design-vue/es/select'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import { ArrowRightOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons-vue'
import { api, formatCost, formatDateTime, formatDuration, formatNumber, sessionLabel, shortPath, type Session } from '../api'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const router = useRouter()
const loading = ref(false)
const sessions = ref<Session[]>([])
const catalogSessions = ref<Session[]>([])
const search = ref('')
const model = ref<string | undefined>()
const agent = ref<string | undefined>()

const columns = [
  { title: 'Session', dataIndex: 'sessionKey', key: 'identity', width: 250 },
  { title: 'Agent', dataIndex: 'agentName', key: 'agent', width: 132 },
  { title: 'Project', dataIndex: 'projectPath', key: 'projectPath' },
  { title: 'Model', dataIndex: 'model', key: 'model', width: 90 },
  { title: 'Tokens', dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 100, align: 'right' },
  { title: 'Cost', dataIndex: 'estimatedCostUsd', key: 'cost', width: 100, align: 'right' },
  { title: 'Tools', dataIndex: 'toolCallCount', key: 'tools', width: 70, align: 'right' },
  { title: 'Wall', dataIndex: 'wallDurationMs', key: 'wall', width: 76, align: 'right' },
  { title: 'Status', dataIndex: 'parseStatus', key: 'status', width: 170 },
  { title: '', key: 'open', width: 44, align: 'right' }
]

const hasActiveFilters = computed(() => Boolean(search.value.trim() || model.value || agent.value))

const modelOptions = computed(() => {
  const source = catalogSessions.value.length ? catalogSessions.value : sessions.value
  const values = new Set(source.map((item) => item.model).filter(Boolean))
  return [...values].sort().map((value) => ({ value, label: value }))
})

const agentOptions = computed(() => {
  const source = catalogSessions.value.length ? catalogSessions.value : sessions.value
  const values = new Map<string, string>()
  for (const item of source) {
    if (item.agentKind) values.set(item.agentKind, item.agentName || item.agentKind)
  }
  return [...values.entries()].sort((left, right) => left[1].localeCompare(right[1])).map(([value, label]) => ({ value, label }))
})

const rowCountText = computed(() => {
  const visible = formatNumber(sessions.value.length)
  const indexed = formatNumber(catalogSessions.value.length || sessions.value.length)
  if (hasActiveFilters.value) return `${visible} matching / ${indexed} indexed`
  return `${visible} indexed`
})

const emptyText = computed(() => {
  if (loading.value) return 'Loading sessions...'
  if (hasActiveFilters.value) return 'No sessions match the current filters'
  return 'No indexed sessions yet. Configure a source and run indexing to populate local history.'
})

async function load() {
  loading.value = true
  try {
    const filters = { search: search.value.trim() || undefined, model: model.value, agent: agent.value, limit: 300 }
    const filtered = api.listSessions(filters)
    const catalog = hasActiveFilters.value ? api.listSessions({ limit: 300 }) : filtered
    const [nextSessions, nextCatalog] = await Promise.all([filtered, catalog])
    sessions.value = nextSessions || []
    catalogSessions.value = nextCatalog || []
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  search.value = ''
  model.value = undefined
  agent.value = undefined
  load()
}

function normalizedStatus(status?: string) {
  return (status || 'unknown').toLowerCase()
}

function statusClass(status?: string) {
  const normalized = normalizedStatus(status)
  if (['ok', 'indexed', 'completed', 'success'].includes(normalized)) return 'status-ok'
  if (['warning', 'pending', 'scanning', 'unknown'].includes(normalized)) return 'status-warning'
  return 'status-error'
}

function statusColor(status?: string) {
  const normalized = normalizedStatus(status)
  if (['ok', 'indexed', 'completed', 'success'].includes(normalized)) return 'success'
  if (normalized === 'scanning') return 'processing'
  if (['warning', 'pending', 'unknown'].includes(normalized)) return 'warning'
  return 'error'
}

function indexStatusHint(record: Session) {
  return record.lastIndexedScanMessage || record.rawSourcePath || 'No index message recorded'
}

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function sessionRow(record: Session) {
  return { class: 'sessions-table-row', onClick: () => openSession(record.id) }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Sessions</h1>
        <div class="page-subtitle">Compact local session workbench for traces, pricing, and indexing state</div>
      </div>
      <a-button @click="load">
        <template #icon>
          <ReloadOutlined />
        </template>
        Refresh
      </a-button>
    </div>

    <section class="panel">
      <div class="panel-body">
        <div class="toolbar sessions-toolbar">
          <div class="toolbar-left sessions-toolbar-controls">
            <a-input
              v-model:value="search"
              class="sessions-search control-wide"
              allow-clear
              placeholder="Search project, model or file"
              @press-enter="load"
            >
              <template #prefix>
                <SearchOutlined />
              </template>
            </a-input>
            <a-select
              v-model:value="agent"
              class="sessions-model-filter control-medium"
              allow-clear
              placeholder="Agent"
              :options="agentOptions"
              @change="load"
            />
            <a-select
              v-model:value="model"
              class="sessions-model-filter control-medium"
              allow-clear
              placeholder="Model"
              :options="modelOptions"
              @change="load"
            />
            <a-button type="primary" @click="load">Apply</a-button>
            <a-button @click="resetFilters">Reset</a-button>
          </div>
          <div class="toolbar-right muted sessions-row-count">{{ rowCountText }}</div>
        </div>

        <a-table
          class="sessions-table"
          :columns="columns"
          :data-source="sessions"
          :loading="loading"
          :locale="{ emptyText }"
          :pagination="{ pageSize: 20, showSizeChanger: true }"
          :scroll="{ x: 1080 }"
          table-layout="fixed"
          row-key="id"
          size="small"
          :custom-row="sessionRow"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'identity'">
              <a-typography-text class="mono path-cell" :ellipsis="{ tooltip: sessionLabel(record) }">
                {{ sessionLabel(record) }}
              </a-typography-text>
              <div class="timeline-event-raw">{{ formatDateTime(record.startedAt) }}</div>
            </template>
            <template v-else-if="column.key === 'agent'">
              <a-tag class="model-lite-tag">{{ record.agentName || record.agentKind || 'unknown' }}</a-tag>
              <div class="timeline-event-raw">{{ record.agentKind || '-' }}</div>
            </template>
            <template v-else-if="column.key === 'projectPath'">
              <a-tooltip :title="record.projectPath" placement="topLeft">
                <span class="sessions-project-path">{{ shortPath(record.projectPath) }}</span>
              </a-tooltip>
              <div class="timeline-event-raw">{{ shortPath(record.rawSourcePath) }}</div>
            </template>
            <template v-else-if="column.key === 'model'">
              <a-tooltip :title="record.model" placement="topLeft">
                <span class="model-name">{{ record.model || 'unknown' }}</span>
              </a-tooltip>
              <div class="timeline-event-raw">{{ record.modelProvider || '-' }}</div>
            </template>
            <template v-else-if="column.key === 'tokens'">
              <span class="number-cell">{{ formatNumber(record.tokenUsage.totalTokens) }}</span>
            </template>
            <template v-else-if="column.key === 'cost'">
              <span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span>
            </template>
            <template v-else-if="column.key === 'tools'">
              <span class="number-cell">{{ formatNumber(record.toolCallCount) }}</span>
            </template>
            <template v-else-if="column.key === 'wall'">
              <span class="number-cell">{{ formatDuration(record.wallDurationMs) }}</span>
            </template>
            <template v-else-if="column.key === 'status'">
              <div class="timeline-event-head">
                <a-tag class="status-tag parse-status-tag" :class="statusClass(record.parseStatus)" :color="statusColor(record.parseStatus)">
                  parse {{ record.parseStatus || 'unknown' }}
                </a-tag>
                <a-tooltip :title="indexStatusHint(record)" placement="topLeft">
                  <a-tag
                    class="status-tag parse-status-tag"
                    :class="statusClass(record.lastIndexedScanStatus)"
                    :color="statusColor(record.lastIndexedScanStatus)"
                  >
                    {{ record.lastIndexedScanStatus || 'unknown' }}
                  </a-tag>
                </a-tooltip>
                <a-tag v-if="record.unpriced" class="status-tag model-status-tag" color="warning">unpriced</a-tag>
              </div>
            </template>
            <template v-else-if="column.key === 'open'">
              <a-tooltip title="Open session">
                <a-button type="text" size="small" @click.stop="openSession(record.id)">
                  <template #icon>
                    <ArrowRightOutlined />
                  </template>
                </a-button>
              </a-tooltip>
            </template>
          </template>
        </a-table>
      </div>
    </section>
  </div>
</template>
