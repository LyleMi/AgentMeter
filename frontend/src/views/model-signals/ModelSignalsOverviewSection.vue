<script setup lang="ts">
import { computed } from 'vue'
import ATag from 'ant-design-vue/es/tag'
import {
  BranchesOutlined,
  DashboardOutlined,
  ExperimentOutlined,
  LineChartOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import {
  formatDisplayNumber,
  formatNumber,
  type ModelSignalCohort,
  type ModelSignals,
  type ModelSignalsHealthSummary
} from '../../api'
import { useMessages } from '../../i18n'
import { createModelSignalsDisplay } from './display'
import { modelSignalsMessages } from './messages'
import ModelSignalsCohortTable from './ModelSignalsCohortTable.vue'

const props = withDefaults(defineProps<{
  signals?: ModelSignals | null
  healthSummary: ModelSignalsHealthSummary
  rows?: ModelSignalCohort[]
  hasData?: boolean
  loading?: boolean
}>(), {
  signals: null,
  rows: () => [],
  hasData: false,
  loading: false
})

const { t } = useMessages(modelSignalsMessages)
const {
  displayPair,
  displayPercent,
  displayRate,
  displayText,
  formatWindow,
  reasonCount,
  reasonSeverity,
  reasonText,
  severityLabel,
  severityMetricTone,
  severityRank,
  severityTagColor
} = createModelSignalsDisplay(t)

const healthReasonRows = computed(() => (props.healthSummary.topReasons || []).map((reason, index) => ({
  key: `${reasonText(reason)}:${index}`,
  reason: reasonText(reason),
  count: reasonCount(reason),
  severity: reasonSeverity(reason)
})))

const topDriftCohorts = computed(() => {
  const driftRows = props.rows.filter((row) => severityRank(row.drift?.severity) > 0)
  return (driftRows.length ? driftRows : props.rows).slice(0, 8)
})

const metricCards = computed(() => {
  const item = props.signals
  const summary = props.healthSummary
  return [
    {
      label: t('metric.health'),
      value: displayText(severityLabel(summary.severity)),
      note: t('metric.healthNote', {
        current: formatWindow(summary.currentWindow),
        baseline: formatWindow(summary.baselineWindow)
      }),
      icon: DashboardOutlined,
      tone: severityMetricTone(summary.severity)
    },
    {
      label: t('metric.cohorts'),
      value: formatDisplayNumber(summary.cohortCount),
      note: t('metric.cohortsNote', {
        sessions: formatDisplayNumber(item?.totalSessions).main,
        calls: formatDisplayNumber(item?.totalModelCalls).main
      }),
      icon: BranchesOutlined,
      tone: 'metric-primary'
    },
    {
      label: t('metric.criticalWarning'),
      value: displayPair(summary.criticalCohorts, summary.warningCohorts),
      note: t('metric.criticalWarningNote', {
        critical: formatDisplayNumber(summary.criticalCohorts).main,
        warning: formatDisplayNumber(summary.warningCohorts).main
      }),
      icon: WarningOutlined,
      tone: summary.criticalCohorts ? 'metric-danger' : summary.warningCohorts ? 'metric-warning' : 'metric-success'
    },
    {
      label: t('metric.lowConfidence'),
      value: formatDisplayNumber(summary.lowConfidenceCohorts),
      note: t('metric.lowConfidenceNote', { count: formatDisplayNumber(summary.lowConfidenceCohorts).main }),
      icon: ExperimentOutlined,
      tone: summary.lowConfidenceCohorts ? 'metric-warning' : 'metric-neutral'
    },
    {
      label: t('metric.throughput'),
      value: displayRate(item?.modelThroughputTokensPerSecond, ' tok/s', 1),
      note: t('metric.throughputNote'),
      icon: LineChartOutlined,
      tone: 'metric-info'
    },
    {
      label: t('metric.toolFailure'),
      value: displayPercent(item?.toolFailureRate),
      note: t('metric.toolFailureNote', {
        failed: formatDisplayNumber(item?.failedToolCalls).main,
        total: formatDisplayNumber(item?.totalToolCalls).main
      }),
      icon: WarningOutlined,
      tone: item?.failedToolCalls ? 'metric-warning' : 'metric-success'
    }
  ]
})
</script>

<template>
  <div class="section-stack">
    <section class="metric-strip model-signals-metric-strip" :class="{ 'is-empty': !hasData }">
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

    <div v-if="healthReasonRows.length" class="model-signals-reason-strip">
      <span class="metric-label">{{ t('topReasons.label') }}</span>
      <div class="model-signals-tags">
        <a-tag
          v-for="reason in healthReasonRows"
          :key="reason.key"
          :color="severityTagColor(reason.severity)"
        >
          {{ reason.reason }}<span v-if="reason.count"> · {{ formatNumber(reason.count) }}</span>
        </a-tag>
      </div>
    </div>

    <ModelSignalsCohortTable
      variant="overview"
      :rows="topDriftCohorts"
      :loading="loading"
    />
  </div>
</template>

<style scoped>
.model-signals-metric-strip {
  grid-template-columns: repeat(6, minmax(136px, 1fr));
}

.metric-danger {
  --metric-accent: var(--am-danger);
  --metric-soft: var(--am-danger-soft);
}

.model-signals-reason-strip {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 9px 12px;
  background: var(--am-surface);
  border: 1px solid var(--am-border);
  border-radius: var(--am-radius);
}

.model-signals-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.model-signals-tags .ant-tag {
  max-width: 100%;
  margin-right: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1420px) {
  .model-signals-metric-strip {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 980px) {
  .model-signals-metric-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .model-signals-reason-strip {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 640px) {
  .model-signals-metric-strip {
    grid-template-columns: 1fr;
  }
}
</style>
