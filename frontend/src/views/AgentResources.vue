<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ADrawer from 'ant-design-vue/es/drawer'
import AInput from 'ant-design-vue/es/input'
import ASelect from 'ant-design-vue/es/select'
import ASpin from 'ant-design-vue/es/spin'
import ASwitch from 'ant-design-vue/es/switch'
import AntTable from 'ant-design-vue/es/table'
import ATabs from 'ant-design-vue/es/tabs'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import message from 'ant-design-vue/es/message'
import {
  ApiOutlined,
  BookOutlined,
  EditOutlined,
  EyeOutlined,
  DatabaseOutlined,
  ReloadOutlined,
  SaveOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatDateTime,
  formatNumber,
  isStaticDemo,
  shortPath,
  type AgentMCPServerResource,
  type AgentMemoryResource,
  type AgentResourceOverview,
  type AgentSkillResource
} from '../api'
import PageHeader from '../components/PageHeader.vue'
import EmptyState from '../components/ui/EmptyState.vue'
import { useMessages } from '../i18n'

const ATable = AntTable as unknown as DefineComponent
const ATabPane = ATabs.TabPane
const ATypographyText = Typography.Text
const ATextarea = AInput.TextArea

const { t } = useMessages({
  en: {
    'title': 'Agent Resources',
    'subtitle': 'Inventory and management for local agent skills, MCP servers and memory files',
    'action.refresh': 'Refresh',
    'action.view': 'View',
    'action.edit': 'Edit',
    'action.reset': 'Reset',
    'action.save': 'Save',
    'action.close': 'Close',
    'filter.agent': 'Agent',
    'filter.allAgents': 'All agents',
    'metric.agent': 'Agent',
    'metric.agentNote.ready': 'Codex home available',
    'metric.agentNote.missing': 'Codex home missing',
    'metric.agentNote.multiple': '{count} agents discovered',
    'metric.skills': 'Skills',
    'metric.skillsNote': 'skills and rules',
    'metric.mcp': 'MCP',
    'metric.mcpNote': 'configured servers',
    'metric.memory': 'Memory',
    'metric.memoryNote': 'Markdown files',
    'tab.skills': 'Skills',
    'tab.mcp': 'MCP',
    'tab.memory': 'Memory',
    'skills.title': 'Skills',
    'skills.kicker': 'Local skill packages discovered from agent resource directories',
    'mcp.title': 'MCP Servers',
    'mcp.kicker': 'Configured MCP server entries from agent config files',
    'memory.title': 'Memory',
    'memory.kicker': 'Markdown memory files discovered from agent memory directories',
    'column.name': 'Name',
    'column.resource': 'Resource',
    'column.server': 'Server',
    'column.memory': 'Memory',
    'column.description': 'Description',
    'column.scope': 'Scope',
    'column.path': 'Path',
    'column.modified': 'Modified',
    'column.command': 'Command',
    'column.commandArgs': 'Command & args',
    'column.args': 'Args',
    'column.env': 'Env keys',
    'column.config': 'Config',
    'column.status': 'Status',
    'column.kind': 'Kind',
    'column.preview': 'Preview',
    'column.agent': 'Agent',
    'column.enabled': 'Enabled',
    'column.actions': 'Actions',
    'scope.system': 'system',
    'scope.user': 'user',
    'status.enabled': 'enabled',
    'status.disabled': 'disabled',
    'status.configured': 'configured',
    'status.incomplete': 'incomplete',
    'status.unsupported': 'unsupported',
    'memory.drawerTitle': 'Memory details',
    'memory.content': 'Content',
    'memory.path': 'Path',
    'memory.size': 'Size',
    'memory.modified': 'Modified',
    'memory.editable': 'Editable',
    'memory.readOnlyStatus': 'Read-only',
    'memory.unsaved': 'Unsaved',
    'memory.readOnly': 'This memory file is read-only from AgentMeter.',
    'message.toggleSaved': 'Resource state updated',
    'message.toggleFailed': 'Unable to update resource state',
    'message.memoryLoadedFallback': 'Full memory content is unavailable; showing read-only table preview.',
    'message.memoryDiscardConfirm': 'Discard unsaved memory changes?',
    'message.memorySaved': 'Memory saved',
    'message.memorySaveFailed': 'Unable to save memory',
    'empty.skills.title': 'No skills found',
    'empty.skills.text': 'No readable skills, commands, subagents, or rules were found.',
    'empty.mcp.title': 'No MCP servers configured',
    'empty.mcp.text': 'No MCP server entries were found in supported agent configs.',
    'empty.memory.title': 'No memory files found',
    'empty.memory.text': 'No editable Markdown resource files were found for supported agents.',
    'warnings.title': 'Warnings',
    'fallback.unknown': 'unknown',
    'fallback.none': 'none'
  },
  'zh-CN': {
    'title': 'Agent 资源',
    'subtitle': '管理本地 Agent 的 Skill、MCP server 和 Memory 文件',
    'action.refresh': '刷新',
    'action.view': '查看',
    'action.edit': '编辑',
    'action.reset': '重置',
    'action.save': '保存',
    'action.close': '关闭',
    'filter.agent': 'Agent',
    'filter.allAgents': '全部 Agent',
    'metric.agent': 'Agent',
    'metric.agentNote.ready': 'Codex 主目录可用',
    'metric.agentNote.missing': '缺少 Codex 主目录',
    'metric.agentNote.multiple': '发现 {count} 个 Agent',
    'metric.skills': 'Skill',
    'metric.skillsNote': 'Skill 和规则',
    'metric.mcp': 'MCP',
    'metric.mcpNote': '已配置 server',
    'metric.memory': 'Memory',
    'metric.memoryNote': 'Markdown 文件',
    'tab.skills': 'Skill',
    'tab.mcp': 'MCP',
    'tab.memory': 'Memory',
    'skills.title': 'Skill',
    'skills.kicker': '从 Agent 资源目录发现的本地 skill 包',
    'mcp.title': 'MCP Server',
    'mcp.kicker': '来自 Agent 配置文件的 MCP server 配置项',
    'memory.title': 'Memory',
    'memory.kicker': '从 Agent memory 目录发现的 Markdown 记忆文件',
    'column.name': '名称',
    'column.resource': '资源',
    'column.server': 'Server',
    'column.memory': 'Memory',
    'column.description': '描述',
    'column.scope': '范围',
    'column.path': '路径',
    'column.modified': '修改时间',
    'column.command': '命令',
    'column.commandArgs': '命令与参数',
    'column.args': '参数',
    'column.env': '环境变量键',
    'column.config': '配置',
    'column.status': '状态',
    'column.kind': '类型',
    'column.preview': '摘要',
    'column.agent': 'Agent',
    'column.enabled': '启用',
    'column.actions': '操作',
    'scope.system': '系统',
    'scope.user': '用户',
    'status.enabled': '已启用',
    'status.disabled': '已停用',
    'status.configured': '已配置',
    'status.incomplete': '不完整',
    'status.unsupported': '不支持',
    'memory.drawerTitle': 'Memory 详情',
    'memory.content': '内容',
    'memory.path': '路径',
    'memory.size': '大小',
    'memory.modified': '修改时间',
    'memory.editable': '可编辑',
    'memory.readOnlyStatus': '只读',
    'memory.unsaved': '未保存',
    'memory.readOnly': '此 Memory 文件不能从 AgentMeter 编辑。',
    'message.toggleSaved': '资源状态已更新',
    'message.toggleFailed': '无法更新资源状态',
    'message.memoryLoadedFallback': '无法加载完整 Memory 内容，正在只读显示表格摘要。',
    'message.memoryDiscardConfirm': '放弃未保存的 Memory 变更？',
    'message.memorySaved': 'Memory 已保存',
    'message.memorySaveFailed': '无法保存 Memory',
    'empty.skills.title': '暂无 Skill',
    'empty.skills.text': '未发现可读取的 skill、command、subagent 或 rule。',
    'empty.mcp.title': '暂无 MCP Server',
    'empty.mcp.text': '支持的 Agent 配置中未发现 MCP server 配置项。',
    'empty.memory.title': '暂无 Memory 文件',
    'empty.memory.text': '未发现支持编辑的 Agent Markdown 资源文件。',
    'warnings.title': '警告',
    'fallback.unknown': '未知',
    'fallback.none': '无'
  }
})

