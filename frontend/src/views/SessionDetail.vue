<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import {
  api,
  formatCost,
  formatDateTime,
  formatDuration,
  formatNumber,
  shortPath,
  type EventItem,
  type SessionDetail
} from '../api'

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const detail = ref<SessionDetail | null>(null)

const modelColumns = [
  { title: 'Ended', dataIndex: 'endedAt', key: 'endedAt', width: 150 },
  { title: 'Model', dataIndex: 'model', key: 'model' },
  { title: 'Duration', dataIndex: 'durationMs', key: 'duration', width: 110 },
  { title: 'Input', dataIndex: 'inputTokens', key: 'input', width: 110 },
  { title: 'Output', dataIndex: 'outputTokens', key: 'output', width: 110 },
  { title: 'Cost', dataIndex: 'costUsd', key: 'cost', width: 120 }
]

const toolColumns = [
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 150 },
  { title: 'Tool', dataIndex: 'toolName', key: 'toolName', width: 160 },
  { title: 'Status', dataIndex: 'status', key: 'status', width: 110 },
  { title: 'Duration', dataIndex: 'durationMs', key: 'duration', width: 110 },
  { title: 'Input', dataIndex: 'inputSummary', key: 'input' },
  { title: 'Output', dataIndex: 'outputSummary', key: 'output' }
]

const rawColumns = [
  { title: 'Line', dataIndex: 'sourceLine', key: 'line', width: 80 },
  { title: 'Time', dataIndex: 'timestamp', key: 'time', width: 150 },
  { title: 'Kind', dataIndex: 'kind', key: 'kind', width: 100 },
  { title: 'Type', dataIndex: 'rawType', key: 'rawType', width: 150 },
  { title: 'Summary', dataIndex: 'summary', key: 'summary' }
]

const events = computed<EventItem[]>(() => detail.value?.events || [])

async function load() {
  loading.value = true
  try {
    detail.value = await api.getSessionDetail(Number(route.params.id))
  } finally {
    loading.value = false
  }
}

