<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import { TableOutlined } from '@ant-design/icons-vue'
import { formatCost, formatNumber, type AgentUsage, type ModelUsage } from '../api'
import { useOverviewContext } from './overviewContext'

const ATable = AntTable as unknown as DefineComponent

const { overview, loading } = useOverviewContext()

const rankedModelUsage = computed(() =>
  [...(overview.value?.modelUsage || [])].sort((left, right) => right.totalTokens - left.totalTokens)
)
const rankedAgentUsage = computed(() =>
  [...(overview.value?.agentUsage || [])].sort((left, right) => right.sessionCount - left.sessionCount)
)
const hasModelUsage = computed(() => rankedModelUsage.value.length > 0)
const hasAgentUsage = computed(() => rankedAgentUsage.value.length > 0)
const unpricedModelCount = computed(() => rankedModelUsage.value.filter((item) => item.unpriced).length)

const modelColumns = [
  { title: 'Model', dataIndex: 'model', key: 'model' },
  { title: 'Sessions', dataIndex: 'sessionCount', key: 'sessionCount', width: 96, align: 'right' },
  { title: 'Tokens', dataIndex: 'totalTokens', key: 'totalTokens', width: 132, align: 'right' },
  { title: 'Cost', dataIndex: 'estimatedCostUsd', key: 'cost', width: 118, align: 'right' }
]

const agentColumns = [
  { title: 'Agent', dataIndex: 'agentName', key: 'agent' },
  { title: 'Sessions', dataIndex: 'sessionCount', key: 'sessionCount', width: 96, align: 'right' },
  { title: 'Tokens', dataIndex: 'totalTokens', key: 'totalTokens', width: 132, align: 'right' },
  { title: 'Tools', dataIndex: 'toolCalls', key: 'tools', width: 90, align: 'right' },
  { title: 'Cost', dataIndex: 'estimatedCostUsd', key: 'cost', width: 118, align: 'right' }
]

function modelRow(record: ModelUsage) {
  return { class: record.unpriced ? 'overview-model-row is-unpriced-row' : 'overview-model-row' }
}

function agentRow(record: AgentUsage) {
  return { class: record.unpriced ? 'overview-model-row is-unpriced-row' : 'overview-model-row' }
}
</script>

<template>
  <a-spin :spinning="loading">
    <div class="overview-breakdown-grid">
      <section class="panel overview-model-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Model Usage</h2>
            <div class="panel-kicker">Ranked by token volume</div>
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
                <span class="model-name">{{ record.model || 'unknown' }}</span>
                <a-tag v-if="record.unpriced" class="model-status-tag" color="warning">unpriced</a-tag>
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
          <div class="empty-state-title">No model usage yet</div>
          <div class="empty-state-text">Model rankings will appear after at least one session is indexed.</div>
        </div>
        <div v-if="unpricedModelCount > 0" class="panel-footer-note status-warning">
          {{ formatNumber(unpricedModelCount) }} model entries need pricing coverage.
        </div>
      </section>

      <section class="panel overview-model-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Agent Usage</h2>
            <div class="panel-kicker">Sessions grouped by local agent source</div>
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
          row-key="agentKind"
          :custom-row="agentRow"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'agent'">
              <div class="model-rank-cell">
                <span class="model-name">{{ record.agentName || record.agentKind || 'unknown' }}</span>
                <a-tag v-if="record.unpriced" class="model-status-tag" color="warning">unpriced</a-tag>
              </div>
              <div class="timeline-event-raw">{{ record.agentKind || '-' }}</div>
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
          <div class="empty-state-title">No agent usage yet</div>
          <div class="empty-state-text">Agent rankings will appear after at least one session is indexed.</div>
        </div>
      </section>
    </div>
  </a-spin>
</template>
