<script setup lang="ts">
import type { Component } from 'vue'

export interface TimeKpiCard {
  label: string
  value: string
  note: string
  icon: Component
}

defineProps<{
  cards: TimeKpiCard[]
}>()
</script>

<template>
  <div class="overview-time-kpis">
    <div v-for="item in cards" :key="item.label" class="overview-kpi-card overview-time-kpi-card">
      <div class="overview-kpi-head">
        <span class="metric-label">{{ item.label }}</span>
        <component :is="item.icon" class="metric-icon" />
      </div>
      <div class="overview-kpi-value" :title="item.value">
        <span>{{ item.value }}</span>
      </div>
      <div class="overview-kpi-note">{{ item.note }}</div>
    </div>
  </div>
</template>

<style scoped>
.overview-time-kpis {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.overview-time-kpi-card:last-child {
  grid-column: 1 / -1;
  min-height: 106px;
}

.overview-time-kpi-card .overview-kpi-value {
  font-size: 24px;
  line-height: 30px;
}
</style>
