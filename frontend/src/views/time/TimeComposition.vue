<script setup lang="ts">
import { FieldTimeOutlined } from '@ant-design/icons-vue'
import { formatDuration } from '../../api'
import { useMessages } from '../../i18n'
import { useTimeContext } from './timeContext'

const { t } = useMessages({
  en: {
    'title': 'Wall-time composition',
    'kicker': 'How indexed session wall time splits between model, tools, and idle gaps',
    'total': 'Total wall time'
  },
  'zh-CN': {
    'title': '墙钟耗时构成',
    'kicker': '已索引会话的墙钟时间在模型、工具和空闲间隙之间的拆分',
    'total': '总墙钟时间'
  }
})

const { compositionSegments: segments, wallDurationMs, formatPercent } = useTimeContext()
</script>

<template>
  <section class="panel overview-time-composition">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ t('title') }}</h2>
        <div class="panel-kicker">{{ t('kicker') }}</div>
      </div>
      <FieldTimeOutlined class="panel-header-icon" />
    </div>
    <div class="overview-time-composition-body">
      <div class="overview-time-total">
        <span class="metric-label">{{ t('total') }}</span>
        <strong>{{ formatDuration(wallDurationMs) }}</strong>
      </div>
      <div class="overview-time-bar" :aria-label="t('title')">
        <span
          v-for="item in segments"
          :key="item.key"
          :class="['overview-time-bar-segment', item.tone]"
          :style="{ width: item.width }"
          :title="`${item.label}: ${formatDuration(item.value)} (${formatPercent(item.share)})`"
        />
      </div>
      <div class="overview-time-segments">
        <div v-for="item in segments" :key="item.key" class="overview-time-segment">
          <span :class="['overview-time-dot', item.tone]"></span>
          <div>
            <div class="overview-time-segment-label">{{ item.label }}</div>
            <div class="overview-time-segment-value">
              {{ formatDuration(item.value) }}
              <span>{{ formatPercent(item.share) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.overview-time-composition {
  min-width: 0;
}

.overview-time-composition-body {
  display: grid;
  gap: 16px;
  padding: 14px;
}

.overview-time-total {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.overview-time-total strong {
  color: var(--am-text);
  font-size: 28px;
  font-weight: 800;
  line-height: 34px;
  font-variant-numeric: tabular-nums;
}

.overview-time-bar {
  display: flex;
  width: 100%;
  height: 18px;
  overflow: hidden;
  background: var(--am-border-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: 999px;
}

.overview-time-bar-segment {
  display: block;
  min-width: 0;
  height: 100%;
}

.overview-time-bar-segment + .overview-time-bar-segment {
  border-left: 1px solid var(--am-surface);
}

.overview-time-segments {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.overview-time-segment {
  display: flex;
  align-items: flex-start;
  min-width: 0;
  gap: 8px;
  padding: 10px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.overview-time-dot {
  flex: 0 0 auto;
  width: 9px;
  height: 9px;
  margin-top: 5px;
  border-radius: 999px;
}

.overview-time-segment-label {
  color: var(--am-text-soft);
  font-size: 12px;
  font-weight: 720;
  line-height: 18px;
}

.overview-time-segment-value {
  margin-top: 2px;
  color: var(--am-text);
  font-size: 13px;
  font-weight: 750;
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}

.overview-time-segment-value span {
  margin-left: 6px;
  color: var(--am-muted);
  font-size: 12px;
  font-weight: 650;
}

.is-model {
  background: var(--am-primary);
}

.is-network {
  background: var(--am-info);
}

.is-tools {
  background: var(--am-success);
}

.is-idle {
  background: var(--am-warning);
}
</style>
