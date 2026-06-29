<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import { api, isStaticDemo } from '../api'
import type {
  PrivacyConfigApplyResult,
  PrivacyConfigSetting,
  PrivacyConfigStatus,
  PrivacyProfileId,
  PrivacyTarget
} from '../api/types'
import PrivacySettingsPanel from '../components/privacy/PrivacySettingsPanel.vue'
import PrivacySummaryPanel from '../components/privacy/PrivacySummaryPanel.vue'
import { useAgentPrivacyEditor } from '../composables/useAgentPrivacyEditor'
import { useMessages } from '../i18n'
import { formatNumber } from '../presentation/formatters'
import { privacyValueType, privacyValueTypeLabel, strictPrivacyValue } from '../presentation/privacyConfig'
import type { PrivacyTranslate } from '../presentation/privacyUi'
import { agentPrivacyMessages } from './agent-privacy/messages'
import { useAgentPrivacyDisplay } from './agent-privacy/privacyDisplay'
import { useAgentPrivacyViewModel } from './agent-privacy/privacyViewModel'

const { t, locale } = useMessages(agentPrivacyMessages)

const loading = ref(true)
const savingAll = ref(false)
const savingId = ref('')
const applyingProfile = ref<PrivacyProfileId | ''>('')
const selectedTarget = ref<PrivacyTarget>('codex')
const selectedSourceKey = ref('')
const privacyStatus = ref<PrivacyConfigStatus | null>(null)
const lastApply = ref<PrivacyConfigApplyResult | null>(null)
const translate = t as PrivacyTranslate
const {
  syncEdits,
  editFor,
  markEditSet,
  useStrict,
  unsetEdit,
  resetEdit,
  isEditChanged,
  changeForSetting,
  canEdit: baseCanEdit
} = useAgentPrivacyEditor()
let loadRequestId = 0
let saveRequestId = 0
let syncingSourceKey = false

function activeSettingTarget(): PrivacyTarget | undefined {
  const target = privacyStatus.value?.target || selectedTarget.value
  if (target === 'codex' || target === 'gemini') return target
  return undefined
}

function canEdit(setting: PrivacyConfigSetting) {
  return !isStaticDemo && baseCanEdit(setting)
}

const {
  formatConfigValue,
  profileTitle,
  profileBehaviorText,
  settingState,
  localizedSettingGroup,
  localizedSettingTitle,
  localizedSettingDescription,
  localizedSettingImpact
} = useAgentPrivacyDisplay({
  t: translate,
  locale,
  activeTarget: activeSettingTarget
})

const {
  targetOptions,
  profileOptions,
  targetLabel,
  settings,
  changedSettings,
  kickerText,
  statusState,
  warningList,
  metricCounts,
  groupedSettings
} = useAgentPrivacyViewModel({
  t: translate,
  selectedTarget,
  privacyStatus,
  lastApply,
  canEdit,
  isEditChanged,
  localizedSettingGroup,
  profileTitle
})

function settingCardClass(setting: PrivacyConfigSetting) {
  return {
    'privacy-setting-card': true,
    'is-attention': !setting.configured && setting.status === 'attention',
    'is-changed': isEditChanged(setting)
  }
}

async function load() {
  const requestId = ++loadRequestId
  loading.value = true
  const sourceKey = selectedSourceKey.value
  try {
    const status = await api.getAgentPrivacy(selectedTarget.value, sourceKey || undefined)
    if (requestId !== loadRequestId) return
    privacyStatus.value = status
    syncingSourceKey = true
    selectedSourceKey.value = status.selectedSourceKey || ''
    syncingSourceKey = false
    syncEdits(status)
  } catch (error) {
    if (requestId !== loadRequestId) return
    message.error(error instanceof Error ? error.message : t('privacy.message.loadFailed'))
  } finally {
    if (requestId === loadRequestId) loading.value = false
  }
}

async function saveSettings(records: PrivacyConfigSetting[], saveAll = false) {
  if (isStaticDemo) {
    message.info(t('privacy.message.demoReadOnly'))
    return
  }
  if (applyingProfile.value) return

  const changes = records.filter((setting) => canEdit(setting) && isEditChanged(setting)).map(changeForSetting)
  if (!changes.length) {
    message.info(t('privacy.message.noChanges'))
    return
  }

  const requestId = ++saveRequestId
  const target = selectedTarget.value
  const sourceKey = selectedSourceKey.value
  if (saveAll) savingAll.value = true
  else savingId.value = changes[0].id

  try {
    const result = await api.applyAgentPrivacyChanges(target, changes, sourceKey || undefined)
    if (requestId !== saveRequestId || selectedTarget.value !== target || selectedSourceKey.value !== sourceKey) return
    if (result.status.target !== target) {
      message.error(t('privacy.message.targetMismatch'))
      return
    }
    privacyStatus.value = result.status
    syncingSourceKey = true
    selectedSourceKey.value = result.status.selectedSourceKey || ''
    syncingSourceKey = false
    lastApply.value = result
    syncEdits(result.status)
    if (result.changed?.length) {
      message.success(t('privacy.message.saved', { count: formatNumber(result.changed.length) }))
    } else {
      message.info(t('privacy.message.noChanges'))
    }
  } catch (error) {
    if (requestId !== saveRequestId || selectedTarget.value !== target || selectedSourceKey.value !== sourceKey) return
    message.error(error instanceof Error ? error.message : t('privacy.message.saveFailed'))
  } finally {
    if (requestId === saveRequestId) {
      savingAll.value = false
      savingId.value = ''
    }
  }
}

