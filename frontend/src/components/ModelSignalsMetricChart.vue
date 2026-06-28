<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASegmented from 'ant-design-vue/es/segmented'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import { BarChartOutlined, LineChartOutlined, UndoOutlined } from '@ant-design/icons-vue'
import type { ModelSignalsDailyMetric } from '../api'
import { useEChart } from '../composables/useEChart'
import { useMessages } from '../i18n'
import { modelSignalsMetricChartMessages } from './model-signals/chartMessages'
import {
  buildDailyChartOption,
  buildProjectChartOption
} from './model-signals/chartOptions'
import {
  buildMetricDefinitions,
  buildMetricGroups,
  defaultMetricsForMode,
  hasBaselineComparison,
  hasChartData,
  metricKindsFor,
  plottedRowsForMode,
  resolveSelectedMetrics,
  shouldNormalizeProjectScale,
  type ChartMode,
  type MetricGroupKey,
  type MetricKey,
  type ProjectChartRow
} from './model-signals/chartMetrics'

const props = withDefaults(
  defineProps<{
    dailyRows?: ModelSignalsDailyMetric[]
    projectRows?: ProjectChartRow[]
    loading?: boolean
    initialMode?: ChartMode
    allowModeSwitch?: boolean
  }>(),
  {
    dailyRows: () => [],
    projectRows: () => [],
    loading: false,
    initialMode: 'daily',
    allowModeSwitch: true
  }
)

const selectedMode = ref<ChartMode>(props.initialMode)
const selectedMetricKeys = ref<MetricKey[]>(defaultMetricsForMode(props.initialMode))
const showBaselineComparison = ref(false)
const { chartEl, getChart, disposeChart } = useEChart()
const { t, locale } = useMessages(modelSignalsMetricChartMessages)

const metricGroups = computed(() => buildMetricGroups(t))
const metricDefinitions = computed(() => buildMetricDefinitions(t))
const selectedMetrics = computed(() => resolveSelectedMetrics(selectedMetricKeys.value, metricDefinitions.value))
const primaryMetric = computed(() => {
  const metric = selectedMetrics.value[0] || metricDefinitions.value[0]
  if (!metric) throw new Error('No model signal metrics configured')
  return metric
})
const modeOptions = computed(() => [
  { label: t('mode.daily'), value: 'daily' },
  { label: t('mode.projects'), value: 'projects' }
])
const chartTitle = computed(() => selectedMode.value === 'projects' ? t('title.projects') : t('title.daily'))
const chartKicker = computed(() => selectedMode.value === 'projects' ? t('kicker.projects') : t('kicker.daily'))
const directionLabel = computed(() => {
  const directions = new Set(selectedMetrics.value.map((metric) => metric.direction))
  if (directions.size === 1) {
    const [direction] = [...directions]
    return t(`direction.${direction}`)
  }
  return t('direction.mixed')
})
const selectionCountLabel = computed(() => t('selection.count', { count: selectedMetrics.value.length }))
const selectedMetricLabel = computed(() => t('selection.metric', { metric: primaryMetric.value.label }))
const plottedRows = computed(() =>
  plottedRowsForMode(selectedMode.value, props.dailyRows, props.projectRows, primaryMetric.value)
)
const hasChart = computed(() => hasChartData(plottedRows.value, selectedMetrics.value, selectedMode.value))
const canCompareBaseline = computed(() =>
  hasBaselineComparison(plottedRows.value, selectedMetrics.value, selectedMode.value)
)
const activeMetricKinds = computed(() => metricKindsFor(selectedMetrics.value))
const normalizeProjectScale = computed(() =>
  shouldNormalizeProjectScale(selectedMode.value, activeMetricKinds.value)
)

watch(() => props.initialMode, (mode) => {
  selectedMode.value = mode
  if (!selectedMetricKeys.value.length) selectedMetricKeys.value = defaultMetricsForMode(mode)
})

watch(selectedMode, (mode, previous) => {
  if (mode !== previous) {
    selectedMetricKeys.value = defaultMetricsForMode(mode)
  }
})

