<script setup lang="ts">
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import AInputNumber from 'ant-design-vue/es/input-number'
import ASelect from 'ant-design-vue/es/select'
import ASwitch from 'ant-design-vue/es/switch'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { CheckOutlined, DeleteOutlined, SafetyCertificateOutlined, UndoOutlined } from '@ant-design/icons-vue'
import type { PrivacyConfigSetting, PrivacyConfigValueType } from '../../api/types'
import type { PrivacyConfigEdit } from '../../composables/useAgentPrivacyEditor'

const ATypographyText = Typography.Text

type Translate = (key: string, params?: Record<string, string>) => string

interface StatusState {
  color: string
  label: string
}

defineProps<{
  t: Translate
  setting: PrivacyConfigSetting
  edit: PrivacyConfigEdit
  state: StatusState
  cardClass: Record<string, boolean>
  title: string
  description: string
  impact: string
  defaultBehavior: string
  recommendedBehavior: string
  valueType: PrivacyConfigValueType
  strictValue: unknown
  changed: boolean
  savingAll: boolean
  savingId: string
  editable: boolean
  formatConfigValue: (value: unknown) => string
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
  <article :class="cardClass">
    <div class="privacy-setting-main">
      <div class="privacy-setting-heading">
        <div>
          <div class="privacy-setting-title">{{ title }}</div>
          <div class="privacy-setting-description">{{ description }}</div>
        </div>
        <div class="privacy-setting-tags">
          <a-tag :color="state.color" class="status-tag">
            {{ state.label }}
          </a-tag>
          <a-tag v-if="edit.op === 'unset' && changed" color="warning" class="status-tag">
            {{ t('privacy.value.pendingUnset') }}
          </a-tag>
          <a-tag v-if="changed" color="processing" class="status-tag">
            {{ t('privacy.value.unsaved') }}
          </a-tag>
        </div>
      </div>

      <div class="privacy-setting-key">
        <a-typography-text class="mono privacy-copy-block" :copyable="{ text: setting.key }">
          {{ setting.key }}
        </a-typography-text>
      </div>

      <div v-if="setting.impact" class="privacy-impact">{{ impact }}</div>
      <div class="privacy-value-block">
        <div class="metadata-label">{{ t('privacy.value.type') }}</div>
        <span class="privacy-type-chip">{{ valueTypeLabel(valueType) }}</span>
      </div>
    </div>

    <div class="privacy-setting-values">
      <div class="privacy-value-block privacy-current-value-block">
        <div class="metadata-label">{{ t('privacy.value.current') }}</div>
        <a-typography-text
          class="mono privacy-copy-block privacy-current-value"
          :copyable="{ text: setting.configured ? formatConfigValue(setting.currentValue) : t('privacy.value.notConfigured') }"
        >
          {{ setting.configured ? formatConfigValue(setting.currentValue) : t('privacy.value.notConfigured') }}
        </a-typography-text>
      </div>
      <div class="privacy-value-block">
        <div class="metadata-label">{{ t('privacy.value.defaultProfile') }}</div>
        <div class="privacy-profile-behavior">{{ defaultBehavior }}</div>
      </div>
      <div class="privacy-value-block">
        <div class="metadata-label">{{ t('privacy.value.recommendedProfile') }}</div>
        <div class="privacy-profile-behavior">{{ recommendedBehavior }}</div>
      </div>
      <div class="privacy-value-block">
        <div class="metadata-label">{{ t('privacy.value.strictProfile') }}</div>
        <a-typography-text class="mono privacy-copy-block" :copyable="{ text: formatConfigValue(strictValue) }">
          {{ formatConfigValue(strictValue) }}
        </a-typography-text>
      </div>
      <div class="privacy-editor">
        <div class="metadata-label">{{ t('privacy.value.editable') }}</div>
        <a-switch
          v-if="valueType === 'bool'"
          v-model:checked="edit.boolValue"
          :disabled="!editable"
          @change="$emit('markSet', setting.id)"
        />
        <a-select
          v-else-if="valueType === 'stringArray'"
          v-model:value="edit.arrayValue"
          mode="tags"
          class="privacy-array-editor"
          :token-separators="[',']"
          :disabled="!editable"
          @update:value="$emit('markSet', setting.id)"
        />
        <a-input-number
          v-else-if="valueType === 'number'"
          v-model:value="edit.numberValue"
          class="privacy-string-editor"
          :min="0"
          :disabled="!editable"
          @update:value="$emit('markSet', setting.id)"
        />
        <a-input
          v-else
          v-model:value="edit.stringValue"
          class="privacy-string-editor"
          :disabled="!editable"
          @update:value="$emit('markSet', setting.id)"
        />
      </div>
    </div>

    <div class="privacy-setting-actions">
      <a-button size="small" :disabled="!editable || savingAll || Boolean(savingId)" @click="$emit('useStrict', setting)">
        <template #icon>
          <SafetyCertificateOutlined />
        </template>
        {{ t('privacy.action.useStrict') }}
      </a-button>
      <a-button
        size="small"
        :disabled="!editable || !setting.supportsUnset || savingAll || Boolean(savingId)"
        @click="$emit('unset', setting)"
      >
        <template #icon>
          <DeleteOutlined />
        </template>
        {{ t('privacy.action.unset') }}
      </a-button>
      <a-button size="small" :disabled="savingAll || Boolean(savingId)" @click="$emit('reset', setting)">
        <template #icon>
          <UndoOutlined />
        </template>
        {{ t('privacy.action.reset') }}
      </a-button>
      <a-button
        size="small"
        type="primary"
        :loading="savingId === setting.id"
        :disabled="!editable || !changed || savingAll || (Boolean(savingId) && savingId !== setting.id)"
        @click="$emit('save', setting)"
      >
        <template #icon>
          <CheckOutlined />
        </template>
        {{ t('privacy.action.save') }}
      </a-button>
    </div>
  </article>
</template>
