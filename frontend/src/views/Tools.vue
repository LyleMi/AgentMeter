<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import { BarChartOutlined, HistoryOutlined, ToolOutlined } from '@ant-design/icons-vue'

const route = useRoute()
const router = useRouter()

const tabs = [
  { key: 'overview', label: 'Overview', path: '/tools/overview', icon: BarChartOutlined },
  { key: 'summary', label: 'Summary', path: '/tools/summary', icon: ToolOutlined },
  { key: 'calls', label: 'Calls', path: '/tools/calls', icon: HistoryOutlined }
]

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
        <h1 class="page-title">Tools</h1>
        <div class="page-subtitle">Aggregated tool-call counts, status, duration and raw call records</div>
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
