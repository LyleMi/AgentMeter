<script setup lang="ts">
import { computed, onMounted, ref, type Component, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import {
  ArrowRightOutlined,
  ClockCircleOutlined,
  FieldTimeOutlined,
  ReloadOutlined,
  TableOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatDateTime,
  formatDuration,
  formatNumber,
  sessionLabel,
  shortPath,
  type AgentTimeUsage,
  type ModelTimeUsage,
  type Overview,
  type Session,
  type ToolTimeUsage
} from '../api'
import PageHeader from '../components/PageHeader.vue'
import { useMessages } from '../i18n'
import { sourceDisplay, sourceInstanceKey } from '../presentation/sourceIdentity'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const router = useRouter()
const loading = ref(true)
const overview = ref<Overview | null>(null)
const { t, createNumberFormatter } = useMessages({
  en: {
    'title': 'Time',
    'subtitle': 'Wall-time attribution across model, tool, source, and slow session activity',
    'action.refresh': 'Refresh',
    'composition.title': 'Time Composition',
    'composition.kicker': 'Exclusive wall-time split. Network-likely tool time is inferred from tool name/input, not network telemetry.',
    'composition.model': 'Model',
    'composition.network': 'Suspected network tools',
    'composition.tools': 'Other tools',
    'composition.idle': 'Idle / unclassified',
    'composition.total': 'Wall total',
    'kpi.wall': 'Wall Time',
    'kpi.wallNote': '{count} indexed sessions',
    'kpi.activeShare': 'Active Share',
    'kpi.activeShareNote': '{duration} measured model and tool time',
    'kpi.toolShare': 'Tool Share',
    'kpi.toolShareNote': '{duration} total tool time',
    'kpi.networkShare': 'Network-Likely Share',
    'kpi.networkShareNote': '{duration} across {count} calls',
    'kpi.slowest': 'Slowest Session',
    'kpi.slowestNote': '{duration} wall time',
    'tools.title': 'Tool Duration Leaders',
    'tools.kicker': 'Ranked by total tool duration',
    'tools.networkHint': 'Network-likely marker uses inferred tool name/input signals.',
    'agent.title': 'Source Time Attribution',
    'agent.kicker': 'Wall, active, model, tool, idle, and inferred network time by source',
    'model.title': 'Model Time Attribution',
    'model.kicker': 'Wall, active, and token volume by model',
    'sessions.title': 'Slow Sessions',
    'sessions.kicker': 'Sorted by wall duration; open a row for the session timeline',
    'column.tool': 'Tool',
    'column.calls': 'Calls',
    'column.success': 'Success',
    'column.failed': 'Failed / Pending',
    'column.total': 'Total',
    'column.average': 'Avg',
    'column.max': 'Max',
    'column.network': 'Network',
    'column.agent': 'Source',
    'column.model': 'Model',
    'column.sessions': 'Sessions',
    'column.tokens': 'Tokens',
    'column.wall': 'Wall',
    'column.active': 'Active',
    'column.modelTime': 'Model',
    'column.toolTime': 'Tools',
    'column.idle': 'Idle',
    'column.project': 'Project',
    'column.started': 'Started',
    'column.open': 'Open',
    'status.networkLikely': 'likely',
    'status.notNetwork': 'no',
    'empty.title': 'No time analysis yet',
    'empty.text': 'Time attribution appears after sessions with model, tool, and wall durations are indexed.',
    'empty.tools': 'No tool duration leaders yet',
    'empty.agents': 'No source time attribution yet',
    'empty.models': 'No model time attribution yet',
    'empty.sessions': 'No slow sessions yet',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    'title': '耗时',
    'subtitle': '按模型、工具、来源和慢会话归因墙钟耗时',
    'action.refresh': '刷新',
    'composition.title': '时间构成',
    'composition.kicker': '按墙钟时间互斥拆分。疑似网络工具耗时由工具名/输入推断，不是网络遥测。',
    'composition.model': '模型',
    'composition.network': '疑似网络工具',
    'composition.tools': '其他工具',
    'composition.idle': '空闲 / 未分类',
    'composition.total': '墙钟总计',
    'kpi.wall': '墙钟时间',
    'kpi.wallNote': '{count} 个已索引会话',
    'kpi.activeShare': '活跃占比',
    'kpi.activeShareNote': '{duration} 已测量模型和工具时间',
    'kpi.toolShare': '工具占比',
    'kpi.toolShareNote': '{duration} 工具总耗时',
    'kpi.networkShare': '疑似网络占比',
    'kpi.networkShareNote': '{duration}，共 {count} 次调用',
    'kpi.slowest': '最慢会话',
    'kpi.slowestNote': '墙钟时间 {duration}',
    'tools.title': '工具耗时排行',
    'tools.kicker': '按工具总耗时排序',
    'tools.networkHint': '疑似网络标记来自工具名/输入信号推断。',
    'agent.title': '来源时间归因',
    'agent.kicker': '按来源展示墙钟、活跃、模型、工具、空闲和疑似网络时间',
    'model.title': '模型时间归因',
    'model.kicker': '按模型展示墙钟、活跃和 Token 规模',
    'sessions.title': '慢会话',
    'sessions.kicker': '按墙钟耗时排序；打开行查看会话时间线',
    'column.tool': '工具',
    'column.calls': '调用',
    'column.success': '成功',
    'column.failed': '失败 / 未完成',
    'column.total': '总计',
    'column.average': '平均',
    'column.max': '最大',
    'column.network': '网络',
    'column.agent': '来源',
    'column.model': '模型',
    'column.sessions': '会话',
    'column.tokens': 'Token',
    'column.wall': '墙钟',
    'column.active': '活跃',
    'column.modelTime': '模型',
    'column.toolTime': '工具',
    'column.idle': '空闲',
    'column.project': '项目',
    'column.started': '开始时间',
    'column.open': '打开',
    'status.networkLikely': '疑似',
    'status.notNetwork': '否',
    'empty.title': '暂无时间分析',
    'empty.text': '索引包含模型、工具和墙钟时长的会话后，会显示时间归因。',
    'empty.tools': '暂无工具耗时排行',
    'empty.agents': '暂无来源时间归因',
    'empty.models': '暂无模型时间归因',
    'empty.sessions': '暂无慢会话',
    'fallback.unknown': '未知'
  }
})

interface TimeSegment {
  key: string
  label: string
  value: number
  share: number
  width: string
  tone: string
}

interface KpiCard {
  label: string
  value: string
  note: string
  icon: Component
}

const hasIndexedData = computed(() => (overview.value?.totalSessions || 0) > 0)
const wallDurationMs = computed(() => Math.max(0, overview.value?.totalWallDurationMs || 0))
const activeDurationMs = computed(() => Math.max(0, overview.value?.totalActiveDurationMs || 0))
const toolDurationMs = computed(() => Math.max(0, overview.value?.totalToolDurationMs || 0))
const suspectedNetworkDurationMs = computed(() =>
  Math.max(0, Math.min(overview.value?.suspectedNetworkToolDurationMs || 0, toolDurationMs.value))
)
const slowSessions = computed(() => overview.value?.slowSessions || [])
const slowestSession = computed(() => slowSessions.value[0])

const compositionSegments = computed<TimeSegment[]>(() => {
  const wall = wallDurationMs.value
  let remaining = wall
  const modelDuration = takeDuration(overview.value?.totalModelDurationMs || 0, remaining)
  remaining -= modelDuration
  const networkDuration = takeDuration(suspectedNetworkDurationMs.value, remaining)
  remaining -= networkDuration
  const otherToolDuration = takeDuration(Math.max(0, toolDurationMs.value - suspectedNetworkDurationMs.value), remaining)
  remaining -= otherToolDuration
  const idleDuration = Math.max(0, remaining)

  return [
    buildSegment('model', t('composition.model'), modelDuration, wall, 'is-model'),
    buildSegment('network', t('composition.network'), networkDuration, wall, 'is-network'),
    buildSegment('tools', t('composition.tools'), otherToolDuration, wall, 'is-tools'),
    buildSegment('idle', t('composition.idle'), idleDuration, wall, 'is-idle')
  ]
})

const kpiCards = computed<KpiCard[]>(() => [
  {
    label: t('kpi.wall'),
    value: formatDuration(wallDurationMs.value),
    note: t('kpi.wallNote', { count: formatNumber(overview.value?.totalSessions) }),
    icon: FieldTimeOutlined
  },
  {
    label: t('kpi.activeShare'),
    value: formatPercent(activeDurationMs.value / Math.max(wallDurationMs.value, 1)),
    note: t('kpi.activeShareNote', { duration: formatDuration(activeDurationMs.value) }),
    icon: ClockCircleOutlined
  },
  {
    label: t('kpi.toolShare'),
    value: formatPercent(toolDurationMs.value / Math.max(wallDurationMs.value, 1)),
    note: t('kpi.toolShareNote', { duration: formatDuration(toolDurationMs.value) }),
    icon: ToolOutlined
  },
  {
    label: t('kpi.networkShare'),
    value: formatPercent(suspectedNetworkDurationMs.value / Math.max(wallDurationMs.value, 1)),
    note: t('kpi.networkShareNote', {
      duration: formatDuration(suspectedNetworkDurationMs.value),
      count: formatNumber(overview.value?.suspectedNetworkToolCalls)
    }),
    icon: ToolOutlined
  },
  {
    label: t('kpi.slowest'),
    value: slowestSession.value ? sessionLabel(slowestSession.value) : '-',
    note: slowestSession.value ? t('kpi.slowestNote', { duration: formatDuration(slowestSession.value.wallDurationMs) }) : t('empty.sessions'),
    icon: ClockCircleOutlined
  }
])

const rankedToolLeaders = computed(() =>
  [...(overview.value?.toolTimeLeaders || [])].sort((left, right) => right.totalDurationMs - left.totalDurationMs)
)
const rankedAgentTimeUsage = computed(() =>
  [...(overview.value?.agentTimeUsage || [])].sort((left, right) => right.wallDurationMs - left.wallDurationMs)
)
const rankedModelTimeUsage = computed(() =>
  [...(overview.value?.modelTimeUsage || [])].sort((left, right) => right.wallDurationMs - left.wallDurationMs)
)

const hasToolLeaders = computed(() => rankedToolLeaders.value.length > 0)
const hasAgentTimeUsage = computed(() => rankedAgentTimeUsage.value.length > 0)
const hasModelTimeUsage = computed(() => rankedModelTimeUsage.value.length > 0)
const hasSlowSessions = computed(() => slowSessions.value.length > 0)

const toolColumns = computed(() => [
  { title: t('column.tool'), dataIndex: 'toolName', key: 'toolName', width: 210 },
  { title: t('column.calls'), dataIndex: 'calls', key: 'calls', width: 86, align: 'right' },
  { title: t('column.success'), dataIndex: 'successCalls', key: 'success', width: 92, align: 'right' },
  { title: t('column.failed'), dataIndex: 'failedCalls', key: 'failed', width: 120, align: 'right' },
  { title: t('column.total'), dataIndex: 'totalDurationMs', key: 'total', width: 110, align: 'right' },
  { title: t('column.average'), dataIndex: 'avgDurationMs', key: 'average', width: 110, align: 'right' },
  { title: t('column.max'), dataIndex: 'maxDurationMs', key: 'max', width: 110, align: 'right' },
  { title: t('column.network'), dataIndex: 'suspectedNetwork', key: 'network', width: 104 }
])

const agentColumns = computed(() => [
  { title: t('column.agent'), dataIndex: 'sourceLabel', key: 'agent', width: 190 },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 86, align: 'right' },
  { title: t('column.calls'), dataIndex: 'toolCalls', key: 'calls', width: 80, align: 'right' },
  { title: t('column.wall'), dataIndex: 'wallDurationMs', key: 'wall', width: 104, align: 'right' },
  { title: t('column.active'), dataIndex: 'activeDurationMs', key: 'active', width: 104, align: 'right' },
  { title: t('column.modelTime'), dataIndex: 'modelDurationMs', key: 'modelTime', width: 104, align: 'right' },
  { title: t('column.toolTime'), dataIndex: 'toolDurationMs', key: 'toolTime', width: 104, align: 'right' },
  { title: t('column.network'), dataIndex: 'suspectedNetworkToolDurationMs', key: 'network', width: 104, align: 'right' },
  { title: t('column.idle'), dataIndex: 'idleDurationMs', key: 'idle', width: 104, align: 'right' }
])

