<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import message from 'ant-design-vue/es/message'
import ASegmented from 'ant-design-vue/es/segmented'
import ASelect from 'ant-design-vue/es/select'
import ASpin from 'ant-design-vue/es/spin'
import ASwitch from 'ant-design-vue/es/switch'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import {
  CheckOutlined,
  DeleteOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SaveOutlined,
  UndoOutlined
} from '@ant-design/icons-vue'
import {
  api,
  formatNumber,
  type PrivacyConfigApplyResult,
  type PrivacyConfigChange,
  type PrivacyConfigSetting,
  type PrivacyConfigStatus,
  type PrivacyConfigValueType,
  type PrivacyTarget
} from '../api'
import { useMessages } from '../i18n'

const ATypographyText = Typography.Text
const { t } = useMessages({
  en: {
    'privacy.title': 'Agent Privacy',
    'privacy.kicker': 'External agent config: {target} user-level {file}',
    'privacy.boundary.title': 'Current support: user-level agent config controls',
    'privacy.boundary.description':
      'This page reads and edits supported user-level privacy settings for Codex and Gemini CLI. It does not scan logs, scan secrets, or infer broad filesystem policy.',
    'privacy.target.codex': 'Codex',
    'privacy.target.gemini': 'Gemini CLI',
    'privacy.action.refresh': 'Refresh',
    'privacy.action.saveAll': 'Save changes',
    'privacy.action.useStrict': 'Use strict',
    'privacy.action.unset': 'Unset',
    'privacy.action.reset': 'Reset',
    'privacy.action.save': 'Save',
    'privacy.meta.target': 'Target',
    'privacy.meta.configPath': 'Config path',
    'privacy.meta.file': 'File',
    'privacy.meta.exists': 'Exists',
    'privacy.meta.missing': 'Missing',
    'privacy.meta.total': 'Total',
    'privacy.meta.strictConfigured': 'Strict',
    'privacy.meta.defaultSafe': 'Default-safe',
    'privacy.meta.customConfigured': 'Custom configured',
    'privacy.meta.missingRequired': 'Unset',
    'privacy.meta.unsavedChanges': 'Unsaved',
    'privacy.meta.backupPath': 'Backup path',
    'privacy.status.ready': 'No review needed',
    'privacy.status.needsChange': 'Needs review',
    'privacy.status.noStatus': 'No status',
    'privacy.status.hardened': 'strict',
    'privacy.status.implicit': 'default-safe',
    'privacy.status.attention': 'needs review',
    'privacy.status.unknown': 'unknown',
    'privacy.settings.title': 'Configuration groups',
    'privacy.settings.kicker': 'Edit current values directly, then save individual rows or all changes',
    'privacy.empty': 'No privacy controls returned',
    'privacy.value.unset': 'unset',
    'privacy.value.notConfigured': 'not configured',
    'privacy.value.pendingUnset': 'will unset on save',
    'privacy.value.type': 'Type',
    'privacy.value.current': 'Current',
    'privacy.value.strict': 'Strict',
    'privacy.value.unsaved': 'Unsaved',
    'privacy.message.loadFailed': 'Load privacy settings failed',
    'privacy.message.saveFailed': 'Save privacy settings failed',
    'privacy.message.saved': 'Saved {count} changes',
    'privacy.message.noChanges': 'No unsaved changes',
    'privacy.result.title': 'Last save result',
    'privacy.result.changed': 'Changed',
    'privacy.result.noChanges': 'No config values changed',
    'privacy.warning.title': 'Warnings',
    'privacy.group.default': 'General'
  },
  'zh-CN': {
    'privacy.title': 'Agent 隐私',
    'privacy.kicker': '外部 Agent 配置：{target} 用户级 {file}',
    'privacy.boundary.title': '当前支持范围：用户级 Agent 配置控制项',
    'privacy.boundary.description':
      '此页面只读取并编辑 Codex 与 Gemini CLI 已支持的用户级隐私设置，不扫描日志、不扫描密钥，也不推断广义文件系统策略。',
    'privacy.target.codex': 'Codex',
    'privacy.target.gemini': 'Gemini CLI',
    'privacy.action.refresh': '刷新',
    'privacy.action.saveAll': '保存变更',
    'privacy.action.useStrict': '使用严格值',
    'privacy.action.unset': '取消设置',
    'privacy.action.reset': '重置',
    'privacy.action.save': '保存',
    'privacy.meta.target': '目标',
    'privacy.meta.configPath': '配置路径',
    'privacy.meta.file': '文件',
    'privacy.meta.exists': '存在',
    'privacy.meta.missing': '缺失',
    'privacy.meta.total': '总数',
    'privacy.meta.strictConfigured': '严格值',
    'privacy.meta.defaultSafe': '默认安全',
    'privacy.meta.customConfigured': '显式自定义',
    'privacy.meta.missingRequired': '未设置',
    'privacy.meta.unsavedChanges': '未保存',
    'privacy.meta.backupPath': '备份路径',
    'privacy.status.ready': '无需检查',
    'privacy.status.needsChange': '需要检查',
    'privacy.status.noStatus': '无状态',
    'privacy.status.hardened': '严格值',
    'privacy.status.implicit': '默认安全',
    'privacy.status.attention': '需要检查',
    'privacy.status.unknown': '未知',
    'privacy.settings.title': '配置分组',
    'privacy.settings.kicker': '直接编辑当前值，再保存单行或所有变更',
    'privacy.empty': '未返回隐私控制项',
    'privacy.value.unset': '未设置',
    'privacy.value.notConfigured': '未配置',
    'privacy.value.pendingUnset': '保存后取消设置',
    'privacy.value.type': '类型',
    'privacy.value.current': '当前',
    'privacy.value.strict': '严格值',
    'privacy.value.unsaved': '未保存',
    'privacy.message.loadFailed': '加载隐私设置失败',
    'privacy.message.saveFailed': '保存隐私设置失败',
    'privacy.message.saved': '已保存 {count} 项变更',
    'privacy.message.noChanges': '没有未保存变更',
    'privacy.result.title': '上次保存结果',
    'privacy.result.changed': '已变更',
    'privacy.result.noChanges': '没有配置值变更',
    'privacy.warning.title': '警告',
    'privacy.group.default': '通用'
  }
})

