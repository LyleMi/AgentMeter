<script setup lang="ts">
import { computed } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import Typography from 'ant-design-vue/es/typography'
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  DollarCircleOutlined,
  FolderOpenOutlined,
  FunctionOutlined,
  PlayCircleOutlined,
  SettingOutlined,
  ToolOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import { formatDisplayCost, formatDisplayNumber, formatDuration, formatNumber, type DisplayNumber } from '../api'
import { useMessages } from '../i18n'
import { useOverviewContext } from './overviewContext'

const ATypographyText = Typography.Text

const { overview, loading, startupIndexing, hasIndexedData, sourcePathDisplay, indexFromOverview } = useOverviewContext()
const { t, createNumberFormatter } = useMessages({
  en: {
    'metric.totalUsage': 'Total Usage',
    'metric.indexedSessions': '{count} indexed sessions',
    'metric.tokens': 'tokens',
    'metric.tokensTitle': '{count} tokens',
    'metric.exactTotal': '{count} exact total',
    'pricing.none': 'No indexed pricing',
    'pricing.unpriced': '{count} unpriced',
    'pricing.covered': 'Pricing covered',
    'token.input': 'Input',
    'token.cached': 'Cached',
    'token.output': 'Output',
    'token.reasoning': 'Reasoning',
    'card.sessions': 'Sessions',
    'card.sessionsNote': '{duration} wall time',
    'card.estimatedCost': 'Estimated Cost',
    'card.missingPricing': '{count} sessions missing pricing',
    'card.allPriced': 'All indexed sessions priced',
    'card.toolCalls': 'Tool Calls',
    'card.toolCallsNote': 'Across indexed sessions',
    'card.activeTime': 'Active Time',
    'card.activeTimeNote': 'Measured model and tool time',
    'efficiency.title': 'Efficiency',
    'efficiency.kicker': 'Session scale, cache reuse, and tool depth',
    'efficiency.avgTokens': 'Avg tokens / session',
    'efficiency.avgTokensNote': 'Total workload size per indexed session',
    'efficiency.toolsPerSession': 'Tools / session',
    'efficiency.toolsPerSessionNote': 'Tool invocations per session',
    'efficiency.cacheHit': 'Cache hit rate',
    'efficiency.cacheHitNote': '{count} cached input tokens',
    'efficiency.outputInput': 'Output / input',
    'efficiency.outputInputNote': 'Response token density',
    'timeCost.title': 'Time & Cost',
    'timeCost.kicker': 'Duration, active share, and spend density',
    'timeCost.avgWall': 'Avg wall / session',
    'timeCost.avgWallNote': 'First to last timestamp',
    'timeCost.activeShare': 'Active share',
    'timeCost.activeShareNote': 'Measured model and tool time',
    'timeCost.costPer1k': 'Cost / 1K tokens',
    'timeCost.completePricingNote': 'Uses complete pricing coverage',
    'timeCost.needsPricingNote': 'Needs pricing for all sessions',
    'timeCost.tokensPerHour': 'Tokens / active hour',
    'timeCost.tokensPerHourNote': 'Token throughput during measured work',
    'fallback.unpriced': 'unpriced',
    'empty.title': 'No indexed sessions yet',
    'empty.text': 'AgentMeter can scan configured local agent sources now and refresh this dashboard when indexing completes.',
    'empty.sources': 'Sources',
    'empty.sourcePath': 'Open Settings to choose a source path',
    'empty.updateIndex': 'Update Index',
    'empty.editSource': 'Edit Source'
  },
  'zh-CN': {
    'metric.totalUsage': '总用量',
    'metric.indexedSessions': '已索引 {count} 个会话',
    'metric.tokens': 'Token',
    'metric.tokensTitle': '{count} 个 Token',
    'metric.exactTotal': '精确总数 {count}',
    'pricing.none': '暂无已索引价格',
    'pricing.unpriced': '{count} 个未定价',
    'pricing.covered': '价格已覆盖',
    'token.input': '输入',
    'token.cached': '缓存',
    'token.output': '输出',
    'token.reasoning': '推理',
    'card.sessions': '会话',
    'card.sessionsNote': '总墙钟时间 {duration}',
    'card.estimatedCost': '预估费用',
    'card.missingPricing': '{count} 个会话缺少价格',
    'card.allPriced': '所有已索引会话都有价格',
    'card.toolCalls': '工具调用',
    'card.toolCallsNote': '跨已索引会话统计',
    'card.activeTime': '活跃时间',
    'card.activeTimeNote': '已测量的模型和工具时间',
    'efficiency.title': '效率',
    'efficiency.kicker': '会话规模、缓存复用和工具深度',
    'efficiency.avgTokens': '平均 Token / 会话',
    'efficiency.avgTokensNote': '每个已索引会话的总工作量',
    'efficiency.toolsPerSession': '工具 / 会话',
    'efficiency.toolsPerSessionNote': '每个会话的工具调用次数',
    'efficiency.cacheHit': '缓存命中率',
    'efficiency.cacheHitNote': '{count} 个缓存输入 Token',
    'efficiency.outputInput': '输出 / 输入',
    'efficiency.outputInputNote': '响应 Token 密度',
    'timeCost.title': '时间与费用',
    'timeCost.kicker': '时长、活跃占比和花费密度',
    'timeCost.avgWall': '平均墙钟 / 会话',
    'timeCost.avgWallNote': '从第一条到最后一条时间戳',
    'timeCost.activeShare': '活跃占比',
    'timeCost.activeShareNote': '已测量的模型和工具时间',
    'timeCost.costPer1k': '每 1K Token 费用',
    'timeCost.completePricingNote': '使用完整价格覆盖计算',
    'timeCost.needsPricingNote': '需要为所有会话补齐价格',
    'timeCost.tokensPerHour': 'Token / 活跃小时',
    'timeCost.tokensPerHourNote': '已测量工作期间的 Token 吞吐',
    'fallback.unpriced': '未定价',
    'empty.title': '还没有已索引会话',
    'empty.text': 'AgentMeter 可以立即扫描已配置的本地代理来源，并在索引完成后刷新此仪表盘。',
    'empty.sources': '来源',
    'empty.sourcePath': '打开设置选择来源路径',
    'empty.updateIndex': '更新索引',
    'empty.editSource': '编辑来源'
  }
})

