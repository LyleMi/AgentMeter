<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import {
  ArrowRightOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatCost,
  formatDateTime,
  formatDuration,
  formatNumber,
  type AttentionItem,
  type AttentionMetric,
  type AttentionResponse
} from '../api'
import PageHeader from '../components/PageHeader.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { applyUsageScopeToQuery, useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import {
  buildUsageAgentOptions,
  buildUsageModelOptions,
  buildUsageProjectOptions,
  useUsageScopeOptionData
} from './useUsageScopeOptions'

const route = useRoute()
const router = useRouter()
const resource = useAsyncResource<AttentionResponse | null>(null)
const attention = computed(() => resource.data.value)
const loading = resource.loading
const error = resource.error
const scope = useUsageScopeRoute(() => {
  void load()
})
const scopeOptions = useUsageScopeOptionData()

const { t } = useMessages({
  en: {
    title: 'Attention',
    subtitle: 'What needs review in the selected period, prioritized by severity and confidence',
    'range.current': 'Current: {from} – {to}',
    'range.baseline': 'Baseline: {from} – {to}',
    'metric.sessions': 'Sessions',
    'metric.tokens': 'Tokens',
    'metric.cost': 'Estimated Cost',
    'metric.active': 'Active Time',
    'metric.tools': 'Tool Calls',
    'metric.noBaseline': 'no baseline',
    'metric.change': '{value} vs baseline',
    'list.title': 'Needs attention',
    'list.count': '{critical} critical · {warning} warning',
    'severity.critical': 'Critical',
    'severity.warning': 'Warning',
    'confidence': '{value}% confidence',
    'action.review': 'Review',
    'empty.title': 'No issues need attention in the current range',
    'empty.text': 'The status summary remains available above. Change the filters or refresh after indexing new data.',
    'error.title': 'Attention could not be loaded',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    title: '关注',
    subtitle: '按严重度和置信度优先展示选定周期内需要检查的事项',
    'range.current': '当前：{from} – {to}',
    'range.baseline': '基线：{from} – {to}',
    'metric.sessions': '会话',
    'metric.tokens': 'Token',
    'metric.cost': '预估费用',
    'metric.active': '活跃时间',
    'metric.tools': '工具调用',
    'metric.noBaseline': '无基线',
    'metric.change': '较基线 {value}',
    'list.title': '需要关注',
    'list.count': '{critical} 条严重 · {warning} 条警告',
    'severity.critical': '严重',
    'severity.warning': '警告',
    'confidence': '置信度 {value}%',
    'action.review': '查看',
    'empty.title': '当前范围未发现需要关注的问题',
    'empty.text': '上方仍保留状态摘要。可调整筛选，或在索引新数据后刷新。',
    'error.title': '关注数据加载失败',
    'fallback.unknown': '未知'
  }
})

const agentOptions = computed(() =>
  buildUsageAgentOptions({
    sources: [
      scopeOptions.optionOverview.value?.agentUsage,
      scopeOptions.optionOverview.value?.recentSessions
    ],
    selected: scope.filters.value.agent,
    fallback: t('fallback.unknown')
  })
)
const modelOptions = computed(() =>
  buildUsageModelOptions({
    modelUsage: [scopeOptions.optionOverview.value?.modelUsage],
    sessions: [scopeOptions.optionOverview.value?.recentSessions],
    selected: scope.filters.value.model
  })
)
const projectOptions = computed(() =>
  buildUsageProjectOptions({
    projects: [
      scopeOptions.projectOptionRows.value,
      scopeOptions.optionOverview.value?.recentSessions
    ],
    selected: scope.filters.value.project,
    fallback: t('fallback.unknown')
  })
)

const cards = computed(() => {
  const snapshot = attention.value?.snapshot
  return [
    { key: 'sessions', label: t('metric.sessions'), metric: snapshot?.sessions, format: formatNumber },
    { key: 'tokens', label: t('metric.tokens'), metric: snapshot?.tokens, format: formatNumber },
    { key: 'cost', label: t('metric.cost'), metric: snapshot?.costUsd, format: (value: number) => formatCost(value) },
    { key: 'active', label: t('metric.active'), metric: snapshot?.activeTime, format: formatDuration },
    { key: 'tools', label: t('metric.tools'), metric: snapshot?.toolCalls, format: formatNumber }
  ]
})

function load() {
  return resource.run(async () => {
    const [nextAttention, optionData] = await Promise.all([
      api.getAttention(scope.apiFilters.value),
      scopeOptions.loadUsageScopeOptionData()
    ])
    scopeOptions.applyUsageScopeOptionData(optionData)
    return nextAttention
  }, { onErrorData: null })
}

async function updateFilters(filters: UsageScopeForm) {
  await scope.updateFilters(filters)
  await load()
}

async function clearFilters() {
  await scope.clearFilters()
  await load()
}

function changeLabel(metric?: AttentionMetric) {
  if (!metric?.changePct && metric?.changePct !== 0) return t('metric.noBaseline')
  const sign = metric.changePct > 0 ? '+' : ''
  return t('metric.change', { value: `${sign}${Math.round(metric.changePct * 100)}%` })
}

function shortDate(value?: string) {
  return value ? formatDateTime(value) : '-'
}

