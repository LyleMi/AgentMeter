<script setup lang="ts">
import { computed, ref } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASegmented from 'ant-design-vue/es/segmented'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import {
  DollarCircleOutlined,
  FolderOpenOutlined,
  RightOutlined
} from '@ant-design/icons-vue'
import {
  formatCost,
  formatDisplayCost,
  formatDisplayNumber,
  formatNumber,
  formatPercent,
  projectDisplay,
  type UsageBreakdownBucket
} from '../../api'
import EmptyState from '../../components/ui/EmptyState.vue'
import { useMessages } from '../../i18n'
import { useTokensContext } from './tokensContext'

type SortMode = 'cost' | 'tokens'

const { analytics, breakdownRows, loading, scopeFilters, updateScopeFilters } = useTokensContext()
const sortMode = ref<SortMode>('cost')

const { t } = useMessages({
  en: {
    'title': 'Project spend',
    'kicker': 'Token volume and estimated cost grouped by local project',
    'metric.projects': 'Active projects',
    'metric.projectsNote': 'Projects with indexed usage in this scope',
    'metric.tokens': 'Project tokens',
    'metric.tokensNote': 'Across {count} indexed sessions',
    'metric.cost': 'Estimated spend',
    'metric.costNote': '{percent} of projects fully priced',
    'sort.cost': 'Cost',
    'sort.tokens': 'Tokens',
    'column.project': 'Project',
    'column.sessions': 'Sessions',
    'column.tokens': 'Tokens',
    'column.cost': 'Estimated cost',
    'column.share': 'Share',
    'action.focus': 'View only',
    'status.unpriced': 'Partly unpriced',
    'empty.title': 'No project usage yet',
    'empty.text': 'Project totals appear after sessions with a project path have been indexed.',
    'fallback.unknown': 'Unknown project'
  },
  'zh-CN': {
    'title': '项目消耗',
    'kicker': '按本地项目汇总 Token 用量和预估金额',
    'metric.projects': '活跃项目',
    'metric.projectsNote': '当前范围内有已索引用量的项目',
    'metric.tokens': '项目 Token',
    'metric.tokensNote': '来自 {count} 个已索引会话',
    'metric.cost': '预估金额',
    'metric.costNote': '{percent} 的项目价格覆盖完整',
    'sort.cost': '按金额',
    'sort.tokens': '按 Token',
    'column.project': '项目',
    'column.sessions': '会话',
    'column.tokens': 'Token',
    'column.cost': '预估金额',
    'column.share': '占比',
    'action.focus': '仅看此项目',
    'status.unpriced': '部分未定价',
    'empty.title': '暂无项目用量',
    'empty.text': '索引到带项目路径的会话后，这里会显示项目汇总。',
    'fallback.unknown': '未知项目'
  }
})

const projects = computed(() => (breakdownRows.value || []).filter((row) => row.projectPath))
const sortedProjects = computed(() => [...projects.value].sort((left, right) => {
  const delta = sortMode.value === 'cost'
    ? (right.estimatedCostUsd || 0) - (left.estimatedCostUsd || 0)
    : right.totalTokens - left.totalTokens
  return delta || right.totalTokens - left.totalTokens || (left.projectPath || '').localeCompare(right.projectPath || '')
}))
const pricedProjects = computed(() => projects.value.filter((row) => !row.unpriced).length)
const pricingCoverage = computed(() => projects.value.length ? pricedProjects.value / projects.value.length : 0)
const totalProjectTokens = computed(() => projects.value.reduce((sum, row) => sum + row.totalTokens, 0))
const totalProjectCost = computed(() => projects.value.reduce((sum, row) => sum + (row.estimatedCostUsd || 0), 0))
const maxTokens = computed(() => Math.max(0, ...projects.value.map((row) => row.totalTokens)))
const maxCost = computed(() => Math.max(0, ...projects.value.map((row) => row.estimatedCostUsd || 0)))

const metrics = computed(() => [
  {
    label: t('metric.projects'),
    value: formatDisplayNumber(projects.value.length),
    note: t('metric.projectsNote'),
    icon: FolderOpenOutlined,
    tone: 'metric-primary'
  },
  {
    label: t('metric.tokens'),
    value: formatDisplayNumber(totalProjectTokens.value),
    note: t('metric.tokensNote', { count: formatDisplayNumber(analytics.value?.totalSessions).main }),
    icon: FolderOpenOutlined,
    tone: 'metric-neutral'
  },
  {
    label: t('metric.cost'),
    value: formatDisplayCost(totalProjectCost.value),
    note: t('metric.costNote', { percent: formatPercent(pricingCoverage.value) }),
    icon: DollarCircleOutlined,
    tone: projects.value.some((row) => row.unpriced) ? 'metric-warning' : 'metric-success'
  }
])

const sortOptions = computed(() => [
  { label: t('sort.cost'), value: 'cost' },
  { label: t('sort.tokens'), value: 'tokens' }
])

function projectInfo(row: UsageBreakdownBucket) {
  const display = projectDisplay(row.projectPath)
  return {
    name: display.main || t('fallback.unknown'),
    path: display.full || row.projectPath || t('fallback.unknown')
  }
}

function ratioWidth(value: number, maximum: number) {
  if (value <= 0 || maximum <= 0) return '0%'
  return `${Math.max(2, Math.round(value / maximum * 100))}%`
}

function share(row: UsageBreakdownBucket) {
  const total = sortMode.value === 'cost' ? totalProjectCost.value : totalProjectTokens.value
  const value = sortMode.value === 'cost' ? row.estimatedCostUsd || 0 : row.totalTokens
  return total > 0 ? value / total : 0
}