const totalTokensDisplay = computed(() => formatDisplayNumber(overview.value?.totalTokens))

const pricingStatus = computed(() => {
  const item = overview.value
  if (!item || item.totalSessions <= 0) {
    return { label: t('pricing.none'), tone: 'is-neutral', icon: WarningOutlined }
  }
  if ((item.unpricedSessions || 0) > 0) {
    return { label: t('pricing.unpriced', { count: formatNumber(item.unpricedSessions) }), tone: 'is-warning', icon: WarningOutlined }
  }
  return { label: t('pricing.covered'), tone: 'is-success', icon: CheckCircleOutlined }
})

const tokenBreakdown = computed(() => {
  const item = overview.value
  const values = [
    { label: t('token.input'), value: item?.totalInputTokens || 0, tone: 'is-input' },
    { label: t('token.cached'), value: item?.totalCachedInputTokens || 0, tone: 'is-cached' },
    { label: t('token.output'), value: item?.totalOutputTokens || 0, tone: 'is-output' },
    { label: t('token.reasoning'), value: item?.totalReasoningTokens || 0, tone: 'is-reasoning' }
  ]
  const total = values.reduce((sum, current) => sum + Math.max(current.value, 0), 0)
  return values.map((current) => {
    const share = total > 0 ? current.value / total : 0
    return {
      ...current,
      display: formatDisplayNumber(current.value),
      exact: formatNumber(current.value),
      shareLabel: formatSharePercent(share),
      shareWidth: formatShareWidth(share)
    }
  })
})

const snapshotCards = computed(() => [
  {
    label: t('card.sessions'),
    value: formatDisplayNumber(overview.value?.totalSessions),
    note: t('card.sessionsNote', { duration: formatDuration(overview.value?.totalWallDurationMs) }),
    icon: ClockCircleOutlined,
    tone: 'metric-primary'
  },
  {
    label: t('card.estimatedCost'),
    value: formatDisplayCost(overview.value?.estimatedCostUsd),
    note:
      (overview.value?.unpricedSessions || 0) > 0
        ? t('card.missingPricing', { count: formatNumber(overview.value?.unpricedSessions) })
        : t('card.allPriced'),
    icon: DollarCircleOutlined,
    tone: 'metric-warning',
    warning: (overview.value?.unpricedSessions || 0) > 0
  },
  {
    label: t('card.toolCalls'),
    value: formatDisplayNumber(overview.value?.totalToolCalls),
    note: t('card.toolCallsNote'),
    icon: ToolOutlined,
    tone: 'metric-info'
  },
  {
    label: t('card.activeTime'),
    value: displayText(formatDuration(overview.value?.totalActiveDurationMs)),
    note: t('card.activeTimeNote'),
    icon: ClockCircleOutlined,
    tone: 'metric-neutral'
  }
])

