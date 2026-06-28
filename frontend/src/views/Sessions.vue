<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import ASelect from 'ant-design-vue/es/select'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import { ArrowRightOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons-vue'
import { api } from '../api/client'
import type { Session } from '../api/types'
import PageHeader from '../components/PageHeader.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { formatCost, formatDateTime, formatDuration, formatNumber, sessionLabel, shortPath } from '../presentation/formatters'
import { statusClass, statusColor } from '../presentation/status'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const router = useRouter()
const { t } = useMessages({
  en: {
    'title': 'Sessions',
    'subtitle': 'Compact local session workbench for traces, pricing, and indexing state',
    'action.refresh': 'Refresh',
    'action.apply': 'Apply',
    'action.reset': 'Reset',
    'filter.searchPlaceholder': 'Search project, model or file',
    'filter.agentPlaceholder': 'Agent',
    'filter.modelPlaceholder': 'Model',
    'rowCount.matching': 'matching',
    'rowCount.indexed': 'indexed',
    'empty.loading': 'Loading sessions...',
    'empty.filtered': 'No sessions match the current filters',
    'empty.unindexed': 'No indexed sessions yet. Configure a source and run indexing to populate local history.',
    'column.session': 'Session',
    'column.agent': 'Agent',
    'column.project': 'Project',
    'column.model': 'Model',
    'column.tokens': 'Tokens',
    'column.cost': 'Cost',
    'column.tools': 'Tools',
    'column.wall': 'Wall',
    'column.status': 'Status',
    'status.parsePrefix': 'parse',
    'status.unpriced': 'unpriced',
    'fallback.unknown': 'unknown',
    'fallback.indexMessage': 'No index message recorded',
    'tooltip.openSession': 'Open session'
  },
  'zh-CN': {
    'title': '会话',
    'subtitle': '用于查看轨迹、费用和索引状态的本地会话工作台',
    'action.refresh': '刷新',
    'action.apply': '应用',
    'action.reset': '重置',
    'filter.searchPlaceholder': '搜索项目、模型或文件',
    'filter.agentPlaceholder': 'Agent',
    'filter.modelPlaceholder': '模型',
    'rowCount.matching': '个匹配',
    'rowCount.indexed': '个已索引',
    'empty.loading': '正在加载会话...',
    'empty.filtered': '没有会话符合当前筛选条件',
    'empty.unindexed': '还没有已索引的会话。请配置来源并运行索引以填充本地历史。',
    'column.session': '会话',
    'column.agent': 'Agent',
    'column.project': '项目',
    'column.model': '模型',
    'column.tokens': 'Token',
    'column.cost': '费用',
    'column.tools': '工具',
    'column.wall': '耗时',
    'column.status': '状态',
    'status.parsePrefix': '解析',
    'status.unpriced': '未定价',
    'fallback.unknown': '未知',
    'fallback.indexMessage': '没有记录索引消息',
    'tooltip.openSession': '打开会话'
  }
})
const sessionRows = useAsyncResource<{ sessions: Session[]; catalogSessions: Session[] }>({
  sessions: [],
  catalogSessions: []
})
const loading = sessionRows.loading
const sessions = computed(() => sessionRows.data.value.sessions)
const catalogSessions = computed(() => sessionRows.data.value.catalogSessions)
const search = ref('')
const model = ref<string | undefined>()
const agent = ref<string | undefined>()

const columns = computed(() => [
  { title: t('column.session'), dataIndex: 'sessionKey', key: 'identity', width: 250 },
  { title: t('column.agent'), dataIndex: 'agentName', key: 'agent', width: 132 },
  { title: t('column.project'), dataIndex: 'projectPath', key: 'projectPath' },
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 90 },
  { title: t('column.tokens'), dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 100, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 100, align: 'right' },
  { title: t('column.tools'), dataIndex: 'toolCallCount', key: 'tools', width: 70, align: 'right' },
  { title: t('column.wall'), dataIndex: 'wallDurationMs', key: 'wall', width: 76, align: 'right' },
  { title: t('column.status'), dataIndex: 'parseStatus', key: 'status', width: 170 },
  { title: '', key: 'open', width: 44, align: 'right' }
])

const hasActiveFilters = computed(() => Boolean(search.value.trim() || model.value || agent.value))

const modelOptions = computed(() => {
  const source = catalogSessions.value.length ? catalogSessions.value : sessions.value
  const values = new Set(source.map((item) => item.model).filter(Boolean))
  return [...values].sort().map((value) => ({ value, label: value }))
})

const agentOptions = computed(() => {
  const source = catalogSessions.value.length ? catalogSessions.value : sessions.value
  const values = new Map<string, string>()
  for (const item of source) {
    if (item.agentKind) values.set(item.agentKind, item.agentName || item.agentKind)
  }
  return [...values.entries()].sort((left, right) => left[1].localeCompare(right[1])).map(([value, label]) => ({ value, label }))
})

const rowCountText = computed(() => {
  const visible = formatNumber(sessions.value.length)
  const indexed = formatNumber(catalogSessions.value.length || sessions.value.length)
  if (hasActiveFilters.value) return `${visible} ${t('rowCount.matching')} / ${indexed} ${t('rowCount.indexed')}`
  return `${visible} ${t('rowCount.indexed')}`
})

const emptyText = computed(() => {
  if (loading.value) return t('empty.loading')
  if (hasActiveFilters.value) return t('empty.filtered')
  return t('empty.unindexed')
})

