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
  ToolOutlined,
  WarningOutlined
} from '@ant-design/icons-vue'
import { api, type Settings } from './api'
import { APP_DATA_CHANGED_EVENT, type AppDataChangeDetail } from './events'
import { antDesignLocale, currentLocale, localeOptions, setLocale, useMessages } from './i18n'

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
const { t } = useMessages({
  en: {
    'brand.subtitle': 'Local agent usage',
    'nav.section': 'Inspect',
    'nav.overview': 'Overview',
    'nav.sessions': 'Sessions',
    'nav.tools': 'Tools',
    'nav.audit': 'Audit',
    'nav.settings': 'Settings',
    'sidebar.expand': 'Expand sidebar',
    'sidebar.collapse': 'Collapse sidebar',
    'source.ready': 'Sources ready',
    'source.missing': 'Sources missing',
    'source.configure': 'Configure local JSONL sources in Settings',
    'source.count': '{count} local sources',
    'source.label': 'Local sources',
    'index.update': 'Update Index',
    'index.rebuild': 'Rebuild Index',
    'index.updateHint': 'Scan enabled sources and parse only new or changed JSONL files.',
    'index.rebuildHint': 'Clear indexed files for enabled sources, then parse every JSONL file again.',
    'index.result': '{indexed} indexed, {skipped} skipped, {failed} failed',
    'index.failed': 'Index failed',
    'index.failedWithMessage': 'Index failed: {message}',
    'language.label': 'Language',
    'language.aria': 'Select language',
    'language.english': 'English',
    'language.chinese': 'Chinese'
  },
  'zh-CN': {
    'brand.subtitle': '本地 Agent 用量',
    'nav.section': '查看',
    'nav.overview': '概览',
    'nav.sessions': '会话',
    'nav.tools': '工具',
    'nav.audit': '审计',
    'nav.settings': '设置',
    'sidebar.expand': '展开侧边栏',
    'sidebar.collapse': '收起侧边栏',
    'source.ready': '来源已就绪',
    'source.missing': '缺少来源',
    'source.configure': '在设置中配置本地 JSONL 来源',
    'source.count': '{count} 个本地来源',
    'source.label': '本地来源',
    'index.update': '更新索引',
    'index.rebuild': '重建索引',
    'index.updateHint': '扫描已启用来源，并只解析新增或变更的 JSONL 文件。',
    'index.rebuildHint': '清除已启用来源的索引记录，然后重新解析所有 JSONL 文件。',
    'index.result': '已索引 {indexed}，跳过 {skipped}，失败 {failed}',
    'index.failed': '索引失败',
    'index.failedWithMessage': '索引失败：{message}',
    'language.label': '语言',
    'language.aria': '选择语言',
    'language.english': '英文',
    'language.chinese': '中文'
  }
})
const settings = ref<Settings | null>(null)
const indexing = ref(false)
const refreshKey = ref(0)
const sidebarCollapsed = ref(false)

const hasSource = computed(() => Boolean(settings.value?.sourcePaths?.length || settings.value?.sourcePath))
const sourceStatusLabel = computed(() => (hasSource.value ? t('source.ready') : t('source.missing')))
const sourcePathDisplay = computed(() => settings.value?.sourcePath || t('source.configure'))
const sourceSummary = computed(() => {
  const count = settings.value?.sourcePaths?.length || 0
  if (count > 1) return t('source.count', { count })
  return sourcePathDisplay.value
})
const sidebarToggleLabel = computed(() => (sidebarCollapsed.value ? t('sidebar.expand') : t('sidebar.collapse')))
const updateIndexHint = computed(() => t('index.updateHint'))
const rebuildIndexHint = computed(() => t('index.rebuildHint'))

const selectedKeys = computed(() => {
  if (route.path.startsWith('/sessions')) return ['sessions']
  if (route.path.startsWith('/tools')) return ['tools']
  if (route.path.startsWith('/audit')) return ['audit']
  if (route.path.startsWith('/settings')) return ['settings']
  return ['overview']
})

const menuItems = computed(() => [
  { key: 'overview', icon: BarChartOutlined, label: t('nav.overview'), path: '/overview' },
  { key: 'sessions', icon: HistoryOutlined, label: t('nav.sessions'), path: '/sessions' },
  { key: 'tools', icon: ToolOutlined, label: t('nav.tools'), path: '/tools' },
  { key: 'audit', icon: WarningOutlined, label: t('nav.audit'), path: '/audit' },
  { key: 'settings', icon: SettingOutlined, label: t('nav.settings'), path: '/settings' }
])
const languageOptions = computed(() =>
  localeOptions.map((option) => ({
    value: option.value,
    label: option.value === 'en' ? t('language.english') : t('language.chinese')
  }))
)

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
    message.success(t('index.result', { indexed: result.indexed, skipped: result.skipped, failed: result.failed }))
    await loadSettings()
    refreshKey.value += 1
  } catch (error) {
    message.error(t('index.failedWithMessage', { message: error instanceof Error ? error.message : t('index.failed') }))
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

function handleLocaleChange(event: Event) {
  setLocale((event.target as HTMLSelectElement).value)
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
    :locale="antDesignLocale"
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
            <div class="brand-subtitle">{{ t('brand.subtitle') }}</div>
          </div>
        </div>
        <div class="nav-section-head">
          <div class="nav-section-label">{{ t('nav.section') }}</div>
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
            <span class="source-label">{{ t('source.label') }}</span>
            <a-typography-text class="source-path" :ellipsis="{ tooltip: sourcePathDisplay }">
              {{ sourceSummary }}
            </a-typography-text>
          </div>
          <div class="header-actions">
            <div class="language-switcher">
              <span class="language-switcher-label">{{ t('language.label') }}</span>
              <select
                class="language-select"
                :value="currentLocale"
                :aria-label="t('language.aria')"
                @change="handleLocaleChange"
              >
                <option v-for="option in languageOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </div>
            <a-tooltip :title="updateIndexHint" placement="bottom">
              <a-button type="primary" :loading="indexing" @click="indexNow(false)">
                <template #icon>
                  <PlayCircleOutlined />
                </template>
                {{ t('index.update') }}
              </a-button>
            </a-tooltip>
            <a-tooltip :title="rebuildIndexHint" placement="bottom">
              <a-button :loading="indexing" @click="indexNow(true)">
                <template #icon>
                  <ReloadOutlined />
                </template>
                {{ t('index.rebuild') }}
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
