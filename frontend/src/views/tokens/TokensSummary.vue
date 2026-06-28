<script setup lang="ts">
import { computed } from 'vue'
import AProgress from 'ant-design-vue/es/progress'
import ASpin from 'ant-design-vue/es/spin'
import {
  CheckCircleOutlined,
  DatabaseOutlined,
  DollarCircleOutlined,
  TableOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import {
  formatDisplayCost,
  formatDisplayNumber,
  type TokenAnalytics
} from '../../api'
import CacheHitTrendChart from '../../components/CacheHitTrendChart.vue'
import { useMessages } from '../../i18n'
import { tokenRatioShares } from '../../presentation/tokenRatios'
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
    'breakdown.title': 'Token Mix',
    'breakdown.kicker': 'Input/output totals with cached input and reasoning overhead shape',
    'trend.title': 'Cache Hit Trend',
    'trend.kicker': 'Daily hit rate with input-weighted 7-day trend',
    'token.input': 'Input',
    'token.cached': 'Cached input',
    'token.output': 'Output',
    'token.reasoning': 'Reasoning overhead'
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
    'breakdown.title': 'Token 构成',
    'breakdown.kicker': '输入/输出总量及输入缓存、推理开销形态',
    'trend.title': '缓存命中趋势',
    'trend.kicker': '每日命中率与按输入 Token 加权的 7 天趋势',
    'token.input': '输入',
    'token.cached': '输入缓存',
    'token.output': '输出',
    'token.reasoning': '推理开销'
  }
})

const tokenMix = computed(() => {
  const item = analytics.value
  const shares = tokenRatioShares({
    inputTokens: item?.totalInputTokens,
    cachedInputTokens: item?.totalCachedInputTokens,
    outputTokens: item?.totalOutputTokens,
    reasoningOutputTokens: item?.totalReasoningTokens
  })
  const values = [
    { key: 'input', label: t('token.input'), value: item?.totalInputTokens || 0, share: shares.input, tone: 'is-input' },
    { key: 'cached', label: t('token.cached'), value: item?.totalCachedInputTokens || 0, share: shares.cachedInput, tone: 'is-cached' },
    { key: 'output', label: t('token.output'), value: item?.totalOutputTokens || 0, share: shares.output, tone: 'is-output' },
    { key: 'reasoning', label: t('token.reasoning'), value: item?.totalReasoningTokens || 0, share: shares.reasoningOutput, tone: 'is-reasoning' }
  ]
  return values.map((current) => ({
    ...current,
    display: formatDisplayNumber(current.value)
  }))
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
      </div>
    </div>
  </a-spin>
</template>

<style scoped>
.tokens-metric-strip {
  grid-template-columns: repeat(4, minmax(160px, 1fr));
}

.tokens-summary-grid {
  display: grid;
  grid-template-columns: minmax(360px, 0.9fr) minmax(520px, 1.1fr);
  gap: var(--am-section-gap);
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
}
</style>
