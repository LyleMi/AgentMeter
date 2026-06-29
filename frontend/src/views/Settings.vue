<script setup lang="ts">
import { computed } from 'vue'
import { DatabaseOutlined, DollarOutlined, EyeOutlined, FolderOpenOutlined } from '@ant-design/icons-vue'
import RoutedPageShell from '../components/RoutedPageShell.vue'
import { useMessages } from '../i18n'
import { useRouteTabKey } from './routeTabs'

const settingsTabMatches = [
  { key: 'database', pathPrefix: '/settings/database' },
  { key: 'price', pathPrefix: '/settings/price' },
  { key: 'display', pathPrefix: '/settings/display' }
] as const

const { t } = useMessages({
  en: {
    'settings.title': 'Settings',
    'settings.subtitle': 'Source, database and price configuration',
    'settings.tab.source': 'Source',
    'settings.tab.database': 'Database',
    'settings.tab.price': 'Price',
    'settings.tab.display': 'Display'
  },
  'zh-CN': {
    'settings.title': '设置',
    'settings.subtitle': '来源、数据库和价格配置',
    'settings.tab.source': '来源',
    'settings.tab.database': '数据库',
    'settings.tab.price': '价格',
    'settings.tab.display': '显示'
  }
})

const tabs = computed(() => [
  { key: 'source', label: t('settings.tab.source'), path: '/settings/source', icon: FolderOpenOutlined },
  { key: 'database', label: t('settings.tab.database'), path: '/settings/database', icon: DatabaseOutlined },
  { key: 'price', label: t('settings.tab.price'), path: '/settings/price', icon: DollarOutlined },
  { key: 'display', label: t('settings.tab.display'), path: '/settings/display', icon: EyeOutlined }
])

const activeKey = useRouteTabKey(settingsTabMatches, 'source')

</script>

<template>
  <RoutedPageShell
    :title="t('settings.title')"
    :subtitle="t('settings.subtitle')"
    :tabs="tabs"
    :active-key="activeKey"
  />
</template>
