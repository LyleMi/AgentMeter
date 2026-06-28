<script setup lang="ts">
import { computed } from 'vue'
import AButton from 'ant-design-vue/es/button'
import ASelect from 'ant-design-vue/es/select'
import { ClearOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { useMessages } from '../i18n'

export interface UsageScopeOption {
  value: string
  label: string
  title?: string
}

export interface UsageScopeBarFilters {
  agent?: string
  model?: string
  from: string
  to: string
}

const props = withDefaults(
  defineProps<{
    filters: UsageScopeBarFilters
    agentOptions?: UsageScopeOption[]
    modelOptions?: UsageScopeOption[]
    loading?: boolean
  }>(),
  {
    agentOptions: () => [],
    modelOptions: () => [],
    loading: false
  }
)

const emit = defineEmits<{
  'update:filters': [filters: UsageScopeBarFilters]
  refresh: []
  clear: []
}>()

const { t } = useMessages({
  en: {
    'filter.agent': 'Source',
    'filter.model': 'Model',
    'filter.from': 'From',
    'filter.to': 'To',
    'filter.fromAria': 'Started from',
    'filter.toAria': 'Started to',
    'action.refresh': 'Refresh',
    'action.clear': 'Clear'
  },
  'zh-CN': {
    'filter.agent': '来源',
    'filter.model': '模型',
    'filter.from': '从',
    'filter.to': '到',
    'filter.fromAria': '开始日期从',
    'filter.toAria': '开始日期到',
    'action.refresh': '刷新',
    'action.clear': '清除'
  }
})

const hasActiveFilters = computed(() =>
  Boolean(props.filters.agent || props.filters.model || props.filters.from || props.filters.to)
)

function cleanSelectValue(value: unknown) {
  return typeof value === 'string' && value ? value : undefined
}

function updateFilter(patch: Partial<UsageScopeBarFilters>) {
  emit('update:filters', {
    agent: props.filters.agent,
    model: props.filters.model,
    from: props.filters.from,
    to: props.filters.to,
    ...patch
  })
}

function updateDateFilter(key: 'from' | 'to', event: Event) {
  updateFilter({ [key]: (event.target as HTMLInputElement).value })
}
</script>

<template>
  <section class="usage-scope-bar">
    <div class="usage-scope-fields">
      <a-select
        class="usage-scope-select"
        allow-clear
        show-search
        :disabled="loading"
        :placeholder="t('filter.agent')"
        :options="agentOptions"
        :value="filters.agent"
        @change="(value) => updateFilter({ agent: cleanSelectValue(value) })"
      />
      <a-select
        class="usage-scope-select usage-scope-model"
        allow-clear
        show-search
        :disabled="loading"
        :placeholder="t('filter.model')"
        :options="modelOptions"
        :value="filters.model"
        @change="(value) => updateFilter({ model: cleanSelectValue(value) })"
      />
      <label class="inline-field usage-scope-date">
        <span>{{ t('filter.from') }}</span>
        <input
          class="native-date-input"
          type="date"
          :aria-label="t('filter.fromAria')"
          :disabled="loading"
          :value="filters.from"
          @change="(event) => updateDateFilter('from', event)"
        />
      </label>
      <label class="inline-field usage-scope-date">
        <span>{{ t('filter.to') }}</span>
        <input
          class="native-date-input"
          type="date"
          :aria-label="t('filter.toAria')"
          :disabled="loading"
          :value="filters.to"
          @change="(event) => updateDateFilter('to', event)"
        />
      </label>
    </div>
    <div class="usage-scope-actions">
      <a-button :loading="loading" @click="emit('refresh')">
        <template #icon>
          <ReloadOutlined />
        </template>
        {{ t('action.refresh') }}
      </a-button>
      <a-button :disabled="loading || !hasActiveFilters" @click="emit('clear')">
        <template #icon>
          <ClearOutlined />
        </template>
        {{ t('action.clear') }}
      </a-button>
    </div>
  </section>
</template>

<style scoped>
.usage-scope-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  margin-bottom: 14px;
  padding: 8px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-radius: var(--am-radius);
}

.usage-scope-fields,
.usage-scope-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.usage-scope-fields {
  flex: 1 1 auto;
  flex-wrap: wrap;
}

.usage-scope-actions {
  flex: 0 0 auto;
  justify-content: flex-end;
}

.usage-scope-select {
  width: 240px;
  max-width: 100%;
}

.usage-scope-model {
  width: 210px;
}

.usage-scope-date {
  width: 178px;
}

@media (max-width: 900px) {
  .usage-scope-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .usage-scope-actions {
    justify-content: flex-start;
  }
}

@media (max-width: 760px) {
  .usage-scope-fields {
    display: grid;
    grid-template-columns: 1fr;
  }

  .usage-scope-select,
  .usage-scope-model,
  .usage-scope-date {
    width: 100%;
  }
}
</style>
