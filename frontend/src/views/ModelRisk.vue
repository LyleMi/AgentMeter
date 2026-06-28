<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch, type DefineComponent } from 'vue'
import AAlert from 'ant-design-vue/es/alert'
import ASelect from 'ant-design-vue/es/select'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import {
  BranchesOutlined,
  DashboardOutlined,
  ExperimentOutlined,
  SafetyCertificateOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatDisplayNumber,
  formatNumber,
  type ModelSignals
} from '../api'
import { chartPalette } from '../chartPalette'
import PageHeader from '../components/PageHeader.vue'
import UsageScopeBar from '../components/UsageScopeBar.vue'
import Panel from '../components/ui/Panel.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useEChart } from '../composables/useEChart'
import { useMessages } from '../i18n'
import {
  buildUsageAgentOptions,
  buildUsageModelOptions,
  buildUsageProjectOptions,
  useUsageScopeOptionData
} from './useUsageScopeOptions'
import { useUsageScopeRoute, type UsageScopeForm } from './useUsageScope'
import ModelCell from './model-signals/ModelCell.vue'
import SourceCell from './model-signals/SourceCell.vue'
import {
  buildQualityRiskRows,
  formatQualityRiskScore,
  qualityRiskMessages,
  qualityRiskRowSummary,
  type QualityRiskLevel,
  type QualityRiskRow
} from './model-signals/risk'

const ATable = AntTable as unknown as DefineComponent

const pageMessages = {
  en: {
    ...qualityRiskMessages.en,
    'title': 'Model Quality Risk',
    'subtitle': 'Explain quality-degradation symptoms across sources and models without treating the score as proof of routing or substitution',
    'notice.title': 'What this page means',
    'notice.text': 'The score combines latency, throughput, failures, retry pressure, cache behavior, and token-shape symptoms. It can flag suspicious degradation, including possible relay, throttling, or weaker-model behavior, but it cannot prove why the service changed.',
    'metric.maxRisk': 'Highest risk',
    'metric.maxRiskNote': '{source} / {model}',
    'metric.elevated': 'Elevated rows',
    'metric.elevatedNote': '{high} high, {elevated} elevated',
    'metric.cells': 'Compared cells',
    'metric.cellsNote': 'Source-model pairs in the current scope',
    'metric.lowConfidence': 'Low confidence',
    'metric.lowConfidenceNote': '{count} rows need more baseline/current samples',
    'formula.title': 'Score drivers',
    'formula.kicker': 'The risk score is a weighted symptom model; every row below shows the strongest visible drivers',
    'formula.copy': 'A high score means the observed behavior looks degraded across multiple signals. The explanation should be read as evidence for review, not as attribution to a relay, provider-side throttling, or model substitution.',
    'chart.title': 'Highest Risk Source-Model Pairs',
    'chart.kicker': 'Sorted by composite quality-risk score',
    'source.title': 'Models in Selected Source',
    'source.kicker': 'Compare risk inside one source without mixing it with latency or cost axes',
    'table.title': 'Risk Explanations',
    'table.kicker': 'Each score includes the reason, confidence, sample note, and top driver contributions',
    'control.source': 'Source',
    'column.source': 'Source',
    'column.model': 'Model',
    'column.risk': 'Risk',
    'column.explanation': 'Explanation',
    'column.confidence': 'Confidence',
    'column.samples': 'Samples',
    'label.cohorts': 'cohorts',
    'driver.weight': '{weight} weight',
    'driver.contribution': '{value} contribution',
    'empty.loading': 'Loading model quality risk...',
    'empty.risk': 'No model quality risk rows match the current scope',
    'fallback.unknown': 'unknown',
    'fallback.noReason': 'No risk reason',
    'error.title': 'Model quality risk failed to load'
  },
  'zh-CN': {
    ...qualityRiskMessages['zh-CN'],
    'title': '模型质量风险',
    'subtitle': '按来源和模型解释质量退化信号，但不把分数当成中转、替换或掺水证明',
    'notice.title': '这个页面如何解读',
    'notice.text': '分数综合延迟、吞吐、失败、重试压力、缓存行为和 token 形态等症状。它能提示可疑退化，包括可能的中转、限速或弱模型表现，但不能证明服务变化的原因。',
    'metric.maxRisk': '最高风险',
    'metric.maxRiskNote': '{source} / {model}',
    'metric.elevated': '升高行数',
    'metric.elevatedNote': '{high} 个高风险，{elevated} 个升高',
    'metric.cells': '对比单元',
    'metric.cellsNote': '当前范围内的来源-模型组合',
    'metric.lowConfidence': '低置信',
    'metric.lowConfidenceNote': '{count} 行需要更多基线/当前样本',
    'formula.title': '分数驱动因子',
    'formula.kicker': '风险分数是加权症状模型；下方每一行都会显示最强的可见驱动因子',
    'formula.copy': '高分代表观测行为在多个信号上看起来退化。解释应作为复核证据，而不是直接归因于中转、官方限速或模型替换。',
    'chart.title': '最高风险来源-模型组合',
    'chart.kicker': '按综合模型质量风险分数排序',
    'source.title': '选中来源内的模型',
    'source.kicker': '在同一个来源内比较风险，不和延迟或费用坐标轴混在一起',
    'table.title': '风险解释',
    'table.kicker': '每个分数都包含原因、置信度、样本说明和主要驱动贡献',
    'control.source': '来源',
    'column.source': '来源',
    'column.model': '模型',
    'column.risk': '风险',
    'column.explanation': '解释',
    'column.confidence': '置信度',
    'column.samples': '样本',
    'label.cohorts': '分组',
    'driver.weight': '{weight} 权重',
    'driver.contribution': '{value} 贡献',
    'empty.loading': '正在加载模型质量风险...',
    'empty.risk': '当前范围没有模型质量风险行',
    'fallback.unknown': '未知',
    'fallback.noReason': '无风险原因',
    'error.title': '模型质量风险加载失败'
  }
} as const

