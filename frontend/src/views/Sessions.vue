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
  { title: 'Tokens', dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120 },
  { title: 'Cost', dataIndex: 'estimatedCostUsd', key: 'cost', width: 120 },
  { title: 'Tools', dataIndex: 'toolCallCount', key: 'tools', width: 90 },
  { title: 'Wall', dataIndex: 'wallDurationMs', key: 'wall', width: 100 },
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

function sessionRow(record: Session) {
  return { onClick: () => router.push(`/sessions/${record.id}`) }
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
        <div class="toolbar">
          <div class="toolbar-left">
            <a-input
              v-model:value="search"
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
              style="width: 180px"
              allow-clear
              placeholder="Model"
              :options="modelOptions"
              @change="load"
            />
            <a-button type="primary" @click="load">Apply</a-button>
            <a-button @click="resetFilters">Reset</a-button>
          </div>
          <div class="toolbar-right muted">{{ formatNumber(sessions.length) }} rows</div>
        </div>

        <a-table
          :columns="columns"
          :data-source="sessions"
          :loading="loading"
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
              <a-typography-text :ellipsis="{ tooltip: record.projectPath }">
                {{ shortPath(record.projectPath) }}
              </a-typography-text>
            </template>
            <template v-else-if="column.key === 'tokens'">
              {{ formatNumber(record.tokenUsage.totalTokens) }}
            </template>
            <template v-else-if="column.key === 'cost'">
              {{ formatCost(record.estimatedCostUsd) }}
            </template>
            <template v-else-if="column.key === 'wall'">
              {{ formatDuration(record.wallDurationMs) }}
            </template>
            <template v-else-if="column.key === 'parse'">
              <span :class="statusClass(record.parseStatus)">{{ record.parseStatus }}</span>
            </template>
          </template>
        </a-table>
      </div>
    </section>
  </div>
</template>
