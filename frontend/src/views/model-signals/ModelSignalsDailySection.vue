<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import ATag from 'ant-design-vue/es/tag'
import AntTable from 'ant-design-vue/es/table'
import ATooltip from 'ant-design-vue/es/tooltip'
import { CalendarOutlined } from '@ant-design/icons-vue'
import {
  formatCost,
  formatDuration,
  formatNumber,
  type ModelSignalsDailyMetric
} from '../../api'
import ModelSignalsMetricChart from '../../components/ModelSignalsMetricChart.vue'
import { useMessages } from '../../i18n'
import {
  buildDailyColumns,
  buildTableLocale
} from './columns'
import { createModelSignalsDisplay } from './display'
import MetricComparisonCell from './MetricComparisonCell.vue'
import { modelSignalsMessages } from './messages'
import ModelSignalsTablePanel from './ModelSignalsTablePanel.vue'
import type { ProjectMetricRow } from './types'

const ATable = AntTable as unknown as DefineComponent

const props = withDefaults(defineProps<{
  rows?: ModelSignalsDailyMetric[]
  projectRows?: ProjectMetricRow[]
  loading?: boolean
}>(), {
  rows: () => [],
  projectRows: () => [],
  loading: false
})

const { t } = useMessages(modelSignalsMessages)
const {
  confidenceReason,
  dailyRowKey,
  driftRowClass,
  failurePressure,
  formatLatency,
  formatOptionalCost,
  formatPercent,
  formatPressure,
  formatRate,
  formatThroughput,
  p10Throughput,
  p90Latency,
  severityLabel,
  severityTagColor,
  unpricedNote
} = createModelSignalsDisplay(t)

const columns = computed(() => buildDailyColumns(t))
const tableLocale = computed(() => buildTableLocale(t, props.loading, 'empty.daily'))
</script>

<template>
  <div class="section-stack">
    <ModelSignalsMetricChart
      :daily-rows="rows"
      :project-rows="projectRows"
      :loading="loading"
      initial-mode="daily"
      :allow-mode-switch="false"
    />

    <ModelSignalsTablePanel :title="t('daily.title')" :kicker="t('daily.kicker')" :icon="CalendarOutlined">
      <a-table
        class="dense-table model-signals-daily-table"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :locale="tableLocale"
        :pagination="{ pageSize: 10 }"
        :row-key="dailyRowKey"
        size="small"
        :custom-row="driftRowClass"
        :scroll="{ x: 1540 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'date'">
            <span class="mono model-signals-date">{{ record.date }}</span>
          </template>
          <template v-else-if="column.key === 'sessions'">
            <MetricComparisonCell
              :primary="formatNumber(record.sessionCount)"
              :secondary="`${formatNumber(record.modelCalls)} ${t('column.modelCalls')}`"
            />
          </template>
          <template v-else-if="column.key === 'cost'">
            <MetricComparisonCell
              :primary="formatCost(record.estimatedCostUsd)"
              :secondary="unpricedNote(record) || formatDuration(record.activeDurationMs)"
            />
          </template>
          <template v-else-if="column.key === 'costPerSession'">
            <span class="number-cell">{{ formatOptionalCost(record.costPerSession) }}</span>
          </template>
          <template v-else-if="column.key === 'costPerActiveHour'">
            <span class="number-cell">{{ formatOptionalCost(record.costPerActiveHour) }}</span>
          </template>
          <template v-else-if="column.key === 'cacheSavings'">
            <span class="number-cell status-ok">{{ formatOptionalCost(record.cacheSavingsUsd) }}</span>
          </template>
          <template v-else-if="column.key === 'p90Latency'">
            <MetricComparisonCell
              :primary="formatLatency(p90Latency(record))"
              :secondary="`p50 ${formatLatency(record.p50ModelLatencyMsPer1kOutputTokens)}`"
            />
          </template>
          <template v-else-if="column.key === 'p10Throughput'">
            <MetricComparisonCell
              :primary="formatThroughput(p10Throughput(record))"
              :secondary="`p50 ${formatThroughput(record.p50ModelThroughputTokensPerSecond)}`"
            />
          </template>
          <template v-else-if="column.key === 'retryPressure'">
            <MetricComparisonCell
              :class="{ 'status-error': record.avgModelCallsPerSession > 1.5 }"
              :primary="`${formatRate(record.avgModelCallsPerSession, 2)}/session`"
              :secondary="`${formatNumber(record.modelCalls)} ${t('column.modelCalls')}`"
            />
          </template>
          <template v-else-if="column.key === 'failurePressure'">
            <MetricComparisonCell
              :class="{ 'status-error': failurePressure(record) > 0 }"
              :primary="formatPressure(failurePressure(record))"
              :secondary="`${formatPercent(record.toolFailureRate)} ${t('column.toolFailure')}`"
            />
          </template>
          <template v-else-if="column.key === 'confidence'">
            <div class="model-signals-confidence-cell">
              <a-tag v-if="record.lowSample" class="status-tag" color="processing">{{ t('label.lowSample') }}</a-tag>
              <a-tag class="status-tag" :color="severityTagColor(record.drift?.severity)">
                {{ severityLabel(record.drift?.severity) }}
              </a-tag>
              <a-tooltip :title="confidenceReason(record)" placement="topLeft">
                <span class="model-signals-reason-text">{{ confidenceReason(record) }}</span>
              </a-tooltip>
            </div>
          </template>
        </template>
      </a-table>
    </ModelSignalsTablePanel>
  </div>
</template>

<style scoped>
.model-signals-date {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: var(--am-text);
  font-variant-numeric: tabular-nums;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.model-signals-confidence-cell {
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr);
  align-items: center;
  gap: 3px;
  min-width: 0;
}

.model-signals-reason-text {
  display: block;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