watch(canCompareBaseline, (available) => {
  if (!available) showBaselineComparison.value = false
})

watch(() => [selectedMode.value, selectedMetricKeys.value, showBaselineComparison.value, props.dailyRows, props.projectRows, locale.value], renderAfterUpdate, { deep: true })

onMounted(() => {
  renderAfterUpdate()
})

async function renderAfterUpdate() {
  await nextTick()
  renderChart()
}

function renderChart() {
  if (!hasChart.value) {
    disposeChart()
    return
  }
  if (selectedMode.value === 'projects') {
    renderProjectChart()
  } else {
    renderDailyChart()
  }
}

function renderDailyChart() {
  const chart = getChart()
  if (!chart) return

  chart.setOption(buildDailyChartOption({
    rows: plottedRows.value as ModelSignalsDailyMetric[],
    selectedMetrics: selectedMetrics.value,
    primaryMetric: primaryMetric.value,
    activeMetricKinds: activeMetricKinds.value,
    showBaselineComparison: showBaselineComparison.value,
    t
  }), true)
}

function renderProjectChart() {
  const chart = getChart()
  if (!chart) return

  chart.setOption(buildProjectChartOption({
    rows: plottedRows.value as ProjectChartRow[],
    selectedMetrics: selectedMetrics.value,
    primaryMetric: primaryMetric.value,
    activeMetricKinds: activeMetricKinds.value,
    normalizeProjectScale: normalizeProjectScale.value,
    showBaselineComparison: showBaselineComparison.value,
    t
  }), true)
}

function selectMetric(key: MetricKey) {
  selectedMetricKeys.value = [key]
}

function resetMetricSelection() {
  selectedMetricKeys.value = defaultMetricsForMode(selectedMode.value)
  showBaselineComparison.value = false
}

function isMetricSelected(key: MetricKey) {
  return selectedMetricKeys.value.includes(key)
}

function metricsForGroup(group: MetricGroupKey) {
  return metricDefinitions.value.filter((metric) => metric.group === group)
}
</script>

<template>
  <section class="panel model-signals-chart-panel">
    <div class="panel-header model-signals-chart-header">
      <div>
        <h2 class="panel-title">{{ chartTitle }}</h2>
        <div class="panel-kicker">{{ chartKicker }}</div>
      </div>
      <div class="model-signals-chart-actions">
        <a-tag class="status-tag model-signals-chart-count" color="processing">
          {{ selectedMetricLabel || selectionCountLabel }}
        </a-tag>
        <a-tooltip :title="selectedMetrics.map((metric) => metric.description).join('\n')">
          <a-tag class="status-tag model-signals-chart-direction" :color="directionLabel === t('direction.higher') ? 'success' : directionLabel === t('direction.context') ? 'default' : 'warning'">
            {{ directionLabel }}
          </a-tag>
        </a-tooltip>
        <LineChartOutlined v-if="selectedMode === 'daily'" class="panel-header-icon" />
        <BarChartOutlined v-else class="panel-header-icon" />
      </div>
    </div>

    <div class="model-signals-chart-toolbar" :aria-label="t('control.metrics')">
      <a-segmented
        v-if="allowModeSwitch"
        v-model:value="selectedMode"
        class="model-signals-chart-segmented"
        :options="modeOptions"
        :aria-label="t('control.mode')"
      />
      <a-tooltip :title="canCompareBaseline ? '' : t('control.baselineUnavailable')">
        <label class="model-signals-baseline-toggle" :class="{ 'is-disabled': !canCompareBaseline }">
          <input v-model="showBaselineComparison" type="checkbox" :disabled="!canCompareBaseline">
          <span>{{ t('control.baseline') }}</span>
        </label>
      </a-tooltip>
      <a-button class="model-signals-reset-button" @click="resetMetricSelection">
        <template #icon>
          <UndoOutlined />
        </template>
        {{ t('action.reset') }}
      </a-button>
    </div>

    <div class="model-signals-metric-picker" role="group" :aria-label="t('control.metrics')">
      <div v-for="group in metricGroups" :key="group.key" class="model-signals-metric-group">
        <div class="model-signals-metric-group-label">{{ group.label }}</div>
        <div class="model-signals-metric-choices">
          <a-tooltip v-for="item in metricsForGroup(group.key)" :key="item.key" :title="item.description">
            <label
              class="model-signals-metric-choice"
              :class="{ 'is-active': isMetricSelected(item.key) }"
              :style="{ '--signal-color': item.color }"
            >
              <input
                type="radio"
                :checked="isMetricSelected(item.key)"
                @change="selectMetric(item.key)"
              >
              <span class="model-signals-metric-swatch"></span>
              <span class="model-signals-metric-label">{{ item.label }}</span>
            </label>
          </a-tooltip>
        </div>
      </div>
    </div>

    <a-spin :spinning="loading">
      <div class="panel-body">
        <div v-if="hasChart" ref="chartEl" class="chart model-signals-metric-chart"></div>
        <div v-else class="empty-state model-signals-metric-empty">
          <LineChartOutlined class="empty-state-icon" />
          <div class="empty-state-title">{{ t('empty.title') }}</div>
          <div class="empty-state-text">{{ t('empty.text') }}</div>
        </div>
      </div>
    </a-spin>
  </section>
