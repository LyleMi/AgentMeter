<script setup lang="ts">
import PrivacySettingCard from './PrivacySettingCard.vue'
import type { PrivacyConfigSetting, PrivacyConfigValueType } from '../../api/types'
import type { PrivacyConfigEdit } from '../../composables/useAgentPrivacyEditor'

type Translate = (key: string, params?: Record<string, string>) => string

interface StatusState {
  color: string
  label: string
}

interface SettingGroup {
  name: string
  items: PrivacyConfigSetting[]
}

defineProps<{
  t: Translate
  settings: PrivacyConfigSetting[]
  groupedSettings: SettingGroup[]
  savingAll: boolean
  savingId: string
  formatNumber: (value: number | undefined) => string
  formatConfigValue: (value: unknown) => string
  editFor: (setting: PrivacyConfigSetting) => PrivacyConfigEdit
  canEdit: (setting: PrivacyConfigSetting) => boolean
  isEditChanged: (setting: PrivacyConfigSetting) => boolean
  settingState: (setting: PrivacyConfigSetting) => StatusState
  settingCardClass: (setting: PrivacyConfigSetting) => Record<string, boolean>
  localizedSettingTitle: (setting: PrivacyConfigSetting) => string
  localizedSettingDescription: (setting: PrivacyConfigSetting) => string
  localizedSettingImpact: (setting: PrivacyConfigSetting) => string
  valueType: (setting: PrivacyConfigSetting) => PrivacyConfigValueType
  strictValue: (setting: PrivacyConfigSetting) => unknown
  valueTypeLabel: (type: PrivacyConfigValueType) => string
}>()

defineEmits<{
  markSet: [id: string]
  useStrict: [setting: PrivacyConfigSetting]
  unset: [setting: PrivacyConfigSetting]
  reset: [setting: PrivacyConfigSetting]
  save: [setting: PrivacyConfigSetting]
}>()
</script>

<template>
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
            <PrivacySettingCard
              v-for="setting in group.items"
              :key="setting.id"
              :t="t"
              :setting="setting"
              :edit="editFor(setting)"
              :state="settingState(setting)"
              :card-class="settingCardClass(setting)"
              :title="localizedSettingTitle(setting)"
              :description="localizedSettingDescription(setting)"
              :impact="localizedSettingImpact(setting)"
              :value-type="valueType(setting)"
              :strict-value="strictValue(setting)"
              :changed="isEditChanged(setting)"
              :saving-all="savingAll"
              :saving-id="savingId"
              :editable="canEdit(setting)"
              :format-config-value="formatConfigValue"
              :value-type-label="valueTypeLabel"
              @mark-set="$emit('markSet', $event)"
              @use-strict="$emit('useStrict', $event)"
              @unset="$emit('unset', $event)"
              @reset="$emit('reset', $event)"
              @save="$emit('save', $event)"
            />
          </div>
        </section>
      </div>
    </div>
  </section>
</template>
