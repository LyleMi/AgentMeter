<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { SearchOutlined } from '@ant-design/icons-vue'
import { api, formatCost, formatDateTime, formatDuration, formatNumber, shortPath, type Session } from '../api'

const router = useRouter()
const loading = ref(false)
const sessions = ref<Session[]>([])
const search = ref('')
const model = ref<string | undefined>()

const columns = [
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 155 },
  { title: 'Project', dataIndex: 'projectPath', key: 'projectPath' },
  { title: 'Model', dataIndex: 'model', key: 'model', width: 120 },
  { title: 'Tokens', dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120, align: 'right' },
  { title: 'Cost', dataIndex: 'estimatedCostUsd', key: 'cost', width: 120, align: 'right' },
  { title: 'Tools', dataIndex: 'toolCallCount', key: 'tools', width: 90, align: 'right' },
  { title: 'Wall', dataIndex: 'wallDurationMs', key: 'wall', width: 100, align: 'right' },
  { title: 'Parse', dataIndex: 'parseStatus', key: 'parse', width: 100 }
]

const modelOptions = computed(() => {
  const values = new Set(sessions.value.map((item) => item.model).filter(Boolean))
  return [...values].sort().map((value) => ({ value, label: value }))
})

async function load() {
  loading.value = true
  try {
    sessions.value = await api.listSessions({ search: search.value, model: model.value, limit: 300 })
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  search.value = ''
  model.value = undefined
  load()
}

function statusClass(status: string) {
  if (status === 'ok') return 'status-ok'
  if (status === 'warning') return 'status-warning'
  return 'status-error'
}

function statusColor(status: string) {
  if (status === 'ok') return 'green'
  if (status === 'warning') return 'orange'
  return 'red'
}

function sessionRow(record: Session) {
  return { class: 'sessions-table-row', onClick: () => router.push(`/sessions/${record.id}`) }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Sessions</h1>
        <div class="page-subtitle">Sortable local history with token, cost and tool-call totals</div>
      </div>
      <a-button @click="load">Refresh</a-button>
    </div>

    <section class="panel">
      <div class="panel-body">
        <div class="toolbar sessions-toolbar">
          <div class="toolbar-left sessions-toolbar-controls">
            <a-input
              v-model:value="search"
              class="sessions-search"
              style="width: 320px"
              allow-clear
              placeholder="Search project, model or file"
              @press-enter="load"
            >
              <template #prefix>
                <SearchOutlined />
              </template>
            </a-input>
            <a-select
              v-model:value="model"
              class="sessions-model-filter"
              style="width: 180px"
              allow-clear
              placeholder="Model"
              :options="modelOptions"
              @change="load"
            />
            <a-button type="primary" @click="load">Apply</a-button>
            <a-button @click="resetFilters">Reset</a-button>
          </div>
          <div class="toolbar-right muted sessions-row-count">{{ formatNumber(sessions.length) }} rows</div>
        </div>

        <a-table
          class="sessions-table"
          :columns="columns"
          :data-source="sessions"
          :loading="loading"
          :locale="{ emptyText: loading ? 'Loading sessions...' : 'No sessions match the current filters' }"
          :pagination="{ pageSize: 20, showSizeChanger: true }"
          row-key="id"
          size="middle"
          :custom-row="sessionRow"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'startedAt'">
              {{ formatDateTime(record.startedAt) }}
            </template>
            <template v-else-if="column.key === 'projectPath'">
              <a-tooltip :title="record.projectPath" placement="topLeft">
                <span class="sessions-project-path">{{ shortPath(record.projectPath) }}</span>
              </a-tooltip>
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
            <template v-else-if="column.key === 'parse'">
              <a-tag class="status-tag parse-status-tag" :class="statusClass(record.parseStatus)" :color="statusColor(record.parseStatus)">
                {{ record.parseStatus }}
              </a-tag>
            </template>
          </template>
        </a-table>
      </div>
    </section>
  </div>
</template>
