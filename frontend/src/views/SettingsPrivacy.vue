<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { CheckOutlined, ReloadOutlined, SafetyCertificateOutlined } from '@ant-design/icons-vue'
import { api, formatNumber, type PrivacyConfigApplyResult, type PrivacyConfigSetting, type PrivacyConfigStatus } from '../api'
import { useMessages } from '../i18n'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const { t } = useMessages({
  en: {
    'privacy.title': 'Privacy',
    'privacy.kicker': 'Codex user-level config.toml',
    'privacy.boundary.title': 'Current support: Codex config.toml controls only',
    'privacy.boundary.description':
      'This page reads and applies user-level Codex config.toml privacy settings. It does not scan logs, scan secrets, or infer broad filesystem policy.',
    'privacy.action.refresh': 'Refresh',
    'privacy.action.applyStrict': 'Apply Strict Config',
    'privacy.action.apply': 'Apply',
    'privacy.meta.target': 'Target',
    'privacy.meta.configPath': 'Config path',
    'privacy.meta.file': 'File',
    'privacy.meta.exists': 'Exists',
    'privacy.meta.missing': 'Missing',
    'privacy.meta.score': 'Privacy score',
    'privacy.meta.total': 'Total',
    'privacy.meta.configured': 'Configured',
    'privacy.meta.defaultSafe': 'Default-safe',
    'privacy.meta.needsChange': 'Needs change',
    'privacy.meta.backupPath': 'Backup path',
    'privacy.status.ready': 'Configured',
    'privacy.status.needsChange': 'Needs change',
    'privacy.status.noStatus': 'No status',
    'privacy.status.hardened': 'configured',
    'privacy.status.implicit': 'default-safe',
    'privacy.status.attention': 'needs change',
    'privacy.status.unknown': 'unknown',
    'privacy.table.title': 'Config controls',
    'privacy.table.kicker': 'Desired strict values from backend policy',
    'privacy.column.setting': 'Setting',
    'privacy.column.key': 'Key',
    'privacy.column.current': 'Current',
    'privacy.column.desired': 'Desired',
    'privacy.column.status': 'Status',
    'privacy.column.impact': 'Impact',
    'privacy.column.action': 'Action',
    'privacy.empty': 'No privacy controls returned',
    'privacy.value.unset': 'unset',
    'privacy.message.loadFailed': 'Load privacy settings failed',
    'privacy.message.applyFailed': 'Apply privacy settings failed',
    'privacy.message.applied': 'Applied {count} changes',
    'privacy.message.noApply': 'No applicable settings',
    'privacy.result.title': 'Last apply result',
    'privacy.result.changed': 'Changed',
    'privacy.result.noChanges': 'No config values changed',
    'privacy.warning.title': 'Warnings'
  },
  'zh-CN': {
    'privacy.title': '隐私',
    'privacy.kicker': 'Codex 用户级 config.toml',
    'privacy.boundary.title': '当前支持范围：仅 Codex config.toml 控制项',
    'privacy.boundary.description':
      '此页面只读取并应用用户级 Codex config.toml 隐私设置，不扫描日志、不扫描密钥，也不推断广义文件系统策略。',
    'privacy.action.refresh': '刷新',
    'privacy.action.applyStrict': '应用严格配置',
    'privacy.action.apply': '应用',
    'privacy.meta.target': '目标',
    'privacy.meta.configPath': '配置路径',
    'privacy.meta.file': '文件',
    'privacy.meta.exists': '存在',
    'privacy.meta.missing': '缺失',
    'privacy.meta.score': '隐私分数',
    'privacy.meta.total': '总数',
    'privacy.meta.configured': '已配置',
    'privacy.meta.defaultSafe': '默认安全',
    'privacy.meta.needsChange': '需要更改',
    'privacy.meta.backupPath': '备份路径',
    'privacy.status.ready': '已配置',
    'privacy.status.needsChange': '需要更改',
    'privacy.status.noStatus': '无状态',
    'privacy.status.hardened': '已配置',
    'privacy.status.implicit': '默认安全',
    'privacy.status.attention': '需要更改',
    'privacy.status.unknown': '未知',
    'privacy.table.title': '配置控制项',
    'privacy.table.kicker': '后端策略提供的严格目标值',
    'privacy.column.setting': '设置',
    'privacy.column.key': '键',
    'privacy.column.current': '当前',
    'privacy.column.desired': '目标',
    'privacy.column.status': '状态',
    'privacy.column.impact': '影响',
    'privacy.column.action': '操作',
    'privacy.empty': '未返回隐私控制项',
    'privacy.value.unset': '未设置',
    'privacy.message.loadFailed': '加载隐私设置失败',
    'privacy.message.applyFailed': '应用隐私设置失败',
    'privacy.message.applied': '已应用 {count} 项变更',
    'privacy.message.noApply': '没有可应用的设置',
    'privacy.result.title': '上次应用结果',
    'privacy.result.changed': '已变更',
    'privacy.result.noChanges': '没有配置值变更',
    'privacy.warning.title': '警告'
  }
})