const efficiencyMetrics = computed(() => {
  const item = overview.value
  if (!item || item.totalSessions <= 0) return []
  const sessions = item.totalSessions
  const inputTokens = Math.max(item.totalInputTokens || 0, 0)
  return [
    {
      label: t('efficiency.avgTokens'),
      value: formatNumber(Math.round((item.totalTokens || 0) / sessions)),
      note: t('efficiency.avgTokensNote')
    },
    {
      label: t('efficiency.toolsPerSession'),
      value: formatRatio((item.totalToolCalls || 0) / sessions),
      note: t('efficiency.toolsPerSessionNote')
    },
    {
      label: t('efficiency.cacheHit'),
      value: formatPercent((item.totalCachedInputTokens || 0) / Math.max(inputTokens, 1)),
      note: t('efficiency.cacheHitNote', { count: formatNumber(item.totalCachedInputTokens) })
    },
    {
      label: t('efficiency.outputInput'),
      value: `${formatRatio((item.totalOutputTokens || 0) / Math.max(inputTokens, 1))}x`,
      note: t('efficiency.outputInputNote')
    }
  ]
})

const timeCostMetrics = computed(() => {
  const item = overview.value
  if (!item || item.totalSessions <= 0) return []
  const sessions = item.totalSessions
  const activeHours = (item.totalActiveDurationMs || 0) / 3_600_000
  const hasCompletePricing = item.estimatedCostUsd !== undefined && item.estimatedCostUsd !== null && item.unpricedSessions === 0
  return [
    {
      label: t('timeCost.avgWall'),
      value: formatDuration((item.totalWallDurationMs || 0) / sessions),
      note: t('timeCost.avgWallNote')
    },
    {
      label: t('timeCost.activeShare'),
      value: formatPercent((item.totalActiveDurationMs || 0) / Math.max(item.totalWallDurationMs || 0, 1)),
      note: t('timeCost.activeShareNote')
    },
    {
      label: t('timeCost.costPer1k'),
      value: hasCompletePricing ? formatCostPerThousand(item.estimatedCostUsd || 0, item.totalTokens || 0) : t('fallback.unpriced'),
      note: hasCompletePricing ? t('timeCost.completePricingNote') : t('timeCost.needsPricingNote')
    },
    {
      label: t('timeCost.tokensPerHour'),
      value: activeHours > 0 ? formatNumber(Math.round((item.totalTokens || 0) / activeHours)) : '-',
      note: t('timeCost.tokensPerHourNote')
    }
  ]
})

function formatPercent(value: number) {
  if (!Number.isFinite(value)) return '0%'
  return `${Math.round(Math.max(0, value) * 100)}%`
}

function formatRatio(value: number) {
  if (!Number.isFinite(value)) return '0'
  const normalized = Math.max(0, value)
  return createNumberFormatter({
    maximumFractionDigits: normalized > 0 && normalized < 0.1 ? 2 : 1
  }).format(normalized)
}

function formatCostPerThousand(cost: number, tokens: number) {
  if (!tokens) return formatUSD(0, 0)
  const value = cost / (tokens / 1000)
  return formatUSD(value, 4)
}

function displayText(value: string): DisplayNumber {
  return { main: value, suffix: '', full: value }
}

function formatUSD(value: number, maximumFractionDigits: number) {
  return createNumberFormatter({ style: 'currency', currency: 'USD', maximumFractionDigits }).format(value)
}

function formatSharePercent(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '0%'
  if (value < 0.01) return '<1%'
  return formatPercent(value)
}

function formatShareWidth(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '0%'
  return `${Math.max(2, Math.round(value * 100))}%`
}
</script>

