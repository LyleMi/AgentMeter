<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { DatabaseOutlined, DollarOutlined, FolderOpenOutlined } from '@ant-design/icons-vue'
import PageHeader from '../components/PageHeader.vue'
import PageTabs from '../components/PageTabs.vue'
import { useMessages } from '../i18n'

const route = useRoute()
const { t } = useMessages({
  en: {
    'settings.title': 'Settings',
    'settings.subtitle': 'Source, database and price configuration',
    'settings.tab.source': 'Source',
    'settings.tab.database': 'Database',
    'settings.tab.price': 'Price'
  },
  'zh-CN': {
    'settings.title': '设置',
    'settings.subtitle': '来源、数据库和价格配置',
    'settings.tab.source': '来源',
    'settings.tab.database': '数据库',
    'settings.tab.price': '价格'
  }
})

const tabs = computed(() => [
  { key: 'source', label: t('settings.tab.source'), path: '/settings/source', icon: FolderOpenOutlined },
  { key: 'database', label: t('settings.tab.database'), path: '/settings/database', icon: DatabaseOutlined },
  { key: 'price', label: t('settings.tab.price'), path: '/settings/price', icon: DollarOutlined }
])

const activeKey = computed(() => {
  if (route.path.startsWith('/settings/database')) return 'database'
  if (route.path.startsWith('/settings/price')) return 'price'
  return 'source'
})

</script>

<template>
  <div class="page">
    <PageHeader :title="t('settings.title')" :subtitle="t('settings.subtitle')" />

    <PageTabs :tabs="tabs" :active-key="activeKey" />

    <RouterView />
  </div>
</template>
