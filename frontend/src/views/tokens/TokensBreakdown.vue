<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import ASegmented from 'ant-design-vue/es/segmented'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import {
  formatCost,
  formatNumber,
  projectDisplay,
  type AgentUsage,
  type ModelUsage,
  type UsageBreakdownBucket
} from '../../api'
import { useMessages } from '../../i18n'
import { sourceDisplay, sourceInstanceKey } from '../../presentation/sourceIdentity'
import { DEFAULT_BREAKDOWN_GROUP, useTokensContext } from './tokensContext'

const ATable = AntTable as unknown as DefineComponent
const { analytics, loading, breakdownRows, breakdownGroup, updateBreakdownGroup } = useTokensContext()
const { t } = useMessages({
  en: {
    'usage.title': 'Usage Breakdown',
    'usage.kicker': 'Grouped token usage for the current scope',
    'group.global': 'Global',
    'group.agent': 'Source',
    'group.model': 'Model',
    'group.agentModel': 'Source + Model',
    'group.day': 'Day',
    'group.project': 'Project',
    'model.title': 'Model Breakdown',
    'model.kicker': 'Token, cache, reasoning, and cost distribution by model',
    'agent.title': 'Source Breakdown',
    'agent.kicker': 'Token, cache, tools, and cost distribution by local source instance',
    'column.model': 'Model',
    'column.agent': 'Source',
    'column.project': 'Project',
    'column.sessions': 'Sessions',
    'column.tokens': 'Tokens',
    'column.input': 'Input',
    'column.cached': 'Cached',
    'column.output': 'Output',
    'column.reasoning': 'Reasoning',
    'column.contextCompression': 'Compression',
    'column.cacheRate': 'Cache',
    'column.cost': 'Cost',
    'column.tools': 'Tools',
    'fallback.unknown': 'unknown',
    'fallback.unpriced': 'unpriced',
    'empty.loading': 'Loading token analytics...',
    'empty.breakdownLoading': 'Loading usage breakdown...',
    'empty.breakdownNone': 'No usage rows match the current scope',
    'empty.none': 'No token usage indexed yet'
  },
  'zh-CN': {
    'usage.title': '用量拆分',
    'usage.kicker': '按当前范围分组的 Token 用量',
    'group.global': '全局',
    'group.agent': '来源',
    'group.model': '模型',
    'group.agentModel': '来源 + 模型',
    'group.day': '日期',
    'group.project': '项目',
    'model.title': '模型拆分',
    'model.kicker': '按模型展示 Token、缓存、推理和费用分布',
    'agent.title': '来源拆分',
    'agent.kicker': '按本地来源实例展示 Token、缓存、工具和费用分布',
    'column.model': '模型',
    'column.agent': '来源',
    'column.project': '项目',
    'column.sessions': '会话',
    'column.tokens': 'Token',
    'column.input': '输入',
    'column.cached': '缓存',
    'column.output': '输出',
    'column.reasoning': '推理',
    'column.contextCompression': '压缩',
    'column.cacheRate': '缓存',
    'column.cost': '费用',
    'column.tools': '工具',
    'fallback.unknown': '未知',
    'fallback.unpriced': '未定价',
    'empty.loading': '正在加载 Token 分析...',
    'empty.breakdownLoading': '正在加载用量拆分...',
    'empty.breakdownNone': '没有用量行符合当前范围',
    'empty.none': '暂无已索引 Token 用量'
  }
})

const breakdownGroupOptions = computed(() => [
  { value: DEFAULT_BREAKDOWN_GROUP, label: t('group.global') },
  { value: 'agent', label: t('group.agent') },
  { value: 'model', label: t('group.model') },
  { value: 'agent,model', label: t('group.agentModel') },
  { value: 'project', label: t('group.project') },
  { value: 'day', label: t('group.day') }
])

const breakdownColumns = computed(() => [
  { title: breakdownScopeColumnTitle.value, dataIndex: 'sourceLabel', key: 'scope', width: 280 },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 90, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 120, align: 'right' },
  { title: t('column.input'), dataIndex: 'inputTokens', key: 'input', width: 110, align: 'right' },
  { title: t('column.cached'), dataIndex: 'cachedInputTokens', key: 'cached', width: 110, align: 'right' },
  { title: t('column.output'), dataIndex: 'outputTokens', key: 'output', width: 110, align: 'right' },
  { title: t('column.reasoning'), dataIndex: 'reasoningOutputTokens', key: 'reasoning', width: 110, align: 'right' },
  { title: t('column.contextCompression'), dataIndex: 'contextCompressionTokens', key: 'contextCompression', width: 110, align: 'right' },
  { title: t('column.cacheRate'), dataIndex: 'cacheUtilizationRate', key: 'cacheRate', width: 100, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 110, align: 'right' }
])