const loading = ref(true)
const overview = ref<AgentResourceOverview>({
  agents: [],
  skills: [],
  mcpServers: [],
  memories: [],
  warnings: []
})
const selectedAgentKind = ref('codex')
const togglingKeys = ref<string[]>([])
const memoryDrawerOpen = ref(false)
const memoryLoading = ref(false)
const memorySaving = ref(false)
const selectedMemory = ref<AgentMemoryResource | null>(null)
const memoryContent = ref('')
const originalMemoryContent = ref('')
const memoryDetailLoaded = ref(false)

const agentOptions = computed(() => {
  const kinds = new Set<string>()
  const options = overview.value.agents.map((item) => {
    kinds.add(item.kind)
    return { label: item.name || item.kind, value: item.kind }
  })
  for (const resource of [...overview.value.skills, ...overview.value.mcpServers, ...overview.value.memories]) {
    if (resource.agentKind && !kinds.has(resource.agentKind)) {
      kinds.add(resource.agentKind)
      options.push({ label: resource.agentKind, value: resource.agentKind })
    }
  }
  return [{ label: t('filter.allAgents'), value: 'all' }, ...options]
})

const selectedAgent = computed(() => overview.value.agents.find((item) => item.kind === selectedAgentKind.value))
const filteredSkills = computed(() => filterByAgent(overview.value.skills))
const filteredMcpServers = computed(() => filterByAgent(overview.value.mcpServers))
const filteredMemories = computed(() => filterByAgent(overview.value.memories))
const visibleWarnings = computed(() => {
  const warnings = selectedAgentKind.value === 'all'
    ? overview.value.warnings
    : [...overview.value.warnings, ...(selectedAgent.value?.warnings || [])]
  return [...new Set(warnings)]
})
const agentReady = computed(() => {
  if (selectedAgentKind.value === 'all') return overview.value.agents.some((item) => item.exists)
  return Boolean(selectedAgent.value?.exists)
})
const agentName = computed(() => {
  if (selectedAgentKind.value === 'all') return t('filter.allAgents')
  return selectedAgent.value?.name || selectedAgentKind.value || t('fallback.unknown')
})
const agentNote = computed(() => {
  if (selectedAgentKind.value === 'all') return t('metric.agentNote.multiple', { count: formatNumber(overview.value.agents.length) })
  return agentReady.value ? t('metric.agentNote.ready') : t('metric.agentNote.missing')
})
const rootPath = computed(() => selectedAgent.value?.rootPath || '')
const memoryDirty = computed(() => memoryContent.value !== originalMemoryContent.value)
const memoryCanEdit = computed(() => Boolean(selectedMemory.value?.canEdit) && memoryDetailLoaded.value && !isStaticDemo)