const resource = useAsyncResource<ModelSignals | null>(null)
const signals = computed(() => resource.data.value)
const loading = resource.loading
const error = resource.error
const scope = useUsageScopeRoute(() => {
  void load()
})
const scopeOptionData = useUsageScopeOptionData()
const { chartEl, getChart, disposeChart } = useEChart()
const { t, locale } = useMessages(pageMessages)
const selectedSourceKey = ref('')

const matrixRows = computed(() => signals.value?.matrix || [])
const matrixCells = computed(() => matrixRows.value.flatMap((row) => row.cells || []))
const riskRows = computed(() => buildQualityRiskRows(matrixRows.value, t, t('fallback.unknown')))
const topRiskRows = computed(() => riskRows.value.slice(0, 10))
const sourceRows = computed(() => riskRows.value.filter((row) => row.sourceKey === selectedSourceKey.value))
const highRows = computed(() => riskRows.value.filter((row) => row.level === 'high'))
const elevatedRows = computed(() => riskRows.value.filter((row) => row.level === 'elevated'))
const lowConfidenceRows = computed(() => riskRows.value.filter((row) => row.drift?.confidence === 'low'))
const hasRows = computed(() => riskRows.value.length > 0)
const tableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.risk') }))
const riskColumns = computed(() => [
  { title: t('column.source'), key: 'source', width: 220 },
  { title: t('column.model'), key: 'model', width: 190 },
  { title: t('column.risk'), key: 'risk', width: 150, align: 'right' },
  { title: t('column.explanation'), key: 'explanation', width: 420 },
  { title: t('column.confidence'), key: 'confidence', width: 122 },
  { title: t('column.samples'), key: 'samples', width: 220 }
])
const sourceOptions = computed(() => {
  const values = new Map<string, { value: string; label: string }>()
  riskRows.value.forEach((row) => {
    if (values.has(row.sourceKey)) return
    values.set(row.sourceKey, {
      value: row.sourceKey,
      label: row.sourceSecondary ? `${row.sourceLabel} · ${row.sourceSecondary}` : row.sourceLabel
    })
  })
  return [...values.values()].sort((left, right) => left.label.localeCompare(right.label))
})

