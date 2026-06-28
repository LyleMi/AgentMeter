<script setup lang="ts">
import AAlert from 'ant-design-vue/es/alert'
import AButton from 'ant-design-vue/es/button'
import ASegmented from 'ant-design-vue/es/segmented'
import ATag from 'ant-design-vue/es/tag'
import Typography from 'ant-design-vue/es/typography'
import { ReloadOutlined, SaveOutlined } from '@ant-design/icons-vue'
import type { PrivacyConfigApplyResult, PrivacyConfigStatus, PrivacyTarget } from '../../api/types'

const ATypographyText = Typography.Text

type Translate = (key: string, params?: Record<string, string>) => string

interface TargetOption {
  label: string
  value: PrivacyTarget
}

interface StatusState {
  color: string
  label: string
}

interface MetricCounts {
  total: number
  strictConfigured: number
  defaultSafe: number
  customConfigured: number
  missingRequired: number
  unsavedChanges: number
}

defineProps<{
  t: Translate
  targetOptions: TargetOption[]
  kickerText: string
  statusState: StatusState
  privacyStatus: PrivacyConfigStatus | null
  lastApply: PrivacyConfigApplyResult | null
  metricCounts: MetricCounts
  changedCount: number
  savingAll: boolean
  savingId: string
  warningList: string[]
  targetLabel: string
  formatNumber: (value: number | undefined) => string
  formatConfigValue: (value: unknown) => string
}>()

defineEmits<{
  refresh: []
  saveAll: []
}>()

const selectedTarget = defineModel<PrivacyTarget>('selectedTarget', { required: true })
</script>

<template>
  <section class="panel agent-privacy-tool-panel">
    <div class="panel-header">
      <div>
        <h2 class="panel-title">{{ t('privacy.title') }}</h2>
        <div class="panel-kicker">{{ kickerText }}</div>
      </div>
      <div class="summary-actions">
        <a-segmented v-model:value="selectedTarget" :options="targetOptions" />
        <a-tag :color="statusState.color" class="status-tag">{{ statusState.label }}</a-tag>
        <a-button @click="$emit('refresh')">
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
              :disabled="!changedCount || Boolean(savingId)"
              @click="$emit('saveAll')"
            >
              <template #icon>
                <SaveOutlined />
              </template>
              {{ t('privacy.action.saveAll') }}
              <span v-if="changedCount">({{ formatNumber(changedCount) }})</span>
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
</template>
