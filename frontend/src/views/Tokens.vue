<script setup lang="ts">
import { computed, onMounted, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import AProgress from 'ant-design-vue/es/progress'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import {
  CheckCircleOutlined,
  DatabaseOutlined,
  DollarCircleOutlined,
  HistoryOutlined,
  ReloadOutlined,
  TableOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import { api, formatCost, formatDateTime, formatNumber, sessionLabel, shortPath, type AgentUsage, type ModelUsage, type Session, type TokenAnalytics } from '../api'
import PageHeader from '../components/PageHeader.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const router = useRouter()
const resource = useAsyncResource<TokenAnalytics | null>(null)
const analytics = computed(() => resource.data.value)
const loading = resource.loading
const error = resource.error

const { t } = useMessages({
  en: {
    'title': 'Tokens',
    'subtitle': 'Standalone token usage, cache reuse, and estimated price consumption',
    'action.refresh': 'Refresh',
    'metric.totalTokens': 'Total Tokens',
    'metric.totalTokensNote': '{count} indexed sessions',
    'metric.cacheRate': 'Cache Utilization',
    'metric.cacheRateNote': '{count} cached input tokens',
    'metric.cost': 'Estimated Cost',
    'metric.costNoteCovered': 'Pricing covered for indexed usage',
    'metric.costNoteMissing': '{count} unpriced usage rows',
    'metric.inputOutput': 'Input / Output',
    'metric.inputOutputNote': 'Prompt and response volume',
    'breakdown.title': 'Token Mix',
    'breakdown.kicker': 'Input, cached input, output, and reasoning totals',
    'token.input': 'Input',
    'token.cached': 'Cached',
    'token.output': 'Output',
    'token.reasoning': 'Reasoning',
    'model.title': 'Model Breakdown',
    'model.kicker': 'Token and cost distribution by model',
    'agent.title': 'Agent Breakdown',
    'agent.kicker': 'Token and cost distribution by local agent source',
    'sessions.title': 'High Token Sessions',
    'sessions.kicker': 'Sessions ranked by total token volume',
    'column.model': 'Model',
    'column.agent': 'Agent',
    'column.session': 'Session',
    'column.project': 'Project',
    'column.started': 'Started',
    'column.sessions': 'Sessions',
    'column.tokens': 'Tokens',
    'column.input': 'Input',
    'column.cached': 'Cached',
    'column.output': 'Output',
    'column.reasoning': 'Reasoning',
    'column.cost': 'Cost',
    'column.tools': 'Tools',
    'fallback.unknown': 'unknown',
    'fallback.unpriced': 'unpriced',
    'empty.loading': 'Loading token analytics...',
    'empty.none': 'No token usage indexed yet',
    'error.title': 'Token analytics failed to load'
  },
  'zh-CN': {
    'title': 'Token',
    'subtitle': '独立查看 Token 用量、缓存复用和预估价格消耗',
    'action.refresh': '刷新',
    'metric.totalTokens': '总 Token',
    'metric.totalTokensNote': '已索引 {count} 个会话',
    'metric.cacheRate': '缓存利用率',
    'metric.cacheRateNote': '{count} 个缓存输入 Token',
    'metric.cost': '预估费用',
    'metric.costNoteCovered': '已索引用量均有价格覆盖',
    'metric.costNoteMissing': '{count} 条用量缺少价格',
    'metric.inputOutput': '输入 / 输出',
    'metric.inputOutputNote': '提示词和响应规模',
    'breakdown.title': 'Token 构成',
    'breakdown.kicker': '输入、缓存输入、输出和推理 Token 总数',
    'token.input': '输入',
    'token.cached': '缓存',
    'token.output': '输出',
    'token.reasoning': '推理',
    'model.title': '模型拆分',
    'model.kicker': '按模型展示 Token 和费用分布',
    'agent.title': 'Agent 拆分',
    'agent.kicker': '按本地 Agent 来源展示 Token 和费用分布',
    'sessions.title': '高 Token 会话',
    'sessions.kicker': '按总 Token 用量排序的会话',
    'column.model': '模型',
    'column.agent': 'Agent',
    'column.session': '会话',
    'column.project': '项目',
    'column.started': '开始',
    'column.sessions': '会话',
    'column.tokens': 'Token',
    'column.input': '输入',
    'column.cached': '缓存',
    'column.output': '输出',
    'column.reasoning': '推理',
    'column.cost': '费用',
    'column.tools': '工具',
    'fallback.unknown': '未知',
    'fallback.unpriced': '未定价',
    'empty.loading': '正在加载 Token 分析...',
    'empty.none': '暂无已索引 Token 用量',
    'error.title': 'Token 分析加载失败'
  }
})

const tokenMix = computed(() => {
  const item = analytics.value
  const values = [
    { key: 'input', label: t('token.input'), value: item?.totalInputTokens || 0, tone: 'is-input' },
    { key: 'cached', label: t('token.cached'), value: item?.totalCachedInputTokens || 0, tone: 'is-cached' },
    { key: 'output', label: t('token.output'), value: item?.totalOutputTokens || 0, tone: 'is-output' },
    { key: 'reasoning', label: t('token.reasoning'), value: item?.totalReasoningTokens || 0, tone: 'is-reasoning' }
  ]
  const total = values.reduce((sum, current) => sum + Math.max(current.value, 0), 0)
  return values.map((current) => ({
    ...current,
    share: total > 0 ? current.value / total : 0
  }))
})

const cacheRatePercent = computed(() => Math.round(Math.max(0, analytics.value?.cacheUtilizationRate || 0) * 100))
const metricCards = computed(() => {
  const item = analytics.value
  return [
    {
      label: t('metric.totalTokens'),
      value: formatNumber(item?.totalTokens),
      note: t('metric.totalTokensNote', { count: formatNumber(item?.totalSessions) }),
      icon: DatabaseOutlined,
      tone: 'metric-primary'
    },
    {
      label: t('metric.cacheRate'),
      value: `${cacheRatePercent.value}%`,
      note: t('metric.cacheRateNote', { count: formatNumber(item?.totalCachedInputTokens) }),
      icon: CheckCircleOutlined,
      tone: 'metric-success'
    },
    {
      label: t('metric.cost'),
      value: formatCost(item?.estimatedCostUsd),
      note: item?.unpricedCount ? t('metric.costNoteMissing', { count: formatNumber(item.unpricedCount) }) : t('metric.costNoteCovered'),
      icon: item?.unpricedCount ? WarningOutlined : DollarCircleOutlined,
      tone: item?.unpricedCount ? 'metric-warning' : 'metric-info'
    },
    {
      label: t('metric.inputOutput'),
      value: `${formatNumber(item?.totalInputTokens)} / ${formatNumber(item?.totalOutputTokens)}`,
      note: t('metric.inputOutputNote'),
      icon: TableOutlined,
      tone: 'metric-neutral'
    }
  ]
})

const modelColumns = computed(() => [
  { title: t('column.model'), dataIndex: 'model', key: 'model' },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 90, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 120, align: 'right' },
  { title: t('column.cached'), dataIndex: 'cachedInputTokens', key: 'cached', width: 110, align: 'right' },
  { title: t('column.reasoning'), dataIndex: 'reasoningOutputTokens', key: 'reasoning', width: 110, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 110, align: 'right' }
])
const agentColumns = computed(() => [
  { title: t('column.agent'), dataIndex: 'agentName', key: 'agent' },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 90, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 120, align: 'right' },
  { title: t('column.cached'), dataIndex: 'cachedInputTokens', key: 'cached', width: 110, align: 'right' },
  { title: t('column.tools'), dataIndex: 'toolCalls', key: 'tools', width: 90, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 110, align: 'right' }
])
const sessionColumns = computed(() => [
  { title: t('column.session'), dataIndex: 'sessionKey', key: 'session', width: 220 },
  { title: t('column.project'), dataIndex: 'projectPath', key: 'project' },
  { title: t('column.started'), dataIndex: 'startedAt', key: 'started', width: 140 },
  { title: t('column.tokens'), dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 110, align: 'right' }
])
const tableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.none') }))