const modelColumns = computed(() => [
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 190 },
  { title: t('column.sessions'), dataIndex: 'sessionCount', key: 'sessions', width: 86, align: 'right' },
  { title: t('column.tokens'), dataIndex: 'totalTokens', key: 'tokens', width: 110, align: 'right' },
  { title: t('column.wall'), dataIndex: 'wallDurationMs', key: 'wall', width: 104, align: 'right' },
  { title: t('column.active'), dataIndex: 'activeDurationMs', key: 'active', width: 104, align: 'right' },
  { title: t('column.modelTime'), dataIndex: 'modelDurationMs', key: 'modelTime', width: 104, align: 'right' },
  { title: t('column.toolTime'), dataIndex: 'toolDurationMs', key: 'toolTime', width: 104, align: 'right' },
  { title: t('column.idle'), dataIndex: 'idleDurationMs', key: 'idle', width: 104, align: 'right' }
])

const slowSessionColumns = computed(() => [
  { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 260 },
  { title: t('column.agent'), dataIndex: 'sourceLabel', key: 'agent', width: 170 },
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 140 },
  { title: t('column.wall'), dataIndex: 'wallDurationMs', key: 'wall', width: 104, align: 'right' },
  { title: t('column.active'), dataIndex: 'activeDurationMs', key: 'active', width: 104, align: 'right' },
  { title: t('column.modelTime'), dataIndex: 'modelDurationMs', key: 'modelTime', width: 104, align: 'right' },
  { title: t('column.toolTime'), dataIndex: 'toolDurationMs', key: 'toolTime', width: 104, align: 'right' },
  { title: t('column.started'), dataIndex: 'startedAt', key: 'started', width: 150 },
  { title: '', key: 'open', width: 52, align: 'right' }
])

