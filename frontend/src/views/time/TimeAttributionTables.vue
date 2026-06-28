<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import AntTable from 'ant-design-vue/es/table'
import Typography from 'ant-design-vue/es/typography'
import { TableOutlined } from '@ant-design/icons-vue'
import {
  formatDuration,
  formatNumber,
  type AgentTimeUsage,
  type ModelTimeUsage
} from '../../api'
import { useMessages } from '../../i18n'
import { sourceDisplay, sourceInstanceKey } from '../../presentation/sourceIdentity'
import { useTimeContext } from './timeContext'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const { t } = useMessages({
  en: {
    'agent.title': 'Source time attribution',
    'agent.kicker': 'Wall, model, tool, and idle time by indexed source',
    'model.title': 'Model time attribution',
    'model.kicker': 'Wall and active time by model',
    'empty.agent': 'No source time rows',
    'empty.model': 'No model time rows',
    'empty.text': 'Indexed sessions with duration data will appear here.',
    'fallback.unknown': 'unknown',
    'column.source': 'Source',
    'column.model': 'Model',
    'column.sessions': 'Sessions',
    'column.calls': 'Calls',
    'column.tokens': 'Tokens',
    'column.wall': 'Wall',
    'column.active': 'Active',
    'column.modelTime': 'Model',
    'column.tool': 'Tool',
    'column.network': 'Network',
    'column.idle': 'Idle'
  },
  'zh-CN': {
    'agent.title': '来源耗时归因',
    'agent.kicker': '按已索引来源查看墙钟、模型、工具和空闲时间',
    'model.title': '模型耗时归因',
    'model.kicker': '按模型查看墙钟和活跃时间',
    'empty.agent': '暂无来源耗时行',
    'empty.model': '暂无模型耗时行',
    'empty.text': '索引包含耗时数据的会话后会显示在这里。',
    'fallback.unknown': '未知',
    'column.source': '来源',
    'column.model': '模型',
    'column.sessions': '会话',
    'column.calls': '调用',
    'column.tokens': 'Token',
    'column.wall': '墙钟',
    'column.active': '活跃',
    'column.modelTime': '模型',
    'column.tool': '工具',
    'column.network': '网络',
    'column.idle': '空闲'
  }
})

const { rankedAgentTimeUsage: agentRows, rankedModelTimeUsage: modelRows } = useTimeContext()

const agentColumns = computed(() => [
  { title: t('column.source'), key: 'agent', fixed: 'left', width: 220 },
  { title: t('column.sessions'), key: 'sessions', dataIndex: 'sessionCount', align: 'right', width: 90 },
  { title: t('column.calls'), key: 'calls', dataIndex: 'toolCalls', align: 'right', width: 90 },
  { title: t('column.wall'), key: 'wall', dataIndex: 'wallDurationMs', align: 'right', width: 120 },
  { title: t('column.active'), key: 'active', dataIndex: 'activeDurationMs', align: 'right', width: 120 },
  { title: t('column.modelTime'), key: 'modelTime', dataIndex: 'modelDurationMs', align: 'right', width: 120 },
  { title: t('column.tool'), key: 'tool', dataIndex: 'toolDurationMs', align: 'right', width: 120 },
  { title: t('column.network'), key: 'network', dataIndex: 'suspectedNetworkToolDurationMs', align: 'right', width: 120 },
  { title: t('column.idle'), key: 'idle', dataIndex: 'idleDurationMs', align: 'right', width: 120 }
])

const modelColumns = computed(() => [
  { title: t('column.model'), key: 'model', fixed: 'left', width: 220 },
  { title: t('column.sessions'), key: 'sessions', dataIndex: 'sessionCount', align: 'right', width: 90 },
  { title: t('column.tokens'), key: 'tokens', dataIndex: 'totalTokens', align: 'right', width: 110 },
  { title: t('column.wall'), key: 'wall', dataIndex: 'wallDurationMs', align: 'right', width: 120 },
  { title: t('column.active'), key: 'active', dataIndex: 'activeDurationMs', align: 'right', width: 120 },
  { title: t('column.modelTime'), key: 'modelTime', dataIndex: 'modelDurationMs', align: 'right', width: 120 },
  { title: t('column.tool'), key: 'tool', dataIndex: 'toolDurationMs', align: 'right', width: 120 },
  { title: t('column.idle'), key: 'idle', dataIndex: 'idleDurationMs', align: 'right', width: 120 }
])

function agentRowKey(record: AgentTimeUsage) {
  return sourceInstanceKey(record, t('fallback.unknown'))
}

function modelRowKey(record: ModelTimeUsage) {
  return record.model || t('fallback.unknown')
}

function sourceInfo(record: AgentTimeUsage) {
  return sourceDisplay(record, t('fallback.unknown'))
}
</script>

<template>
  <section class="overview-time-attribution-grid">
    <section class="panel overview-time-panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('agent.title') }}</h2>
          <div class="panel-kicker">{{ t('agent.kicker') }}</div>
        </div>
        <TableOutlined class="panel-header-icon" />
      </div>
      <a-table
        v-if="agentRows.length"
        class="dense-table overview-time-table"
        size="small"
        :columns="agentColumns"
        :data-source="agentRows"
        :pagination="false"
        :row-key="agentRowKey"
        :scroll="{ x: 1130 }"
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
        <div class="empty-state-title">{{ t('empty.agent') }}</div>
        <div class="empty-state-text">{{ t('empty.text') }}</div>
      </div>
    </section>

    <section class="panel overview-time-panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('model.title') }}</h2>
          <div class="panel-kicker">{{ t('model.kicker') }}</div>
        </div>
        <TableOutlined class="panel-header-icon" />
      </div>
      <a-table
        v-if="modelRows.length"
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
              {{ record.model || t('fallback.unknown') }}
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
        <div class="empty-state-title">{{ t('empty.model') }}</div>
        <div class="empty-state-text">{{ t('empty.text') }}</div>
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
