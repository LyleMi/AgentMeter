<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
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
import { useMessages } from '../../i18n'
import { sourceDisplay } from '../../presentation/sourceIdentity'
import { useTimeContext } from './timeContext'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const router = useRouter()
const { slowSessions: rows } = useTimeContext()
const { t } = useMessages({
  en: {
    'title': 'Slow sessions',
    'kicker': 'Sessions ranked by wall-clock duration',
    'empty.title': 'No slow sessions yet',
    'empty.text': 'Indexed sessions with wall-time data will appear here.',
    'fallback.unknown': 'unknown',
    'action.open': 'Open session',
    'column.project': 'Project / session',
    'column.source': 'Source',
    'column.model': 'Model',
    'column.wall': 'Wall',
    'column.active': 'Active',
    'column.modelTime': 'Model',
    'column.toolTime': 'Tool',
    'column.started': 'Started',
    'column.open': ''
  },
  'zh-CN': {
    'title': '慢会话',
    'kicker': '按墙钟耗时排序的会话',
    'empty.title': '暂无慢会话',
    'empty.text': '索引包含墙钟耗时数据的会话后会显示在这里。',
    'fallback.unknown': '未知',
    'action.open': '打开会话',
    'column.project': '项目 / 会话',
    'column.source': '来源',
    'column.model': '模型',
    'column.wall': '墙钟',
    'column.active': '活跃',
    'column.modelTime': '模型',
    'column.toolTime': '工具',
    'column.started': '开始',
    'column.open': ''
  }
})

const title = computed(() => t('title'))
const kicker = computed(() => t('kicker'))
const emptyTitle = computed(() => t('empty.title'))
const emptyText = computed(() => t('empty.text'))
const fallbackUnknown = computed(() => t('fallback.unknown'))
const openLabel = computed(() => t('action.open'))
const hasRows = computed(() => rows.value.length > 0)
const columns = computed(() => [
  { title: t('column.project'), key: 'project', fixed: 'left', width: 260 },
  { title: t('column.source'), key: 'agent', width: 220 },
  { title: t('column.model'), key: 'model', dataIndex: 'model', width: 180 },
  { title: t('column.wall'), key: 'wall', dataIndex: 'wallDurationMs', align: 'right', width: 120 },
  { title: t('column.active'), key: 'active', dataIndex: 'activeDurationMs', align: 'right', width: 120 },
  { title: t('column.modelTime'), key: 'modelTime', dataIndex: 'modelDurationMs', align: 'right', width: 120 },
  { title: t('column.toolTime'), key: 'toolTime', dataIndex: 'toolDurationMs', align: 'right', width: 120 },
  { title: t('column.started'), key: 'started', dataIndex: 'startedAt', width: 150 },
  { title: t('column.open'), key: 'open', width: 64, align: 'center' }
])

function openSession(id: number) {
  void router.push(`/sessions/${id}`)
}

function rowProps(record: Session) {
  return { class: 'is-clickable-row', onClick: () => openSession(record.id) }
}

function sourceInfo(record: Session) {
  return sourceDisplay(record, fallbackUnknown.value)
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
