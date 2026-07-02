<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import {
  api,
  type ModelSignals
} from '../api'
import ModelSignalsMetricChart from '../components/ModelSignalsMetricChart.vue'
import PageHeader from '../components/PageHeader.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import {
  buildUsageAgentOptions,
  buildUsageModelOptions,
  buildUsageProjectOptions,
  useUsageScopeOptionData
} from './useUsageScopeOptions'
import { createModelSignalsDisplay } from './model-signals/display'
import ModelSignalsAnomaliesTable from './model-signals/ModelSignalsAnomaliesTable.vue'
import ModelSignalsCohortTable from './model-signals/ModelSignalsCohortTable.vue'
import ModelSignalsDailySection from './model-signals/ModelSignalsDailySection.vue'
import ModelSignalsMatrixSection from './model-signals/ModelSignalsMatrixSection.vue'
import ModelSignalsOverviewSection from './model-signals/ModelSignalsOverviewSection.vue'
import ModelSignalsProjectsSection from './model-signals/ModelSignalsProjectsSection.vue'
import { modelSignalsMessages } from './model-signals/messages'
import { buildModelSignalsTabs, type ModelSignalsTabKey } from './model-signals/tabs'
import type { ProjectMetricRow } from './model-signals/types'

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
  fallbackHealthSummary,
  normalizeAnomaly
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

const tabs = computed(() => buildModelSignalsTabs(t))

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

        <ModelSignalsOverviewSection
          v-else-if="activeTab === 'overview'"
          :signals="signals"
          :health-summary="healthSummary"
          :rows="cohortRows"
          :has-data="hasData"
          :loading="loading"
        />

        <ModelSignalsDailySection
          v-else-if="activeTab === 'daily'"
          :rows="dailyMetricRows"
          :project-rows="projectRows"
          :loading="loading"
        />

        <ModelSignalsCohortTable
          v-else-if="activeTab === 'cohorts'"
          variant="cohorts"
          :rows="cohortRows"
          :loading="loading"
        />

        <ModelSignalsMatrixSection
          v-else-if="activeTab === 'matrix'"
          :rows="matrixRows"
          :loading="loading"
        />

        <ModelSignalsProjectsSection
          v-else-if="activeTab === 'projects'"
          :daily-rows="dailyMetricRows"
          :rows="projectRows"
          :has-project-metrics="hasProjectMetrics"
          :loading="loading"
        />

        <ModelSignalsAnomaliesTable
          v-else-if="activeTab === 'anomalies'"
          :rows="normalizedAnomalies"
          :loading="loading"
          @open-session="openSession"
        />
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

@media (max-width: 980px) {
  .model-signals-tabs .ant-btn {
    min-width: 112px;
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
}
</style>
