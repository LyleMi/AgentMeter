<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import ACard from 'ant-design-vue/es/card'
import AEmpty from 'ant-design-vue/es/empty'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATabs from 'ant-design-vue/es/tabs'
import ATag from 'ant-design-vue/es/tag'
import ATimeline from 'ant-design-vue/es/timeline'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import {
  ArrowLeftOutlined,
  ClockCircleOutlined,
  DollarCircleOutlined,
  EyeOutlined,
  FunctionOutlined,
  ReloadOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import ToolCallDetailDrawer from '../components/ToolCallDetailDrawer.vue'
import {
  api,
  formatCost,
  formatDateTime,
  formatDuration,
  formatNumber,
  sessionLabel,
  shortPath,
  type EventItem,
  type SessionDetail,
  type ToolCall
} from '../api'

const ATabPane = ATabs.TabPane
const ATable = AntTable as unknown as DefineComponent
const ATimelineItem = ATimeline.Item
const ATypographyParagraph = Typography.Paragraph
const ATypographyText = Typography.Text

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const detail = ref<SessionDetail | null>(null)
const selectedToolCall = ref<ToolCall | null>(null)

const modelColumns = [
  { title: 'Ended', dataIndex: 'endedAt', key: 'endedAt', width: 150 },
  { title: 'Model', dataIndex: 'model', key: 'model', width: 220 },
  { title: 'Status', dataIndex: 'status', key: 'status', width: 130 },
  { title: 'Duration', dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
  { title: 'Input', dataIndex: 'inputTokens', key: 'input', width: 100, align: 'right' },
  { title: 'Cached', dataIndex: 'cachedInputTokens', key: 'cached', width: 100, align: 'right' },
  { title: 'Output', dataIndex: 'outputTokens', key: 'output', width: 100, align: 'right' },
  { title: 'Reasoning', dataIndex: 'reasoningOutputTokens', key: 'reasoning', width: 110, align: 'right' },
  { title: 'Total', dataIndex: 'totalTokens', key: 'total', width: 110, align: 'right' },
  { title: 'Cost', dataIndex: 'costUsd', key: 'cost', width: 120, align: 'right' }
]

const toolColumns = [
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 150 },
  { title: 'Ended', dataIndex: 'endedAt', key: 'endedAt', width: 150 },
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName', width: 160 },
  { title: 'Status', dataIndex: 'status', key: 'status', width: 110 },
  { title: 'Duration', dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
  { title: 'Raw Event', dataIndex: 'rawEventId', key: 'rawEvent', width: 100, align: 'right' },
  { title: 'Input', dataIndex: 'inputSummary', key: 'input' },
  { title: 'Output', dataIndex: 'outputSummary', key: 'output' },
  { title: '', key: 'detail', width: 56, align: 'right' }
]

const rawColumns = [
  { title: 'Line', dataIndex: 'sourceLine', key: 'line', width: 80, align: 'right' },
  { title: 'Time', dataIndex: 'timestamp', key: 'time', width: 150 },
  { title: 'Kind', dataIndex: 'kind', key: 'kind', width: 100 },
  { title: 'Type', dataIndex: 'rawType', key: 'rawType', width: 150 },
  { title: 'Summary', dataIndex: 'summary', key: 'summary' }
]

const events = computed<EventItem[]>(() => detail.value?.events || [])
const rawEventsExpandable = {
  rowExpandable: (record: EventItem) => Boolean(record.rawJson)
}

async function load() {
  loading.value = true
  try {
    detail.value = await api.getSessionDetail(Number(route.params.id))
  } finally {
    loading.value = false
  }
}

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

function eventColor(kind: string) {
  if (kind === 'model') return 'blue'
  if (kind === 'tool') return 'purple'
  if (kind === 'error') return 'red'
  return 'default'
}

function indexStatusHint(session: SessionDetail['session']) {
  return session.lastIndexedScanMessage || session.rawSourcePath || 'No index message recorded'
}

function goBack() {
  router.push('/sessions')
}

function openToolCall(call: ToolCall) {
  selectedToolCall.value = call
}

function closeToolCall() {
  selectedToolCall.value = null
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <a-button type="text" @click="goBack">
          <template #icon>
            <ArrowLeftOutlined />
          </template>
          Sessions
        </a-button>
        <h1 class="page-title">Session Trace</h1>
        <div class="page-subtitle">
          {{ detail ? shortPath(detail.session.projectPath) : 'Timeline, calls, metadata, and raw local events' }}
        </div>
      </div>
      <a-button @click="load">
        <template #icon>
          <ReloadOutlined />
        </template>
        Refresh
      </a-button>
    </div>

    <a-spin :spinning="loading">
      <template v-if="detail">
        <section class="summary-panel session-summary-panel">
          <div class="session-summary-main">
            <div class="metric-label">Trace</div>
            <div class="summary-title mono">{{ sessionLabel(detail.session) }}</div>
            <a-tooltip :title="detail.session.projectPath" placement="topLeft">
              <div class="session-summary-project">{{ shortPath(detail.session.projectPath) }}</div>
            </a-tooltip>
            <div class="summary-meta">
              <a-tag class="status-tag parse-status-tag" :class="statusClass(detail.session.parseStatus)" :color="statusColor(detail.session.parseStatus)">
                parse {{ detail.session.parseStatus || 'unknown' }}
              </a-tag>
              <a-tooltip :title="indexStatusHint(detail.session)" placement="topLeft">
                <a-tag
                  class="status-tag parse-status-tag"
                  :class="statusClass(detail.session.lastIndexedScanStatus)"
                  :color="statusColor(detail.session.lastIndexedScanStatus)"
                >
                  {{ detail.session.lastIndexedScanStatus || 'unknown' }}
                </a-tag>
              </a-tooltip>
              <a-tag v-if="detail.session.unpriced" class="status-tag model-status-tag" color="warning">unpriced</a-tag>
            </div>
          </div>
          <div class="session-summary-meta">
            <div class="session-summary-item">
              <span class="metric-label">Agent</span>
              <strong>{{ detail.session.agentName || detail.session.agentKind || 'unknown' }}</strong>
            </div>
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
        </section>

        <div class="metric-grid session-metric-grid">
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-card-topline">
              <div class="metric-label">Tokens</div>
              <FunctionOutlined class="metric-icon" />
            </div>
            <div class="metric-value">{{ formatNumber(detail.session.tokenUsage.totalTokens) }}</div>
            <div class="metric-note">
              {{ formatNumber(detail.session.tokenUsage.inputTokens) }} in ·
              {{ formatNumber(detail.session.tokenUsage.outputTokens) }} out ·
              {{ formatNumber(detail.session.tokenUsage.cachedInputTokens) }} cached ·
              {{ detail.session.tokenUsage.source }}
            </div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-card-topline">
              <div class="metric-label">Estimated Cost</div>
              <DollarCircleOutlined class="metric-icon" />
            </div>
            <div class="metric-value">{{ formatCost(detail.session.estimatedCostUsd) }}</div>
            <div class="metric-note" :class="{ 'metric-note-warning': detail.session.unpriced }">
              {{ detail.session.unpriced ? 'Missing local pricing for this model' : detail.session.model }}
            </div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-card-topline">
              <div class="metric-label">Timing</div>
              <ClockCircleOutlined class="metric-icon" />
            </div>
            <div class="metric-value">{{ formatDuration(detail.session.wallDurationMs) }}</div>
            <div class="metric-note">
              {{ formatDuration(detail.session.activeDurationMs) }} active ·
              {{ formatDuration(detail.session.idleDurationMs) }} idle
            </div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-card-topline">
              <div class="metric-label">Calls / Events</div>
              <ToolOutlined class="metric-icon" />
            </div>
            <div class="metric-value">{{ formatNumber(detail.session.toolCallCount) }}</div>
            <div class="metric-note">
              {{ formatNumber(detail.modelCalls.length) }} model ·
              {{ formatNumber(detail.session.eventCount) }} events
            </div>
          </a-card>
        </div>

        <div class="section-stack">
          <section class="panel session-timeline-panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">Timeline</h2>
                <div class="panel-kicker">Primary inspection surface ordered by local event time</div>
              </div>
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

          <section class="panel session-calls-panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">Calls</h2>
                <div class="panel-kicker">Model and tool invocations with aligned usage and duration</div>
              </div>
            </div>
            <a-tabs class="panel-body calls-tabs">
              <a-tab-pane key="model" :tab="'Model (' + formatNumber(detail.modelCalls.length) + ')'">
                <a-table
                  class="calls-table model-calls-table"
                  size="small"
                  :columns="modelColumns"
                  :data-source="detail.modelCalls"
                  :locale="{ emptyText: 'No model calls captured for this session' }"
                  :pagination="{ pageSize: 8 }"
                  :scroll="{ x: 1200 }"
                  row-key="id"
                >
                  <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'endedAt'">{{ formatDateTime(record.endedAt) }}</template>
                    <template v-else-if="column.key === 'model'">
                      <a-typography-text class="model-name" :ellipsis="{ tooltip: record.model }">
                        {{ record.model || 'unknown' }}
                      </a-typography-text>
                      <div class="timeline-event-raw">{{ record.provider || '-' }}</div>
                    </template>
                    <template v-else-if="column.key === 'status'">
                      <div class="timeline-event-head">
                        <a-tag class="status-tag call-status-tag" :class="statusClass(record.status)" :color="statusColor(record.status)">
                          {{ record.status || 'unknown' }}
                        </a-tag>
                        <a-tag v-if="record.unpriced" class="status-tag model-status-tag" color="warning">unpriced</a-tag>
                      </div>
                    </template>
                    <template v-else-if="column.key === 'duration'">
                      <span class="number-cell">{{ formatDuration(record.durationMs) }}</span>
                    </template>
                    <template v-else-if="column.key === 'input'">
                      <span class="number-cell">{{ formatNumber(record.inputTokens) }}</span>
                    </template>
                    <template v-else-if="column.key === 'cached'">
                      <span class="number-cell">{{ formatNumber(record.cachedInputTokens) }}</span>
                    </template>
                    <template v-else-if="column.key === 'output'">
                      <span class="number-cell">{{ formatNumber(record.outputTokens) }}</span>
                    </template>
                    <template v-else-if="column.key === 'reasoning'">
                      <span class="number-cell">{{ formatNumber(record.reasoningOutputTokens) }}</span>
                    </template>
                    <template v-else-if="column.key === 'total'">
                      <span class="number-cell">{{ formatNumber(record.totalTokens) }}</span>
                    </template>
                    <template v-else-if="column.key === 'cost'">
                      <span class="number-cell">{{ formatCost(record.costUsd) }}</span>
                    </template>
                  </template>
                </a-table>
              </a-tab-pane>
              <a-tab-pane key="tools" :tab="'Tools (' + formatNumber(detail.toolCalls.length) + ')'">
                <a-table
                  class="calls-table tool-calls-table"
                  size="small"
                  :columns="toolColumns"
                  :data-source="detail.toolCalls"
                  :locale="{ emptyText: 'No tool calls captured for this session' }"
                  :pagination="{ pageSize: 8 }"
                  :scroll="{ x: 1180 }"
                  row-key="id"
                >
                  <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'startedAt'">{{ formatDateTime(record.startedAt) }}</template>
                    <template v-else-if="column.key === 'endedAt'">{{ formatDateTime(record.endedAt) }}</template>
                    <template v-else-if="column.key === 'status'">
                      <a-tooltip :title="record.error || record.status || 'unknown'">
                        <a-tag class="status-tag call-status-tag" :class="statusClass(record.status)" :color="statusColor(record.status)">
                          {{ record.status || 'unknown' }}
                        </a-tag>
                      </a-tooltip>
                    </template>
                    <template v-else-if="column.key === 'duration'">
                      <span class="number-cell">{{ formatDuration(record.durationMs) }}</span>
                    </template>
                    <template v-else-if="column.key === 'rawEvent'">
                      <span class="number-cell">{{ formatNumber(record.rawStartEventId || record.rawEventId) }}</span>
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
                    <template v-else-if="column.key === 'detail'">
                      <a-tooltip title="View details">
                        <a-button type="text" size="small" @click="openToolCall(record)">
                          <template #icon>
                            <EyeOutlined />
                          </template>
                        </a-button>
                      </a-tooltip>
                    </template>
                  </template>
                </a-table>
              </a-tab-pane>
            </a-tabs>
          </section>

          <section class="panel session-metadata-panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">Metadata</h2>
                <div class="panel-kicker">Session source, timing breakdown, parser, and index context</div>
              </div>
            </div>
            <div class="panel-body metadata-grid">
              <div class="metadata-item">
                <div class="metadata-label">Session row</div>
                <div class="metadata-value number-cell">{{ formatNumber(detail.session.id) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Session key</div>
                <div class="metadata-value mono">{{ sessionLabel(detail.session) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Agent</div>
                <div class="metadata-value">{{ detail.session.agentName || detail.session.agentKind || '-' }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Agent kind</div>
                <div class="metadata-value">{{ detail.session.agentKind || '-' }}</div>
              </div>
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
              <div class="metadata-item">
                <div class="metadata-label">Originator</div>
                <div class="metadata-value">{{ detail.session.originator || '-' }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Thread source</div>
                <div class="metadata-value">{{ detail.session.threadSource || '-' }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Parse status</div>
                <a-tag class="status-tag parse-status-tag" :class="statusClass(detail.session.parseStatus)" :color="statusColor(detail.session.parseStatus)">
                  {{ detail.session.parseStatus || 'unknown' }}
                </a-tag>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Index status</div>
                <a-tag
                  class="status-tag parse-status-tag"
                  :class="statusClass(detail.session.lastIndexedScanStatus)"
                  :color="statusColor(detail.session.lastIndexedScanStatus)"
                >
                  {{ detail.session.lastIndexedScanStatus || 'unknown' }}
                </a-tag>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Usage source</div>
                <div class="metadata-value">{{ detail.session.tokenUsage.source }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Pricing</div>
                <div class="metadata-value">{{ detail.session.unpriced ? 'unpriced' : 'priced' }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Wall</div>
                <div class="metadata-value number-cell">{{ formatDuration(detail.session.wallDurationMs) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Active</div>
                <div class="metadata-value number-cell">{{ formatDuration(detail.session.activeDurationMs) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Model time</div>
                <div class="metadata-value number-cell">{{ formatDuration(detail.session.modelDurationMs) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Tool time</div>
                <div class="metadata-value number-cell">{{ formatDuration(detail.session.toolDurationMs) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Idle</div>
                <div class="metadata-value number-cell">{{ formatDuration(detail.session.idleDurationMs) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Events</div>
                <div class="metadata-value number-cell">{{ formatNumber(detail.session.eventCount) }}</div>
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
              <div v-if="detail.session.lastIndexedScanMessage" class="metadata-item is-wide">
                <div class="metadata-label">Index message</div>
                <div class="metadata-value">{{ detail.session.lastIndexedScanMessage }}</div>
              </div>
            </div>
          </section>

          <section class="panel raw-events-panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">Raw Events</h2>
                <div class="panel-kicker">Source lines, raw types, and event summaries</div>
              </div>
            </div>
            <a-table
              class="raw-events-table"
              size="small"
              :columns="rawColumns"
              :data-source="events"
              :expandable="rawEventsExpandable"
              :pagination="{ pageSize: 10 }"
              row-key="id"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'line'">
                  <span class="number-cell">{{ formatNumber(record.sourceLine) }}</span>
                </template>
                <template v-else-if="column.key === 'time'">{{ formatDateTime(record.timestamp) }}</template>
                <template v-else-if="column.key === 'kind'">
                  <a-tag class="status-tag event-kind-tag" :color="eventColor(record.kind)">{{ record.kind }}</a-tag>
                </template>
                <template v-else-if="column.key === 'rawType'">
                  <a-typography-text :ellipsis="{ tooltip: record.rawType }">
                    {{ record.rawType || '-' }}
                  </a-typography-text>
                </template>
                <template v-else-if="column.key === 'summary'">
                  <a-typography-text :ellipsis="{ tooltip: record.summary }">
                    {{ record.summary }}
                  </a-typography-text>
                </template>
              </template>
              <template #expandedRowRender="{ record }">
                <a-typography-paragraph class="metadata-value mono" copyable>
                  {{ record.rawJson || 'No raw JSON recorded' }}
                </a-typography-paragraph>
              </template>
            </a-table>
          </section>
        </div>
      </template>
      <a-empty v-else-if="!loading" description="Session not found" />
    </a-spin>

    <ToolCallDetailDrawer :open="Boolean(selectedToolCall)" :call="selectedToolCall" :show-session-link="false" @close="closeToolCall" />
  </div>
</template>
