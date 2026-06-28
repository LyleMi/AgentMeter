<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ToolOutlined } from '@ant-design/icons-vue'
import { formatDuration, formatNumber, type ToolTimeUsage } from '../../api'
import { useMessages } from '../../i18n'
import { useTimeContext } from './timeContext'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const { rankedToolLeaders: rows } = useTimeContext()
const { t } = useMessages({
  en: {
    'title': 'Tool duration leaders',
    'kicker': 'Tools ranked by total measured duration',
    'networkHint': 'Network-likely tools are inferred from tool names and may include web, fetch, browser, and shell network activity.',
    'empty.title': 'No tool duration rows',
    'empty.text': 'Indexed tool calls with duration data will appear here.',
    'fallback.unknown': 'unknown',
    'network.likely': 'Network likely',
    'network.not': 'No',
    'column.tool': 'Tool',
    'column.calls': 'Calls',
    'column.success': 'Success',
    'column.failed': 'Failed',
    'column.total': 'Total',
    'column.average': 'Avg',
    'column.max': 'Max',
    'column.network': 'Network'
  },
  'zh-CN': {
    'title': '工具耗时排行',
    'kicker': '按总测量耗时排序的工具',
    'networkHint': '疑似网络工具基于工具名称推断，可能包含 web、fetch、browser 和 shell 网络活动。',
    'empty.title': '暂无工具耗时行',
    'empty.text': '索引包含耗时数据的工具调用后会显示在这里。',
    'fallback.unknown': '未知',
    'network.likely': '疑似网络',
    'network.not': '否',
    'column.tool': '工具',
    'column.calls': '调用',
    'column.success': '成功',
    'column.failed': '失败',
    'column.total': '总耗时',
    'column.average': '平均',
    'column.max': '最大',
    'column.network': '网络'
  }
})

const columns = computed(() => [
  { title: t('column.tool'), key: 'toolName', fixed: 'left', width: 220 },
  { title: t('column.calls'), key: 'calls', dataIndex: 'calls', align: 'right', width: 90 },
  { title: t('column.success'), key: 'success', dataIndex: 'successCalls', align: 'right', width: 90 },
  { title: t('column.failed'), key: 'failed', dataIndex: 'failedCalls', align: 'right', width: 90 },
  { title: t('column.total'), key: 'total', dataIndex: 'totalDurationMs', align: 'right', width: 120 },
  { title: t('column.average'), key: 'average', dataIndex: 'avgDurationMs', align: 'right', width: 120 },
  { title: t('column.max'), key: 'max', dataIndex: 'maxDurationMs', align: 'right', width: 120 },
  { title: t('column.network'), key: 'network', dataIndex: 'suspectedNetwork', width: 130 }
])

const hasRows = computed(() => rows.value.length > 0)
const fallbackUnknown = computed(() => t('fallback.unknown'))
const title = computed(() => t('title'))
const kicker = computed(() => t('kicker'))
const networkHint = computed(() => t('networkHint'))
const emptyTitle = computed(() => t('empty.title'))
const emptyText = computed(() => t('empty.text'))
const networkLikelyLabel = computed(() => t('network.likely'))
const notNetworkLabel = computed(() => t('network.not'))

function rowKey(record: ToolTimeUsage) {
  return record.toolName || fallbackUnknown.value
}
</script>

<template>
  <section class="panel overview-time-panel">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ title }}</h2>
        <div class="panel-kicker">{{ kicker }}</div>
      </div>
      <ToolOutlined class="panel-header-icon" />
    </div>
    <a-table
      v-if="hasRows"
      class="dense-table overview-time-table"
      size="small"
      :columns="columns"
      :data-source="rows"
      :pagination="false"
      :row-key="rowKey"
      :scroll="{ x: 940 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'toolName'">
          <a-typography-text :ellipsis="{ tooltip: record.toolName }">
            {{ record.toolName || fallbackUnknown }}
          </a-typography-text>
        </template>
        <template v-else-if="column.key === 'calls'">
          <span class="number-cell">{{ formatNumber(record.calls) }}</span>
        </template>
        <template v-else-if="column.key === 'success'">
          <span class="number-cell">{{ formatNumber(record.successCalls) }}</span>
        </template>
        <template v-else-if="column.key === 'failed'">
          <span class="number-cell" :class="{ 'status-error': record.failedCalls > 0 }">{{ formatNumber(record.failedCalls) }}</span>
        </template>
        <template v-else-if="column.key === 'total'">
          <span class="number-cell duration-cell">{{ formatDuration(record.totalDurationMs) }}</span>
        </template>
        <template v-else-if="column.key === 'average'">
          <span class="number-cell duration-cell">{{ formatDuration(record.avgDurationMs) }}</span>
        </template>
        <template v-else-if="column.key === 'max'">
          <span class="number-cell duration-cell">{{ formatDuration(record.maxDurationMs) }}</span>
        </template>
        <template v-else-if="column.key === 'network'">
          <a-tag v-if="record.suspectedNetwork" color="processing" class="status-tag">{{ networkLikelyLabel }}</a-tag>
          <span v-else class="muted">{{ notNetworkLabel }}</span>
        </template>
      </template>
    </a-table>
    <div v-else class="empty-state empty-state-compact">
      <ToolOutlined class="empty-state-icon" />
      <div class="empty-state-title">{{ emptyTitle }}</div>
      <div class="empty-state-text">{{ emptyText }}</div>
    </div>
    <div class="panel-footer-note">{{ networkHint }}</div>
  </section>
</template>

<style scoped>
.overview-time-panel {
  min-width: 0;
}

.overview-time-table {
  display: block;
}
</style>
