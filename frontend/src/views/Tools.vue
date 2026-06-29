<script setup lang="ts">
import { computed } from 'vue'
import { BarChartOutlined, CodeOutlined, HistoryOutlined, ToolOutlined } from '@ant-design/icons-vue'
import RoutedPageShell from '../components/RoutedPageShell.vue'
import { useMessages } from '../i18n'
import { useRouteTabKey } from './routeTabs'

const toolsTabMatches = [
  { key: 'summary', pathPrefix: '/tools/summary' },
  { key: 'shell', pathPrefix: '/tools/shell' },
  { key: 'calls', pathPrefix: '/tools/calls' }
] as const

const { t } = useMessages({
  en: {
    pageTitle: 'Tools',
    pageSubtitle: 'Aggregated tool-call counts, status, duration and raw call records',
    tabOverview: 'Overview',
    tabSummary: 'Summary',
    tabShell: 'Shell',
    tabCalls: 'Calls'
  },
  'zh-CN': {
    pageTitle: '\u5de5\u5177',
    pageSubtitle: '\u6c47\u603b\u5de5\u5177\u8c03\u7528\u6b21\u6570\u3001\u72b6\u6001\u3001\u8017\u65f6\u548c\u539f\u59cb\u8c03\u7528\u8bb0\u5f55',
    tabOverview: '\u6982\u89c8',
    tabSummary: '\u6c47\u603b',
    tabShell: 'Shell \u547d\u4ee4',
    tabCalls: '\u8c03\u7528'
  }
})

const tabs = computed(() => [
  { key: 'overview', label: t('tabOverview'), path: '/tools/overview', icon: BarChartOutlined },
  { key: 'summary', label: t('tabSummary'), path: '/tools/summary', icon: ToolOutlined },
  { key: 'shell', label: t('tabShell'), path: '/tools/shell', icon: CodeOutlined },
  { key: 'calls', label: t('tabCalls'), path: '/tools/calls', icon: HistoryOutlined }
])

const activeKey = useRouteTabKey(toolsTabMatches, 'overview')

</script>

<template>
  <RoutedPageShell
    :title="t('pageTitle')"
    :subtitle="t('pageSubtitle')"
    :tabs="tabs"
    :active-key="activeKey"
  />
</template>