const loading = ref(true)
const applyingAll = ref(false)
const applyingId = ref('')
const privacyStatus = ref<PrivacyConfigStatus | null>(null)
const lastApply = ref<PrivacyConfigApplyResult | null>(null)

const summary = computed(
  () =>
    privacyStatus.value?.summary || {
      score: 0,
      total: 0,
      hardened: 0,
      attention: 0,
      implicit: 0
    }
)
const settings = computed(() => privacyStatus.value?.settings || [])
const strictSettingIds = computed(() => settings.value.filter((setting) => setting.canApply).map((setting) => setting.id))
const scoreLabel = computed(() => `${formatNumber(summary.value.score)}%`)
const tableLocale = computed(() => ({ emptyText: t('privacy.empty') }))
const statusState = computed(() => {
  if (!privacyStatus.value) return { color: 'default', label: t('privacy.status.noStatus') }
  if (summary.value.attention > 0) return { color: 'warning', label: t('privacy.status.needsChange') }
  return { color: 'success', label: t('privacy.status.ready') }
})
const warningList = computed(() => {
  const values = [...(privacyStatus.value?.warnings || []), ...(lastApply.value?.warnings || [])]
  return [...new Set(values.filter(Boolean))]
})

const privacyColumns = computed(() => [
  { title: t('privacy.column.setting'), dataIndex: 'title', key: 'setting', width: 330 },
  { title: t('privacy.column.key'), dataIndex: 'key', key: 'key', width: 220 },
  { title: t('privacy.column.current'), dataIndex: 'currentValue', key: 'current', width: 140 },
  { title: t('privacy.column.desired'), dataIndex: 'desiredValue', key: 'desired', width: 140 },
  { title: t('privacy.column.status'), dataIndex: 'status', key: 'status', width: 130 },
  { title: t('privacy.column.impact'), dataIndex: 'impact', key: 'impact', width: 260 },
  { title: t('privacy.column.action'), key: 'action', width: 112, align: 'right' }
])

function statusLabel(status: string) {
  if (status === 'hardened') return t('privacy.status.hardened')
  if (status === 'implicit') return t('privacy.status.implicit')
  if (status === 'attention') return t('privacy.status.attention')
  return status || t('privacy.status.unknown')
}

function statusColor(status: string) {
  if (status === 'hardened') return 'success'
  if (status === 'implicit') return 'default'
  if (status === 'attention') return 'warning'
  return 'default'
}

function formatConfigValue(value: unknown) {
  if (value === undefined || value === null || value === '') return t('privacy.value.unset')
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)

  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function settingRowClass(record: PrivacyConfigSetting) {
  return record.status === 'attention' ? 'privacy-row-attention' : ''
}

