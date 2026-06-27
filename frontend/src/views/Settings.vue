<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import { DatabaseOutlined, DollarOutlined, FolderOpenOutlined } from '@ant-design/icons-vue'

const route = useRoute()
const router = useRouter()

const tabs = [
  { key: 'source', label: 'Source', path: '/settings/source', icon: FolderOpenOutlined },
  { key: 'database', label: 'Database', path: '/settings/database', icon: DatabaseOutlined },
  { key: 'price', label: 'Price', path: '/settings/price', icon: DollarOutlined }
]

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
        <h1 class="page-title">Settings</h1>
        <div class="page-subtitle">Source, database and price configuration</div>
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
