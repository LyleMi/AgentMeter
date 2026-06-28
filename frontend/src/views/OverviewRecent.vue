<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ClockCircleOutlined } from '@ant-design/icons-vue'
import { formatDateTime, formatNumber, projectDisplay, sessionFullLabel, sessionLabel, type Session } from '../api'
import EmptyState from '../components/ui/EmptyState.vue'
import Panel from '../components/ui/Panel.vue'
import { useMessages } from '../i18n'
import { useOverviewContext } from './overviewContext'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const router = useRouter()
const { overview, loading } = useOverviewContext()
const { t } = useMessages({
  en: {
    'title': 'Recent Sessions',
    'kicker': 'Open a row to inspect timeline and calls',
    'action.viewAll': 'View all',
    'column.agent': 'Agent',
    'column.project': 'Project',
    'column.model': 'Model',
    'column.tokens': 'Tokens',
    'column.tools': 'Tools',
    'column.started': 'Started',
    'empty.title': 'No recent sessions',
    'empty.text': 'Recently indexed sessions will be listed here for quick inspection.',
    'fallback.unknown': 'unknown'
  },
  'zh-CN': {
    'title': '最近会话',
    'kicker': '打开任意行以查看时间线和调用',
    'action.viewAll': '查看全部',
    'column.agent': 'Agent',
    'column.project': '项目',
    'column.model': '模型',
    'column.tokens': 'Token',
    'column.tools': '工具',
    'column.started': '开始时间',
    'empty.title': '暂无最近会话',
    'empty.text': '最近索引的会话会在这里列出，便于快速查看。',
    'fallback.unknown': '未知'
  }
})

const hasRecentSessions = computed(() => (overview.value?.recentSessions?.length || 0) > 0)

const recentColumns = computed(() => [
  { title: t('column.agent'), dataIndex: 'agentName', key: 'agent', width: 132 },
  { title: t('column.project'), dataIndex: 'projectPath', key: 'projectPath' },
  { title: t('column.model'), dataIndex: 'model', key: 'model', width: 132 },
  { title: t('column.tokens'), dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120, align: 'right' },
  { title: t('column.tools'), dataIndex: 'toolCallCount', key: 'tools', width: 80, align: 'right' },
  { title: t('column.started'), dataIndex: 'startedAt', key: 'startedAt', width: 150 }
])

function openSession(id: number) {
  router.push(`/sessions/${id}`)
}

function recentRow(record: Session) {
  return { class: 'overview-session-row is-clickable-row', onClick: () => openSession(record.id) }
}

function projectInfo(record: Session) {
  return projectDisplay(record.projectPath)
}
</script>

<template>
  <a-spin :spinning="loading">
    <Panel class="overview-recent-panel" :title="t('title')" :kicker="t('kicker')">
      <template #actions>
        <a-button type="link" @click="$router.push('/sessions')">{{ t('action.viewAll') }}</a-button>
      </template>
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
            <a-tag class="model-lite-tag">{{ record.agentName || record.agentKind || t('fallback.unknown') }}</a-tag>
          </template>
          <template v-else-if="column.key === 'projectPath'">
            <div class="overview-session-identity">
              <a-typography-text class="overview-session-project" :ellipsis="{ tooltip: record.projectPath }">
                {{ projectInfo(record).main }}
              </a-typography-text>
              <a-typography-text class="overview-session-meta mono" :ellipsis="{ tooltip: sessionFullLabel(record) }">
                {{ sessionLabel(record) }}
              </a-typography-text>
            </div>
          </template>
          <template v-else-if="column.key === 'model'">
            <a-tag class="model-lite-tag">{{ record.model || t('fallback.unknown') }}</a-tag>
          </template>
          <template v-else-if="column.key === 'tokens'">
            <span class="number-cell">{{ formatNumber(record.tokenUsage.totalTokens) }}</span>
          </template>
          <template v-else-if="column.key === 'tools'">
            <span class="number-cell">{{ formatNumber(record.toolCallCount) }}</span>
          </template>
        </template>
      </a-table>
      <EmptyState
        v-else
        :title="t('empty.title')"
        :text="t('empty.text')"
        :icon="ClockCircleOutlined"
        compact
      />
    </Panel>
  </a-spin>
</template>