async function load() {
  loading.value = true
  try {
    privacyStatus.value = await api.getCodexPrivacy()
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('privacy.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function applySettings(settingIds: string[], strictApply = false) {
  if (!settingIds.length) {
    message.info(t('privacy.message.noApply'))
    return
  }

  if (strictApply) applyingAll.value = true
  else applyingId.value = settingIds[0]

  try {
    const result = await api.applyCodexPrivacy(settingIds)
    privacyStatus.value = result.status
    lastApply.value = result
    message.success(t('privacy.message.applied', { count: formatNumber(result.changed?.length || 0) }))
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('privacy.message.applyFailed'))
  } finally {
    applyingAll.value = false
    applyingId.value = ''
  }
}

onMounted(load)
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="panel settings-tool-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('privacy.title') }}</h2>
            <div class="panel-kicker">{{ t('privacy.kicker') }}</div>
          </div>
          <div class="summary-actions">
            <a-tag :color="statusState.color" class="status-tag">{{ statusState.label }}</a-tag>
            <a-button @click="load">
              <template #icon>
                <ReloadOutlined />
              </template>
              {{ t('privacy.action.refresh') }}
            </a-button>
          </div>
        </div>
        <div class="panel-body">
          <div class="section-stack">
            <a-alert
              class="privacy-boundary"
              type="info"
              show-icon
              :message="t('privacy.boundary.title')"
              :description="t('privacy.boundary.description')"
            />

            <div class="metadata-grid">
              <div class="metadata-item">
                <div class="metadata-label">{{ t('privacy.meta.target') }}</div>
                <div class="metadata-value">
                  {{ privacyStatus?.name || 'Codex' }}
                  <span v-if="privacyStatus?.target" class="muted">({{ privacyStatus.target }})</span>
                </div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('privacy.meta.file') }}</div>
                <div class="metadata-value">
                  <a-tag :color="privacyStatus?.exists ? 'success' : 'warning'" class="status-tag">
                    {{ privacyStatus?.exists ? t('privacy.meta.exists') : t('privacy.meta.missing') }}
                  </a-tag>
                </div>
              </div>
              <div class="metadata-item is-wide">
                <div class="metadata-label">{{ t('privacy.meta.configPath') }}</div>
                <div class="metadata-value">
                  <a-typography-text :ellipsis="{ tooltip: privacyStatus?.configPath }">
                    {{ privacyStatus?.configPath || '-' }}
                  </a-typography-text>
                </div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('privacy.meta.score') }}</div>
                <div class="metadata-value privacy-score">
                  <strong :class="summary.attention ? 'status-warning' : 'status-ok'">{{ scoreLabel }}</strong>
                </div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('privacy.meta.total') }}</div>
                <div class="metadata-value number-cell">{{ formatNumber(summary.total) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('privacy.meta.configured') }}</div>
                <div class="metadata-value number-cell status-ok">{{ formatNumber(summary.hardened) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('privacy.meta.defaultSafe') }}</div>
                <div class="metadata-value number-cell">{{ formatNumber(summary.implicit) }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('privacy.meta.needsChange') }}</div>
                <div class="metadata-value number-cell" :class="summary.attention ? 'status-warning' : 'status-ok'">
                  {{ formatNumber(summary.attention) }}
                </div>
              </div>
              <div v-if="lastApply" class="metadata-item is-wide">
                <div class="metadata-label">{{ t('privacy.meta.backupPath') }}</div>
                <div class="metadata-value">
                  <a-typography-text :ellipsis="{ tooltip: lastApply.backupPath }">
                    {{ lastApply.backupPath || '-' }}
                  </a-typography-text>
                </div>
              </div>
            </div>

            <div class="toolbar">
              <div class="toolbar-left">
                <a-button
                  type="primary"
                  :loading="applyingAll"
                  :disabled="!strictSettingIds.length || Boolean(applyingId)"
                  @click="applySettings(strictSettingIds, true)"
                >
                  <template #icon>
                    <SafetyCertificateOutlined />
                  </template>
                  {{ t('privacy.action.applyStrict') }}
                </a-button>
              </div>
            </div>

            <div v-if="lastApply" class="index-result-block">
              <div class="index-result-header">
                <div>
                  <div class="index-result-title">{{ t('privacy.result.title') }}</div>
                  <div class="muted">
                    {{ lastApply.changed?.length ? t('privacy.result.changed') : t('privacy.result.noChanges') }}
                  </div>
                </div>
                <a-tag color="success" class="status-tag">
                  {{ formatNumber(lastApply.changed?.length || 0) }} {{ t('privacy.result.changed') }}
                </a-tag>
              </div>
              <div v-if="lastApply.changed?.length" class="privacy-change-list">
                <div v-for="change in lastApply.changed.slice(0, 6)" :key="change.id" class="privacy-change-row">
                  <span class="mono">{{ change.key }}</span>
                  <span>
                    {{ formatConfigValue(change.before) }} -> {{ formatConfigValue(change.after) }}
                  </span>
                </div>
              </div>
            </div>

            <div v-if="warningList.length" class="index-result-warnings">
              <div class="metadata-label">{{ t('privacy.warning.title') }}</div>
              <ul>
                <li v-for="warning in warningList" :key="warning">{{ warning }}</li>
              </ul>
            </div>
          </div>
        </div>
      </section>

      <section class="panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('privacy.table.title') }}</h2>
            <div class="panel-kicker">{{ t('privacy.table.kicker') }}</div>
          </div>
          <span class="row-count">{{ formatNumber(settings.length) }} {{ t('privacy.meta.total') }}</span>
        </div>
        <a-table
          class="dense-table privacy-settings-table"
          size="small"
          :columns="privacyColumns"
          :data-source="settings"
          row-key="id"
          :pagination="false"
          :locale="tableLocale"
          :scroll="{ x: 1260 }"
          :row-class-name="settingRowClass"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'setting'">
              <div class="privacy-setting-cell">
                <div class="privacy-setting-head">
                  <span class="privacy-setting-title">{{ record.title }}</span>
                  <a-tag v-if="record.group" class="status-tag">{{ record.group }}</a-tag>
                </div>
                <div class="privacy-setting-description">{{ record.description }}</div>
              </div>
            </template>
            <template v-else-if="column.key === 'key'">
              <a-typography-text class="mono privacy-value" :ellipsis="{ tooltip: record.key }">
                {{ record.key }}
              </a-typography-text>
            </template>
            <template v-else-if="column.key === 'current'">
              <a-typography-text class="mono privacy-value" :ellipsis="{ tooltip: formatConfigValue(record.currentValue) }">
                {{ formatConfigValue(record.currentValue) }}
              </a-typography-text>
            </template>
            <template v-else-if="column.key === 'desired'">
              <a-typography-text class="mono privacy-value" :ellipsis="{ tooltip: formatConfigValue(record.desiredValue) }">
                {{ formatConfigValue(record.desiredValue) }}
              </a-typography-text>
            </template>
            <template v-else-if="column.key === 'status'">
              <a-tag :color="statusColor(record.status)" class="status-tag">
                {{ statusLabel(record.status) }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'impact'">
              <span class="privacy-impact">{{ record.impact || '-' }}</span>
            </template>
            <template v-else-if="column.key === 'action'">
              <a-button
                size="small"
                :disabled="!record.canApply || applyingAll || (Boolean(applyingId) && applyingId !== record.id)"
                :loading="applyingId === record.id"
                @click="applySettings([record.id])"
              >
                <template #icon>
                  <CheckOutlined />
                </template>
                {{ t('privacy.action.apply') }}
              </a-button>
            </template>
          </template>
        </a-table>
      </section>
    </div>
  </a-spin>
</template>
