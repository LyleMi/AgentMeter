<script setup lang="ts">
import { computed } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import Typography from 'ant-design-vue/es/typography'
import {
  ClockCircleOutlined,
  DollarCircleOutlined,
  FolderOpenOutlined,
  FunctionOutlined,
  PlayCircleOutlined,
  SettingOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import { formatCost, formatDuration, formatNumber } from '../api'
import { useOverviewContext } from './overviewContext'

const ATypographyText = Typography.Text

const { overview, loading, startupIndexing, hasIndexedData, sourcePathDisplay, indexFromOverview } = useOverviewContext()

const derivedMetrics = computed(() => {
  const item = overview.value
  if (!item || item.totalSessions <= 0) return []
  const sessions = item.totalSessions
  const inputTokens = Math.max(item.totalInputTokens || 0, 0)
  const activeHours = (item.totalActiveDurationMs || 0) / 3_600_000
  const hasCompletePricing = item.estimatedCostUsd !== undefined && item.estimatedCostUsd !== null && item.unpricedSessions === 0
  return [
    {
      label: 'Avg tokens / session',
      value: formatNumber(Math.round((item.totalTokens || 0) / sessions)),
      note: 'Total tokens divided by sessions'
    },
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
      label: 'Tools / session',
      value: formatRatio((item.totalToolCalls || 0) / sessions),
      note: 'Tool invocations per session'
    },
    {
      label: 'Cache hit rate',
      value: formatPercent((item.totalCachedInputTokens || 0) / Math.max(inputTokens, 1)),
      note: 'Cached input over input tokens'
    },
    {
      label: 'Output / input',
      value: `${formatRatio((item.totalOutputTokens || 0) / Math.max(inputTokens, 1))}x`,
      note: 'Output token density'
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
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(Math.max(0, value))
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
    <section class="metric-strip overview-summary-strip">
      <div class="metric-strip-item metric-primary">
        <div class="metric-strip-head">
          <span class="metric-label">Sessions</span>
          <ClockCircleOutlined class="metric-strip-icon" />
        </div>
        <div class="metric-strip-value">{{ formatNumber(overview?.totalSessions) }}</div>
        <div class="metric-strip-note">{{ formatDuration(overview?.totalWallDurationMs) }} wall time</div>
      </div>
      <div class="metric-strip-item metric-success">
        <div class="metric-strip-head">
          <span class="metric-label">Tokens</span>
          <FunctionOutlined class="metric-strip-icon" />
        </div>
        <div class="metric-strip-value">{{ formatNumber(overview?.totalTokens) }}</div>
        <div class="metric-strip-note">
          {{ formatNumber(overview?.totalInputTokens) }} in / {{ formatNumber(overview?.totalOutputTokens) }} out /
          {{ formatNumber(overview?.totalCachedInputTokens) }} cached
        </div>
      </div>
      <div class="metric-strip-item metric-warning">
        <div class="metric-strip-head">
          <span class="metric-label">Estimated Cost</span>
          <DollarCircleOutlined class="metric-strip-icon" />
        </div>
        <div class="metric-strip-value">{{ formatCost(overview?.estimatedCostUsd) }}</div>
        <div class="metric-strip-note" :class="{ 'metric-note-warning': (overview?.unpricedSessions || 0) > 0 }">
          {{ formatNumber(overview?.unpricedSessions) }} sessions missing pricing
        </div>
      </div>
      <div class="metric-strip-item metric-info">
        <div class="metric-strip-head">
          <span class="metric-label">Tool Calls</span>
          <ToolOutlined class="metric-strip-icon" />
        </div>
        <div class="metric-strip-value">{{ formatNumber(overview?.totalToolCalls) }}</div>
        <div class="metric-strip-note">Across indexed sessions</div>
      </div>
      <div class="metric-strip-item metric-neutral">
        <div class="metric-strip-head">
          <span class="metric-label">Active Time</span>
          <ClockCircleOutlined class="metric-strip-icon" />
        </div>
        <div class="metric-strip-value">{{ formatDuration(overview?.totalActiveDurationMs) }}</div>
        <div class="metric-strip-note">Measured model and tool time</div>
      </div>
    </section>

    <section v-if="hasIndexedData" class="info-block overview-derived-block">
      <div class="info-block-title">Derived Signals</div>
      <div class="info-block-grid overview-derived-grid">
        <div v-for="item in derivedMetrics" :key="item.label" class="info-stat">
          <div class="info-stat-label">{{ item.label }}</div>
          <div class="info-stat-value">{{ item.value }}</div>
          <div class="metric-note">{{ item.note }}</div>
        </div>
      </div>
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