const breakdownScopeColumnTitle = computed(() => {
  if (breakdownGroup.value === 'agent') return t('column.agent')
  if (breakdownGroup.value === 'model') return t('column.model')
  if (breakdownGroup.value === 'day') return t('group.day')
  if (breakdownGroup.value === 'project') return t('column.project')
  if (breakdownGroup.value === 'agent,model') return t('group.agentModel')
  return t('group.global')
})

const modelColumns = computed(() => [
  { title: t('column.model'), dataIndex: 'model', key: 'model' },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 90, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 120, align: 'right' },
  { title: t('column.cached'), dataIndex: 'cachedInputTokens', key: 'cached', width: 110, align: 'right' },
  { title: t('column.reasoning'), dataIndex: 'reasoningOutputTokens', key: 'reasoning', width: 110, align: 'right' },
  { title: t('column.contextCompression'), dataIndex: 'contextCompressionTokens', key: 'contextCompression', width: 110, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 110, align: 'right' }
])

const agentColumns = computed(() => [
  { title: t('column.agent'), dataIndex: 'sourceLabel', key: 'agent' },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 90, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 120, align: 'right' },
  { title: t('column.cached'), dataIndex: 'cachedInputTokens', key: 'cached', width: 110, align: 'right' },
  { title: t('column.contextCompression'), dataIndex: 'contextCompressionTokens', key: 'contextCompression', width: 110, align: 'right' },
  { title: t('column.tools'), dataIndex: 'toolCalls', key: 'tools', width: 90, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 110, align: 'right' }
])

const tableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.none') }))
const breakdownTableLocale = computed(() => ({
  emptyText: loading.value ? t('empty.breakdownLoading') : t('empty.breakdownNone')
}))

function rowClass(record: ModelUsage | AgentUsage) {
  return { class: record.unpriced ? 'is-unpriced-row' : '' }
}

function breakdownRowClass(record: UsageBreakdownBucket) {
  return { class: record.unpriced ? 'is-unpriced-row' : '' }
}

function formatRate(value: number | undefined) {
  if (!Number.isFinite(value)) return '0%'
  return `${Math.round(Math.max(0, Math.min(1, value || 0)) * 100)}%`
}

function modelName(record: ModelUsage) {
  return record.model || t('fallback.unknown')
}

function sourceInfo(record: AgentUsage) {
  return sourceDisplay(record, t('fallback.unknown'))
}

function breakdownScope(record: UsageBreakdownBucket) {
  if (breakdownGroup.value === DEFAULT_BREAKDOWN_GROUP) {
    return { label: t('group.global'), secondary: '', title: t('group.global') }
  }
  if (breakdownGroup.value === 'model') {
    const label = record.model || t('fallback.unknown')
    return { label, secondary: '', title: label }
  }
  if (breakdownGroup.value === 'day') {
    const label = record.date || t('fallback.unknown')
    return { label, secondary: '', title: label }
  }
  if (breakdownGroup.value === 'project') {
    const project = projectDisplay(record.projectPath)
    return {
      label: project.main,
      secondary: project.collapsed ? project.full : '',
      title: project.full
    }
  }
  const source = sourceDisplay(record, t('fallback.unknown'))
  if (breakdownGroup.value === 'agent,model') {
    const model = record.model || t('fallback.unknown')
    return {
      label: source.label,
      secondary: [model, source.secondary].filter(Boolean).join(' · '),
      title: [source.title, model].filter(Boolean).join('\n')
    }
  }
  return source
}

function breakdownRowKey(record: UsageBreakdownBucket) {
  return [
    breakdownGroup.value,
    record.date,
    record.model,
    record.projectPath,
    sourceInstanceKey(record)
  ].filter(Boolean).join(':')
}