<template>
  <a-spin :spinning="loading">
    <section class="overview-summary-layout">
      <div class="overview-snapshot-primary">
        <div class="overview-snapshot-head">
          <div>
            <div class="metric-label">{{ t('metric.totalUsage') }}</div>
            <div class="overview-snapshot-caption">
              {{ t('metric.indexedSessions', { count: formatNumber(overview?.totalSessions) }) }}
            </div>
          </div>
          <span class="overview-coverage-chip" :class="pricingStatus.tone">
            <component :is="pricingStatus.icon" />
            {{ pricingStatus.label }}
          </span>
        </div>
        <div class="overview-snapshot-value" :title="t('metric.tokensTitle', { count: totalTokensDisplay.full })">
          <span>{{ totalTokensDisplay.main }}</span>
          <em v-if="totalTokensDisplay.suffix">{{ totalTokensDisplay.suffix }}</em>
        </div>
        <div class="overview-snapshot-unit">{{ t('metric.tokens') }}</div>
        <div class="overview-snapshot-exact">{{ t('metric.exactTotal', { count: totalTokensDisplay.full }) }}</div>
        <div class="overview-token-breakdown">
          <div v-for="item in tokenBreakdown" :key="item.label" class="overview-token-item" :class="item.tone">
            <div class="overview-token-row">
              <span>{{ item.label }}</span>
              <strong :title="item.exact">
                {{ item.display.main }}<em v-if="item.display.suffix">{{ item.display.suffix }}</em>
              </strong>
            </div>
            <div class="overview-token-meter" :aria-label="`${item.label} ${item.shareLabel}`">
              <span :style="{ width: item.shareWidth }"></span>
            </div>
            <div class="overview-token-share">{{ item.shareLabel }}</div>
          </div>
        </div>
      </div>

      <div class="overview-kpi-grid">
        <div
          v-for="item in snapshotCards"
          :key="item.label"
          class="overview-kpi-card"
          :class="[item.tone, { 'has-warning': item.warning }]"
        >
          <div class="overview-kpi-head">
            <span class="metric-label">{{ item.label }}</span>
            <component :is="item.icon" class="metric-icon" />
          </div>
          <div class="overview-kpi-value" :title="item.value.full">
            <span>{{ item.value.main }}</span>
            <em v-if="item.value.suffix">{{ item.value.suffix }}</em>
          </div>
          <div class="overview-kpi-note" :class="{ 'metric-note-warning': item.warning }">{{ item.note }}</div>
        </div>
      </div>
    </section>

    <section v-if="hasIndexedData" class="overview-signal-layout">
      <section class="panel overview-signal-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('efficiency.title') }}</h2>
            <div class="panel-kicker">{{ t('efficiency.kicker') }}</div>
          </div>
          <FunctionOutlined class="panel-header-icon" />
        </div>
        <div class="overview-signal-list">
          <div v-for="item in efficiencyMetrics" :key="item.label" class="overview-signal-item">
            <div>
              <div class="overview-signal-label">{{ item.label }}</div>
              <div class="overview-signal-note">{{ item.note }}</div>
            </div>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </section>

      <section class="panel overview-signal-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('timeCost.title') }}</h2>
            <div class="panel-kicker">{{ t('timeCost.kicker') }}</div>
          </div>
          <DollarCircleOutlined class="panel-header-icon" />
        </div>
        <div class="overview-signal-list">
          <div v-for="item in timeCostMetrics" :key="item.label" class="overview-signal-item">
            <div>
              <div class="overview-signal-label">{{ item.label }}</div>
              <div class="overview-signal-note">{{ item.note }}</div>
            </div>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </section>
    </section>

    <div v-if="!loading && !hasIndexedData" class="empty-callout overview-empty-callout">
      <div class="empty-callout-main">
        <div class="empty-callout-title">{{ t('empty.title') }}</div>
        <div class="empty-callout-text">
          {{ t('empty.text') }}
        </div>
        <div class="empty-source-line">
          <FolderOpenOutlined />
          <span class="source-label">{{ t('empty.sources') }}</span>
          <a-typography-text class="empty-source-path" :ellipsis="{ tooltip: sourcePathDisplay }">
            {{ sourcePathDisplay || t('empty.sourcePath') }}
          </a-typography-text>
        </div>
      </div>
      <div class="empty-callout-actions">
        <a-button type="primary" :loading="startupIndexing" :disabled="!sourcePathDisplay" @click="indexFromOverview">
          <template #icon>
            <PlayCircleOutlined />
          </template>
          {{ t('empty.updateIndex') }}
        </a-button>
        <a-button @click="$router.push('/settings')">
          <template #icon>
            <SettingOutlined />
          </template>
          {{ t('empty.editSource') }}
        </a-button>
      </div>
    </div>
  </a-spin>
</template>
