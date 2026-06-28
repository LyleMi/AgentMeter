<script setup lang="ts">
import type { DefineComponent } from 'vue'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ToolOutlined } from '@ant-design/icons-vue'
import { formatDuration, formatNumber, type ToolTimeUsage } from '../../api'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

defineProps<{
  title: string
  kicker: string
  networkHint: string
  emptyTitle: string
  emptyText: string
  fallbackUnknown: string
  networkLikelyLabel: string
  notNetworkLabel: string
  columns: unknown[]
  rows: ToolTimeUsage[]
  hasRows: boolean
  rowKey: (record: ToolTimeUsage) => string
}>()
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
