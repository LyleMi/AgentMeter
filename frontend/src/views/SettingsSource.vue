<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import ASwitch from 'ant-design-vue/es/switch'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { DeleteOutlined, FolderAddOutlined, PlusOutlined, ReloadOutlined, SaveOutlined } from '@ant-design/icons-vue'
import { api, formatBytes, formatDateTime, formatNumber, isStaticDemo, type Settings, type SourceEntry, type SourceStorage } from '../api'
import { notifyAppDataChanged } from '../events'
import { useMessages } from '../i18n'

const ATypographyText = Typography.Text
const { t } = useMessages({
  en: {
    'source.title': 'Source',
    'source.kicker': 'Local agent roots',
    'source.action.refresh': 'Refresh',
    'source.action.add': 'Add',
    'source.action.save': 'Save',
    'source.action.useDefaults': 'Use Defaults',
    'source.action.remove': 'Remove',
    'source.placeholder.label': 'Optional label',
    'source.placeholder.path': 'Source path',
    'source.meta.sources': 'Sources',
    'source.meta.enabled': 'Enabled',
    'source.meta.disabled': 'Disabled',
    'source.meta.defaults': 'Defaults',
    'source.meta.active': 'Active',
    'source.meta.enabledSources': 'enabled sources',
    'source.storage.title': 'Directory storage',
    'source.storage.kicker': 'Space used by local model and session files',
    'source.storage.total': 'Total size',
    'source.storage.files': 'Files',
    'source.storage.scanned': 'Scanned',
    'source.storage.unavailable': 'Unavailable',
    'source.storage.partial': 'Partial result',
    'source.storage.empty': 'No directory storage is available',
    'source.state.unsaved': 'Unsaved',
    'source.state.demo': 'Static demo',
    'source.state.enabledSuffix': 'enabled',
    'source.state.allDisabled': 'All disabled',
    'source.state.missing': 'Missing',
    'source.empty': 'No sources configured',
    'source.message.saved': 'Source settings saved',
    'source.message.saveFailed': 'Save failed',
    'source.message.duplicate': 'Source already exists',
    'source.message.noDefaults': 'No default sources detected',
    'source.message.demoReadOnly': 'Static demo mode is read-only.'
  },
  'zh-CN': {
    'source.title': '来源',
    'source.kicker': '本地代理根目录',
    'source.action.refresh': '刷新',
    'source.action.add': '添加',
    'source.action.save': '保存',
    'source.action.useDefaults': '使用默认值',
    'source.action.remove': '移除',
    'source.placeholder.label': '可选标签',
    'source.placeholder.path': '来源路径',
    'source.meta.sources': '来源',
    'source.meta.enabled': '已启用',
    'source.meta.disabled': '已禁用',
    'source.meta.defaults': '默认值',
    'source.meta.active': '当前',
    'source.meta.enabledSources': '个已启用来源',
    'source.storage.title': '目录存储占用',
    'source.storage.kicker': '本地模型与会话文件占用的空间',
    'source.storage.total': '总占用',
    'source.storage.files': '文件数',
    'source.storage.scanned': '统计时间',
    'source.storage.unavailable': '不可用',
    'source.storage.partial': '部分结果',
    'source.storage.empty': '暂无可统计的目录',
    'source.state.unsaved': '未保存',
    'source.state.demo': '静态演示',
    'source.state.enabledSuffix': '个已启用',
    'source.state.allDisabled': '全部禁用',
    'source.state.missing': '缺失',
    'source.empty': '尚未配置来源',
    'source.message.saved': '来源设置已保存',
    'source.message.saveFailed': '保存失败',
    'source.message.duplicate': '来源已存在',
    'source.message.noDefaults': '未检测到默认来源',
    'source.message.demoReadOnly': '静态演示模式为只读。'
  }
})

const loading = ref(true)
const saving = ref(false)
const settings = ref<Settings | null>(null)
const storage = ref<SourceStorage | null>(null)
const sourceEntries = ref<SourceEntry[]>([])
const newSourcePath = ref('')

const normalizedEntries = computed(() => normalizeEntries(sourceEntries.value))
const savedEntries = computed(() => normalizeEntries(settings.value?.sourceEntries || []))
const enabledCount = computed(() => normalizedEntries.value.filter((entry) => entry.enabled).length)
const disabledCount = computed(() => normalizedEntries.value.length - enabledCount.value)
const entriesChanged = computed(() => JSON.stringify(normalizedEntries.value) !== JSON.stringify(savedEntries.value))
const largestDirectorySize = computed(() => Math.max(0, ...(storage.value?.directories || []).map((item) => item.sizeBytes)))

