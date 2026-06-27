<script setup lang="ts">
import { onMounted, ref, type DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import message from 'ant-design-vue/es/message'
import ASpin from 'ant-design-vue/es/spin'
import AntTable from 'ant-design-vue/es/table'
import Typography from 'ant-design-vue/es/typography'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { api, formatDateTime, formatNumber, type PricingModel } from '../api'

const ATable = AntTable as unknown as DefineComponent
const ATypographyText = Typography.Text

const loading = ref(true)
const pricingModels = ref<PricingModel[]>([])

const pricingColumns = [
  { title: 'Model', dataIndex: 'model', key: 'model', width: 300 },
  { title: 'Input / 1M', dataIndex: 'inputPer1m', key: 'input', width: 112, align: 'right' },
  { title: 'Cached / 1M', dataIndex: 'cachedInputPer1m', key: 'cached', width: 122, align: 'right' },
  { title: 'Output / 1M', dataIndex: 'outputPer1m', key: 'output', width: 122, align: 'right' },
  { title: 'Source', dataIndex: 'source', key: 'source', width: 180 }
]

function formatPrice(value: number) {
  return `$${value.toFixed(4)}`
}

async function load() {
  loading.value = true
  try {
    pricingModels.value = await api.getPricingModels()
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Load pricing failed')
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
          <h2 class="panel-title">Price</h2>
          <div class="panel-kicker">Pricing registry</div>
        </div>
        <div class="summary-actions">
          <span class="row-count">{{ formatNumber(pricingModels.length) }} models</span>
          <a-button @click="load">
            <template #icon>
              <ReloadOutlined />
            </template>
            Refresh
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
