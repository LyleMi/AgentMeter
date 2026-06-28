<script setup lang="ts">
import type { DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import AntTable from 'ant-design-vue/es/table'
import Typography from 'ant-design-vue/es/typography'
import { ArrowRightOutlined, ClockCircleOutlined } from '@ant-design/icons-vue'
import {
  formatDateTime,
  formatDuration,
  projectDisplay,
  sessionFullLabel,
  sessionLabel,
  type Session
} from '../../api'
import { sourceDisplay } from '../../presentation/sourceIdentity'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const props = defineProps<{
  title: string
  kicker: string
  emptyTitle: string
  emptyText: string
  openLabel: string
  columns: unknown[]
  rows: Session[]
  hasRows: boolean
  fallbackUnknown: string
  openSession: (id: number) => void
  rowProps: (record: Session) => Record<string, unknown>
}>()

function sourceInfo(record: Session) {
  return sourceDisplay(record, props.fallbackUnknown)
}

function projectInfo(record: Session) {
  return projectDisplay(record.projectPath)
}
</script>

<template>
  <section class="panel overview-time-panel">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ title }}</h2>
        <div class="panel-kicker">{{ kicker }}</div>
      </div>
      <ClockCircleOutlined class="panel-header-icon" />
    </div>
    <a-table
      v-if="hasRows"
      class="overview-session-table overview-time-table"
      size="small"
      :columns="columns"
      :data-source="rows"
      :pagination="false"
      row-key="id"
      :custom-row="rowProps"
      :scroll="{ x: 1150 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'project'">
          <div class="overview-session-identity">
            <a-typography-text class="overview-session-project" :ellipsis="{ tooltip: record.projectPath }">
              {{ projectInfo(record).main }}
            </a-typography-text>
            <a-typography-text class="overview-session-meta mono" :ellipsis="{ tooltip: sessionFullLabel(record) }">
              {{ sessionLabel(record) }}
            </a-typography-text>
          </div>
        </template>
        <template v-else-if="column.key === 'agent'">
          <div class="source-identity-cell">
            <span class="source-identity-name">{{ sourceInfo(record).label }}</span>
          </div>
          <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
        </template>
        <template v-else-if="column.key === 'model'">
          <a-typography-text class="model-name" :ellipsis="{ tooltip: record.model }">
            {{ record.model || fallbackUnknown }}
          </a-typography-text>
        </template>
        <template v-else-if="column.key === 'wall'">
          <span class="number-cell">{{ formatDuration(record.wallDurationMs) }}</span>
        </template>
        <template v-else-if="column.key === 'active'">
          <span class="number-cell">{{ formatDuration(record.activeDurationMs) }}</span>
        </template>
        <template v-else-if="column.key === 'modelTime'">
          <span class="number-cell">{{ formatDuration(record.modelDurationMs) }}</span>
        </template>
        <template v-else-if="column.key === 'toolTime'">
          <span class="number-cell">{{ formatDuration(record.toolDurationMs) }}</span>
        </template>
        <template v-else-if="column.key === 'started'">
          {{ formatDateTime(record.startedAt) }}
        </template>
        <template v-else-if="column.key === 'open'">
          <a-button type="text" size="small" :aria-label="openLabel" @click.stop="openSession(record.id)">
            <template #icon>
              <ArrowRightOutlined />
            </template>
          </a-button>
        </template>
      </template>
    </a-table>
    <div v-else class="empty-state empty-state-compact">
      <ClockCircleOutlined class="empty-state-icon" />
      <div class="empty-state-title">{{ emptyTitle }}</div>
      <div class="empty-state-text">{{ emptyText }}</div>
    </div>
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
