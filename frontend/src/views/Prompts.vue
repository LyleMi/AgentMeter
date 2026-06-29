<script setup lang="ts">
import { computed, onMounted, reactive, ref, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import AInputNumber from 'ant-design-vue/es/input-number'
import AModal from 'ant-design-vue/es/modal'
import APopconfirm from 'ant-design-vue/es/popconfirm'
import ASelect from 'ant-design-vue/es/select'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import ATooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import message from 'ant-design-vue/es/message'
import {
  ArrowRightOutlined,
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeInvisibleOutlined,
  FileTextOutlined,
  ReloadOutlined,
  SaveOutlined,
  SearchOutlined
} from '@ant-design/icons-vue'
import { api } from '../api/client'
import type { PromptExample, PromptSuggestion, SavedPrompt, Session } from '../api/types'
import PageHeader from '../components/PageHeader.vue'
import EmptyState from '../components/ui/EmptyState.vue'
import { useAsyncResource } from '../composables/useAsyncResource'
import { useMessages } from '../i18n'
import { formatDateTime, formatNumber, projectDisplay, sessionLabel, shortPath } from '../presentation/formatters'
import { sourceDisplay, sourceFilterOptions } from '../presentation/sourceIdentity'

const ATable = AntTable as unknown as DefineComponent
const ATypographyParagraph = Typography.Paragraph
const ATypographyText = Typography.Text
const ATextarea = AInput.TextArea

interface PromptPageData {
  suggestions: PromptSuggestion[]
  saved: SavedPrompt[]
  catalogSessions: Session[]
}

const router = useRouter()
const { t } = useMessages({
  en: {
    'title': 'Prompts',
    'subtitle': 'Frequent user messages promoted into reusable local prompts',
    'action.refresh': 'Refresh',
    'action.apply': 'Apply',
    'action.reset': 'Reset',
    'action.copy': 'Copy',
    'action.save': 'Save',
    'action.edit': 'Edit',
    'action.delete': 'Delete',
    'action.ignore': 'Ignore',
    'action.openSession': 'Open session',
    'filter.searchPlaceholder': 'Search prompt text',
    'filter.agentPlaceholder': 'Source',
    'filter.projectPlaceholder': 'Project path',
    'filter.minCount': 'Min count',
    'metric.suggestions': 'Suggestions',
    'metric.suggestionsNote': 'candidate clusters',
    'metric.saved': 'Saved',
    'metric.savedNote': 'reusable prompts',
    'metric.occurrences': 'Occurrences',
    'metric.occurrencesNote': 'matched user messages',
    'metric.copies': 'Copies',
    'metric.copiesNote': 'saved prompt copies',
    'suggestions.title': 'Suggestions',
    'suggestions.kicker': 'Repeated user messages grouped by exact and near-duplicate matches',
    'saved.title': 'Saved Prompts',
    'saved.kicker': 'Local prompt library stored in AgentMeter',
    'column.prompt': 'Prompt',
    'column.count': 'Count',
    'column.context': 'Context',
    'column.lastUsed': 'Last used',
    'column.actions': 'Actions',
    'column.title': 'Title',
    'column.updated': 'Updated',
    'tag.exact': 'exact',
    'tag.near': 'near',
    'tag.variants': '{count} variants',
    'tag.sessions': '{count} sessions',
    'empty.suggestions.title': 'No prompt suggestions',
    'empty.suggestions.text': 'Repeated user messages appear after local sessions are indexed.',
    'empty.saved.title': 'No saved prompts',
    'empty.saved.text': 'Save a suggestion to build a reusable local prompt library.',
    'modal.createTitle': 'Save prompt',
    'modal.editTitle': 'Edit prompt',
    'form.title': 'Title',
    'form.content': 'Prompt',
    'message.copied': 'Prompt copied',
    'message.saved': 'Prompt saved',
    'message.updated': 'Prompt updated',
    'message.deleted': 'Prompt deleted',
    'message.ignored': 'Suggestion ignored',
    'message.copyFailed': 'Unable to copy prompt',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    'title': 'Prompt',
    'subtitle': '把高频用户输入沉淀为可复用的本地 prompt',
    'action.refresh': '刷新',
    'action.apply': '应用',
    'action.reset': '重置',
    'action.copy': '复制',
    'action.save': '保存',
    'action.edit': '编辑',
    'action.delete': '删除',
    'action.ignore': '忽略',
    'action.openSession': '打开会话',
    'filter.searchPlaceholder': '搜索 prompt 文本',
    'filter.agentPlaceholder': '来源',
    'filter.projectPlaceholder': '项目路径',
    'filter.minCount': '最小次数',
    'metric.suggestions': '建议',
    'metric.suggestionsNote': '候选聚类',
    'metric.saved': '已保存',
    'metric.savedNote': '可复用 prompt',
    'metric.occurrences': '出现次数',
    'metric.occurrencesNote': '匹配的用户输入',
    'metric.copies': '复制',
    'metric.copiesNote': '已保存 prompt 复制次数',
    'suggestions.title': '建议',
    'suggestions.kicker': '按完全重复和近似重复归并的高频用户输入',
    'saved.title': '已保存 Prompt',
    'saved.kicker': '保存在 AgentMeter 本地的 prompt 库',
    'column.prompt': 'Prompt',
    'column.count': '次数',
    'column.context': '上下文',
    'column.lastUsed': '最近使用',
    'column.actions': '操作',
    'column.title': '标题',
    'column.updated': '更新时间',
    'tag.exact': '精确',
    'tag.near': '近似',
    'tag.variants': '{count} 个变体',
    'tag.sessions': '{count} 个会话',
    'empty.suggestions.title': '暂无 prompt 建议',
    'empty.suggestions.text': '索引本地会话后，重复出现的用户输入会显示在这里。',
    'empty.saved.title': '暂无已保存 prompt',
    'empty.saved.text': '保存一个建议后即可形成可复用的本地 prompt 库。',
    'modal.createTitle': '保存 prompt',
    'modal.editTitle': '编辑 prompt',
    'form.title': '标题',
    'form.content': 'Prompt',
    'message.copied': 'Prompt 已复制',
    'message.saved': 'Prompt 已保存',
    'message.updated': 'Prompt 已更新',
    'message.deleted': 'Prompt 已删除',
    'message.ignored': '建议已忽略',
    'message.copyFailed': '无法复制 prompt',
    'fallback.unknown': '未知'
  }
})

const promptData = useAsyncResource<PromptPageData>({
  suggestions: [],
  saved: [],
  catalogSessions: []
})
const loading = promptData.loading
const search = ref('')
const agent = ref<string | undefined>()
const project = ref('')
const minCount = ref(2)
const editorOpen = ref(false)
const editorSaving = ref(false)
const editingPrompt = ref<SavedPrompt | null>(null)
const editor = reactive({
  title: '',
  content: '',
  sourceSuggestionKey: ''
})

const suggestions = computed(() => promptData.data.value.suggestions)
const savedPrompts = computed(() => promptData.data.value.saved)
const catalogSessions = computed(() => promptData.data.value.catalogSessions)
const totalOccurrences = computed(() => suggestions.value.reduce((sum, item) => sum + item.count, 0))
const totalCopies = computed(() => savedPrompts.value.reduce((sum, item) => sum + item.copyCount, 0))
const hasActiveFilters = computed(() => Boolean(search.value.trim() || agent.value || project.value.trim() || minCount.value !== 2))

const suggestionColumns = computed(() => [
  { title: t('column.prompt'), dataIndex: 'text', key: 'prompt' },
  { title: t('column.count'), dataIndex: 'count', key: 'count', width: 128, align: 'right' },
  { title: t('column.context'), key: 'context', width: 230 },
  { title: t('column.lastUsed'), dataIndex: 'lastUsedAt', key: 'lastUsedAt', width: 132 },
  { title: t('column.actions'), key: 'actions', width: 128, align: 'right' }
])

const savedColumns = computed(() => [
  { title: t('column.title'), dataIndex: 'title', key: 'title' },
  { title: t('column.count'), dataIndex: 'copyCount', key: 'copyCount', width: 82, align: 'right' },
  { title: t('column.updated'), dataIndex: 'updatedAt', key: 'updatedAt', width: 132 },
  { title: t('column.actions'), key: 'actions', width: 126, align: 'right' }
])

const agentOptions = computed(() => {
  const examples = suggestions.value.flatMap((suggestion) => suggestion.examples)
  return sourceFilterOptions([...catalogSessions.value, ...examples], t('fallback.unknown'))
})

async function load() {
  await promptData.run(async () => {
    const filters = {
      search: search.value.trim() || undefined,
      agent: agent.value,
      project: project.value.trim() || undefined,
      minCount: minCount.value || undefined,
      limit: 100
    }
    const [nextSuggestions, nextSaved, nextSessions] = await Promise.all([
      api.getPromptSuggestions(filters),
      api.listSavedPrompts(),
      api.listSessions({ limit: 300 })
    ])
    return {
      suggestions: nextSuggestions || [],
      saved: nextSaved || [],
      catalogSessions: nextSessions || []
    }
  })
}

function resetFilters() {
  search.value = ''
  agent.value = undefined
  project.value = ''
  minCount.value = 2
  load()
}

function updateSavedPrompt(next: SavedPrompt) {
  const current = promptData.data.value
  const index = current.saved.findIndex((item) => item.id === next.id)
  const saved = [...current.saved]
  if (index >= 0) saved.splice(index, 1, next)
  promptData.data.value = { ...current, saved }
}

async function copyPrompt(text: string, saved?: SavedPrompt) {
  try {
    await navigator.clipboard.writeText(text)
    if (saved) {
      const updated = await api.recordPromptCopy(saved.id)
      updateSavedPrompt(updated)
    }
    message.success(t('message.copied'))
  } catch {
    message.error(t('message.copyFailed'))
  }
}

function promptTitleFromText(text: string) {
  const firstLine = text.trim().split(/\r?\n/)[0] || t('modal.createTitle')
  return firstLine.length > 58 ? `${firstLine.slice(0, 57)}...` : firstLine
}

function openSavePrompt(suggestion: PromptSuggestion) {
  editingPrompt.value = null
  editor.title = promptTitleFromText(suggestion.text)
  editor.content = suggestion.text
  editor.sourceSuggestionKey = suggestion.key
  editorOpen.value = true
}

function openEditPrompt(prompt: SavedPrompt) {
  editingPrompt.value = prompt
  editor.title = prompt.title
  editor.content = prompt.content
  editor.sourceSuggestionKey = prompt.sourceSuggestionKey || ''
  editorOpen.value = true
}

async function submitEditor() {
  editorSaving.value = true
  try {
    const input = {
      title: editor.title.trim(),
      content: editor.content.trim(),
      sourceSuggestionKey: editor.sourceSuggestionKey || undefined
    }
    if (editingPrompt.value) {
      await api.updateSavedPrompt(editingPrompt.value.id, input)
      message.success(t('message.updated'))
    } else {
      await api.savePrompt(input)
      message.success(t('message.saved'))
    }
    editorOpen.value = false
    await load()
  } finally {
    editorSaving.value = false
  }
}

async function deletePrompt(prompt: SavedPrompt) {
  await api.deleteSavedPrompt(prompt.id)
  message.success(t('message.deleted'))
  await load()
}

async function ignoreSuggestion(suggestion: PromptSuggestion) {
  await api.ignorePromptSuggestion(suggestion.key)
  message.success(t('message.ignored'))
  await load()
}

function exampleSource(example?: PromptExample) {
  return sourceDisplay(example || {}, t('fallback.unknown'))
}

function exampleProject(example?: PromptExample) {
  return projectDisplay(example?.projectPath)
}

function openExample(example?: PromptExample) {
  if (!example?.sessionId) return
  router.push(`/sessions/${example.sessionId}`)
}

function variantsLabel(suggestion: PromptSuggestion) {
  return t('tag.variants', { count: formatNumber(suggestion.variantCount) })
}

function sessionsLabel(suggestion: PromptSuggestion) {
  return t('tag.sessions', { count: formatNumber(suggestion.sessionCount) })
}

onMounted(load)
</script>

<template>
  <div class="page prompts-page">
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

    <section class="metric-strip prompts-metric-strip">
      <div class="metric-strip-item metric-primary">
        <div class="metric-label">{{ t('metric.suggestions') }}</div>
        <div class="metric-strip-value">{{ formatNumber(suggestions.length) }}</div>
        <div class="metric-strip-note">{{ t('metric.suggestionsNote') }}</div>
      </div>
      <div class="metric-strip-item metric-success">
        <div class="metric-label">{{ t('metric.saved') }}</div>
        <div class="metric-strip-value">{{ formatNumber(savedPrompts.length) }}</div>
        <div class="metric-strip-note">{{ t('metric.savedNote') }}</div>
      </div>
      <div class="metric-strip-item metric-info">
        <div class="metric-label">{{ t('metric.occurrences') }}</div>
        <div class="metric-strip-value">{{ formatNumber(totalOccurrences) }}</div>
        <div class="metric-strip-note">{{ t('metric.occurrencesNote') }}</div>
      </div>
      <div class="metric-strip-item metric-warning">
        <div class="metric-label">{{ t('metric.copies') }}</div>
        <div class="metric-strip-value">{{ formatNumber(totalCopies) }}</div>
        <div class="metric-strip-note">{{ t('metric.copiesNote') }}</div>
      </div>
    </section>

    <div class="prompts-layout">
      <section class="panel prompts-suggestions-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('suggestions.title') }}</h2>
            <div class="panel-kicker">{{ t('suggestions.kicker') }}</div>
          </div>
          <FileTextOutlined class="panel-header-icon" />
        </div>
        <div class="panel-body">
          <div class="toolbar prompts-toolbar">
            <div class="toolbar-left">
              <a-input
                v-model:value="search"
                class="control-wide"
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
                class="control-medium"
                allow-clear
                :placeholder="t('filter.agentPlaceholder')"
                :options="agentOptions"
                @change="load"
              />
              <a-input
                v-model:value="project"
                class="prompt-project-filter"
                allow-clear
                :placeholder="t('filter.projectPlaceholder')"
                @press-enter="load"
              />
              <span class="inline-field">
                {{ t('filter.minCount') }}
                <a-input-number v-model:value="minCount" class="prompt-min-count" :min="1" :max="99" @press-enter="load" />
              </span>
              <a-button type="primary" @click="load">{{ t('action.apply') }}</a-button>
              <a-button @click="resetFilters">{{ t('action.reset') }}</a-button>
            </div>
            <div class="toolbar-right muted sessions-row-count">
              {{ formatNumber(suggestions.length) }}
            </div>
          </div>

          <a-table
            :columns="suggestionColumns"
            :data-source="suggestions"
            :loading="loading"
            :pagination="{ pageSize: 10, showSizeChanger: true }"
            :scroll="{ x: 880 }"
            table-layout="fixed"
            row-key="key"
            size="small"
          >
            <template #emptyText>
              <EmptyState
                :title="t('empty.suggestions.title')"
                :text="hasActiveFilters ? '' : t('empty.suggestions.text')"
                compact
                :icon="FileTextOutlined"
              />
            </template>
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'prompt'">
                <div class="prompt-text-cell">
                  <a-typography-paragraph class="prompt-snippet" :ellipsis="{ rows: 3, expandable: true, symbol: 'more' }">
                    {{ record.text }}
                  </a-typography-paragraph>
                  <div class="prompt-tag-row">
                    <a-tag class="status-tag" :color="record.matchKind === 'exact' ? 'success' : 'processing'">
                      {{ record.matchKind === 'exact' ? t('tag.exact') : t('tag.near') }}
                    </a-tag>
                    <a-tag class="status-tag" color="default">{{ variantsLabel(record) }}</a-tag>
                    <a-tag class="status-tag" color="default">{{ sessionsLabel(record) }}</a-tag>
                  </div>
                </div>
              </template>
              <template v-else-if="column.key === 'count'">
                <div class="number-cell prompt-count">{{ formatNumber(record.count) }}</div>
              </template>
              <template v-else-if="column.key === 'context'">
                <a-tooltip :title="exampleSource(record.examples[0]).title" placement="topLeft">
                  <div class="source-identity-name">{{ exampleSource(record.examples[0]).label }}</div>
                </a-tooltip>
                <a-tooltip :title="record.examples[0]?.projectPath" placement="topLeft">
                  <div class="timeline-event-raw">{{ exampleProject(record.examples[0]).main }}</div>
                </a-tooltip>
                <button class="inline-link mono" type="button" @click="openExample(record.examples[0])">
                  {{ sessionLabel({ id: record.examples[0]?.sessionId || 0, sessionKey: record.examples[0]?.sessionKey || '', codexSessionId: record.examples[0]?.codexSessionId }) }}
                </button>
              </template>
              <template v-else-if="column.key === 'lastUsedAt'">
                <div>{{ formatDateTime(record.lastUsedAt) }}</div>
                <div class="timeline-event-raw">{{ shortPath(record.examples[0]?.rawSourcePath || '') }}</div>
              </template>
              <template v-else-if="column.key === 'actions'">
                <div class="row-actions">
                  <a-tooltip :title="t('action.copy')">
                    <a-button type="text" size="small" @click="copyPrompt(record.text)">
                      <template #icon>
                        <CopyOutlined />
                      </template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip :title="t('action.save')">
                    <a-button type="text" size="small" @click="openSavePrompt(record)">
                      <template #icon>
                        <SaveOutlined />
                      </template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip :title="t('action.ignore')">
                    <a-button type="text" size="small" @click="ignoreSuggestion(record)">
                      <template #icon>
                        <EyeInvisibleOutlined />
                      </template>
                    </a-button>
                  </a-tooltip>
                </div>
              </template>
            </template>
          </a-table>
        </div>
      </section>

      <section class="panel prompts-saved-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('saved.title') }}</h2>
            <div class="panel-kicker">{{ t('saved.kicker') }}</div>
          </div>
          <div class="panel-actions">
            <a-button type="primary" size="small" @click="openSavePrompt({ key: '', text: '', count: 0, sessionCount: 0, variantCount: 0, firstUsedAt: '', lastUsedAt: '', matchKind: 'manual', confidence: 1, examples: [], variants: [] })">
              <template #icon>
                <SaveOutlined />
              </template>
              {{ t('action.save') }}
            </a-button>
          </div>
        </div>
        <div class="panel-body">
          <a-table
            :columns="savedColumns"
            :data-source="savedPrompts"
            :loading="loading"
            :pagination="{ pageSize: 8, showSizeChanger: false }"
            :scroll="{ x: 620 }"
            table-layout="fixed"
            row-key="id"
            size="small"
          >
            <template #emptyText>
              <EmptyState :title="t('empty.saved.title')" :text="t('empty.saved.text')" compact :icon="SaveOutlined" />
            </template>
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'title'">
                <a-typography-text class="prompt-saved-title" :ellipsis="{ tooltip: record.title }">
                  {{ record.title }}
                </a-typography-text>
                <a-typography-paragraph class="prompt-saved-content" :ellipsis="{ rows: 2 }">
                  {{ record.content }}
                </a-typography-paragraph>
              </template>
              <template v-else-if="column.key === 'copyCount'">
                <span class="number-cell">{{ formatNumber(record.copyCount) }}</span>
              </template>
              <template v-else-if="column.key === 'updatedAt'">
                <div>{{ formatDateTime(record.updatedAt) }}</div>
                <div v-if="record.lastCopiedAt" class="timeline-event-raw">{{ formatDateTime(record.lastCopiedAt) }}</div>
              </template>
              <template v-else-if="column.key === 'actions'">
                <div class="row-actions">
                  <a-tooltip :title="t('action.copy')">
                    <a-button type="text" size="small" @click="copyPrompt(record.content, record)">
                      <template #icon>
                        <CopyOutlined />
                      </template>
                    </a-button>
                  </a-tooltip>
                  <a-tooltip :title="t('action.edit')">
                    <a-button type="text" size="small" @click="openEditPrompt(record)">
                      <template #icon>
                        <EditOutlined />
                      </template>
                    </a-button>
                  </a-tooltip>
                  <a-popconfirm :title="t('action.delete')" @confirm="deletePrompt(record)">
                    <a-tooltip :title="t('action.delete')">
                      <a-button type="text" size="small">
                        <template #icon>
                          <DeleteOutlined />
                        </template>
                      </a-button>
                    </a-tooltip>
                  </a-popconfirm>
                </div>
              </template>
            </template>
          </a-table>
        </div>
      </section>
    </div>

    <a-modal
      v-model:open="editorOpen"
      :title="editingPrompt ? t('modal.editTitle') : t('modal.createTitle')"
      :confirm-loading="editorSaving"
      @ok="submitEditor"
    >
      <div class="prompt-editor">
        <label class="prompt-editor-field">
          <span>{{ t('form.title') }}</span>
          <a-input v-model:value="editor.title" />
        </label>
        <label class="prompt-editor-field">
          <span>{{ t('form.content') }}</span>
          <a-textarea v-model:value="editor.content" :auto-size="{ minRows: 7, maxRows: 14 }" />
        </label>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.prompts-metric-strip {
  grid-template-columns: repeat(4, minmax(160px, 1fr));
}

.prompts-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.45fr) minmax(360px, 0.75fr);
  gap: 16px;
}

