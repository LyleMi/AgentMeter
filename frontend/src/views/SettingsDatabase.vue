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
import { api, formatDateTime, formatDuration, formatNumber, isStaticDemo, type Settings } from '../api'
import { notifyAppDataChanged } from '../events'
import { useMessages } from '../i18n'

const ATooltip = Tooltip
const ATypographyText = Typography.Text
const { t } = useMessages({
  en: {
    'database.title': 'Database',
    'database.kicker': 'Local AgentMeter store',
    'database.action.refresh': 'Refresh',
    'database.action.updateIndex': 'Update Index',
    'database.action.rebuildIndex': 'Rebuild Index',
    'database.action.updateHint': 'Scan enabled sources and parse only new or changed JSONL files.',
    'database.action.rebuildHint': 'Clear indexed files for enabled sources, then parse every JSONL file again.',
    'database.action.demoHint': 'Static demo mode is read-only. Indexing is disabled.',
    'database.meta.path': 'Database path',
    'database.meta.lastRun': 'Last run',
    'database.meta.indexState': 'Index state',
    'database.status.localStore': 'Local store',
    'database.status.noPath': 'No database path',
    'database.status.noIndexRun': 'No index run',
    'database.status.noCompletedRun': 'No completed run yet',
    'database.status.failed': 'Failed',
    'database.status.failedCount': 'failed',
    'database.status.warnings': 'Warnings',
    'database.status.warningsCount': 'warnings',
    'database.status.completed': 'Completed',
    'database.status.indexedCount': 'indexed',
    'database.message.indexed': 'indexed',
    'database.message.skipped': 'skipped',
    'database.message.failed': 'failed',
    'database.message.indexFailed': 'Index failed',
    'database.result.title': 'Last index result',
    'database.result.filesSeen': 'Files seen',
    'database.result.sources': 'Sources',
    'database.result.mode': 'Mode',
    'database.result.modeRebuild': 'Rebuild',
    'database.result.modeIncremental': 'Incremental',
    'database.result.sessions': 'Sessions',
    'database.result.indexed': 'Indexed',
    'database.result.skipped': 'Skipped',
    'database.result.failed': 'Failed',
    'database.result.warnings': 'Warnings',
    'database.result.duration': 'Duration',
    'database.result.empty': 'No index result has been recorded for this database.',
    'database.result.more': 'more'
  },
  'zh-CN': {
    'database.title': '数据库',
    'database.kicker': '本地 AgentMeter 存储',
    'database.action.refresh': '刷新',
    'database.action.updateIndex': '更新索引',
    'database.action.rebuildIndex': '重建索引',
    'database.action.updateHint': '扫描已启用来源，只解析新增或已变更的 JSONL 文件。',
    'database.action.rebuildHint': '清除已启用来源的索引文件，然后重新解析每个 JSONL 文件。',
    'database.action.demoHint': '静态演示模式为只读，索引功能已禁用。',
    'database.meta.path': '数据库路径',
    'database.meta.lastRun': '上次运行',
    'database.meta.indexState': '索引状态',
    'database.status.localStore': '本地存储',
    'database.status.noPath': '无数据库路径',
    'database.status.noIndexRun': '未运行索引',
    'database.status.noCompletedRun': '尚无已完成运行',
    'database.status.failed': '失败',
    'database.status.failedCount': '个失败',
    'database.status.warnings': '警告',
    'database.status.warningsCount': '个警告',
    'database.status.completed': '已完成',
    'database.status.indexedCount': '个已索引',
    'database.message.indexed': '个已索引',
    'database.message.skipped': '个已跳过',
    'database.message.failed': '个失败',
    'database.message.indexFailed': '索引失败',
    'database.result.title': '上次索引结果',
    'database.result.filesSeen': '已扫描文件',
    'database.result.sources': '来源',
    'database.result.mode': '模式',
    'database.result.modeRebuild': '重建',
    'database.result.modeIncremental': '增量',
    'database.result.sessions': '会话',
    'database.result.indexed': '已索引',
    'database.result.skipped': '已跳过',
    'database.result.failed': '失败',
    'database.result.warnings': '警告',
    'database.result.duration': '耗时',
    'database.result.empty': '此数据库尚未记录索引结果。',
    'database.result.more': '条更多'
  }
})

const loading = ref(true)
const indexing = ref(false)
const settings = ref<Settings | null>(null)

const databaseState = computed(() => {
  if (settings.value?.databasePath) return { color: 'success', label: t('database.status.localStore') }
  return { color: 'warning', label: t('database.status.noPath') }
})

