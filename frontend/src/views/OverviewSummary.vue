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
import { formatCost, formatDuration, formatNumber } from '../api'
import { useOverviewContext } from './overviewContext'

const ATypographyText = Typography.Text

const { overview, loading, startupIndexing, hasIndexedData, sourcePathDisplay, indexFromOverview } = useOverviewContext()

const pricingStatus = computed(() => {
  const item = overview.value
  if (!item || item.totalSessions <= 0) {
    return { label: 'No indexed pricing', tone: 'is-neutral', icon: WarningOutlined }
  }
  if ((item.unpricedSessions || 0) > 0) {
    return { label: `${formatNumber(item.unpricedSessions)} unpriced`, tone: 'is-warning', icon: WarningOutlined }
  }
  return { label: 'Pricing covered', tone: 'is-success', icon: CheckCircleOutlined }
})

const tokenBreakdown = computed(() => [
  { label: 'Input', value: formatNumber(overview.value?.totalInputTokens) },
  { label: 'Cached', value: formatNumber(overview.value?.totalCachedInputTokens) },
  { label: 'Output', value: formatNumber(overview.value?.totalOutputTokens) },
  { label: 'Reasoning', value: formatNumber(overview.value?.totalReasoningTokens) }
])

const snapshotCards = computed(() => [
  {
    label: 'Sessions',
    value: formatNumber(overview.value?.totalSessions),
    note: `${formatDuration(overview.value?.totalWallDurationMs)} wall time`,
    icon: ClockCircleOutlined,
    tone: 'metric-primary'
  },
  {
    label: 'Estimated Cost',
    value: formatCost(overview.value?.estimatedCostUsd),
    note:
      (overview.value?.unpricedSessions || 0) > 0
        ? `${formatNumber(overview.value?.unpricedSessions)} sessions missing pricing`
        : 'All indexed sessions priced',
    icon: DollarCircleOutlined,
    tone: 'metric-warning',
    warning: (overview.value?.unpricedSessions || 0) > 0
  },
  {
    label: 'Tool Calls',
    value: formatNumber(overview.value?.totalToolCalls),
    note: 'Across indexed sessions',
    icon: ToolOutlined,
    tone: 'metric-info'
  },
  {
    label: 'Active Time',
    value: formatDuration(overview.value?.totalActiveDurationMs),
    note: 'Measured model and tool time',
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
      label: 'Avg tokens / session',
      value: formatNumber(Math.round((item.totalTokens || 0) / sessions)),
      note: 'Total workload size per indexed session'
    },
    {
      label: 'Tools / session',
      value: formatRatio((item.totalToolCalls || 0) / sessions),
      note: 'Tool invocations per session'
    },
    {
      label: 'Cache hit rate',
      value: formatPercent((item.totalCachedInputTokens || 0) / Math.max(inputTokens, 1)),
      note: `${formatNumber(item.totalCachedInputTokens)} cached input tokens`
    },
    {
      label: 'Output / input',
      value: `${formatRatio((item.totalOutputTokens || 0) / Math.max(inputTokens, 1))}x`,
      note: 'Response token density'
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
      label: 'Avg wall / session',
      value: formatDuration((item.totalWallDurationMs || 0) / sessions),
      note: 'First to last timestamp'
    },
    {
      label: 'Active share',
      value: formatPercent((item.totalActiveDurationMs || 0) / Math.max(item.totalWallDurationMs || 0, 1)),
      note: 'Measured model and tool time'
    },
    {
      label: 'Cost / 1K tokens',
      value: hasCompletePricing ? formatCostPerThousand(item.estimatedCostUsd || 0, item.totalTokens || 0) : 'unpriced',
      note: hasCompletePricing ? 'Uses complete pricing coverage' : 'Needs pricing for all sessions'
    },
    {
      label: 'Tokens / active hour',
      value: activeHours > 0 ? formatNumber(Math.round((item.totalTokens || 0) / activeHours)) : '-',
      note: 'Token throughput during measured work'
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
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: normalized > 0 && normalized < 0.1 ? 2 : 1
  }).format(normalized)
}

function formatCostPerThousand(cost: number, tokens: number) {
  if (!tokens) return '$0'
  const value = cost / (tokens / 1000)
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 4
  }).format(value)
}
</script>

<template>
  <a-spin :spinning="loading">
    <section class="overview-summary-layout">
      <div class="overview-snapshot-primary">
        <div class="overview-snapshot-head">
          <div>
            <div class="metric-label">Total Usage</div>
            <div class="overview-snapshot-caption">
              {{ formatNumber(overview?.totalSessions) }} indexed sessions
            </div>
          </div>
          <span class="overview-coverage-chip" :class="pricingStatus.tone">
            <component :is="pricingStatus.icon" />
            {{ pricingStatus.label }}
          </span>
        </div>
        <div class="overview-snapshot-value">{{ formatNumber(overview?.totalTokens) }}</div>
        <div class="overview-snapshot-unit">tokens</div>
        <div class="overview-token-breakdown">
          <div v-for="item in tokenBreakdown" :key="item.label" class="overview-token-item">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
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
          <div class="overview-kpi-value">{{ item.value }}</div>
          <div class="overview-kpi-note" :class="{ 'metric-note-warning': item.warning }">{{ item.note }}</div>
        </div>
      </div>
    </section>

    <section v-if="hasIndexedData" class="overview-signal-layout">
      <section class="panel overview-signal-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Efficiency</h2>
            <div class="panel-kicker">Session scale, cache reuse, and tool depth</div>
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
            <h2 class="panel-title">Time & Cost</h2>
            <div class="panel-kicker">Duration, active share, and spend density</div>
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
        <div class="empty-callout-title">No indexed sessions yet</div>
        <div class="empty-callout-text">
          AgentMeter can scan configured local agent sources now and refresh this dashboard when indexing completes.
        </div>
        <div class="empty-source-line">
          <FolderOpenOutlined />
          <span class="source-label">Sources</span>
          <a-typography-text class="empty-source-path" :ellipsis="{ tooltip: sourcePathDisplay }">
            {{ sourcePathDisplay || 'Open Settings to choose a source path' }}
          </a-typography-text>
        </div>
      </div>
      <div class="empty-callout-actions">
        <a-button type="primary" :loading="startupIndexing" :disabled="!sourcePathDisplay" @click="indexFromOverview">
          <template #icon>
            <PlayCircleOutlined />
          </template>
          Index Now
        </a-button>
        <a-button @click="$router.push('/settings')">
          <template #icon>
            <SettingOutlined />
          </template>
          Edit Source
        </a-button>
      </div>
    </div>
  </a-spin>
</template>