const sourceState = computed(() => {
  if (isStaticDemo) return { color: 'processing', label: t('source.state.demo') }
  if (entriesChanged.value) return { color: 'warning', label: t('source.state.unsaved') }
  if (enabledCount.value > 0) {
    return { color: 'success', label: `${formatNumber(enabledCount.value)} ${t('source.state.enabledSuffix')}` }
  }
  if (normalizedEntries.value.length > 0) return { color: 'default', label: t('source.state.allDisabled') }
  return { color: 'warning', label: t('source.state.missing') }
})

function entryKey(path: string) {
  return path.trim().toLowerCase()
}

function normalizeEntries(entries: SourceEntry[]) {
  const seen = new Set<string>()
  const result: SourceEntry[] = []
  for (const entry of entries) {
    const path = entry.path.trim()
    if (!path) continue
    const key = entryKey(path)
    if (seen.has(key)) continue
    seen.add(key)
    const normalized: SourceEntry = { path, enabled: Boolean(entry.enabled) }
    const label = entry.label?.trim()
    if (label) normalized.label = label
    result.push(normalized)
  }
  return result
}

function copyEntries(entries: SourceEntry[]) {
  return entries.map((entry) => ({ path: entry.path, enabled: Boolean(entry.enabled), label: entry.label || '' }))
}

function storageBarWidth(sizeBytes: number) {
  if (!sizeBytes || !largestDirectorySize.value) return '0%'
  return `${Math.max(4, (sizeBytes / largestDirectorySize.value) * 100)}%`
}

function entriesFromPaths(paths: string[], enabled = true) {
  return paths.map((path) => ({ path, enabled, label: '' }))
}

async function load() {
  loading.value = true
  try {
    const [value, storageValue] = await Promise.all([api.getSettings(), api.getSourceStorage()])
    settings.value = value
    storage.value = storageValue
    const entries = value.sourceEntries?.length ? value.sourceEntries : entriesFromPaths(value.sourcePaths || [])
    sourceEntries.value = copyEntries(entries)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (isStaticDemo) {
    message.info(t('source.message.demoReadOnly'))
    return
  }
  saving.value = true
  try {
    settings.value = await api.saveSourceSettings(normalizedEntries.value)
    sourceEntries.value = copyEntries(settings.value.sourceEntries || [])
    notifyAppDataChanged('settings')
    message.success(t('source.message.saved'))
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('source.message.saveFailed'))
  } finally {
    saving.value = false
  }
}

function addSource() {
  if (isStaticDemo) {
    message.info(t('source.message.demoReadOnly'))
    return
  }
  const path = newSourcePath.value.trim()
  if (!path) return
  if (normalizedEntries.value.some((entry) => entryKey(entry.path) === entryKey(path))) {
    message.warning(t('source.message.duplicate'))
    return
  }
  sourceEntries.value.push({ path, enabled: true, label: '' })
  newSourcePath.value = ''
}

function removeSource(index: number) {
  if (isStaticDemo) {
    message.info(t('source.message.demoReadOnly'))
    return
  }
  sourceEntries.value.splice(index, 1)
}

function useDefaults() {
  if (isStaticDemo) {
    message.info(t('source.message.demoReadOnly'))
    return
  }
  const paths = settings.value?.defaultSourcePaths || []
  if (!paths.length) {
    message.warning(t('source.message.noDefaults'))
    return
  }
  sourceEntries.value = entriesFromPaths(paths)
}

