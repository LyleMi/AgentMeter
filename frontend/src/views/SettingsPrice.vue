<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import AInput from 'ant-design-vue/es/input'
import AInputNumber from 'ant-design-vue/es/input-number'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import ATag from 'ant-design-vue/es/tag'
import AntTable from 'ant-design-vue/es/table'
import Typography from 'ant-design-vue/es/typography'
import { DollarCircleOutlined, ReloadOutlined, SaveOutlined } from '@ant-design/icons-vue'
import { api, formatDateTime, formatNumber, isStaticDemo, type PricingModel } from '../api'
import { notifyAppDataChanged } from '../events'
import { useMessages } from '../i18n'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
type PriceValue = number | undefined

const { t } = useMessages({
  en: {
    'price.title': 'Price',
    'price.kicker': 'Pricing registry and custom rates',
    'price.action.refresh': 'Refresh',
    'price.action.save': 'Save price',
    'price.meta.models': 'models',
    'price.meta.custom': 'custom',
    'price.meta.seeded': 'seeded',
    'price.meta.active': 'Active',
    'price.state.demo': 'Static demo',
    'price.state.custom': '{count} custom',
    'price.state.seeded': 'Seeded only',
    'price.column.model': 'Model',
    'price.column.input': 'Input / 1M',
    'price.column.cached': 'Cached / 1M',
    'price.column.output': 'Output / 1M',
    'price.column.source': 'Source',
    'price.form.model': 'Model name',
    'price.form.input': 'Input',
    'price.form.cached': 'Cached',
    'price.form.output': 'Output',
    'price.form.source': 'Source note',
    'price.empty': 'No pricing models',
    'price.message.loadFailed': 'Load pricing failed',
    'price.message.saved': 'Price saved',
    'price.message.saveFailed': 'Save pricing failed',
    'price.message.modelRequired': 'Model name is required',
    'price.message.invalidPrice': 'Prices must be zero or greater',
    'price.message.demoReadOnly': 'Static demo mode is read-only.',
    'price.tag.custom': 'custom'
  },
  'zh-CN': {
    'price.title': '价格',
    'price.kicker': '定价注册表与自定义费率',
    'price.action.refresh': '刷新',
    'price.action.save': '保存价格',
    'price.meta.models': '个模型',
    'price.meta.custom': '自定义',
    'price.meta.seeded': '内置',
    'price.meta.active': '当前',
    'price.state.demo': '静态演示',
    'price.state.custom': '{count} 个自定义',
    'price.state.seeded': '仅内置',
    'price.column.model': '模型',
    'price.column.input': '输入 / 1M',
    'price.column.cached': '缓存 / 1M',
    'price.column.output': '输出 / 1M',
    'price.column.source': '来源',
    'price.form.model': '模型名称',
    'price.form.input': '输入',
    'price.form.cached': '缓存',
    'price.form.output': '输出',
    'price.form.source': '来源备注',
    'price.empty': '暂无定价模型',
    'price.message.loadFailed': '加载定价失败',
    'price.message.saved': '价格已保存',
    'price.message.saveFailed': '保存价格失败',
    'price.message.modelRequired': '需要填写模型名称',
    'price.message.invalidPrice': '价格必须大于或等于 0',
    'price.message.demoReadOnly': '静态演示模式为只读。',
    'price.tag.custom': '自定义'
  }
})

const loading = ref(true)
const saving = ref(false)
const pricingModels = ref<PricingModel[]>([])
const customModel = ref('')
const customSource = ref('')
const customInputPer1m = ref<PriceValue>()
const customCachedInputPer1m = ref<PriceValue>()
const customOutputPer1m = ref<PriceValue>()

const customCount = computed(() => pricingModels.value.filter((item) => item.isCustom).length)
const seededCount = computed(() => pricingModels.value.length - customCount.value)
const priceState = computed(() => {
  if (isStaticDemo) return { color: 'processing', label: t('price.state.demo') }
  if (customCount.value > 0) return { color: 'success', label: t('price.state.custom', { count: formatNumber(customCount.value) }) }
  return { color: 'default', label: t('price.state.seeded') }
})

const pricingColumns = computed(() => [
  { title: t('price.column.model'), dataIndex: 'model', key: 'model', width: 300 },
  { title: t('price.column.input'), dataIndex: 'inputPer1m', key: 'input', width: 112, align: 'right' },
  { title: t('price.column.cached'), dataIndex: 'cachedInputPer1m', key: 'cached', width: 122, align: 'right' },
  { title: t('price.column.output'), dataIndex: 'outputPer1m', key: 'output', width: 122, align: 'right' },
  { title: t('price.column.source'), dataIndex: 'source', key: 'source', width: 180 }
])
const tableLocale = computed(() => ({ emptyText: t('price.empty') }))

function formatPrice(value: number) {
  return `$${value.toFixed(4)}`
}

