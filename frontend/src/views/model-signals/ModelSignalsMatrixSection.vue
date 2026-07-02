<script setup lang="ts">
import { computed, type DefineComponent } from 'vue'
import AntTable from 'ant-design-vue/es/table'
import ATooltip from 'ant-design-vue/es/tooltip'
import { TableOutlined } from '@ant-design/icons-vue'
import {
  formatNumber,
  type ModelSignalMatrixRow
} from '../../api'
import { useMessages } from '../../i18n'
import {
  buildMatrixColumns,
  buildTableLocale
} from './columns'
import { createModelSignalsDisplay } from './display'
import { modelSignalsMessages } from './messages'
import ModelSignalsTablePanel from './ModelSignalsTablePanel.vue'
import SeverityTag from './SeverityTag.vue'
import SourceCell from './SourceCell.vue'
import SourceModelComparison from './SourceModelComparison.vue'

const ATable = AntTable as unknown as DefineComponent

const props = withDefaults(defineProps<{
  rows?: ModelSignalMatrixRow[]
  loading?: boolean
}>(), {
  rows: () => [],
  loading: false
})

const { t } = useMessages(modelSignalsMessages)
const {
  formatConfidence,
  formatLatency,
  formatRate,
  matrixCellKey,
  matrixCellTitle,
  matrixRowKey,
  severityClass,
  severityLabel,
  severityTagColor,
  sourceInfo
} = createModelSignalsDisplay(t)

const columns = computed(() => buildMatrixColumns(t))
const tableLocale = computed(() => buildTableLocale(t, props.loading, 'empty.matrix'))
</script>

<template>
  <div class="section-stack">
    <SourceModelComparison :rows="rows" :loading="loading" />

    <ModelSignalsTablePanel :title="t('matrix.title')" :kicker="t('matrix.kicker')" :icon="TableOutlined">
      <a-table
        class="dense-table model-signals-matrix-table"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :locale="tableLocale"
        :pagination="false"
        :row-key="matrixRowKey"
        size="small"
        :scroll="{ x: 980 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'source'">
            <SourceCell :info="sourceInfo(record)" />
          </template>
          <template v-else-if="column.key === 'models'">
            <div class="model-signals-matrix-cells">
              <a-tooltip
                v-for="cell in record.cells"
                :key="matrixCellKey(cell)"
                :title="matrixCellTitle(cell)"
                placement="topLeft"
              >
                <div class="model-signals-matrix-cell" :class="severityClass(cell.severity)">
                  <div class="model-signals-matrix-cell-head">
                    <span class="model-name">{{ cell.model || t('fallback.unknown') }}</span>
                    <SeverityTag :color="severityTagColor(cell.severity)" :label="severityLabel(cell.severity)" />
                  </div>
                  <div class="source-identity-meta">{{ cell.modelProvider || '-' }} · {{ formatNumber(cell.cohortCount) }} {{ t('tab.cohorts') }}</div>
                  <div class="model-signals-matrix-metrics">
                    <span>{{ formatLatency(cell.current?.modelLatencyMsPer1kOutputTokens) }}</span>
                    <span>{{ formatRate(cell.current?.modelThroughputTokensPerSecond, 1) }} tok/s</span>
                    <span>{{ formatConfidence(cell.confidence) }}</span>
                  </div>
                  <div class="model-signals-matrix-reason">{{ cell.keyReason || t('fallback.noReason') }}</div>
                </div>
              </a-tooltip>
            </div>
          </template>
        </template>
      </a-table>
    </ModelSignalsTablePanel>
  </div>
</template>

<style scoped>
.model-signals-matrix-cells {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.model-signals-matrix-cell {
  width: 244px;
  min-width: 0;
  padding: 8px 9px;
  background: var(--am-surface-subtle);
  border: 1px solid var(--am-border-subtle);
  border-left: 3px solid var(--am-success);
  border-radius: var(--am-radius-sm);
}

.model-signals-matrix-cell.severity-watch {
  border-left-color: var(--am-info);
}

.model-signals-matrix-cell.severity-warning {
  border-left-color: var(--am-warning);
  background: var(--am-warning-soft);
}

.model-signals-matrix-cell.severity-critical {
  border-left-color: var(--am-danger);
  background: var(--am-danger-soft);
}

.model-signals-matrix-cell-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}

.model-signals-matrix-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 6px;
  margin-top: 6px;
  color: var(--am-text-soft);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  line-height: 15px;
}

.model-signals-matrix-metrics span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-signals-matrix-reason {
  margin-top: 4px;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 11px;
  line-height: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
