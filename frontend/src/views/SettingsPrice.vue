<script setup lang="ts">
import { computed, onMounted, ref, type DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import Typography from 'ant-design-vue/es/typography'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDateTime, formatNumber, type PricingModel } from '../api'
import { useMessages } from '../i18n'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text
const { t } = useMessages({
  en: {
    'price.title': 'Price',
    'price.kicker': 'Pricing registry',
    'price.action.refresh': 'Refresh',
    'price.meta.models': 'models',
    'price.column.model': 'Model',
    'price.column.input': 'Input / 1M',
    'price.column.cached': 'Cached / 1M',
    'price.column.output': 'Output / 1M',
    'price.column.source': 'Source',
    'price.empty': 'No pricing models',
    'price.message.loadFailed': 'Load pricing failed'
  },
  'zh-CN': {
    'price.title': '价格',
    'price.kicker': '定价注册表',
    'price.action.refresh': '刷新',
    'price.meta.models': '个模型',
    'price.column.model': '模型',
    'price.column.input': '输入 / 1M',
    'price.column.cached': '缓存 / 1M',
    'price.column.output': '输出 / 1M',
    'price.column.source': '来源',
    'price.empty': '暂无定价模型',
    'price.message.loadFailed': '加载定价失败'
  }
})

const loading = ref(true)
const pricingModels = ref<PricingModel[]>([])

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

async function load() {
  loading.value = true
  try {
    pricingModels.value = await api.getPricingModels()
  } catch (error) {
    message.error(error instanceof Error ? error.message : t('price.message.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <a-spin :spinning="loading">
    <section class="panel">
      <div class="panel-header">
        <div>
          <h2 class="panel-title">{{ t('price.title') }}</h2>
          <div class="panel-kicker">{{ t('price.kicker') }}</div>
        </div>
        <div class="summary-actions">
          <span class="row-count">{{ formatNumber(pricingModels.length) }} {{ t('price.meta.models') }}</span>
          <a-button @click="load">
            <template #icon>
              <ReloadOutlined />
            </template>
            {{ t('price.action.refresh') }}
          </a-button>
        </div>
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
            <a-typography-text :ellipsis="{ tooltip: record.model }">
              {{ record.model }}
            </a-typography-text>
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
    </section>
  </a-spin>
</template>