const agentOptions = computed(() =>
  buildUsageAgentOptions({
    sources: [
      matrixRows.value,
      scopeOptionData.optionOverview.value?.agentUsage,
      scopeOptionData.optionOverview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.agent,
    fallback: t('fallback.unknown')
  })
)

const modelOptions = computed(() =>
  buildUsageModelOptions({
    modelUsage: [
      matrixCells.value,
      signals.value?.modelBreakdown,
      scopeOptionData.optionOverview.value?.modelUsage
    ],
    sessions: [
      scopeOptionData.optionOverview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.model
  })
)

const projectOptions = computed(() =>
  buildUsageProjectOptions({
    projects: [
      signals.value?.projectMetrics,
      signals.value?.projectHotspots,
      scopeOptionData.projectOptionRows.value,
      scopeOptionData.optionOverview.value?.recentSessions,
      scopeOptionData.optionOverview.value?.slowSessions
    ],
    selected: scope.filters.value.project,
    fallback: t('fallback.unknown')
  })
)

const metricCards = computed(() => {
  const top = riskRows.value[0]
  return [
    {
      label: t('metric.maxRisk'),
      value: formatQualityRiskScore(top?.score),
      note: top ? t('metric.maxRiskNote', { source: top.sourceLabel, model: top.model }) : t('fallback.unknown'),
      icon: WarningOutlined,
      tone: riskMetricTone(top?.level)
    },
    {
      label: t('metric.elevated'),
      value: formatDisplayNumber(highRows.value.length + elevatedRows.value.length).main,
      note: t('metric.elevatedNote', { high: highRows.value.length, elevated: elevatedRows.value.length }),
      icon: SafetyCertificateOutlined,
      tone: highRows.value.length ? 'metric-danger' : elevatedRows.value.length ? 'metric-warning' : 'metric-success'
    },
    {
      label: t('metric.cells'),
      value: formatDisplayNumber(riskRows.value.length).main,
      note: t('metric.cellsNote'),
      icon: BranchesOutlined,
      tone: 'metric-primary'
    },
    {
      label: t('metric.lowConfidence'),
      value: formatDisplayNumber(lowConfidenceRows.value.length).main,
      note: t('metric.lowConfidenceNote', { count: lowConfidenceRows.value.length }),
      icon: ExperimentOutlined,
      tone: lowConfidenceRows.value.length ? 'metric-warning' : 'metric-neutral'
    }
  ]
})

onMounted(load)

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

watch(sourceOptions, (options) => {
  if (!options.length) {
    selectedSourceKey.value = ''
    return
  }
  if (!options.some((option) => option.value === selectedSourceKey.value)) {
    selectedSourceKey.value = options[0].value
  }
}, { immediate: true })

watch(() => [topRiskRows.value, locale.value], renderRiskChart, { deep: true })

async function renderRiskChart() {
  await nextTick()
  if (!topRiskRows.value.length) {
    disposeChart()
    return
  }
  const chart = getChart()
  if (!chart) return
  chart.setOption({
    tooltip: {
      trigger: 'axis',
      backgroundColor: chartPalette.tooltipBg,
      borderWidth: 0,
      textStyle: { color: chartPalette.tooltipText, fontSize: 12 },
      axisPointer: { type: 'shadow', shadowStyle: { color: chartPalette.pointer } },
      formatter: (params: unknown) => riskTooltipMarkup(params)
    },
    grid: { left: 158, right: 36, top: 16, bottom: 34 },
    xAxis: {
      type: 'value',
      max: 1,
      axisLabel: { color: chartPalette.axis, fontSize: 11, formatter: (value: number) => formatQualityRiskScore(value) },
      splitLine: { lineStyle: { color: chartPalette.grid } }
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: topRiskRows.value.map((row) => `${row.sourceLabel} / ${row.model}`),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: chartPalette.border } },
      axisLabel: { color: chartPalette.text, fontSize: 11, overflow: 'truncate', width: 138 }
    },
    series: [{
      name: t('column.risk'),
      type: 'bar',
      barMaxWidth: 14,
      itemStyle: {
        color: (input: { dataIndex: number }) => riskColor(topRiskRows.value[input.dataIndex]?.level),
        borderRadius: [0, 3, 3, 0]
      },
      data: topRiskRows.value.map((row) => row.score)
    }]
  }, true)
}

