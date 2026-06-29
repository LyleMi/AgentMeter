<script setup lang="ts">
import { computed, onMounted, ref, watch, type DefineComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import ASelect from 'ant-design-vue/es/select'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import { EyeOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons-vue'
import type { AuditFinding } from '../api/types'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { formatDateTime, formatNumber, projectDisplay, sessionFullLabel, sessionLabel, shortPath } from '../presentation/formatters'
import { sourceDisplay } from '../presentation/sourceIdentity'
import {
  auditPath,
  categoryColor,
  cleanQueryValue,
  cleanRouteQuery,
  listAuditFindings,
  normalized,
  severityColor,
  titleCaseFallback
} from './auditSupport'
import { optionalFirstTrimmedRouteQueryValue } from './routeQuery'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const FINDINGS_LIMIT = 500
const route = useRoute()
const router = useRouter()
const findingsResource = useAsyncResource<AuditFinding[]>([])
const findings = computed(() => findingsResource.data.value)
const loading = findingsResource.loading
const error = findingsResource.error
const categoryFilter = ref<string | undefined>(optionalFirstTrimmedRouteQueryValue(route.query.category))
const severityFilter = ref<string | undefined>(optionalFirstTrimmedRouteQueryValue(route.query.severity))
const shellFilter = ref<string | undefined>(optionalFirstTrimmedRouteQueryValue(route.query.shell))
const search = ref(optionalFirstTrimmedRouteQueryValue(route.query.search) || '')
const selectedAgent = computed(() => cleanQueryValue(route.query.agent) || undefined)
let applyingRouteUpdate = false

const { t } = useMessages({
  en: {
    'action.apply': 'Apply',
    'action.refresh': 'Refresh',
    'action.reset': 'Reset',
    'panel.findings': 'Findings',
    'panel.findingsKicker': 'Filtered local audit records grouped by rule evidence and source context',
    'count.loaded': '{count} loaded findings',
    'count.filtered': '{count} loaded matching findings',
    'filter.category': 'Category',
    'filter.severity': 'Severity',
    'filter.shell': 'Shell',
    'filter.search': 'Search title, command, evidence or project',
    'column.severity': 'Severity',
    'column.finding': 'Finding',
    'column.evidence': 'Evidence',
    'column.command': 'Command',
    'column.runtime': 'Shell / Platform',
    'column.source': 'Source / Session',
    'column.time': 'Time',
    'category.command': 'Command',
    'category.privacy': 'Privacy',
    'category.egress': 'Egress',
    'category.file': 'File',
    'severity.critical': 'Critical',
    'severity.high': 'High',
    'severity.medium': 'Medium',
    'severity.low': 'Low',
    'shell.posix': 'POSIX',
    'shell.powershell': 'PowerShell',
    'shell.cmd': 'cmd.exe',
    'shell.unknown': 'Unknown shell',
    'label.rule': 'Rule',
    'label.toolCall': 'Tool',
    'label.line': 'line',
    'label.rawEvent': 'raw',
    'fallback.unknown': 'unknown',
    'fallback.none': '-',
    'empty.loading': 'Loading audit findings...',
    'empty.filtered': 'No findings match the current filters',
    'empty.none': 'No audit findings indexed yet',
    'empty.error': 'Findings unavailable',
    'error.title': 'Audit findings failed to load',
    'tooltip.viewDetails': 'View detail'
  },
  'zh-CN': {
    'action.apply': '应用',
    'action.refresh': '刷新',
    'action.reset': '重置',
    'panel.findings': '发现',
    'panel.findingsKicker': '按规则证据和来源上下文筛选本地审计记录',
    'count.loaded': '已加载 {count} 个发现',
    'count.filtered': '已加载 {count} 个匹配发现',
    'filter.category': '类别',
    'filter.severity': '严重性',
    'filter.shell': 'Shell',
    'filter.search': '搜索标题、命令、证据或项目',
    'column.severity': '严重性',
    'column.finding': '发现',
    'column.evidence': '证据',
    'column.command': '命令',
    'column.runtime': 'Shell / 平台',
    'column.source': '来源 / 会话',
    'column.time': '时间',
    'category.command': '命令',
    'category.privacy': '隐私',
    'category.egress': '外连',
    'category.file': '文件',
    'severity.critical': '严重',
    'severity.high': '高危',
    'severity.medium': '中危',
    'severity.low': '低危',
    'shell.posix': 'POSIX',
    'shell.powershell': 'PowerShell',
    'shell.cmd': 'cmd.exe',
    'shell.unknown': '未知 Shell',
    'label.rule': '规则',
    'label.toolCall': '工具',
    'label.line': '行',
    'label.rawEvent': '原始',
    'fallback.unknown': '未知',
    'fallback.none': '-',
    'empty.loading': '正在加载审计发现...',
    'empty.filtered': '没有发现符合当前筛选条件',
    'empty.none': '还没有已索引的审计发现',
    'empty.error': '无法获取发现',
    'error.title': '审计发现加载失败',
    'tooltip.viewDetails': '查看详情'
  }
})

const columns = computed(() => [
  { title: t('column.severity'), dataIndex: 'severity', key: 'severity', width: 116 },
  { title: t('column.finding'), dataIndex: 'title', key: 'finding', width: 330 },
  { title: t('column.evidence'), dataIndex: 'evidence', key: 'evidence', width: 340 },
  { title: t('column.command'), dataIndex: 'command', key: 'command', width: 310 },
  { title: t('column.runtime'), dataIndex: 'shellFamily', key: 'runtime', width: 170 },
  { title: t('column.source'), dataIndex: 'sessionId', key: 'source', width: 250 },
  { title: t('column.time'), dataIndex: 'timestamp', key: 'time', width: 132 },
  { title: '', key: 'detail', width: 56, align: 'right' }
])
const categoryOptions = computed(() => [
  { value: 'command', label: t('category.command') },
  { value: 'privacy', label: t('category.privacy') },
  { value: 'egress', label: t('category.egress') },
  { value: 'file', label: t('category.file') }
])
const severityOptions = computed(() => [
  { value: 'critical', label: t('severity.critical') },
  { value: 'high', label: t('severity.high') },
  { value: 'medium', label: t('severity.medium') },
  { value: 'low', label: t('severity.low') }
])
const shellOptions = computed(() => [
  { value: 'posix', label: t('shell.posix') },
  { value: 'powershell', label: t('shell.powershell') },
  { value: 'cmd', label: t('shell.cmd') },
  { value: 'unknown', label: t('shell.unknown') }
])
const hasActiveFilters = computed(() => Boolean(selectedAgent.value || categoryFilter.value || severityFilter.value || shellFilter.value || search.value.trim()))
const tableLocale = computed(() => {
  if (loading.value) return { emptyText: t('empty.loading') }
  if (error.value) return { emptyText: t('empty.error') }
  if (hasActiveFilters.value) return { emptyText: t('empty.filtered') }
  return { emptyText: t('empty.none') }
})
const rowCountText = computed(() => {
  const count = formatNumber(findings.value.length)
  return hasActiveFilters.value ? t('count.filtered', { count }) : t('count.loaded', { count })
})

function currentFindingFilters() {
  return {
    agent: selectedAgent.value,
    category: categoryFilter.value,
    severity: severityFilter.value,
    shell: shellFilter.value,
    search: search.value.trim() || undefined,
    limit: FINDINGS_LIMIT,
    offset: 0
  }
}

function load() {
  return findingsResource.run(() => listAuditFindings(currentFindingFilters()), { onErrorData: [] })
}

async function replaceRouteQuery() {
  applyingRouteUpdate = true
  try {
    await router.replace({
      path: '/audit/findings',
      query: {
        ...cleanRouteQuery({ agent: selectedAgent.value }),
        ...cleanRouteQuery({
          category: categoryFilter.value,
          severity: severityFilter.value,
          shell: shellFilter.value,
          search: search.value.trim()
        })
      }
    })
  } finally {
    applyingRouteUpdate = false
  }
}

async function applyFilters() {
  await replaceRouteQuery()
  load()
}

function syncFiltersFromRoute() {
  categoryFilter.value = optionalFirstTrimmedRouteQueryValue(route.query.category)
  severityFilter.value = optionalFirstTrimmedRouteQueryValue(route.query.severity)
  shellFilter.value = optionalFirstTrimmedRouteQueryValue(route.query.shell)
  search.value = optionalFirstTrimmedRouteQueryValue(route.query.search) || ''
}

function resetFilters() {
  categoryFilter.value = undefined
  severityFilter.value = undefined
  shellFilter.value = undefined
  search.value = ''
  applyFilters()
}

function titleCase(value?: string | null) {
  return titleCaseFallback(value, t('fallback.unknown'))
}

function severityLabel(value?: string | null) {
  const severity = normalized(value)
  if (severity === 'critical') return t('severity.critical')
  if (severity === 'high') return t('severity.high')
  if (severity === 'medium') return t('severity.medium')
  if (severity === 'low') return t('severity.low')
  return titleCase(value)
}

function categoryLabel(value?: string | null) {
  const category = normalized(value)
  if (category === 'command') return t('category.command')
  if (category === 'privacy') return t('category.privacy')
  if (category === 'egress') return t('category.egress')
  if (category === 'file') return t('category.file')
  return titleCase(value)
}

function shellLabel(value?: string | null) {
  const shell = normalized(value)
  if (shell === 'posix' || shell === 'bash' || shell === 'sh' || shell === 'zsh') return t('shell.posix')
  if (shell === 'powershell' || shell === 'pwsh') return t('shell.powershell')
  if (shell === 'cmd') return t('shell.cmd')
  if (shell === 'unknown') return t('shell.unknown')
  return titleCase(value)
}

function findingTitle(record: AuditFinding) {
  return record.title || record.ruleId || t('fallback.unknown')
}

function snippet(value?: string | null) {
  return value?.trim() || t('fallback.none')
}

function sessionDisplay(record: AuditFinding) {
  if (!record.sessionId && !record.sessionKey && !record.codexSessionId) return t('fallback.none')
  return sessionLabel({ id: record.sessionId, sessionKey: record.sessionKey || '', codexSessionId: record.codexSessionId || '' })
}

function sessionTitle(record: AuditFinding) {
  if (!record.sessionId && !record.sessionKey && !record.codexSessionId) return t('fallback.none')
  return sessionFullLabel({ id: record.sessionId, sessionKey: record.sessionKey || '', codexSessionId: record.codexSessionId || '' })
}

function sourceContext(record: AuditFinding) {
  const parts: string[] = []
  if (record.sourceLine) parts.push(`${t('label.line')} ${formatNumber(record.sourceLine)}`)
  if (record.rawEventId) parts.push(`${t('label.rawEvent')} #${formatNumber(record.rawEventId)}`)
  return parts.join(' · ')
}

function sourceInfo(record: AuditFinding) {
  return sourceDisplay(record, t('fallback.unknown'))
}

function sourceSecondary(record: AuditFinding) {
  if (sourceInfo(record).secondary) return sourceInfo(record).secondary
  if (record.projectPath) return projectDisplay(record.projectPath).main
  return shortPath(record.rawSourcePath || '')
}

function safeDateTime(value?: string | null) {
  if (!value) return t('fallback.none')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return formatDateTime(value)
}

function openFinding(record: AuditFinding) {
  router.push(auditPath(`/audit/findings/${record.id}`, { agent: selectedAgent.value }))
}

watch(
  () => [route.query.agent, route.query.category, route.query.severity, route.query.shell, route.query.search],
  () => {
    if (applyingRouteUpdate) return
    syncFiltersFromRoute()
    load()
  }
)

onMounted(load)
</script>

<template>
  <div class="section-stack audit-findings-page">
    <a-alert
      v-if="error"
      class="audit-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <section class="panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('panel.findings') }}</h2>
          <div class="panel-kicker">{{ t('panel.findingsKicker') }}</div>
        </div>
        <div class="panel-actions">
          <span class="row-count">{{ rowCountText }}</span>
          <a-button :loading="loading" @click="load">
            <template #icon>
              <ReloadOutlined />
            </template>
            {{ t('action.refresh') }}
          </a-button>
        </div>
      </div>
      <div class="panel-body">
        <div class="toolbar toolbar-compact audit-toolbar">
          <div class="toolbar-left">
            <a-input
              v-model:value="search"
              class="control-wide audit-search"
              allow-clear
              :placeholder="t('filter.search')"
              @press-enter="applyFilters"
            >
              <template #prefix>
                <SearchOutlined />
              </template>
            </a-input>
            <a-select
              v-model:value="categoryFilter"
              class="control-medium"
              allow-clear
              :placeholder="t('filter.category')"
              :options="categoryOptions"
              @change="applyFilters"
            />
            <a-select
              v-model:value="severityFilter"
              class="control-medium"
              allow-clear
              :placeholder="t('filter.severity')"
              :options="severityOptions"
              @change="applyFilters"
            />
            <a-select
              v-model:value="shellFilter"
              class="control-medium"
              allow-clear
              :placeholder="t('filter.shell')"
              :options="shellOptions"
              @change="applyFilters"
            />
            <a-button type="primary" @click="applyFilters">{{ t('action.apply') }}</a-button>
            <a-button @click="resetFilters">{{ t('action.reset') }}</a-button>
          </div>
        </div>

        <a-table
          class="dense-table audit-table"
          :columns="columns"
          :data-source="findings"
          :loading="loading"
          :locale="tableLocale"
          :pagination="{ pageSize: 20, showSizeChanger: true }"
          :scroll="{ x: 1700 }"
          row-key="id"
          size="small"
          table-layout="fixed"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'severity'">
              <div class="audit-tag-stack">
                <a-tag class="status-tag audit-severity-tag" :color="severityColor(record.severity)">
                  {{ severityLabel(record.severity) }}
                </a-tag>
                <a-tag class="status-tag audit-category-tag" :color="categoryColor(record.category)">
                  {{ categoryLabel(record.category) }}
                </a-tag>
              </div>
            </template>

            <template v-else-if="column.key === 'finding'">
              <a-typography-text class="audit-finding-title" :ellipsis="{ tooltip: findingTitle(record) }">
                {{ findingTitle(record) }}
              </a-typography-text>
              <div v-if="record.description" class="audit-description">{{ record.description }}</div>
              <div class="audit-rule-row">
                <span>{{ t('label.rule') }}</span>
                <code class="mono">{{ record.ruleId || t('fallback.unknown') }}</code>
              </div>
            </template>

            <template v-else-if="column.key === 'evidence'">
              <a-tooltip :title="snippet(record.evidence)" placement="topLeft">
                <pre class="audit-snippet">{{ snippet(record.evidence) }}</pre>
              </a-tooltip>
            </template>

            <template v-else-if="column.key === 'command'">
              <a-tooltip :title="snippet(record.command)" placement="topLeft">
                <pre class="audit-snippet audit-command">{{ snippet(record.command) }}</pre>
              </a-tooltip>
              <div v-if="record.toolCallId" class="timeline-event-raw mono">
                {{ t('label.toolCall') }} #{{ formatNumber(record.toolCallId) }}
              </div>
            </template>

            <template v-else-if="column.key === 'runtime'">
              <div class="audit-runtime-tags">
                <a-tag class="model-lite-tag">{{ shellLabel(record.shellFamily) }}</a-tag>
                <a-tag class="model-lite-tag">{{ record.platform || t('fallback.unknown') }}</a-tag>
              </div>
            </template>

            <template v-else-if="column.key === 'source'">
              <a-button type="link" size="small" class="audit-session-link source-identity-name" @click="openFinding(record)">
                {{ sourceInfo(record).label }}
              </a-button>
              <a-tooltip :title="sourceInfo(record).title || record.projectPath || record.rawSourcePath || ''" placement="topLeft">
                <div class="source-identity-meta">{{ sourceSecondary(record) }}</div>
              </a-tooltip>
              <a-tooltip :title="sessionTitle(record)" placement="topLeft">
                <div class="timeline-event-raw mono">{{ sessionDisplay(record) }}</div>
              </a-tooltip>
              <div v-if="sourceContext(record)" class="timeline-event-raw mono">{{ sourceContext(record) }}</div>
            </template>

            <template v-else-if="column.key === 'time'">
              <span class="audit-time">{{ safeDateTime(record.timestamp) }}</span>
            </template>

            <template v-else-if="column.key === 'detail'">
              <a-tooltip :title="t('tooltip.viewDetails')">
                <a-button type="text" size="small" @click="openFinding(record)">
                  <template #icon>
                    <EyeOutlined />
                  </template>
                </a-button>
              </a-tooltip>
            </template>
          </template>
        </a-table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.audit-error {
  margin-bottom: var(--am-section-gap);
}

