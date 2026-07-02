<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import AButton from 'ant-design-vue/es/button'
import AntTable from 'ant-design-vue/es/table'
import ATooltip from 'ant-design-vue/es/tooltip'
import { ArrowRightOutlined, WarningOutlined } from '@ant-design/icons-vue'
import {
  formatDateTime,
  formatDuration,
  formatNumber
} from '../../api'
import { useMessages } from '../../i18n'
import {
  buildAnomalyColumns,
  buildTableLocale
} from './columns'
import { createModelSignalsDisplay } from './display'
import ModelCell from './ModelCell.vue'
import { modelSignalsMessages } from './messages'
import ModelSignalsTablePanel from './ModelSignalsTablePanel.vue'
import ProjectCell from './ProjectCell.vue'
import ReasonTags from './ReasonTags.vue'
import SourceCell from './SourceCell.vue'
import type { NormalizedAnomalySession } from './types'

const ATable = AntTable as unknown as DefineComponent

const props = withDefaults(defineProps<{
  rows?: NormalizedAnomalySession[]
  loading?: boolean
}>(), {
  rows: () => [],
  loading: false
})

const emit = defineEmits<{
  'open-session': [id: number]
}>()

const { t } = useMessages(modelSignalsMessages)
const {
  anomalyRowClass,
  anomalyRowKey,
  formatPercent,
  formatRate,
  projectInfo,
  sessionInfo,
  sourceInfo
} = createModelSignalsDisplay(t)

const columns = computed(() => buildAnomalyColumns(t))
const tableLocale = computed(() => buildTableLocale(t, props.loading, 'empty.anomalies'))

function openSession(id: number) {
  emit('open-session', id)
}
</script>

<template>
  <ModelSignalsTablePanel :title="t('anomaly.title')" :kicker="t('anomaly.kicker')" :icon="WarningOutlined">
    <a-table
      class="dense-table model-signals-anomaly-table"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :locale="tableLocale"
      :pagination="{ pageSize: 8 }"
      :row-key="anomalyRowKey"
      size="small"
      :custom-row="anomalyRowClass"
      :scroll="{ x: 1600 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'session'">
          <span class="mono model-signals-session" :title="sessionInfo(record).full">{{ sessionInfo(record).main }}</span>
          <div class="source-identity-meta">{{ formatNumber(record.totalTokens) }} {{ t('column.tokens') }}</div>
        </template>
        <template v-else-if="column.key === 'source'">
          <SourceCell :info="sourceInfo(record)" />
        </template>
        <template v-else-if="column.key === 'project'">
          <ProjectCell :info="projectInfo(record)" />
        </template>
        <template v-else-if="column.key === 'model'">
          <ModelCell :model="record.model" :fallback="t('fallback.unknown')" />
        </template>
        <template v-else-if="column.key === 'signal'">
          <ReasonTags
            :reasons="record.reasons"
            :color="record.failedToolCalls > 0 || record.score >= 0.45 ? 'warning' : 'processing'"
            :empty-text="t('fallback.noReason')"
          />
        </template>
        <template v-else-if="column.key === 'outputExpansion'"><span class="number-cell">{{ formatPercent(record.outputExpansionRate) }}</span></template>
        <template v-else-if="column.key === 'reasoning'"><span class="number-cell">{{ formatPercent(record.reasoningOverheadRate) }}</span></template>
        <template v-else-if="column.key === 'cacheMiss'"><span class="number-cell">{{ formatPercent(record.cacheMissRate) }}</span></template>
        <template v-else-if="column.key === 'failedTools'">
          <span class="number-cell" :class="{ 'status-error': record.failedToolCalls > 0 }">{{ formatNumber(record.failedToolCalls) }}</span>
        </template>
        <template v-else-if="column.key === 'throughput'"><span class="number-cell">{{ formatRate(record.modelThroughputTokensPerSecond, 1) }}</span></template>
        <template v-else-if="column.key === 'started'">{{ formatDateTime(record.startedAt) }}</template>
        <template v-else-if="column.key === 'duration'"><span class="number-cell">{{ formatDuration(record.modelDurationMs) }}</span></template>
        <template v-else-if="column.key === 'open'">
          <a-tooltip :title="t('action.openSession')">
            <a-button type="text" size="small" :disabled="!record.id" @click="openSession(record.id)">
              <template #icon>
                <ArrowRightOutlined />
              </template>
            </a-button>
          </a-tooltip>
        </template>
      </template>
    </a-table>
  </ModelSignalsTablePanel>
</template>

<style scoped>
.model-signals-session {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  color: var(--am-text);
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}
</style>
