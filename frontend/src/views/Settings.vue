<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { DatabaseOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDateTime, formatDuration, formatNumber, type Settings } from '../api'
import { notifyAppDataChanged } from '../events'

const ATextarea = AInput.TextArea
const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const loading = ref(true)
const saving = ref(false)
const indexing = ref(false)
const settings = ref<Settings | null>(null)
const sourcePath = ref('')

const pricingColumns = [
  { title: 'Model', dataIndex: 'model', key: 'model', width: 300 },
  { title: 'Input / 1M', dataIndex: 'inputPer1m', key: 'input', width: 112, align: 'right' },
  { title: 'Cached / 1M', dataIndex: 'cachedInputPer1m', key: 'cached', width: 122, align: 'right' },
  { title: 'Output / 1M', dataIndex: 'outputPer1m', key: 'output', width: 122, align: 'right' },
  { title: 'Source', dataIndex: 'source', key: 'source', width: 180 }
]

const sourceState = computed(() => {
  const saved = settings.value?.sourcePath || ''
  if (sourcePath.value !== saved) return { color: 'warning', label: 'Unsaved change' }
  if (saved) return { color: 'success', label: 'Configured' }
  return { color: 'warning', label: 'Missing source' }
})

const databaseState = computed(() => {
  if (settings.value?.databasePath) return { color: 'success', label: 'Local store' }
  return { color: 'warning', label: 'No database path' }
})

const indexStatus = computed(() => {
  const result = settings.value?.lastIndexResult
  if (!result) return { color: 'default', label: 'No index run', detail: 'Run indexing to populate the local database' }
  const duration = formatDuration(result.durationMs)
  if (result.failed > 0) {
    return { color: 'error', label: 'Failed', detail: `${formatNumber(result.failed)} failed · ${duration}` }
  }
  if ((result.warnings?.length || 0) > 0) {
    return { color: 'warning', label: 'Warnings', detail: `${formatNumber(result.warnings.length)} warnings · ${duration}` }
  }
  return { color: 'success', label: 'Completed', detail: `${formatNumber(result.indexed)} indexed · ${duration}` }
})

function formatPrice(value: number) {
  return `$${value.toFixed(4)}`
}

async function load() {
  loading.value = true
  try {
    settings.value = await api.getSettings()
    sourcePath.value = settings.value.sourcePath
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    settings.value = await api.saveSettings(sourcePath.value)
    notifyAppDataChanged('settings')
    message.success('Settings saved')
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Save failed')
  } finally {
    saving.value = false
  }
}