.prompt-project-filter {
  width: 220px;
  max-width: 100%;
}

.prompt-min-count {
  width: 74px;
}

.prompt-text-cell,
.prompt-saved-panel,
.prompt-suggestions-panel {
  min-width: 0;
}

.prompt-snippet {
  margin-bottom: 6px !important;
  color: var(--am-text);
  font-size: 13px;
  line-height: 19px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.prompt-tag-row,
.row-actions {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.row-actions {
  justify-content: flex-end;
}

.prompt-count {
  color: var(--am-text);
  font-size: 18px;
  font-weight: 760;
}

.inline-link {
  display: inline;
  max-width: 100%;
  padding: 0;
  color: var(--am-primary);
  cursor: pointer;
  background: transparent;
  border: 0;
  font-size: 12px;
  line-height: 18px;
  text-align: left;
}

.inline-link:hover,
.inline-link:focus-visible {
  color: var(--am-primary-strong);
  text-decoration: underline;
}

.prompt-saved-title {
  display: block;
  max-width: 100%;
  color: var(--am-text);
  font-weight: 700;
}

.prompt-saved-content {
  margin: 3px 0 0 !important;
  color: var(--am-muted);
  font-size: 12px;
  line-height: 18px;
}

.prompt-editor {
  display: grid;
  gap: 12px;
}

.prompt-editor-field {
  display: grid;
  gap: 6px;
  color: var(--am-text-soft);
  font-size: 12px;
  font-weight: 700;
}

@media (max-width: 1180px) {
  .prompts-layout,
  .prompts-metric-strip {
    grid-template-columns: 1fr;
  }
}
</style>
