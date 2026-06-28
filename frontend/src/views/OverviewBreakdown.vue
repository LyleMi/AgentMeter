<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import { TableOutlined } from '@ant-design/icons-vue'
import { formatCost, formatNumber, type AgentUsage, type ModelUsage } from '../api'
import { useMessages } from '../i18n'
import { sourceDisplay, sourceInstanceKey } from '../presentation/sourceIdentity'
import { useOverviewContext } from './overviewContext'

const ATable = AntTable as unknown as DefineComponent

const { overview, loading } = useOverviewContext()
const { t } = useMessages({
  en: {
    'column.model': 'Model',
    'column.agent': 'Source',
    'column.sessions': 'Sessions',
    'column.tokens': 'Tokens',
    'column.tools': 'Tools',
    'column.cost': 'Cost',
    'model.title': 'Model Usage',
    'model.kicker': 'Ranked by token volume',
    'model.emptyTitle': 'No model usage yet',
    'model.emptyText': 'Model rankings will appear after at least one session is indexed.',
    'model.unpricedNote': '{count} model entries need pricing coverage.',
    'agent.title': 'Source Usage',
    'agent.kicker': 'Sessions grouped by local source instance',
    'agent.emptyTitle': 'No source usage yet',
    'agent.emptyText': 'Source rankings will appear after at least one session is indexed.',
    'fallback.unknown': 'unknown',
    'fallback.unpriced': 'unpriced'
  },
  'zh-CN': {
    'column.model': '模型',
    'column.agent': '来源',
    'column.sessions': '会话',
    'column.tokens': 'Token',
    'column.tools': '工具',
    'column.cost': '费用',
    'model.title': '模型用量',
    'model.kicker': '按 Token 用量排序',
    'model.emptyTitle': '暂无模型用量',
    'model.emptyText': '至少索引一个会话后会显示模型排行。',
    'model.unpricedNote': '{count} 个模型条目需要价格覆盖。',
    'agent.title': '来源用量',
    'agent.kicker': '按本地来源实例分组的会话',
    'agent.emptyTitle': '暂无来源用量',
    'agent.emptyText': '至少索引一个会话后会显示来源排行。',
    'fallback.unknown': '未知',
    'fallback.unpriced': '未定价'
  }
})

const rankedModelUsage = computed(() =>
  [...(overview.value?.modelUsage || [])].sort((left, right) => right.totalTokens - left.totalTokens)
)
const rankedAgentUsage = computed(() =>
  [...(overview.value?.agentUsage || [])].sort((left, right) => right.sessionCount - left.sessionCount)
)
const hasModelUsage = computed(() => rankedModelUsage.value.length > 0)
const hasAgentUsage = computed(() => rankedAgentUsage.value.length > 0)
const unpricedModelCount = computed(() => rankedModelUsage.value.filter((item) => item.unpriced).length)

const modelColumns = computed(() => [
  { title: t('column.model'), dataIndex: 'model', key: 'model' },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessionCount', width: 96, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'totalTokens', width: 132, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 118, align: 'right' }
])

const agentColumns = computed(() => [
  { title: t('column.agent'), dataIndex: 'sourceLabel', key: 'agent' },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessionCount', width: 96, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'totalTokens', width: 132, align: 'right' },
  { title: t('column.tools'), dataIndex: 'toolCalls', key: 'tools', width: 90, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 118, align: 'right' }
])

function modelRow(record: ModelUsage) {
  return { class: record.unpriced ? 'overview-model-row is-unpriced-row' : 'overview-model-row' }
}

function agentRow(record: AgentUsage) {
  return { class: record.unpriced ? 'overview-model-row is-unpriced-row' : 'overview-model-row' }
}

function agentSource(record: AgentUsage) {
  return sourceDisplay(record, t('fallback.unknown'))
}

function agentRowKey(record: AgentUsage) {
  return sourceInstanceKey(record, t('fallback.unknown'))
}
</script>

<template>
  <a-spin :spinning="loading">
    <div class="overview-breakdown-grid">
      <section class="panel overview-model-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('model.title') }}</h2>
            <div class="panel-kicker">{{ t('model.kicker') }}</div>
          </div>
          <TableOutlined class="panel-header-icon" />
        </div>
        <a-table
          v-if="hasModelUsage"
          class="overview-model-table"
          size="small"
          :columns="modelColumns"
          :data-source="rankedModelUsage"
          :pagination="false"
          row-key="model"
          :custom-row="modelRow"
        >
          <template #bodyCell="{ column, record, index }">
            <template v-if="column.key === 'model'">
              <div class="model-rank-cell">
                <span class="model-rank">{{ index + 1 }}</span>
                <span class="model-name">{{ record.model || t('fallback.unknown') }}</span>
                <a-tag v-if="record.unpriced" class="model-status-tag" color="warning">{{ t('fallback.unpriced') }}</a-tag>
              </div>
            </template>
            <template v-else-if="column.key === 'sessionCount'">
              <span class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
            </template>
            <template v-else-if="column.key === 'totalTokens'">
              <span class="number-cell">{{ formatNumber(record.totalTokens) }}</span>
            </template>
            <template v-else-if="column.key === 'cost'">
              <span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span>
            </template>
          </template>
        </a-table>
        <div v-else class="empty-state empty-state-compact">
          <TableOutlined class="empty-state-icon" />
          <div class="empty-state-title">{{ t('model.emptyTitle') }}</div>
          <div class="empty-state-text">{{ t('model.emptyText') }}</div>
        </div>
        <div v-if="unpricedModelCount > 0" class="panel-footer-note status-warning">
          {{ t('model.unpricedNote', { count: formatNumber(unpricedModelCount) }) }}
        </div>
      </section>

      <section class="panel overview-model-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('agent.title') }}</h2>
            <div class="panel-kicker">{{ t('agent.kicker') }}</div>
          </div>
          <TableOutlined class="panel-header-icon" />
        </div>
        <a-table
          v-if="hasAgentUsage"
          class="overview-model-table"
          size="small"
          :columns="agentColumns"
          :data-source="rankedAgentUsage"
          :pagination="false"
          :row-key="agentRowKey"
          :custom-row="agentRow"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'agent'">
              <div class="source-identity-cell">
                <span class="source-identity-name">{{ agentSource(record).label }}</span>
                <a-tag v-if="record.unpriced" class="model-status-tag" color="warning">{{ t('fallback.unpriced') }}</a-tag>
              </div>
              <div class="source-identity-meta">{{ agentSource(record).secondary || '-' }}</div>
            </template>
            <template v-else-if="column.key === 'sessionCount'">
              <span class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
            </template>
            <template v-else-if="column.key === 'totalTokens'">
              <span class="number-cell">{{ formatNumber(record.totalTokens) }}</span>
            </template>
            <template v-else-if="column.key === 'tools'">
              <span class="number-cell">{{ formatNumber(record.toolCalls) }}</span>
            </template>
            <template v-else-if="column.key === 'cost'">
              <span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span>
            </template>
          </template>
        </a-table>
        <div v-else class="empty-state empty-state-compact">
          <TableOutlined class="empty-state-icon" />
          <div class="empty-state-title">{{ t('agent.emptyTitle') }}</div>
          <div class="empty-state-text">{{ t('agent.emptyText') }}</div>
        </div>
      </section>
    </div>
  </a-spin>
</template>