const skillColumns = computed(() => [
  { title: t('column.resource'), key: 'resource', width: 320 },
  { title: t('column.description'), dataIndex: 'description', key: 'description' },
  { title: t('column.path'), key: 'path', width: 320 },
  { title: t('column.modified'), dataIndex: 'modifiedAt', key: 'modifiedAt', width: 160 }
])

const mcpColumns = computed(() => [
  { title: t('column.server'), key: 'server', width: 260 },
  { title: t('column.commandArgs'), key: 'commandArgs' },
  { title: t('column.config'), key: 'config', width: 280 },
  { title: t('column.status'), key: 'status', width: 170 }
])

const memoryColumns = computed(() => [
  { title: t('column.memory'), key: 'memory', width: 300 },
  { title: t('column.preview'), dataIndex: 'preview', key: 'preview' },
  { title: t('column.path'), key: 'path', width: 340 },
  { title: t('column.actions'), key: 'actions', width: 104, align: 'right' }
])

async function load() {
  loading.value = true
  try {
    overview.value = await api.getAgentResources()
    if (!agentOptions.value.some((option) => option.value === selectedAgentKind.value)) {
      selectedAgentKind.value = agentOptions.value.some((option) => option.value === 'codex') ? 'codex' : 'all'
    }
  } finally {
    loading.value = false
  }
}

function filterByAgent<T extends { agentKind: string }>(items: T[]) {
  if (selectedAgentKind.value === 'all') return items
  return items.filter((item) => item.agentKind === selectedAgentKind.value)
}

function formatBytes(value: number) {
  if (!value) return '0 B'
  if (value < 1024) return `${formatNumber(value)} B`
  if (value < 1024 * 1024) return `${formatNumber(Math.round(value / 102.4) / 10)} KB`
  return `${formatNumber(Math.round(value / 1024 / 102.4) / 10)} MB`
}

function joined(values: string[]) {
  return values?.length ? values.join(' ') : t('fallback.none')
}