async function index(rebuild = false) {
  indexing.value = true
  try {
    const result = await api.indexNow(rebuild)
    message.success(`${result.indexed} indexed, ${result.skipped} skipped, ${result.failed} failed`)
    await load()
    notifyAppDataChanged('index')
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Index failed')
  } finally {
    indexing.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">Settings</h1>
        <div class="page-subtitle">Local source, database and built-in pricing registry</div>
      </div>
      <a-button @click="load">
        <template #icon>
          <ReloadOutlined />
        </template>
        Refresh
      </a-button>
    </div>

    <a-spin :spinning="loading">
      <div class="section-stack">
        <div class="split-row">
          <section class="panel settings-tool-panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">Source</h2>
                <div class="panel-kicker">One local agent root per line</div>
              </div>
              <a-tag :color="sourceState.color" class="status-tag">{{ sourceState.label }}</a-tag>
            </div>
            <div class="panel-body">
              <div class="section-stack">
                <a-textarea v-model:value="sourcePath" :auto-size="{ minRows: 3, maxRows: 8 }" />

                <div class="toolbar">
                  <div class="toolbar-left">
                    <a-button type="primary" :loading="saving" @click="save">Save</a-button>
                    <a-button @click="sourcePath = settings?.defaultSourcePath || sourcePath">Use Defaults</a-button>
                  </div>
                </div>

                <div class="metadata-grid">
                  <div class="metadata-item">
                    <div class="metadata-label">Current sources</div>
                    <div class="metadata-value">
                      <a-typography-text :ellipsis="{ tooltip: settings?.sourcePath || sourcePath }">
                        {{ settings?.sourcePaths?.length ? `${formatNumber(settings.sourcePaths.length)} configured` : '-' }}
                      </a-typography-text>
                    </div>
                  </div>
                  <div class="metadata-item">
                    <div class="metadata-label">Default sources</div>
                    <div class="metadata-value">
                      <a-typography-text :ellipsis="{ tooltip: settings?.defaultSourcePath }">
                        {{ settings?.defaultSourcePaths?.length ? `${formatNumber(settings.defaultSourcePaths.length)} detected` : '-' }}
                      </a-typography-text>
                    </div>
                  </div>
                </div>

                <div class="settings-meta-line">
                  <span class="muted">Editing</span>
                  <a-typography-text :ellipsis="{ tooltip: sourcePath }">{{ sourcePath || '-' }}</a-typography-text>
                </div>
              </div>
            </div>
          </section>

          <section class="panel settings-tool-panel">
            <div class="panel-header">
              <div>
                <h2 class="panel-title">Database</h2>
                <div class="panel-kicker">Local AgentMeter store</div>
              </div>
              <div class="summary-actions">
                <a-tag :color="databaseState.color" class="status-tag">{{ databaseState.label }}</a-tag>
                <a-tag :color="indexStatus.color" class="status-tag">{{ indexStatus.label }}</a-tag>
              </div>
            </div>
            <div class="panel-body">
              <div class="section-stack">
                <a-input :value="settings?.databasePath" readonly>
                  <template #prefix>
                    <DatabaseOutlined />
                  </template>
                </a-input>

                <div class="metadata-grid">
                  <div class="metadata-item is-wide">
                    <div class="metadata-label">Database path</div>
                    <div class="metadata-value">
                      <a-typography-text :ellipsis="{ tooltip: settings?.databasePath }">
                        {{ settings?.databasePath || '-' }}
                      </a-typography-text>
                    </div>
                  </div>
                  <div class="metadata-item">
                    <div class="metadata-label">Last run</div>
                    <div class="metadata-value">
                      {{ settings?.lastIndexStartedAt ? formatDateTime(settings.lastIndexStartedAt) : '-' }}
                    </div>
                  </div>
                  <div class="metadata-item">
                    <div class="metadata-label">Index state</div>
                    <div class="metadata-value">{{ indexStatus.detail }}</div>
                  </div>
                </div>

                <div class="toolbar">
                  <div class="toolbar-left">
                    <a-button type="primary" :loading="indexing" @click="index(false)">Index Now</a-button>
                    <a-button :loading="indexing" @click="index(true)">
                      <template #icon>
                        <ReloadOutlined />
                      </template>
                      Rebuild Index
                    </a-button>
                  </div>
                </div>

                <div class="index-result-block">
                  <div class="index-result-header">
                    <div>
                      <div class="index-result-title">Last index result</div>
                      <div class="muted">
                        {{ settings?.lastIndexStartedAt ? formatDateTime(settings.lastIndexStartedAt) : 'No completed run yet' }}
                      </div>
                    </div>
                    <a-tag :color="indexStatus.color" class="status-tag">{{ indexStatus.detail }}</a-tag>
                  </div>
                  <div v-if="settings?.lastIndexResult" class="index-result-grid">
                    <div class="index-result-metric">
                      <span class="muted">Files seen</span>
                      <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.filesSeen) }}</strong>
                    </div>
                    <div class="index-result-metric">
                      <span class="muted">Sources</span>
                      <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.sourcePaths?.length || 0) }}</strong>
                    </div>
                    <div class="index-result-metric">
                      <span class="muted">Sessions</span>
                      <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.sessions) }}</strong>
                    </div>
                    <div class="index-result-metric">
                      <span class="muted">Indexed</span>
                      <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.indexed) }}</strong>
                    </div>
                    <div class="index-result-metric">
                      <span class="muted">Skipped</span>
                      <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.skipped) }}</strong>
                    </div>
                    <div class="index-result-metric">
                      <span class="muted">Failed</span>
                      <strong class="number-cell" :class="settings.lastIndexResult.failed ? 'status-error' : 'status-ok'">
                        {{ formatNumber(settings.lastIndexResult.failed) }}
                      </strong>
                    </div>
                    <div class="index-result-metric">
                      <span class="muted">Warnings</span>
                      <strong
                        class="number-cell"
                        :class="settings.lastIndexResult.warnings?.length ? 'status-warning' : 'status-ok'"
                      >
                        {{ formatNumber(settings.lastIndexResult.warnings?.length || 0) }}
                      </strong>
                    </div>
                    <div class="index-result-metric">
                      <span class="muted">Duration</span>
                      <strong class="number-cell duration-cell">
                        {{ formatDuration(settings.lastIndexResult.durationMs) }}
                      </strong>
                    </div>
                  </div>
                  <div v-else class="metric-note">No index result has been recorded for this database.</div>
                  <div v-if="settings?.lastIndexResult?.warnings?.length" class="index-result-warnings">
                    <div class="metadata-label">Warnings</div>
                    <ul>
                      <li v-for="warning in settings.lastIndexResult.warnings.slice(0, 3)" :key="warning">
                        {{ warning }}
                      </li>
                      <li v-if="settings.lastIndexResult.warnings.length > 3" class="muted">
                        +{{ formatNumber(settings.lastIndexResult.warnings.length - 3) }} more
                      </li>
                    </ul>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </div>

        <section class="panel">
          <div class="panel-header">
            <div>
              <h2 class="panel-title">Pricing Registry</h2>
              <div class="panel-kicker">API list-price estimates used for local cost calculations</div>
            </div>
            <span class="row-count">{{ formatNumber(settings?.pricingModels?.length || 0) }} models</span>
          </div>
          <a-table
            class="dense-table pricing-table"
            size="small"
            :columns="pricingColumns"
            :data-source="settings?.pricingModels || []"
            row-key="id"
            :pagination="false"
            :scroll="{ x: 900 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'model'">
                <a-typography-text :ellipsis="{ tooltip: record.model }">
                  {{ record.model }}
                </a-typography-text>
                <div v-if="record.normalizedModel && record.normalizedModel !== record.model" class="muted pricing-source-date">
                  {{ record.normalizedModel }}
                </div>
              </template>
              <template v-else-if="column.key === 'input'">
                <span class="number-cell price-cell">{{ formatPrice(record.inputPer1m) }}</span>
              </template>
              <template v-else-if="column.key === 'cached'">
                <span class="number-cell price-cell">{{ formatPrice(record.cachedInputPer1m) }}</span>
              </template>
              <template v-else-if="column.key === 'output'">
                <span class="number-cell price-cell">{{ formatPrice(record.outputPer1m) }}</span>
              </template>
              <template v-else-if="column.key === 'source'">
                <a-typography-text :ellipsis="{ tooltip: `${record.source} · ${formatDateTime(record.effectiveFrom)}` }">
                  {{ record.source }}
                </a-typography-text>
                <div class="muted pricing-source-date">{{ formatDateTime(record.effectiveFrom) }}</div>
              </template>
            </template>
          </a-table>
        </section>
      </div>
    </a-spin>
  </div>
</template>
