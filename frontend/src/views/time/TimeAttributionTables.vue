<script setup lang="ts">
import type { DefineComponent } from 'vue'
import AntTable from 'ant-design-vue/es/table'
import Typography from 'ant-design-vue/es/typography'
import { TableOutlined } from '@ant-design/icons-vue'
import {
  formatDuration,
  formatNumber,
  type AgentTimeUsage,
  type ModelTimeUsage
} from '../../api'
import { sourceDisplay, sourceInstanceKey } from '../../presentation/sourceIdentity'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const props = defineProps<{
  agentTitle: string
  agentKicker: string
  modelTitle: string
  modelKicker: string
  emptyAgentTitle: string
  emptyModelTitle: string
  emptyText: string
  agentColumns: unknown[]
  modelColumns: unknown[]
  agentRows: AgentTimeUsage[]
  modelRows: ModelTimeUsage[]
  hasAgentRows: boolean
  hasModelRows: boolean
  fallbackUnknown: string
}>()

function agentRowKey(record: AgentTimeUsage) {
  return sourceInstanceKey(record, props.fallbackUnknown)
}

function modelRowKey(record: ModelTimeUsage) {
  return record.model || props.fallbackUnknown
}

function sourceInfo(record: AgentTimeUsage) {
  return sourceDisplay(record, props.fallbackUnknown)
}
</script>

<template>
  <section class="overview-time-attribution-grid">
    <section class="panel overview-time-panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ agentTitle }}</h2>
          <div class="panel-kicker">{{ agentKicker }}</div>
        </div>
        <TableOutlined class="panel-header-icon" />
      </div>
      <a-table
        v-if="hasAgentRows"
        class="dense-table overview-time-table"
        size="small"
        :columns="agentColumns"
        :data-source="agentRows"
        :pagination="false"
        :row-key="agentRowKey"
        :scroll="{ x: 930 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'agent'">
            <div class="source-identity-cell">
              <span class="source-identity-name">{{ sourceInfo(record).label }}</span>
            </div>
            <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
          </template>
          <template v-else-if="column.key === 'sessions'">
            <span class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
          </template>
          <template v-else-if="column.key === 'calls'">
            <span class="number-cell">{{ formatNumber(record.toolCalls) }}</span>
          </template>
          <template v-else>
            <span class="number-cell duration-cell">{{ formatDuration(record[column.dataIndex]) }}</span>
          </template>
        </template>
      </a-table>
      <div v-else class="empty-state empty-state-compact">
        <TableOutlined class="empty-state-icon" />
        <div class="empty-state-title">{{ emptyAgentTitle }}</div>
        <div class="empty-state-text">{{ emptyText }}</div>
      </div>
    </section>

    <section class="panel overview-time-panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ modelTitle }}</h2>
          <div class="panel-kicker">{{ modelKicker }}</div>
        </div>
        <TableOutlined class="panel-header-icon" />
      </div>
      <a-table
        v-if="hasModelRows"
        class="dense-table overview-time-table"
        size="small"
        :columns="modelColumns"
        :data-source="modelRows"
        :pagination="false"
        :row-key="modelRowKey"
        :scroll="{ x: 880 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'model'">
            <a-typography-text class="model-name" :ellipsis="{ tooltip: record.model }">
              {{ record.model || fallbackUnknown }}
            </a-typography-text>
          </template>
          <template v-else-if="column.key === 'sessions'">
            <span class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
          </template>
          <template v-else-if="column.key === 'tokens'">
            <span class="number-cell">{{ formatNumber(record.totalTokens) }}</span>
          </template>
          <template v-else>
            <span class="number-cell duration-cell">{{ formatDuration(record[column.dataIndex]) }}</span>
          </template>
        </template>
      </a-table>
      <div v-else class="empty-state empty-state-compact">
        <TableOutlined class="empty-state-icon" />
        <div class="empty-state-title">{{ emptyModelTitle }}</div>
        <div class="empty-state-text">{{ emptyText }}</div>
      </div>
    </section>
  </section>
</template>

<style scoped>
.overview-time-attribution-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(360px, 1fr));
  gap: var(--am-section-gap);
}

.overview-time-panel {
  min-width: 0;
}

.overview-time-table {
  display: block;
}

@media (max-width: 1180px) {
  .overview-time-attribution-grid {
    grid-template-columns: 1fr;
  }
}
</style>