function tagColor(value: string) {
  if (value === 'configured' || value === 'primary' || value === 'enabled') return 'success'
  if (value === 'incomplete') return 'warning'
  if (value === 'disabled') return 'default'
  if (value === 'system') return 'processing'
  return 'default'
}

function statusLabel(value: string) {
  if (value === 'configured') return t('status.configured')
  if (value === 'enabled') return t('status.enabled')
  if (value === 'disabled') return t('status.disabled')
  if (value === 'incomplete') return t('status.incomplete')
  return value || t('fallback.unknown')
}

function resourceTypeLabel(record: AgentSkillResource) {
  return record.resourceType || 'skill'
}

function agentDisplay(agentKind: string) {
  return overview.value.agents.find((item) => item.kind === agentKind)?.name || agentKind || t('fallback.unknown')
}

function supportsToggle(record: { canToggle?: boolean }) {
  return Boolean(record.canToggle) && !isStaticDemo
}

function toggleTooltip(record: { canToggle?: boolean }) {
  if (supportsToggle(record)) return ''
  return t('status.unsupported')
}

function memoryActionLabel(record: { canEdit?: boolean }) {
  return record.canEdit && !isStaticDemo ? t('action.edit') : t('action.view')
}

function memoryStatusLabel(record: { canEdit?: boolean }) {
  return record.canEdit && !isStaticDemo ? t('memory.editable') : t('memory.readOnlyStatus')
}

function resourceEnabled(record: { enabled?: boolean }) {
  return record.enabled !== false
}

function resourceKey(record: { agentKind: string; name?: string; path?: string }) {
  return `${record.agentKind}:${record.path || record.name || ''}`
}

function isToggling(record: { agentKind: string; name?: string; path?: string }) {
  return togglingKeys.value.includes(resourceKey(record))
}

function setToggling(record: { agentKind: string; name?: string; path?: string }, value: boolean) {
  const key = resourceKey(record)
  togglingKeys.value = value ? [...togglingKeys.value, key] : togglingKeys.value.filter((item) => item !== key)
}

async function toggleSkill(record: AgentSkillResource, enabled: boolean) {
  if (!supportsToggle(record)) return
  setToggling(record, true)
  try {
    overview.value = await api.setAgentSkillEnabled({
      agentKind: record.agentKind,
      name: record.name,
      path: record.path,
      relativePath: record.relativePath,
      enabled
    })
    message.success(t('message.toggleSaved'))
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('message.toggleFailed'))
  } finally {
    setToggling(record, false)
  }
}

function onSkillSwitchChange(record: AgentSkillResource, checked: unknown) {
  toggleSkill(record, checked === true)
}

async function toggleMcpServer(record: AgentMCPServerResource, enabled: boolean) {
  if (!supportsToggle(record)) return
  setToggling(record, true)
  try {
    overview.value = await api.setAgentMCPServerEnabled({
      agentKind: record.agentKind,
      name: record.name,
      enabled
    })
    message.success(t('message.toggleSaved'))
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('message.toggleFailed'))
  } finally {
    setToggling(record, false)
  }
}

function onMcpSwitchChange(record: AgentMCPServerResource, checked: unknown) {
  toggleMcpServer(record, checked === true)
}

async function openMemory(record: AgentMemoryResource) {
  selectedMemory.value = record
  memoryContent.value = record.content || record.preview || ''
  originalMemoryContent.value = memoryContent.value
  memoryDetailLoaded.value = false
  memoryDrawerOpen.value = true
  memoryLoading.value = true
  try {
    const detail = await api.getAgentMemory({
      agentKind: record.agentKind,
      path: record.path,
      relativePath: record.relativePath
    })
    selectedMemory.value = detail
    memoryContent.value = detail.content || detail.preview || ''
    originalMemoryContent.value = memoryContent.value
    memoryDetailLoaded.value = true
  } catch {
    selectedMemory.value = { ...record, canEdit: false }
    message.info(t('message.memoryLoadedFallback'))
  } finally {
    memoryLoading.value = false
  }
}

function closeMemoryDrawer() {
  if (memoryDirty.value && !window.confirm(t('message.memoryDiscardConfirm'))) return
  memoryDrawerOpen.value = false
}

function resetMemoryContent() {
  memoryContent.value = originalMemoryContent.value
}

