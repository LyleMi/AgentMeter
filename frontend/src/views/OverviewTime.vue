<script setup lang="ts">
import { computed, onMounted, ref, type Component } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import {
  ClockCircleOutlined,
  FieldTimeOutlined,
  ReloadOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatDuration,
  formatNumber,
  sessionLabel,
  type Overview,
  type Session,
  type ToolTimeUsage
} from '../api'
import PageHeader from '../components/PageHeader.vue'
import { useMessages } from '../i18n'
import SlowSessionsTable from './time/SlowSessionsTable.vue'
import TimeAttributionTables from './time/TimeAttributionTables.vue'
import TimeComposition from './time/TimeComposition.vue'
import TimeKpiGrid from './time/TimeKpiGrid.vue'
import ToolDurationLeaders from './time/ToolDurationLeaders.vue'

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

interface TimeKpiCard {
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

const kpiCards = computed<TimeKpiCard[]>(() => [
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

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function slowSessionRow(record: Session) {
  return { class: 'overview-session-row is-clickable-row', onClick: () => openSession(record.id) }
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
          <TimeComposition
            :title="t('composition.title')"
            :kicker="t('composition.kicker')"
            :total-label="t('composition.total')"
            :total-value="formatDuration(wallDurationMs)"
            :segments="compositionSegments"
            :format-duration="formatDuration"
            :format-percent="formatPercent"
          />

          <TimeKpiGrid :cards="kpiCards" />
        </section>

        <ToolDurationLeaders
          :title="t('tools.title')"
          :kicker="t('tools.kicker')"
          :network-hint="t('tools.networkHint')"
          :empty-title="t('empty.tools')"
          :empty-text="t('empty.text')"
          :fallback-unknown="t('fallback.unknown')"
          :network-likely-label="t('status.networkLikely')"
          :not-network-label="t('status.notNetwork')"
          :columns="toolColumns"
          :rows="rankedToolLeaders"
          :has-rows="hasToolLeaders"
          :row-key="toolRowKey"
        />

        <TimeAttributionTables
          :agent-title="t('agent.title')"
          :agent-kicker="t('agent.kicker')"
          :model-title="t('model.title')"
          :model-kicker="t('model.kicker')"
          :empty-agent-title="t('empty.agents')"
          :empty-model-title="t('empty.models')"
          :empty-text="t('empty.text')"
          :agent-columns="agentColumns"
          :model-columns="modelColumns"
          :agent-rows="rankedAgentTimeUsage"
          :model-rows="rankedModelTimeUsage"
          :has-agent-rows="hasAgentTimeUsage"
          :has-model-rows="hasModelTimeUsage"
          :fallback-unknown="t('fallback.unknown')"
        />

        <SlowSessionsTable
          :title="t('sessions.title')"
          :kicker="t('sessions.kicker')"
          :empty-title="t('empty.sessions')"
          :empty-text="t('empty.text')"
          :open-label="t('column.open')"
          :columns="slowSessionColumns"
          :rows="slowSessions"
          :has-rows="hasSlowSessions"
          :fallback-unknown="t('fallback.unknown')"
          :open-session="openSession"
          :row-props="slowSessionRow"
        />
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

@media (max-width: 1180px) {
  .overview-time-top {
    grid-template-columns: 1fr;
  }
}
</style>
