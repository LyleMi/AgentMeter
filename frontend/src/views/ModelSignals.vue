<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import {
  ArrowRightOutlined,
  BranchesOutlined,
  CalendarOutlined,
  DashboardOutlined,
  ExperimentOutlined,
  LineChartOutlined,
  TableOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatCost,
  formatDateTime,
  formatDisplayNumber,
  formatDuration,
  formatNumber,
  type ModelSignals
} from '../api'
import ModelSignalsMetricChart from '../components/ModelSignalsMetricChart.vue'
import PageHeader from '../components/PageHeader.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import Panel from '../components/ui/Panel.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import {
  buildUsageAgentOptions,
  buildUsageModelOptions,
  buildUsageProjectOptions,
  useUsageScopeOptionData
} from './useUsageScopeOptions'
import {
  buildAnomalyColumns,
  buildCohortColumns,
  buildDailyColumns,
  buildMatrixColumns,
  buildOverviewColumns,
  buildProjectColumns,
  buildTableLocale
} from './model-signals/columns'
import { createModelSignalsDisplay } from './model-signals/display'
import MetricComparisonCell from './model-signals/MetricComparisonCell.vue'
import ModelCell from './model-signals/ModelCell.vue'
import { modelSignalsMessages } from './model-signals/messages'
import ProjectCell from './model-signals/ProjectCell.vue'
import ReasonTags from './model-signals/ReasonTags.vue'
import SeverityTag from './model-signals/SeverityTag.vue'
import SourceCell from './model-signals/SourceCell.vue'
import SourceModelComparison from './model-signals/SourceModelComparison.vue'
import { buildModelSignalsTabs, type ModelSignalsTabKey } from './model-signals/tabs'
import type { ProjectMetricRow } from './model-signals/types'

const ATable = AntTable as unknown as DefineComponent

const router = useRouter()
const resource = useAsyncResource<ModelSignals | null>(null)
const signals = computed(() => resource.data.value)
const loading = resource.loading
const error = resource.error
const activeTab = ref<ModelSignalsTabKey>('charts')
const scope = useUsageScopeRoute(() => {
  void load()
})
const scopeOptionData = useUsageScopeOptionData()

const { t } = useMessages(modelSignalsMessages)
const {
  anomalyRowClass,
  anomalyRowKey,
  cohortRowKey,
  confidenceReason,
  dailyRowKey,
  displayPair,
  displayPercent,
  displayRate,
  displayText,
  driftRowClass,
  failurePressure,
  fallbackHealthSummary,
  formatConfidence,
  formatLatency,
  formatOptionalCost,
  formatPercent,
  formatPressure,
  formatRate,
  formatThroughput,
  formatWindow,
  matrixCellKey,
  matrixCellTitle,
  matrixRowKey,
  metricClass,
  normalizeAnomaly,
  p10Throughput,
  p90Latency,
  projectHealthTitle,
  projectInfo,
  projectMixInfo,
  projectRowKey,
  reasonCount,
  reasonSeverity,
  reasonText,
  sessionInfo,
  severityClass,
  severityLabel,
  severityMetricTone,
  severityRank,
  severityTagColor,
  sourceInfo,
  unpricedNote
} = createModelSignalsDisplay(t)

const healthSummary = computed(() => signals.value?.healthSummary || fallbackHealthSummary(signals.value))
const cohortRows = computed(() => signals.value?.cohorts || [])
const matrixRows = computed(() => signals.value?.matrix || [])
const matrixCells = computed(() => matrixRows.value.flatMap((row) => row.cells || []))
const projectHotspotRows = computed(() => signals.value?.projectHotspots || [])
const dailyMetricRows = computed(() => signals.value?.dailyMetrics || [])
const projectMetricRows = computed(() => signals.value?.projectMetrics || [])
const hasProjectMetrics = computed(() => projectMetricRows.value.length > 0)
const projectRows = computed<ProjectMetricRow[]>(() => hasProjectMetrics.value ? projectMetricRows.value : projectHotspotRows.value)
const normalizedAnomalies = computed(() => (signals.value?.anomalySessions || []).map(normalizeAnomaly))
const hasData = computed(() => Boolean(signals.value?.totalSessions || healthSummary.value.cohortCount || dailyMetricRows.value.length))

