<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import {
  ArrowLeftOutlined,
  FileSearchOutlined,
  HistoryOutlined,
  ReloadOutlined
} from '@ant-design/icons-vue'
import { api, formatDateTime, formatNumber, sessionLabel, shortPath, type SessionDetail } from '../api'
import type { AuditFinding } from '../api/types'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { sourceDisplay } from '../presentation/sourceIdentity'
import {
  auditPath,
  categoryColor,
  cleanQueryValue,
  getAuditFinding,
  normalized,
  severityColor,
  titleCaseFallback
} from './auditSupport'

interface AuditDetailState {
  finding: AuditFinding | null
  sessionDetail: SessionDetail | null
}

const route = useRoute()
const router = useRouter()
const resource = useAsyncResource<AuditDetailState>({ finding: null, sessionDetail: null })
const loading = resource.loading
const error = resource.error
const finding = computed(() => resource.data.value.finding)
const sessionDetail = computed(() => resource.data.value.sessionDetail)
const selectedAgent = computed(() => cleanQueryValue(route.query.agent) || undefined)
const findingId = computed(() => Number(cleanQueryValue(route.params.id)))

const { t } = useMessages({
  en: {
    'action.back': 'Back to findings',
    'action.refresh': 'Refresh',
    'action.openSession': 'Open session',
    'panel.finding': 'Finding Detail',
    'panel.findingKicker': 'Rule, evidence, command and source event context',
    'panel.session': 'Linked Session',
    'panel.sessionKicker': 'The audit finding is tied back to this indexed local session',
    'label.severity': 'Severity',
    'label.category': 'Category',
    'label.rule': 'Rule',
    'label.decision': 'Decision',
    'label.command': 'Command',
    'label.evidence': 'Evidence',
    'label.description': 'Description',
    'label.shell': 'Shell',
    'label.platform': 'Platform',
    'label.time': 'Time',
    'label.sourceLine': 'Source line',
    'label.rawEvent': 'Raw event',
    'label.toolCall': 'Tool call',
    'label.session': 'Session',
    'label.agent': 'Source',
    'label.project': 'Project',
    'label.model': 'Model',
    'label.tokens': 'Tokens',
    'label.source': 'Source',
    'severity.critical': 'Critical',
    'severity.high': 'High',
    'severity.medium': 'Medium',
    'severity.low': 'Low',
    'category.command': 'Command',
    'category.privacy': 'Privacy',
    'category.egress': 'Egress',
    'category.file': 'File',
    'empty.missing': 'Audit finding not found',
    'error.title': 'Audit detail failed to load',
    'fallback.unknown': 'unknown',
    'fallback.none': '-'
  },
  'zh-CN': {
    'action.back': '返回发现列表',
    'action.refresh': '刷新',
    'action.openSession': '打开会话',
    'panel.finding': '发现详情',
    'panel.findingKicker': '规则、证据、命令和来源事件上下文',
    'panel.session': '关联会话',
    'panel.sessionKicker': '该审计发现已关联回这个本地索引会话',
    'label.severity': '严重性',
    'label.category': '类别',
    'label.rule': '规则',
    'label.decision': '判断',
    'label.command': '命令',
    'label.evidence': '证据',
    'label.description': '描述',
    'label.shell': 'Shell',
    'label.platform': '平台',
    'label.time': '时间',
    'label.sourceLine': '来源行',
    'label.rawEvent': '原始事件',
    'label.toolCall': '工具调用',
    'label.session': '会话',
    'label.agent': '来源',
    'label.project': '项目',
    'label.model': '模型',
    'label.tokens': 'Token',
    'label.source': '来源',
    'severity.critical': '严重',
    'severity.high': '高危',
    'severity.medium': '中危',
    'severity.low': '低危',
    'category.command': '命令',
    'category.privacy': '隐私',
    'category.egress': '外连',
    'category.file': '文件',
    'empty.missing': '未找到审计发现',
    'error.title': '审计详情加载失败',
    'fallback.unknown': '未知',
    'fallback.none': '-'
  }
})

const findingTitle = computed(() => finding.value?.title || finding.value?.ruleId || t('fallback.unknown'))
const sessionName = computed(() => {
  const item = sessionDetail.value?.session
  if (item) return sessionLabel(item)
  const record = finding.value
  if (!record) return t('fallback.none')
  return record.sessionKey || record.codexSessionId || `#${formatNumber(record.sessionId)}`
})
const linkedSource = computed(() => {
  const record = sessionDetail.value?.session || finding.value
  return record ? sourceDisplay(record, t('fallback.unknown')) : null
})