async function saveMemory() {
  if (!selectedMemory.value || !memoryCanEdit.value || !memoryDirty.value) return
  memorySaving.value = true
  try {
    const saved = await api.saveAgentMemory({
      agentKind: selectedMemory.value.agentKind,
      path: selectedMemory.value.path,
      relativePath: selectedMemory.value.relativePath,
      content: memoryContent.value
    })
    selectedMemory.value = saved
    memoryContent.value = saved.content || memoryContent.value
    originalMemoryContent.value = memoryContent.value
    memoryDetailLoaded.value = true
    const memoryIndex = overview.value.memories.findIndex((item) => (
      item.agentKind === saved.agentKind && item.path === saved.path
    ))
    if (memoryIndex >= 0) {
      const memories = [...overview.value.memories]
      memories.splice(memoryIndex, 1, saved)
      overview.value = { ...overview.value, memories }
    }
    message.success(t('message.memorySaved'))
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('message.memorySaveFailed'))
  } finally {
    memorySaving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page agent-resources-page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')">
      <template #actions>
        <a-select
          v-model:value="selectedAgentKind"
          class="agent-resource-agent-filter"
          :options="agentOptions"
          :aria-label="t('filter.agent')"
        />
        <a-button @click="load">
          <template #icon>
            <ReloadOutlined />
          </template>
          {{ t('action.refresh') }}
        </a-button>
      </template>
    </PageHeader>

    <a-spin :spinning="loading">
      <section class="metric-strip agent-resource-metrics">
        <div class="metric-strip-item metric-primary">
          <div class="metric-strip-head">
            <div class="metric-label">{{ t('metric.agent') }}</div>
            <DatabaseOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ agentName }}</div>
          <a-tooltip :title="rootPath">
            <div class="metric-strip-note">
              {{ agentNote }}<template v-if="rootPath"> · {{ shortPath(rootPath) }}</template>
            </div>
          </a-tooltip>
        </div>
        <div class="metric-strip-item metric-success">
          <div class="metric-strip-head">
            <div class="metric-label">{{ t('metric.skills') }}</div>
            <ToolOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(filteredSkills.length) }}</div>
          <div class="metric-strip-note">{{ t('metric.skillsNote') }}</div>
        </div>
        <div class="metric-strip-item metric-info">
          <div class="metric-strip-head">
            <div class="metric-label">{{ t('metric.mcp') }}</div>
            <ApiOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(filteredMcpServers.length) }}</div>
          <div class="metric-strip-note">{{ t('metric.mcpNote') }}</div>
        </div>
        <div class="metric-strip-item metric-warning">
          <div class="metric-strip-head">
            <div class="metric-label">{{ t('metric.memory') }}</div>
            <BookOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(filteredMemories.length) }}</div>
          <div class="metric-strip-note">{{ t('metric.memoryNote') }}</div>
        </div>
      </section>

      <section v-if="visibleWarnings.length" class="index-result-warnings agent-resource-warnings">
        <div class="metadata-label">{{ t('warnings.title') }}</div>
        <ul>
          <li v-for="warning in visibleWarnings" :key="warning">{{ warning }}</li>
        </ul>
      </section>

      <section class="panel">
        <div class="panel-body">
          <a-tabs class="agent-resource-tabs">
            <a-tab-pane key="skills" :tab="t('tab.skills')">
              <div class="panel-header agent-resource-inner-header">
                <div>
                  <h2 class="panel-title">{{ t('skills.title') }}</h2>
                  <div class="panel-kicker">{{ t('skills.kicker') }}</div>
                </div>
                <ToolOutlined class="panel-header-icon" />
              </div>
              <a-table
                :columns="skillColumns"
                :data-source="filteredSkills"
                :pagination="{ pageSize: 12, showSizeChanger: true }"
                :scroll="{ x: 920 }"
                row-key="path"
                size="small"
                table-layout="fixed"
              >
                <template #emptyText>
                  <EmptyState :title="t('empty.skills.title')" :text="t('empty.skills.text')" compact :icon="ToolOutlined" />
                </template>
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'resource'">
                    <div class="resource-main">
                      <div class="resource-title-row">
                        <div class="resource-name">{{ record.name }}</div>
                      </div>
                      <div class="resource-subtitle">{{ record.title }}</div>
                      <div class="resource-meta-line">
                        <a-tag class="status-tag">{{ agentDisplay(record.agentKind) }}</a-tag>
                        <a-tag class="status-tag" :color="tagColor(resourceTypeLabel(record))">{{ resourceTypeLabel(record) }}</a-tag>
                        <a-tag class="status-tag" :color="record.system ? tagColor('system') : 'default'">
                          {{ record.system ? t('scope.system') : t('scope.user') }}
                        </a-tag>
                        <span class="resource-switch-meta">
                          <a-tooltip :title="toggleTooltip(record)">
                            <a-switch
                              size="small"
                              :checked="resourceEnabled(record)"
                              :disabled="!supportsToggle(record)"
                              :loading="isToggling(record)"
                              @change="(checked) => onSkillSwitchChange(record, checked)"
                            />
                          </a-tooltip>
                          <span>{{ resourceEnabled(record) ? t('status.enabled') : t('status.disabled') }}</span>
                        </span>
                      </div>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'description'">
                    <a-tooltip :title="record.description" placement="topLeft">
                      <div class="resource-detail-text resource-detail-text-two-line">
                        {{ record.description || t('fallback.none') }}
                      </div>
                    </a-tooltip>
                  </template>
                  <template v-else-if="column.key === 'path'">
                    <div class="resource-path-block">
                      <a-tooltip :title="record.path" placement="topLeft">
                        <a-typography-text class="mono resource-ellipsis" :ellipsis="{ tooltip: record.path }">
                          {{ record.relativePath }}
                        </a-typography-text>
                      </a-tooltip>
                      <div class="timeline-event-raw">{{ formatBytes(record.sizeBytes) }}</div>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'modifiedAt'">
                    {{ formatDateTime(record.modifiedAt) }}
                  </template>
                </template>
              </a-table>
            </a-tab-pane>

            <a-tab-pane key="mcp" :tab="t('tab.mcp')">
              <div class="panel-header agent-resource-inner-header">
                <div>
                  <h2 class="panel-title">{{ t('mcp.title') }}</h2>
                  <div class="panel-kicker">{{ t('mcp.kicker') }}</div>
                </div>
                <ApiOutlined class="panel-header-icon" />
              </div>
              <a-table
                :columns="mcpColumns"
                :data-source="filteredMcpServers"
                :pagination="{ pageSize: 12, showSizeChanger: true }"
                :scroll="{ x: 900 }"
                :row-key="resourceKey"
                size="small"
                table-layout="fixed"
              >
                <template #emptyText>
                  <EmptyState :title="t('empty.mcp.title')" :text="t('empty.mcp.text')" compact :icon="ApiOutlined" />
                </template>
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'server'">
                    <div class="resource-main">
                      <div class="resource-name">{{ record.name }}</div>
                      <div class="resource-meta-line">
                        <a-tag class="status-tag">{{ agentDisplay(record.agentKind) }}</a-tag>
                      </div>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'commandArgs'">
                    <div class="resource-command-block">
                      <a-tooltip :title="record.command" placement="topLeft">
                        <a-typography-text class="mono resource-ellipsis" :ellipsis="{ tooltip: record.command }">
                          {{ record.command || t('fallback.none') }}
                        </a-typography-text>
                      </a-tooltip>
                      <a-typography-text class="mono resource-ellipsis resource-args-line" :ellipsis="{ tooltip: joined(record.args) }">
                        {{ joined(record.args) }}
                      </a-typography-text>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'config'">
                    <div class="resource-path-block">
                      <a-tooltip :title="record.configPath" placement="topLeft">
                        <a-typography-text class="mono resource-ellipsis" :ellipsis="{ tooltip: record.configPath }">
                          {{ shortPath(record.configPath) }}
                        </a-typography-text>
                      </a-tooltip>
                      <div class="resource-tag-list resource-env-list">
                        <a-tag v-for="key in record.envKeys" :key="key" class="status-tag">{{ key }}</a-tag>
                        <span v-if="!record.envKeys.length" class="muted">{{ t('fallback.none') }}</span>
                      </div>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'status'">
                    <div class="resource-status-stack">
                      <a-tooltip :title="record.warning">
                        <a-tag class="status-tag" :color="tagColor(record.status)">
                          {{ statusLabel(record.status) }}
                        </a-tag>
                      </a-tooltip>
                      <span class="resource-switch-meta">
                        <a-tooltip :title="toggleTooltip(record)">
                          <a-switch
                            size="small"
                            :checked="resourceEnabled(record)"
                            :disabled="!supportsToggle(record)"
                            :loading="isToggling(record)"
                            @change="(checked) => onMcpSwitchChange(record, checked)"
                          />
                        </a-tooltip>
                        <span>{{ resourceEnabled(record) ? t('status.enabled') : t('status.disabled') }}</span>
                      </span>
                    </div>
                  </template>
                </template>
              </a-table>
            </a-tab-pane>

            <a-tab-pane key="memory" :tab="t('tab.memory')">
              <div class="panel-header agent-resource-inner-header">
                <div>
                  <h2 class="panel-title">{{ t('memory.title') }}</h2>
                  <div class="panel-kicker">{{ t('memory.kicker') }}</div>
                </div>
                <BookOutlined class="panel-header-icon" />
              </div>
              <a-table
                :columns="memoryColumns"
                :data-source="filteredMemories"
                :pagination="{ pageSize: 12, showSizeChanger: true }"
                :scroll="{ x: 920 }"
                row-key="path"
                size="small"
                table-layout="fixed"
              >
                <template #emptyText>
                  <EmptyState :title="t('empty.memory.title')" :text="t('empty.memory.text')" compact :icon="BookOutlined" />
                </template>
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'memory'">
                    <div class="resource-main">
                      <div class="resource-name">{{ record.title || record.name }}</div>
                      <div class="resource-subtitle">{{ record.name }}</div>
                      <div class="resource-meta-line">
                        <a-tag class="status-tag">{{ agentDisplay(record.agentKind) }}</a-tag>
                        <a-tag class="status-tag" :color="tagColor(record.kind)">{{ record.kind }}</a-tag>
                        <a-tag class="status-tag" :color="record.canEdit && !isStaticDemo ? tagColor('enabled') : 'default'">
                          {{ memoryStatusLabel(record) }}
                        </a-tag>
                      </div>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'preview'">
                    <div class="resource-detail-text resource-detail-text-two-line">
                      {{ record.preview || t('fallback.none') }}
                    </div>
                  </template>
                  <template v-else-if="column.key === 'path'">
                    <div class="resource-path-block">
                      <a-tooltip :title="record.path" placement="topLeft">
                        <a-typography-text class="mono resource-ellipsis" :ellipsis="{ tooltip: record.path }">
                          {{ record.relativePath }}
                        </a-typography-text>
                      </a-tooltip>
                      <div class="timeline-event-raw">
                        {{ formatBytes(record.sizeBytes) }} · {{ formatDateTime(record.modifiedAt) }}
                      </div>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'actions'">
                    <a-button size="small" type="text" @click="openMemory(record)">
                      <template #icon>
                        <EditOutlined v-if="record.canEdit && !isStaticDemo" />
                        <EyeOutlined v-else />
                      </template>
                      {{ memoryActionLabel(record) }}
                    </a-button>
                  </template>
                </template>
              </a-table>
            </a-tab-pane>
          </a-tabs>
        </div>
      </section>
    </a-spin>

    <a-drawer
      class="agent-memory-drawer"
      :open="memoryDrawerOpen"
      :width="'min(720px, 100vw)'"
      placement="right"
      @close="closeMemoryDrawer"
    >
      <template #title>
        {{ t('memory.drawerTitle') }}
      </template>
      <a-spin :spinning="memoryLoading">
        <div v-if="selectedMemory" class="memory-detail">
          <div class="memory-detail-head">
            <div>
              <div class="resource-name">{{ selectedMemory.title || selectedMemory.name }}</div>
              <div class="timeline-event-raw">{{ agentDisplay(selectedMemory.agentKind) }} · {{ selectedMemory.kind }}</div>
            </div>
            <div class="memory-status-line">
              <a-tag v-if="memoryDirty" class="status-tag" color="warning">{{ t('memory.unsaved') }}</a-tag>
              <a-tooltip :title="memoryCanEdit ? '' : t('memory.readOnly')">
                <a-tag class="status-tag" :color="memoryCanEdit ? tagColor('enabled') : 'default'">
                  {{ memoryCanEdit ? t('memory.editable') : t('memory.readOnlyStatus') }}
                </a-tag>
              </a-tooltip>
            </div>
          </div>
          <div class="memory-meta-grid">
            <div class="memory-field is-wide">
              <div class="metadata-label">{{ t('memory.path') }}</div>
              <a-tooltip :title="selectedMemory.path" placement="topLeft">
                <a-typography-text class="mono" :ellipsis="{ tooltip: selectedMemory.path }">
                  {{ selectedMemory.relativePath }}
                </a-typography-text>
              </a-tooltip>
            </div>
            <div class="memory-field">
              <div class="metadata-label">{{ t('memory.size') }}</div>
              <div class="memory-meta-value">{{ formatBytes(selectedMemory.sizeBytes) }}</div>
            </div>
            <div class="memory-field">
              <div class="metadata-label">{{ t('memory.modified') }}</div>
              <div class="memory-meta-value">{{ formatDateTime(selectedMemory.modifiedAt) }}</div>
            </div>
          </div>
          <div class="memory-field">
            <div class="metadata-label">{{ t('memory.content') }}</div>
            <a-textarea
              v-model:value="memoryContent"
              class="memory-editor"
              :auto-size="{ minRows: 14, maxRows: 26 }"
              :readonly="!memoryCanEdit"
            />
          </div>
          <div class="memory-actions">
            <a-button :disabled="!memoryDirty || memorySaving" @click="resetMemoryContent">{{ t('action.reset') }}</a-button>
            <a-button @click="closeMemoryDrawer">{{ t('action.close') }}</a-button>
            <a-button
              type="primary"
              :disabled="!memoryCanEdit || !memoryDirty || memoryLoading"
              :loading="memorySaving"
              @click="saveMemory"
            >
              <template #icon>
                <SaveOutlined />
              </template>
              {{ t('action.save') }}
            </a-button>
          </div>
        </div>
      </a-spin>
    </a-drawer>
  </div>
