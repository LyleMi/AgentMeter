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
  { title: 'Input / 1M', dataIndex: 'inputPer1m', key: 'input', width: 120 },
  { title: 'Cached / 1M', dataIndex: 'cachedInputPer1m', key: 'cached', width: 130 },
  { title: 'Output / 1M', dataIndex: 'outputPer1m', key: 'output', width: 130 },
  { title: 'Source', dataIndex: 'source', key: 'source' }
]

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
        <section class="panel">
          <div class="panel-header">
            <h2 class="panel-title">Source</h2>
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
            </a-space>
          </div>
        </section>

        <section class="panel">
          <div class="panel-header">
            <h2 class="panel-title">Database</h2>
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
              <a-alert
                v-if="settings?.lastIndexResult"
                type="info"
                show-icon
                :message="`${formatNumber(settings.lastIndexResult.indexed)} indexed, ${formatNumber(settings.lastIndexResult.skipped)} skipped, ${formatNumber(settings.lastIndexResult.failed)} failed`"
                :description="`${formatNumber(settings.lastIndexResult.filesSeen)} files seen in ${formatDuration(settings.lastIndexResult.durationMs)}`"
              />
            </a-space>
          </div>
        </section>
      </div>

      <section class="panel" style="margin-top: 18px">
        <div class="panel-header">
          <h2 class="panel-title">Pricing Registry</h2>
          <span class="muted">API list-price estimates</span>
        </div>
        <a-table
          size="middle"
          :columns="pricingColumns"
          :data-source="settings?.pricingModels || []"
          row-key="id"
          :pagination="false"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'input'">${{ record.inputPer1m.toFixed(4) }}</template>
            <template v-else-if="column.key === 'cached'">${{ record.cachedInputPer1m.toFixed(4) }}</template>
            <template v-else-if="column.key === 'output'">${{ record.outputPer1m.toFixed(4) }}</template>
            <template v-else-if="column.key === 'source'">
              <a-typography-text :ellipsis="{ tooltip: `${record.source} · ${formatDateTime(record.effectiveFrom)}` }">
                {{ record.source }}
              </a-typography-text>
            </template>
          </template>
        </a-table>
      </section>
    </a-spin>
  </div>
</template>