function destination(item: AttentionItem) {
  const target = item.destination
  const query = applyUsageScopeToQuery(route.query, scope.filters.value)
  switch (target.kind) {
    case 'session':
      return { path: `/sessions/${target.sessionId}`, query: { returnTo: route.fullPath } }
    case 'audit_finding':
      return { path: `/safety/audit/findings/${target.auditFindingId}`, query }
    case 'model_analysis':
      return {
        path: '/analysis/models',
        query: {
          ...query,
          agent: target.agent || query.agent,
          model: target.model || query.model,
          project: target.project || query.project
        }
      }
    case 'privacy':
      return { path: '/safety/privacy', query: target.privacyTarget ? { target: target.privacyTarget } : {} }
    case 'settings':
      return { path: `/settings/${target.settingsPanel || 'data'}` }
    default:
      return { path: '/attention', query }
  }
}

function review(item: AttentionItem) {
  router.push(destination(item))
}

function severityColor(item: AttentionItem) {
  return item.severity === 'critical' ? 'error' : 'warning'
}

onMounted(load)
</script>

<template>
  <div class="page attention-page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')" />

    <UsageScopeBar
      :filters="scope.filters.value"
      :agent-options="agentOptions"
      :model-options="modelOptions"
      :project-options="projectOptions"
      :loading="loading"
      @update:filters="updateFilters"
      @refresh="load"
      @clear="clearFilters"
    />

    <a-alert
      v-if="error"
      class="attention-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <a-spin :spinning="loading && !attention">
      <div v-if="attention" class="attention-range">
        <span>{{ t('range.current', { from: shortDate(attention.window.current.from), to: shortDate(attention.window.current.to) }) }}</span>
        <span>{{ t('range.baseline', { from: shortDate(attention.window.baseline.from), to: shortDate(attention.window.baseline.to) }) }}</span>
      </div>

      <section class="attention-kpis">
        <article v-for="card in cards" :key="card.key" class="metric-card attention-kpi">
          <div class="metric-label">{{ card.label }}</div>
          <div class="metric-value">{{ card.format(card.metric?.current || 0) }}</div>
          <div class="metric-note">{{ changeLabel(card.metric) }}</div>
        </article>
      </section>

      <section class="panel attention-list-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('list.title') }}</h2>
            <div v-if="attention" class="panel-kicker">
              {{ t('list.count', { critical: attention.counts.critical, warning: attention.counts.warning }) }}
            </div>
          </div>
          <WarningOutlined v-if="attention?.items.length" class="panel-header-icon" />
          <CheckCircleOutlined v-else class="panel-header-icon status-ok" />
        </div>

        <div v-if="attention?.items.length" class="attention-items">
          <article
            v-for="item in attention.items"
            :key="item.key"
            class="attention-item"
            :class="`is-${item.severity}`"
          >
            <div class="attention-item-icon">
              <ExclamationCircleOutlined />
            </div>
            <div class="attention-item-copy">
              <div class="attention-item-heading">
                <a-tag :color="severityColor(item)">
                  {{ t(`severity.${item.severity}`) }}
                </a-tag>
                <strong>{{ item.subject }}</strong>
              </div>
              <p>{{ item.reason }}</p>
              <div class="attention-item-meta">
                <span>{{ t('confidence', { value: Math.round(item.confidence * 100) }) }}</span>
                <span v-if="item.metric">{{ item.metric.label }}: {{ formatNumber(item.metric.value) }}</span>
                <span v-if="item.occurredAt">{{ formatDateTime(item.occurredAt) }}</span>
              </div>
            </div>
            <a-button type="link" @click="review(item)">
              {{ t('action.review') }}
              <template #icon><ArrowRightOutlined /></template>
            </a-button>
          </article>
        </div>

        <div v-else class="empty-state">
          <CheckCircleOutlined class="empty-state-icon status-ok" />
          <div class="empty-state-title">{{ t('empty.title') }}</div>
          <div class="empty-state-text">{{ t('empty.text') }}</div>
        </div>
      </section>
    </a-spin>
  </div>
</template>

<style scoped>
.attention-page {
  max-width: 1560px;
}

.attention-error,
.attention-range {
  margin-bottom: var(--am-section-gap);
}

.attention-range {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 20px;
  color: var(--am-text-muted);
  font-size: 12px;
}

.attention-kpis {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: var(--am-section-gap);
}

.attention-kpi {
  min-width: 0;
  padding: 14px 15px;
}

.attention-list-panel {
  overflow: hidden;
}

.attention-items {
  display: grid;
}

.attention-item {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: start;
  padding: 16px 18px;
  border-top: 1px solid var(--am-border-subtle);
}

.attention-item.is-critical {
  box-shadow: inset 3px 0 #dc2626;
}

.attention-item.is-warning {
  box-shadow: inset 3px 0 #d97706;
}

.attention-item-icon {
  padding-top: 3px;
  color: #d97706;
  font-size: 18px;
}

.attention-item.is-critical .attention-item-icon {
  color: #dc2626;
}

.attention-item-heading,
.attention-item-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.attention-item-copy p {
  margin: 7px 0;
  color: var(--am-text-secondary);
}

.attention-item-meta {
  color: var(--am-text-muted);
  font-size: 12px;
}

@media (max-width: 1100px) {
  .attention-kpis {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .attention-kpis {
    grid-template-columns: 1fr 1fr;
  }

  .attention-item {
    grid-template-columns: 24px minmax(0, 1fr);
  }

  .attention-item > .ant-btn {
    grid-column: 2;
    justify-self: start;
    padding-left: 0;
  }
}
</style>