</template>

<style scoped>
.agent-resource-metrics {
  grid-template-columns: repeat(4, minmax(170px, 1fr));
}

.agent-resource-agent-filter {
  min-width: 180px;
}

.agent-resource-warnings {
  margin-bottom: var(--am-section-gap);
}

.agent-resource-tabs :deep(.ant-tabs-nav) {
  margin: 0 0 12px;
}

.agent-resource-inner-header {
  margin: -2px 0 12px;
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius-sm);
}

.agent-resources-page :deep(.ant-table-wrapper),
.agent-resources-page :deep(.ant-table-cell) {
  min-width: 0;
}

.agent-resources-page :deep(.ant-table-cell) {
  overflow: hidden;
  vertical-align: top;
}

.resource-main,
.resource-command-block,
.resource-path-block,
.resource-status-stack {
  display: grid;
  gap: 6px;
  min-width: 0;
  overflow: hidden;
}

.resource-title-row {
  align-items: flex-start;
  display: flex;
  gap: 8px;
  justify-content: space-between;
  min-width: 0;
}

.resource-name {
  color: var(--am-text);
  font-weight: 720;
  overflow-wrap: anywhere;
}

.resource-subtitle,
.resource-detail-text {
  color: var(--am-text-soft);
  font-size: 12px;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.resource-detail-text-two-line {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
}

.resource-ellipsis {
  display: block;
  max-width: 100%;
  min-width: 0;
}

.resource-args-line {
  color: var(--am-text-soft);
}

.resource-meta-line,
.resource-switch-meta {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.resource-switch-meta {
  color: var(--am-text-soft);
  font-size: 12px;
  line-height: 18px;
}

.resource-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  min-width: 0;
}

.resource-meta-line .status-tag,
.resource-tag-list .status-tag,
.resource-status-stack .status-tag {
  margin-inline-end: 0;
}

.resource-env-list {
  max-height: 48px;
  overflow: hidden;
}

.memory-detail {
  display: grid;
  gap: 16px;
}

.memory-detail-head {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.memory-status-line {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: flex-end;
}

.memory-meta-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.memory-field {
  display: grid;
  gap: 6px;
}

.memory-field.is-wide {
  grid-column: 1 / -1;
}

.memory-meta-value {
  color: var(--am-text-soft);
  font-size: 13px;
  line-height: 20px;
}

.memory-editor {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  line-height: 1.55;
}

.memory-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: flex-end;
}

@media (max-width: 1180px) {
  .agent-resource-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .agent-resource-metrics {
    grid-template-columns: 1fr;
  }

  .memory-detail-head {
    display: grid;
  }

  .memory-status-line {
    justify-content: flex-start;
  }

  .memory-meta-grid {
    grid-template-columns: 1fr;
  }
}
</style>