function riskTooltipMarkup(params: unknown) {
  const items = Array.isArray(params) ? params : [params]
  const first = items[0] as { dataIndex?: number } | undefined
  const row = topRiskRows.value[first?.dataIndex ?? 0]
  if (!row) return ''
  return [
    `<strong>${escapeHtml(row.sourceLabel)} / ${escapeHtml(row.model)}</strong>`,
    `<div>${escapeHtml(t('column.risk'))}: ${escapeHtml(formatQualityRiskScore(row.score))}</div>`,
    `<div>${escapeHtml(row.primaryReason)}</div>`,
    ...row.drivers.map((driver) => `<div>${escapeHtml(driver.label)}: ${escapeHtml(driver.formattedValue)}</div>`)
  ].join('')
}

function levelLabel(level: QualityRiskLevel) {
  return t(`risk.level.${level}`)
}

function riskColor(level?: QualityRiskLevel) {
  if (level === 'high') return chartPalette.danger
  if (level === 'elevated') return chartPalette.warning
  if (level === 'watch') return chartPalette.info
  return chartPalette.success
}

function riskTagColor(level?: QualityRiskLevel) {
  if (level === 'high') return 'error'
  if (level === 'elevated') return 'warning'
  if (level === 'watch') return 'processing'
  return 'success'
}

function riskMetricTone(level?: QualityRiskLevel) {
  if (level === 'high') return 'metric-danger'
  if (level === 'elevated') return 'metric-warning'
  if (level === 'watch') return 'metric-info'
  return 'metric-success'
}

function sourceInfo(row: QualityRiskRow) {
  return { label: row.sourceLabel, secondary: row.sourceSecondary }
}

function driverContribution(value: number) {
  return t('driver.contribution', { value: formatQualityRiskScore(value) })
}

function driverWeight(value: number) {
  return t('driver.weight', { weight: formatQualityRiskScore(value) })
}

function escapeHtml(value: string | number | undefined) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}
</script>

