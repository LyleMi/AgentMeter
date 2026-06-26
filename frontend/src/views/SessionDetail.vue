<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import {
  api,
  formatCost,
  formatDateTime,
  formatDuration,
  formatNumber,
  shortPath,
  type EventItem,
  type SessionDetail
} from '../api'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const detail = ref<SessionDetail | null>(null)

const modelColumns = [
  { title: 'Ended', dataIndex: 'endedAt', key: 'endedAt', width: 150 },
  { title: 'Model', dataIndex: 'model', key: 'model' },
  { title: 'Status', dataIndex: 'status', key: 'status', width: 100 },
  { title: 'Duration', dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
  { title: 'Input', dataIndex: 'inputTokens', key: 'input', width: 110, align: 'right' },
  { title: 'Output', dataIndex: 'outputTokens', key: 'output', width: 110, align: 'right' },
  { title: 'Cost', dataIndex: 'costUsd', key: 'cost', width: 120, align: 'right' }
]

const toolColumns = [
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 150 },
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName', width: 160 },
  { title: 'Status', dataIndex: 'status', key: 'status', width: 110 },
  { title: 'Duration', dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
  { title: 'Input', dataIndex: 'inputSummary', key: 'input' },
  { title: 'Output', dataIndex: 'outputSummary', key: 'output' }
]

const rawColumns = [
  { title: 'Line', dataIndex: 'sourceLine', key: 'line', width: 80, align: 'right' },
  { title: 'Time', dataIndex: 'timestamp', key: 'time', width: 150 },
  { title: 'Kind', dataIndex: 'kind', key: 'kind', width: 100 },
  { title: 'Type', dataIndex: 'rawType', key: 'rawType', width: 150 },
  { title: 'Summary', dataIndex: 'summary', key: 'summary' }
]

const events = computed<EventItem[]>(() => detail.value?.events || [])

async function load() {
  loading.value = true
  try {
    detail.value = await api.getSessionDetail(Number(route.params.id))
  } finally {
    loading.value = false
  }
}

function statusColor(status: string) {
  if (status === 'completed' || status === 'ok') return 'green'
  if (status === 'pending' || status === 'warning') return 'orange'
  return 'red'
}

function eventColor(kind: string) {
  if (kind === 'model') return 'blue'
  if (kind === 'tool') return 'purple'
  if (kind === 'error') return 'red'
  return 'default'
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <a-button type="text" @click="router.push('/sessions')">
          <template #icon>
            <ArrowLeftOutlined />
          </template>
          Sessions
        </a-button>
        <h1 class="page-title">
          {{ detail?.session.codexSessionId || 'Session Detail' }}
        </h1>
        <div class="page-subtitle">
          {{ shortPath(detail?.session.projectPath || '') }}
        </div>
      </div>
      <a-button @click="load">Refresh</a-button>
    </div>

    <a-spin :spinning="loading">
      <template v-if="detail">
        <section class="panel session-summary-panel">
          <div class="panel-body session-summary">
            <div class="session-summary-main">
              <div class="metric-label">Session</div>
              <div class="session-summary-id mono">{{ detail.session.codexSessionId }}</div>
              <a-tooltip :title="detail.session.projectPath" placement="topLeft">
                <div class="session-summary-project">{{ shortPath(detail.session.projectPath) }}</div>
              </a-tooltip>
            </div>
            <div class="session-summary-meta">
              <div class="session-summary-item">
                <span class="metric-label">Model</span>
                <strong>{{ detail.session.model }}</strong>
              </div>
              <div class="session-summary-item">
                <span class="metric-label">Started</span>
                <strong>{{ formatDateTime(detail.session.startedAt) }}</strong>
              </div>
              <div class="session-summary-item">
                <span class="metric-label">Ended</span>
                <strong>{{ formatDateTime(detail.session.endedAt) }}</strong>
              </div>
            </div>
          </div>
        </section>

        <div class="metric-grid session-metric-grid">
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-label">Tokens</div>
            <div class="metric-value">{{ formatNumber(detail.session.tokenUsage.totalTokens) }}</div>
            <div class="metric-note">{{ detail.session.tokenUsage.source }}</div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-label">Estimated Cost</div>
            <div class="metric-value">{{ formatCost(detail.session.estimatedCostUsd) }}</div>
            <div class="metric-note">{{ detail.session.model }}</div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-label">Timing</div>
            <div class="metric-value">{{ formatDuration(detail.session.wallDurationMs) }}</div>
            <div class="metric-note">{{ formatDuration(detail.session.activeDurationMs) }} active</div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-label">Tools</div>
            <div class="metric-value">{{ formatNumber(detail.session.toolCallCount) }}</div>
            <div class="metric-note">{{ formatNumber(detail.session.eventCount) }} events</div>
          </a-card>
        </div>

        <section class="panel session-timeline-panel">
          <div class="panel-header">
            <h2 class="panel-title">Timeline</h2>
            <span class="muted">{{ formatNumber(events.length) }} events</span>
          </div>
          <div class="panel-body timeline-list session-timeline-list">
            <a-timeline>
              <a-timeline-item v-for="event in events" :key="event.id">
                <div class="timeline-event">
                  <div class="timeline-event-header">
                    <a-tag class="status-tag event-kind-tag" :color="eventColor(event.kind)">{{ event.kind }}</a-tag>
                    <span class="timeline-event-time">{{ formatDateTime(event.timestamp) }}</span>
                  </div>
                  <div class="timeline-event-summary">{{ event.summary }}</div>
                  <div class="muted mono timeline-event-raw">line {{ event.sourceLine }} · {{ event.rawType }}</div>
                </div>
              </a-timeline-item>
            </a-timeline>
          </div>
        </section>

        <section class="panel session-calls-panel" style="margin-top: 18px">
          <div class="panel-header">
            <h2 class="panel-title">Calls</h2>
          </div>
          <a-tabs class="panel-body calls-tabs" :tab-bar-style="{ marginBottom: '14px' }">
            <a-tab-pane key="model" tab="Model">
              <a-table
                class="calls-table model-calls-table"
                size="small"
                :columns="modelColumns"
                :data-source="detail.modelCalls"
                :pagination="{ pageSize: 8 }"
                row-key="id"
              >
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'endedAt'">{{ formatDateTime(record.endedAt) }}</template>
                  <template v-else-if="column.key === 'status'">
                    <a-tag class="status-tag call-status-tag" :color="statusColor(record.status)">{{ record.status }}</a-tag>
                  </template>
                  <template v-else-if="column.key === 'duration'">
                    <span class="number-cell">{{ formatDuration(record.durationMs) }}</span>
                  </template>
                  <template v-else-if="column.key === 'input'">
                    <span class="number-cell">{{ formatNumber(record.inputTokens) }}</span>
                  </template>
                  <template v-else-if="column.key === 'output'">
                    <span class="number-cell">{{ formatNumber(record.outputTokens) }}</span>
                  </template>
                  <template v-else-if="column.key === 'cost'">
                    <span class="number-cell">{{ formatCost(record.costUsd) }}</span>
                  </template>
                </template>
              </a-table>
            </a-tab-pane>
            <a-tab-pane key="tools" tab="Tools">
              <a-table
                class="calls-table tool-calls-table"
                size="small"
                :columns="toolColumns"
                :data-source="detail.toolCalls"
                :pagination="{ pageSize: 8 }"
                row-key="id"
              >
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'startedAt'">{{ formatDateTime(record.startedAt) }}</template>
                  <template v-else-if="column.key === 'status'">
                    <a-tag class="status-tag call-status-tag" :color="statusColor(record.status)">{{ record.status }}</a-tag>
                  </template>
                  <template v-else-if="column.key === 'duration'">
                    <span class="number-cell">{{ formatDuration(record.durationMs) }}</span>
                  </template>
                  <template v-else-if="column.key === 'input'">
                    <a-typography-text :ellipsis="{ tooltip: record.inputSummary }">
                      {{ record.inputSummary || '-' }}
                    </a-typography-text>
                  </template>
                  <template v-else-if="column.key === 'output'">
                    <a-typography-text :ellipsis="{ tooltip: record.outputSummary || record.error }">
                      {{ record.outputSummary || record.error || '-' }}
                    </a-typography-text>
                  </template>
                </template>
              </a-table>
            </a-tab-pane>
          </a-tabs>
        </section>

        <section class="panel session-metadata-panel" style="margin-top: 18px">
          <div class="panel-header">
            <h2 class="panel-title">Metadata</h2>
            <span class="muted">Session source and indexing context</span>
          </div>
          <div class="panel-body metadata-grid">
            <div class="metadata-item">
              <div class="metadata-label">Started</div>
              <div class="metadata-value">{{ formatDateTime(detail.session.startedAt) }}</div>
            </div>
            <div class="metadata-item">
              <div class="metadata-label">Ended</div>
              <div class="metadata-value">{{ formatDateTime(detail.session.endedAt) }}</div>
            </div>
            <div class="metadata-item">
              <div class="metadata-label">Model</div>
              <div class="metadata-value">{{ detail.session.model }}</div>
            </div>
            <div class="metadata-item">
              <div class="metadata-label">Provider</div>
              <div class="metadata-value">{{ detail.session.modelProvider || '-' }}</div>
            </div>
            <div class="metadata-item is-wide">
              <div class="metadata-label">Project</div>
              <a-typography-text class="metadata-value detail-path" :ellipsis="{ tooltip: detail.session.projectPath }">
                {{ detail.session.projectPath }}
              </a-typography-text>
            </div>
            <div class="metadata-item is-wide">
              <div class="metadata-label">Raw source</div>
              <a-typography-text class="metadata-value detail-path mono" :ellipsis="{ tooltip: detail.session.rawSourcePath }">
                {{ detail.session.rawSourcePath }}
              </a-typography-text>
            </div>
          </div>
        </section>

        <section class="panel raw-events-panel" style="margin-top: 18px">
          <div class="panel-header">
            <h2 class="panel-title">Raw Events</h2>
          </div>
          <a-table
            class="raw-events-table"
            size="small"
            :columns="rawColumns"
            :data-source="events"
            :pagination="{ pageSize: 10 }"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'line'">
                <span class="number-cell">{{ formatNumber(record.sourceLine) }}</span>
              </template>
              <template v-else-if="column.key === 'time'">{{ formatDateTime(record.timestamp) }}</template>
              <template v-else-if="column.key === 'summary'">
                <a-typography-text :ellipsis="{ tooltip: record.summary }">
                  {{ record.summary }}
                </a-typography-text>
              </template>
            </template>
          </a-table>
        </section>
      </template>
    </a-spin>
  </div>
</template>
