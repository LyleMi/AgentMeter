<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ClockCircleOutlined } from '@ant-design/icons-vue'
import { formatDateTime, formatNumber, sessionLabel, shortPath, type Session } from '../api'
import { useOverviewContext } from './overviewContext'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const router = useRouter()
const { overview, loading } = useOverviewContext()

const hasRecentSessions = computed(() => (overview.value?.recentSessions?.length || 0) > 0)

const recentColumns = [
  { title: 'Agent', dataIndex: 'agentName', key: 'agent', width: 132 },
  { title: 'Project', dataIndex: 'projectPath', key: 'projectPath' },
  { title: 'Model', dataIndex: 'model', key: 'model', width: 132 },
  { title: 'Tokens', dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120, align: 'right' },
  { title: 'Tools', dataIndex: 'toolCallCount', key: 'tools', width: 80, align: 'right' },
  { title: 'Started', dataIndex: 'startedAt', key: 'startedAt', width: 150 }
]

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function recentRow(record: Session) {
  return { class: 'overview-session-row is-clickable-row', onClick: () => openSession(record.id) }
}
</script>

<template>
  <a-spin :spinning="loading">
    <section class="panel overview-recent-panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">Recent Sessions</h2>
          <div class="panel-kicker">Open a row to inspect timeline and calls</div>
        </div>
        <a-button type="link" @click="$router.push('/sessions')">View all</a-button>
      </div>
      <a-table
        v-if="hasRecentSessions"
        class="overview-session-table"
        size="middle"
        :columns="recentColumns"
        :data-source="overview?.recentSessions || []"
        :pagination="false"
        row-key="id"
        :custom-row="recentRow"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'startedAt'">
            {{ formatDateTime(record.startedAt) }}
          </template>
          <template v-else-if="column.key === 'agent'">
            <a-tag class="model-lite-tag">{{ record.agentName || record.agentKind || 'unknown' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'projectPath'">
            <div class="overview-session-identity">
              <a-typography-text class="overview-session-project" :ellipsis="{ tooltip: record.projectPath }">
                {{ shortPath(record.projectPath) }}
              </a-typography-text>
              <span class="overview-session-meta mono">{{ sessionLabel(record) }}</span>
            </div>
          </template>
          <template v-else-if="column.key === 'model'">
            <a-tag class="model-lite-tag">{{ record.model || 'unknown' }}</a-tag>
          </template>
          <template v-else-if="column.key === 'tokens'">
            <span class="number-cell">{{ formatNumber(record.tokenUsage.totalTokens) }}</span>
          </template>
          <template v-else-if="column.key === 'tools'">
            <span class="number-cell">{{ formatNumber(record.toolCallCount) }}</span>
          </template>
        </template>
      </a-table>
      <div v-else class="empty-state empty-state-compact">
        <ClockCircleOutlined class="empty-state-icon" />
        <div class="empty-state-title">No recent sessions</div>
        <div class="empty-state-text">Recently indexed sessions will be listed here for quick inspection.</div>
      </div>
    </section>
  </a-spin>
</template>