function takeDuration(value: number, remaining: number) {
  return Math.min(Math.max(0, value), Math.max(0, remaining))
}

function buildSegment(key: string, label: string, value: number, wall: number, tone: string): TimeSegment {
  const share = wall > 0 ? value / wall : 0
  return {
    key,
    label,
    value,
    share,
    width: share > 0 ? `${Math.max(1, share * 100)}%` : '0%',
    tone
  }
}

function formatPercent(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '0%'
  if (value < 0.01) return '<1%'
  return createNumberFormatter({ style: 'percent', maximumFractionDigits: 0 }).format(value)
}

function toolRowKey(record: ToolTimeUsage) {
  return record.toolName || t('fallback.unknown')
}

function agentRowKey(record: AgentTimeUsage) {
  return sourceInstanceKey(record, t('fallback.unknown'))
}

function modelRowKey(record: ModelTimeUsage) {
  return record.model || t('fallback.unknown')
}

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function slowSessionRow(record: Session) {
  return { class: 'overview-session-row is-clickable-row', onClick: () => openSession(record.id) }
}

function sourceInfo(record: AgentTimeUsage | Session) {
  return sourceDisplay(record, t('fallback.unknown'))
}

async function load() {
  loading.value = true
  try {
    overview.value = await api.getOverview()
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon>
            <ReloadOutlined />
          </template>
          {{ t('action.refresh') }}
        </a-button>
      </template>
    </PageHeader>

    <a-spin :spinning="loading">
      <div v-if="hasIndexedData" class="overview-time-view">
        <section class="overview-time-top">
          <section class="panel overview-time-composition">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">{{ t('composition.title') }}</h2>
                <div class="panel-kicker">{{ t('composition.kicker') }}</div>
              </div>
              <FieldTimeOutlined class="panel-header-icon" />
            </div>
            <div class="overview-time-composition-body">
              <div class="overview-time-total">
                <span class="metric-label">{{ t('composition.total') }}</span>
                <strong>{{ formatDuration(wallDurationMs) }}</strong>
              </div>
              <div class="overview-time-bar" :aria-label="t('composition.title')">
                <span
                  v-for="item in compositionSegments"
                  :key="item.key"
                  :class="['overview-time-bar-segment', item.tone]"
                  :style="{ width: item.width }"
                  :title="`${item.label}: ${formatDuration(item.value)} (${formatPercent(item.share)})`"
                />
              </div>
              <div class="overview-time-segments">
                <div v-for="item in compositionSegments" :key="item.key" class="overview-time-segment">
                  <span :class="['overview-time-dot', item.tone]"></span>
                  <div>
                    <div class="overview-time-segment-label">{{ item.label }}</div>
                    <div class="overview-time-segment-value">
                      {{ formatDuration(item.value) }}
                      <span>{{ formatPercent(item.share) }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <div class="overview-time-kpis">
            <div v-for="item in kpiCards" :key="item.label" class="overview-kpi-card overview-time-kpi-card">
              <div class="overview-kpi-head">
                <span class="metric-label">{{ item.label }}</span>
                <component :is="item.icon" class="metric-icon" />
              </div>
              <div class="overview-kpi-value" :title="item.value">
                <span>{{ item.value }}</span>
              </div>
              <div class="overview-kpi-note">{{ item.note }}</div>
            </div>
          </div>
        </section>

        <section class="panel overview-time-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('tools.title') }}</h2>
              <div class="panel-kicker">{{ t('tools.kicker') }}</div>
            </div>
            <ToolOutlined class="panel-header-icon" />
          </div>
          <a-table
            v-if="hasToolLeaders"
            class="dense-table overview-time-table"
            size="small"
            :columns="toolColumns"
            :data-source="rankedToolLeaders"
            :pagination="false"
            :row-key="toolRowKey"
            :scroll="{ x: 940 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'toolName'">
                <a-typography-text :ellipsis="{ tooltip: record.toolName }">
                  {{ record.toolName || t('fallback.unknown') }}
                </a-typography-text>
              </template>
              <template v-else-if="column.key === 'calls'">
                <span class="number-cell">{{ formatNumber(record.calls) }}</span>
              </template>
              <template v-else-if="column.key === 'success'">
                <span class="number-cell">{{ formatNumber(record.successCalls) }}</span>
              </template>
              <template v-else-if="column.key === 'failed'">
                <span class="number-cell" :class="{ 'status-error': record.failedCalls > 0 }">{{ formatNumber(record.failedCalls) }}</span>
              </template>
              <template v-else-if="column.key === 'total'">
                <span class="number-cell duration-cell">{{ formatDuration(record.totalDurationMs) }}</span>
              </template>
              <template v-else-if="column.key === 'average'">
                <span class="number-cell duration-cell">{{ formatDuration(record.avgDurationMs) }}</span>
              </template>
              <template v-else-if="column.key === 'max'">
                <span class="number-cell duration-cell">{{ formatDuration(record.maxDurationMs) }}</span>
              </template>
              <template v-else-if="column.key === 'network'">
                <a-tag v-if="record.suspectedNetwork" color="processing" class="status-tag">{{ t('status.networkLikely') }}</a-tag>
                <span v-else class="muted">{{ t('status.notNetwork') }}</span>
              </template>
            </template>
          </a-table>
          <div v-else class="empty-state empty-state-compact">
            <ToolOutlined class="empty-state-icon" />
            <div class="empty-state-title">{{ t('empty.tools') }}</div>
            <div class="empty-state-text">{{ t('empty.text') }}</div>
          </div>
          <div class="panel-footer-note">{{ t('tools.networkHint') }}</div>
        </section>

        <section class="overview-time-attribution-grid">
          <section class="panel overview-time-panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">{{ t('agent.title') }}</h2>
                <div class="panel-kicker">{{ t('agent.kicker') }}</div>
              </div>
              <TableOutlined class="panel-header-icon" />
            </div>
            <a-table
              v-if="hasAgentTimeUsage"
              class="dense-table overview-time-table"
              size="small"
              :columns="agentColumns"
              :data-source="rankedAgentTimeUsage"
              :pagination="false"
              :row-key="agentRowKey"
              :scroll="{ x: 930 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'agent'">
                  <div class="source-identity-cell">
                    <span class="source-identity-name">{{ sourceInfo(record).label }}</span>
                  </div>
                  <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
                </template>
                <template v-else-if="column.key === 'sessions'">
                  <span class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
                </template>
                <template v-else-if="column.key === 'calls'">
                  <span class="number-cell">{{ formatNumber(record.toolCalls) }}</span>
                </template>
                <template v-else>
                  <span class="number-cell duration-cell">{{ formatDuration(record[column.dataIndex]) }}</span>
                </template>
              </template>
            </a-table>
            <div v-else class="empty-state empty-state-compact">
              <TableOutlined class="empty-state-icon" />
              <div class="empty-state-title">{{ t('empty.agents') }}</div>
              <div class="empty-state-text">{{ t('empty.text') }}</div>
            </div>
          </section>

          <section class="panel overview-time-panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">{{ t('model.title') }}</h2>
                <div class="panel-kicker">{{ t('model.kicker') }}</div>
              </div>
              <TableOutlined class="panel-header-icon" />
            </div>
            <a-table
              v-if="hasModelTimeUsage"
              class="dense-table overview-time-table"
              size="small"
              :columns="modelColumns"
              :data-source="rankedModelTimeUsage"
              :pagination="false"
              :row-key="modelRowKey"
              :scroll="{ x: 880 }"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'model'">
                  <a-typography-text class="model-name" :ellipsis="{ tooltip: record.model }">
                    {{ record.model || t('fallback.unknown') }}
                  </a-typography-text>
                </template>
                <template v-else-if="column.key === 'sessions'">
                  <span class="number-cell">{{ formatNumber(record.sessionCount) }}</span>
                </template>
                <template v-else-if="column.key === 'tokens'">
                  <span class="number-cell">{{ formatNumber(record.totalTokens) }}</span>
                </template>
                <template v-else>
                  <span class="number-cell duration-cell">{{ formatDuration(record[column.dataIndex]) }}</span>
                </template>
              </template>
            </a-table>
            <div v-else class="empty-state empty-state-compact">
              <TableOutlined class="empty-state-icon" />
              <div class="empty-state-title">{{ t('empty.models') }}</div>
              <div class="empty-state-text">{{ t('empty.text') }}</div>
            </div>
          </section>
        </section>

        <section class="panel overview-time-panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">{{ t('sessions.title') }}</h2>
              <div class="panel-kicker">{{ t('sessions.kicker') }}</div>
            </div>
            <ClockCircleOutlined class="panel-header-icon" />
          </div>
          <a-table
            v-if="hasSlowSessions"
            class="overview-session-table overview-time-table"
            size="small"
            :columns="slowSessionColumns"
            :data-source="slowSessions"
            :pagination="false"
            row-key="id"
            :custom-row="slowSessionRow"
            :scroll="{ x: 1150 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'project'">
                <div class="overview-session-identity">
                  <a-typography-text class="overview-session-project" :ellipsis="{ tooltip: record.projectPath }">
                    {{ shortPath(record.projectPath) }}
                  </a-typography-text>
                  <span class="overview-session-meta mono">{{ sessionLabel(record) }}</span>
                </div>
              </template>
              <template v-else-if="column.key === 'agent'">
                <div class="source-identity-cell">
                  <span class="source-identity-name">{{ sourceInfo(record).label }}</span>
                </div>
                <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
              </template>
              <template v-else-if="column.key === 'model'">
                <a-typography-text class="model-name" :ellipsis="{ tooltip: record.model }">
                  {{ record.model || t('fallback.unknown') }}
                </a-typography-text>
              </template>
              <template v-else-if="column.key === 'wall'">
                <span class="number-cell">{{ formatDuration(record.wallDurationMs) }}</span>
              </template>
              <template v-else-if="column.key === 'active'">
                <span class="number-cell">{{ formatDuration(record.activeDurationMs) }}</span>
              </template>
              <template v-else-if="column.key === 'modelTime'">
                <span class="number-cell">{{ formatDuration(record.modelDurationMs) }}</span>
              </template>
              <template v-else-if="column.key === 'toolTime'">
                <span class="number-cell">{{ formatDuration(record.toolDurationMs) }}</span>
              </template>
              <template v-else-if="column.key === 'started'">
                {{ formatDateTime(record.startedAt) }}
              </template>
              <template v-else-if="column.key === 'open'">
                <a-button type="text" size="small" :aria-label="t('column.open')" @click.stop="openSession(record.id)">
                  <template #icon>
                    <ArrowRightOutlined />
                  </template>
                </a-button>
              </template>
            </template>
          </a-table>
          <div v-else class="empty-state empty-state-compact">
            <ClockCircleOutlined class="empty-state-icon" />
            <div class="empty-state-title">{{ t('empty.sessions') }}</div>
            <div class="empty-state-text">{{ t('empty.text') }}</div>
          </div>
        </section>
      </div>

      <div v-else-if="!loading" class="empty-state">
        <FieldTimeOutlined class="empty-state-icon" />
        <div class="empty-state-title">{{ t('empty.title') }}</div>
        <div class="empty-state-text">{{ t('empty.text') }}</div>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.overview-time-view {
  display: grid;
  gap: var(--am-section-gap);
}

.overview-time-top {
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(420px, 0.92fr);
  gap: var(--am-section-gap);
}

.overview-time-composition {
  min-width: 0;
}

.overview-time-composition-body {
  display: grid;
  gap: 16px;
  padding: 14px;
}

.overview-time-total {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.overview-time-total strong {
  color: var(--am-text);
  font-size: 28px;
  font-weight: 800;
  line-height: 34px;
  font-variant-numeric: tabular-nums;
}

.overview-time-bar {
  display: flex;
  width: 100%;
  height: 18px;
  overflow: hidden;
  background: var(--am-border-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: 999px;
}

.overview-time-bar-segment {
  display: block;
  min-width: 0;
  height: 100%;
}

.overview-time-bar-segment + .overview-time-bar-segment {
  border-left: 1px solid rgb(255 255 255 / 76%);
}

.overview-time-segments {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.overview-time-segment {
  display: flex;
  align-items: flex-start;
  min-width: 0;
  gap: 8px;
  padding: 10px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.overview-time-dot {
  flex: 0 0 auto;
  width: 9px;
  height: 9px;
  margin-top: 5px;
  border-radius: 999px;
}

.overview-time-segment-label {
  color: var(--am-text-soft);
  font-size: 12px;
  font-weight: 720;
  line-height: 18px;
}

.overview-time-segment-value {
  margin-top: 2px;
  color: var(--am-text);
  font-size: 13px;
  font-weight: 750;
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}

.overview-time-segment-value span {
  margin-left: 6px;
  color: var(--am-muted);
  font-size: 12px;
  font-weight: 650;
}

.is-model {
  background: var(--am-primary);
}

.is-network {
  background: var(--am-info);
}

.is-tools {
  background: var(--am-success);
}

.is-idle {
  background: var(--am-warning);
}

.overview-time-kpis {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.overview-time-kpi-card:last-child {
  grid-column: 1 / -1;
  min-height: 106px;
}

.overview-time-kpi-card .overview-kpi-value {
  font-size: 24px;
  line-height: 30px;
}

.overview-time-panel {
  min-width: 0;
}

.overview-time-table {
  display: block;
}

.overview-time-attribution-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(360px, 1fr));
  gap: var(--am-section-gap);
}

@media (max-width: 1180px) {
  .overview-time-top,
  .overview-time-attribution-grid {
    grid-template-columns: 1fr;
  }
}
</style>
