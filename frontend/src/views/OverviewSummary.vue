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
import { formatDuration, formatNumber } from '../api'
import { useOverviewContext } from './overviewContext'

const ATypographyText = Typography.Text

const { overview, loading, startupIndexing, hasIndexedData, sourcePathDisplay, indexFromOverview } = useOverviewContext()

interface DisplayNumber {
  main: string
  suffix: string
  full: string
}

const totalTokensDisplay = computed(() => compactNumber(overview.value?.totalTokens))

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

const tokenBreakdown = computed(() => {
  const item = overview.value
  const values = [
    { label: 'Input', value: item?.totalInputTokens || 0, tone: 'is-input' },
    { label: 'Cached', value: item?.totalCachedInputTokens || 0, tone: 'is-cached' },
    { label: 'Output', value: item?.totalOutputTokens || 0, tone: 'is-output' },
    { label: 'Reasoning', value: item?.totalReasoningTokens || 0, tone: 'is-reasoning' }
  ]
  const total = values.reduce((sum, current) => sum + Math.max(current.value, 0), 0)
  return values.map((current) => {
    const share = total > 0 ? current.value / total : 0
    return {
      ...current,
      display: compactNumber(current.value),
      exact: formatNumber(current.value),
      shareLabel: formatSharePercent(share),
      shareWidth: formatShareWidth(share)
    }
  })
})

const snapshotCards = computed(() => [
  {
    label: 'Sessions',
    value: compactNumber(overview.value?.totalSessions),
    note: `${formatDuration(overview.value?.totalWallDurationMs)} wall time`,
    icon: ClockCircleOutlined,
    tone: 'metric-primary'
  },
  {
    label: 'Estimated Cost',
    value: compactCurrency(overview.value?.estimatedCostUsd),
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
    value: compactNumber(overview.value?.totalToolCalls),
    note: 'Across indexed sessions',
    icon: ToolOutlined,
    tone: 'metric-info'
  },
  {
    label: 'Active Time',
    value: textMetric(formatDuration(overview.value?.totalActiveDurationMs)),
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
  return formatUSD(value, 4)
}

function compactNumber(value: number | undefined): DisplayNumber {
  const normalized = Math.max(0, Number(value || 0))
  const full = formatNumber(normalized)
  if (normalized < 10_000) return { main: full, suffix: '', full }

  const tiers = [
    { value: 1_000_000_000, suffix: 'B' },
    { value: 1_000_000, suffix: 'M' },
    { value: 1_000, suffix: 'K' }
  ]
  const tier = tiers.find((candidate) => normalized >= candidate.value)
  if (!tier) return { main: full, suffix: '', full }

  const scaled = normalized / tier.value
  const maximumFractionDigits = scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2
  return {
    main: new Intl.NumberFormat(undefined, { maximumFractionDigits }).format(scaled),
    suffix: tier.suffix,
    full
  }
}

function textMetric(value: string): DisplayNumber {
  return { main: value, suffix: '', full: value }
}

function compactCurrency(value: number | undefined): DisplayNumber {
  if (value === undefined || value === null) return textMetric('unpriced')
  const normalized = Math.max(0, Number(value || 0))
  const full = formatUSD(normalized, 4)
  if (normalized < 1_000) {
    return { main: formatUSD(normalized, normalized < 1 ? 4 : 2), suffix: '', full }
  }

  const tiers = [
    { value: 1_000_000_000, suffix: 'B' },
    { value: 1_000_000, suffix: 'M' },
    { value: 1_000, suffix: 'K' }
  ]
  const tier = tiers.find((candidate) => normalized >= candidate.value)
  if (!tier) return { main: full, suffix: '', full }

  const scaled = normalized / tier.value
  const maximumFractionDigits = scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2
  return {
    main: formatUSD(scaled, maximumFractionDigits),
    suffix: tier.suffix,
    full
  }
}

function formatUSD(value: number, maximumFractionDigits: number) {
  return `$${new Intl.NumberFormat(undefined, { maximumFractionDigits }).format(value)}`
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
        <div class="overview-snapshot-value" :title="`${totalTokensDisplay.full} tokens`">
          <span>{{ totalTokensDisplay.main }}</span>
          <em v-if="totalTokensDisplay.suffix">{{ totalTokensDisplay.suffix }}</em>
        </div>
        <div class="overview-snapshot-unit">tokens</div>
        <div class="overview-snapshot-exact">{{ totalTokensDisplay.full }} exact total</div>
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
          Update Index
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