<template>
  <div class="page model-risk-page">
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
      class="model-risk-notice"
      type="info"
      show-icon
      :message="t('notice.title')"
      :description="t('notice.text')"
    />

    <a-alert
      v-if="error"
      class="model-risk-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <a-spin :spinning="loading && !signals">
      <div class="section-stack">
        <section class="metric-strip model-risk-metric-strip" :class="{ 'is-empty': !hasRows }">
          <div v-for="item in metricCards" :key="item.label" class="metric-strip-item" :class="item.tone">
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

        <Panel :title="t('formula.title')" :kicker="t('formula.kicker')" :icon="DashboardOutlined">
          <div class="model-risk-formula">
            <p>{{ t('formula.copy') }}</p>
            <div class="model-risk-driver-list">
              <div v-for="driver in riskRows[0]?.drivers || []" :key="driver.key" class="model-risk-driver-card">
                <div class="model-risk-driver-head">
                  <span>{{ driver.label }}</span>
                  <a-tag class="status-tag" :color="riskTagColor(driver.severity)">{{ levelLabel(driver.severity) }}</a-tag>
                </div>
                <div class="model-risk-driver-value">{{ driver.formattedValue }}</div>
                <div class="source-identity-meta">{{ driver.explanation }}</div>
                <div class="source-identity-meta">{{ driverContribution(driver.contribution) }} · {{ driverWeight(driver.weight) }}</div>
              </div>
            </div>
          </div>
        </Panel>

        <Panel :title="t('chart.title')" :kicker="t('chart.kicker')" :icon="WarningOutlined">
          <div v-if="topRiskRows.length" ref="chartEl" class="chart model-risk-chart"></div>
          <div v-else class="empty-state model-risk-empty">
            <WarningOutlined class="empty-state-icon" />
            <div class="empty-state-title">{{ t('empty.risk') }}</div>
          </div>
        </Panel>

        <Panel :title="t('source.title')" :kicker="t('source.kicker')" :icon="BranchesOutlined">
          <template #actions>
            <a-select
              v-model:value="selectedSourceKey"
              class="model-risk-source-select"
              :options="sourceOptions"
              :placeholder="t('control.source')"
              show-search
              option-filter-prop="label"
            />
          </template>
          <a-table
            class="dense-table model-risk-table"
            :columns="riskColumns"
            :data-source="sourceRows"
            :loading="loading"
            :locale="tableLocale"
            :pagination="{ pageSize: 6 }"
            row-key="key"
            size="small"
            :scroll="{ x: 1320 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'source'">
                <SourceCell :info="sourceInfo(record)" />
              </template>
              <template v-else-if="column.key === 'model'">
                <ModelCell :model="record.model" :provider="record.modelProvider" :fallback="t('fallback.unknown')" show-provider />
              </template>
              <template v-else-if="column.key === 'risk'">
                <div class="model-risk-score-cell">
                  <span class="number-cell">{{ formatQualityRiskScore(record.score) }}</span>
                  <a-tag class="status-tag" :color="riskTagColor(record.level)">{{ levelLabel(record.level) }}</a-tag>
                </div>
              </template>
              <template v-else-if="column.key === 'explanation'">
                <div class="model-risk-explanation-cell">
                  <strong>{{ record.primaryReason || t('fallback.noReason') }}</strong>
                  <div class="model-risk-driver-tags">
                    <a-tooltip v-for="driver in record.drivers" :key="driver.key" :title="`${driver.explanation}\n${driverContribution(driver.contribution)} / ${driverWeight(driver.weight)}`">
                      <a-tag class="status-tag" :color="riskTagColor(driver.severity)">
                        {{ driver.label }} · {{ driver.formattedValue }}
                      </a-tag>
                    </a-tooltip>
                  </div>
                </div>
              </template>
              <template v-else-if="column.key === 'confidence'">
                <div class="model-risk-confidence-cell">
                  <span>{{ record.drift?.confidence || t('fallback.unknown') }}</span>
                  <span class="source-identity-meta">{{ record.sampleNote }}</span>
                </div>
              </template>
              <template v-else-if="column.key === 'samples'">
                <div class="model-risk-sample-cell">
                  <span>{{ qualityRiskRowSummary(record, t) }}</span>
                  <span class="source-identity-meta">{{ formatNumber(record.cell.cohortCount) }} {{ t('label.cohorts') }}</span>
                </div>
              </template>
            </template>
          </a-table>
        </Panel>

        <Panel :title="t('table.title')" :kicker="t('table.kicker')" :icon="SafetyCertificateOutlined">
          <a-table
            class="dense-table model-risk-table"
            :columns="riskColumns"
            :data-source="riskRows"
            :loading="loading"
            :locale="tableLocale"
            :pagination="{ pageSize: 10 }"
            row-key="key"
            size="small"
            :scroll="{ x: 1320 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'source'">
                <SourceCell :info="sourceInfo(record)" />
              </template>
              <template v-else-if="column.key === 'model'">
                <ModelCell :model="record.model" :provider="record.modelProvider" :fallback="t('fallback.unknown')" show-provider />
              </template>
              <template v-else-if="column.key === 'risk'">
                <div class="model-risk-score-cell">
                  <span class="number-cell">{{ formatQualityRiskScore(record.score) }}</span>
                  <a-tag class="status-tag" :color="riskTagColor(record.level)">{{ levelLabel(record.level) }}</a-tag>
                </div>
              </template>
              <template v-else-if="column.key === 'explanation'">
                <div class="model-risk-explanation-cell">
                  <strong>{{ record.primaryReason || t('fallback.noReason') }}</strong>
                  <div class="model-risk-driver-tags">
                    <a-tooltip v-for="driver in record.drivers" :key="driver.key" :title="`${driver.explanation}\n${driverContribution(driver.contribution)} / ${driverWeight(driver.weight)}`">
                      <a-tag class="status-tag" :color="riskTagColor(driver.severity)">
                        {{ driver.label }} · {{ driver.formattedValue }}
                      </a-tag>
                    </a-tooltip>
                  </div>
                </div>
              </template>
              <template v-else-if="column.key === 'confidence'">
                <div class="model-risk-confidence-cell">
                  <span>{{ record.drift?.confidence || t('fallback.unknown') }}</span>
                  <span class="source-identity-meta">{{ record.sampleNote }}</span>
                </div>
              </template>
              <template v-else-if="column.key === 'samples'">
                <div class="model-risk-sample-cell">
                  <span>{{ qualityRiskRowSummary(record, t) }}</span>
                  <span class="source-identity-meta">{{ formatNumber(record.cell.cohortCount) }} {{ t('label.cohorts') }}</span>
                </div>
              </template>
            </template>
          </a-table>
        </Panel>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.model-risk-page {
  max-width: 1560px;
}