function load() {
  return resource.run(() => api.getTokenAnalytics(), { onErrorData: null })
}

function rowClass(record: ModelUsage | AgentUsage) {
  return { class: record.unpriced ? 'is-unpriced-row' : '' }
}

function sessionRow(record: Session) {
  return { class: 'is-clickable-row', onClick: () => router.push(`/sessions/${record.id}`) }
}

function mixWidth(share: number) {
  if (!Number.isFinite(share) || share <= 0) return '0%'
  return `${Math.max(2, Math.round(share * 100))}%`
}

function mixPercent(share: number) {
  if (!Number.isFinite(share) || share <= 0) return '0%'
  if (share < 0.01) return '<1%'
  return `${Math.round(share * 100)}%`
}

function modelName(record: ModelUsage) {
  return record.model || t('fallback.unknown')
}

function agentName(record: AgentUsage) {
  return record.agentName || record.agentKind || t('fallback.unknown')
}

onMounted(load)
</script>

<template>
  <div class="page tokens-page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon>
            <ReloadOutlined />
          </template>
          {{ t('action.refresh') }}
        </a-button>
      </template>
    </PageHeader>

    <a-alert
      v-if="error"
      class="tokens-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <a-spin :spinning="loading">
      <div class="section-stack">
        <section class="metric-strip tokens-metric-strip">
          <div v-for="item in metricCards" :key="item.label" class="metric-strip-item" :class="item.tone">
            <div class="metric-strip-head">
              <span class="metric-label">{{ item.label }}</span>
              <span class="metric-strip-icon">
                <component :is="item.icon" />
              </span>
            </div>
            <div class="metric-strip-value">{{ item.value }}</div>
            <div class="metric-strip-note">{{ item.note }}</div>
          </div>
        </section>

        <section class="panel tokens-mix-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('breakdown.title') }}</h2>
              <div class="panel-kicker">{{ t('breakdown.kicker') }}</div>
            </div>
            <div class="tokens-cache-rate">
              <span>{{ t('metric.cacheRate') }}</span>
              <a-progress type="circle" :percent="cacheRatePercent" :size="58" />
            </div>
          </div>
          <div class="tokens-mix-list">
            <div v-for="item in tokenMix" :key="item.key" class="tokens-mix-item" :class="item.tone">
              <div class="tokens-mix-row">
                <span>{{ item.label }}</span>
                <strong>{{ formatNumber(item.value) }}</strong>
              </div>
              <div class="tokens-mix-meter">
                <span :style="{ width: mixWidth(item.share) }"></span>
              </div>
              <div class="tokens-mix-share">{{ mixPercent(item.share) }}</div>
            </div>
          </div>
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
              :scroll="{ x: 760 }"
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
              row-key="agentKind"
              size="small"
              :custom-row="rowClass"
              :scroll="{ x: 760 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'agent'">
                  <span class="model-name">{{ agentName(record) }}</span>
                  <div class="timeline-event-raw">{{ record.agentKind || '-' }}</div>
                </template>
                <template v-else-if="column.key === 'sessions'"><span class="number-cell">{{ formatNumber(record.sessionCount) }}</span></template>
                <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.totalTokens) }}</span></template>
                <template v-else-if="column.key === 'cached'"><span class="number-cell">{{ formatNumber(record.cachedInputTokens) }}</span></template>
                <template v-else-if="column.key === 'tools'"><span class="number-cell">{{ formatNumber(record.toolCalls) }}</span></template>
                <template v-else-if="column.key === 'cost'"><span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span></template>
              </template>
            </a-table>
          </section>
        </div>

        <section class="panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('sessions.title') }}</h2>
              <div class="panel-kicker">{{ t('sessions.kicker') }}</div>
            </div>
            <HistoryOutlined class="panel-header-icon" />
          </div>
          <a-table
            class="dense-table"
            :columns="sessionColumns"
            :data-source="analytics?.highTokenSessions || []"
            :loading="loading"
            :locale="tableLocale"
            :pagination="{ pageSize: 10 }"
            row-key="id"
            size="small"
            :custom-row="sessionRow"
            :scroll="{ x: 900 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'session'">
                <span class="mono">{{ sessionLabel(record) }}</span>
                <div class="timeline-event-raw">{{ record.agentName || record.agentKind || t('fallback.unknown') }}</div>
              </template>
              <template v-else-if="column.key === 'project'">
                <a-typography-text :ellipsis="{ tooltip: record.projectPath || record.rawSourcePath }">
                  {{ shortPath(record.projectPath || record.rawSourcePath || '') }}
                </a-typography-text>
              </template>
              <template v-else-if="column.key === 'started'">{{ formatDateTime(record.startedAt) }}</template>
              <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.tokenUsage.totalTokens) }}</span></template>
              <template v-else-if="column.key === 'cost'"><span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span></template>
            </template>
          </a-table>
        </section>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.tokens-page {
  max-width: 1560px;
}

