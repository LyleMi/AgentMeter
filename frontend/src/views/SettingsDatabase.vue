<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import Tooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import { DatabaseOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDateTime, formatDuration, formatNumber, type Settings } from '../api'
import { notifyAppDataChanged } from '../events'

const ATooltip = Tooltip
const ATypographyText = Typography.Text

const loading = ref(true)
const indexing = ref(false)
const settings = ref<Settings | null>(null)
const updateIndexHint = 'Scan enabled sources and parse only new or changed JSONL files.'
const rebuildIndexHint = 'Clear indexed files for enabled sources, then parse every JSONL file again.'

const databaseState = computed(() => {
  if (settings.value?.databasePath) return { color: 'success', label: 'Local store' }
  return { color: 'warning', label: 'No database path' }
})

const indexStatus = computed(() => {
  const result = settings.value?.lastIndexResult
  if (!result) return { color: 'default', label: 'No index run', detail: 'No completed run yet' }
  const duration = formatDuration(result.durationMs)
  if (result.failed > 0) {
    return { color: 'error', label: 'Failed', detail: `${formatNumber(result.failed)} failed · ${duration}` }
  }
  if ((result.warnings?.length || 0) > 0) {
    return { color: 'warning', label: 'Warnings', detail: `${formatNumber(result.warnings.length)} warnings · ${duration}` }
  }
  return { color: 'success', label: 'Completed', detail: `${formatNumber(result.indexed)} indexed · ${duration}` }
})

async function load() {
  loading.value = true
  try {
    settings.value = await api.getSettings()
  } finally {
    loading.value = false
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
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="panel settings-tool-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">Database</h2>
            <div class="panel-kicker">Local AgentMeter store</div>
          </div>
          <div class="summary-actions">
            <a-tag :color="databaseState.color" class="status-tag">{{ databaseState.label }}</a-tag>
            <a-tag :color="indexStatus.color" class="status-tag">{{ indexStatus.label }}</a-tag>
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
                <a-tooltip :title="updateIndexHint">
                  <a-button type="primary" :loading="indexing" @click="index(false)">
                    <template #icon>
                      <PlayCircleOutlined />
                    </template>
                    Update Index
                  </a-button>
                </a-tooltip>
                <a-tooltip :title="rebuildIndexHint">
                  <a-button :loading="indexing" @click="index(true)">
                    <template #icon>
                      <ReloadOutlined />
                    </template>
                    Rebuild Index
                  </a-button>
                </a-tooltip>
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
                  <span class="muted">Mode</span>
                  <strong>{{ settings.lastIndexResult.rebuild ? 'Rebuild' : 'Incremental' }}</strong>
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
  </a-spin>
</template>
