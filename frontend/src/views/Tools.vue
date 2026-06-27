<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import { BarChartOutlined, HistoryOutlined, ToolOutlined } from '@ant-design/icons-vue'
import { useMessages } from '../i18n'

const route = useRoute()
const router = useRouter()
const { t } = useMessages({
  en: {
    pageTitle: 'Tools',
    pageSubtitle: 'Aggregated tool-call counts, status, duration and raw call records',
    tabOverview: 'Overview',
    tabSummary: 'Summary',
    tabCalls: 'Calls'
  },
  'zh-CN': {
    pageTitle: '\u5de5\u5177',
    pageSubtitle: '\u6c47\u603b\u5de5\u5177\u8c03\u7528\u6b21\u6570\u3001\u72b6\u6001\u3001\u8017\u65f6\u548c\u539f\u59cb\u8c03\u7528\u8bb0\u5f55',
    tabOverview: '\u6982\u89c8',
    tabSummary: '\u6c47\u603b',
    tabCalls: '\u8c03\u7528'
  }
})

const tabs = computed(() => [
  { key: 'overview', label: t('tabOverview'), path: '/tools/overview', icon: BarChartOutlined },
  { key: 'summary', label: t('tabSummary'), path: '/tools/summary', icon: ToolOutlined },
  { key: 'calls', label: t('tabCalls'), path: '/tools/calls', icon: HistoryOutlined }
])

const activeKey = computed(() => {
  if (route.path.startsWith('/tools/summary')) return 'summary'
  if (route.path.startsWith('/tools/calls')) return 'calls'
  return 'overview'
})

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('pageTitle') }}</h1>
        <div class="page-subtitle">{{ t('pageSubtitle') }}</div>
      </div>
    </div>

    <div class="settings-subnav tools-subnav">
      <a-button
        v-for="item in tabs"
        :key="item.key"
        :type="item.key === activeKey ? 'primary' : 'default'"
        @click="navigate(item.path)"
      >
        <template #icon>
          <component :is="item.icon" />
        </template>
        {{ item.label }}
      </a-button>
    </div>

    <RouterView />
  </div>
</template>
