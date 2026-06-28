<script setup lang="ts">
import Empty from 'ant-design-vue/es/empty'
import TimeKpiGrid from './TimeKpiGrid.vue'
import TimeComposition from './TimeComposition.vue'
import TimeAttributionTables from './TimeAttributionTables.vue'
import { useTimeContext } from './timeContext'

const { hasIndexedData, loading } = useTimeContext()
</script>

<template>
  <div v-if="hasIndexedData || loading" class="section-stack">
    <TimeKpiGrid />
    <TimeComposition />
    <TimeAttributionTables />
  </div>
  <Empty
    v-else
    :description="false"
  />
</template>