async function applyProfile(profile: PrivacyProfileId) {
  if (isStaticDemo) {
    message.info(t('privacy.message.demoReadOnly'))
    return
  }
  if (savingAll.value || savingId.value) return

  const requestId = ++saveRequestId
  const target = selectedTarget.value
  const sourceKey = selectedSourceKey.value
  applyingProfile.value = profile
  savingId.value = `profile:${profile}`

  try {
    const result = await api.applyAgentPrivacyProfile(target, profile, sourceKey || undefined)
    if (requestId !== saveRequestId || selectedTarget.value !== target || selectedSourceKey.value !== sourceKey) return
    if (result.status.target !== target) {
      message.error(t('privacy.message.targetMismatch'))
      return
    }
    privacyStatus.value = result.status
    syncingSourceKey = true
    selectedSourceKey.value = result.status.selectedSourceKey || ''
    syncingSourceKey = false
    lastApply.value = result
    syncEdits(result.status)
    const title = profileTitle(profile)
    if (result.changed?.length) {
      message.success(
        t('privacy.message.profileApplied', { profile: title, count: formatNumber(result.changed.length) })
      )
    } else {
      message.info(t('privacy.message.profileNoChanges', { profile: title }))
    }
  } catch (error) {
    if (requestId !== saveRequestId || selectedTarget.value !== target || selectedSourceKey.value !== sourceKey) return
    message.error(error instanceof Error ? error.message : t('privacy.message.profileFailed'))
  } finally {
    if (requestId === saveRequestId) {
      applyingProfile.value = ''
      savingId.value = ''
    }
  }
}

onMounted(load)
watch(selectedTarget, () => {
  saveRequestId++
  savingAll.value = false
  savingId.value = ''
  applyingProfile.value = ''
  syncingSourceKey = true
  selectedSourceKey.value = ''
  syncingSourceKey = false
  lastApply.value = null
  load()
})
watch(selectedSourceKey, () => {
  if (syncingSourceKey) return
  saveRequestId++
  savingAll.value = false
  savingId.value = ''
  applyingProfile.value = ''
  lastApply.value = null
  load()
})
</script>

<template>
  <a-spin :spinning="loading">
    <div class="section-stack">
      <PrivacySummaryPanel
        v-model:selected-target="selectedTarget"
        v-model:selected-source-key="selectedSourceKey"
        :t="translate"
        :target-options="targetOptions"
        :kicker-text="kickerText"
        :status-state="statusState"
        :privacy-status="privacyStatus"
        :profile-options="profileOptions"
        :last-apply="lastApply"
        :metric-counts="metricCounts"
        :changed-count="changedSettings.length"
        :saving-all="savingAll"
        :saving-id="savingId"
        :applying-profile="applyingProfile"
        :warning-list="warningList"
        :read-only="isStaticDemo"
        :target-label="targetLabel"
        :format-number="formatNumber"
        :format-config-value="formatConfigValue"
        @refresh="load"
        @save-all="saveSettings(changedSettings, true)"
        @apply-profile="applyProfile"
      />

      <PrivacySettingsPanel
        :t="translate"
        :settings="settings"
        :grouped-settings="groupedSettings"
        :saving-all="savingAll"
        :saving-id="savingId"
        :format-number="formatNumber"
        :format-config-value="formatConfigValue"
        :edit-for="editFor"
        :can-edit="canEdit"
        :is-edit-changed="isEditChanged"
        :setting-state="settingState"
        :setting-card-class="settingCardClass"
        :localized-setting-title="localizedSettingTitle"
        :localized-setting-description="localizedSettingDescription"
        :localized-setting-impact="localizedSettingImpact"
        :profile-behavior="profileBehaviorText"
        :value-type="privacyValueType"
        :strict-value="strictPrivacyValue"
        :value-type-label="privacyValueTypeLabel"
        @mark-set="markEditSet"
        @use-strict="useStrict"
        @unset="unsetEdit"
        @reset="resetEdit"
        @save="(setting) => saveSettings([setting])"
      />
    </div>
  </a-spin>
</template>
