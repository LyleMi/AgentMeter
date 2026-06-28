<script setup lang="ts">
import { computed } from 'vue'
import ASegmented from 'ant-design-vue/es/segmented'
import { formatDisplayCost, formatDisplayNumber, formatNumber } from '../api'
import {
  numberDisplayMode,
  numberDisplayModeOptions,
  setNumberDisplayMode,
  useMessages,
  type NumberDisplayMode
} from '../i18n'

const { t } = useMessages({
  en: {
    'display.title': 'Display',
    'display.kicker': 'Number notation preference',
    'display.numberMode': 'Number display',
    'display.numberModeNote':
      'Exact tables, identifiers and line numbers stay full. KPI-scale values use this preference.',
    'display.mode.auto': 'Auto',
    'display.mode.full': 'Full',
    'display.mode.compact': 'Compact',
    'display.sample.title': 'Preview',
    'display.sample.tokens': 'Tokens',
    'display.sample.cost': 'Estimated cost',
    'display.sample.exact': 'Exact reference'
  },
  'zh-CN': {
    'display.title': '显示',
    'display.kicker': '数字显示偏好',
    'display.numberMode': '数字显示',
    'display.numberModeNote': '表格精确值、ID 和行号保持完整；KPI 类大数字使用此偏好。',
    'display.mode.auto': '自动',
    'display.mode.full': '完整',
    'display.mode.compact': '缩写',
    'display.sample.title': '预览',
    'display.sample.tokens': 'Token',
    'display.sample.cost': '预估费用',
    'display.sample.exact': '精确参考'
  }
})

const sampleTokens = 12_345_678
const sampleCost = 123_456.789

const modeOptions = computed(() =>
  numberDisplayModeOptions.map((option) => ({
    value: option.value,
    label: t(`display.mode.${option.value}` as const)
  }))
)
const tokenPreview = computed(() => formatDisplayNumber(sampleTokens))
const costPreview = computed(() => formatDisplayCost(sampleCost))

function updateNumberDisplayMode(value: string | number) {
  setNumberDisplayMode(String(value) as NumberDisplayMode)
}
</script>

<template>
  <div class="section-stack">
    <section class="panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('display.title') }}</h2>
          <div class="panel-kicker">{{ t('display.kicker') }}</div>
        </div>
      </div>
      <div class="panel-body">
        <div class="settings-display-grid">
          <div class="settings-display-control">
            <div class="metadata-label">{{ t('display.numberMode') }}</div>
            <a-segmented
              class="settings-display-segmented"
              :value="numberDisplayMode"
              :options="modeOptions"
              @change="updateNumberDisplayMode"
            />
            <div class="metric-note">{{ t('display.numberModeNote') }}</div>
          </div>

          <div class="settings-display-preview">
            <div class="metadata-label">{{ t('display.sample.title') }}</div>
            <div class="metadata-grid">
              <div class="metadata-item">
                <div class="metadata-label">{{ t('display.sample.tokens') }}</div>
                <div class="metadata-value number-cell" :title="tokenPreview.full">{{ tokenPreview.main }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('display.sample.cost') }}</div>
                <div class="metadata-value number-cell" :title="costPreview.full">{{ costPreview.main }}</div>
              </div>
              <div class="metadata-item">
                <div class="metadata-label">{{ t('display.sample.exact') }}</div>
                <div class="metadata-value number-cell">{{ formatNumber(sampleTokens) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings-display-grid {
  display: grid;
  grid-template-columns: minmax(260px, 0.8fr) minmax(320px, 1.2fr);
  gap: 16px;
  align-items: start;
}

.settings-display-control {
  display: grid;
  gap: 8px;
}

.settings-display-segmented {
  width: fit-content;
  max-width: 100%;
}

.settings-display-preview {
  display: grid;
  gap: 8px;
  min-width: 0;
}

@media (max-width: 760px) {
  .settings-display-grid {
    grid-template-columns: 1fr;
  }
}
</style>