function statusColor(status: string) {
  if (status === 'completed' || status === 'ok') return 'green'
  if (status === 'pending' || status === 'warning') return 'orange'
  return 'red'
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <a-button type="text" @click="router.push('/sessions')">
          <template #icon>
            <ArrowLeftOutlined />
          </template>
          Sessions
        </a-button>
        <h1 class="page-title">
          {{ detail?.session.codexSessionId || 'Session Detail' }}
        </h1>
        <div class="page-subtitle">
          {{ shortPath(detail?.session.projectPath || '') }}
        </div>
      </div>
      <a-button @click="load">Refresh</a-button>
    </div>

    <a-spin :spinning="loading">
      <template v-if="detail">
        <div class="metric-grid">
          <a-card class="metric-card" :bordered="false">
            <div class="metric-label">Tokens</div>
            <div class="metric-value">{{ formatNumber(detail.session.tokenUsage.totalTokens) }}</div>
            <div class="metric-note">{{ detail.session.tokenUsage.source }}</div>
          </a-card>
          <a-card class="metric-card" :bordered="false">
            <div class="metric-label">Estimated Cost</div>
            <div class="metric-value">{{ formatCost(detail.session.estimatedCostUsd) }}</div>
            <div class="metric-note">{{ detail.session.model }}</div>
          </a-card>
          <a-card class="metric-card" :bordered="false">
            <div class="metric-label">Timing</div>
            <div class="metric-value">{{ formatDuration(detail.session.wallDurationMs) }}</div>
            <div class="metric-note">{{ formatDuration(detail.session.activeDurationMs) }} active</div>
          </a-card>
          <a-card class="metric-card" :bordered="false">
            <div class="metric-label">Tools</div>
            <div class="metric-value">{{ formatNumber(detail.session.toolCallCount) }}</div>
            <div class="metric-note">{{ formatNumber(detail.session.eventCount) }} events</div>
          </a-card>
        </div>

        <section class="panel" style="margin-bottom: 18px">
          <div class="panel-header">
            <h2 class="panel-title">Metadata</h2>
          </div>
          <div class="panel-body">
            <a-descriptions size="small" bordered :column="2">
              <a-descriptions-item label="Started">{{ formatDateTime(detail.session.startedAt) }}</a-descriptions-item>
              <a-descriptions-item label="Ended">{{ formatDateTime(detail.session.endedAt) }}</a-descriptions-item>
              <a-descriptions-item label="Model">{{ detail.session.model }}</a-descriptions-item>
              <a-descriptions-item label="Provider">{{ detail.session.modelProvider || '-' }}</a-descriptions-item>
              <a-descriptions-item label="Project" :span="2">
                <a-typography-text class="detail-path" :ellipsis="{ tooltip: detail.session.projectPath }">
                  {{ detail.session.projectPath }}
                </a-typography-text>
              </a-descriptions-item>
              <a-descriptions-item label="Raw source" :span="2">
                <a-typography-text class="detail-path mono" :ellipsis="{ tooltip: detail.session.rawSourcePath }">
                  {{ detail.session.rawSourcePath }}
                </a-typography-text>
              </a-descriptions-item>
            </a-descriptions>
          </div>
        </section>

        <div class="split-row">
          <section class="panel">
            <div class="panel-header">
              <h2 class="panel-title">Timeline</h2>
            </div>
            <div class="panel-body timeline-list">
              <a-timeline>
                <a-timeline-item v-for="event in events" :key="event.id">
                  <div>
                    <a-tag>{{ event.kind }}</a-tag>
                    <span class="muted">{{ formatDateTime(event.timestamp) }}</span>
                  </div>
                  <div style="margin-top: 4px">{{ event.summary }}</div>
                  <div class="muted mono" style="margin-top: 2px">line {{ event.sourceLine }} · {{ event.rawType }}</div>
                </a-timeline-item>
              </a-timeline>
            </div>
          </section>

          <section class="panel">
            <div class="panel-header">
              <h2 class="panel-title">Calls</h2>
            </div>
            <a-tabs class="panel-body" :tab-bar-style="{ marginBottom: '14px' }">
              <a-tab-pane key="model" tab="Model">
                <a-table
                  size="small"
                  :columns="modelColumns"
                  :data-source="detail.modelCalls"
                  :pagination="{ pageSize: 8 }"
                  row-key="id"
                >
                  <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'endedAt'">{{ formatDateTime(record.endedAt) }}</template>
                    <template v-else-if="column.key === 'duration'">{{ formatDuration(record.durationMs) }}</template>
                    <template v-else-if="column.key === 'input'">{{ formatNumber(record.inputTokens) }}</template>
                    <template v-else-if="column.key === 'output'">{{ formatNumber(record.outputTokens) }}</template>
                    <template v-else-if="column.key === 'cost'">{{ formatCost(record.costUsd) }}</template>
                  </template>
                </a-table>
              </a-tab-pane>
              <a-tab-pane key="tools" tab="Tools">
                <a-table
                  size="small"
                  :columns="toolColumns"
                  :data-source="detail.toolCalls"
                  :pagination="{ pageSize: 8 }"
                  row-key="id"
                >
                  <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'startedAt'">{{ formatDateTime(record.startedAt) }}</template>
                    <template v-else-if="column.key === 'status'">
                      <a-tag :color="statusColor(record.status)">{{ record.status }}</a-tag>
                    </template>
                    <template v-else-if="column.key === 'duration'">{{ formatDuration(record.durationMs) }}</template>
                    <template v-else-if="column.key === 'input'">
                      <a-typography-text :ellipsis="{ tooltip: record.inputSummary }">
                        {{ record.inputSummary || '-' }}
                      </a-typography-text>
                    </template>
                    <template v-else-if="column.key === 'output'">
                      <a-typography-text :ellipsis="{ tooltip: record.outputSummary || record.error }">
                        {{ record.outputSummary || record.error || '-' }}
                      </a-typography-text>
                    </template>
                  </template>
                </a-table>
              </a-tab-pane>
              <a-tab-pane key="events" tab="Raw Events">
                <a-table
                  size="small"
                  :columns="rawColumns"
                  :data-source="events"
                  :pagination="{ pageSize: 10 }"
                  row-key="id"
                >
                  <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'time'">{{ formatDateTime(record.timestamp) }}</template>
                    <template v-else-if="column.key === 'summary'">
                      <a-typography-text :ellipsis="{ tooltip: record.summary }">
                        {{ record.summary }}
                      </a-typography-text>
                    </template>
                  </template>
                </a-table>
              </a-tab-pane>
            </a-tabs>
          </section>
        </div>
      </template>
    </a-spin>
  </div>
</template>
