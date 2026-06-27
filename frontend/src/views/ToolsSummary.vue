<script setup lang="ts">
import { onMounted, ref, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDuration, formatNumber, type ToolStat } from '../api'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const router = useRouter()
const loading = ref(true)
const tools = ref<ToolStat[]>([])

const statColumns = [
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName' },
  { title: 'Calls', dataIndex: 'calls', key: 'calls', width: 120, align: 'right' },
  { title: 'Success', dataIndex: 'successCalls', key: 'success', width: 140, align: 'right' },
  { title: 'Failed / Pending', dataIndex: 'failedCalls', key: 'failed', width: 160, align: 'right' },
  { title: 'Total Duration', dataIndex: 'totalDurationMs', key: 'totalDuration', width: 150, align: 'right' },
  { title: 'Average', dataIndex: 'avgDurationMs', key: 'average', width: 120, align: 'right' }
]

async function load() {
  loading.value = true
  try {
    tools.value = (await api.getTools()) || []
  } finally {
    loading.value = false
  }
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

function selectTool(toolName: string) {
  router.push({ path: '/tools/calls', query: toolName ? { tool: toolName } : {} })
}

function toolStatRow(record: ToolStat) {
  return { class: 'is-clickable-row', onClick: () => selectTool(record.toolName) }
}

onMounted(load)
</script>

<template>
  <section class="panel">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">Tool Summary</h2>
        <div class="panel-kicker">Status and duration by tool name</div>
      </div>
      <div class="panel-actions">
        <span class="row-count">{{ formatNumber(tools.length) }} tools</span>
        <a-button @click="load">
          <template #icon>
            <ReloadOutlined />
          </template>
          Refresh
        </a-button>
      </div>
    </div>
    <a-table
      class="dense-table tools-table"
      :columns="statColumns"
      :data-source="tools"
      row-key="toolName"
      size="middle"
      :loading="loading"
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
</template>
