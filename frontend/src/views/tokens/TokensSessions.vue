<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import { useRouter } from 'vue-router'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import Typography from 'ant-design-vue/es/typography'
import { HistoryOutlined } from '@ant-design/icons-vue'
import {
  formatCost,
  formatDateTime,
  formatNumber,
  projectDisplay,
  sessionFullLabel,
  sessionLabel,
  type Session
} from '../../api'
import { useMessages } from '../../i18n'
import { sourceDisplay } from '../../presentation/sourceIdentity'
import { useTokensContext } from './tokensContext'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const router = useRouter()
const { analytics, loading } = useTokensContext()
const { t } = useMessages({
  en: {
    'sessions.title': 'High Token Sessions',
    'sessions.kicker': 'Sessions ranked by total token volume',
    'column.session': 'Session',
    'column.project': 'Project',
    'column.started': 'Started',
    'column.tokens': 'Tokens',
    'column.cost': 'Cost',
    'fallback.unknown': 'unknown',
    'empty.loading': 'Loading token analytics...',
    'empty.none': 'No token usage indexed yet'
  },
  'zh-CN': {
    'sessions.title': '高 Token 会话',
    'sessions.kicker': '按总 Token 用量排序的会话',
    'column.session': '会话',
    'column.project': '项目',
    'column.started': '开始',
    'column.tokens': 'Token',
    'column.cost': '费用',
    'fallback.unknown': '未知',
    'empty.loading': '正在加载 Token 分析...',
    'empty.none': '暂无已索引 Token 用量'
  }
})

const sessionColumns = computed(() => [
  { title: t('column.session'), dataIndex: 'sessionKey', key: 'session', width: 220 },
  { title: t('column.project'), dataIndex: 'projectPath', key: 'project' },
  { title: t('column.started'), dataIndex: 'startedAt', key: 'started', width: 140 },
  { title: t('column.tokens'), dataIndex: ['tokenUsage', 'totalTokens'], key: 'tokens', width: 120, align: 'right' },
  { title: t('column.cost'), dataIndex: 'estimatedCostUsd', key: 'cost', width: 110, align: 'right' }
])

const tableLocale = computed(() => ({ emptyText: loading.value ? t('empty.loading') : t('empty.none') }))

function sessionRow(record: Session) {
  return { class: 'is-clickable-row', onClick: () => router.push(`/sessions/${record.id}`) }
}

function sourceInfo(record: Session) {
  return sourceDisplay(record, t('fallback.unknown'))
}

function sessionProject(record: Session) {
  return projectDisplay(record.projectPath || record.rawSourcePath)
}
</script>

<template>
  <a-spin :spinning="loading">
    <section class="panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('sessions.title') }}</h2>
          <div class="panel-kicker">{{ t('sessions.kicker') }}</div>
        </div>
        <HistoryOutlined class="panel-header-icon" />
      </div>
      <a-table
        class="dense-table"
        :columns="sessionColumns"
        :data-source="analytics?.highTokenSessions || []"
        :loading="loading"
        :locale="tableLocale"
        :pagination="{ pageSize: 10 }"
        row-key="id"
        size="small"
        :custom-row="sessionRow"
        :scroll="{ x: 900 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'session'">
            <a-typography-text class="mono" :ellipsis="{ tooltip: sessionFullLabel(record) }">
              {{ sessionLabel(record) }}
            </a-typography-text>
            <div class="source-identity-meta">{{ sourceInfo(record).label }}</div>
          </template>
          <template v-else-if="column.key === 'project'">
            <a-typography-text :ellipsis="{ tooltip: sessionProject(record).full }">
              {{ sessionProject(record).main }}
            </a-typography-text>
          </template>
          <template v-else-if="column.key === 'started'">{{ formatDateTime(record.startedAt) }}</template>
          <template v-else-if="column.key === 'tokens'"><span class="number-cell">{{ formatNumber(record.tokenUsage.totalTokens) }}</span></template>
          <template v-else-if="column.key === 'cost'"><span class="number-cell">{{ formatCost(record.estimatedCostUsd) }}</span></template>
        </template>
      </a-table>
    </section>
  </a-spin>
</template>
