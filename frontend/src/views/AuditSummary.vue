<script setup lang="ts">
import { computed, onMounted, watch, type DefineComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import {
  CodeOutlined,
  GlobalOutlined,
  HistoryOutlined,
  LockOutlined,
  ReloadOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import type { AuditFinding, AuditSummary } from '../api/types'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { formatDateTime, formatNumber, projectDisplay, sessionFullLabel, sessionLabel, shortPath } from '../presentation/formatters'
import { sourceDisplay } from '../presentation/sourceIdentity'
import { auditPath, categoryColor, cleanQueryValue, getAuditSummary, severityColor, titleCaseFallback } from './auditSupport'

const ATable = AntTable as unknown as DefineComponent
const route = useRoute()
const router = useRouter()
const summaryResource = useAsyncResource<AuditSummary | null>(null)
const summary = computed(() => summaryResource.data.value)
const loading = summaryResource.loading
const error = summaryResource.error
const selectedAgent = computed(() => cleanQueryValue(route.query.agent) || undefined)
const { t } = useMessages({
  en: {
    'action.refresh': 'Refresh',
    'action.viewAll': 'View all findings',
    'metric.total': 'Total Findings',
    'metric.totalNote': 'Indexed audit records',
    'metric.criticalHigh': 'Critical / High',
    'metric.criticalHighNote': 'Findings needing prompt review',
    'metric.command': 'Command',
    'metric.commandNote': 'Shell and execution risks',
    'metric.privacy': 'Privacy',
    'metric.privacyNote': 'Sensitive data exposure',
    'metric.egressFile': 'Egress / File',
    'metric.egressFileNote': 'Network and filesystem evidence',
    'metric.sessions': 'Sessions',
    'metric.sessionsNote': 'Sessions with findings',
    'recent.title': 'Recent Findings',
    'recent.kicker': 'Latest local audit findings with session context',
    'column.severity': 'Severity',
    'column.finding': 'Finding',
    'column.session': 'Source',
    'column.time': 'Time',
    'severity.critical': 'Critical',
    'severity.high': 'High',
    'severity.medium': 'Medium',
    'severity.low': 'Low',
    'category.command': 'Command',
    'category.privacy': 'Privacy',
    'category.egress': 'Egress',
    'category.file': 'File',
    'empty.loading': 'Loading audit summary...',
    'empty.none': 'No audit findings indexed yet',
    'error.title': 'Audit summary failed to load',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    'action.refresh': '刷新',
    'action.viewAll': '查看全部发现',
    'metric.total': '发现总数',
    'metric.totalNote': '已索引审计记录',
    'metric.criticalHigh': '严重 / 高危',
    'metric.criticalHighNote': '需要优先检查的发现',
    'metric.command': '命令',
    'metric.commandNote': 'Shell 与执行风险',
    'metric.privacy': '隐私',
    'metric.privacyNote': '敏感数据暴露',
    'metric.egressFile': '外连 / 文件',
    'metric.egressFileNote': '网络与文件系统证据',
    'metric.sessions': '会话',
    'metric.sessionsNote': '包含发现的会话',
    'recent.title': '最近发现',
    'recent.kicker': '包含会话上下文的最新本地审计发现',
    'column.severity': '严重性',
    'column.finding': '发现',
    'column.session': '来源',
    'column.time': '时间',
    'severity.critical': '严重',
    'severity.high': '高危',
    'severity.medium': '中危',
    'severity.low': '低危',
    'category.command': '命令',
    'category.privacy': '隐私',
    'category.egress': '外连',
    'category.file': '文件',
    'empty.loading': '正在加载审计汇总...',
    'empty.none': '还没有已索引的审计发现',
    'error.title': '审计汇总加载失败',
    'fallback.unknown': '未知'
  }
})

const columns = computed(() => [
  { title: t('column.severity'), dataIndex: 'severity', key: 'severity', width: 130 },
  { title: t('column.finding'), dataIndex: 'title', key: 'finding' },
  { title: t('column.session'), dataIndex: 'sessionId', key: 'session', width: 260 },
  { title: t('column.time'), dataIndex: 'timestamp', key: 'time', width: 140 }
])
const tableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.none') }))
const summaryCards = computed(() => {
  const item = summary.value
  return [
    {
      label: t('metric.total'),
      value: formatNumber(item?.totalFindings),
      note: t('metric.totalNote'),
      icon: WarningOutlined,
      tone: 'metric-primary'
    },
    {
      label: t('metric.criticalHigh'),
      value: `${formatNumber(item?.criticalFindings)} / ${formatNumber(item?.highFindings)}`,
      note: t('metric.criticalHighNote'),
      icon: WarningOutlined,
      tone: 'metric-danger'
    },
    {
      label: t('metric.command'),
      value: formatNumber(item?.commandFindings),
      note: t('metric.commandNote'),
      icon: CodeOutlined,
      tone: 'metric-info'
    },
    {
      label: t('metric.privacy'),
      value: formatNumber(item?.privacyFindings),
      note: t('metric.privacyNote'),
      icon: LockOutlined,
      tone: 'metric-warning'
    },
    {
      label: t('metric.egressFile'),
      value: `${formatNumber(item?.egressFindings)} / ${formatNumber(item?.fileFindings)}`,
      note: t('metric.egressFileNote'),
      icon: GlobalOutlined,
      tone: 'metric-neutral'
    },
    {
      label: t('metric.sessions'),
      value: formatNumber(item?.sessionsWithFindings),
      note: t('metric.sessionsNote'),
      icon: HistoryOutlined,
      tone: 'metric-success'
    }
  ]
})

function load() {
  return summaryResource.run(() => getAuditSummary({ agent: selectedAgent.value }))
}

function title(record: AuditFinding) {
  return record.title || record.ruleId || t('fallback.unknown')
}

function severityLabel(value?: string | null) {
  const severity = (value || '').trim().toLowerCase()
  if (severity === 'critical') return t('severity.critical')
  if (severity === 'high') return t('severity.high')
  if (severity === 'medium') return t('severity.medium')
  if (severity === 'low') return t('severity.low')
  return titleCaseFallback(value, t('fallback.unknown'))
}

function categoryLabel(value?: string | null) {
  const category = (value || '').trim().toLowerCase()
  if (category === 'command') return t('category.command')
  if (category === 'privacy') return t('category.privacy')
  if (category === 'egress') return t('category.egress')
  if (category === 'file') return t('category.file')
  return titleCaseFallback(value, t('fallback.unknown'))
}

function sessionDisplay(record: AuditFinding) {
  return sessionLabel({ id: record.sessionId, sessionKey: record.sessionKey || '', codexSessionId: record.codexSessionId || '' })
}

function sessionTitle(record: AuditFinding) {
  return sessionFullLabel({ id: record.sessionId, sessionKey: record.sessionKey || '', codexSessionId: record.codexSessionId || '' })
}

function sourceSecondary(record: AuditFinding) {
  if (sourceInfo(record).secondary) return sourceInfo(record).secondary
  if (record.projectPath) return projectDisplay(record.projectPath).main
  return shortPath(record.rawSourcePath || '')
}

function sourceInfo(record: AuditFinding) {
  return sourceDisplay(record, t('fallback.unknown'))
}

function openFinding(record: AuditFinding) {
  router.push(auditPath(`/audit/findings/${record.id}`, { agent: selectedAgent.value }))
}

function openList() {
  router.push(auditPath('/audit/findings', { agent: selectedAgent.value }))
}

function summaryRow(record: AuditFinding) {
  return { class: 'is-clickable-row', onClick: () => openFinding(record) }
}

watch(() => route.query.agent, load)
onMounted(load)
</script>

<template>
  <div class="section-stack audit-summary-page">
    <a-alert
      v-if="error"
      class="audit-error"
      type="error"
      show-icon
      :message="t('error.title')"
      :description="error"
    />

    <section class="metric-strip audit-metric-strip">
      <div v-for="item in summaryCards" :key="item.label" class="metric-strip-item" :class="item.tone">
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

    <section class="panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('recent.title') }}</h2>
          <div class="panel-kicker">{{ t('recent.kicker') }}</div>
        </div>
        <div class="panel-actions">
          <a-button @click="openList">{{ t('action.viewAll') }}</a-button>
          <a-button :loading="loading" @click="load">
            <template #icon>
              <ReloadOutlined />
            </template>
            {{ t('action.refresh') }}
          </a-button>
        </div>
      </div>
      <a-table
        class="dense-table audit-summary-table"
        :columns="columns"
        :data-source="summary?.recentFindings || []"
        :loading="loading"
        :locale="tableLocale"
        :pagination="false"
        row-key="id"
        size="small"
        :scroll="{ x: 900 }"
        :custom-row="summaryRow"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'severity'">
            <div class="audit-tag-stack">
              <a-tag class="status-tag" :color="severityColor(record.severity)">{{ severityLabel(record.severity) }}</a-tag>
              <a-tag class="status-tag" :color="categoryColor(record.category)">{{ categoryLabel(record.category) }}</a-tag>
            </div>
          </template>
          <template v-else-if="column.key === 'finding'">
            <div class="audit-finding-title">{{ title(record) }}</div>
            <div class="timeline-event-raw mono">{{ record.ruleId || t('fallback.unknown') }}</div>
          </template>
          <template v-else-if="column.key === 'session'">
            <div class="source-identity-name">{{ sourceInfo(record).label }}</div>
            <a-tooltip :title="sourceInfo(record).title || record.projectPath || record.rawSourcePath || ''" placement="topLeft">
              <div class="source-identity-meta">{{ sourceSecondary(record) }}</div>
            </a-tooltip>
            <a-tooltip :title="sessionTitle(record)" placement="topLeft">
              <div class="timeline-event-raw mono">{{ sessionDisplay(record) }}</div>
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'time'">
            <span class="audit-time">{{ formatDateTime(record.timestamp) }}</span>
          </template>
        </template>
      </a-table>
    </section>
  </div>
</template>

<style scoped>
.audit-error {
  margin-bottom: var(--am-section-gap);
}

.audit-metric-strip {
  grid-template-columns: repeat(6, minmax(132px, 1fr));
}

.metric-danger {
  --metric-accent: var(--am-danger);
  --metric-soft: var(--am-danger-soft);
}

.audit-tag-stack {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.audit-finding-title {
  color: var(--am-text);
  font-weight: 700;
}

.audit-source-path,
.audit-time {
  color: var(--am-text-soft);
  font-size: 12px;
}

.audit-source-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1280px) {
  .audit-metric-strip {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
