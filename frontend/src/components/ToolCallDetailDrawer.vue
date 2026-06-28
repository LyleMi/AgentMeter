<script setup lang="ts">
import { computed } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ADrawer from 'ant-design-vue/es/drawer'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ArrowRightOutlined } from '@ant-design/icons-vue'
import { formatDateTime, formatDuration, formatNumber, projectDisplay, sessionFullLabel, sessionLabel, shortPath, type ToolCall } from '../api'
import { useMessages } from '../i18n'
import { sourceDisplay } from '../presentation/sourceIdentity'
import { statusClass, statusColor } from '../presentation/status'
import { parseToolInput } from '../toolInput'

const ATypographyParagraph = Typography.Paragraph
const ATypographyText = Typography.Text

const props = withDefaults(
  defineProps<{
    open: boolean
    call: ToolCall | null
    showSessionLink?: boolean
  }>(),
  { showSessionLink: true }
)

const emit = defineEmits<{
  close: []
  openSession: [id: number]
}>()

const { t } = useMessages({
  en: {
    'title.default': 'Tool Call Details',
    'title.named': '{tool} Details',
    'label.tool': 'Tool',
    'label.started': 'Started',
    'label.ended': 'Ended',
    'label.duration': 'Duration',
    'label.session': 'Session',
    'label.agent': 'Source',
    'label.rawEvents': 'Raw Events',
    'label.project': 'Project',
    'label.rawSource': 'Raw Source',
    'label.input': 'Input',
    'label.output': 'Output',
    'label.error': 'Error',
    'label.rawEvent': 'Raw Event',
    'action.session': 'Session',
    'raw.start': 'Start raw event',
    'raw.end': 'End raw event',
    'raw.line': 'line',
    'raw.none': 'No raw event recorded',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    'title.default': '工具调用详情',
    'title.named': '{tool} 详情',
    'label.tool': '工具',
    'label.started': '开始',
    'label.ended': '结束',
    'label.duration': '耗时',
    'label.session': '会话',
    'label.agent': '来源',
    'label.rawEvents': '原始事件',
    'label.project': '项目',
    'label.rawSource': '原始来源',
    'label.input': '输入',
    'label.output': '输出',
    'label.error': '错误',
    'label.rawEvent': '原始事件',
    'action.session': '会话',
    'raw.start': '开始原始事件',
    'raw.end': '结束原始事件',
    'raw.line': '行',
    'raw.none': '没有记录原始事件',
    'fallback.unknown': '未知'
  }
})

const drawerTitle = computed(() => {
  if (!props.call?.toolName) return t('title.default')
  return t('title.named', { tool: props.call.toolName })
})

const parsedInput = computed(() => parseToolInput(props.call))
const callSource = computed(() => (props.call ? sourceDisplay(props.call, t('fallback.unknown')) : null))

function sessionName(call: ToolCall) {
  return sessionLabel({ id: call.sessionId, sessionKey: call.sessionKey || '', codexSessionId: call.codexSessionId })
}

function sessionFullName(call: ToolCall) {
  return sessionFullLabel({ id: call.sessionId, sessionKey: call.sessionKey || '', codexSessionId: call.codexSessionId })
}

function hasText(value?: string) {
  return Boolean(value && value.trim())
}

function formatLine(value?: number) {
  return value ? formatNumber(value) : '-'
}

function openSession(call: ToolCall) {
  emit('openSession', call.sessionId)
}

function hasDistinctEndRaw(call: ToolCall) {
  return hasText(call.rawEndEventJson) && call.rawEndEventJson !== call.rawStartEventJson
}

function projectName(call: ToolCall) {
  return call.projectPath ? projectDisplay(call.projectPath).main : '-'
}
</script>