type EditOp = 'set' | 'unset'

interface SettingEdit {
  id: string
  op: EditOp
  valueType: PrivacyConfigValueType
  boolValue: boolean
  stringValue: string
  arrayValue: string[]
}

const loading = ref(true)
const savingAll = ref(false)
const savingId = ref('')
const selectedTarget = ref<PrivacyTarget>('codex')
const privacyStatus = ref<PrivacyConfigStatus | null>(null)
const lastApply = ref<PrivacyConfigApplyResult | null>(null)
const edits = ref<Record<string, SettingEdit>>({})
let loadRequestId = 0

const targetOptions = computed<{ label: string; value: PrivacyTarget }[]>(() => [
  { label: t('privacy.target.codex'), value: 'codex' },
  { label: t('privacy.target.gemini'), value: 'gemini' }
])
const targetLabel = computed(() => {
  if (privacyStatus.value?.name) return privacyStatus.value.name
  return selectedTarget.value === 'gemini' ? t('privacy.target.gemini') : t('privacy.target.codex')
})
const targetFile = computed(() => (selectedTarget.value === 'gemini' ? 'settings.json' : 'config.toml'))
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
const changedSettings = computed(() => settings.value.filter((setting) => canEdit(setting) && isEditChanged(setting)))
const kickerText = computed(() => t('privacy.kicker', { target: targetLabel.value, file: targetFile.value }))
const statusState = computed(() => {
  if (!privacyStatus.value) return { color: 'default', label: t('privacy.status.noStatus') }
  if (metricCounts.value.missingRequired > 0) {
    return { color: 'warning', label: t('privacy.status.needsChange') }
  }
  return { color: 'success', label: t('privacy.status.ready') }
})
const warningList = computed(() => {
  const values = [...(privacyStatus.value?.warnings || []), ...(lastApply.value?.warnings || [])]
  return [...new Set(values.filter(Boolean))]
})
const metricCounts = computed(() => {
  const total = settings.value.length || summary.value.total
  const strictConfigured = settings.value.filter(
    (setting) => setting.configured && valuesEqual(setting.currentValue, strictValue(setting), valueType(setting))
  ).length
  const defaultSafe = settings.value.filter((setting) => !setting.configured && setting.status === 'implicit').length
  const customConfigured = settings.value.filter(
    (setting) => setting.configured && !valuesEqual(setting.currentValue, strictValue(setting), valueType(setting))
  ).length
  const missingRequired = settings.value.filter((setting) => !setting.configured && setting.status === 'attention').length
  const unsavedChanges = changedSettings.value.length
  return { total, strictConfigured, defaultSafe, customConfigured, missingRequired, unsavedChanges }
})
const groupedSettings = computed(() => {
  const groups = new Map<string, PrivacyConfigSetting[]>()
  for (const setting of settings.value) {
    const group = setting.group || t('privacy.group.default')
    groups.set(group, [...(groups.get(group) || []), setting])
  }
  return [...groups.entries()].map(([name, items]) => ({ name, items }))
})

