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
import { useMessages } from '../i18n'
import { statusClass, statusColor } from '../presentation/status'
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
const { t } = useMessages({
  en: {
    'nav.sessions': 'Sessions',
    'title': 'Session Trace',
    'subtitle.fallback': 'Timeline, calls, metadata, and raw local events',
    'action.refresh': 'Refresh',
    'summary.trace': 'Trace',
    'status.parsePrefix': 'parse',
    'status.unpriced': 'unpriced',
    'fallback.unknown': 'unknown',
    'fallback.indexMessage': 'No index message recorded',
    'fallback.noRawJson': 'No raw JSON recorded',
    'metric.agent': 'Agent',
    'metric.model': 'Model',
    'metric.started': 'Started',
    'metric.ended': 'Ended',
    'metric.tokens': 'Tokens',
    'metric.estimatedCost': 'Estimated Cost',
    'metric.timing': 'Timing',
    'metric.callsEvents': 'Calls / Events',
    'metric.tokenIn': 'in',
    'metric.tokenOut': 'out',
    'metric.tokenCached': 'cached',
    'metric.missingPricing': 'Missing local pricing for this model',
    'metric.active': 'active',
    'metric.idle': 'idle',
    'count.model': 'model',
    'count.events': 'events',
    'tab.timeline': 'Timeline',
    'tab.calls': 'Calls',
    'tab.model': 'Model',
    'tab.tools': 'Tools',
    'tab.metadata': 'Metadata',
    'tab.rawEvents': 'Raw Events',
    'panel.timeline.kicker': 'Primary inspection surface ordered by local event time',
    'panel.calls.kicker': 'Model and tool invocations with aligned usage and duration',
    'panel.metadata.kicker': 'Session source, timing breakdown, parser, and index context',
    'panel.rawEvents.kicker': 'Source lines, raw types, and event summaries',
    'empty.modelCalls': 'No model calls captured for this session',
    'empty.toolCalls': 'No tool calls captured for this session',
    'empty.sessionNotFound': 'Session not found',
    'empty.rawEvents': 'No raw events captured for this session',
    'column.ended': 'Ended',
    'column.model': 'Model',
    'column.status': 'Status',
    'column.duration': 'Duration',
    'column.input': 'Input',
    'column.cached': 'Cached',
    'column.output': 'Output',
    'column.reasoning': 'Reasoning',
    'column.total': 'Total',
    'column.cost': 'Cost',
    'column.started': 'Started',
    'column.tool': 'Tool',
    'column.rawEvent': 'Raw Event',
    'column.line': 'Line',
    'column.time': 'Time',
    'column.kind': 'Kind',
    'column.type': 'Type',
    'column.summary': 'Summary',
    'tooltip.viewDetails': 'View details',
    'metadata.sessionRow': 'Session row',
    'metadata.sessionKey': 'Session key',
    'metadata.agent': 'Agent',
    'metadata.agentKind': 'Agent kind',
    'metadata.started': 'Started',
    'metadata.ended': 'Ended',
    'metadata.model': 'Model',
    'metadata.provider': 'Provider',
    'metadata.originator': 'Originator',
    'metadata.threadSource': 'Thread source',
    'metadata.parseStatus': 'Parse status',
    'metadata.indexStatus': 'Index status',
    'metadata.usageSource': 'Usage source',
    'metadata.pricing': 'Pricing',
    'metadata.wall': 'Wall',
    'metadata.active': 'Active',
    'metadata.modelTime': 'Model time',
    'metadata.toolTime': 'Tool time',
    'metadata.idle': 'Idle',
    'metadata.events': 'Events',
    'metadata.project': 'Project',
    'metadata.rawSource': 'Raw source',
    'metadata.indexMessage': 'Index message',
    'pricing.unpriced': 'unpriced',
    'pricing.priced': 'priced',
    'event.linePrefix': 'line'
  },
  'zh-CN': {
    'nav.sessions': '会话',
    'title': '会话轨迹',
    'subtitle.fallback': '时间线、调用、元数据和本地原始事件',
    'action.refresh': '刷新',
    'summary.trace': '轨迹',
    'status.parsePrefix': '解析',
    'status.unpriced': '未定价',
    'fallback.unknown': '未知',
    'fallback.indexMessage': '没有记录索引消息',
    'fallback.noRawJson': '没有记录原始 JSON',
    'metric.agent': 'Agent',
    'metric.model': '模型',
    'metric.started': '开始',
    'metric.ended': '结束',
    'metric.tokens': 'Token',
    'metric.estimatedCost': '预估费用',
    'metric.timing': '耗时',
    'metric.callsEvents': '调用 / 事件',
    'metric.tokenIn': '输入',
    'metric.tokenOut': '输出',
    'metric.tokenCached': '缓存',
    'metric.missingPricing': '本地缺少此模型的价格',
    'metric.active': '活跃',
    'metric.idle': '空闲',
    'count.model': '模型',
    'count.events': '事件',
    'tab.timeline': '时间线',
    'tab.calls': '调用',
    'tab.model': '模型',
    'tab.tools': '工具',
    'tab.metadata': '元数据',
    'tab.rawEvents': '原始事件',
    'panel.timeline.kicker': '按本地事件时间排序的主要检查视图',
    'panel.calls.kicker': '模型和工具调用，以及对应的用量和耗时',
    'panel.metadata.kicker': '会话来源、耗时拆分、解析器和索引上下文',
    'panel.rawEvents.kicker': '来源行、原始类型和事件摘要',
    'empty.modelCalls': '此会话没有捕获到模型调用',
    'empty.toolCalls': '此会话没有捕获到工具调用',
    'empty.sessionNotFound': '未找到会话',
    'empty.rawEvents': '此会话没有捕获到原始事件',
    'column.ended': '结束',
    'column.model': '模型',
    'column.status': '状态',
    'column.duration': '耗时',
    'column.input': '输入',
    'column.cached': '缓存',
    'column.output': '输出',
    'column.reasoning': '推理',
    'column.total': '总计',
    'column.cost': '费用',
    'column.started': '开始',
    'column.tool': '工具',
    'column.rawEvent': '原始事件',
    'column.line': '行',
    'column.time': '时间',
    'column.kind': '种类',
    'column.type': '类型',
    'column.summary': '摘要',
    'tooltip.viewDetails': '查看详情',
    'metadata.sessionRow': '会话行',
    'metadata.sessionKey': '会话 key',
    'metadata.agent': 'Agent',
    'metadata.agentKind': 'Agent 类型',
    'metadata.started': '开始',
    'metadata.ended': '结束',
    'metadata.model': '模型',
    'metadata.provider': '提供方',
    'metadata.originator': '来源方',
    'metadata.threadSource': '线程来源',
    'metadata.parseStatus': '解析状态',
    'metadata.indexStatus': '索引状态',
    'metadata.usageSource': '用量来源',
    'metadata.pricing': '价格',
    'metadata.wall': '总耗时',
    'metadata.active': '活跃',
    'metadata.modelTime': '模型耗时',
    'metadata.toolTime': '工具耗时',
    'metadata.idle': '空闲',
    'metadata.events': '事件',
    'metadata.project': '项目',
    'metadata.rawSource': '原始来源',
    'metadata.indexMessage': '索引消息',
    'pricing.unpriced': '未定价',
    'pricing.priced': '已定价',
    'event.linePrefix': '行'
  }
})
const loading = ref(true)
const detail = ref<SessionDetail | null>(null)
const selectedToolCall = ref<ToolCall | null>(null)