function agentRowKey(record: AgentUsage) {
  return sourceInstanceKey(record, t('fallback.unknown'))
}
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="panel tokens-usage-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('usage.title') }}</h2>
            <div class="panel-kicker">{{ t('usage.kicker') }}</div>
          </div>
          <a-segmented
            class="tokens-breakdown-segmented"
            :value="breakdownGroup"
            :options="breakdownGroupOptions"
            @change="updateBreakdownGroup"
          />
        </div>
        <a-table
          class="dense-table"
          :columns="breakdownColumns"
          :data-source="breakdownRows"
          :loading="loading"
          :locale="breakdownTableLocale"
          :pagination="{ pageSize: 12 }"
          :row-key="breakdownRowKey"
          size="small"
          :custom-row="breakdownRowClass"
          :scroll="{ x: 1270 }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'scope'">
              <span class="source-identity-name" :title="breakdownScope(record).title">{{ breakdownScope(record).label }}</span>
              <div class="source-identity-meta" :title="breakdownScope(record).title">{{ breakdownScope(record).secondary || '-' }}</div>
            </template>
            <template v-else-if="column.key === 'sessions'"><span class="number-cell">{{ formatNumber(record.sessionCount) }}</span></template>
            <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.totalTokens) }}</span></template>
            <template v-else-if="column.key === 'input'"><span class="number-cell">{{ formatNumber(record.inputTokens) }}</span></template>
            <template v-else-if="column.key === 'cached'"><span class="number-cell">{{ formatNumber(record.cachedInputTokens) }}</span></template>
            <template v-else-if="column.key === 'output'"><span class="number-cell">{{ formatNumber(record.outputTokens) }}</span></template>
            <template v-else-if="column.key === 'reasoning'"><span class="number-cell">{{ formatNumber(record.reasoningOutputTokens) }}</span></template>
            <template v-else-if="column.key === 'contextCompression'"><span class="number-cell">{{ formatNumber(record.contextCompressionTokens) }}</span></template>
            <template v-else-if="column.key === 'cacheRate'"><span class="number-cell">{{ formatRate(record.cacheUtilizationRate) }}</span></template>
            <template v-else-if="column.key === 'cost'">
              <span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span>
              <a-tag v-if="record.unpriced" color="warning" class="model-status-tag">{{ t('fallback.unpriced') }}</a-tag>
            </template>
          </template>
        </a-table>
      </section>

      <div class="tokens-breakdown-grid">
        <section class="panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('model.title') }}</h2>
              <div class="panel-kicker">{{ t('model.kicker') }}</div>
            </div>
          </div>
          <a-table
            class="dense-table"
            :columns="modelColumns"
            :data-source="analytics?.modelUsage || []"
            :loading="loading"
            :locale="tableLocale"
            :pagination="{ pageSize: 10 }"
            row-key="model"
            size="small"
            :custom-row="rowClass"
            :scroll="{ x: 870 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'model'">
                <span class="model-name">{{ modelName(record) }}</span>
                <a-tag v-if="record.unpriced" color="warning" class="model-status-tag">{{ t('fallback.unpriced') }}</a-tag>
              </template>
              <template v-else-if="column.key === 'sessions'"><span class="number-cell">{{ formatNumber(record.sessionCount) }}</span></template>
              <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.totalTokens) }}</span></template>
              <template v-else-if="column.key === 'cached'"><span class="number-cell">{{ formatNumber(record.cachedInputTokens) }}</span></template>
              <template v-else-if="column.key === 'reasoning'"><span class="number-cell">{{ formatNumber(record.reasoningOutputTokens) }}</span></template>
              <template v-else-if="column.key === 'contextCompression'"><span class="number-cell">{{ formatNumber(record.contextCompressionTokens) }}</span></template>
              <template v-else-if="column.key === 'cost'"><span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span></template>
            </template>
          </a-table>
        </section>

        <section class="panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('agent.title') }}</h2>
              <div class="panel-kicker">{{ t('agent.kicker') }}</div>
            </div>
          </div>
          <a-table
            class="dense-table"
            :columns="agentColumns"
            :data-source="analytics?.agentUsage || []"
            :loading="loading"
            :locale="tableLocale"
            :pagination="{ pageSize: 10 }"
            :row-key="agentRowKey"
            size="small"
            :custom-row="rowClass"
            :scroll="{ x: 870 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'agent'">
                <span class="source-identity-name">{{ sourceInfo(record).label }}</span>
                <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
              </template>
              <template v-else-if="column.key === 'sessions'"><span class="number-cell">{{ formatNumber(record.sessionCount) }}</span></template>
              <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.totalTokens) }}</span></template>
              <template v-else-if="column.key === 'cached'"><span class="number-cell">{{ formatNumber(record.cachedInputTokens) }}</span></template>
              <template v-else-if="column.key === 'contextCompression'"><span class="number-cell">{{ formatNumber(record.contextCompressionTokens) }}</span></template>
              <template v-else-if="column.key === 'tools'"><span class="number-cell">{{ formatNumber(record.toolCalls) }}</span></template>
              <template v-else-if="column.key === 'cost'"><span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span></template>
            </template>
          </a-table>
        </section>
      </div>
    </div>
  </a-spin>
</template>

<style scoped>
.tokens-usage-panel .panel-header {
  gap: 10px;
  flex-wrap: wrap;
}

.tokens-breakdown-segmented {
  max-width: 100%;
  overflow-x: auto;
}

.tokens-breakdown-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--am-section-gap);
}

@media (max-width: 1200px) {
  .tokens-breakdown-grid {
    grid-template-columns: 1fr;
  }
}
</style>
