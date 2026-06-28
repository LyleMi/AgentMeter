<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASelect from 'ant-design-vue/es/select'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import { EyeOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import ToolCallDetailDrawer from '../../components/ToolCallDetailDrawer.vue'
import ToolInputInline from '../../components/ToolInputInline.vue'
import FilterToolbar from '../../components/ui/FilterToolbar.vue'
import { formatDateTime, formatDuration, formatNumber, sessionFullLabel, sessionLabel, type ToolCall } from '../../api'
import { useMessages } from '../../i18n'
import { sourceDisplay, sourceFilterOptions } from '../../presentation/sourceIdentity'
import { statusClass, statusColor } from '../../presentation/status'
import {
  commandSummary,
  commandTooltip,
  invokedCommand,
  inputContext,
  projectContext,
  projectTooltip,
  rawSourceContext
} from './shellTool'
import { DEFAULT_SORT, useToolCallExplorer, type ToolCallExplorerMode } from './useToolCallExplorer'

const props = defineProps<{
  mode: ToolCallExplorerMode
}>()

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const explorer = useToolCallExplorer(props.mode)
const { t } = useMessages({
  en: {
    'title.all': 'Recent Tool Calls',
    'title.shell': 'Shell Commands',
    'kicker.all': 'Individual calls with parsed input, output, raw events and session context',
    'kicker.shell': 'Shell and terminal tool calls by source, command, session and project',
    'count.all.matching': '{count} matching calls',
    'count.all.sorted': '{count} sorted calls',
    'count.all.recent': '{count} recent calls',
    'count.shell.matching': '{count} matching shell commands',
    'count.shell.sorted': '{count} sorted shell commands',
    'count.shell.recent': '{count} recent shell commands',
    'column.started': 'Started',
    'column.tool': 'Tool',
    'column.agentTool': 'Source / Tool',
    'column.command': 'Command / Input',
    'column.status': 'Status',
    'column.duration': 'Duration',
    'column.session': 'Session',
    'column.input': 'Input',
    'column.output': 'Output',
    'column.project': 'Project',
    'filter.agent': 'Source',
    'filter.command': 'Invoked command',
    'filter.tool.all': 'Tool',
    'filter.tool.shell': 'Shell tool',
    'filter.from': 'From',
    'filter.to': 'To',
    'filter.fromAria': 'Started from',
    'filter.toAria': 'Started to',
    'sort.recent': 'Recent first',
    'sort.durationDesc': 'Duration high to low',
    'sort.durationAsc': 'Duration low to high',
    'action.reset': 'Reset',
    'action.refresh': 'Refresh',
    'label.rawSource': 'raw',
    'empty.all.loading': 'Loading tool calls...',
    'empty.all.none': 'No tool calls indexed',
    'empty.shell.loading': 'Loading shell commands...',
    'empty.shell.none': 'No shell command calls indexed',
    'tooltip.viewDetails': 'View details',
    'fallback.unknown': 'unknown',
    'fallback.none': '-'
  },
  'zh-CN': {
    'title.all': '最近工具调用',
    'title.shell': 'Shell 命令',
    'kicker.all': '包含解析输入、输出、原始事件和会话上下文的单次调用',
    'kicker.shell': '按来源、命令、会话和项目查看 Shell 与终端工具调用',
    'count.all.matching': '{count} 个匹配调用',
    'count.all.sorted': '{count} 个已排序调用',
    'count.all.recent': '{count} 个最近调用',
    'count.shell.matching': '{count} 个匹配 Shell 命令',
    'count.shell.sorted': '{count} 个已排序 Shell 命令',
    'count.shell.recent': '{count} 个最近 Shell 命令',
    'column.started': '开始',
    'column.tool': '工具',
    'column.agentTool': '来源 / 工具',
    'column.command': '命令 / 输入',
    'column.status': '状态',
    'column.duration': '耗时',
    'column.session': '会话',
    'column.input': '输入',
    'column.output': '输出',
    'column.project': '项目',
    'filter.agent': '来源',
    'filter.command': '调用命令',
    'filter.tool.all': '工具',
    'filter.tool.shell': 'Shell 工具',
    'filter.from': '从',
    'filter.to': '到',
    'filter.fromAria': '开始时间从',
    'filter.toAria': '开始时间到',
    'sort.recent': '最近优先',
    'sort.durationDesc': '耗时从高到低',
    'sort.durationAsc': '耗时从低到高',
    'action.reset': '重置',
    'action.refresh': '刷新',
    'label.rawSource': '原始',
    'empty.all.loading': '正在加载工具调用...',
    'empty.all.none': '暂无已索引工具调用',
    'empty.shell.loading': '正在加载 Shell 命令...',
    'empty.shell.none': '暂无已索引 Shell 命令调用',
    'tooltip.viewDetails': '查看详情',
    'fallback.unknown': '未知',
    'fallback.none': '-'
  }
})

const modeKey = computed(() => props.mode)
const callColumns = computed(() =>
  props.mode === 'shell'
    ? [
        { title: t('column.started'), dataIndex: 'startedAt', key: 'startedAt', width: 132 },
        { title: t('column.agentTool'), dataIndex: 'agentName', key: 'agentTool', width: 180 },
        { title: t('column.command'), dataIndex: 'inputSummary', key: 'command', width: 470 },
        { title: t('column.status'), dataIndex: 'status', key: 'status', width: 105 },
        { title: t('column.duration'), dataIndex: 'durationMs', key: 'duration', width: 96, align: 'right' },
        { title: t('column.session'), dataIndex: 'sessionId', key: 'session', width: 120 },
        { title: t('column.project'), dataIndex: 'projectPath', key: 'project', width: 260 },
        { title: '', key: 'detail', width: 56, align: 'right' }
      ]
    : [
        { title: t('column.started'), dataIndex: 'startedAt', key: 'startedAt', width: 140 },
        { title: t('column.tool'), dataIndex: 'toolName', key: 'toolName', width: 140 },
        { title: t('column.status'), dataIndex: 'status', key: 'status', width: 105 },
        { title: t('column.duration'), dataIndex: 'durationMs', key: 'duration', width: 96, align: 'right' },
        { title: t('column.session'), dataIndex: 'sessionId', key: 'session', width: 96 },
        { title: t('column.input'), dataIndex: 'inputSummary', key: 'input', width: 440 },
        { title: t('column.output'), dataIndex: 'outputSummary', key: 'output', width: 280 },
        { title: '', key: 'detail', width: 56, align: 'right' }
      ]
)
const toolOptions = computed(() => {
  const values = props.mode === 'shell' ? [...explorer.availableTools.value].sort((left, right) => left.toolName.localeCompare(right.toolName)) : explorer.availableTools.value
  return values.map((item) => ({
    value: item.toolName,
    label: props.mode === 'shell' ? `${item.toolName || t('fallback.unknown')} (${formatNumber(item.calls)})` : item.toolName || t('fallback.unknown')
  }))
})
const commandOptions = computed(() =>
  explorer.commandOptions.value.map((item) => ({
    value: item.command,
    label: `${item.command} (${formatNumber(item.calls)})`
  }))
)
const agentOptions = computed(() => sourceFilterOptions(explorer.agents.value, t('fallback.unknown')))
const sortOptions = computed(() => [
  { value: DEFAULT_SORT, label: t('sort.recent') },
  { value: 'duration_desc', label: t('sort.durationDesc') },
  { value: 'duration_asc', label: t('sort.durationAsc') }
])
const tableLocale = computed(() => ({ emptyText: explorer.callLoading.value ? t(`empty.${modeKey.value}.loading`) : t(`empty.${modeKey.value}.none`) }))
const hasActiveFilters = computed(() => Boolean(explorer.toolFilter.value || explorer.commandFilter.value || explorer.agentFilter.value || explorer.fromFilter.value || explorer.toFilter.value))
const countText = computed(() => {
  const visible = formatNumber(explorer.toolCalls.value.length)
  if (hasActiveFilters.value) return t(`count.${modeKey.value}.matching`, { count: visible })
  if (explorer.sortFilter.value !== DEFAULT_SORT) return t(`count.${modeKey.value}.sorted`, { count: visible })
  return t(`count.${modeKey.value}.recent`, { count: visible })
})
const tableClass = computed(() => (props.mode === 'shell' ? 'dense-table tool-shell-table' : 'dense-table tool-call-detail-table'))
const tableScroll = computed(() => ({ x: props.mode === 'shell' ? 1450 : 1350 }))

function callSessionLabel(call: ToolCall) {
  return sessionLabel({ id: call.sessionId, sessionKey: call.sessionKey || '', codexSessionId: call.codexSessionId })
}

function callSessionFullLabel(call: ToolCall) {
  return sessionFullLabel({ id: call.sessionId, sessionKey: call.sessionKey || '', codexSessionId: call.codexSessionId })
}

function callSessionShort(call: ToolCall) {
  return `#${formatNumber(call.sessionId)}`
}

function callSessionTooltip(call: ToolCall) {
  const context = call.projectPath || call.rawSourcePath || ''
  return context ? `${callSessionFullLabel(call)}\n${context}` : callSessionFullLabel(call)
}

function sourceInfo(call: ToolCall) {
  return sourceDisplay(call, t('fallback.unknown'))
}

function commandName(call: ToolCall) {
  return invokedCommand(call)
}
</script>

<template>
  <section class="panel">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ t(`title.${modeKey}`) }}</h2>
        <div class="panel-kicker">{{ t(`kicker.${modeKey}`) }}</div>
      </div>
      <span class="row-count">{{ countText }}</span>
    </div>
    <div class="panel-body">
      <FilterToolbar compact>
        <template #left>
          <a-select
            v-model:value="explorer.agentFilter.value"
            class="control-medium"
            allow-clear
            :placeholder="t('filter.agent')"
            :options="agentOptions"
            :loading="explorer.loading.value"
            @change="explorer.updateFilters('agent')"
          />
          <a-select
            v-model:value="explorer.toolFilter.value"
            class="control-medium"
            allow-clear
            :placeholder="t(`filter.tool.${modeKey}`)"
            :options="toolOptions"
            :loading="explorer.loading.value || explorer.toolLoading.value"
            @change="() => explorer.updateFilters()"
          />
          <a-select
            v-if="mode === 'shell'"
            v-model:value="explorer.commandFilter.value"
            class="control-medium shell-command-filter"
            allow-clear
            show-search
            option-filter-prop="label"
            :placeholder="t('filter.command')"
            :options="commandOptions"
            :loading="explorer.callLoading.value"
            @change="() => explorer.updateFilters()"
          />
          <label class="inline-field tool-time-filter">
            <span>{{ t('filter.from') }}</span>
            <input v-model="explorer.fromFilter.value" class="native-date-input" type="datetime-local" :aria-label="t('filter.fromAria')" @change="() => explorer.updateFilters()" />
          </label>
          <label class="inline-field tool-time-filter">
            <span>{{ t('filter.to') }}</span>
            <input v-model="explorer.toFilter.value" class="native-date-input" type="datetime-local" :aria-label="t('filter.toAria')" @change="() => explorer.updateFilters()" />
          </label>
          <a-select
            v-model:value="explorer.sortFilter.value"
            class="control-medium"
            :options="sortOptions"
            @change="() => explorer.updateFilters()"
          />
          <a-button @click="explorer.resetFilters">{{ t('action.reset') }}</a-button>
        </template>
        <template #right>
          <a-button @click="explorer.load">
            <template #icon>
              <ReloadOutlined />
            </template>
            {{ t('action.refresh') }}
          </a-button>
        </template>
      </FilterToolbar>

      <a-table
        :class="tableClass"
        :columns="callColumns"
        :data-source="explorer.toolCalls.value"
        row-key="id"
        size="small"
        :loading="explorer.callLoading.value"
        :locale="tableLocale"
        :pagination="{ pageSize: 20, showSizeChanger: true }"
        :scroll="tableScroll"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'startedAt'">{{ formatDateTime(record.startedAt) }}</template>

          <template v-else-if="column.key === 'toolName'">
            <a-typography-text :ellipsis="{ tooltip: record.toolName }">
              {{ record.toolName || t('fallback.unknown') }}
            </a-typography-text>
          </template>

          <template v-else-if="column.key === 'agentTool'">
            <a-typography-text class="source-identity-name" :ellipsis="{ tooltip: sourceInfo(record).title }">
              {{ sourceInfo(record).label }}
            </a-typography-text>
            <div class="source-identity-meta">{{ sourceInfo(record).secondary || '-' }}</div>
            <div class="tool-shell-meta">
              <a-tag class="model-lite-tag">{{ record.toolName || t('fallback.unknown') }}</a-tag>
            </div>
          </template>

          <template v-else-if="column.key === 'command'">
            <div v-if="commandName(record)" class="tool-shell-command-label">
              <a-tag class="tool-shell-command-tag">{{ commandName(record) }}</a-tag>
            </div>
            <a-tooltip :title="commandTooltip(record, t('fallback.none'))" placement="topLeft">
              <pre class="tool-shell-command mono">{{ commandSummary(record) || t('fallback.none') }}</pre>
            </a-tooltip>
            <div v-if="inputContext(record)" class="tool-shell-input">{{ inputContext(record) }}</div>
          </template>

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

          <template v-else-if="column.key === 'session'">
            <a-tooltip :title="callSessionTooltip(record)" placement="topLeft">
              <a-button type="link" size="small" class="tool-call-session-link" @click="explorer.openSession(record.sessionId)">
                {{ callSessionShort(record) }}
              </a-button>
            </a-tooltip>
            <div class="tool-call-session-meta">{{ mode === 'shell' ? callSessionLabel(record) : sourceInfo(record).label }}</div>
          </template>

          <template v-else-if="column.key === 'input'">
            <ToolInputInline :call="record" />
          </template>

          <template v-else-if="column.key === 'output'">
            <a-typography-text :ellipsis="{ tooltip: record.outputSummary || record.error }">
              {{ record.outputSummary || record.error || '-' }}
            </a-typography-text>
          </template>

          <template v-else-if="column.key === 'project'">
            <a-tooltip :title="projectTooltip(record, t('fallback.none'))" placement="topLeft">
              <a-typography-text class="path-cell" :ellipsis="{ tooltip: projectTooltip(record, t('fallback.none')) }">
                {{ projectContext(record, t('fallback.none')) }}
              </a-typography-text>
            </a-tooltip>
            <div v-if="rawSourceContext(record, t('label.rawSource'))" class="tool-shell-meta mono">{{ rawSourceContext(record, t('label.rawSource')) }}</div>
          </template>

          <template v-else-if="column.key === 'detail'">
            <a-tooltip :title="t('tooltip.viewDetails')">
              <a-button type="text" size="small" @click="explorer.openToolCall(record)">
                <template #icon>
                  <EyeOutlined />
                </template>
              </a-button>
            </a-tooltip>
          </template>
        </template>
      </a-table>
    </div>

    <ToolCallDetailDrawer
      :open="Boolean(explorer.selectedToolCall.value)"
      :call="explorer.selectedToolCall.value"
      @close="explorer.closeToolCall"
      @open-session="explorer.openSession"
    />
  </section>
</template>

<style scoped>
.shell-command-filter {
  width: 170px;
}

.tool-shell-table :deep(.ant-table-tbody > tr > td) {
  vertical-align: top;
}

.tool-shell-command-label {
  margin-bottom: 4px;
}

.tool-shell-command-tag {
  max-width: 100%;
  margin-inline-end: 0;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
  text-overflow: ellipsis;
  text-transform: none;
  vertical-align: top;
}

.tool-shell-command {
  display: -webkit-box;
  overflow: hidden;
  max-width: 100%;
  min-height: 34px;
  max-height: 52px;
  margin: 0;
  padding: 6px 7px;
  color: var(--am-text);
  font-size: 11px;
  line-height: 16px;
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--am-surface);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.tool-shell-input,
.tool-shell-meta {
  max-width: 100%;
  margin-top: 4px;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 11px;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