function valueType(setting: PrivacyConfigSetting): PrivacyConfigValueType {
  if (setting.valueType) return setting.valueType
  const sample = strictValue(setting) ?? setting.currentValue
  if (typeof sample === 'boolean') return 'bool'
  if (Array.isArray(sample)) return 'stringArray'
  return 'string'
}

function strictValue(setting: PrivacyConfigSetting) {
  return setting.strictValue !== undefined ? setting.strictValue : setting.desiredValue
}

function normalizeValue(value: unknown, type: PrivacyConfigValueType): unknown {
  if (type === 'bool') {
    if (typeof value === 'string') return value.toLowerCase() === 'true'
    return value === true
  }
  if (type === 'stringArray') {
    if (!Array.isArray(value)) return value === undefined || value === null || value === '' ? [] : [String(value)]
    return value.map((item) => String(item))
  }
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : formatConfigValue(value)
}

function valuesEqual(left: unknown, right: unknown, type: PrivacyConfigValueType) {
  return JSON.stringify(normalizeValue(left, type)) === JSON.stringify(normalizeValue(right, type))
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

function createEdit(setting: PrivacyConfigSetting): SettingEdit {
  const type = valueType(setting)
  const baseValue = setting.configured ? setting.currentValue : undefined
  const normalized = normalizeValue(baseValue, type)
  return {
    id: setting.id,
    op: setting.configured ? 'set' : 'unset',
    valueType: type,
    boolValue: normalized === true,
    stringValue: typeof normalized === 'string' ? normalized : formatConfigValue(normalized),
    arrayValue: Array.isArray(normalized) ? normalized : []
  }
}

function syncEdits(status: PrivacyConfigStatus | null) {
  const next: Record<string, SettingEdit> = {}
  for (const setting of status?.settings || []) {
    next[setting.id] = createEdit(setting)
  }
  edits.value = next
}

function editFor(setting: PrivacyConfigSetting) {
  if (!edits.value[setting.id]) edits.value[setting.id] = createEdit(setting)
  return edits.value[setting.id]
}

function editValue(setting: PrivacyConfigSetting) {
  const edit = editFor(setting)
  if (edit.valueType === 'bool') return edit.boolValue
  if (edit.valueType === 'stringArray') return edit.arrayValue
  return edit.stringValue
}

function markEditSet(id: string) {
  if (edits.value[id]) edits.value[id].op = 'set'
}

function useStrict(setting: PrivacyConfigSetting) {
  const edit = editFor(setting)
  const type = valueType(setting)
  const normalized = normalizeValue(strictValue(setting), type)
  edit.op = 'set'
  edit.valueType = type
  edit.boolValue = normalized === true
  edit.stringValue = typeof normalized === 'string' ? normalized : formatConfigValue(normalized)
  edit.arrayValue = Array.isArray(normalized) ? normalized : []
}

function unsetEdit(setting: PrivacyConfigSetting) {
  editFor(setting).op = 'unset'
}

function resetEdit(setting: PrivacyConfigSetting) {
  edits.value[setting.id] = createEdit(setting)
}

function baselineOp(setting: PrivacyConfigSetting): EditOp {
  return setting.configured ? 'set' : 'unset'
}

function isEditChanged(setting: PrivacyConfigSetting) {
  const edit = editFor(setting)
  if (edit.op !== baselineOp(setting)) return true
  if (edit.op === 'unset') return false
  return !valuesEqual(editValue(setting), setting.currentValue, valueType(setting))
}

function changeForSetting(setting: PrivacyConfigSetting): PrivacyConfigChange {
  const edit = editFor(setting)
  if (edit.op === 'unset') return { id: setting.id, op: 'unset' }
  return { id: setting.id, op: 'set', value: editValue(setting) }
}

function settingState(setting: PrivacyConfigSetting) {
  if (setting.configured && valuesEqual(setting.currentValue, strictValue(setting), valueType(setting))) {
    return { color: 'success', label: t('privacy.status.hardened') }
  }
  if (setting.configured) return { color: 'processing', label: t('privacy.meta.customConfigured') }
  if (setting.status === 'implicit') return { color: 'default', label: t('privacy.status.implicit') }
  return { color: 'warning', label: t('privacy.value.notConfigured') }
}

function settingCardClass(setting: PrivacyConfigSetting) {
  return {
    'privacy-setting-card': true,
    'is-attention': !setting.configured && setting.status === 'attention',
    'is-changed': isEditChanged(setting)
  }
}

function canEdit(setting: PrivacyConfigSetting) {
  return setting.canApply !== false
}

function valueTypeLabel(type: PrivacyConfigValueType) {
  if (type === 'bool') return 'bool'
  if (type === 'stringArray') return 'string[]'
  return 'string'
}

async function load() {
  const requestId = ++loadRequestId
  loading.value = true
  try {
    const status = await api.getAgentPrivacy(selectedTarget.value)
    if (requestId !== loadRequestId) return
    privacyStatus.value = status
    syncEdits(status)
  } catch (error) {
    if (requestId !== loadRequestId) return
    message.error(error instanceof Error ? error.message : t('privacy.message.loadFailed'))
  } finally {
    if (requestId === loadRequestId) loading.value = false
  }
}

async function saveSettings(records: PrivacyConfigSetting[], saveAll = false) {
  const changes = records.filter((setting) => canEdit(setting) && isEditChanged(setting)).map(changeForSetting)
  if (!changes.length) {
    message.info(t('privacy.message.noChanges'))
    return
  }

  if (saveAll) savingAll.value = true
  else savingId.value = changes[0].id

  try {
    const result = await api.applyAgentPrivacyChanges(selectedTarget.value, changes)
    privacyStatus.value = result.status
    lastApply.value = result
    syncEdits(result.status)
    if (result.changed?.length) {
      message.success(t('privacy.message.saved', { count: formatNumber(result.changed.length) }))
    } else {
      message.info(t('privacy.message.noChanges'))
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('privacy.message.saveFailed'))
  } finally {
    savingAll.value = false
    savingId.value = ''
  }
}

onMounted(load)
watch(selectedTarget, () => {
  lastApply.value = null
  load()
})
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <section class="panel agent-privacy-tool-panel">
        <div class="panel-header">
          <div>
            <h2 class="panel-title">{{ t('privacy.title') }}</h2>
            <div class="panel-kicker">{{ kickerText }}</div>
          </div>
          <div class="summary-actions">
            <a-segmented v-model:value="selectedTarget" :options="targetOptions" />
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

            <div class="metadata-grid privacy-config-meta">
              <div class="metadata-item">
                <div class="metadata-label">{{ t('privacy.meta.target') }}</div>
                <div class="metadata-value">
                  {{ privacyStatus?.name || targetLabel }}
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
                  <a-typography-text class="mono privacy-copy-block" :copyable="{ text: privacyStatus?.configPath || '' }">
                    {{ privacyStatus?.configPath || '-' }}
                  </a-typography-text>
                </div>
              </div>
              <div v-if="lastApply" class="metadata-item is-wide">
                <div class="metadata-label">{{ t('privacy.meta.backupPath') }}</div>
                <div class="metadata-value">
                  <a-typography-text class="mono privacy-copy-block" :copyable="{ text: lastApply.backupPath || '' }">
                    {{ lastApply.backupPath || '-' }}
                  </a-typography-text>
                </div>
              </div>
            </div>

            <div class="privacy-count-grid">
              <div class="privacy-count-item">
                <div class="metadata-label">{{ t('privacy.meta.total') }}</div>
                <strong>{{ formatNumber(metricCounts.total) }}</strong>
              </div>
              <div class="privacy-count-item is-success">
                <div class="metadata-label">{{ t('privacy.meta.strictConfigured') }}</div>
                <strong>{{ formatNumber(metricCounts.strictConfigured) }}</strong>
              </div>
              <div class="privacy-count-item">
                <div class="metadata-label">{{ t('privacy.meta.defaultSafe') }}</div>
                <strong>{{ formatNumber(metricCounts.defaultSafe) }}</strong>
              </div>
              <div class="privacy-count-item">
                <div class="metadata-label">{{ t('privacy.meta.customConfigured') }}</div>
                <strong>{{ formatNumber(metricCounts.customConfigured) }}</strong>
              </div>
              <div class="privacy-count-item" :class="{ 'is-warning': metricCounts.missingRequired > 0 }">
                <div class="metadata-label">{{ t('privacy.meta.missingRequired') }}</div>
                <strong>{{ formatNumber(metricCounts.missingRequired) }}</strong>
              </div>
              <div class="privacy-count-item" :class="{ 'is-warning': metricCounts.unsavedChanges > 0 }">
                <div class="metadata-label">{{ t('privacy.meta.unsavedChanges') }}</div>
                <strong>{{ formatNumber(metricCounts.unsavedChanges) }}</strong>
              </div>
            </div>

            <div class="toolbar privacy-edit-toolbar">
              <div class="toolbar-left">
                <a-button
                  type="primary"
                  :loading="savingAll"
                  :disabled="!changedSettings.length || Boolean(savingId)"
                  @click="saveSettings(changedSettings, true)"
                >
                  <template #icon>
                    <SaveOutlined />
                  </template>
                  {{ t('privacy.action.saveAll') }}
                  <span v-if="changedSettings.length">({{ formatNumber(changedSettings.length) }})</span>
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
                  <span>{{ formatConfigValue(change.before) }} -> {{ formatConfigValue(change.after) }}</span>
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
            <h2 class="panel-title">{{ t('privacy.settings.title') }}</h2>
            <div class="panel-kicker">{{ t('privacy.settings.kicker') }}</div>
          </div>
          <span class="row-count">{{ formatNumber(settings.length) }} {{ t('privacy.meta.total') }}</span>
        </div>
        <div class="panel-body">
          <div v-if="!settings.length" class="empty-state empty-state-compact">
            <div>
              <div class="empty-state-title">{{ t('privacy.empty') }}</div>
            </div>
          </div>
          <div v-else class="privacy-group-list">
            <section v-for="group in groupedSettings" :key="group.name" class="privacy-group-section">
              <div class="privacy-group-header">
                <h3>{{ group.name }}</h3>
                <span class="row-count">{{ formatNumber(group.items.length) }}</span>
              </div>
              <div class="privacy-setting-list">
                <article v-for="setting in group.items" :key="setting.id" :class="settingCardClass(setting)">
                  <div class="privacy-setting-main">
                    <div class="privacy-setting-heading">
                      <div>
                        <div class="privacy-setting-title">{{ setting.title }}</div>
                        <div class="privacy-setting-description">{{ setting.description }}</div>
                      </div>
                      <div class="privacy-setting-tags">
                        <a-tag :color="settingState(setting).color" class="status-tag">
                          {{ settingState(setting).label }}
                        </a-tag>
                        <a-tag v-if="editFor(setting).op === 'unset' && isEditChanged(setting)" color="warning" class="status-tag">
                          {{ t('privacy.value.pendingUnset') }}
                        </a-tag>
                        <a-tag v-if="isEditChanged(setting)" color="processing" class="status-tag">
                          {{ t('privacy.value.unsaved') }}
                        </a-tag>
                      </div>
                    </div>

                    <div class="privacy-setting-key">
                      <a-typography-text class="mono privacy-copy-block" :copyable="{ text: setting.key }">
                        {{ setting.key }}
                      </a-typography-text>
                    </div>

                    <div v-if="setting.impact" class="privacy-impact">{{ setting.impact }}</div>
                  </div>

                  <div class="privacy-setting-values">
                    <div class="privacy-value-block">
                      <div class="metadata-label">{{ t('privacy.value.strict') }}</div>
                      <a-typography-text class="mono privacy-copy-block" :copyable="{ text: formatConfigValue(strictValue(setting)) }">
                        {{ formatConfigValue(strictValue(setting)) }}
                      </a-typography-text>
                    </div>
                    <div class="privacy-value-block">
                      <div class="metadata-label">{{ t('privacy.value.type') }}</div>
                      <span class="privacy-type-chip">{{ valueTypeLabel(valueType(setting)) }}</span>
                    </div>
                  </div>

                  <div class="privacy-editor">
                    <div class="metadata-label">{{ t('privacy.value.current') }}</div>
                    <a-switch
                      v-if="valueType(setting) === 'bool'"
                      v-model:checked="editFor(setting).boolValue"
                      :disabled="!canEdit(setting)"
                      @change="markEditSet(setting.id)"
                    />
                    <a-select
                      v-else-if="valueType(setting) === 'stringArray'"
                      v-model:value="editFor(setting).arrayValue"
                      mode="tags"
                      class="privacy-array-editor"
                      :token-separators="[',']"
                      :disabled="!canEdit(setting)"
                      @update:value="markEditSet(setting.id)"
                    />
                    <a-input
                      v-else
                      v-model:value="editFor(setting).stringValue"
                      class="privacy-string-editor"
                      :disabled="!canEdit(setting)"
                      @update:value="markEditSet(setting.id)"
                    />
                  </div>

                  <div class="privacy-setting-actions">
                    <a-button size="small" :disabled="!canEdit(setting) || savingAll || Boolean(savingId)" @click="useStrict(setting)">
                      <template #icon>
                        <SafetyCertificateOutlined />
                      </template>
                      {{ t('privacy.action.useStrict') }}
                    </a-button>
                    <a-button
                      size="small"
                      :disabled="!canEdit(setting) || !setting.supportsUnset || savingAll || Boolean(savingId)"
                      @click="unsetEdit(setting)"
                    >
                      <template #icon>
                        <DeleteOutlined />
                      </template>
                      {{ t('privacy.action.unset') }}
                    </a-button>
                    <a-button size="small" :disabled="savingAll || Boolean(savingId)" @click="resetEdit(setting)">
                      <template #icon>
                        <UndoOutlined />
                      </template>
                      {{ t('privacy.action.reset') }}
                    </a-button>
                    <a-button
                      size="small"
                      type="primary"
                      :loading="savingId === setting.id"
                      :disabled="!canEdit(setting) || !isEditChanged(setting) || savingAll || (Boolean(savingId) && savingId !== setting.id)"
                      @click="saveSettings([setting])"
                    >
                      <template #icon>
                        <CheckOutlined />
                      </template>
                      {{ t('privacy.action.save') }}
                    </a-button>
                  </div>
                </article>
              </div>
            </section>
          </div>
        </div>
      </section>
    </div>
  </a-spin>
</template>