function load() {
  if (!Number.isFinite(findingId.value) || findingId.value <= 0) {
    resource.run(async () => ({ finding: null, sessionDetail: null }), { onErrorData: { finding: null, sessionDetail: null } })
    return
  }
  return resource.run(async () => {
    const detail = await getAuditFinding(findingId.value, { agent: selectedAgent.value })
    let loadedSession: SessionDetail | null = null
    if (detail.finding.sessionId) {
      try {
        loadedSession = await api.getSessionDetail(detail.finding.sessionId)
      } catch {
        loadedSession = null
      }
    }
    return { finding: detail.finding, sessionDetail: loadedSession }
  }, { onErrorData: { finding: null, sessionDetail: null } })
}

function label(value?: string | null) {
  return titleCaseFallback(value, t('fallback.unknown'))
}

function severityLabel(value?: string | null) {
  const severity = normalized(value)
  if (severity === 'critical') return t('severity.critical')
  if (severity === 'high') return t('severity.high')
  if (severity === 'medium') return t('severity.medium')
  if (severity === 'low') return t('severity.low')
  return label(value)
}

function categoryLabel(value?: string | null) {
  const category = normalized(value)
  if (category === 'command') return t('category.command')
  if (category === 'privacy') return t('category.privacy')
  if (category === 'egress') return t('category.egress')
  if (category === 'file') return t('category.file')
  return label(value)
}

function codeBlock(value?: string | null) {
  return value?.trim() || t('fallback.none')
}

function sourceContext(record: AuditFinding) {
  const items = [
    record.sourceLine ? `${t('label.sourceLine')} ${formatNumber(record.sourceLine)}` : '',
    record.rawEventId ? `${t('label.rawEvent')} #${formatNumber(record.rawEventId)}` : '',
    record.toolCallId ? `${t('label.toolCall')} #${formatNumber(record.toolCallId)}` : ''
  ].filter(Boolean)
  return items.join(' · ') || t('fallback.none')
}

function backToList() {
  router.push(auditPath('/audit/findings', { agent: selectedAgent.value }))
}

function openSession() {
  const id = finding.value?.sessionId
  if (id) router.push(`/sessions/${id}`)
}