const modelColumns = computed(() => [
  { title: t('column.ended'), dataIndex: 'endedAt', key: 'endedAt', width: 150 },
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 220 },
  { title: t('column.status'), dataIndex: 'status', key: 'status', width: 130 },
  { title: t('column.duration'), dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
  { title: t('column.input'), dataIndex: 'inputTokens', key: 'input', width: 100, align: 'right' },
  { title: t('column.cached'), dataIndex: 'cachedInputTokens', key: 'cached', width: 100, align: 'right' },
  { title: t('column.output'), dataIndex: 'outputTokens', key: 'output', width: 100, align: 'right' },
  { title: t('column.reasoning'), dataIndex: 'reasoningOutputTokens', key: 'reasoning', width: 110, align: 'right' },
  { title: t('column.total'), dataIndex: 'totalTokens', key: 'total', width: 110, align: 'right' },
  { title: t('column.cost'), dataIndex: 'costUsd', key: 'cost', width: 120, align: 'right' }
])

const toolColumns = computed(() => [
  { title: t('column.started'), dataIndex: 'startedAt', key: 'startedAt', width: 150 },
  { title: t('column.ended'), dataIndex: 'endedAt', key: 'endedAt', width: 150 },
  { title: t('column.tool'), dataIndex: 'toolName', key: 'toolName', width: 160 },
  { title: t('column.status'), dataIndex: 'status', key: 'status', width: 110 },
  { title: t('column.duration'), dataIndex: 'durationMs', key: 'duration', width: 110, align: 'right' },
  { title: t('column.rawEvent'), dataIndex: 'rawEventId', key: 'rawEvent', width: 100, align: 'right' },
  { title: t('column.input'), dataIndex: 'inputSummary', key: 'input' },
  { title: t('column.output'), dataIndex: 'outputSummary', key: 'output' },
  { title: '', key: 'detail', width: 56, align: 'right' }
])