const indexStatus = computed(() => {
  const result = settings.value?.lastIndexResult
  if (!result) {
    return {
      color: 'default',
      label: t('database.status.noIndexRun'),
      detail: t('database.status.noCompletedRun')
    }
  }
  const duration = formatDuration(result.durationMs)
  if (result.failed > 0) {
    return {
      color: 'error',
      label: t('database.status.failed'),
      detail: `${formatNumber(result.failed)} ${t('database.status.failedCount')} · ${duration}`
    }
  }
  if ((result.warnings?.length || 0) > 0) {
    return {
      color: 'warning',
      label: t('database.status.warnings'),
      detail: `${formatNumber(result.warnings.length)} ${t('database.status.warningsCount')} · ${duration}`
    }
  }
  return {
    color: 'success',
    label: t('database.status.completed'),
    detail: `${formatNumber(result.indexed)} ${t('database.status.indexedCount')} · ${duration}`
  }
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
  if (isStaticDemo) {
    message.info(t('database.action.demoHint'))
    return
  }
  indexing.value = true
  try {
    const result = await api.indexNow(rebuild)
    message.success(
      `${formatNumber(result.indexed)} ${t('database.message.indexed')}, ${formatNumber(result.skipped)} ${t(
        'database.message.skipped'
      )}, ${formatNumber(result.failed)} ${t('database.message.failed')}`
    )
    await load()
    notifyAppDataChanged('index')
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('database.message.indexFailed'))
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
            <h2 class="panel-title">{{ t('database.title') }}</h2>
            <div class="panel-kicker">{{ t('database.kicker') }}</div>
          </div>
          <div class="summary-actions">
            <a-tag :color="databaseState.color" class="status-tag">{{ databaseState.label }}</a-tag>
            <a-tag :color="indexStatus.color" class="status-tag">{{ indexStatus.label }}</a-tag>
            <a-button @click="load">
              <template #icon>
                <ReloadOutlined />
              </template>
              {{ t('database.action.refresh') }}
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
                <div class="metadata-label">{{ t('database.meta.path') }}</div>
                <div class="metadata-value">
                  <a-typography-text :ellipsis="{ tooltip: settings?.databasePath }">
                    {{ settings?.databasePath || '-' }}
                  </a-typography-text>
                </div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('database.meta.lastRun') }}</div>
                <div class="metadata-value">
                  {{ settings?.lastIndexStartedAt ? formatDateTime(settings.lastIndexStartedAt) : '-' }}
                </div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('database.meta.indexState') }}</div>
                <div class="metadata-value">{{ indexStatus.detail }}</div>
              </div>
            </div>

            <div class="toolbar">
              <div class="toolbar-left">
                <a-tooltip :title="isStaticDemo ? t('database.action.demoHint') : t('database.action.updateHint')">
                  <a-button type="primary" :loading="indexing" :disabled="isStaticDemo" @click="index(false)">
                    <template #icon>
                      <PlayCircleOutlined />
                    </template>
                    {{ t('database.action.updateIndex') }}
                  </a-button>
                </a-tooltip>
                <a-tooltip :title="isStaticDemo ? t('database.action.demoHint') : t('database.action.rebuildHint')">
                  <a-button :loading="indexing" :disabled="isStaticDemo" @click="index(true)">
                    <template #icon>
                      <ReloadOutlined />
                    </template>
                    {{ t('database.action.rebuildIndex') }}
                  </a-button>
                </a-tooltip>
              </div>
            </div>

            <div class="index-result-block">
              <div class="index-result-header">
                <div>
                  <div class="index-result-title">{{ t('database.result.title') }}</div>
                  <div class="muted">
                    {{
                      settings?.lastIndexStartedAt
                        ? formatDateTime(settings.lastIndexStartedAt)
                        : t('database.status.noCompletedRun')
                    }}
                  </div>
                </div>
                <a-tag :color="indexStatus.color" class="status-tag">{{ indexStatus.detail }}</a-tag>
              </div>
              <div v-if="settings?.lastIndexResult" class="index-result-grid">
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.filesSeen') }}</span>
                  <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.filesSeen) }}</strong>
                </div>
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.sources') }}</span>
                  <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.sourcePaths?.length || 0) }}</strong>
                </div>
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.mode') }}</span>
                  <strong>
                    {{
                      settings.lastIndexResult.rebuild
                        ? t('database.result.modeRebuild')
                        : t('database.result.modeIncremental')
                    }}
                  </strong>
                </div>
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.sessions') }}</span>
                  <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.sessions) }}</strong>
                </div>
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.indexed') }}</span>
                  <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.indexed) }}</strong>
                </div>
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.skipped') }}</span>
                  <strong class="number-cell">{{ formatNumber(settings.lastIndexResult.skipped) }}</strong>
                </div>
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.failed') }}</span>
                  <strong class="number-cell" :class="settings.lastIndexResult.failed ? 'status-error' : 'status-ok'">
                    {{ formatNumber(settings.lastIndexResult.failed) }}
                  </strong>
                </div>
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.warnings') }}</span>
                  <strong
                    class="number-cell"
                    :class="settings.lastIndexResult.warnings?.length ? 'status-warning' : 'status-ok'"
                  >
                    {{ formatNumber(settings.lastIndexResult.warnings?.length || 0) }}
                  </strong>
                </div>
                <div class="index-result-metric">
                  <span class="muted">{{ t('database.result.duration') }}</span>
                  <strong class="number-cell duration-cell">
                    {{ formatDuration(settings.lastIndexResult.durationMs) }}
                  </strong>
                </div>
              </div>
              <div v-else class="metric-note">{{ t('database.result.empty') }}</div>
              <div v-if="settings?.lastIndexResult?.warnings?.length" class="index-result-warnings">
                <div class="metadata-label">{{ t('database.result.warnings') }}</div>
                <ul>
                  <li v-for="warning in settings.lastIndexResult.warnings.slice(0, 3)" :key="warning">
                    {{ warning }}
                  </li>
                  <li v-if="settings.lastIndexResult.warnings.length > 3" class="muted">
                    +{{ formatNumber(settings.lastIndexResult.warnings.length - 3) }} {{ t('database.result.more') }}
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
