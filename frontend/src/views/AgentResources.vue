<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATabs from 'ant-design-vue/es/tabs'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import {
  ApiOutlined,
  BookOutlined,
  DatabaseOutlined,
  ReloadOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import { api, formatDateTime, formatNumber, shortPath, type AgentResourceOverview } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EmptyState from '../components/ui/EmptyState.vue'
import { useMessages } from '../i18n'

const ATable = AntTable as unknown as DefineComponent
const ATabPane = ATabs.TabPane
const ATypographyText = Typography.Text

const { t } = useMessages({
  en: {
    'title': 'Agent Resources',
    'subtitle': 'Read-only inventory of local agent skills, MCP servers and memory files',
    'action.refresh': 'Refresh',
    'metric.agent': 'Agent',
    'metric.agentNote.ready': 'Codex home available',
    'metric.agentNote.missing': 'Codex home missing',
    'metric.skills': 'Skills',
    'metric.skillsNote': 'SKILL.md packages',
    'metric.mcp': 'MCP',
    'metric.mcpNote': 'configured servers',
    'metric.memory': 'Memory',
    'metric.memoryNote': 'Markdown files',
    'tab.skills': 'Skills',
    'tab.mcp': 'MCP',
    'tab.memory': 'Memory',
    'skills.title': 'Skills',
    'skills.kicker': 'Local Codex skill packages discovered from SKILL.md files',
    'mcp.title': 'MCP Servers',
    'mcp.kicker': 'Configured Codex MCP server entries from config.toml',
    'memory.title': 'Memory',
    'memory.kicker': 'Markdown memory files under the Codex memory directory',
    'column.name': 'Name',
    'column.description': 'Description',
    'column.scope': 'Scope',
    'column.path': 'Path',
    'column.modified': 'Modified',
    'column.command': 'Command',
    'column.args': 'Args',
    'column.env': 'Env keys',
    'column.status': 'Status',
    'column.kind': 'Kind',
    'column.preview': 'Preview',
    'scope.system': 'system',
    'scope.user': 'user',
    'status.configured': 'configured',
    'status.incomplete': 'incomplete',
    'empty.skills.title': 'No skills found',
    'empty.skills.text': 'No readable SKILL.md files were found under the Codex skills directory.',
    'empty.mcp.title': 'No MCP servers configured',
    'empty.mcp.text': 'No mcp_servers entries were found in Codex config.toml.',
    'empty.memory.title': 'No memory files found',
    'empty.memory.text': 'No Markdown files were found under the Codex memory directory.',
    'warnings.title': 'Warnings',
    'fallback.unknown': 'unknown',
    'fallback.none': 'none'
  },
  'zh-CN': {
    'title': 'Agent 资源',
    'subtitle': '只读查看本地 Agent 的 Skill、MCP server 和 Memory 文件',
    'action.refresh': '刷新',
    'metric.agent': 'Agent',
    'metric.agentNote.ready': 'Codex 主目录可用',
    'metric.agentNote.missing': '缺少 Codex 主目录',
    'metric.skills': 'Skill',
    'metric.skillsNote': 'SKILL.md 包',
    'metric.mcp': 'MCP',
    'metric.mcpNote': '已配置 server',
    'metric.memory': 'Memory',
    'metric.memoryNote': 'Markdown 文件',
    'tab.skills': 'Skill',
    'tab.mcp': 'MCP',
    'tab.memory': 'Memory',
    'skills.title': 'Skill',
    'skills.kicker': '从 SKILL.md 文件发现的本地 Codex skill 包',
    'mcp.title': 'MCP Server',
    'mcp.kicker': '来自 Codex config.toml 的 MCP server 配置项',
    'memory.title': 'Memory',
    'memory.kicker': 'Codex memory 目录下的 Markdown 记忆文件',
    'column.name': '名称',
    'column.description': '描述',
    'column.scope': '范围',
    'column.path': '路径',
    'column.modified': '修改时间',
    'column.command': '命令',
    'column.args': '参数',
    'column.env': '环境变量键',
    'column.status': '状态',
    'column.kind': '类型',
    'column.preview': '摘要',
    'scope.system': '系统',
    'scope.user': '用户',
    'status.configured': '已配置',
    'status.incomplete': '不完整',
    'empty.skills.title': '暂无 Skill',
    'empty.skills.text': 'Codex skills 目录下未发现可读取的 SKILL.md 文件。',
    'empty.mcp.title': '暂无 MCP Server',
    'empty.mcp.text': 'Codex config.toml 中未发现 mcp_servers 配置项。',
    'empty.memory.title': '暂无 Memory 文件',
    'empty.memory.text': 'Codex memory 目录下未发现 Markdown 文件。',
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

const agent = computed(() => overview.value.agents[0])
const agentReady = computed(() => Boolean(agent.value?.exists))
const agentNote = computed(() => (agentReady.value ? t('metric.agentNote.ready') : t('metric.agentNote.missing')))
const rootPath = computed(() => agent.value?.rootPath || '')

const skillColumns = computed(() => [
  { title: t('column.name'), key: 'name', width: 230 },
  { title: t('column.description'), dataIndex: 'description', key: 'description' },
  { title: t('column.scope'), key: 'scope', width: 104 },
  { title: t('column.path'), key: 'path', width: 260 },
  { title: t('column.modified'), dataIndex: 'modifiedAt', key: 'modifiedAt', width: 150 }
])

const mcpColumns = computed(() => [
  { title: t('column.name'), dataIndex: 'name', key: 'name', width: 180 },
  { title: t('column.command'), key: 'command' },
  { title: t('column.args'), key: 'args', width: 180 },
  { title: t('column.env'), key: 'env', width: 180 },
  { title: t('column.status'), key: 'status', width: 120 }
])

const memoryColumns = computed(() => [
  { title: t('column.name'), key: 'name', width: 220 },
  { title: t('column.kind'), dataIndex: 'kind', key: 'kind', width: 110 },
  { title: t('column.preview'), dataIndex: 'preview', key: 'preview' },
  { title: t('column.path'), key: 'path', width: 260 },
  { title: t('column.modified'), dataIndex: 'modifiedAt', key: 'modifiedAt', width: 150 }
])

async function load() {
  loading.value = true
  try {
    overview.value = await api.getAgentResources()
  } finally {
    loading.value = false
  }
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
  if (value === 'configured' || value === 'primary') return 'success'
  if (value === 'incomplete') return 'warning'
  if (value === 'system') return 'processing'
  return 'default'
}

onMounted(load)
</script>

<template>
  <div class="page agent-resources-page">
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

    <a-spin :spinning="loading">
      <section class="metric-strip agent-resource-metrics">
        <div class="metric-strip-item metric-primary">
          <div class="metric-strip-head">
            <div class="metric-label">{{ t('metric.agent') }}</div>
            <DatabaseOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ agent?.name || t('fallback.unknown') }}</div>
          <a-tooltip :title="rootPath">
            <div class="metric-strip-note">{{ agentNote }} · {{ shortPath(rootPath) }}</div>
          </a-tooltip>
        </div>
        <div class="metric-strip-item metric-success">
          <div class="metric-strip-head">
            <div class="metric-label">{{ t('metric.skills') }}</div>
            <ToolOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(overview.skills.length) }}</div>
          <div class="metric-strip-note">{{ t('metric.skillsNote') }}</div>
        </div>
        <div class="metric-strip-item metric-info">
          <div class="metric-strip-head">
            <div class="metric-label">{{ t('metric.mcp') }}</div>
            <ApiOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(overview.mcpServers.length) }}</div>
          <div class="metric-strip-note">{{ t('metric.mcpNote') }}</div>
        </div>
        <div class="metric-strip-item metric-warning">
          <div class="metric-strip-head">
            <div class="metric-label">{{ t('metric.memory') }}</div>
            <BookOutlined class="metric-strip-icon" />
          </div>
          <div class="metric-strip-value">{{ formatNumber(overview.memories.length) }}</div>
          <div class="metric-strip-note">{{ t('metric.memoryNote') }}</div>
        </div>
      </section>

      <section v-if="overview.warnings.length" class="index-result-warnings agent-resource-warnings">
        <div class="metadata-label">{{ t('warnings.title') }}</div>
        <ul>
          <li v-for="warning in overview.warnings" :key="warning">{{ warning }}</li>
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
                :data-source="overview.skills"
                :pagination="{ pageSize: 12, showSizeChanger: true }"
                :scroll="{ x: 980 }"
                row-key="path"
                size="small"
                table-layout="fixed"
              >
                <template #emptyText>
                  <EmptyState :title="t('empty.skills.title')" :text="t('empty.skills.text')" compact :icon="ToolOutlined" />
                </template>
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'name'">
                    <div class="resource-name">{{ record.name }}</div>
                    <div class="timeline-event-raw">{{ record.title }}</div>
                  </template>
                  <template v-else-if="column.key === 'scope'">
                    <a-tag class="status-tag" :color="record.system ? tagColor('system') : 'default'">
                      {{ record.system ? t('scope.system') : t('scope.user') }}
                    </a-tag>
                  </template>
                  <template v-else-if="column.key === 'path'">
                    <a-tooltip :title="record.path" placement="topLeft">
                      <a-typography-text class="mono" :ellipsis="{ tooltip: record.path }">
                        {{ record.relativePath }}
                      </a-typography-text>
                    </a-tooltip>
                    <div class="timeline-event-raw">{{ formatBytes(record.sizeBytes) }}</div>
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
                :data-source="overview.mcpServers"
                :pagination="{ pageSize: 12, showSizeChanger: true }"
                :scroll="{ x: 900 }"
                row-key="name"
                size="small"
                table-layout="fixed"
              >
                <template #emptyText>
                  <EmptyState :title="t('empty.mcp.title')" :text="t('empty.mcp.text')" compact :icon="ApiOutlined" />
                </template>
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'command'">
                    <a-tooltip :title="record.command" placement="topLeft">
                      <a-typography-text class="mono" :ellipsis="{ tooltip: record.command }">
                        {{ record.command || t('fallback.none') }}
                      </a-typography-text>
                    </a-tooltip>
                    <div class="timeline-event-raw">{{ shortPath(record.configPath) }}</div>
                  </template>
                  <template v-else-if="column.key === 'args'">
                    <a-typography-text class="mono" :ellipsis="{ tooltip: joined(record.args) }">
                      {{ joined(record.args) }}
                    </a-typography-text>
                  </template>
                  <template v-else-if="column.key === 'env'">
                    <div class="resource-tag-list">
                      <a-tag v-for="key in record.envKeys" :key="key" class="status-tag">{{ key }}</a-tag>
                      <span v-if="!record.envKeys.length" class="muted">{{ t('fallback.none') }}</span>
                    </div>
                  </template>
                  <template v-else-if="column.key === 'status'">
                    <a-tooltip :title="record.warning">
                      <a-tag class="status-tag" :color="tagColor(record.status)">
                        {{ record.status === 'configured' ? t('status.configured') : t('status.incomplete') }}
                      </a-tag>
                    </a-tooltip>
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
                :data-source="overview.memories"
                :pagination="{ pageSize: 12, showSizeChanger: true }"
                :scroll="{ x: 980 }"
                row-key="path"
                size="small"
                table-layout="fixed"
              >
                <template #emptyText>
                  <EmptyState :title="t('empty.memory.title')" :text="t('empty.memory.text')" compact :icon="BookOutlined" />
                </template>
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'name'">
                    <div class="resource-name">{{ record.title || record.name }}</div>
                    <div class="timeline-event-raw">{{ record.name }}</div>
                  </template>
                  <template v-else-if="column.key === 'kind'">
                    <a-tag class="status-tag" :color="tagColor(record.kind)">{{ record.kind }}</a-tag>
                  </template>
                  <template v-else-if="column.key === 'preview'">
                    <div class="resource-preview">{{ record.preview || t('fallback.none') }}</div>
                  </template>
                  <template v-else-if="column.key === 'path'">
                    <a-tooltip :title="record.path" placement="topLeft">
                      <a-typography-text class="mono" :ellipsis="{ tooltip: record.path }">
                        {{ record.relativePath }}
                      </a-typography-text>
                    </a-tooltip>
                    <div class="timeline-event-raw">{{ formatBytes(record.sizeBytes) }}</div>
                  </template>
                  <template v-else-if="column.key === 'modifiedAt'">
                    {{ formatDateTime(record.modifiedAt) }}
                  </template>
                </template>
              </a-table>
            </a-tab-pane>
          </a-tabs>
        </div>
      </section>
    </a-spin>
  </div>
</template>

<style scoped>
.agent-resource-metrics {
  grid-template-columns: repeat(4, minmax(170px, 1fr));
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

.resource-name {
  color: var(--am-text);
  font-weight: 720;
  overflow-wrap: anywhere;
}

.resource-preview {
  color: var(--am-text-soft);
  font-size: 12px;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.resource-tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
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
}
</style>