async function focusProject(row: UsageBreakdownBucket) {
  if (!row.projectPath) return
  await updateScopeFilters({ ...scopeFilters.value, project: row.projectPath })
}
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="metric-strip project-metrics">
        <div v-for="item in metrics" :key="item.label" class="metric-strip-item" :class="item.tone">
          <div class="metric-strip-head">
            <span class="metric-label">{{ item.label }}</span>
            <span class="metric-strip-icon"><component :is="item.icon" /></span>
          </div>
          <div class="metric-strip-value" :title="item.value.full">{{ item.value.main }}<small>{{ item.value.suffix }}</small></div>
          <div class="metric-strip-note">{{ item.note }}</div>
        </div>
      </section>

      <section class="panel project-ledger">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('title') }}</h2>
            <div class="panel-kicker">{{ t('kicker') }}</div>
          </div>
          <a-segmented v-model:value="sortMode" :options="sortOptions" />
        </div>

        <div v-if="sortedProjects.length" class="project-table" role="table" :aria-label="t('title')">
          <div class="project-table-head" role="row">
            <span>{{ t('column.project') }}</span>
            <span>{{ t('column.sessions') }}</span>
            <span>{{ t('column.tokens') }}</span>
            <span>{{ t('column.cost') }}</span>
            <span>{{ t('column.share') }}</span>
            <span aria-hidden="true"></span>
          </div>
          <div v-for="row in sortedProjects" :key="row.projectPath" class="project-row" role="row">
            <div class="project-cell project-identity" role="cell">
              <span class="project-rank">{{ String(sortedProjects.indexOf(row) + 1).padStart(2, '0') }}</span>
              <span class="project-copy">
                <strong :title="projectInfo(row).path">{{ projectInfo(row).name }}</strong>
                <span :title="projectInfo(row).path">{{ projectInfo(row).path }}</span>
              </span>
            </div>
            <span class="project-cell number-cell" role="cell">{{ formatNumber(row.sessionCount) }}</span>
            <span class="project-cell number-cell" role="cell">{{ formatNumber(row.totalTokens) }}</span>
            <span class="project-cell cost-cell" role="cell">
              <span class="number-cell">{{ formatCost(row.estimatedCostUsd) }}</span>
              <a-tag v-if="row.unpriced" color="warning">{{ t('status.unpriced') }}</a-tag>
            </span>
            <div class="project-cell project-share" role="cell">
              <span>{{ formatPercent(share(row)) }}</span>
              <span class="project-bars" aria-hidden="true">
                <i class="token-bar" :style="{ width: ratioWidth(row.totalTokens, maxTokens) }"></i>
                <i class="cost-bar" :style="{ width: ratioWidth(row.estimatedCostUsd || 0, maxCost) }"></i>
              </span>
            </div>
            <div class="project-cell project-action" role="cell">
              <a-button type="text" size="small" @click="focusProject(row)">
                {{ t('action.focus') }} <RightOutlined />
              </a-button>
            </div>
          </div>
        </div>
        <EmptyState v-else :title="t('empty.title')" :text="t('empty.text')" />
      </section>
    </div>
  </a-spin>
</template>

<style scoped>
.project-metrics { grid-template-columns: repeat(3, minmax(180px, 1fr)); }
.metric-strip-value small { margin-left: 3px; font-size: .48em; font-weight: 650; }
.project-ledger { overflow: hidden; }
.project-table { min-width: 880px; }
.project-table-head,
.project-row { display: grid; grid-template-columns: minmax(280px, 1.7fr) 84px 120px 130px minmax(160px, .8fr) 112px; align-items: center; column-gap: 12px; }
.project-table-head { padding: 10px 14px; border-bottom: 1px solid var(--am-border-subtle); color: var(--am-muted); font-size: 12px; font-weight: 650; text-transform: uppercase; letter-spacing: .04em; }
.project-table-head span:not(:first-child) { text-align: right; }
.project-row { position: relative; min-height: 76px; padding: 12px 14px; border-bottom: 1px solid var(--am-border-subtle); transition: background-color .16s ease; }
.project-row:last-child { border-bottom: 0; }
.project-row:hover { background: var(--am-row-hover); }
.project-cell { min-width: 0; }
.project-identity { display: flex; align-items: center; gap: 12px; }
.project-rank { flex: 0 0 30px; color: var(--am-primary); font: 700 12px/1 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; letter-spacing: .08em; }
.project-copy { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.project-copy strong { overflow: hidden; color: var(--am-text); font-size: 14px; text-overflow: ellipsis; white-space: nowrap; }
.project-copy > span { overflow: hidden; color: var(--am-muted); font: 12px/1.4 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; text-overflow: ellipsis; white-space: nowrap; }
.cost-cell { display: flex; align-items: flex-end; flex-direction: column; gap: 4px; }
.cost-cell :deep(.ant-tag) { margin: 0; font-size: 10px; line-height: 16px; }
.project-share { display: grid; grid-template-columns: 48px 1fr; align-items: center; gap: 8px; color: var(--am-muted); font-variant-numeric: tabular-nums; text-align: right; }
.project-bars { display: flex; flex-direction: column; gap: 3px; }
.project-bars i { display: block; height: 3px; border-radius: 2px; }
.token-bar { background: var(--am-primary); }
.cost-bar { background: var(--am-warning); }
.project-action { text-align: right; }
.project-action :deep(.ant-btn) { color: var(--am-muted); font-size: 12px; }
.project-action :deep(.ant-btn:hover) { color: var(--am-primary); }
@media (max-width: 900px) {
  .project-metrics { grid-template-columns: 1fr; }
  .project-ledger { overflow-x: auto; }
}
@media (prefers-reduced-motion: reduce) { .project-row { transition: none; } }
</style>