watch(() => [route.params.id, route.query.agent], load)
onMounted(load)
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack audit-detail-page">
      <a-alert
        v-if="error"
        class="audit-error"
        type="error"
        show-icon
        :message="t('error.title')"
        :description="error"
      />

      <div class="toolbar toolbar-compact audit-detail-actions">
        <div class="toolbar-left">
          <a-button @click="backToList">
            <template #icon>
              <ArrowLeftOutlined />
            </template>
            {{ t('action.back') }}
          </a-button>
        </div>
        <div class="toolbar-right">
          <a-button :loading="loading" @click="load">
            <template #icon>
              <ReloadOutlined />
            </template>
            {{ t('action.refresh') }}
          </a-button>
        </div>
      </div>

      <div v-if="!finding && !loading" class="empty-state empty-state-compact">
        <FileSearchOutlined class="empty-state-icon" />
        <div class="empty-state-title">{{ t('empty.missing') }}</div>
      </div>

      <template v-if="finding">
        <section class="panel audit-detail-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('panel.finding') }}</h2>
              <div class="panel-kicker">{{ t('panel.findingKicker') }}</div>
            </div>
            <div class="audit-detail-tags">
              <a-tag class="status-tag" :color="severityColor(finding.severity)">
                {{ severityLabel(finding.severity) }}
              </a-tag>
              <a-tag class="status-tag" :color="categoryColor(finding.category)">
                {{ categoryLabel(finding.category) }}
              </a-tag>
            </div>
          </div>
          <div class="audit-detail-grid">
            <div class="audit-detail-main">
              <h3>{{ findingTitle }}</h3>
              <p>{{ finding.description || t('fallback.none') }}</p>
              <div class="audit-detail-fields">
                <div>
                  <span>{{ t('label.rule') }}</span>
                  <strong class="mono">{{ finding.ruleId || t('fallback.unknown') }}</strong>
                </div>
                <div>
                  <span>{{ t('label.decision') }}</span>
                  <strong>{{ finding.decision || t('fallback.unknown') }}</strong>
                </div>
                <div>
                  <span>{{ t('label.shell') }}</span>
                  <strong>{{ label(finding.shellFamily) }}</strong>
                </div>
                <div>
                  <span>{{ t('label.platform') }}</span>
                  <strong>{{ finding.platform || t('fallback.unknown') }}</strong>
                </div>
                <div>
                  <span>{{ t('label.time') }}</span>
                  <strong>{{ formatDateTime(finding.timestamp) }}</strong>
                </div>
                <div>
                  <span>{{ t('label.source') }}</span>
                  <strong class="mono">{{ sourceContext(finding) }}</strong>
                </div>
              </div>
            </div>
            <div class="audit-detail-code">
              <div class="audit-code-block">
                <div class="audit-code-label">{{ t('label.command') }}</div>
                <pre>{{ codeBlock(finding.command) }}</pre>
              </div>
              <div class="audit-code-block">
                <div class="audit-code-label">{{ t('label.evidence') }}</div>
                <pre>{{ codeBlock(finding.evidence) }}</pre>
              </div>
            </div>
          </div>
        </section>

        <section class="panel audit-detail-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('panel.session') }}</h2>
              <div class="panel-kicker">{{ t('panel.sessionKicker') }}</div>
            </div>
            <a-button type="primary" :disabled="!finding.sessionId" @click="openSession">
              <template #icon>
                <HistoryOutlined />
              </template>
              {{ t('action.openSession') }}
            </a-button>
          </div>
          <div class="audit-session-grid">
            <div class="audit-session-field">
              <span>{{ t('label.session') }}</span>
              <strong class="mono">{{ sessionName }}</strong>
            </div>
            <div class="audit-session-field">
              <span>{{ t('label.agent') }}</span>
              <strong>{{ linkedSource?.label || t('fallback.unknown') }}</strong>
              <div v-if="linkedSource?.secondary" class="source-identity-meta">{{ linkedSource.secondary }}</div>
            </div>
            <div class="audit-session-field">
              <span>{{ t('label.model') }}</span>
              <strong>{{ sessionDetail?.session.model || t('fallback.unknown') }}</strong>
            </div>
            <div class="audit-session-field">
              <span>{{ t('label.tokens') }}</span>
              <strong>{{ formatNumber(sessionDetail?.session.tokenUsage.totalTokens) }}</strong>
            </div>
            <div class="audit-session-field audit-session-field-wide">
              <span>{{ t('label.project') }}</span>
              <a-tooltip :title="sessionDetail?.session.projectPath || finding.projectPath || finding.rawSourcePath || ''" placement="topLeft">
                <strong>{{ shortPath(sessionDetail?.session.projectPath || finding.projectPath || finding.rawSourcePath || '') }}</strong>
              </a-tooltip>
            </div>
            <div class="audit-session-field audit-session-field-wide">
              <span>{{ t('label.source') }}</span>
              <a-tooltip :title="sessionDetail?.session.sourceSessionsPath || sessionDetail?.session.rawSourcePath || finding.sourceSessionsPath || finding.rawSourcePath || ''" placement="topLeft">
                <strong>{{ shortPath(sessionDetail?.session.sourceSessionsPath || sessionDetail?.session.rawSourcePath || finding.sourceSessionsPath || finding.rawSourcePath || '') }}</strong>
              </a-tooltip>
            </div>
          </div>
        </section>
      </template>
    </div>
  </a-spin>
</template>

<style scoped>
.audit-error {
  margin-bottom: var(--am-section-gap);
}

.audit-detail-actions {
  margin-bottom: 0;
}

.audit-detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.audit-detail-grid {
  display: grid;
  grid-template-columns: minmax(0, 0.9fr) minmax(360px, 1.1fr);
  gap: 18px;
}

.audit-detail-main h3 {
  margin: 0;
  color: var(--am-text);
  font-size: 18px;
}

.audit-detail-main p {
  margin: 8px 0 0;
  color: var(--am-muted);
  line-height: 20px;
}

.audit-detail-fields,
.audit-session-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 16px;
}

.audit-detail-fields div,
.audit-session-field {
  min-width: 0;
  padding: 10px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.audit-detail-fields span,
.audit-session-field span,
.audit-code-label {
  display: block;
  color: var(--am-muted);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}

.audit-detail-fields strong,
.audit-session-field strong {
  display: block;
  min-width: 0;
  margin-top: 4px;
  overflow: hidden;
  color: var(--am-text);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.audit-detail-code {
  display: grid;
  gap: 12px;
}

.audit-code-block pre {
  max-height: 220px;
  margin: 6px 0 0;
  overflow: auto;
  padding: 10px;
  color: var(--am-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 18px;
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--am-surface);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.audit-session-field-wide {
  grid-column: span 2;
}

@media (max-width: 1100px) {
  .audit-detail-grid,
  .audit-detail-fields,
  .audit-session-grid {
    grid-template-columns: 1fr;
  }

  .audit-session-field-wide {
    grid-column: span 1;
  }
}
</style>
