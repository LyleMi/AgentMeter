<script setup lang="ts">
import { computed, onMounted, provide, ref } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import message from 'ant-design-vue/es/message'
import {
  BarChartOutlined,
  ClockCircleOutlined,
  DatabaseOutlined,
  FieldTimeOutlined,
  HistoryOutlined,
  ReloadOutlined
} from '@ant-design/icons-vue'
import { api, type Overview, type Settings } from '../api'
import PageHeader from '../components/PageHeader.vue'
import PageTabs from '../components/PageTabs.vue'
import { notifyAppDataChanged } from '../events'
import { useMessages } from '../i18n'
import { overviewContextKey, type OverviewContext } from './overviewContext'

const route = useRoute()
const loading = ref(true)
const startupIndexing = ref(false)
const overview = ref<Overview | null>(null)
const settings = ref<Settings | null>(null)
const { t } = useMessages({
  en: {
    'title': 'Overview',
    'subtitle': 'Indexed coding-agent usage across local JSONL sessions',
    'action.refresh': 'Refresh',
    'tab.summary': 'Summary',
    'tab.trends': 'Trends',
    'tab.time': 'Time',
    'tab.breakdown': 'Breakdown',
    'tab.recent': 'Recent',
    'message.indexed': '{indexed} indexed, {skipped} skipped, {failed} failed',
    'message.indexFailed': 'Index failed'
  },
  'zh-CN': {
    'title': '概览',
    'subtitle': '基于本地 JSONL 会话索引的编码代理用量',
    'action.refresh': '刷新',
    'tab.summary': '汇总',
    'tab.trends': '趋势',
    'tab.time': '时间',
    'tab.breakdown': '拆分',
    'tab.recent': '最近',
    'message.indexed': '已索引 {indexed}，已跳过 {skipped}，失败 {failed}',
    'message.indexFailed': '索引失败'
  }
})

const hasIndexedData = computed(() => (overview.value?.totalSessions || 0) > 0)
const sourcePathDisplay = computed(() => settings.value?.sourcePath || settings.value?.defaultSourcePath || '')

const tabs = computed(() => [
  { key: 'summary', label: t('tab.summary'), path: '/overview/summary', icon: BarChartOutlined },
  { key: 'trends', label: t('tab.trends'), path: '/overview/trends', icon: ClockCircleOutlined },
  { key: 'time', label: t('tab.time'), path: '/overview/time', icon: FieldTimeOutlined },
  { key: 'breakdown', label: t('tab.breakdown'), path: '/overview/breakdown', icon: DatabaseOutlined },
  { key: 'recent', label: t('tab.recent'), path: '/overview/recent', icon: HistoryOutlined }
])

const activeKey = computed(() => {
  if (route.path.startsWith('/overview/trends')) return 'trends'
  if (route.path.startsWith('/overview/time')) return 'time'
  if (route.path.startsWith('/overview/breakdown')) return 'breakdown'
  if (route.path.startsWith('/overview/recent')) return 'recent'
  return 'summary'
})

async function load() {
  loading.value = true
  try {
    const [settingsValue, overviewValue] = await Promise.all([api.getSettings(), api.getOverview()])
    settings.value = settingsValue
    overview.value = overviewValue
  } finally {
    loading.value = false
  }
}

async function indexFromOverview() {
  startupIndexing.value = true
  try {
    const result = await api.indexNow(false)
    message.success(
      t('message.indexed', {
        indexed: result.indexed,
        skipped: result.skipped,
        failed: result.failed
      })
    )
    await load()
    notifyAppDataChanged('index')
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('message.indexFailed'))
  } finally {
    startupIndexing.value = false
  }
}

const context: OverviewContext = {
  overview,
  settings,
  loading,
  startupIndexing,
  hasIndexedData,
  sourcePathDisplay,
  load,
  indexFromOverview
}

provide(overviewContextKey, context)

onMounted(load)
</script>

<template>
  <div class="page">
    <PageHeader :title="t('title')" :subtitle="t('subtitle')">
      <template #actions>
        <a-button :loading="loading" @click="load">
          <template #icon>
            <ReloadOutlined />
          </template>
          {{ t('action.refresh') }}
        </a-button>
      </template>
    </PageHeader>

    <PageTabs class="overview-subnav" :tabs="tabs" :active-key="activeKey" />

    <RouterView />
  </div>
</template>
