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
import { api, formatNumber, type Settings, type SourceEntry } from '../api'
import { notifyAppDataChanged } from '../events'

const ATypographyText = Typography.Text

const loading = ref(true)
const saving = ref(false)
const settings = ref<Settings | null>(null)
const sourceEntries = ref<SourceEntry[]>([])
const newSourcePath = ref('')

const normalizedEntries = computed(() => normalizeEntries(sourceEntries.value))
const savedEntries = computed(() => normalizeEntries(settings.value?.sourceEntries || []))
const enabledCount = computed(() => normalizedEntries.value.filter((entry) => entry.enabled).length)
const disabledCount = computed(() => normalizedEntries.value.length - enabledCount.value)
const entriesChanged = computed(() => JSON.stringify(normalizedEntries.value) !== JSON.stringify(savedEntries.value))

const sourceState = computed(() => {
  if (entriesChanged.value) return { color: 'warning', label: 'Unsaved' }
  if (enabledCount.value > 0) return { color: 'success', label: `${formatNumber(enabledCount.value)} enabled` }
  if (normalizedEntries.value.length > 0) return { color: 'default', label: 'All disabled' }
  return { color: 'warning', label: 'Missing' }
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
    result.push({ path, enabled: Boolean(entry.enabled) })
  }
  return result
}

function copyEntries(entries: SourceEntry[]) {
  return entries.map((entry) => ({ path: entry.path, enabled: Boolean(entry.enabled) }))
}

function entriesFromPaths(paths: string[], enabled = true) {
  return paths.map((path) => ({ path, enabled }))
}

async function load() {
  loading.value = true
  try {
    const value = await api.getSettings()
    settings.value = value
    const entries = value.sourceEntries?.length ? value.sourceEntries : entriesFromPaths(value.sourcePaths || [])
    sourceEntries.value = copyEntries(entries)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    settings.value = await api.saveSourceSettings(normalizedEntries.value)
    sourceEntries.value = copyEntries(settings.value.sourceEntries || [])
    notifyAppDataChanged('settings')
    message.success('Source settings saved')
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Save failed')
  } finally {
    saving.value = false
  }
}

function addSource() {
  const path = newSourcePath.value.trim()
  if (!path) return
  if (normalizedEntries.value.some((entry) => entryKey(entry.path) === entryKey(path))) {
    message.warning('Source already exists')
    return
  }
  sourceEntries.value.push({ path, enabled: true })
  newSourcePath.value = ''
}

function removeSource(index: number) {
  sourceEntries.value.splice(index, 1)
}

function useDefaults() {
  const paths = settings.value?.defaultSourcePaths || []
  if (!paths.length) {
    message.warning('No default sources detected')
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
            <h2 class="panel-title">Source</h2>
            <div class="panel-kicker">Local agent roots</div>
          </div>
          <div class="summary-actions">
            <a-tag :color="sourceState.color" class="status-tag">{{ sourceState.label }}</a-tag>
            <a-button @click="load">
              <template #icon>
                <ReloadOutlined />
              </template>
              Refresh
            </a-button>
          </div>
        </div>
        <div class="panel-body">
          <div class="section-stack">
            <div class="metadata-grid">
              <div class="metadata-item">
                <div class="metadata-label">Sources</div>
                <div class="metadata-value">{{ formatNumber(normalizedEntries.length) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Enabled</div>
                <div class="metadata-value status-ok">{{ formatNumber(enabledCount) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Disabled</div>
                <div class="metadata-value">{{ formatNumber(disabledCount) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">Defaults</div>
                <div class="metadata-value">{{ formatNumber(settings?.defaultSourcePaths?.length || 0) }}</div>
              </div>
            </div>

            <div class="source-entry-list">
              <div
                v-for="(entry, index) in sourceEntries"
                :key="index"
                class="source-entry-row"
                :class="{ 'is-disabled': !entry.enabled }"
              >
                <div class="source-entry-state">
                  <a-switch v-model:checked="entry.enabled" size="small" />
                  <a-tag :color="entry.enabled ? 'success' : 'default'" class="status-tag">
                    {{ entry.enabled ? 'Enabled' : 'Disabled' }}
                  </a-tag>
                </div>
                <a-input v-model:value="entry.path" class="source-entry-input" />
                <a-button type="text" danger title="Remove" @click="removeSource(index)">
                  <template #icon>
                    <DeleteOutlined />
                  </template>
                </a-button>
              </div>
              <div v-if="!sourceEntries.length" class="empty-state empty-state-compact">
                <FolderAddOutlined class="empty-state-icon" />
                <div class="empty-state-title">No sources configured</div>
              </div>
            </div>

            <div class="source-add-row">
              <a-input v-model:value="newSourcePath" placeholder="Source path" @pressEnter="addSource" />
              <a-button @click="addSource">
                <template #icon>
                  <PlusOutlined />
                </template>
                Add
              </a-button>
            </div>

            <div class="toolbar">
              <div class="toolbar-left">
                <a-button type="primary" :loading="saving" :disabled="!entriesChanged" @click="save">
                  <template #icon>
                    <SaveOutlined />
                  </template>
                  Save
                </a-button>
                <a-button @click="useDefaults">
                  <template #icon>
                    <FolderAddOutlined />
                  </template>
                  Use Defaults
                </a-button>
              </div>
            </div>

            <div class="settings-meta-line">
              <span class="muted">Active</span>
              <a-typography-text :ellipsis="{ tooltip: settings?.sourcePath }">
                {{ settings?.sourcePaths?.length ? `${formatNumber(settings.sourcePaths.length)} enabled sources` : '-' }}
              </a-typography-text>
            </div>
          </div>
        </div>
      </section>
    </div>
  </a-spin>
</template>