const rawColumns = computed(() => [
  { title: t('column.line'), dataIndex: 'sourceLine', key: 'line', width: 80, align: 'right' },
  { title: t('column.time'), dataIndex: 'timestamp', key: 'time', width: 150 },
  { title: t('column.kind'), dataIndex: 'kind', key: 'kind', width: 100 },
  { title: t('column.type'), dataIndex: 'rawType', key: 'rawType', width: 150 },
  { title: t('column.summary'), dataIndex: 'summary', key: 'summary' }
])

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

function eventColor(kind: string) {
  if (kind === 'model') return 'blue'
  if (kind === 'tool') return 'purple'
  if (kind === 'error') return 'red'
  return 'default'
}

function indexStatusHint(session: SessionDetail['session']) {
  return session.lastIndexedScanMessage || session.rawSourcePath || t('fallback.indexMessage')
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
          {{ t('nav.sessions') }}
        </a-button>
        <h1 class="page-title">{{ t('title') }}</h1>
        <div class="page-subtitle">
          {{ detail ? shortPath(detail.session.projectPath) : t('subtitle.fallback') }}
        </div>
      </div>
      <a-button @click="load">
        <template #icon>
          <ReloadOutlined />
        </template>
        {{ t('action.refresh') }}
      </a-button>
    </div>

    <a-spin :spinning="loading">
      <template v-if="detail">
        <section class="summary-panel session-summary-panel">
          <div class="session-summary-main">
            <div class="metric-label">{{ t('summary.trace') }}</div>
            <div class="summary-title mono">{{ sessionLabel(detail.session) }}</div>
            <a-tooltip :title="detail.session.projectPath" placement="topLeft">
              <div class="session-summary-project">{{ shortPath(detail.session.projectPath) }}</div>
            </a-tooltip>
            <div class="summary-meta">
              <a-tag class="status-tag parse-status-tag" :class="statusClass(detail.session.parseStatus)" :color="statusColor(detail.session.parseStatus)">
                {{ t('status.parsePrefix') }} {{ detail.session.parseStatus || t('fallback.unknown') }}
              </a-tag>
              <a-tooltip :title="indexStatusHint(detail.session)" placement="topLeft">
                <a-tag
                  class="status-tag parse-status-tag"
                  :class="statusClass(detail.session.lastIndexedScanStatus)"
                  :color="statusColor(detail.session.lastIndexedScanStatus)"
                >
                  {{ detail.session.lastIndexedScanStatus || t('fallback.unknown') }}
                </a-tag>
              </a-tooltip>
              <a-tag v-if="detail.session.unpriced" class="status-tag model-status-tag" color="warning">{{ t('status.unpriced') }}</a-tag>
            </div>
          </div>
          <div class="session-summary-meta">
            <div class="session-summary-item">
              <span class="metric-label">{{ t('metric.agent') }}</span>
              <strong>{{ detail.session.agentName || detail.session.agentKind || t('fallback.unknown') }}</strong>
            </div>
            <div class="session-summary-item">
              <span class="metric-label">{{ t('metric.model') }}</span>
              <strong>{{ detail.session.model }}</strong>
            </div>
            <div class="session-summary-item">
              <span class="metric-label">{{ t('metric.started') }}</span>
              <strong>{{ formatDateTime(detail.session.startedAt) }}</strong>
            </div>
            <div class="session-summary-item">
              <span class="metric-label">{{ t('metric.ended') }}</span>
              <strong>{{ formatDateTime(detail.session.endedAt) }}</strong>
            </div>
          </div>
        </section>

        <div class="metric-grid session-metric-grid">
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-card-topline">
              <div class="metric-label">{{ t('metric.tokens') }}</div>
              <FunctionOutlined class="metric-icon" />
            </div>
            <div class="metric-value">{{ formatNumber(detail.session.tokenUsage.totalTokens) }}</div>
            <div class="metric-note">
              {{ formatNumber(detail.session.tokenUsage.inputTokens) }} {{ t('metric.tokenIn') }} ·
              {{ formatNumber(detail.session.tokenUsage.outputTokens) }} {{ t('metric.tokenOut') }} ·
              {{ formatNumber(detail.session.tokenUsage.cachedInputTokens) }} {{ t('metric.tokenCached') }} ·
              {{ detail.session.tokenUsage.source }}
            </div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-card-topline">
              <div class="metric-label">{{ t('metric.estimatedCost') }}</div>
              <DollarCircleOutlined class="metric-icon" />
            </div>
            <div class="metric-value">{{ formatCost(detail.session.estimatedCostUsd) }}</div>
            <div class="metric-note" :class="{ 'metric-note-warning': detail.session.unpriced }">
              {{ detail.session.unpriced ? t('metric.missingPricing') : detail.session.model }}
            </div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-card-topline">
              <div class="metric-label">{{ t('metric.timing') }}</div>
              <ClockCircleOutlined class="metric-icon" />
            </div>
            <div class="metric-value">{{ formatDuration(detail.session.wallDurationMs) }}</div>
            <div class="metric-note">
              {{ formatDuration(detail.session.activeDurationMs) }} {{ t('metric.active') }} ·
              {{ formatDuration(detail.session.idleDurationMs) }} {{ t('metric.idle') }}
            </div>
          </a-card>
          <a-card class="metric-card session-metric-card" :bordered="false">
            <div class="metric-card-topline">
              <div class="metric-label">{{ t('metric.callsEvents') }}</div>
              <ToolOutlined class="metric-icon" />
            </div>
            <div class="metric-value">{{ formatNumber(detail.session.toolCallCount) }}</div>
            <div class="metric-note">
              {{ formatNumber(detail.modelCalls.length) }} {{ t('count.model') }} ·
              {{ formatNumber(detail.session.eventCount) }} {{ t('count.events') }}
            </div>
          </a-card>
        </div>

        <a-tabs class="session-detail-tabs" type="card">
          <a-tab-pane key="timeline" :tab="t('tab.timeline') + ' (' + formatNumber(events.length) + ')'">
            <section class="panel session-timeline-panel">
              <div class="panel-header">
                <div>
                  <h2 class="panel-title">{{ t('tab.timeline') }}</h2>
                  <div class="panel-kicker">{{ t('panel.timeline.kicker') }}</div>
                </div>
                <span class="muted">{{ formatNumber(events.length) }} {{ t('count.events') }}</span>
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
                      <div class="muted mono timeline-event-raw">{{ t('event.linePrefix') }} {{ event.sourceLine }} · {{ event.rawType }}</div>
                    </div>
                  </a-timeline-item>
                </a-timeline>
              </div>
            </section>
          </a-tab-pane>

          <a-tab-pane key="calls" :tab="t('tab.calls') + ' (' + formatNumber(detail.modelCalls.length + detail.toolCalls.length) + ')'">
            <section class="panel session-calls-panel">
              <div class="panel-header">
                <div>
                  <h2 class="panel-title">{{ t('tab.calls') }}</h2>
                  <div class="panel-kicker">{{ t('panel.calls.kicker') }}</div>
                </div>
              </div>
              <a-tabs class="panel-body calls-tabs">
                <a-tab-pane key="model" :tab="t('tab.model') + ' (' + formatNumber(detail.modelCalls.length) + ')'">
                  <a-table
                    class="calls-table model-calls-table"
                    size="small"
                    :columns="modelColumns"
                    :data-source="detail.modelCalls"
                    :locale="{ emptyText: t('empty.modelCalls') }"
                    :pagination="{ pageSize: 8 }"
                    :scroll="{ x: 1200 }"
                    row-key="id"
                  >
                    <template #bodyCell="{ column, record }">
                      <template v-if="column.key === 'endedAt'">{{ formatDateTime(record.endedAt) }}</template>
                      <template v-else-if="column.key === 'model'">
                        <a-typography-text class="model-name" :ellipsis="{ tooltip: record.model }">
                          {{ record.model || t('fallback.unknown') }}
                        </a-typography-text>
                        <div class="timeline-event-raw">{{ record.provider || '-' }}</div>
                      </template>
                      <template v-else-if="column.key === 'status'">
                        <div class="timeline-event-head">
                          <a-tag class="status-tag call-status-tag" :class="statusClass(record.status)" :color="statusColor(record.status)">
                            {{ record.status || t('fallback.unknown') }}
                          </a-tag>
                          <a-tag v-if="record.unpriced" class="status-tag model-status-tag" color="warning">{{ t('status.unpriced') }}</a-tag>
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
                <a-tab-pane key="tools" :tab="t('tab.tools') + ' (' + formatNumber(detail.toolCalls.length) + ')'">
                  <a-table
                    class="calls-table tool-calls-table"
                    size="small"
                    :columns="toolColumns"
                    :data-source="detail.toolCalls"
                    :locale="{ emptyText: t('empty.toolCalls') }"
                    :pagination="{ pageSize: 8 }"
                    :scroll="{ x: 1180 }"
                    row-key="id"
                  >
                    <template #bodyCell="{ column, record }">
                      <template v-if="column.key === 'startedAt'">{{ formatDateTime(record.startedAt) }}</template>
                      <template v-else-if="column.key === 'endedAt'">{{ formatDateTime(record.endedAt) }}</template>
                      <template v-else-if="column.key === 'status'">
                        <a-tooltip :title="record.error || record.status || t('fallback.unknown')">
                          <a-tag class="status-tag call-status-tag" :class="statusClass(record.status)" :color="statusColor(record.status)">
                            {{ record.status || t('fallback.unknown') }}
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
                        <a-tooltip :title="t('tooltip.viewDetails')">
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
          </a-tab-pane>

          <a-tab-pane key="metadata" :tab="t('tab.metadata')">
            <section class="panel session-metadata-panel">
              <div class="panel-header">
                <div>
                  <h2 class="panel-title">{{ t('tab.metadata') }}</h2>
                  <div class="panel-kicker">{{ t('panel.metadata.kicker') }}</div>
                </div>
              </div>
              <div class="panel-body metadata-grid">
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.sessionRow') }}</div>
                  <div class="metadata-value number-cell">{{ formatNumber(detail.session.id) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.sessionKey') }}</div>
                  <div class="metadata-value mono">{{ sessionLabel(detail.session) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.agent') }}</div>
                  <div class="metadata-value">{{ detail.session.agentName || detail.session.agentKind || '-' }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.agentKind') }}</div>
                  <div class="metadata-value">{{ detail.session.agentKind || '-' }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.started') }}</div>
                  <div class="metadata-value">{{ formatDateTime(detail.session.startedAt) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.ended') }}</div>
                  <div class="metadata-value">{{ formatDateTime(detail.session.endedAt) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.model') }}</div>
                  <div class="metadata-value">{{ detail.session.model }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.provider') }}</div>
                  <div class="metadata-value">{{ detail.session.modelProvider || '-' }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.originator') }}</div>
                  <div class="metadata-value">{{ detail.session.originator || '-' }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.threadSource') }}</div>
                  <div class="metadata-value">{{ detail.session.threadSource || '-' }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.parseStatus') }}</div>
                  <a-tag class="status-tag parse-status-tag" :class="statusClass(detail.session.parseStatus)" :color="statusColor(detail.session.parseStatus)">
                    {{ detail.session.parseStatus || t('fallback.unknown') }}
                  </a-tag>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.indexStatus') }}</div>
                  <a-tag
                    class="status-tag parse-status-tag"
                    :class="statusClass(detail.session.lastIndexedScanStatus)"
                    :color="statusColor(detail.session.lastIndexedScanStatus)"
                  >
                    {{ detail.session.lastIndexedScanStatus || t('fallback.unknown') }}
                  </a-tag>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.usageSource') }}</div>
                  <div class="metadata-value">{{ detail.session.tokenUsage.source }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.pricing') }}</div>
                  <div class="metadata-value">{{ detail.session.unpriced ? t('pricing.unpriced') : t('pricing.priced') }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.wall') }}</div>
                  <div class="metadata-value number-cell">{{ formatDuration(detail.session.wallDurationMs) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.active') }}</div>
                  <div class="metadata-value number-cell">{{ formatDuration(detail.session.activeDurationMs) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.modelTime') }}</div>
                  <div class="metadata-value number-cell">{{ formatDuration(detail.session.modelDurationMs) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.toolTime') }}</div>
                  <div class="metadata-value number-cell">{{ formatDuration(detail.session.toolDurationMs) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.idle') }}</div>
                  <div class="metadata-value number-cell">{{ formatDuration(detail.session.idleDurationMs) }}</div>
                </div>
                <div class="metadata-item">
                  <div class="metadata-label">{{ t('metadata.events') }}</div>
                  <div class="metadata-value number-cell">{{ formatNumber(detail.session.eventCount) }}</div>
                </div>
                <div class="metadata-item is-wide">
                  <div class="metadata-label">{{ t('metadata.project') }}</div>
                  <a-typography-text class="metadata-value detail-path" :ellipsis="{ tooltip: detail.session.projectPath }">
                    {{ detail.session.projectPath }}
                  </a-typography-text>
                </div>
                <div class="metadata-item is-wide">
                  <div class="metadata-label">{{ t('metadata.rawSource') }}</div>
                  <a-typography-text class="metadata-value detail-path mono" :ellipsis="{ tooltip: detail.session.rawSourcePath }">
                    {{ detail.session.rawSourcePath }}
                  </a-typography-text>
                </div>
                <div v-if="detail.session.lastIndexedScanMessage" class="metadata-item is-wide">
                  <div class="metadata-label">{{ t('metadata.indexMessage') }}</div>
                  <div class="metadata-value">{{ detail.session.lastIndexedScanMessage }}</div>
                </div>
              </div>
            </section>
          </a-tab-pane>

          <a-tab-pane key="raw" :tab="t('tab.rawEvents') + ' (' + formatNumber(events.length) + ')'">
            <section class="panel raw-events-panel">
              <div class="panel-header">
                <div>
                  <h2 class="panel-title">{{ t('tab.rawEvents') }}</h2>
                  <div class="panel-kicker">{{ t('panel.rawEvents.kicker') }}</div>
                </div>
              </div>
              <a-table
                class="raw-events-table"
                size="small"
                :columns="rawColumns"
                :data-source="events"
                :expandable="rawEventsExpandable"
                :pagination="{ pageSize: 10 }"
                :locale="{ emptyText: t('empty.rawEvents') }"
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
                    {{ record.rawJson || t('fallback.noRawJson') }}
                  </a-typography-paragraph>
                </template>
              </a-table>
            </section>
          </a-tab-pane>
        </a-tabs>
      </template>
      <a-empty v-else-if="!loading" :description="t('empty.sessionNotFound')" />
    </a-spin>

    <ToolCallDetailDrawer :open="Boolean(selectedToolCall)" :call="selectedToolCall" :show-session-link="false" @close="closeToolCall" />
  </div>
</template>
