<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import AntTable from 'ant-design-vue/es/table'
import { BranchesOutlined, WarningOutlined } from '@ant-design/icons-vue'
import { formatNumber, type ModelSignalCohort } from '../../api'
import { useMessages } from '../../i18n'
import {
  buildCohortColumns,
  buildOverviewColumns,
  buildTableLocale
} from './columns'
import { createModelSignalsDisplay } from './display'
import MetricComparisonCell from './MetricComparisonCell.vue'
import ModelCell from './ModelCell.vue'
import { modelSignalsMessages } from './messages'
import ModelSignalsTablePanel from './ModelSignalsTablePanel.vue'
import ProjectCell from './ProjectCell.vue'
import ReasonTags from './ReasonTags.vue'
import SeverityTag from './SeverityTag.vue'
import SourceCell from './SourceCell.vue'

type CohortTableVariant = 'overview' | 'cohorts'

const ATable = AntTable as unknown as DefineComponent

const props = withDefaults(defineProps<{
  variant: CohortTableVariant
  rows?: ModelSignalCohort[]
  loading?: boolean
}>(), {
  rows: () => [],
  loading: false
})

const { t } = useMessages(modelSignalsMessages)
const {
  cohortRowKey,
  driftRowClass,
  formatConfidence,
  formatLatency,
  formatPercent,
  formatRate,
  metricClass,
  projectInfo,
  severityLabel,
  severityTagColor,
  sourceInfo
} = createModelSignalsDisplay(t)

const isOverview = computed(() => props.variant === 'overview')
const panelTitle = computed(() => isOverview.value ? t('overview.title') : t('cohorts.title'))
const panelKicker = computed(() => isOverview.value ? t('overview.kicker') : t('cohorts.kicker'))
const panelIcon = computed(() => isOverview.value ? WarningOutlined : BranchesOutlined)
const columns = computed(() => isOverview.value ? buildOverviewColumns(t) : buildCohortColumns(t))
const tableLocale = computed(() => buildTableLocale(t, props.loading, isOverview.value ? 'empty.overview' : 'empty.cohorts'))
const tableClass = computed(() => [
  'dense-table',
  isOverview.value ? 'model-signals-overview-table' : 'model-signals-cohort-table'
])
const pagination = computed(() => isOverview.value ? false : { pageSize: 10 })
const scroll = computed(() => ({ x: isOverview.value ? 1340 : 1740 }))
</script>

<template>
  <ModelSignalsTablePanel :title="panelTitle" :kicker="panelKicker" :icon="panelIcon">
    <a-table
      :class="tableClass"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :locale="tableLocale"
      :pagination="pagination"
      :row-key="cohortRowKey"
      size="small"
      :custom-row="driftRowClass"
      :scroll="scroll"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'source'">
          <SourceCell :info="sourceInfo(record)" />
        </template>
        <template v-else-if="column.key === 'project'">
          <ProjectCell :info="projectInfo(record)" />
        </template>
        <template v-else-if="column.key === 'model'">
          <ModelCell :model="record.model" :provider="record.modelProvider" :fallback="t('fallback.unknown')" show-provider />
        </template>
        <template v-else-if="column.key === 'samples'">
          <MetricComparisonCell
            :primary="`${formatNumber(record.sessionCount)} ${t('column.sessions')}`"
            :secondary="`${formatNumber(record.modelCalls)} ${t('column.modelCalls')}`"
          />
        </template>
        <template v-else-if="column.key === 'latency'">
          <MetricComparisonCell
            :class="metricClass(record.current?.modelLatencyMsPer1kOutputTokens, record.baseline?.modelLatencyMsPer1kOutputTokens, true)"
            :primary="formatLatency(record.current?.modelLatencyMsPer1kOutputTokens)"
            :secondary="`${t('metric.baseline')} ${formatLatency(record.baseline?.modelLatencyMsPer1kOutputTokens)}`"
          />
        </template>
        <template v-else-if="column.key === 'throughput'">
          <MetricComparisonCell
            :class="metricClass(record.current?.modelThroughputTokensPerSecond, record.baseline?.modelThroughputTokensPerSecond)"
            :primary="`${formatRate(record.current?.modelThroughputTokensPerSecond, 1)} tok/s`"
            :secondary="`${t('metric.baseline')} ${formatRate(record.baseline?.modelThroughputTokensPerSecond, 1)}`"
          />
        </template>
        <template v-else-if="column.key === 'outputThroughput'">
          <MetricComparisonCell
            :class="metricClass(record.current?.modelThroughputOutputTokensPerSecond, record.baseline?.modelThroughputOutputTokensPerSecond)"
            :primary="formatRate(record.current?.modelThroughputOutputTokensPerSecond, 1)"
            :secondary="`${t('metric.baseline')} ${formatRate(record.baseline?.modelThroughputOutputTokensPerSecond, 1)}`"
          />
        </template>
        <template v-else-if="column.key === 'toolFailure'">
          <MetricComparisonCell
            :class="metricClass(record.current?.toolFailureRate, record.baseline?.toolFailureRate, true)"
            :primary="formatPercent(record.current?.toolFailureRate)"
            :secondary="`${formatNumber(record.failedToolCalls)} / ${formatNumber(record.toolCalls)}`"
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
</template>
