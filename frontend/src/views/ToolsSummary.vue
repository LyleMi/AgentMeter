<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDuration, formatNumber, type ToolStat } from '../api'
import Panel from '../components/ui/Panel.vue'
import { useMessages } from '../i18n'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const router = useRouter()
const loading = ref(true)
const tools = ref<ToolStat[]>([])
const { t } = useMessages({
  en: {
    'column.tool': 'Tool',
    'column.calls': 'Calls',
    'column.success': 'Success',
    'column.failed': 'Failed / Pending',
    'column.totalDuration': 'Total Duration',
    'column.average': 'Average',
    'status.noCalls': 'No calls',
    'status.ok': '{rate}% ok',
    'status.clear': 'Clear',
    'status.affected': '{count} affected',
    'title': 'Tool Summary',
    'kicker': 'Status and duration by tool name',
    'rowCount': '{count} tools',
    'action.refresh': 'Refresh',
    'empty.loading': 'Loading tools...',
    'empty.none': 'No tool calls indexed',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    'column.tool': '工具',
    'column.calls': '调用',
    'column.success': '成功',
    'column.failed': '失败 / 未完成',
    'column.totalDuration': '总耗时',
    'column.average': '平均',
    'status.noCalls': '暂无调用',
    'status.ok': '{rate}% 正常',
    'status.clear': '正常',
    'status.affected': '{count} 个受影响',
    'title': '工具汇总',
    'kicker': '按工具名展示状态和耗时',
    'rowCount': '{count} 个工具',
    'action.refresh': '刷新',
    'empty.loading': '正在加载工具...',
    'empty.none': '暂无已索引工具调用',
    'fallback.unknown': '未知'
  }
})

const statColumns = computed(() => [
  { title: t('column.tool'), dataIndex: 'toolName', key: 'toolName' },
  { title: t('column.calls'), dataIndex: 'calls', key: 'calls', width: 120, align: 'right' },
  { title: t('column.success'), dataIndex: 'successCalls', key: 'success', width: 140, align: 'right' },
  { title: t('column.failed'), dataIndex: 'failedCalls', key: 'failed', width: 160, align: 'right' },
  { title: t('column.totalDuration'), dataIndex: 'totalDurationMs', key: 'totalDuration', width: 150, align: 'right' },
  { title: t('column.average'), dataIndex: 'avgDurationMs', key: 'average', width: 120, align: 'right' }
])
const tableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.none') }))

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
  if (!record.calls) return { color: 'default', label: t('status.noCalls') }
  if (rate >= 99) return { color: 'success', label: t('status.ok', { rate }) }
  if (rate >= 90) return { color: 'warning', label: t('status.ok', { rate }) }
  return { color: 'error', label: t('status.ok', { rate }) }
}

function failureStatus(record: ToolStat) {
  if (!record.failedCalls) return { color: 'success', label: t('status.clear') }
  const rate = Math.round((record.failedCalls / Math.max(record.calls, 1)) * 100)
  return {
    color: rate >= 10 ? 'error' : 'warning',
    label: t('status.affected', { count: formatNumber(record.failedCalls) })
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
  <Panel :title="t('title')" :kicker="t('kicker')">
    <template #actions>
      <span class="row-count">{{ t('rowCount', { count: formatNumber(tools.length) }) }}</span>
      <a-button @click="load">
        <template #icon>
          <ReloadOutlined />
        </template>
        {{ t('action.refresh') }}
      </a-button>
    </template>
    <a-table
      class="dense-table tools-table"
      :columns="statColumns"
      :data-source="tools"
      row-key="toolName"
      size="middle"
      :loading="loading"
      :locale="tableLocale"
      :pagination="{ pageSize: 20, showSizeChanger: true }"
      :scroll="{ x: 900 }"
      :custom-row="toolStatRow"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'toolName'">
          <a-typography-text :ellipsis="{ tooltip: record.toolName }">
            {{ record.toolName || t('fallback.unknown') }}
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
  </Panel>
</template>
