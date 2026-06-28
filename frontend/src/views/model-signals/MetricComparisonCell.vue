<script setup lang="ts">
defineProps<{
  primary: string | number
  secondary?: string | number
}>()
</script>

<template>
  <div class="metric-comparison">
    <span>{{ primary }}</span>
    <span v-if="secondary !== undefined && secondary !== ''">{{ secondary }}</span>
  </div>
</template>

<style scoped>
.metric-comparison {
  display: grid;
  justify-items: end;
  gap: 1px;
  min-width: 0;
  font-variant-numeric: tabular-nums;
}

.metric-comparison > span:first-child {
  color: inherit;
  font-weight: 650;
}

.metric-comparison > span:last-child {
  max-width: 100%;
  overflow: hidden;
  color: var(--am-muted);
  font-size: 11px;
  line-height: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