onMounted(load)
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="panel settings-tool-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('source.title') }}</h2>
            <div class="panel-kicker">{{ t('source.kicker') }}</div>
          </div>
          <div class="summary-actions">
            <a-tag :color="sourceState.color" class="status-tag">{{ sourceState.label }}</a-tag>
            <a-button @click="load">
              <template #icon>
                <ReloadOutlined />
              </template>
              {{ t('source.action.refresh') }}
            </a-button>
          </div>
        </div>
        <div class="panel-body">
          <div class="section-stack">
            <div class="metadata-grid">
              <div class="metadata-item">
                <div class="metadata-label">{{ t('source.meta.sources') }}</div>
                <div class="metadata-value">{{ formatNumber(normalizedEntries.length) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('source.meta.enabled') }}</div>
                <div class="metadata-value status-ok">{{ formatNumber(enabledCount) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('source.meta.disabled') }}</div>
                <div class="metadata-value">{{ formatNumber(disabledCount) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('source.meta.defaults') }}</div>
                <div class="metadata-value">{{ formatNumber(settings?.defaultSourcePaths?.length || 0) }}</div>
              </div>
            </div>

            <section class="source-storage-block" aria-labelledby="source-storage-title">
              <div class="source-storage-header">
                <div>
                  <h3 id="source-storage-title" class="source-storage-title">{{ t('source.storage.title') }}</h3>
                  <div class="muted">{{ t('source.storage.kicker') }}</div>
                </div>
                <div class="source-storage-totals">
                  <div>
                    <span>{{ t('source.storage.total') }}</span>
                    <strong>{{ formatBytes(storage?.totalSizeBytes) }}</strong>
                  </div>
                  <div>
                    <span>{{ t('source.storage.files') }}</span>
                    <strong>{{ formatNumber(storage?.totalFileCount) }}</strong>
                  </div>
                </div>
              </div>
              <div v-if="storage?.directories.length" class="source-storage-list">
                <div v-for="directory in storage.directories" :key="directory.path" class="source-storage-row" :class="{ 'is-disabled': !directory.enabled }">
                  <div class="source-storage-row-heading">
                    <div class="source-storage-identity">
                      <span class="source-storage-name">{{ directory.label || directory.path }}</span>
                      <a-tag v-if="!directory.exists" class="status-tag">{{ t('source.storage.unavailable') }}</a-tag>
                      <a-tag v-else-if="directory.error" color="warning" class="status-tag">{{ t('source.storage.partial') }}</a-tag>
                    </div>
                    <div class="source-storage-measure">
                      <strong>{{ formatBytes(directory.sizeBytes) }}</strong>
                      <span>{{ formatNumber(directory.fileCount) }} {{ t('source.storage.files') }}</span>
                    </div>
                  </div>
                  <div class="source-storage-path" :title="directory.path">{{ directory.path }}</div>
                  <div class="source-storage-track" aria-hidden="true">
                    <div class="source-storage-fill" :style="{ width: storageBarWidth(directory.sizeBytes) }" />
                  </div>
                </div>
              </div>
              <div v-else class="empty-state empty-state-compact">{{ t('source.storage.empty') }}</div>
              <div v-if="storage?.scannedAt" class="source-storage-scanned">
                {{ t('source.storage.scanned') }} · {{ formatDateTime(storage.scannedAt) }}
              </div>
            </section>

            <div class="source-entry-list">
              <div
                v-for="(entry, index) in sourceEntries"
                :key="index"
                class="source-entry-row"
                :class="{ 'is-disabled': !entry.enabled }"
              >
                <div class="source-entry-state">
                  <a-switch v-model:checked="entry.enabled" size="small" :disabled="isStaticDemo" />
                  <a-tag :color="entry.enabled ? 'success' : 'default'" class="status-tag">
                    {{ entry.enabled ? t('source.meta.enabled') : t('source.meta.disabled') }}
                  </a-tag>
                </div>
                <a-input v-model:value="entry.label" class="source-entry-label" :placeholder="t('source.placeholder.label')" :disabled="isStaticDemo" />
                <a-input v-model:value="entry.path" class="source-entry-input" :placeholder="t('source.placeholder.path')" :disabled="isStaticDemo" />
                <a-button type="text" danger :title="t('source.action.remove')" :disabled="isStaticDemo" @click="removeSource(index)">
                  <template #icon>
                    <DeleteOutlined />
                  </template>
                </a-button>
              </div>
              <div v-if="!sourceEntries.length" class="empty-state empty-state-compact">
                <FolderAddOutlined class="empty-state-icon" />
                <div class="empty-state-title">{{ t('source.empty') }}</div>
              </div>
            </div>

            <div class="source-add-row">
              <a-input v-model:value="newSourcePath" :placeholder="t('source.placeholder.path')" :disabled="isStaticDemo" @pressEnter="addSource" />
              <a-button :disabled="isStaticDemo" @click="addSource">
                <template #icon>
                  <PlusOutlined />
                </template>
                {{ t('source.action.add') }}
              </a-button>
            </div>

            <div class="toolbar">
              <div class="toolbar-left">
                <a-button type="primary" :loading="saving" :disabled="isStaticDemo || !entriesChanged" @click="save">
                  <template #icon>
                    <SaveOutlined />
                  </template>
                  {{ t('source.action.save') }}
                </a-button>
                <a-button :disabled="isStaticDemo" @click="useDefaults">
                  <template #icon>
                    <FolderAddOutlined />
                  </template>
                  {{ t('source.action.useDefaults') }}
                </a-button>
              </div>
            </div>

            <div class="settings-meta-line">
              <span class="muted">{{ t('source.meta.active') }}</span>
              <a-typography-text :ellipsis="{ tooltip: settings?.sourcePath }">
                {{ settings?.sourcePaths?.length ? `${formatNumber(settings.sourcePaths.length)} ${t('source.meta.enabledSources')}` : '-' }}
              </a-typography-text>
            </div>
          </div>
        </div>
      </section>
    </div>
  </a-spin>
</template>