.model-risk-notice,
.model-risk-error {
  margin-bottom: var(--am-section-gap);
}

.model-risk-metric-strip {
  grid-template-columns: repeat(4, minmax(156px, 1fr));
}

.model-risk-formula {
  display: grid;
  gap: 14px;
}

.model-risk-formula p {
  max-width: 920px;
  margin: 0;
  color: var(--am-text-soft);
  font-size: 13px;
  line-height: 20px;
}

.model-risk-driver-list {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.model-risk-driver-card {
  min-width: 0;
  padding: 10px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.model-risk-driver-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--am-text);
  font-size: 12px;
  font-weight: 750;
}

.model-risk-driver-value {
  margin-top: 7px;
  color: var(--am-text);
  font-size: 20px;
  font-weight: 760;
  font-variant-numeric: tabular-nums;
  line-height: 26px;
}

.model-risk-chart {
  height: 360px;
}

.model-risk-empty {
  min-height: 260px;
}

.model-risk-source-select {
  width: min(360px, 72vw);
}

.model-risk-table :deep(.ant-table-cell) {
  vertical-align: top;
}

.model-risk-score-cell,
.model-risk-confidence-cell,
.model-risk-sample-cell,
.model-risk-explanation-cell {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.model-risk-score-cell {
  justify-items: end;
}

.model-risk-explanation-cell strong {
  color: var(--am-text);
  font-size: 12px;
  line-height: 18px;
}

.model-risk-driver-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.model-risk-driver-tags .ant-tag {
  max-width: 100%;
  margin-right: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.metric-danger {
  --metric-accent: var(--am-danger);
  --metric-soft: var(--am-danger-soft);
}

@media (max-width: 1180px) {
  .model-risk-driver-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .model-risk-metric-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 680px) {
  .model-risk-driver-list,
  .model-risk-metric-strip {
    grid-template-columns: 1fr;
  }

  .model-risk-chart {
    height: 310px;
  }
}
</style>
