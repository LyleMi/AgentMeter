<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  BarChartOutlined,
  DatabaseOutlined,
  HistoryOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SettingOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import { api, type Settings } from './api'

const route = useRoute()
const router = useRouter()
const settings = ref<Settings | null>(null)
const indexing = ref(false)
const refreshKey = ref(0)

const hasSource = computed(() => Boolean(settings.value?.sourcePath))
const sourceStatusLabel = computed(() => (hasSource.value ? 'Source ready' : 'Source missing'))
const sourcePathDisplay = computed(() => settings.value?.sourcePath || 'Configure a local JSONL source in Settings')

const selectedKeys = computed(() => {
  if (route.path.startsWith('/sessions')) return ['sessions']
  if (route.path.startsWith('/tools')) return ['tools']
  if (route.path.startsWith('/settings')) return ['settings']
  return ['overview']
})

const menuItems = [
  { key: 'overview', icon: BarChartOutlined, label: 'Overview', path: '/overview' },
  { key: 'sessions', icon: HistoryOutlined, label: 'Sessions', path: '/sessions' },
  { key: 'tools', icon: ToolOutlined, label: 'Tools', path: '/tools' },
  { key: 'settings', icon: SettingOutlined, label: 'Settings', path: '/settings' }
]

async function loadSettings() {
  settings.value = await api.getSettings()
}

async function indexNow(rebuild = false) {
  indexing.value = true
  try {
    const result = await api.indexNow(rebuild)
    message.success(`${result.indexed} indexed, ${result.skipped} skipped, ${result.failed} failed`)
    await loadSettings()
    refreshKey.value += 1
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Index failed')
  } finally {
    indexing.value = false
  }
}

function navigate(path: string) {
  router.push(path)
}

onMounted(loadSettings)
</script>

<template>
  <a-config-provider
    :theme="{
      token: {
        colorPrimary: '#1d4ed8',
        colorInfo: '#0891b2',
        colorSuccess: '#0f766e',
        colorWarning: '#b45309',
        colorError: '#b91c1c',
        borderRadius: 8,
        fontFamily:
          'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif'
      }
    }"
  >
    <a-layout class="app-shell">
      <a-layout-sider class="app-sider" width="216">
        <div class="brand">
          <div class="brand-mark">
            <DatabaseOutlined />
          </div>
          <div class="brand-copy">
            <div class="brand-title">AgentMeter</div>
            <div class="brand-subtitle">Local Codex usage</div>
          </div>
        </div>
        <div class="nav-section-label">Inspect</div>
        <a-menu class="nav-menu" mode="inline" :selected-keys="selectedKeys">
          <a-menu-item v-for="item in menuItems" :key="item.key" @click="navigate(item.path)">
            <template #icon>
              <component :is="item.icon" />
            </template>
            {{ item.label }}
          </a-menu-item>
        </a-menu>
      </a-layout-sider>

      <a-layout>
        <a-layout-header class="app-header">
          <div class="source-bar" :class="{ 'is-configured': hasSource, 'is-missing': !hasSource }">
            <div class="source-status-chip">
              <span class="source-dot"></span>
              <span>{{ sourceStatusLabel }}</span>
            </div>
            <span class="source-label">Local source</span>
            <a-typography-text class="source-path" :ellipsis="{ tooltip: sourcePathDisplay }">
              {{ sourcePathDisplay }}
            </a-typography-text>
          </div>
          <div class="header-actions">
            <a-button type="primary" :loading="indexing" @click="indexNow(false)">
              <template #icon>
                <PlayCircleOutlined />
              </template>
              Index Now
            </a-button>
            <a-button :loading="indexing" @click="indexNow(true)">
              <template #icon>
                <ReloadOutlined />
              </template>
              Rebuild
            </a-button>
          </div>
        </a-layout-header>

        <a-layout-content class="app-content">
          <router-view :key="`${route.fullPath}:${refreshKey}`" />
        </a-layout-content>
      </a-layout>
    </a-layout>
  </a-config-provider>
</template>