.audit-toolbar {
  align-items: flex-start;
}

.audit-search {
  width: 360px;
}

.audit-table :deep(.ant-table-tbody > tr > td) {
  vertical-align: top;
}

.audit-tag-stack,
.audit-runtime-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.audit-severity-tag,
.audit-category-tag {
  margin-inline-end: 0;
}

.audit-finding-title {
  display: block;
  max-width: 100%;
  color: var(--am-text);
  font-weight: 700;
}

.audit-description {
  display: -webkit-box;
  overflow: hidden;
  margin-top: 4px;
  color: var(--am-muted);
  font-size: 12px;
  line-height: 18px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.audit-rule-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  color: var(--am-muted);
  font-size: 11px;
}

.audit-rule-row code {
  min-width: 0;
  overflow: hidden;
  padding: 1px 5px;
  color: var(--am-text-soft);
  text-overflow: ellipsis;
  white-space: nowrap;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.audit-snippet {
  display: -webkit-box;
  overflow: hidden;
  max-width: 100%;
  min-height: 34px;
  max-height: 52px;
  margin: 0;
  padding: 6px 7px;
  color: var(--am-text-soft);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 16px;
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.audit-command {
  color: var(--am-text);
  background: var(--am-surface);
}

.audit-session-link {
  display: inline-block;
  height: auto;
  max-width: 100%;
  padding: 0;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  vertical-align: top;
  white-space: nowrap;
}

.audit-source-path {
  max-width: 100%;
  margin-top: 2px;
  overflow: hidden;
  color: var(--am-text-soft);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.audit-time {
  color: var(--am-text-soft);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
</style>