async function load() {
  await sessionRows.run(async () => {
    const filters = { search: search.value.trim() || undefined, model: model.value, agent: agent.value, limit: 300 }
    const filtered = api.listSessions(filters)
    const catalog = hasActiveFilters.value ? api.listSessions({ limit: 300 }) : filtered
    const [nextSessions, nextCatalog] = await Promise.all([filtered, catalog])
    return { sessions: nextSessions || [], catalogSessions: nextCatalog || [] }
  })
}

function resetFilters() {
  search.value = ''
  model.value = undefined
  agent.value = undefined
  load()
}

function indexStatusHint(record: Session) {
  return record.lastIndexedScanMessage || record.rawSourcePath || t('fallback.indexMessage')
}

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function sessionRow(record: Session) {
  return { class: 'sessions-table-row', onClick: () => openSession(record.id) }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')">
      <template #actions>
        <a-button @click="load">
          <template #icon>
            <ReloadOutlined />
          </template>
          {{ t('action.refresh') }}
        </a-button>
      </template>
    </PageHeader>

    <section class="panel">
      <div class="panel-body">
        <div class="toolbar sessions-toolbar">
          <div class="toolbar-left sessions-toolbar-controls">
            <a-input
              v-model:value="search"
              class="sessions-search control-wide"
              allow-clear
              :placeholder="t('filter.searchPlaceholder')"
              @press-enter="load"
            >
              <template #prefix>
                <SearchOutlined />
              </template>
            </a-input>
            <a-select
              v-model:value="agent"
              class="sessions-model-filter control-medium"
              allow-clear
              :placeholder="t('filter.agentPlaceholder')"
              :options="agentOptions"
              @change="load"
            />
            <a-select
              v-model:value="model"
              class="sessions-model-filter control-medium"
              allow-clear
              :placeholder="t('filter.modelPlaceholder')"
              :options="modelOptions"
              @change="load"
            />
            <a-button type="primary" @click="load">{{ t('action.apply') }}</a-button>
            <a-button @click="resetFilters">{{ t('action.reset') }}</a-button>
          </div>
          <div class="toolbar-right muted sessions-row-count">{{ rowCountText }}</div>
        </div>

        <a-table
          class="sessions-table"
          :columns="columns"
          :data-source="sessions"
          :loading="loading"
          :locale="{ emptyText }"
          :pagination="{ pageSize: 20, showSizeChanger: true }"
          :scroll="{ x: 1080 }"
          table-layout="fixed"
          row-key="id"
          size="small"
          :custom-row="sessionRow"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'identity'">
              <a-typography-text class="mono path-cell" :ellipsis="{ tooltip: sessionLabel(record) }">
                {{ sessionLabel(record) }}
              </a-typography-text>
              <div class="timeline-event-raw">{{ formatDateTime(record.startedAt) }}</div>
            </template>
            <template v-else-if="column.key === 'agent'">
              <a-tag class="model-lite-tag">{{ record.agentName || record.agentKind || t('fallback.unknown') }}</a-tag>
              <div class="timeline-event-raw">{{ record.agentKind || '-' }}</div>
            </template>
            <template v-else-if="column.key === 'projectPath'">
              <a-tooltip :title="record.projectPath" placement="topLeft">
                <span class="sessions-project-path">{{ shortPath(record.projectPath) }}</span>
              </a-tooltip>
              <div class="timeline-event-raw">{{ shortPath(record.rawSourcePath) }}</div>
            </template>
            <template v-else-if="column.key === 'model'">
              <a-tooltip :title="record.model" placement="topLeft">
                <span class="model-name">{{ record.model || t('fallback.unknown') }}</span>
              </a-tooltip>
              <div class="timeline-event-raw">{{ record.modelProvider || '-' }}</div>
            </template>
            <template v-else-if="column.key === 'tokens'">
              <span class="number-cell">{{ formatNumber(record.tokenUsage.totalTokens) }}</span>
            </template>
            <template v-else-if="column.key === 'cost'">
              <span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span>
            </template>
            <template v-else-if="column.key === 'tools'">
              <span class="number-cell">{{ formatNumber(record.toolCallCount) }}</span>
            </template>
            <template v-else-if="column.key === 'wall'">
              <span class="number-cell">{{ formatDuration(record.wallDurationMs) }}</span>
            </template>
            <template v-else-if="column.key === 'status'">
              <div class="timeline-event-head">
                <a-tag class="status-tag parse-status-tag" :class="statusClass(record.parseStatus)" :color="statusColor(record.parseStatus)">
                  {{ t('status.parsePrefix') }} {{ record.parseStatus || t('fallback.unknown') }}
                </a-tag>
                <a-tooltip :title="indexStatusHint(record)" placement="topLeft">
                  <a-tag
                    class="status-tag parse-status-tag"
                    :class="statusClass(record.lastIndexedScanStatus)"
                    :color="statusColor(record.lastIndexedScanStatus)"
                  >
                    {{ record.lastIndexedScanStatus || t('fallback.unknown') }}
                  </a-tag>
                </a-tooltip>
                <a-tag v-if="record.unpriced" class="status-tag model-status-tag" color="warning">{{ t('status.unpriced') }}</a-tag>
              </div>
            </template>
            <template v-else-if="column.key === 'open'">
              <a-tooltip :title="t('tooltip.openSession')">
                <a-button type="text" size="small" @click.stop="openSession(record.id)">
                  <template #icon>
                    <ArrowRightOutlined />
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
