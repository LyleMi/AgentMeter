<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message } from 'ant-design-vue'
import { DatabaseOutlined, FolderOpenOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDateTime, formatDuration, formatNumber, type Settings } from '../api'

const loading = ref(true)
const saving = ref(false)
const indexing = ref(false)
const settings = ref<Settings | null>(null)
const sourcePath = ref('')

const pricingColumns = [
  { title: 'Model', dataIndex: 'model', key: 'model' },
  { title: 'Input / 1M', dataIndex: 'inputPer1m', key: 'input', width: 120, align: 'right' },
  { title: 'Cached / 1M', dataIndex: 'cachedInputPer1m', key: 'cached', width: 130, align: 'right' },
  { title: 'Output / 1M', dataIndex: 'outputPer1m', key: 'output', width: 130, align: 'right' },
  { title: 'Source', dataIndex: 'source', key: 'source', width: 180 }
]

function formatPrice(value: number) {
  return `$${value.toFixed(4)}`
}

function indexResultStatus() {
  const result = settings.value?.lastIndexResult
  if (!result) return { type: 'default', label: 'No index run' }
  if (result.failed > 0) return { type: 'error', label: `${formatNumber(result.failed)} failed` }
  if ((result.warnings?.length || 0) > 0) return { type: 'warning', label: `${formatNumber(result.warnings.length)} warnings` }
  return { type: 'success', label: 'Completed' }
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
      <a-button @click="load">Refresh</a-button>
    </div>

    <a-spin :spinning="loading">
      <div class="split-row">
        <section class="panel settings-tool-panel">
          <div class="panel-header">
            <h2 class="panel-title">Source</h2>
            <span class="muted">JSONL session folder</span>
          </div>
          <div class="panel-body">
            <a-space direction="vertical" style="width: 100%" size="middle">
              <a-input v-model:value="sourcePath">
                <template #prefix>
                  <FolderOpenOutlined />
                </template>
              </a-input>
              <div class="toolbar">
                <div class="toolbar-left">
                  <a-button type="primary" :loading="saving" @click="save">Save</a-button>
                  <a-button @click="sourcePath = settings?.defaultSourcePath || sourcePath">Use Default</a-button>
                </div>
                <a-typography-text class="muted" :ellipsis="{ tooltip: settings?.defaultSourcePath }">
                  {{ settings?.defaultSourcePath }}
                </a-typography-text>
              </div>
              <div class="settings-meta-line">
                <span class="muted">Current source</span>
                <a-typography-text :ellipsis="{ tooltip: sourcePath }">{{ sourcePath || '-' }}</a-typography-text>
              </div>
            </a-space>
          </div>
        </section>

        <section class="panel settings-tool-panel">
          <div class="panel-header">
            <h2 class="panel-title">Database</h2>
            <span class="muted">Local AgentMeter store</span>
          </div>
          <div class="panel-body">
            <a-space direction="vertical" style="width: 100%" size="middle">
              <a-input :value="settings?.databasePath" readonly>
                <template #prefix>
                  <DatabaseOutlined />
                </template>
              </a-input>
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
              <div v-if="settings?.lastIndexResult" class="index-result-block">
                <div class="index-result-header">
                  <div>
                    <div class="index-result-title">Last index result</div>
                    <div class="muted">
                      {{ settings.lastIndexStartedAt ? formatDateTime(settings.lastIndexStartedAt) : 'Most recent run' }}
                    </div>
                  </div>
                  <a-tag :color="indexResultStatus().type" class="status-tag">{{ indexResultStatus().label }}</a-tag>
                </div>
                <div class="index-result-grid">
                  <div class="index-result-metric">
                    <span class="muted">Files seen</span>
                    <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.filesSeen) }}</strong>
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
                    <strong class="number-cell status-error">{{ formatNumber(settings.lastIndexResult.failed) }}</strong>
                  </div>
                  <div class="index-result-metric">
                    <span class="muted">Warnings</span>
                    <strong class="number-cell status-warning">
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
                <div v-if="settings.lastIndexResult.warnings?.length" class="index-result-warnings">
                  <div class="metadata-label">Warnings</div>
                  <ul>
                    <li v-for="warning in settings.lastIndexResult.warnings.slice(0, 3)" :key="warning">
                      {{ warning }}
                    </li>
                  </ul>
                </div>
              </div>
            </a-space>
          </div>
        </section>
      </div>

      <section class="panel" style="margin-top: 18px">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Pricing Registry</h2>
            <div class="muted">API list-price estimates used for local cost calculations</div>
          </div>
        </div>
        <a-table
          class="dense-table pricing-table"
          size="middle"
          :columns="pricingColumns"
          :data-source="settings?.pricingModels || []"
          row-key="id"
          :pagination="false"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'input'">
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
    </a-spin>
  </div>
</template>
