<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AButton from 'ant-design-vue/es/button'
import AConfigProvider from 'ant-design-vue/es/config-provider'
import Layout from 'ant-design-vue/es/layout'
import Menu from 'ant-design-vue/es/menu'
import message from 'ant-design-vue/es/message'
import Tooltip from 'ant-design-vue/es/tooltip'
import Typography from 'ant-design-vue/es/typography'
import {
  BarChartOutlined,
  HistoryOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SettingOutlined,
  ToolOutlined
} from '@ant-design/icons-vue'
import { api, type Settings } from './api'
import { APP_DATA_CHANGED_EVENT, type AppDataChangeDetail } from './events'

const ALayout = Layout
const ALayoutContent = Layout.Content
const ALayoutHeader = Layout.Header
const ALayoutSider = Layout.Sider
const AMenu = Menu
const AMenuItem = Menu.Item
const ATooltip = Tooltip
const ATypographyText = Typography.Text

const route = useRoute()
const router = useRouter()
const settings = ref<Settings | null>(null)
const indexing = ref(false)
const refreshKey = ref(0)
const sidebarCollapsed = ref(false)

const hasSource = computed(() => Boolean(settings.value?.sourcePaths?.length || settings.value?.sourcePath))
const sourceStatusLabel = computed(() => (hasSource.value ? 'Sources ready' : 'Sources missing'))
const sourcePathDisplay = computed(() => settings.value?.sourcePath || 'Configure local JSONL sources in Settings')
const sourceSummary = computed(() => {
  const count = settings.value?.sourcePaths?.length || 0
  if (count > 1) return `${count} local sources`
  return sourcePathDisplay.value
})
const sidebarToggleLabel = computed(() => (sidebarCollapsed.value ? 'Expand sidebar' : 'Collapse sidebar'))
const updateIndexHint = 'Scan enabled sources and parse only new or changed JSONL files.'
const rebuildIndexHint = 'Clear indexed files for enabled sources, then parse every JSONL file again.'

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

async function handleAppDataChanged(event: Event) {
  const detail = (event as CustomEvent<AppDataChangeDetail>).detail
  await loadSettings()
  if (detail?.reason === 'index') {
    refreshKey.value += 1
  }
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

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

onMounted(() => {
  loadSettings()
  window.addEventListener(APP_DATA_CHANGED_EVENT, handleAppDataChanged)
})
onBeforeUnmount(() => {
  window.removeEventListener(APP_DATA_CHANGED_EVENT, handleAppDataChanged)
})
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
      <a-layout-sider
        class="app-sider"
        :class="{ 'is-collapsed': sidebarCollapsed }"
        width="216"
        :collapsed-width="68"
        :collapsed="sidebarCollapsed"
        collapsible
        :trigger="null"
      >
        <div class="brand">
          <div class="brand-mark">
            <img class="brand-logo" src="/favicon.png" alt="AgentMeter" />
          </div>
          <div class="brand-copy">
            <div class="brand-title">AgentMeter</div>
            <div class="brand-subtitle">Local agent usage</div>
          </div>
        </div>
        <div class="nav-section-head">
          <div class="nav-section-label">Inspect</div>
          <a-tooltip :title="sidebarToggleLabel" placement="right">
            <a-button
              class="sidebar-toggle"
              type="text"
              :aria-expanded="!sidebarCollapsed"
              :aria-label="sidebarToggleLabel"
              @click="toggleSidebar"
            >
              <template #icon>
                <component :is="sidebarCollapsed ? MenuUnfoldOutlined : MenuFoldOutlined" />
              </template>
            </a-button>
          </a-tooltip>
        </div>
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
            <span class="source-label">Local sources</span>
            <a-typography-text class="source-path" :ellipsis="{ tooltip: sourcePathDisplay }">
              {{ sourceSummary }}
            </a-typography-text>
          </div>
          <div class="header-actions">
            <a-tooltip :title="updateIndexHint" placement="bottom">
              <a-button type="primary" :loading="indexing" @click="indexNow(false)">
                <template #icon>
                  <PlayCircleOutlined />
                </template>
                Update Index
              </a-button>
            </a-tooltip>
            <a-tooltip :title="rebuildIndexHint" placement="bottom">
              <a-button :loading="indexing" @click="indexNow(true)">
                <template #icon>
                  <ReloadOutlined />
                </template>
                Rebuild Index
              </a-button>
            </a-tooltip>
          </div>
        </a-layout-header>

        <a-layout-content class="app-content">
          <router-view :key="`${route.fullPath}:${refreshKey}`" />
        </a-layout-content>
      </a-layout>
    </a-layout>
  </a-config-provider>
</template>
