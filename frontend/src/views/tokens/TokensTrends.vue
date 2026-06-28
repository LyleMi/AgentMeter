<script setup lang="ts">
import { computed } from 'vue'
import ASpin from 'ant-design-vue/es/spin'
import { CheckCircleOutlined, LineChartOutlined, WarningOutlined } from '@ant-design/icons-vue'
import { formatDisplayNumber, formatNumber, formatPercent } from '../../api'
import CacheHitTrendChart from '../../components/CacheHitTrendChart.vue'
import { useMessages } from '../../i18n'
import { useTokensContext } from './tokensContext'

const { analytics, loading } = useTokensContext()
const { t } = useMessages({
  en: {
    'trend.title': 'Cache Hit Trend',
    'trend.kicker': 'Daily cache reuse for the selected source, model, project, and date scope',
    'metric.latest': 'Latest hit rate',
    'metric.latestNote': '{date} · {count} input tokens',
    'metric.rolling': '7-day weighted',
    'metric.rollingNote': 'Weighted by input tokens to reduce low-volume skew',
    'metric.lowVolume': 'Low-volume days',
    'metric.lowVolumeNote': 'Marked in warning color on the daily line'
  },
  'zh-CN': {
    'trend.title': '缓存命中趋势',
    'trend.kicker': '按当前来源、模型、项目和日期范围展示每日缓存复用',
    'metric.latest': '最近命中率',
    'metric.latestNote': '{date} · {count} 输入 Token',
    'metric.rolling': '7 天加权',
    'metric.rollingNote': '按输入 Token 加权，降低低用量日期偏差',
    'metric.lowVolume': '低用量日期',
    'metric.lowVolumeNote': '在每日曲线上以警示色标记'
  }
})

const trendPoints = computed(() => analytics.value?.cacheHitTrend || [])
const latestPoint = computed(() => [...trendPoints.value].reverse().find((point) => point.hasUsage || point.inputTokens > 0))
const lowVolumeCount = computed(() => trendPoints.value.filter((point) => point.lowInputVolume).length)

const trendMetrics = computed(() => [
  {
    label: t('metric.latest'),
    value: formatPercent(latestPoint.value?.cacheUtilizationRate || 0),
    note: t('metric.latestNote', {
      date: latestPoint.value?.date || '-',
      count: formatDisplayNumber(latestPoint.value?.inputTokens).main
    }),
    icon: CheckCircleOutlined,
    tone: 'metric-success'
  },
  {
    label: t('metric.rolling'),
    value: formatPercent(latestPoint.value?.rollingCacheUtilizationRate || 0),
    note: t('metric.rollingNote'),
    icon: LineChartOutlined,
    tone: 'metric-primary'
  },
  {
    label: t('metric.lowVolume'),
    value: formatNumber(lowVolumeCount.value),
    note: t('metric.lowVolumeNote'),
    icon: WarningOutlined,
    tone: lowVolumeCount.value > 0 ? 'metric-warning' : 'metric-neutral'
  }
])

</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="metric-strip tokens-trend-strip">
        <div v-for="item in trendMetrics" :key="item.label" class="metric-strip-item" :class="item.tone">
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

      <CacheHitTrendChart
        :points="trendPoints"
        :title="t('trend.title')"
        :kicker="t('trend.kicker')"
        :loading="loading"
      />
    </div>
  </a-spin>
</template>

<style scoped>
.tokens-trend-strip {
  grid-template-columns: repeat(3, minmax(180px, 1fr));
}

@media (max-width: 900px) {
  .tokens-trend-strip {
    grid-template-columns: 1fr;
  }
}
</style>
