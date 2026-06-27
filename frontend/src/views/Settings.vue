<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import { DatabaseOutlined, DollarOutlined, FolderOpenOutlined } from '@ant-design/icons-vue'
import { useMessages } from '../i18n'

const route = useRoute()
const router = useRouter()
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

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('settings.title') }}</h1>
        <div class="page-subtitle">{{ t('settings.subtitle') }}</div>
      </div>
    </div>

    <div class="settings-subnav">
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
