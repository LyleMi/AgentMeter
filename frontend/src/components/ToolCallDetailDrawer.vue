<script setup lang="ts">
import { computed } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ADrawer from 'ant-design-vue/es/drawer'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ArrowRightOutlined } from '@ant-design/icons-vue'
import { formatDateTime, formatDuration, formatNumber, shortPath, type ToolCall } from '../api'

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

const drawerTitle = computed(() => {
  if (!props.call?.toolName) return 'Tool Call Details'
  return `${props.call.toolName} Details`
})

function normalizedStatus(status?: string) {
  return (status || 'unknown').toLowerCase()
}

function statusClass(status?: string) {
  const normalized = normalizedStatus(status)
  if (['completed', 'ok', 'indexed', 'success'].includes(normalized)) return 'status-ok'
  if (['pending', 'warning', 'scanning', 'unknown', 'started'].includes(normalized)) return 'status-warning'
  return 'status-error'
}

function statusColor(status?: string) {
  const normalized = normalizedStatus(status)
  if (['completed', 'ok', 'indexed', 'success'].includes(normalized)) return 'success'
  if (normalized === 'scanning') return 'processing'
  if (['pending', 'warning', 'unknown', 'started'].includes(normalized)) return 'warning'
  return 'error'
}

function sessionName(call: ToolCall) {
  return call.sessionKey || call.codexSessionId || `#${call.sessionId}`
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
</script>

<template>
  <a-drawer class="tool-call-drawer" :open="props.open" :width="720" placement="right" @close="emit('close')">
    <template #title>{{ drawerTitle }}</template>

    <template v-if="props.call">
      <div class="tool-detail-summary">
        <div class="tool-detail-heading">
          <div class="metric-label">Tool</div>
          <div class="summary-title">{{ props.call.toolName || 'unknown' }}</div>
          <div class="summary-meta">
            <a-tag class="status-tag call-status-tag" :class="statusClass(props.call.status)" :color="statusColor(props.call.status)">
              {{ props.call.status || 'unknown' }}
            </a-tag>
            <span class="summary-chip mono">#{{ formatNumber(props.call.id) }}</span>
            <span v-if="props.call.callId" class="summary-chip mono">{{ props.call.callId }}</span>
          </div>
        </div>
        <a-button v-if="props.showSessionLink && props.call.sessionId" @click="openSession(props.call)">
          <template #icon>
            <ArrowRightOutlined />
          </template>
          Session
        </a-button>
      </div>

      <div class="metadata-grid tool-detail-grid">
        <div class="metadata-item">
          <div class="metadata-label">Started</div>
          <div class="metadata-value">{{ formatDateTime(props.call.startedAt) }}</div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">Ended</div>
          <div class="metadata-value">{{ formatDateTime(props.call.endedAt) }}</div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">Duration</div>
          <div class="metadata-value number-cell">{{ formatDuration(props.call.durationMs) }}</div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">Session</div>
          <div class="metadata-value mono">{{ sessionName(props.call) }}</div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">Agent</div>
          <div class="metadata-value">{{ props.call.agentName || props.call.agentKind || '-' }}</div>
        </div>
        <div class="metadata-item">
          <div class="metadata-label">Raw Events</div>
          <div class="metadata-value mono">
            {{ formatLine(props.call.rawStartEventLine || props.call.rawEventLine) }} -> {{ formatLine(props.call.rawEndEventLine) }}
          </div>
        </div>
        <div class="metadata-item is-wide">
          <div class="metadata-label">Project</div>
          <a-typography-text class="metadata-value detail-path" :ellipsis="{ tooltip: props.call.projectPath }">
            {{ props.call.projectPath || '-' }}
          </a-typography-text>
        </div>
        <div class="metadata-item is-wide">
          <div class="metadata-label">Raw Source</div>
          <a-typography-text class="metadata-value detail-path mono" :ellipsis="{ tooltip: props.call.rawSourcePath }">
            {{ props.call.rawSourcePath ? shortPath(props.call.rawSourcePath) : '-' }}
          </a-typography-text>
        </div>
      </div>

      <section class="detail-section">
        <div class="metadata-label">Input</div>
        <a-typography-paragraph class="detail-pre mono" copyable>
          {{ props.call.inputSummary || '-' }}
        </a-typography-paragraph>
      </section>

      <section class="detail-section">
        <div class="metadata-label">Output</div>
        <a-typography-paragraph class="detail-pre mono" copyable>
          {{ props.call.outputSummary || '-' }}
        </a-typography-paragraph>
      </section>

      <section v-if="props.call.error" class="detail-section">
        <div class="metadata-label">Error</div>
        <a-typography-paragraph class="detail-pre detail-pre-error mono" copyable>
          {{ props.call.error }}
        </a-typography-paragraph>
      </section>

      <details v-if="hasText(props.call.rawStartEventJson)" class="raw-detail" open>
        <summary>
          Start raw event
          <span class="muted mono">line {{ formatLine(props.call.rawStartEventLine || props.call.rawEventLine) }} · {{ props.call.rawStartEventType || '-' }}</span>
        </summary>
        <div v-if="props.call.rawStartEventSummary" class="raw-detail-summary">{{ props.call.rawStartEventSummary }}</div>
        <a-typography-paragraph class="detail-pre raw-json mono" copyable>
          {{ props.call.rawStartEventJson }}
        </a-typography-paragraph>
      </details>

      <details v-if="hasDistinctEndRaw(props.call)" class="raw-detail">
        <summary>
          End raw event
          <span class="muted mono">line {{ formatLine(props.call.rawEndEventLine) }} · {{ props.call.rawEndEventType || '-' }}</span>
        </summary>
        <div v-if="props.call.rawEndEventSummary" class="raw-detail-summary">{{ props.call.rawEndEventSummary }}</div>
        <a-typography-paragraph class="detail-pre raw-json mono" copyable>
          {{ props.call.rawEndEventJson }}
        </a-typography-paragraph>
      </details>

      <div v-if="!hasText(props.call.rawStartEventJson) && !hasText(props.call.rawEndEventJson)" class="metadata-item">
        <div class="metadata-label">Raw Event</div>
        <div class="metadata-value">No raw event recorded</div>
      </div>
    </template>
  </a-drawer>
</template>
