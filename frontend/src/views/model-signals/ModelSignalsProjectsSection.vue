<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import AntTable from 'ant-design-vue/es/table'
import ATooltip from 'ant-design-vue/es/tooltip'
import { DashboardOutlined } from '@ant-design/icons-vue'
import {
  formatCost,
  formatNumber,
  type ModelSignalsDailyMetric
} from '../../api'
import ModelSignalsMetricChart from '../../components/ModelSignalsMetricChart.vue'
import { useMessages } from '../../i18n'
import {
  buildProjectColumns,
  buildTableLocale
} from './columns'
import { createModelSignalsDisplay } from './display'
import MetricComparisonCell from './MetricComparisonCell.vue'
import { modelSignalsMessages } from './messages'
import ModelSignalsTablePanel from './ModelSignalsTablePanel.vue'
import ProjectCell from './ProjectCell.vue'
import ReasonTags from './ReasonTags.vue'
import SeverityTag from './SeverityTag.vue'
import type { ProjectMetricRow } from './types'

const ATable = AntTable as unknown as DefineComponent

const props = withDefaults(defineProps<{
  dailyRows?: ModelSignalsDailyMetric[]
  rows?: ProjectMetricRow[]
  hasProjectMetrics?: boolean
  loading?: boolean
}>(), {
  dailyRows: () => [],
  rows: () => [],
  hasProjectMetrics: false,
  loading: false
})

const { t } = useMessages(modelSignalsMessages)
const {
  driftRowClass,
  failurePressure,
  formatConfidence,
  formatLatency,
  formatOptionalCost,
  formatPressure,
  formatRate,
  formatThroughput,
  metricClass,
  p10Throughput,
  p90Latency,
  projectHealthTitle,
  projectInfo,
  projectMixInfo,
  projectRowKey,
  severityLabel,
  severityTagColor,
  unpricedNote
} = createModelSignalsDisplay(t)

const columns = computed(() => buildProjectColumns(t, props.hasProjectMetrics))
const tableLocale = computed(() => buildTableLocale(t, props.loading, 'empty.projects'))
</script>

<template>
  <div class="section-stack">
    <ModelSignalsMetricChart
      :daily-rows="dailyRows"
      :project-rows="rows"
      :loading="loading"
      initial-mode="projects"
      :allow-mode-switch="false"
    />

    <ModelSignalsTablePanel :title="t('projects.title')" :kicker="t('projects.kicker')" :icon="DashboardOutlined">
      <a-table
        class="dense-table model-signals-project-table"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :locale="tableLocale"
        :pagination="{ pageSize: 10 }"
        :row-key="projectRowKey"
        size="small"
        :custom-row="driftRowClass"
        :scroll="{ x: 1580 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'project'">
            <ProjectCell :info="projectInfo(record)" show-meta />
          </template>
          <template v-else-if="column.key === 'sessions'">
            <span v-if="!hasProjectMetrics" class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
            <MetricComparisonCell
              v-else
              :primary="formatNumber(record.sessionCount)"
              :secondary="`${formatNumber(record.totalTokens)} ${t('column.tokens')}`"
            />
          </template>
          <template v-else-if="column.key === 'sources'"><span class="number-cell">{{ formatNumber(record.sourceCount) }}</span></template>
          <template v-else-if="column.key === 'models'"><span class="number-cell">{{ formatNumber(record.modelCount) }}</span></template>
          <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.totalTokens) }}</span></template>
          <template v-else-if="column.key === 'mix'">
            <a-tooltip :title="projectMixInfo(record).full" placement="topLeft">
              <div class="model-signals-mix-cell">
                <span class="model-name">{{ projectMixInfo(record).model }}</span>
                <span class="source-identity-meta">{{ projectMixInfo(record).summary || projectMixInfo(record).provider || '-' }}</span>
              </div>
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'costBurn'">
            <MetricComparisonCell
              :primary="formatCost(record.current?.estimatedCostUsd)"
              :secondary="unpricedNote(record.current) || formatOptionalCost(record.current?.costPerActiveHour)"
            />
          </template>
          <template v-else-if="column.key === 'cacheSavings'">
            <span class="number-cell status-ok">{{ formatOptionalCost(record.current?.cacheSavingsUsd) }}</span>
          </template>
          <template v-else-if="column.key === 'health'">
            <a-tooltip :title="projectHealthTitle(record)" placement="topLeft">
              <div class="model-signals-health-cell">
                <SeverityTag :color="severityTagColor(record.drift?.severity)" :label="severityLabel(record.drift?.severity)" />
                <span class="source-identity-meta">{{ formatConfidence(record.drift?.confidence) }}</span>
              </div>
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'latency'">
            <MetricComparisonCell
              :class="metricClass(p90Latency(record.current), p90Latency(record.baseline), true)"
              :primary="formatLatency(p90Latency(record.current))"
              :secondary="`${t('metric.baseline')} ${formatLatency(p90Latency(record.baseline))}`"
            />
          </template>
          <template v-else-if="column.key === 'throughput'">
            <MetricComparisonCell
              :class="metricClass(p10Throughput(record.current), p10Throughput(record.baseline))"
              :primary="formatThroughput(p10Throughput(record.current))"
              :secondary="`${t('metric.baseline')} ${formatThroughput(p10Throughput(record.baseline))}`"
            />
          </template>
          <template v-else-if="column.key === 'pressure'">
            <MetricComparisonCell
              :class="metricClass(failurePressure(record.current), failurePressure(record.baseline), true)"
              :primary="formatPressure(failurePressure(record.current))"
              :secondary="`${formatRate(record.current?.avgModelCallsPerSession, 2)}/session`"
            />
          </template>
          <template v-else-if="column.key === 'severity'">
            <SeverityTag :color="severityTagColor(record.drift?.severity)" :label="severityLabel(record.drift?.severity)" />
          </template>
          <template v-else-if="column.key === 'confidence'">
            <span class="number-cell">{{ formatConfidence(record.drift?.confidence) }}</span>
            <div v-if="record.drift?.sampleNote" class="source-identity-meta">{{ record.drift.sampleNote }}</div>
          </template>
          <template v-else-if="column.key === 'reasons'">
            <ReasonTags
              :reasons="record.drift?.reasons"
              :color="severityTagColor(record.drift?.severity)"
              :empty-text="t('fallback.noReason')"
            />
          </template>
        </template>
      </a-table>
    </ModelSignalsTablePanel>
  </div>
</template>

<style scoped>
.model-signals-health-cell,
.model-signals-mix-cell {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.model-signals-health-cell {
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
}

.model-signals-mix-cell .model-name {
  display: block;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-mix-cell .source-identity-meta,
.model-signals-health-cell .source-identity-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