.tokens-error {
  margin-bottom: var(--am-section-gap);
}

.tokens-metric-strip {
  grid-template-columns: repeat(4, minmax(160px, 1fr));
}

.tokens-cache-rate {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--am-muted);
  font-size: 12px;
  font-weight: 700;
}

.tokens-mix-list {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.tokens-mix-item {
  --token-accent: var(--am-primary);
  padding: 12px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.tokens-mix-item.is-input {
  --token-accent: var(--am-success);
}

.tokens-mix-item.is-cached {
  --token-accent: var(--am-primary);
}

.tokens-mix-item.is-output {
  --token-accent: var(--am-info);
}

.tokens-mix-item.is-reasoning {
  --token-accent: var(--am-warning);
}

.tokens-mix-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.tokens-mix-row span,
.tokens-mix-share {
  color: var(--am-muted);
  font-size: 12px;
}

.tokens-mix-row strong {
  color: var(--am-text);
  font-size: 16px;
  font-variant-numeric: tabular-nums;
}

.tokens-mix-meter {
  height: 8px;
  margin-top: 10px;
  overflow: hidden;
  background: #e5e7eb;
  border-radius: 999px;
}

.tokens-mix-meter span {
  display: block;
  height: 100%;
  background: var(--token-accent);
}

.tokens-mix-share {
  margin-top: 6px;
}

.tokens-breakdown-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--am-section-gap);
}

@media (max-width: 1200px) {
  .tokens-metric-strip,
  .tokens-mix-list,
  .tokens-breakdown-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .tokens-metric-strip,
  .tokens-mix-list,
  .tokens-breakdown-grid {
    grid-template-columns: 1fr;
  }
}
</style>
