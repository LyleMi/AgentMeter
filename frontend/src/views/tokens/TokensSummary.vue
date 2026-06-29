<script setup lang="ts">
import { computed } from 'vue'
import AProgress from 'ant-design-vue/es/progress'
import ASpin from 'ant-design-vue/es/spin'
import {
  BarChartOutlined,
  CheckCircleOutlined,
  CompressOutlined,
  DatabaseOutlined,
  DollarCircleOutlined,
  TableOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import {
  formatDisplayCost,
  formatDisplayNumber,
  type AgentUsage,
  type TokenAnalytics
} from '../../api'
import CacheHitTrendChart from '../../components/CacheHitTrendChart.vue'
import { useMessages } from '../../i18n'
import { sourceDisplay } from '../../presentation/sourceIdentity'
import { cachedInputRatio, tokenRatioShares } from '../../presentation/tokenRatios'
import { useTokensContext } from './tokensContext'

const { analytics, loading } = useTokensContext()
const { t } = useMessages({
  en: {
    'metric.totalTokens': 'Total Tokens',
    'metric.totalTokensNote': '{count} indexed sessions',
    'metric.cacheRate': 'Cache Utilization',
    'metric.cacheRateNote': '{count} cached input tokens',
    'metric.cost': 'Estimated Cost',
    'metric.costNoteCovered': 'Pricing covered for indexed usage',
    'metric.costNoteMissing': '{count} unpriced usage rows',
    'metric.inputOutput': 'Input / Output',
    'metric.inputOutputNote': 'Prompt and response volume',
    'metric.contextCompression': 'Context Compression',
    'metric.contextCompressionNote': '{percent} of total token volume',
    'breakdown.title': 'Token Mix',
    'breakdown.kicker': 'Input/output totals with cached input, reasoning, and context compression separated',
    'trend.title': 'Cache Hit Trend',
    'trend.kicker': 'Daily hit rate with input-weighted 7-day trend',
    'sourceCache.title': 'Source Cache Hit Rate',
    'sourceCache.kicker': 'Horizontal comparison across local sources, with input volume beside each rate',
    'sourceCache.input': '{count} input',
    'sourceCache.emptyTitle': 'No source cache rate yet',
    'sourceCache.emptyText': 'Cache hit rates appear after indexed source sessions include input token usage.',
    'token.input': 'Input',
    'token.cached': 'Cached input',
    'token.output': 'Output',
    'token.reasoning': 'Reasoning overhead',
    'token.contextCompression': 'Context compression',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    'metric.totalTokens': '总 Token',
    'metric.totalTokensNote': '已索引 {count} 个会话',
    'metric.cacheRate': '缓存利用率',
    'metric.cacheRateNote': '{count} 个缓存输入 Token',
    'metric.cost': '预估费用',
    'metric.costNoteCovered': '已索引用量均有价格覆盖',
    'metric.costNoteMissing': '{count} 条用量缺少价格',
    'metric.inputOutput': '输入 / 输出',
    'metric.inputOutputNote': '提示词和响应规模',
    'metric.contextCompression': '上下文压缩',
    'metric.contextCompressionNote': '占总 Token 的 {percent}',
    'breakdown.title': 'Token 构成',
    'breakdown.kicker': '单独展示输入/输出、输入缓存、推理和上下文压缩',
    'trend.title': '缓存命中趋势',
    'trend.kicker': '每日命中率与按输入 Token 加权的 7 天趋势',
    'sourceCache.title': '来源缓存命中率',
    'sourceCache.kicker': '横向对比各本地来源，右侧保留输入规模',
    'sourceCache.input': '{count} 输入',
    'sourceCache.emptyTitle': '暂无来源缓存命中率',
    'sourceCache.emptyText': '索引到包含输入 Token 的来源会话后，这里会显示缓存命中率。',
    'token.input': '输入',
    'token.cached': '输入缓存',
    'token.output': '输出',
    'token.reasoning': '推理开销',
    'token.contextCompression': '上下文压缩',
    'fallback.unknown': '未知'
  }
})

interface SourceCacheRow {
  key: string
  label: string
  secondary: string
  title: string
  inputTokens: number
  cachedInputTokens: number
  rate: number
  rateLabel: string
  width: string
  inputLabel: string
  inputTitle: string
}

const tokenMix = computed(() => {
  const item = analytics.value
  const shares = tokenRatioShares({
    inputTokens: item?.totalInputTokens,
    cachedInputTokens: item?.totalCachedInputTokens,
    outputTokens: item?.totalOutputTokens,
    reasoningOutputTokens: item?.totalReasoningTokens
  })
  const compressionShare = contextCompressionShare(item)
  const values = [
    { key: 'input', label: t('token.input'), value: item?.totalInputTokens || 0, share: shares.input, tone: 'is-input' },
    { key: 'cached', label: t('token.cached'), value: item?.totalCachedInputTokens || 0, share: shares.cachedInput, tone: 'is-cached' },
    { key: 'output', label: t('token.output'), value: item?.totalOutputTokens || 0, share: shares.output, tone: 'is-output' },
    { key: 'reasoning', label: t('token.reasoning'), value: item?.totalReasoningTokens || 0, share: shares.reasoningOutput, tone: 'is-reasoning' },
    { key: 'contextCompression', label: t('token.contextCompression'), value: item?.totalContextCompressionTokens || 0, share: compressionShare, tone: 'is-compression' }
  ]
  return values.map((current) => ({
    ...current,
    display: formatDisplayNumber(current.value)
  }))
})

const sourceCacheRows = computed<SourceCacheRow[]>(() => {
  const rows = (analytics.value?.agentUsage || []).map((item, index) => {
    const source = sourceDisplay(item, t('fallback.unknown'))
    const rate = sourceCacheRate(item)
    const inputTokens = item.inputTokens || 0
    const cachedInputTokens = item.cachedInputTokens || 0
    const inputDisplay = formatDisplayNumber(inputTokens)
    return {
      key: source.key || `${item.agentKind || 'source'}:${item.agentName || index}`,
      label: source.label,
      secondary: source.secondary,
      title: source.title,
      inputTokens,
      cachedInputTokens,
      rate,
      rateLabel: formatRateLabel(rate),
      width: sourceCacheWidth(rate),
      inputLabel: t('sourceCache.input', { count: inputDisplay.main }),
      inputTitle: t('sourceCache.input', { count: inputDisplay.full })
    }
  }).filter((row) => row.inputTokens > 0 || row.cachedInputTokens > 0)

  return rows.sort((left, right) => right.rate - left.rate || right.inputTokens - left.inputTokens || left.label.localeCompare(right.label))
})

const cacheRatePercent = computed(() => Math.round(Math.max(0, Math.min(1, analytics.value?.cacheUtilizationRate || 0)) * 100))
const metricCards = computed(() => {
  const item = analytics.value
  return [
    {
      label: t('metric.totalTokens'),
      value: formatDisplayNumber(item?.totalTokens),
      note: t('metric.totalTokensNote', { count: formatDisplayNumber(item?.totalSessions).main }),
      icon: DatabaseOutlined,
      tone: 'metric-primary'
    },
    {
      label: t('metric.cacheRate'),
      value: { main: `${cacheRatePercent.value}%`, full: `${cacheRatePercent.value}%`, suffix: '' },
      note: t('metric.cacheRateNote', { count: formatDisplayNumber(item?.totalCachedInputTokens).main }),
      icon: CheckCircleOutlined,
      tone: 'metric-success'
    },
    {
      label: t('metric.cost'),
      value: formatDisplayCost(item?.estimatedCostUsd),
      note: item?.unpricedCount ? t('metric.costNoteMissing', { count: formatDisplayNumber(item.unpricedCount).main }) : t('metric.costNoteCovered'),
      icon: item?.unpricedCount ? WarningOutlined : DollarCircleOutlined,
      tone: item?.unpricedCount ? 'metric-warning' : 'metric-info'
    },
    {
      label: t('metric.inputOutput'),
      value: displayPair(item),
      note: t('metric.inputOutputNote'),
      icon: TableOutlined,
      tone: 'metric-neutral'
    },
    {
      label: t('metric.contextCompression'),
      value: formatDisplayNumber(item?.totalContextCompressionTokens),
      note: t('metric.contextCompressionNote', { percent: mixPercent(contextCompressionShare(item)) }),
      icon: CompressOutlined,
      tone: 'metric-neutral'
    }
  ]
})

function displayPair(item: TokenAnalytics | null) {
  const leftDisplay = formatDisplayNumber(item?.totalInputTokens)
  const rightDisplay = formatDisplayNumber(item?.totalOutputTokens)
  return {
    main: `${leftDisplay.main} / ${rightDisplay.main}`,
    suffix: '',
    full: `${leftDisplay.full} / ${rightDisplay.full}`
  }
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

function contextCompressionShare(item: TokenAnalytics | null | undefined) {
  const value = item?.totalContextCompressionTokens || 0
  const total = item?.totalTokens || 0
  if (value <= 0) return 0
  if (total <= 0) return 1
  return clampRatio(value / total)
}

function sourceCacheRate(item: AgentUsage) {
  if (Number.isFinite(item.cacheUtilizationRate)) return clampRatio(item.cacheUtilizationRate)
  return cachedInputRatio(item.inputTokens, item.cachedInputTokens)
}

function sourceCacheWidth(rate: number) {
  return `${Math.round(clampRatio(rate) * 1000) / 10}%`
}

function formatRateLabel(rate: number) {
  const normalized = clampRatio(rate)
  if (normalized > 0 && normalized < 0.01) return '<1%'
  return `${Math.round(normalized * 100)}%`
}

function sourceCacheAria(row: SourceCacheRow) {
  return `${row.label}: ${row.rateLabel}, ${row.inputTitle}`
}

function clampRatio(value: number) {
  if (!Number.isFinite(value)) return 0
  return Math.max(0, Math.min(1, value))
}
</script>

<template>
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
          <div class="metric-strip-value" :title="item.value.full">{{ item.value.main }}</div>
          <div class="metric-strip-note">{{ item.note }}</div>
        </div>
      </section>

      <div class="tokens-summary-grid">
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
                <strong :title="item.display.full">{{ item.display.main }}</strong>
              </div>
              <div class="tokens-mix-meter">
                <span :style="{ width: mixWidth(item.share) }"></span>
              </div>
              <div class="tokens-mix-share">{{ mixPercent(item.share) }}</div>
            </div>
          </div>
        </section>

        <CacheHitTrendChart
          :points="analytics?.cacheHitTrend || []"
          :title="t('trend.title')"
          :kicker="t('trend.kicker')"
          compact
          :loading="loading"
        />

        <section class="panel source-cache-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('sourceCache.title') }}</h2>
              <div class="panel-kicker">{{ t('sourceCache.kicker') }}</div>
            </div>
            <BarChartOutlined class="panel-header-icon" />
          </div>
          <div v-if="sourceCacheRows.length" class="source-cache-list" role="list">
            <div
              v-for="item in sourceCacheRows"
              :key="item.key"
              class="source-cache-row"
              :class="{ 'is-zero': item.rate <= 0 }"
              role="listitem"
            >
              <div class="source-cache-label">
                <span class="source-cache-name" :title="item.title">{{ item.label }}</span>
                <span class="source-cache-meta" :title="item.title">{{ item.secondary || '-' }}</span>
              </div>
              <div class="source-cache-track" :aria-label="sourceCacheAria(item)">
                <span class="source-cache-fill" :style="{ width: item.width }"></span>
              </div>
              <div class="source-cache-values">
                <strong>{{ item.rateLabel }}</strong>
                <span :title="item.inputTitle">{{ item.inputLabel }}</span>
              </div>
            </div>
          </div>
          <div v-else class="empty-state empty-state-compact">
            <BarChartOutlined class="empty-state-icon" />
            <div class="empty-state-title">{{ t('sourceCache.emptyTitle') }}</div>
            <div class="empty-state-text">{{ t('sourceCache.emptyText') }}</div>
          </div>
        </section>
      </div>
    </div>
  </a-spin>
</template>

<style scoped>
.tokens-metric-strip {
  grid-template-columns: repeat(5, minmax(150px, 1fr));
}

.tokens-summary-grid {
  display: grid;
  grid-template-columns: minmax(360px, 0.9fr) minmax(520px, 1.1fr);
  gap: var(--am-section-gap);
}

.source-cache-panel {
  grid-column: 1 / -1;
}

.source-cache-list {
  display: grid;
  gap: 10px;
}

.source-cache-row {
  display: grid;
  grid-template-columns: minmax(150px, 0.34fr) minmax(160px, 1fr) minmax(112px, auto);
  align-items: center;
  gap: 12px;
  min-height: 58px;
  padding: 10px 12px;
  background: linear-gradient(90deg, var(--am-surface-subtle), var(--am-surface));
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.source-cache-label {
  min-width: 0;
}

.source-cache-name,
.source-cache-meta {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-cache-name {
  color: var(--am-text);
  font-size: 13px;
  font-weight: 700;
}

.source-cache-meta {
  margin-top: 3px;
  color: var(--am-muted);
  font-size: 12px;
}

.source-cache-track {
  position: relative;
  height: 16px;
  padding: 2px;
  overflow: hidden;
  background:
    repeating-linear-gradient(90deg, transparent 0 calc(20% - 1px), rgb(100 116 139 / 18%) calc(20% - 1px) 20%),
    var(--am-surface);
  border: 1px solid var(--am-border-subtle);
  border-radius: 999px;
}

.source-cache-fill {
  position: relative;
  z-index: 1;
  display: block;
  height: 100%;
  min-width: 3px;
  background: linear-gradient(90deg, var(--am-success), var(--am-primary));
  border-radius: 999px;
}

.source-cache-row.is-zero .source-cache-fill {
  min-width: 0;
}

.source-cache-values {
  min-width: 0;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.source-cache-values strong {
  display: block;
  color: var(--am-text);
  font-size: 18px;
  line-height: 1.1;
}

.source-cache-values span {
  display: block;
  margin-top: 3px;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  grid-template-columns: repeat(2, minmax(0, 1fr));
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

.tokens-mix-item.is-compression {
  --token-accent: var(--am-danger);
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
  background: var(--am-border-subtle);
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

@media (max-width: 1200px) {
  .tokens-metric-strip,
  .tokens-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 860px) {
  .tokens-summary-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 760px) {
  .tokens-metric-strip,
  .tokens-mix-list {
    grid-template-columns: 1fr;
  }

  .source-cache-row {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .source-cache-values {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
    text-align: left;
  }
}
</style>