</template>

<style scoped>
.model-signals-chart-header {
  align-items: flex-start;
  gap: 16px;
}

.model-signals-chart-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex-wrap: wrap;
  flex-shrink: 0;
  justify-content: flex-end;
}

.model-signals-chart-count,
.model-signals-chart-direction {
  margin-right: 0;
  white-space: nowrap;
}

.model-signals-chart-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-width: 0;
  margin: 12px 0;
  padding: 0 14px;
}

.model-signals-chart-segmented {
  flex-shrink: 0;
  max-width: 100%;
  overflow-x: auto;
}

.model-signals-baseline-toggle {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 32px;
  padding: 0 10px;
  color: var(--am-text);
  font-size: 12px;
  font-weight: 650;
  background: var(--am-surface);
  border: 1px solid var(--am-border);
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
}

.model-signals-baseline-toggle input {
  width: 14px;
  height: 14px;
  margin: 0;
  accent-color: var(--am-primary);
}

.model-signals-baseline-toggle.is-disabled {
  color: var(--am-muted);
  background: var(--am-surface-subtle);
  cursor: not-allowed;
}

.model-signals-reset-button {
  flex-shrink: 0;
}

.model-signals-metric-picker {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 14px;
  padding: 0 14px;
}

.model-signals-metric-group {
  min-width: 0;
  padding: 10px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.model-signals-metric-group-label {
  margin-bottom: 8px;
  color: var(--am-muted);
  font-size: 11px;
  font-weight: 750;
  text-transform: uppercase;
}

.model-signals-metric-choices {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.model-signals-metric-choice {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-height: 28px;
  padding: 0 8px;
  color: var(--am-text-soft);
  font-size: 12px;
  font-weight: 600;
  background: var(--am-surface);
  border: 1px solid var(--am-border);
  border-radius: 6px;
  cursor: pointer;
  user-select: none;
}

.model-signals-metric-choice input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.model-signals-metric-choice.is-active {
  color: var(--am-text);
  border-color: color-mix(in srgb, var(--signal-color) 46%, var(--am-border));
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--signal-color) 22%, transparent);
}

.model-signals-metric-choice.is-disabled {
  cursor: default;
}

.model-signals-metric-swatch {
  width: 8px;
  height: 8px;
  flex: 0 0 auto;
  background: var(--signal-color);
  border-radius: 50%;
}

.model-signals-metric-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-metric-chart {
  height: 380px;
}

.model-signals-metric-empty {
  min-height: 260px;
}

@media (max-width: 1180px) {
  .model-signals-metric-picker {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .model-signals-chart-header,
  .model-signals-chart-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .model-signals-chart-actions {
    justify-content: flex-start;
  }

  .model-signals-metric-picker {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 720px) {
  .model-signals-metric-chart {
    height: 320px;
  }
}
</style>