const agentOptions = computed(() =>
  buildUsageAgentOptions({
    sources: [
      cohortRows.value,
      matrixRows.value,
      scopeOptionData.optionOverview.value?.agentUsage,
      scopeOptionData.optionOverview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.slowSessions,
      normalizedAnomalies.value
    ],
    selected: scope.filters.value.agent,
    fallback: t('fallback.unknown')
  })
)

const modelOptions = computed(() =>
  buildUsageModelOptions({
    modelUsage: [
      cohortRows.value,
      matrixCells.value,
      signals.value?.modelBreakdown,
      scopeOptionData.optionOverview.value?.modelUsage
    ],
    sessions: [
      normalizedAnomalies.value,
      scopeOptionData.optionOverview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.model
  })
)

const projectOptions = computed(() =>
  buildUsageProjectOptions({
    projects: [
      cohortRows.value,
      projectRows.value,
      scopeOptionData.projectOptionRows.value,
      normalizedAnomalies.value,
      scopeOptionData.optionOverview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.project,
    fallback: t('fallback.unknown')
  })
)

const healthReasonRows = computed(() => (healthSummary.value.topReasons || []).map((reason, index) => ({
  key: `${reasonText(reason)}:${index}`,
  reason: reasonText(reason),
  count: reasonCount(reason),
  severity: reasonSeverity(reason)
})))

const topDriftCohorts = computed(() => {
  const driftRows = cohortRows.value.filter((row) => severityRank(row.drift?.severity) > 0)
  return (driftRows.length ? driftRows : cohortRows.value).slice(0, 8)
})

const metricCards = computed(() => {
  const item = signals.value
  const summary = healthSummary.value
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

const tabs = computed(() => buildModelSignalsTabs(t))
const overviewColumns = computed(() => buildOverviewColumns(t))
const dailyColumns = computed(() => buildDailyColumns(t))
const cohortColumns = computed(() => buildCohortColumns(t))
const matrixColumns = computed(() => buildMatrixColumns(t))
const projectColumns = computed(() => buildProjectColumns(t, hasProjectMetrics.value))
const anomalyColumns = computed(() => buildAnomalyColumns(t))

const overviewTableLocale = computed(() => buildTableLocale(t, loading.value, 'empty.overview'))
const dailyTableLocale = computed(() => buildTableLocale(t, loading.value, 'empty.daily'))
const cohortTableLocale = computed(() => buildTableLocale(t, loading.value, 'empty.cohorts'))
const matrixTableLocale = computed(() => buildTableLocale(t, loading.value, 'empty.matrix'))
const projectTableLocale = computed(() => buildTableLocale(t, loading.value, 'empty.projects'))
const anomalyTableLocale = computed(() => buildTableLocale(t, loading.value, 'empty.anomalies'))

async function load() {
  return resource.run(async () => {
    const filters = scope.apiFilters.value
    const [nextSignals, optionData] = await Promise.all([
      api.getModelSignals(filters),
      scopeOptionData.loadUsageScopeOptionData()
    ])
    scopeOptionData.applyUsageScopeOptionData(optionData)
    return nextSignals
  }, { onErrorData: null })
}

async function updateScopeFilters(nextFilters: UsageScopeForm) {
  await scope.updateFilters(nextFilters)
  await load()
}

async function clearScopeFilters() {
  await scope.clearFilters()
  await load()
}

function openSession(id: number) {
  if (id) router.push(`/sessions/${id}`)
}

onMounted(load)
</script>

<template>
  <div class="page model-signals-page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')" />

    <UsageScopeBar
      :filters="scope.filters.value"
      :agent-options="agentOptions"
      :model-options="modelOptions"
      :project-options="projectOptions"
      :loading="loading"
      @update:filters="updateScopeFilters"
      @refresh="load"
      @clear="clearScopeFilters"
    />

    <a-alert
      v-if="error"
      class="model-signals-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <a-spin :spinning="loading && !signals">
      <div class="section-stack">
        <div class="model-signals-tabs" role="tablist" :aria-label="t('title')">
          <a-button
            v-for="tab in tabs"
            :key="tab.key"
            :type="activeTab === tab.key ? 'primary' : 'default'"
            role="tab"
            :aria-selected="activeTab === tab.key"
            @click="activeTab = tab.key"
          >
            <template #icon>
              <component :is="tab.icon" />
            </template>
            {{ tab.label }}
          </a-button>
        </div>

        <div v-if="activeTab === 'charts'">
          <ModelSignalsMetricChart
            :daily-rows="dailyMetricRows"
            :project-rows="projectRows"
            :loading="loading"
          />
        </div>

        <div v-else-if="activeTab === 'overview'">
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

            <Panel
              class="model-signals-table-panel"
              :title="t('overview.title')"
              :kicker="t('overview.kicker')"
              :icon="WarningOutlined"
            >
              <a-table
                class="dense-table model-signals-overview-table"
                :columns="overviewColumns"
                :data-source="topDriftCohorts"
                :loading="loading"
                :locale="overviewTableLocale"
                :pagination="false"
                :row-key="cohortRowKey"
                size="small"
                :custom-row="driftRowClass"
                :scroll="{ x: 1340 }"
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
                  <template v-else-if="column.key === 'severity'">
                    <SeverityTag :color="severityTagColor(record.drift?.severity)" :label="severityLabel(record.drift?.severity)" />
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
            </Panel>
          </div>
        </div>

        <div v-else-if="activeTab === 'daily'">
          <div class="section-stack">
            <ModelSignalsMetricChart
              :daily-rows="dailyMetricRows"
              :project-rows="projectRows"
              :loading="loading"
              initial-mode="daily"
              :allow-mode-switch="false"
            />

            <Panel
              class="model-signals-table-panel"
              :title="t('daily.title')"
              :kicker="t('daily.kicker')"
              :icon="CalendarOutlined"
            >
              <a-table
                class="dense-table model-signals-daily-table"
                :columns="dailyColumns"
                :data-source="dailyMetricRows"
                :loading="loading"
                :locale="dailyTableLocale"
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
            </Panel>
          </div>
        </div>

        <div v-else-if="activeTab === 'cohorts'">
          <Panel
            class="model-signals-table-panel"
            :title="t('cohorts.title')"
            :kicker="t('cohorts.kicker')"
            :icon="BranchesOutlined"
          >
            <a-table
              class="dense-table model-signals-cohort-table"
              :columns="cohortColumns"
              :data-source="cohortRows"
              :loading="loading"
              :locale="cohortTableLocale"
              :pagination="{ pageSize: 10 }"
              :row-key="cohortRowKey"
              size="small"
              :custom-row="driftRowClass"
              :scroll="{ x: 1740 }"
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
          </Panel>
        </div>

        <div v-else-if="activeTab === 'matrix'">
          <div class="section-stack">
            <SourceModelComparison :rows="matrixRows" :loading="loading" />

            <Panel
              class="model-signals-table-panel"
              :title="t('matrix.title')"
              :kicker="t('matrix.kicker')"
              :icon="TableOutlined"
            >
              <a-table
                class="dense-table model-signals-matrix-table"
                :columns="matrixColumns"
                :data-source="matrixRows"
                :loading="loading"
                :locale="matrixTableLocale"
                :pagination="false"
                :row-key="matrixRowKey"
                size="small"
                :scroll="{ x: 980 }"
                >
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'source'">
                    <SourceCell :info="sourceInfo(record)" />
                  </template>
                  <template v-else-if="column.key === 'models'">
                    <div class="model-signals-matrix-cells">
                      <a-tooltip
                        v-for="cell in record.cells"
                        :key="matrixCellKey(cell)"
                        :title="matrixCellTitle(cell)"
                        placement="topLeft"
                      >
                        <div class="model-signals-matrix-cell" :class="severityClass(cell.severity)">
                          <div class="model-signals-matrix-cell-head">
                            <span class="model-name">{{ cell.model || t('fallback.unknown') }}</span>
                            <SeverityTag :color="severityTagColor(cell.severity)" :label="severityLabel(cell.severity)" />
                          </div>
                          <div class="source-identity-meta">{{ cell.modelProvider || '-' }} · {{ formatNumber(cell.cohortCount) }} {{ t('tab.cohorts') }}</div>
                          <div class="model-signals-matrix-metrics">
                            <span>{{ formatLatency(cell.current?.modelLatencyMsPer1kOutputTokens) }}</span>
                            <span>{{ formatRate(cell.current?.modelThroughputTokensPerSecond, 1) }} tok/s</span>
                            <span>{{ formatConfidence(cell.confidence) }}</span>
                          </div>
                          <div class="model-signals-matrix-reason">{{ cell.keyReason || t('fallback.noReason') }}</div>
                        </div>
                      </a-tooltip>
                    </div>
                  </template>
                </template>
              </a-table>
            </Panel>
          </div>
        </div>

        <div v-else-if="activeTab === 'projects'">
          <div class="section-stack">
            <ModelSignalsMetricChart
              :daily-rows="dailyMetricRows"
              :project-rows="projectRows"
              :loading="loading"
              initial-mode="projects"
              :allow-mode-switch="false"
            />

            <Panel
              class="model-signals-table-panel"
              :title="t('projects.title')"
              :kicker="t('projects.kicker')"
              :icon="DashboardOutlined"
            >
              <a-table
                class="dense-table model-signals-project-table"
                :columns="projectColumns"
                :data-source="projectRows"
                :loading="loading"
                :locale="projectTableLocale"
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
            </Panel>
          </div>
        </div>

        <div v-else-if="activeTab === 'anomalies'">
          <Panel
            class="model-signals-table-panel"
            :title="t('anomaly.title')"
            :kicker="t('anomaly.kicker')"
            :icon="WarningOutlined"
          >
            <a-table
              class="dense-table model-signals-anomaly-table"
              :columns="anomalyColumns"
              :data-source="normalizedAnomalies"
              :loading="loading"
              :locale="anomalyTableLocale"
              :pagination="{ pageSize: 8 }"
              :row-key="anomalyRowKey"
              size="small"
              :custom-row="anomalyRowClass"
              :scroll="{ x: 1600 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'session'">
                  <span class="mono model-signals-session" :title="sessionInfo(record).full">{{ sessionInfo(record).main }}</span>
                  <div class="source-identity-meta">{{ formatNumber(record.totalTokens) }} {{ t('column.tokens') }}</div>
                </template>
                <template v-else-if="column.key === 'source'">
                  <SourceCell :info="sourceInfo(record)" />
                </template>
                <template v-else-if="column.key === 'project'">
                  <ProjectCell :info="projectInfo(record)" />
                </template>
                <template v-else-if="column.key === 'model'">
                  <ModelCell :model="record.model" :fallback="t('fallback.unknown')" />
                </template>
                <template v-else-if="column.key === 'signal'">
                  <ReasonTags
                    :reasons="record.reasons"
                    :color="record.failedToolCalls > 0 || record.score >= 0.45 ? 'warning' : 'processing'"
                    :empty-text="t('fallback.noReason')"
                  />
                </template>
                <template v-else-if="column.key === 'outputExpansion'"><span class="number-cell">{{ formatPercent(record.outputExpansionRate) }}</span></template>
                <template v-else-if="column.key === 'reasoning'"><span class="number-cell">{{ formatPercent(record.reasoningOverheadRate) }}</span></template>
                <template v-else-if="column.key === 'cacheMiss'"><span class="number-cell">{{ formatPercent(record.cacheMissRate) }}</span></template>
                <template v-else-if="column.key === 'failedTools'">
                  <span class="number-cell" :class="{ 'status-error': record.failedToolCalls > 0 }">{{ formatNumber(record.failedToolCalls) }}</span>
                </template>
                <template v-else-if="column.key === 'throughput'"><span class="number-cell">{{ formatRate(record.modelThroughputTokensPerSecond, 1) }}</span></template>
                <template v-else-if="column.key === 'started'">{{ formatDateTime(record.startedAt) }}</template>
                <template v-else-if="column.key === 'duration'"><span class="number-cell">{{ formatDuration(record.modelDurationMs) }}</span></template>
                <template v-else-if="column.key === 'open'">
                  <a-tooltip :title="t('action.openSession')">
                    <a-button type="text" size="small" :disabled="!record.id" @click="openSession(record.id)">
                      <template #icon>
                        <ArrowRightOutlined />
                      </template>
                    </a-button>
                  </a-tooltip>
                </template>
              </template>
            </a-table>
          </Panel>
        </div>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.model-signals-page {
  max-width: 1560px;
}

.model-signals-error {
  margin-bottom: var(--am-section-gap);
}

.model-signals-table-panel :deep(.panel-body) {
  padding: 0;
}

.model-signals-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  padding: 8px;
  background: var(--am-surface);
  border: 1px solid var(--am-border);
  border-radius: var(--am-radius);
  box-shadow: var(--am-shadow);
}

.model-signals-tabs .ant-btn {
  min-width: 126px;
  justify-content: center;
}

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

.model-signals-session,
.model-signals-date {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: var(--am-text);
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.model-signals-date {
  color: var(--am-text);
  font-variant-numeric: tabular-nums;
}

.model-signals-confidence-cell,
.model-signals-health-cell,
.model-signals-mix-cell {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.model-signals-confidence-cell {
  grid-template-columns: auto auto minmax(0, 1fr);
  align-items: center;
}

.model-signals-health-cell {
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
}

.model-signals-mix-cell .model-name,
.model-signals-reason-text {
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

.model-signals-matrix-cells {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.model-signals-matrix-cell {
  width: 244px;
  min-width: 0;
  padding: 8px 9px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-left: 3px solid var(--am-success);
  border-radius: var(--am-radius-sm);
}

.model-signals-matrix-cell.severity-watch {
  border-left-color: var(--am-info);
}

.model-signals-matrix-cell.severity-warning {
  border-left-color: var(--am-warning);
  background: var(--am-warning-soft);
}

.model-signals-matrix-cell.severity-critical {
  border-left-color: var(--am-danger);
  background: var(--am-danger-soft);
}

.model-signals-matrix-cell-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}

.model-signals-matrix-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  margin-top: 6px;
  color: var(--am-text-soft);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  line-height: 15px;
}

.model-signals-matrix-metrics span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-matrix-reason {
  margin-top: 4px;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 11px;
  line-height: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.model-signals-warning-row td) {
  background: rgba(254, 243, 199, 0.36);
}

:deep(.model-signals-critical-row td) {
  background: rgba(254, 226, 226, 0.46);
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

  .model-signals-tabs .ant-btn {
    min-width: 112px;
  }

  .model-signals-reason-strip {
    align-items: flex-start;
    flex-direction: column;
  }
}

@media (max-width: 640px) {
  .model-signals-tabs {
    flex-wrap: nowrap;
    overflow-x: auto;
  }

  .model-signals-tabs .ant-btn {
    flex: 0 0 auto;
  }

  .model-signals-metric-strip {
    grid-template-columns: 1fr;
  }
}
</style>