function inputValue(value: PriceValue) {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

function priceInputStatus(value: PriceValue) {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function sortPricingModels(rows: PricingModel[]) {
  return [...rows].sort((left, right) => left.normalizedModel.localeCompare(right.normalizedModel))
}

function mergeSavedPricing(saved: PricingModel) {
  const next = pricingModels.value.slice()
  const index = next.findIndex((item) => item.normalizedModel === saved.normalizedModel)
  if (index >= 0) {
    next.splice(index, 1, saved)
  } else {
    next.push(saved)
  }
  pricingModels.value = sortPricingModels(next)
}

function resetCustomForm() {
  customModel.value = ''
  customSource.value = ''
  customInputPer1m.value = undefined
  customCachedInputPer1m.value = undefined
  customOutputPer1m.value = undefined
}

async function load() {
  loading.value = true
  try {
    pricingModels.value = sortPricingModels(await api.getPricingModels())
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('price.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function saveCustomPricing() {
  if (isStaticDemo) {
    message.info(t('price.message.demoReadOnly'))
    return
  }
  const model = customModel.value.trim()
  if (!model) {
    message.warning(t('price.message.modelRequired'))
    return
  }
  if (![customInputPer1m.value, customCachedInputPer1m.value, customOutputPer1m.value].every(priceInputStatus)) {
    message.warning(t('price.message.invalidPrice'))
    return
  }
  saving.value = true
  try {
    const saved = await api.savePricingModel({
      model,
      inputPer1m: inputValue(customInputPer1m.value),
      cachedInputPer1m: inputValue(customCachedInputPer1m.value),
      outputPer1m: inputValue(customOutputPer1m.value),
      source: customSource.value.trim() || undefined
    })
    mergeSavedPricing(saved)
    resetCustomForm()
    notifyAppDataChanged('pricing')
    message.success(t('price.message.saved'))
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('price.message.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <a-spin :spinning="loading">
    <section class="panel settings-tool-panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('price.title') }}</h2>
          <div class="panel-kicker">{{ t('price.kicker') }}</div>
        </div>
        <div class="summary-actions">
          <a-tag :color="priceState.color" class="status-tag">{{ priceState.label }}</a-tag>
          <span class="row-count">{{ formatNumber(pricingModels.length) }} {{ t('price.meta.models') }}</span>
          <a-button @click="load">
            <template #icon>
              <ReloadOutlined />
            </template>
            {{ t('price.action.refresh') }}
          </a-button>
        </div>
      </div>
      <div class="panel-body">
        <div class="section-stack">
          <div class="metadata-grid">
            <div class="metadata-item">
              <div class="metadata-label">{{ t('price.meta.models') }}</div>
              <div class="metadata-value">{{ formatNumber(pricingModels.length) }}</div>
            </div>
            <div class="metadata-item">
              <div class="metadata-label">{{ t('price.meta.custom') }}</div>
              <div class="metadata-value status-ok">{{ formatNumber(customCount) }}</div>
            </div>
            <div class="metadata-item">
              <div class="metadata-label">{{ t('price.meta.seeded') }}</div>
              <div class="metadata-value">{{ formatNumber(seededCount) }}</div>
            </div>
            <div class="metadata-item">
              <div class="metadata-label">{{ t('price.meta.active') }}</div>
              <div class="metadata-value">{{ priceState.label }}</div>
            </div>
          </div>

          <div class="pricing-edit-row">
            <a-input v-model:value="customModel" class="pricing-model-input" :placeholder="t('price.form.model')" :disabled="isStaticDemo" @pressEnter="saveCustomPricing">
              <template #prefix>
                <DollarCircleOutlined />
              </template>
            </a-input>
            <a-input-number v-model:value="customInputPer1m" class="pricing-rate-input" :min="0" :precision="4" :step="0.01" :placeholder="t('price.form.input')" :disabled="isStaticDemo" />
            <a-input-number v-model:value="customCachedInputPer1m" class="pricing-rate-input" :min="0" :precision="4" :step="0.01" :placeholder="t('price.form.cached')" :disabled="isStaticDemo" />
            <a-input-number v-model:value="customOutputPer1m" class="pricing-rate-input" :min="0" :precision="4" :step="0.01" :placeholder="t('price.form.output')" :disabled="isStaticDemo" />
            <a-input v-model:value="customSource" class="pricing-source-input" :placeholder="t('price.form.source')" :disabled="isStaticDemo" @pressEnter="saveCustomPricing" />
            <a-button type="primary" :loading="saving" :disabled="isStaticDemo" @click="saveCustomPricing">
              <template #icon>
                <SaveOutlined />
              </template>
              {{ t('price.action.save') }}
            </a-button>
          </div>

          <a-table
            class="dense-table pricing-table"
            size="small"
            :columns="pricingColumns"
            :data-source="pricingModels"
            row-key="id"
            :pagination="false"
            :locale="tableLocale"
            :scroll="{ x: 900 }"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'model'">
                <div class="pricing-model-cell">
                  <a-typography-text :ellipsis="{ tooltip: record.model }">
                    {{ record.model }}
                  </a-typography-text>
                  <a-tag v-if="record.isCustom" color="success" class="status-tag">{{ t('price.tag.custom') }}</a-tag>
                </div>
                <div v-if="record.normalizedModel && record.normalizedModel !== record.model" class="muted pricing-source-date">
                  {{ record.normalizedModel }}
                </div>
              </template>
              <template v-else-if="column.key === 'input'">
                <span class="number-cell price-cell">{{ formatPrice(record.inputPer1m) }}</span>
              </template>
              <template v-else-if="column.key === 'cached'">
                <span class="number-cell price-cell">{{ formatPrice(record.cachedInputPer1m) }}</span>
              </template>
              <template v-else-if="column.key === 'output'">
                <span class="number-cell price-cell">{{ formatPrice(record.outputPer1m) }}</span>
              </template>
              <template v-else-if="column.key === 'source'">
                <a-typography-text :ellipsis="{ tooltip: `${record.source} · ${formatDateTime(record.effectiveFrom)}` }">
                  {{ record.source }}
                </a-typography-text>
                <div class="muted pricing-source-date">{{ formatDateTime(record.effectiveFrom) }}</div>
              </template>
            </template>
          </a-table>
        </div>
      </div>
    </section>
  </a-spin>
</template>