<template>
  <a-drawer class="tool-call-drawer" :open="props.open" :width="'min(720px, 100vw)'" placement="right" @close="emit('close')">
    <template #title>{{ drawerTitle }}</template>

    <template v-if="props.call">
      <div class="tool-detail-summary">
        <div class="tool-detail-heading">
          <div class="metric-label">{{ t('label.tool') }}</div>
          <div class="summary-title">{{ props.call.toolName || t('fallback.unknown') }}</div>
          <div class="summary-meta">
            <a-tag class="status-tag call-status-tag" :class="statusClass(props.call.status)" :color="statusColor(props.call.status)">
              {{ props.call.status || t('fallback.unknown') }}
            </a-tag>
            <span class="summary-chip mono">#{{ formatNumber(props.call.id) }}</span>
            <span v-if="props.call.callId" class="summary-chip mono">{{ props.call.callId }}</span>
          </div>
        </div>
        <a-button v-if="props.showSessionLink && props.call.sessionId" @click="openSession(props.call)">
          <template #icon>
            <ArrowRightOutlined />
          </template>
          {{ t('action.session') }}
        </a-button>
      </div>

      <div class="metadata-grid tool-detail-grid">
        <div class="metadata-item">
          <div class="metadata-label">{{ t('label.started') }}</div>
          <div class="metadata-value">{{ formatDateTime(props.call.startedAt) }}</div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">{{ t('label.ended') }}</div>
          <div class="metadata-value">{{ formatDateTime(props.call.endedAt) }}</div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">{{ t('label.duration') }}</div>
          <div class="metadata-value number-cell">{{ formatDuration(props.call.durationMs) }}</div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">{{ t('label.session') }}</div>
          <a-typography-text class="metadata-value mono" :ellipsis="{ tooltip: sessionFullName(props.call) }">
            {{ sessionName(props.call) }}
          </a-typography-text>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">{{ t('label.agent') }}</div>
          <div class="metadata-value">
            {{ callSource?.label || '-' }}
            <div v-if="callSource?.secondary" class="source-identity-meta">{{ callSource.secondary }}</div>
          </div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">{{ t('label.rawEvents') }}</div>
          <div class="metadata-value mono">
            {{ formatLine(props.call.rawStartEventLine || props.call.rawEventLine) }} -> {{ formatLine(props.call.rawEndEventLine) }}
          </div>
        </div>
        <div class="metadata-item is-wide">
          <div class="metadata-label">{{ t('label.project') }}</div>
          <a-typography-text class="metadata-value detail-path" :ellipsis="{ tooltip: props.call.projectPath }">
            {{ projectName(props.call) }}
          </a-typography-text>
        </div>
        <div class="metadata-item is-wide">
          <div class="metadata-label">{{ t('label.rawSource') }}</div>
          <a-typography-text class="metadata-value detail-path mono" :ellipsis="{ tooltip: props.call.rawSourcePath }">
            {{ props.call.rawSourcePath ? shortPath(props.call.rawSourcePath) : '-' }}
          </a-typography-text>
        </div>
      </div>

      <section class="detail-section">
        <div class="metadata-label">{{ t('label.input') }}</div>
        <div v-if="parsedInput.isStructured" class="tool-input-detail-grid">
          <div v-for="field in parsedInput.fields" :key="field.key" class="tool-input-detail-field" :class="{ 'is-wide': field.isLong }">
            <div class="tool-input-detail-label">{{ field.label }}</div>
            <a-typography-paragraph class="tool-input-detail-value mono" copyable>
              {{ field.value || '-' }}
            </a-typography-paragraph>
          </div>
        </div>
        <a-typography-paragraph v-else class="detail-pre mono" copyable>
          {{ parsedInput.rawText || '-' }}
        </a-typography-paragraph>
      </section>

      <section class="detail-section">
        <div class="metadata-label">{{ t('label.output') }}</div>
        <a-typography-paragraph class="detail-pre mono" copyable>
          {{ props.call.outputSummary || '-' }}
        </a-typography-paragraph>
      </section>

      <section v-if="props.call.error" class="detail-section">
        <div class="metadata-label">{{ t('label.error') }}</div>
        <a-typography-paragraph class="detail-pre detail-pre-error mono" copyable>
          {{ props.call.error }}
        </a-typography-paragraph>
      </section>

      <details v-if="hasText(props.call.rawStartEventJson)" class="raw-detail" open>
        <summary>
          {{ t('raw.start') }}
          <span class="muted mono">{{ t('raw.line') }} {{ formatLine(props.call.rawStartEventLine || props.call.rawEventLine) }} · {{ props.call.rawStartEventType || '-' }}</span>
        </summary>
        <div v-if="props.call.rawStartEventSummary" class="raw-detail-summary">{{ props.call.rawStartEventSummary }}</div>
        <a-typography-paragraph class="detail-pre raw-json mono" copyable>
          {{ props.call.rawStartEventJson }}
        </a-typography-paragraph>
      </details>

      <details v-if="hasDistinctEndRaw(props.call)" class="raw-detail">
        <summary>
          {{ t('raw.end') }}
          <span class="muted mono">{{ t('raw.line') }} {{ formatLine(props.call.rawEndEventLine) }} · {{ props.call.rawEndEventType || '-' }}</span>
        </summary>
        <div v-if="props.call.rawEndEventSummary" class="raw-detail-summary">{{ props.call.rawEndEventSummary }}</div>
        <a-typography-paragraph class="detail-pre raw-json mono" copyable>
          {{ props.call.rawEndEventJson }}
        </a-typography-paragraph>
      </details>

      <div v-if="!hasText(props.call.rawStartEventJson) && !hasText(props.call.rawEndEventJson)" class="metadata-item">
        <div class="metadata-label">{{ t('label.rawEvent') }}</div>
        <div class="metadata-value">{{ t('raw.none') }}</div>
      </div>
    </template>
  </a-drawer>
</template>
